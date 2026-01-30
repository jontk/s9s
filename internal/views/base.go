package views

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/debug"
	"github.com/rivo/tview"
)

// ModalHandler provides methods for showing and hiding modals
type ModalHandler interface {
	ShowModal(name string, modal tview.Primitive)
	HideModal(name string)
}

// View represents a base interface for all views in S9s
type View interface {
	// Name returns the unique name of the view (e.g., "jobs", "nodes")
	Name() string

	// Title returns the display title for the view
	Title() string

	// Hints returns keyboard shortcut hints for the status bar
	Hints() []string

	// Init initializes the view with the given context
	Init(ctx context.Context) error

	// Render returns the tview primitive to be displayed
	Render() tview.Primitive

	// Refresh updates the view data from the backend
	Refresh() error

	// OnKey handles keyboard events, returns nil if handled
	OnKey(event *tcell.EventKey) *tcell.EventKey

	// OnFocus is called when the view gains focus
	OnFocus() error

	// OnLoseFocus is called when the view loses focus
	OnLoseFocus() error

	// Stop cleanly shuts down the view
	Stop() error

	// SetSwitchViewFn sets the callback function to switch to another view
	SetSwitchViewFn(func(string))

	// SwitchToView switches to another view
	SwitchToView(string)
}

// BaseView provides common functionality for all views
type BaseView struct {
	ctx         context.Context
	name        string
	title       string
	app         *tview.Application
	pages       *tview.Pages
	viewMgr     *ViewManager
	switchViewFn func(string) // Callback to switch to a view
	refreshing  bool
	lastError   error
}

// NewBaseView creates a new base view instance
func NewBaseView(name, title string) *BaseView {
	return &BaseView{
		name:  name,
		title: title,
	}
}

// Name returns the view name
func (v *BaseView) Name() string {
	return v.name
}

// Title returns the view title
func (v *BaseView) Title() string {
	return v.title
}

// Hints returns keyboard shortcut hints for the status bar (default implementation)
func (v *BaseView) Hints() []string {
	return []string{}
}

// Init initializes the base view
func (v *BaseView) Init(ctx context.Context) error {
	v.ctx = ctx
	return nil
}

// SetApp sets the tview application reference
func (v *BaseView) SetApp(app *tview.Application) {
	v.app = app
}

// GetApp returns the tview application reference
func (v *BaseView) GetApp() *tview.Application {
	return v.app
}

// SetViewManager sets the view manager reference
func (v *BaseView) SetViewManager(viewMgr *ViewManager) {
	v.viewMgr = viewMgr
}

// GetViewManager returns the view manager reference
func (v *BaseView) GetViewManager() *ViewManager {
	return v.viewMgr
}

// SetSwitchViewFn sets the callback function to switch views
func (v *BaseView) SetSwitchViewFn(fn func(string)) {
	v.switchViewFn = fn
}

// SwitchToView switches to another view using the registered callback
func (v *BaseView) SwitchToView(viewName string) {
	debug.Logger.Printf("[BaseView] SwitchToView called: currentView=%s, targetView=%s, hasFn=%v\n", v.name, viewName, v.switchViewFn != nil)
	if v.switchViewFn != nil {
		debug.Logger.Printf("[BaseView] Calling switchViewFn\n")
		v.switchViewFn(viewName)
		debug.Logger.Printf("[BaseView] switchViewFn returned\n")
	} else {
		debug.Logger.Printf("[BaseView] ERROR: switchViewFn is nil!\n")
	}
}

// IsRefreshing returns true if the view is currently refreshing
func (v *BaseView) IsRefreshing() bool {
	return v.refreshing
}

// SetRefreshing sets the refreshing state
func (v *BaseView) SetRefreshing(refreshing bool) {
	v.refreshing = refreshing
}

// GetLastError returns the last error that occurred
func (v *BaseView) GetLastError() error {
	return v.lastError
}

// SetLastError sets the last error
func (v *BaseView) SetLastError(err error) {
	v.lastError = err
}

// OnFocus provides default focus handling
func (v *BaseView) OnFocus() error {
	return nil
}

// OnLoseFocus provides default focus loss handling
func (v *BaseView) OnLoseFocus() error {
	return nil
}

