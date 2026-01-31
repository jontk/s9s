// Package views provides display views for various s9s data types.
package views

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/ui/components"
	"github.com/jontk/s9s/internal/ui/filters"
	"github.com/jontk/s9s/internal/ui/styles"
	"github.com/rivo/tview"
)

// AccountsView displays the accounts list
type AccountsView struct {
	*BaseView
	client         dao.SlurmClient
	table          *components.Table
	accounts       []*dao.Account
	mu             sync.RWMutex
	refreshTimer   *time.Timer
	refreshRate    time.Duration
	filter         string
	container      *tview.Flex
	filterInput    *tview.InputField
	statusBar      *tview.TextView
	app            *tview.Application
	pages          *tview.Pages
	filterBar      *components.FilterBar
	advancedFilter *filters.Filter
	isAdvancedMode bool
	globalSearch   *GlobalSearch
}

// SetPages sets the pages reference for modal handling
func (v *AccountsView) SetPages(pages *tview.Pages) {
	v.pages = pages
	// Set pages for filter bar if it exists
	if v.filterBar != nil {
		v.filterBar.SetPages(pages)
	}
}

// SetApp sets the application reference
func (v *AccountsView) SetApp(app *tview.Application) {
	v.app = app

	// Create filter bar now that we have app reference
	v.filterBar = components.NewFilterBar("accounts", app)
	v.filterBar.SetPages(v.pages)
	v.filterBar.SetOnFilterChange(v.onAdvancedFilterChange)
	v.filterBar.SetOnClose(v.closeAdvancedFilter)

	// Create global search
	v.globalSearch = NewGlobalSearch(v.client, app)
}

// NewAccountsView creates a new accounts view
func NewAccountsView(client dao.SlurmClient) *AccountsView {
	v := &AccountsView{
		BaseView:    NewBaseView("accounts", "Accounts"),
		client:      client,
		refreshRate: 30 * time.Second,
		accounts:    []*dao.Account{},
	}

	// Create table with account columns
	columns := []components.Column{
		components.NewColumn("Name").Width(20).Build(),
		components.NewColumn("Description").Width(30).Build(),
		components.NewColumn("Organization").Width(20).Build(),
		components.NewColumn("Parent").Width(15).Build(),
		components.NewColumn("Default QoS").Width(15).Build(),
		components.NewColumn("Max Jobs").Width(10).Align(tview.AlignRight).Sortable(true).Build(),
		components.NewColumn("Max Nodes").Width(10).Align(tview.AlignRight).Sortable(true).Build(),
		components.NewColumn("Max CPUs").Width(10).Align(tview.AlignRight).Sortable(true).Build(),
		components.NewColumn("Coordinators").Width(25).Build(),
	}

	v.table = components.NewTableBuilder().
		WithColumns(columns...).
		WithSelectable(true).
		WithHeader(true).
		WithColors(tcell.ColorYellow, tcell.ColorTeal, tcell.ColorWhite).
		Build()

	// Set up callbacks
	v.table.SetOnSelect(v.onAccountSelect)
	v.table.SetOnSort(v.onSort)

	// Create filter input with styled colors for visibility across themes
	v.filterInput = styles.NewStyledInputField().
		SetLabel("Filter: ").
		SetFieldWidth(30).
		SetChangedFunc(v.onFilterChange).
		SetDoneFunc(v.onFilterDone)

	// Create status bar
	v.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	// Create container layout (removed individual status bar to prevent conflicts with main status bar)
	v.container = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(v.filterInput, 1, 0, false).
		AddItem(v.table, 0, 1, true)

	return v
}

// Init initializes the accounts view
func (v *AccountsView) Init(ctx context.Context) error {
	_ = v.BaseView.Init(ctx)
	return v.Refresh()
}

// Render returns the view's main component
func (v *AccountsView) Render() tview.Primitive {
	return v.container
}

// Refresh updates the accounts data
func (v *AccountsView) Refresh() error {
	v.SetRefreshing(true)
	defer v.SetRefreshing(false)

	return v.refreshInternal()
}

// refreshInternal performs the actual refresh operation
func (v *AccountsView) refreshInternal() error {
	// Fetch accounts from backend
	accountsList, err := v.client.Accounts().List()
	if err != nil {
		v.SetLastError(err)
		// Note: Error handling removed since individual view status bars are no longer used
		return err
	}

	v.mu.Lock()
	v.accounts = accountsList.Accounts
	v.mu.Unlock()

	// Update table
	v.updateTable()
	// Note: No longer updating individual view status bar since we use main app status bar for hints

	// Schedule next refresh
	v.scheduleRefresh()

	return nil
}

