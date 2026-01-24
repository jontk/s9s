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

// ReservationsView displays the reservations list
type ReservationsView struct {
	*BaseView
	client         dao.SlurmClient
	table          *components.Table
	reservations   []*dao.Reservation
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
func (v *ReservationsView) SetPages(pages *tview.Pages) {
	v.pages = pages
	// Set pages for filter bar if it exists
	if v.filterBar != nil {
		v.filterBar.SetPages(pages)
	}
}

// SetApp sets the application reference
func (v *ReservationsView) SetApp(app *tview.Application) {
	v.app = app

	// Initialize loading manager
	if v.pages != nil {
		v.loadingManager = components.NewLoadingManager(app, v.pages)
		v.loadingWrapper = components.NewLoadingWrapper(v.loadingManager, "reservations")
	}

	// Create filter bar now that we have app reference
	v.filterBar = components.NewFilterBar("reservations", app)
	v.filterBar.SetPages(v.pages)
	v.filterBar.SetOnFilterChange(v.onAdvancedFilterChange)
	v.filterBar.SetOnClose(v.closeAdvancedFilter)

	// Create global search
	v.globalSearch = NewGlobalSearch(v.client, app)
}

// NewReservationsView creates a new reservations view
func NewReservationsView(client dao.SlurmClient) *ReservationsView {
	v := &ReservationsView{
		BaseView:     NewBaseView("reservations", "Reservations"),
		client:       client,
		refreshRate:  30 * time.Second,
		reservations: []*dao.Reservation{},
	}

	// Create table with reservation columns
	columns := []components.Column{
		components.NewColumn("Name").Width(20).Build(),
		components.NewColumn("State").Width(12).Sortable(true).Build(),
		components.NewColumn("Start Time").Width(20).Sortable(true).Build(),
		components.NewColumn("End Time").Width(20).Sortable(true).Build(),
		components.NewColumn("Duration").Width(12).Align(tview.AlignRight).Build(),
		components.NewColumn("Nodes").Width(10).Align(tview.AlignRight).Sortable(true).Build(),
		components.NewColumn("Cores").Width(10).Align(tview.AlignRight).Sortable(true).Build(),
		components.NewColumn("Users").Width(30).Build(),
		components.NewColumn("Accounts").Width(20).Build(),
	}

	v.table = components.NewTableBuilder().
		WithColumns(columns...).
		WithSelectable(true).
		WithHeader(true).
		WithColors(tcell.ColorYellow, tcell.ColorTeal, tcell.ColorWhite).
		Build()

	// Set up callbacks
	v.table.SetOnSelect(v.onReservationSelect)
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

// Init initializes the reservations view
func (v *ReservationsView) Init(ctx context.Context) error {
	_ = v.BaseView.Init(ctx)
	return v.Refresh()
}

// Render returns the view's main component
func (v *ReservationsView) Render() tview.Primitive {
	return v.container
}

// Refresh updates the reservations data
func (v *ReservationsView) Refresh() error {
	v.SetRefreshing(true)
	defer v.SetRefreshing(false)

	// Show loading indicator for operations that might take time
	if v.loadingWrapper != nil {
		return v.loadingWrapper.WithLoading("Loading reservations...", func() error {
			return v.refreshInternal()
		})
	}

	return v.refreshInternal()
}

// refreshInternal performs the actual refresh operation
func (v *ReservationsView) refreshInternal() error {
	// Fetch reservations from backend
	resList, err := v.client.Reservations().List()
	if err != nil {
		v.SetLastError(err)
		// Note: Error handling removed since individual view status bars are no longer used
		return err
	}

	v.mu.Lock()
	v.reservations = resList.Reservations
	v.mu.Unlock()

	// Update table
	v.updateTable()
	// Note: No longer updating individual view status bar since we use main app status bar for hints

	// Schedule next refresh
	v.scheduleRefresh()

	return nil
}

// Stop stops the view
func (v *ReservationsView) Stop() error {
	if v.refreshTimer != nil {
		v.refreshTimer.Stop()
	}
	return nil
}

// Hints returns keyboard hints
func (v *ReservationsView) Hints() []string {
	hints := []string{
		"[yellow]Enter[white] Details",
		"[yellow]/[white] Filter",
		"[yellow]F3[white] Adv Filter",
		"[yellow]Ctrl+F[white] Search",
		"[yellow]1-9[white] Sort",
		"[yellow]R[white] Refresh",
		"[yellow]a[white] Active Only",
		"[yellow]f[white] Future Only",
	}

	if v.isAdvancedMode {
		hints = append([]string{"[yellow]ESC[white] Exit Adv Filter"}, hints...)
	}

	return hints
}

// OnKey handles keyboard events
func (v *ReservationsView) OnKey(event *tcell.EventKey) *tcell.EventKey {
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
			v.toggleActiveFilter()
			return nil
		case 'f', 'F':
			v.toggleFutureFilter()
			return nil
		}
	case tcell.KeyEnter:
		v.showReservationDetails()
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
func (v *ReservationsView) OnFocus() error {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
	// Refresh when gaining focus if we haven't loaded data yet
	if len(v.reservations) == 0 && !v.IsRefreshing() {
		go func() { _ = v.Refresh() }()
	}
	return nil
}

// OnLoseFocus handles loss of focus
func (v *ReservationsView) OnLoseFocus() error {
	return nil
}

// updateTable updates the table with current reservation data
func (v *ReservationsView) updateTable() {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Apply advanced filter if active
	filteredReservations := v.reservations
	if v.advancedFilter != nil && len(v.advancedFilter.Expressions) > 0 {
		filteredReservations = v.applyAdvancedFilter(v.reservations)
	}

	data := make([][]string, 0, len(filteredReservations))
	now := time.Now()

	for _, res := range filteredReservations {
		// Determine state color
		stateColor := getReservationStateColor(res.State, res.StartTime, res.EndTime, now)
		coloredState := fmt.Sprintf("[%s]%s[white]", stateColor, res.State)

		// Format times
		startTime := res.StartTime.Format("2006-01-02 15:04:05")
		endTime := res.EndTime.Format("2006-01-02 15:04:05")

		// Format duration
		duration := formatReservationDuration(res.Duration)

		// Format nodes
		nodes := fmt.Sprintf("%d", res.NodeCount)
		if len(res.Nodes) > 0 && len(res.Nodes) < 5 {
			nodes = strings.Join(res.Nodes, ",")
		}

		// Format users and accounts
		users := strings.Join(res.Users, ",")
		if len(users) > 29 {
			users = users[:26] + "..."
		}

		accounts := strings.Join(res.Accounts, ",")
		if len(accounts) > 19 {
			accounts = accounts[:16] + "..."
		}

		data = append(data, []string{
			res.Name,
			coloredState,
			startTime,
			endTime,
			duration,
			nodes,
			fmt.Sprintf("%d", res.CoreCount),
			users,
			accounts,
		})
	}

	v.table.SetData(data)
}

// formatReservationDuration formats a time.Duration into a readable string
func formatReservationDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// getReservationStateColor returns the color for a reservation based on its state
func getReservationStateColor(state string, start, end, now time.Time) string {
	switch state {
	case "ACTIVE":
		if now.After(start) && now.Before(end) {
			return "green"
		}
		return "blue"
	case "INACTIVE":
		return "gray"
	default:
		// Check if it's future or past
		if now.Before(start) {
			return "yellow" // Future
		} else if now.After(end) {
			return "gray" // Past
		}
		return "white"
	}
}

/*
TODO(lint): Review unused code - func (*ReservationsView).updateStatusBar is unused

updateStatusBar updates the status bar
func (v *ReservationsView) updateStatusBar(message string) {
	if message != "" {
		v.statusBar.SetText(message)
		return
	}

	v.mu.RLock()
	total := len(v.reservations)
	active := 0
	future := 0
	past := 0
	now := time.Now()

	for _, res := range v.reservations {
		if now.After(res.StartTime) && now.Before(res.EndTime) {
			active++
		} else if now.Before(res.StartTime) {
			future++
		} else {
			past++
		}
	}
	v.mu.RUnlock()

	filtered := len(v.table.GetFilteredData())

	status := fmt.Sprintf("Total: %d | [green]Active: %d[white] | [yellow]Future: %d[white] | [gray]Past: %d[white]",
		total, active, future, past)

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
func (v *ReservationsView) scheduleRefresh() {
	// Remove automatic refresh scheduling to prevent memory leak
	// Refresh will be handled by the main app refresh timer
}

// onReservationSelect handles reservation selection
func (v *ReservationsView) onReservationSelect(row, col int) {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		return
	}

	// Note: Selection handling removed since individual view status bars are no longer used
	_ = data[0] // resName no longer used
}

// onSort handles column sorting
func (v *ReservationsView) onSort(col int, ascending bool) {
	// Note: Sort feedback removed since individual view status bars are no longer used
}

// onFilterChange handles filter input changes
func (v *ReservationsView) onFilterChange(text string) {
	v.filter = text
	v.table.SetFilter(text)
	// Note: Status bar update removed since individual view status bars are no longer used
}

// onFilterDone handles filter input completion
func (v *ReservationsView) onFilterDone(key tcell.Key) {
	if v.app != nil {
		v.app.SetFocus(v.table.Table)
	}
}

// toggleActiveFilter toggles showing only active reservations
func (v *ReservationsView) toggleActiveFilter() {
	// TODO: Implement active filter
	// Note: Filter status removed since individual view status bars are no longer used
}

// toggleFutureFilter toggles showing only future reservations
func (v *ReservationsView) toggleFutureFilter() {
	// TODO: Implement future filter
	// Note: Filter status removed since individual view status bars are no longer used
}

// showReservationDetails shows detailed information for the selected reservation
func (v *ReservationsView) showReservationDetails() {
	data := v.table.GetSelectedData()
	if len(data) == 0 {
		return
	}

	resName := data[0]

	// Find the full reservation object
	var reservation *dao.Reservation
	v.mu.RLock()
	for _, res := range v.reservations {
		if res.Name == resName {
			reservation = res
			break
		}
	}
	v.mu.RUnlock()

	if reservation == nil {
		// Note: Error message removed since individual view status bars are no longer used
		return
	}

	// Create details view
	details := v.formatReservationDetails(reservation)

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(details).
		SetScrollable(true)

	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, true).
		AddItem(tview.NewTextView().SetText("Press ESC to close"), 1, 0, false)

	modal.SetBorder(true).
		SetTitle(fmt.Sprintf(" Reservation %s Details ", resName)).
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
				v.pages.RemovePage("reservation-details")
			}
			return nil
		}
		return event
	})

	if v.pages != nil {
		v.pages.AddPage("reservation-details", centeredModal, true, true)
	}
}

// formatReservationDetails formats reservation details for display
func (v *ReservationsView) formatReservationDetails(res *dao.Reservation) string {
	var details strings.Builder
	now := time.Now()

	details.WriteString(fmt.Sprintf("[yellow]Reservation Name:[white] %s\n", res.Name))

	stateColor := getReservationStateColor(res.State, res.StartTime, res.EndTime, now)
	details.WriteString(fmt.Sprintf("[yellow]State:[white] [%s]%s[white]\n", stateColor, res.State))

	// Time information
	details.WriteString("\n[teal]Time Information:[white]\n")
	details.WriteString(fmt.Sprintf("[yellow]  Start Time:[white] %s\n", res.StartTime.Format("2006-01-02 15:04:05")))
	details.WriteString(fmt.Sprintf("[yellow]  End Time:[white] %s\n", res.EndTime.Format("2006-01-02 15:04:05")))
	details.WriteString(fmt.Sprintf("[yellow]  Duration:[white] %s\n", formatReservationDuration(res.Duration)))

	// Time status
	if now.Before(res.StartTime) {
		timeUntil := res.StartTime.Sub(now)
		details.WriteString(fmt.Sprintf("[yellow]  Status:[white] Starts in %s\n", formatReservationDuration(timeUntil)))
	} else if now.After(res.EndTime) {
		timeSince := now.Sub(res.EndTime)
		details.WriteString(fmt.Sprintf("[yellow]  Status:[white] Ended %s ago\n", formatReservationDuration(timeSince)))
	} else {
		timeLeft := res.EndTime.Sub(now)
		details.WriteString(fmt.Sprintf("[yellow]  Status:[white] Active, %s remaining\n", formatReservationDuration(timeLeft)))
	}

	// Resource information
	details.WriteString("\n[teal]Resources:[white]\n")
	details.WriteString(fmt.Sprintf("[yellow]  Node Count:[white] %d\n", res.NodeCount))
	details.WriteString(fmt.Sprintf("[yellow]  Core Count:[white] %d\n", res.CoreCount))

	if len(res.Nodes) > 0 {
		details.WriteString(fmt.Sprintf("[yellow]  Nodes:[white] %s\n", strings.Join(res.Nodes, ", ")))
	}

	// Access information
	details.WriteString("\n[teal]Access Information:[white]\n")
	if len(res.Users) > 0 {
		details.WriteString(fmt.Sprintf("[yellow]  Users:[white] %s\n", strings.Join(res.Users, ", ")))
	}
	if len(res.Accounts) > 0 {
		details.WriteString(fmt.Sprintf("[yellow]  Accounts:[white] %s\n", strings.Join(res.Accounts, ", ")))
	}

	return details.String()
}

// showAdvancedFilter shows the advanced filter bar
func (v *ReservationsView) showAdvancedFilter() {
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
func (v *ReservationsView) closeAdvancedFilter() {
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
func (v *ReservationsView) onAdvancedFilterChange(filter *filters.Filter) {
	v.advancedFilter = filter
	v.updateTable()

	// Note: Status bar updates removed since individual view status bars are no longer used
}

// applyAdvancedFilter applies the advanced filter to reservations
func (v *ReservationsView) applyAdvancedFilter(reservations []*dao.Reservation) []*dao.Reservation {
	if v.advancedFilter == nil || len(v.advancedFilter.Expressions) == 0 {
		return reservations
	}

	var filtered []*dao.Reservation
	for _, reservation := range reservations {
		// Convert reservation to map for filter evaluation
		reservationData := v.reservationToMap(reservation)
		if v.advancedFilter.Evaluate(reservationData) {
			filtered = append(filtered, reservation)
		}
	}

	return filtered
}

// reservationToMap converts a reservation to a map for filter evaluation
func (v *ReservationsView) reservationToMap(reservation *dao.Reservation) map[string]interface{} {
	return map[string]interface{}{
		"Name":      reservation.Name,
		"State":     reservation.State,
		"StartTime": reservation.StartTime.Format("2006-01-02 15:04:05"),
		"EndTime":   reservation.EndTime.Format("2006-01-02 15:04:05"),
		"Duration":  reservation.Duration.String(),
		"NodeCount": reservation.NodeCount,
		"CoreCount": reservation.CoreCount,
		"Users":     strings.Join(reservation.Users, ","),
		"Accounts":  strings.Join(reservation.Accounts, ","),
		"Nodes":     strings.Join(reservation.Nodes, ","),
	}
}

// showGlobalSearch shows the global search interface
func (v *ReservationsView) showGlobalSearch() {
	if v.globalSearch == nil || v.pages == nil {
		return
	}

	v.globalSearch.Show(v.pages, func(result SearchResult) {
		// Handle search result selection
		switch result.Type {
		case "reservation":
			// Focus on the selected reservation
			if reservation, ok := result.Data.(*dao.Reservation); ok {
				v.focusOnReservation(reservation.Name)
			}
		default:
			// For other types, just close the search
			// Note: Search result status removed since individual view status bars are no longer used
		}
	})
}

// focusOnReservation focuses the table on a specific reservation
func (v *ReservationsView) focusOnReservation(reservationName string) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	// Find the reservation in our reservation list
	for i, reservation := range v.reservations {
		if reservation.Name == reservationName {
			// Select the row in the table
			v.table.Select(i, 0)
			// Note: Focus status removed since individual view status bars are no longer used
			return
		}
	}

	// Note: Error message removed since individual view status bars are no longer used
}
