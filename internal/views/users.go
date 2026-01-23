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
	"github.com/rivo/tview"
)

// UsersView displays the users list
type UsersView struct {
	*BaseView
	client         dao.SlurmClient
	table          *components.Table
	users          []*dao.User
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
	loadingManager *components.LoadingManager
	loadingWrapper *components.LoadingWrapper
}

// SetPages sets the pages reference for modal handling
func (v *UsersView) SetPages(pages *tview.Pages) {
	v.pages = pages
	// Set pages for filter bar if it exists
	if v.filterBar != nil {
		v.filterBar.SetPages(pages)
	}
}

// SetApp sets the application reference
func (v *UsersView) SetApp(app *tview.Application) {
	v.app = app

	// Initialize loading manager
	if v.pages != nil {
		v.loadingManager = components.NewLoadingManager(app, v.pages)
		v.loadingWrapper = components.NewLoadingWrapper(v.loadingManager, "users")
	}

	// Create filter bar now that we have app reference
	v.filterBar = components.NewFilterBar("users", app)
	v.filterBar.SetPages(v.pages)
	v.filterBar.SetOnFilterChange(v.onAdvancedFilterChange)
	v.filterBar.SetOnClose(v.closeAdvancedFilter)

	// Create global search
	v.globalSearch = NewGlobalSearch(v.client, app)
}

// NewUsersView creates a new users view
func NewUsersView(client dao.SlurmClient) *UsersView {
	v := &UsersView{
		BaseView:    NewBaseView("users", "Users"),
		client:      client,
		refreshRate: 30 * time.Second,
		users:       []*dao.User{},
	}

	// Create table with user columns
	columns := []components.Column{
		components.NewColumn("Name").Width(15).Build(),
		components.NewColumn("UID").Width(8).Align(tview.AlignRight).Sortable(true).Build(),
		components.NewColumn("Default Account").Width(20).Build(),
		components.NewColumn("Admin Level").Width(12).Build(),
		components.NewColumn("Default QoS").Width(15).Build(),
		components.NewColumn("Max Jobs").Width(10).Align(tview.AlignRight).Sortable(true).Build(),
		components.NewColumn("Max Nodes").Width(10).Align(tview.AlignRight).Sortable(true).Build(),
		components.NewColumn("Max CPUs").Width(10).Align(tview.AlignRight).Sortable(true).Build(),
		components.NewColumn("Accounts").Width(40).Build(),
	}

	v.table = components.NewTableBuilder().
		WithColumns(columns...).
		WithSelectable(true).
		WithHeader(true).
		WithColors(tcell.ColorYellow, tcell.ColorTeal, tcell.ColorWhite).
		Build()

	// Set up callbacks
	v.table.SetOnSelect(v.onUserSelect)
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

	// Create container layout (removed individual status bar to prevent conflicts with main status bar)
	v.container = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(v.filterInput, 1, 0, false).
		AddItem(v.table, 0, 1, true)

	return v
}

// Init initializes the users view
func (v *UsersView) Init(ctx context.Context) error {
	_ = v.BaseView.Init(ctx)
	return v.Refresh()
}

// Render returns the view's main component
func (v *UsersView) Render() tview.Primitive {
	return v.container
}

// Refresh updates the users data
func (v *UsersView) Refresh() error {
	v.SetRefreshing(true)
	defer v.SetRefreshing(false)

	// Show loading indicator for operations that might take time
	if v.loadingWrapper != nil {
		return v.loadingWrapper.WithLoading("Loading users...", func() error {
			return v.refreshInternal()
		})
	}

	return v.refreshInternal()
}

// refreshInternal performs the actual refresh operation
func (v *UsersView) refreshInternal() error {
	// Fetch users from backend
	usersList, err := v.client.Users().List()
	if err != nil {
		v.SetLastError(err)
		// Note: Error handling removed since individual view status bars are no longer used
		return err
	}

	v.mu.Lock()
	v.users = usersList.Users
	v.mu.Unlock()

	// Update table
	v.updateTable()
	// Note: No longer updating individual view status bar since we use main app status bar for hints

	// Schedule next refresh
	v.scheduleRefresh()

	return nil
}