// Stop stops the view
func (v *AccountsView) Stop() error {
	if v.refreshTimer != nil {
		v.refreshTimer.Stop()
	}
	return nil
}

// Hints returns keyboard hints
func (v *AccountsView) Hints() []string {
	hints := []string{
		"[yellow]Enter[white] Details",
		"[yellow]/[white] Filter",
		"[yellow]F3[white] Adv Filter",
		"[yellow]Ctrl+F[white] Search",
		"[yellow]1-9[white] Sort",
		"[yellow]R[white] Refresh",
		"[yellow]H[white] Show Hierarchy",
	}

	if v.isAdvancedMode {
		hints = append([]string{"[yellow]ESC[white] Exit Adv Filter"}, hints...)
	}

	return hints
}

// OnKey handles keyboard events
func (v *AccountsView) OnKey(event *tcell.EventKey) *tcell.EventKey {
	// Always prioritize filter input handling if it has focus
	// This allows the filter to maintain focus even when modals are present
	if v.filterInput != nil && v.filterInput.HasFocus() {
		if event.Key() == tcell.KeyEsc {
			v.app.SetFocus(v.table.Table)
			return nil
		}
		// Let the filter handle all keys when it has focus
		return event
	}

	// If a modal is open (and filter doesn't have focus), let it handle keys
	if v.pages != nil && v.pages.GetPageCount() > 1 {
		return event
	}

	// Handle advanced filter mode
	if v.isAdvancedMode && event.Key() == tcell.KeyEsc {
		v.closeAdvancedFilter()
		return nil
	}

	if handler, ok := v.accountsKeyHandlers()[event.Key()]; ok {
		handler()
		return nil
	}

	if event.Key() == tcell.KeyRune {
		if handler, ok := v.accountsRuneHandlers()[event.Rune()]; ok {
			handler()
			return nil
		}
	}

	return event
}

// accountsKeyHandlers returns a map of function key handlers
func (v *AccountsView) accountsKeyHandlers() map[tcell.Key]func() {
	return map[tcell.Key]func(){
		tcell.KeyF3:    v.showAdvancedFilter,
		tcell.KeyCtrlF: v.showGlobalSearch,
		tcell.KeyEnter: v.showAccountDetails,
	}
}

// accountsRuneHandlers returns a map of rune handlers
func (v *AccountsView) accountsRuneHandlers() map[rune]func() {
	return map[rune]func(){
		'R': func() { go func() { _ = v.Refresh() }() },
		'/': func() { v.app.SetFocus(v.filterInput) },
		'H': v.showAccountHierarchy,
	}
}

// OnFocus handles focus events
func (v *AccountsView) OnFocus() error {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
	// Refresh when gaining focus if we haven't loaded data yet
	if len(v.accounts) == 0 && !v.IsRefreshing() {
		go func() { _ = v.Refresh() }()
	}
	return nil
}

// OnLoseFocus handles loss of focus
func (v *AccountsView) OnLoseFocus() error {
	return nil
}

// updateTable updates the table with current account data
func (v *AccountsView) updateTable() {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Apply advanced filter if active
	filteredAccounts := v.accounts
	if v.advancedFilter != nil && len(v.advancedFilter.Expressions) > 0 {
		filteredAccounts = v.applyAdvancedFilter(v.accounts)
	}

	data := make([][]string, len(filteredAccounts))
	for i, account := range filteredAccounts {
		// Format limits
		maxJobs := formatLimit(account.MaxJobs)
		maxNodes := formatLimit(account.MaxNodes)
		maxCPUs := formatLimit(account.MaxCPUs)

		// Format coordinators
		coordinators := strings.Join(account.Coordinators, ", ")
		if len(coordinators) > 24 {
			coordinators = coordinators[:21] + "..."
		}

		// Color code parent/child relationships
		parent := account.Parent
		if parent == "" {
			parent = "[green]<root>[white]"
		} else if len(account.Children) > 0 {
			parent = fmt.Sprintf("[yellow]%s[white]", parent)
		}

		data[i] = []string{
			account.Name,
			account.Description,
			account.Organization,
			parent,
			account.DefaultQoS,
			maxJobs,
			maxNodes,
			maxCPUs,
			coordinators,
		}
	}

	v.table.SetData(data)
}

// formatLimit formats a limit value (0 or -1 means unlimited)
func formatLimit(limit int) string {
	if limit <= 0 {
		return "unlimited"
	}
	return fmt.Sprintf("%d", limit)
}

