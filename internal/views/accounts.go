package views

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/ui/components"
)

// AccountsView displays the accounts list
type AccountsView struct {
	*BaseView
	client       dao.SlurmClient
	table        *components.Table
	accounts     []*dao.Account
	mu           sync.RWMutex
	refreshTimer *time.Timer
	refreshRate  time.Duration
	filter       string
	container    *tview.Flex
	filterInput  *tview.InputField
	statusBar    *tview.TextView
	app          *tview.Application
	pages        *tview.Pages
}

// SetPages sets the pages reference for modal handling
func (v *AccountsView) SetPages(pages *tview.Pages) {
	v.pages = pages
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

	// Create filter input
	v.filterInput = tview.NewInputField().
		SetLabel("Filter: ").
		SetFieldWidth(30).
		SetChangedFunc(v.onFilterChange).
		SetDoneFunc(v.onFilterDone)

	// Create status bar
	v.statusBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	// Create container layout
	v.container = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(v.filterInput, 1, 0, false).
		AddItem(v.table.Table, 0, 1, true).
		AddItem(v.statusBar, 1, 0, false)

	return v
}

// Init initializes the accounts view
func (v *AccountsView) Init(ctx context.Context) error {
	v.BaseView.Init(ctx)
	// Don't refresh on init - let it happen when view is shown
	return nil
}

// Render returns the view's main component
func (v *AccountsView) Render() tview.Primitive {
	return v.container
}

// Refresh updates the accounts data
func (v *AccountsView) Refresh() error {
	v.SetRefreshing(true)
	defer v.SetRefreshing(false)

	// For now, return empty since Accounts manager isn't in the interface yet
	// TODO: Add Accounts manager to dao.SlurmClient interface
	v.mu.Lock()
	v.accounts = []*dao.Account{} // Empty for now
	v.mu.Unlock()

	// Update table
	v.updateTable()
	v.updateStatusBar("")

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
	return []string{
		"[yellow]Enter[white] Details",
		"[yellow]/[white] Filter",
		"[yellow]1-9[white] Sort",
		"[yellow]R[white] Refresh",
		"[yellow]h[white] Show Hierarchy",
	}
}

// OnKey handles keyboard events
func (v *AccountsView) OnKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyRune:
		switch event.Rune() {
		case 'R':
			go v.Refresh()
			return nil
		case '/':
			v.app.SetFocus(v.filterInput)
			return nil
		case 'h', 'H':
			v.showAccountHierarchy()
			return nil
		}
	case tcell.KeyEnter:
		v.showAccountDetails()
		return nil
	case tcell.KeyEsc:
		if v.filterInput.HasFocus() {
			v.app.SetFocus(v.table.Table)
			return nil
		}
	}

	return event
}

// OnFocus handles focus events
func (v *AccountsView) OnFocus() error {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
	// Refresh when gaining focus if we haven't loaded data yet
	if len(v.accounts) == 0 && !v.IsRefreshing() {
		go v.Refresh()
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

	data := make([][]string, len(v.accounts))
	for i, account := range v.accounts {
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

// updateStatusBar updates the status bar
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

// scheduleRefresh schedules the next refresh
func (v *AccountsView) scheduleRefresh() {
	// Remove automatic refresh scheduling to prevent memory leak
	// Refresh will be handled by the main app refresh timer
}

// onAccountSelect handles account selection
func (v *AccountsView) onAccountSelect(row, col int) {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	accountName := data[0]
	v.updateStatusBar(fmt.Sprintf("Selected account: %s", accountName))
}

// onSort handles column sorting
func (v *AccountsView) onSort(col int, ascending bool) {
	v.updateStatusBar(fmt.Sprintf("Sorted by column %d", col+1))
}

// onFilterChange handles filter input changes
func (v *AccountsView) onFilterChange(text string) {
	v.filter = text
	v.table.SetFilter(text)
	v.updateStatusBar("")
}

// onFilterDone handles filter input completion
func (v *AccountsView) onFilterDone(key tcell.Key) {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
}

// showAccountHierarchy shows the account hierarchy tree
func (v *AccountsView) showAccountHierarchy() {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if len(v.accounts) == 0 {
		v.updateStatusBar("[yellow]No accounts to display[white]")
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

	tree.WriteString(fmt.Sprintf("%s%s[green]%s[white]", prefix, connector, account.Name))
	if account.Description != "" {
		tree.WriteString(fmt.Sprintf(" (%s)", account.Description))
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
	if data == nil || len(data) == 0 {
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
		v.updateStatusBar(fmt.Sprintf("[red]Account %s not found[white]", accountName))
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