// OnKey provides default key handling
func (v *BaseView) OnKey(event *tcell.EventKey) *tcell.EventKey {
	return event
}

// Render provides default render (must be overridden)
func (v *BaseView) Render() tview.Primitive {
	return tview.NewTextView().SetText("Not implemented")
}

// Refresh provides default refresh (must be overridden)
func (v *BaseView) Refresh() error {
	return nil
}

// Stop provides default stop implementation
func (v *BaseView) Stop() error {
	return nil
}

// ColoredState returns a colored string based on state type
func ColoredState(state, stateType string) string {
	var color string

	switch stateType {
	case "job":
		color = GetJobStateColor(state)
	case "node":
		color = GetNodeStateColor(state)
	case "partition":
		color = GetPartitionStateColor(state)
	default:
		color = "white"
	}

	return fmt.Sprintf("[%s]%s[white]", color, state)
}

// jobStateColors maps job states to their display colors
var jobStateColors = map[string]string{
	"RUNNING":     "green",
	"PENDING":     "yellow",
	"COMPLETED":   "cyan",
	"FAILED":      "red",
	"CANCELED":    "gray",
	"SUSPENDED":   "orange",
	"COMPLETING":  "blue",
	"CONFIGURING": "yellow",
	"PREEMPTED":   "orange",
	"TIMEOUT":     "red",
}

// nodeStateColors maps node states to their display colors
var nodeStateColors = map[string]string{
	"IDLE":        "green",
	"ALLOCATED":   "blue",
	"MIXED":       "blue",
	"DOWN":        "red",
	"DRAIN":       "red",
	"DRAINING":    "red",
	"RESERVED":    "yellow",
	"MAINTENANCE": "orange",
}

// partitionStateColors maps partition states to their display colors
var partitionStateColors = map[string]string{
	"UP":       "green",
	"DOWN":     "red",
	"DRAIN":    "orange",
	"INACTIVE": "gray",
}

// GetJobStateColor returns the color for a job state
func GetJobStateColor(state string) string {
	if color, exists := jobStateColors[state]; exists {
		return color
	}
	return "white"
}

// GetNodeStateColor returns the color for a node state
func GetNodeStateColor(state string) string {
	if color, exists := nodeStateColors[state]; exists {
		return color
	}
	return "white"
}

// GetPartitionStateColor returns the color for a partition state
func GetPartitionStateColor(state string) string {
	if color, exists := partitionStateColors[state]; exists {
		return color
	}
	return "white"
}

// FormatDuration formats a duration into a human-readable string
func FormatDuration(d string) string {
	// Handle SLURM time formats like "1-12:30:00" (1 day, 12 hours, 30 minutes)
	// or "12:30:00" (12 hours, 30 minutes)
	// For now, return as-is, but this can be enhanced
	return d
}

// FormatTimeDuration formats a time.Duration for display
func FormatTimeDuration(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%.0fs", d.Seconds())
	case d < time.Hour:
		return fmt.Sprintf("%.0fm", d.Minutes())
	case d < 24*time.Hour:
		return fmt.Sprintf("%.1fh", d.Hours())
	default:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	}
}

// FormatMemory formats memory size in MB to human-readable format
func FormatMemory(mb int64) string {
	if mb < 1024 {
		return fmt.Sprintf("%dMB", mb)
	}
	return fmt.Sprintf("%.1fGB", float64(mb)/1024)
}

// FormatCPUUsage formats CPU usage percentage
func FormatCPUUsage(used, total int) string {
	if total == 0 {
		return "0%"
	}
	percentage := float64(used) * 100 / float64(total)
	return fmt.Sprintf("%.1f%%", percentage)
}

// TruncateString truncates a string to the specified length with ellipsis
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// ViewManager manages multiple views and handles switching between them
type ViewManager struct {
	views       map[string]View
	currentView string
	viewOrder   []string
	app         *tview.Application
	pages       *tview.Pages
}

// NewViewManager creates a new view manager
func NewViewManager(app *tview.Application) *ViewManager {
	return &ViewManager{
		views:     make(map[string]View),
		app:       app,
		viewOrder: []string{},
	}
}

// SetPages sets the pages reference for modal handling
func (vm *ViewManager) SetPages(pages *tview.Pages) {
	vm.pages = pages
}