/*
TODO(lint): Review unused code - func (*AccountsView).updateStatusBar is unused

updateStatusBar updates the status bar
func (v *AccountsView) updateStatusBar(message string) {
	if message != "" {
		v.statusBar.SetText(message)
		return
	}

	v.mu.RLock()
	total := len(v.accounts)
	rootAccounts := 0
	childAccounts := 0

	for _, acc := range v.accounts {
		if acc.Parent == "" {
			rootAccounts++
		} else {
			childAccounts++
		}
	}
	v.mu.RUnlock()

	filtered := len(v.table.GetFilteredData())

	status := fmt.Sprintf("Total: %d | Root: %d | Child: %d", total, rootAccounts, childAccounts)

	if filtered < total {
		status += fmt.Sprintf(" | Filtered: %d", filtered)
	}

	if v.IsRefreshing() {
		status += " | [yellow]Refreshing...[white]"
	}

	v.statusBar.SetText(status)
}
*/

// scheduleRefresh schedules the next refresh
func (v *AccountsView) scheduleRefresh() {
	// Remove automatic refresh scheduling to prevent memory leak
	// Refresh will be handled by the main app refresh timer
}

// onAccountSelect handles account selection
func (v *AccountsView) onAccountSelect(_, _ int) {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		return
	}

	// Note: Selection handling removed since individual view status bars are no longer used
	_ = data[0] // accountName no longer used
}

// onSort handles column sorting
func (v *AccountsView) onSort(_ int, _ bool) {
	// Note: Sort feedback removed since individual view status bars are no longer used
}

// onFilterChange handles filter input changes
func (v *AccountsView) onFilterChange(text string) {
	v.filter = text
	v.table.SetFilter(text)
	// Note: Status bar update removed since individual view status bars are no longer used
}

// onFilterDone handles filter input completion
func (v *AccountsView) onFilterDone(_ tcell.Key) {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
}

// showAccountHierarchy shows the account hierarchy tree
func (v *AccountsView) showAccountHierarchy() {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if len(v.accounts) == 0 {
		// Note: Warning message removed since individual view status bars are no longer used
		return
	}

	// Build hierarchy tree
	tree := v.buildAccountTree()

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(tree).
		SetScrollable(true)

	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, true).
		AddItem(tview.NewTextView().SetText("Press ESC to close"), 1, 0, false)

	modal.SetBorder(true).
		SetTitle(" Account Hierarchy ").
		SetTitleAlign(tview.AlignCenter)

	// Create centered modal layout
	centeredModal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(modal, 0, 8, true).
			AddItem(nil, 0, 1, false), 0, 8, true).
		AddItem(nil, 0, 1, false)

	// Handle ESC key
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			if v.pages != nil {
				v.pages.RemovePage("account-hierarchy")
			}
			return nil
		}
		return event
	})

	if v.pages != nil {
		v.pages.AddPage("account-hierarchy", centeredModal, true, true)
	}
}

// buildAccountTree builds a hierarchical tree of accounts
func (v *AccountsView) buildAccountTree() string {
	// Create a map for quick lookup
	accountMap := make(map[string]*dao.Account)
	for _, acc := range v.accounts {
		accountMap[acc.Name] = acc
	}

	// Find root accounts
	var roots []*dao.Account
	for _, acc := range v.accounts {
		if acc.Parent == "" {
			roots = append(roots, acc)
		}
	}

	// Build tree
	var tree strings.Builder
	tree.WriteString("[yellow]Account Hierarchy Tree[white]\n\n")

	for _, root := range roots {
		v.buildAccountSubtree(&tree, root, accountMap, "", true)
	}

	return tree.String()
}

// buildAccountSubtree recursively builds account subtree
func (v *AccountsView) buildAccountSubtree(tree *strings.Builder, account *dao.Account, accountMap map[string]*dao.Account, prefix string, isLast bool) {
	// Write current account
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	_, _ = fmt.Fprintf(tree, "%s%s[green]%s[white]", prefix, connector, account.Name)
	if account.Description != "" {
		_, _ = fmt.Fprintf(tree, " (%s)", account.Description)
	}
	tree.WriteString("\n")

	// Prepare new prefix for children
	newPrefix := prefix
	if isLast {
		newPrefix += "    "
	} else {
		newPrefix += "│   "
	}

	// Process children
	for i, childName := range account.Children {
		if child, ok := accountMap[childName]; ok {
			isLastChild := i == len(account.Children)-1
			v.buildAccountSubtree(tree, child, accountMap, newPrefix, isLastChild)
		}
	}
}

