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

// ReservationsView displays the reservations list
type ReservationsView struct {
	*BaseView
	client       dao.SlurmClient
	table        *components.Table
	reservations []*dao.Reservation
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
func (v *ReservationsView) SetPages(pages *tview.Pages) {
	v.pages = pages
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

	// Create container layout
	v.container = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(v.filterInput, 1, 0, false).
		AddItem(v.table.Table, 0, 1, true).
		AddItem(v.statusBar, 1, 0, false)

	return v
}

// Init initializes the reservations view
func (v *ReservationsView) Init(ctx context.Context) error {
	v.BaseView.Init(ctx)
	// Don't refresh on init - let it happen when view is shown
	return nil
}

// Render returns the view's main component
func (v *ReservationsView) Render() tview.Primitive {
	return v.container
}

// Refresh updates the reservations data
func (v *ReservationsView) Refresh() error {
	v.SetRefreshing(true)
	defer v.SetRefreshing(false)

	// Fetch reservations from backend
	resList, err := v.client.Reservations().List()
	if err != nil {
		v.SetLastError(err)
		v.updateStatusBar(fmt.Sprintf("[red]Error: %v[white]", err))
		return err
	}

	v.mu.Lock()
	v.reservations = resList.Reservations
	v.mu.Unlock()

	// Update table
	v.updateTable()
	v.updateStatusBar("")

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
	return []string{
		"[yellow]Enter[white] Details",
		"[yellow]/[white] Filter",
		"[yellow]1-9[white] Sort",
		"[yellow]R[white] Refresh",
		"[yellow]a[white] Active Only",
		"[yellow]f[white] Future Only",
	}
}

// OnKey handles keyboard events
func (v *ReservationsView) OnKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyRune:
		switch event.Rune() {
		case 'R':
			go v.Refresh()
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
		go v.Refresh()
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

	data := make([][]string, 0, len(v.reservations))
	now := time.Now()

	for _, res := range v.reservations {
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

// updateStatusBar updates the status bar
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

// scheduleRefresh schedules the next refresh
func (v *ReservationsView) scheduleRefresh() {
	// Remove automatic refresh scheduling to prevent memory leak
	// Refresh will be handled by the main app refresh timer
}

// onReservationSelect handles reservation selection
func (v *ReservationsView) onReservationSelect(row, col int) {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
		return
	}

	resName := data[0]
	v.updateStatusBar(fmt.Sprintf("Selected reservation: %s", resName))
}

// onSort handles column sorting
func (v *ReservationsView) onSort(col int, ascending bool) {
	v.updateStatusBar(fmt.Sprintf("Sorted by column %d", col+1))
}

// onFilterChange handles filter input changes
func (v *ReservationsView) onFilterChange(text string) {
	v.filter = text
	v.table.SetFilter(text)
	v.updateStatusBar("")
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
	v.updateStatusBar("[yellow]Active filter not yet implemented[white]")
}

// toggleFutureFilter toggles showing only future reservations
func (v *ReservationsView) toggleFutureFilter() {
	// TODO: Implement future filter
	v.updateStatusBar("[yellow]Future filter not yet implemented[white]")
}

// showReservationDetails shows detailed information for the selected reservation
func (v *ReservationsView) showReservationDetails() {
	data := v.table.GetSelectedData()
	if data == nil || len(data) == 0 {
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
		v.updateStatusBar(fmt.Sprintf("[red]Reservation %s not found[white]", resName))
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