// AddView adds a view to the manager
func (vm *ViewManager) AddView(view View) error {
	name := view.Name()
	if _, exists := vm.views[name]; exists {
		return fmt.Errorf("view %s already exists", name)
	}

	vm.views[name] = view
	vm.viewOrder = append(vm.viewOrder, name)

	vm.setViewReferences(view)
	return nil
}

// setViewReferences sets app and pages references on views that support it
func (vm *ViewManager) setViewReferences(view View) {
	// Set BaseView reference
	view.(interface{ SetApp(*tview.Application) }).SetApp(vm.app)

	// Set app and pages fields on view if they exist
	rv := reflect.ValueOf(view).Elem()

	// Set app field
	if appField := rv.FieldByName("app"); appField.IsValid() && appField.CanSet() {
		appField.Set(reflect.ValueOf(vm.app))
	}

	// Set pages field - handle both unconditional and conditional assignment
	if pagesField := rv.FieldByName("pages"); pagesField.IsValid() && pagesField.CanSet() {
		// For PerformanceView, only set if nil
		if pagesField.IsNil() {
			pagesField.Set(reflect.ValueOf(vm.pages))
		}
	}

	// Set viewMgr field if view has it
	if viewMgrField := rv.FieldByName("viewMgr"); viewMgrField.IsValid() && viewMgrField.CanSet() {
		viewMgrField.Set(reflect.ValueOf(vm))
	}
}

// GetView returns a view by name
func (vm *ViewManager) GetView(name string) (View, error) {
	view, exists := vm.views[name]
	if !exists {
		return nil, fmt.Errorf("view %s not found", name)
	}
	return view, nil
}

// GetCurrentView returns the current active view
func (vm *ViewManager) GetCurrentView() (View, error) {
	if vm.currentView == "" {
		return nil, fmt.Errorf("no current view set")
	}
	return vm.GetView(vm.currentView)
}

// SetCurrentView sets the current active view
func (vm *ViewManager) SetCurrentView(name string) error {
	newView, err := vm.GetView(name)
	if err != nil {
		return err
	}

	// Call OnLoseFocus on the old view
	if vm.currentView != "" {
		if oldView, err := vm.GetView(vm.currentView); err == nil {
			_ = oldView.OnLoseFocus()
		}
	}

	// Set the new current view
	vm.currentView = name

	// Call OnFocus on the new view
	return newView.OnFocus()
}

// NextView switches to the next view in the order
func (vm *ViewManager) NextView() error {
	if len(vm.viewOrder) == 0 {
		return fmt.Errorf("no views available")
	}

	currentIndex := -1
	for i, name := range vm.viewOrder {
		if name == vm.currentView {
			currentIndex = i
			break
		}
	}

	nextIndex := (currentIndex + 1) % len(vm.viewOrder)
	return vm.SetCurrentView(vm.viewOrder[nextIndex])
}

// PreviousView switches to the previous view in the order
func (vm *ViewManager) PreviousView() error {
	if len(vm.viewOrder) == 0 {
		return fmt.Errorf("no views available")
	}

	currentIndex := -1
	for i, name := range vm.viewOrder {
		if name == vm.currentView {
			currentIndex = i
			break
		}
	}

	prevIndex := currentIndex - 1
	if prevIndex < 0 {
		prevIndex = len(vm.viewOrder) - 1
	}

	return vm.SetCurrentView(vm.viewOrder[prevIndex])
}

// GetViewNames returns all registered view names in order
func (vm *ViewManager) GetViewNames() []string {
	return vm.viewOrder
}

// RefreshCurrentView refreshes the current active view
func (vm *ViewManager) RefreshCurrentView() error {
	if vm.currentView == "" {
		return fmt.Errorf("no current view set")
	}

	view, err := vm.GetView(vm.currentView)
	if err != nil {
		return err
	}

	// Check if the view supports refreshing
	if refreshable, ok := view.(interface{ Refresh() error }); ok {
		return refreshable.Refresh()
	}

	return fmt.Errorf("current view does not support refreshing")
}

// StopAll stops all views
func (vm *ViewManager) StopAll() error {
	for _, view := range vm.views {
		if err := view.Stop(); err != nil {
			return fmt.Errorf("error stopping view %s: %w", view.Name(), err)
		}
	}
	return nil
}