// showAccountDetails shows detailed information for the selected account
func (v *AccountsView) showAccountDetails() {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		return
	}

	accountName := data[0]

	// Find the full account object
	var account *dao.Account
	v.mu.RLock()
	for _, acc := range v.accounts {
		if acc.Name == accountName {
			account = acc
			break
		}
	}
	v.mu.RUnlock()

	if account == nil {
		// Note: Error message removed since individual view status bars are no longer used
		return
	}

	// Create details view
	details := v.formatAccountDetails(account)

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(details).
		SetScrollable(true)

	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, true).
		AddItem(tview.NewTextView().SetText("Press ESC to close"), 1, 0, false)

	modal.SetBorder(true).
		SetTitle(fmt.Sprintf(" Account %s Details ", accountName)).
		SetTitleAlign(tview.AlignCenter)

	// Create centered modal layout
	centeredModal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(modal, 0, 8, true).
			AddItem(nil, 0, 1, false), 0, 8, true).
		AddItem(nil, 0, 1, false)

	// Handle ESC key
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			if v.pages != nil {
				v.pages.RemovePage("account-details")
			}
			return nil
		}
		return event
	})

	if v.pages != nil {
		v.pages.AddPage("account-details", centeredModal, true, true)
	}
}

// formatAccountDetails formats account details for display
func (v *AccountsView) formatAccountDetails(account *dao.Account) string {
	var details strings.Builder

	details.WriteString(fmt.Sprintf("[yellow]Account Name:[white] %s\n", account.Name))
	if account.Description != "" {
		details.WriteString(fmt.Sprintf("[yellow]Description:[white] %s\n", account.Description))
	}
	if account.Organization != "" {
		details.WriteString(fmt.Sprintf("[yellow]Organization:[white] %s\n", account.Organization))
	}

	// Hierarchy information
	details.WriteString("\n[teal]Hierarchy:[white]\n")
	if account.Parent != "" {
		details.WriteString(fmt.Sprintf("[yellow]  Parent:[white] %s\n", account.Parent))
	} else {
		details.WriteString("[yellow]  Parent:[white] [green]<root account>[white]\n")
	}

	if len(account.Children) > 0 {
		details.WriteString(fmt.Sprintf("[yellow]  Children:[white] %s\n", strings.Join(account.Children, ", ")))
	}

	// QoS information
	details.WriteString("\n[teal]Quality of Service:[white]\n")
	details.WriteString(fmt.Sprintf("[yellow]  Default QoS:[white] %s\n", account.DefaultQoS))
	if len(account.QoSList) > 0 {
		details.WriteString(fmt.Sprintf("[yellow]  Available QoS:[white] %s\n", strings.Join(account.QoSList, ", ")))
	}

	// Resource limits
	details.WriteString("\n[teal]Resource Limits:[white]\n")
	details.WriteString(fmt.Sprintf("[yellow]  Max Jobs:[white] %s\n", formatLimit(account.MaxJobs)))
	details.WriteString(fmt.Sprintf("[yellow]  Max Submit:[white] %s\n", formatLimit(account.MaxSubmit)))
	details.WriteString(fmt.Sprintf("[yellow]  Max Nodes:[white] %s\n", formatLimit(account.MaxNodes)))
	details.WriteString(fmt.Sprintf("[yellow]  Max CPUs:[white] %s\n", formatLimit(account.MaxCPUs)))
	if account.MaxWall > 0 {
		details.WriteString(fmt.Sprintf("[yellow]  Max Wall Time:[white] %s\n", formatTimeLimit(account.MaxWall)))
	}

	// Coordinators
	if len(account.Coordinators) > 0 {
		details.WriteString(fmt.Sprintf("\n[yellow]Coordinators:[white] %s\n", strings.Join(account.Coordinators, ", ")))
	}

	return details.String()
}

// formatTimeLimit formats a time limit in minutes
func formatTimeLimit(minutes int) string {
	if minutes <= 0 {
		return "unlimited"
	}

	days := minutes / (24 * 60)
	hours := (minutes % (24 * 60)) / 60
	mins := minutes % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh", days, hours)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

// showAdvancedFilter shows the advanced filter bar
func (v *AccountsView) showAdvancedFilter() {
	if v.filterBar == nil || v.pages == nil {
		return
	}

	v.isAdvancedMode = true

	// Replace the simple filter with advanced filter bar
	v.container.Clear()
	v.container.
		AddItem(v.filterBar, 5, 0, true).
		AddItem(v.table, 0, 1, false)

	v.filterBar.Show()
	// Note: Advanced filter status removed since individual view status bars are no longer used
}

// closeAdvancedFilter closes the advanced filter bar
func (v *AccountsView) closeAdvancedFilter() {
	v.isAdvancedMode = false

	// Restore the simple filter
	v.container.Clear()
	v.container.
		AddItem(v.filterInput, 1, 0, false).
		AddItem(v.table, 0, 1, true)

	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}

	// Note: Status bar update removed since individual view status bars are no longer used
}