// Stop stops the view
func (v *UsersView) Stop() error {
	if v.refreshTimer != nil {
		v.refreshTimer.Stop()
	}
	return nil
}

// Hints returns keyboard hints
func (v *UsersView) Hints() []string {
	hints := []string{
		"[yellow]Enter[white] Details",
		"[yellow]/[white] Filter",
		"[yellow]F3[white] Adv Filter",
		"[yellow]Ctrl+F[white] Search",
		"[yellow]1-9[white] Sort",
		"[yellow]R[white] Refresh",
		"[yellow]a[white] Show Admins",
	}

	if v.isAdvancedMode {
		hints = append([]string{"[yellow]ESC[white] Exit Adv Filter"}, hints...)
	}

	return hints
}

// OnKey handles keyboard events
func (v *UsersView) OnKey(event *tcell.EventKey) *tcell.EventKey {
	// Check if a modal is open - if so, don't process view shortcuts
	if v.pages != nil && v.pages.GetPageCount() > 1 {
		return event // Let modal handle it
	}

	// Handle advanced filter mode
	if v.isAdvancedMode && event.Key() == tcell.KeyEsc {
		v.closeAdvancedFilter()
		return nil
	}

	switch event.Key() {
	case tcell.KeyF3:
		v.showAdvancedFilter()
		return nil
	case tcell.KeyCtrlF:
		v.showGlobalSearch()
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case 'R':
			go func() { _ = v.Refresh() }()
			return nil
		case '/':
			v.app.SetFocus(v.filterInput)
			return nil
		case 'a', 'A':
			v.toggleAdminFilter()
			return nil
		}
	case tcell.KeyEnter:
		v.showUserDetails()
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
func (v *UsersView) OnFocus() error {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
	// Refresh when gaining focus if we haven't loaded data yet
	if len(v.users) == 0 && !v.IsRefreshing() {
		go func() { _ = v.Refresh() }()
	}
	return nil
}

// OnLoseFocus handles loss of focus
func (v *UsersView) OnLoseFocus() error {
	return nil
}

// updateTable updates the table with current user data
func (v *UsersView) updateTable() {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Apply advanced filter if active
	filteredUsers := v.users
	if v.advancedFilter != nil && len(v.advancedFilter.Expressions) > 0 {
		filteredUsers = v.applyAdvancedFilter(v.users)
	}

	data := make([][]string, len(filteredUsers))
	for i, user := range filteredUsers {
		// Format limits
		maxJobs := formatLimit(user.MaxJobs)
		maxNodes := formatLimit(user.MaxNodes)
		maxCPUs := formatLimit(user.MaxCPUs)

		// Format UID
		uid := fmt.Sprintf("%d", user.UID)

		// Format admin level with color
		adminLevel := user.AdminLevel
		switch adminLevel {
		case "Administrator":
			adminLevel = fmt.Sprintf("[red]%s[white]", adminLevel)
		case "Operator":
			adminLevel = fmt.Sprintf("[yellow]%s[white]", adminLevel)
		}

		// Format accounts list
		accounts := strings.Join(user.Accounts, ", ")
		if len(accounts) > 39 {
			accounts = accounts[:36] + "..."
		}

		data[i] = []string{
			user.Name,
			uid,
			user.DefaultAccount,
			adminLevel,
			user.DefaultQoS,
			maxJobs,
			maxNodes,
			maxCPUs,
			accounts,
		}
	}

	v.table.SetData(data)
}

/*
TODO(lint): Review unused code - func (*UsersView).updateStatusBar is unused

updateStatusBar updates the status bar
func (v *UsersView) updateStatusBar(message string) {
	if message != "" {
		v.statusBar.SetText(message)
		return
	}

	v.mu.RLock()
	total := len(v.users)
	admins := 0
	operators := 0
	regular := 0

	for _, user := range v.users {
		switch user.AdminLevel {
		case "Administrator":
			admins++
		case "Operator":
			operators++
		default:
			regular++
		}
	}
	v.mu.RUnlock()

	filtered := len(v.table.GetFilteredData())

	status := fmt.Sprintf("Total: %d | Admins: %d | Operators: %d | Regular: %d",
		total, admins, operators, regular)

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
func (v *UsersView) scheduleRefresh() {
	// Remove automatic refresh scheduling to prevent memory leak
	// Refresh will be handled by the main app refresh timer
}

// onUserSelect handles user selection
func (v *UsersView) onUserSelect(row, col int) {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	// Note: Selection handling removed since individual view status bars are no longer used
	_ = data[0] // userName no longer used
}

// onSort handles column sorting
func (v *UsersView) onSort(col int, ascending bool) {
	// Note: Sort feedback removed since individual view status bars are no longer used
}

// onFilterChange handles filter input changes
func (v *UsersView) onFilterChange(text string) {
	v.filter = text
	v.table.SetFilter(text)
	// Note: Status bar update removed since individual view status bars are no longer used
}

// onFilterDone handles filter input completion
func (v *UsersView) onFilterDone(key tcell.Key) {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
}

// toggleAdminFilter toggles showing only admin users
func (v *UsersView) toggleAdminFilter() {
	// TODO: Implement admin filter
	// Note: Filter status removed since individual view status bars are no longer used
}

// showUserDetails shows detailed information for the selected user
func (v *UsersView) showUserDetails() {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	userName := data[0]

	// Find the full user object
	var user *dao.User
	v.mu.RLock()
	for _, u := range v.users {
		if u.Name == userName {
			user = u
			break
		}
	}
	v.mu.RUnlock()

	if user == nil {
		// Note: Error message removed since individual view status bars are no longer used
		return
	}

	// Create details view
	details := v.formatUserDetails(user)

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(details).
		SetScrollable(true)

	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, true).
		AddItem(tview.NewTextView().SetText("Press ESC to close"), 1, 0, false)

	modal.SetBorder(true).
		SetTitle(fmt.Sprintf(" User %s Details ", userName)).
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
				v.pages.RemovePage("user-details")
			}
			return nil
		}
		return event
	})

	if v.pages != nil {
		v.pages.AddPage("user-details", centeredModal, true, true)
	}
}

// formatUserDetails formats user details for display
func (v *UsersView) formatUserDetails(user *dao.User) string {
	var details strings.Builder

	// Basic information
	details.WriteString(fmt.Sprintf("[yellow]User Name:[white] %s\n", user.Name))
	details.WriteString(fmt.Sprintf("[yellow]UID:[white] %d\n", user.UID))

	// Admin level with color
	adminColor := "white"
	switch user.AdminLevel {
	case "Administrator":
		adminColor = "red"
	case "Operator":
		adminColor = "yellow"
	}
	details.WriteString(fmt.Sprintf("[yellow]Admin Level:[white] [%s]%s[white]\n", adminColor, user.AdminLevel))

	// Account information
	details.WriteString("\n[teal]Account Information:[white]\n")
	details.WriteString(fmt.Sprintf("[yellow]  Default Account:[white] %s\n", user.DefaultAccount))
	if len(user.Accounts) > 0 {
		details.WriteString("[yellow]  All Accounts:[white]\n")
		for _, acc := range user.Accounts {
			if acc == user.DefaultAccount {
				details.WriteString(fmt.Sprintf("    - %s [green](default)[white]\n", acc))
			} else {
				details.WriteString(fmt.Sprintf("    - %s\n", acc))
			}
		}
	}

	// QoS information
	details.WriteString("\n[teal]Quality of Service:[white]\n")
	details.WriteString(fmt.Sprintf("[yellow]  Default QoS:[white] %s\n", user.DefaultQoS))
	if len(user.QoSList) > 0 {
		details.WriteString("[yellow]  Available QoS:[white]\n")
		for _, qos := range user.QoSList {
			if qos == user.DefaultQoS {
				details.WriteString(fmt.Sprintf("    - %s [green](default)[white]\n", qos))
			} else {
				details.WriteString(fmt.Sprintf("    - %s\n", qos))
			}
		}
	}

	// Resource limits
	details.WriteString("\n[teal]Resource Limits:[white]\n")
	details.WriteString(fmt.Sprintf("[yellow]  Max Jobs:[white] %s\n", formatLimit(user.MaxJobs)))
	details.WriteString(fmt.Sprintf("[yellow]  Max Submit:[white] %s\n", formatLimit(user.MaxSubmit)))
	details.WriteString(fmt.Sprintf("[yellow]  Max Nodes:[white] %s\n", formatLimit(user.MaxNodes)))
	details.WriteString(fmt.Sprintf("[yellow]  Max CPUs:[white] %s\n", formatLimit(user.MaxCPUs)))

	// Current usage (placeholder - would need real data)
	details.WriteString("\n[teal]Current Usage:[white]\n")
	details.WriteString("[yellow]  Running Jobs:[white] N/A\n")
	details.WriteString("[yellow]  Pending Jobs:[white] N/A\n")
	details.WriteString("[yellow]  CPU Hours Used:[white] N/A\n")

	return details.String()
}

// showAdvancedFilter shows the advanced filter bar
func (v *UsersView) showAdvancedFilter() {
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
func (v *UsersView) closeAdvancedFilter() {
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
func (v *UsersView) onAdvancedFilterChange(filter *filters.Filter) {
	v.advancedFilter = filter
	v.updateTable()

	if filter != nil && len(filter.Expressions) > 0 {
		// Note: Filter status removed since individual view status bars are no longer used
	} else {
		// Note: Status bar update removed since individual view status bars are no longer used
	}
}

// applyAdvancedFilter applies the advanced filter to users list
func (v *UsersView) applyAdvancedFilter(users []*dao.User) []*dao.User {
	if v.advancedFilter == nil || len(v.advancedFilter.Expressions) == 0 {
		return users
	}

	var filtered []*dao.User
	for _, user := range users {
		// Convert user to map for filter evaluation
		userData := v.userToMap(user)
		if v.advancedFilter.Evaluate(userData) {
			filtered = append(filtered, user)
		}
	}

	return filtered
}

// userToMap converts a user to a map for filter evaluation
func (v *UsersView) userToMap(user *dao.User) map[string]interface{} {
	return map[string]interface{}{
		"Name":           user.Name,
		"UID":            user.UID,
		"DefaultAccount": user.DefaultAccount,
		"AdminLevel":     user.AdminLevel,
		"DefaultQoS":     user.DefaultQoS,
		"MaxJobs":        user.MaxJobs,
		"MaxSubmit":      user.MaxSubmit,
		"MaxNodes":       user.MaxNodes,
		"MaxCPUs":        user.MaxCPUs,
		"Accounts":       strings.Join(user.Accounts, ","),
		"QoSList":        strings.Join(user.QoSList, ","),
	}
}

// showGlobalSearch shows the global search interface
func (v *UsersView) showGlobalSearch() {
	if v.globalSearch == nil || v.pages == nil {
		return
	}

	v.globalSearch.Show(v.pages, func(result SearchResult) {
		// Handle search result selection
		switch result.Type {
		case "user":
			// Focus on the selected user
			if user, ok := result.Data.(*dao.User); ok {
				v.focusOnUser(user.Name)
			}
		default:
			// For other types, just close the search
			// Note: Search result status removed since individual view status bars are no longer used
		}
	})
}

// focusOnUser focuses the table on a specific user
func (v *UsersView) focusOnUser(userName string) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Find the user in our user list
	for i, user := range v.users {
		if user.Name == userName {
			// Select the row in the table
			v.table.Select(i, 0)
			// Note: Focus status removed since individual view status bars are no longer used
			return
		}
	}

	// Note: Error message removed since individual view status bars are no longer used
}