// onAdvancedFilterChange handles advanced filter changes
func (v *AccountsView) onAdvancedFilterChange(filter *filters.Filter) {
	v.advancedFilter = filter
	v.updateTable()

	// Note: Status bar updates removed since individual view status bars are no longer used
}

// applyAdvancedFilter applies the advanced filter to accounts list
func (v *AccountsView) applyAdvancedFilter(accounts []*dao.Account) []*dao.Account {
	if v.advancedFilter == nil || len(v.advancedFilter.Expressions) == 0 {
		return accounts
	}

	var filtered []*dao.Account
	for _, account := range accounts {
		// Convert account to map for filter evaluation
		accountData := v.accountToMap(account)
		if v.advancedFilter.Evaluate(accountData) {
			filtered = append(filtered, account)
		}
	}

	return filtered
}

// accountToMap converts an account to a map for filter evaluation
func (v *AccountsView) accountToMap(account *dao.Account) map[string]interface{} {
	return map[string]interface{}{
		"Name":         account.Name,
		"Description":  account.Description,
		"Organization": account.Organization,
		"Parent":       account.Parent,
		"DefaultQoS":   account.DefaultQoS,
		"MaxJobs":      account.MaxJobs,
		"MaxSubmit":    account.MaxSubmit,
		"MaxNodes":     account.MaxNodes,
		"MaxCPUs":      account.MaxCPUs,
		"MaxWall":      account.MaxWall,
		"Coordinators": strings.Join(account.Coordinators, ","),
		"Children":     strings.Join(account.Children, ","),
		"QoSList":      strings.Join(account.QoSList, ","),
	}
}

// showGlobalSearch shows the global search interface
func (v *AccountsView) showGlobalSearch() {
	if v.globalSearch == nil || v.pages == nil {
		return
	}

	v.globalSearch.Show(v.pages, func(result SearchResult) {
		// This callback is called from an event handler, so direct primitive
		// manipulation is safe. Do NOT use QueueUpdateDraw here - it will deadlock!
		switch result.Type {
		case "account":
			if account, ok := result.Data.(*dao.Account); ok {
				v.focusOnAccount(account.Name)
			}
		case "job":
			if job, ok := result.Data.(*dao.Job); ok {
				v.SwitchToView("jobs")
				if jv, err := v.viewMgr.GetView("jobs"); err == nil {
					if jobsView, ok := jv.(*JobsView); ok {
						jobsView.focusOnJob(job.ID)
					}
				}
			}
		case "node":
			if node, ok := result.Data.(*dao.Node); ok {
				v.SwitchToView("nodes")
				if nv, err := v.viewMgr.GetView("nodes"); err == nil {
					if nodesView, ok := nv.(*NodesView); ok {
						nodesView.focusOnNode(node.Name)
					}
				}
			}
		case "partition":
			if partition, ok := result.Data.(*dao.Partition); ok {
				v.SwitchToView("partitions")
				if pv, err := v.viewMgr.GetView("partitions"); err == nil {
					if partitionsView, ok := pv.(*PartitionsView); ok {
						partitionsView.focusOnPartition(partition.Name)
					}
				}
			}
		case "user":
			if user, ok := result.Data.(*dao.User); ok {
				v.SwitchToView("users")
				if uv, err := v.viewMgr.GetView("users"); err == nil {
					if usersView, ok := uv.(*UsersView); ok {
						usersView.focusOnUser(user.Name)
					}
				}
			}
		case "qos":
			if qos, ok := result.Data.(*dao.QoS); ok {
				v.SwitchToView("qos")
				if qv, err := v.viewMgr.GetView("qos"); err == nil {
					if qosView, ok := qv.(*QoSView); ok {
						qosView.focusOnQoS(qos.Name)
					}
				}
			}
		case "reservation":
			if reservation, ok := result.Data.(*dao.Reservation); ok {
				v.SwitchToView("reservations")
				if rv, err := v.viewMgr.GetView("reservations"); err == nil {
					if reservationsView, ok := rv.(*ReservationsView); ok {
						reservationsView.focusOnReservation(reservation.Name)
					}
				}
			}
		}
	})
}

// focusOnAccount focuses the table on a specific account
func (v *AccountsView) focusOnAccount(accountName string) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Find the account in our account list
	for i, account := range v.accounts {
		if account.Name == accountName {
			// Select the row in the table
			v.table.Select(i, 0)
			// Note: Focus status removed since individual view status bars are no longer used
			return
		}
	}

	// Note: Error message removed since individual view status bars are no longer used
}
