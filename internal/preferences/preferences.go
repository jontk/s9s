package preferences

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
)

// UserPreferences represents all user-configurable preferences
type UserPreferences struct {
	mu            *sync.RWMutex
	configPath    string
	General       GeneralPrefs         `json:"general"`
	Display       DisplayPrefs         `json:"display"`
	Colors        ColorPrefs           `json:"colors"`
	KeyBindings   map[string]string    `json:"key_bindings"`
	ViewSettings  map[string]ViewPrefs `json:"view_settings"`
	Filters       FilterPrefs          `json:"filters"`
	JobSubmission JobSubmissionPrefs   `json:"job_submission"`
	Alerts        AlertPrefs           `json:"alerts"`
	Performance   PerformancePrefs     `json:"performance"`
	Layouts       LayoutPrefs          `json:"layouts"`
	lastSaved     time.Time
	onChange      []func()
}

// GeneralPrefs contains general application preferences
type GeneralPrefs struct {
	AutoRefresh     bool   `json:"auto_refresh"`
	RefreshInterval string `json:"refresh_interval"` // e.g., "5s", "30s", "1m"
	Theme           string `json:"theme"`            // "default", "dark", "light"
	DateFormat      string `json:"date_format"`      // e.g., "2006-01-02 15:04:05"
	RelativeTime    bool   `json:"relative_time"`    // Show times as "5m ago"
	ConfirmOnExit   bool   `json:"confirm_on_exit"`
	ShowWelcome     bool   `json:"show_welcome"`
	DefaultView     string `json:"default_view"` // Starting view
	SaveWindowSize  bool   `json:"save_window_size"`
	Language        string `json:"language"` // For future i18n
}

// DisplayPrefs contains display-related preferences
type DisplayPrefs struct {
	ShowHeader        bool   `json:"show_header"`
	ShowStatusBar     bool   `json:"show_status_bar"`
	ShowLineNumbers   bool   `json:"show_line_numbers"`
	CompactMode       bool   `json:"compact_mode"`
	ShowGridLines     bool   `json:"show_grid_lines"`
	AnimationsEnabled bool   `json:"animations_enabled"`
	HighlightChanges  bool   `json:"highlight_changes"`
	TruncateLongText  bool   `json:"truncate_long_text"`
	MaxColumnWidth    int    `json:"max_column_width"`
	TimeZone          string `json:"timezone"` // Local, UTC, or specific TZ
}

// ColorPrefs contains color scheme preferences
type ColorPrefs struct {
	Scheme          string            `json:"scheme"` // "default", "solarized", "monokai", etc.
	HighContrast    bool              `json:"high_contrast"`
	ColorBlindMode  string            `json:"colorblind_mode"` // "", "protanopia", "deuteranopia", "tritanopia"
	CustomColors    map[string]string `json:"custom_colors"`
	SyntaxHighlight bool              `json:"syntax_highlight"`
}

// ViewPrefs contains view-specific preferences
type ViewPrefs struct {
	SortColumn       string         `json:"sort_column"`
	SortOrder        string         `json:"sort_order"` // "asc" or "desc"
	VisibleColumns   []string       `json:"visible_columns"`
	ColumnWidths     map[string]int `json:"column_widths"`
	GroupBy          string         `json:"group_by"`
	ShowDetails      bool           `json:"show_details"`
	AutoExpandGroups bool           `json:"auto_expand_groups"`
	PageSize         int            `json:"page_size"`
}

// FilterPrefs contains filter-related preferences
type FilterPrefs struct {
	SaveHistory     bool     `json:"save_history"`
	HistorySize     int      `json:"history_size"`
	DefaultOperator string   `json:"default_operator"`
	CaseSensitive   bool     `json:"case_sensitive"`
	UseRegex        bool     `json:"use_regex"`
	ShowAdvanced    bool     `json:"show_advanced"`
	QuickFilters    []string `json:"quick_filters"`
}

// JobSubmissionPrefs contains job submission preferences
type JobSubmissionPrefs struct {
	DefaultTemplate     string            `json:"default_template"`
	SaveHistory         bool              `json:"save_history"`
	HistorySize         int               `json:"history_size"`
	DefaultValues       map[string]string `json:"default_values"`
	ValidateOnType      bool              `json:"validate_on_type"`
	ShowAdvancedOptions bool              `json:"show_advanced_options"`
	AutoSuggest         bool              `json:"auto_suggest"`
}

// AlertPrefs contains alert-related preferences
type AlertPrefs struct {
	ShowBadge        bool   `json:"show_badge"`
	BadgePosition    string `json:"badge_position"` // "top-right", "top-left", etc.
	AutoDismissInfo  bool   `json:"auto_dismiss_info"`
	InfoDismissTime  string `json:"info_dismiss_time"` // e.g., "5s"
	PlaySound        bool   `json:"play_sound"`
	FlashWindow      bool   `json:"flash_window"`
	ShowDesktopNotif bool   `json:"show_desktop_notif"`
}

// PerformancePrefs contains performance-related preferences
type PerformancePrefs struct {
	LazyLoading      bool   `json:"lazy_loading"`
	CacheSize        int    `json:"cache_size_mb"`
	MaxConcurrentReq int    `json:"max_concurrent_requests"`
	RequestTimeout   string `json:"request_timeout"`
	EnableProfiling  bool   `json:"enable_profiling"`
	DebugMode        bool   `json:"debug_mode"`
}

// LayoutPrefs contains dashboard layout preferences
type LayoutPrefs struct {
	CurrentLayout      string                 `json:"current_layout"`
	AutoSave           bool                   `json:"auto_save"`
	CustomLayouts      []string               `json:"custom_layouts"`
	LayoutHistory      []string               `json:"layout_history"`
	ResponsiveMode     bool                   `json:"responsive_mode"`
	GridSnap           bool                   `json:"grid_snap"`
	ShowGrid           bool                   `json:"show_grid"`
	AnimateTransitions bool                   `json:"animate_transitions"`
	WidgetSettings     map[string]WidgetPrefs `json:"widget_settings"`
}

// WidgetPrefs contains widget-specific preferences
type WidgetPrefs struct {
	Enabled      bool                   `json:"enabled"`
	UpdateRate   string                 `json:"update_rate"`
	ShowBorder   bool                   `json:"show_border"`
	ShowTitle    bool                   `json:"show_title"`
	ColorScheme  string                 `json:"color_scheme"`
	CustomConfig map[string]interface{} `json:"custom_config"`
}

// NewUserPreferences creates a new preferences manager
func NewUserPreferences(configPath string) (*UserPreferences, error) {
	up := &UserPreferences{
		mu:           &sync.RWMutex{},
		configPath:   configPath,
		KeyBindings:  make(map[string]string),
		ViewSettings: make(map[string]ViewPrefs),
		onChange:     make([]func(), 0),
	}

	// Set defaults
	up.setDefaults()

	// Try to load existing preferences
	if err := up.Load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load preferences: %w", err)
	}

	return up, nil
}

// setDefaults sets default preference values
func (up *UserPreferences) setDefaults() {
	up.General = GeneralPrefs{
		AutoRefresh:     true,
		RefreshInterval: "30s",
		Theme:           "default",
		DateFormat:      "2006-01-02 15:04:05",
		RelativeTime:    true,
		ConfirmOnExit:   true,
		ShowWelcome:     true,
		DefaultView:     "jobs",
		SaveWindowSize:  true,
		Language:        "en",
	}

	up.Display = DisplayPrefs{
		ShowHeader:        true,
		ShowStatusBar:     true,
		ShowLineNumbers:   false,
		CompactMode:       false,
		ShowGridLines:     true,
		AnimationsEnabled: true,
		HighlightChanges:  true,
		TruncateLongText:  true,
		MaxColumnWidth:    50,
		TimeZone:          "Local",
	}

	up.Colors = ColorPrefs{
		Scheme:          "default",
		HighContrast:    false,
		ColorBlindMode:  "",
		CustomColors:    make(map[string]string),
		SyntaxHighlight: true,
	}

	up.Filters = FilterPrefs{
		SaveHistory:     true,
		HistorySize:     50,
		DefaultOperator: "contains",
		CaseSensitive:   false,
		UseRegex:        false,
		ShowAdvanced:    false,
		QuickFilters:    []string{},
	}

	up.JobSubmission = JobSubmissionPrefs{
		DefaultTemplate:     "",
		SaveHistory:         true,
		HistorySize:         20,
		DefaultValues:       make(map[string]string),
		ValidateOnType:      true,
		ShowAdvancedOptions: false,
		AutoSuggest:         true,
	}

	up.Alerts = AlertPrefs{
		ShowBadge:        true,
		BadgePosition:    "top-right",
		AutoDismissInfo:  true,
		InfoDismissTime:  "5s",
		PlaySound:        false,
		FlashWindow:      false,
		ShowDesktopNotif: true,
	}

	up.Performance = PerformancePrefs{
		LazyLoading:      true,
		CacheSize:        100,
		MaxConcurrentReq: 5,
		RequestTimeout:   "30s",
		EnableProfiling:  false,
		DebugMode:        false,
	}

	up.Layouts = LayoutPrefs{
		CurrentLayout:      "standard",
		AutoSave:           true,
		CustomLayouts:      []string{},
		LayoutHistory:      []string{},
		ResponsiveMode:     true,
		GridSnap:           true,
		ShowGrid:           false,
		AnimateTransitions: true,
		WidgetSettings:     make(map[string]WidgetPrefs),
	}

	// Default key bindings
	up.KeyBindings = map[string]string{
		"quit":            "q",
		"help":            "?",
		"refresh":         "F5",
		"search":          "/",
		"filter":          "f",
		"next_view":       "Tab",
		"prev_view":       "Shift+Tab",
		"select":          "Enter",
		"cancel":          "Esc",
		"page_up":         "PgUp",
		"page_down":       "PgDn",
		"home":            "Home",
		"end":             "End",
		"delete":          "d",
		"edit":            "e",
		"new":             "n",
		"save":            "Ctrl+s",
		"copy":            "Ctrl+c",
		"paste":           "Ctrl+v",
		"undo":            "Ctrl+z",
		"redo":            "Ctrl+y",
		"toggle_details":  "Space",
		"expand_all":      "+",
		"collapse_all":    "-",
		"mark":            "m",
		"mark_all":        "Ctrl+a",
		"clear_marks":     "Ctrl+u",
		"layout_switcher": "F4",
		"layout_edit":     "Ctrl+l",
	}
}

// Load loads preferences from disk
func (up *UserPreferences) Load() error {
	up.mu.Lock()
	defer up.mu.Unlock()

	data, err := ioutil.ReadFile(up.configPath)
	if err != nil {
		return err
	}

	// Parse JSON
	var prefs UserPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return fmt.Errorf("failed to parse preferences: %w", err)
	}

	// Update current preferences (keeping non-zero values)
	if prefs.General.RefreshInterval != "" {
		up.General = prefs.General
	}
	if prefs.Display.MaxColumnWidth > 0 {
		up.Display = prefs.Display
	}
	up.Colors = prefs.Colors

	// Merge maps
	for k, v := range prefs.KeyBindings {
		up.KeyBindings[k] = v
	}
	for k, v := range prefs.ViewSettings {
		up.ViewSettings[k] = v
	}

	up.Filters = prefs.Filters
	up.JobSubmission = prefs.JobSubmission
	up.Alerts = prefs.Alerts
	up.Performance = prefs.Performance
	up.Layouts = prefs.Layouts

	return nil
}

// Save saves preferences to disk
func (up *UserPreferences) Save() error {
	up.mu.RLock()
	defer up.mu.RUnlock()

	// Create directory if needed
	dir := filepath.Dir(up.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(up, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	// Write to temp file first
	tempFile := up.configPath + ".tmp"
	if err := ioutil.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write preferences: %w", err)
	}

	// Rename to actual file
	if err := os.Rename(tempFile, up.configPath); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("failed to save preferences: %w", err)
	}

	up.lastSaved = time.Now()
	return nil
}

// Get returns a copy of current preferences
func (up *UserPreferences) Get() UserPreferences {
	up.mu.RLock()
	defer up.mu.RUnlock()

	// Return a copy without the mutex
	prefs := UserPreferences{
		configPath:    up.configPath,
		General:       up.General,
		Display:       up.Display,
		Colors:        up.Colors,
		Filters:       up.Filters,
		JobSubmission: up.JobSubmission,
		Alerts:        up.Alerts,
		Performance:   up.Performance,
		Layouts:       up.Layouts,
		lastSaved:     up.lastSaved,
	}

	// Deep copy maps
	prefs.KeyBindings = make(map[string]string)
	for k, v := range up.KeyBindings {
		prefs.KeyBindings[k] = v
	}

	prefs.ViewSettings = make(map[string]ViewPrefs)
	for k, v := range up.ViewSettings {
		prefs.ViewSettings[k] = v
	}

	// Note: mu is nil in the returned copy (safe because it's a pointer)
	return prefs
}

// Update updates preferences with validation
func (up *UserPreferences) Update(update func(*UserPreferences) error) error {
	up.mu.Lock()
	defer up.mu.Unlock()

	// Create a copy for validation (mutex will be nil but that's ok for temp)
	temp := UserPreferences{
		configPath:    up.configPath,
		General:       up.General,
		Display:       up.Display,
		Colors:        up.Colors,
		Filters:       up.Filters,
		JobSubmission: up.JobSubmission,
		Alerts:        up.Alerts,
		Performance:   up.Performance,
		Layouts:       up.Layouts,
		lastSaved:     up.lastSaved,
	}
	// Deep copy maps
	temp.KeyBindings = make(map[string]string)
	for k, v := range up.KeyBindings {
		temp.KeyBindings[k] = v
	}
	temp.ViewSettings = make(map[string]ViewPrefs)
	for k, v := range up.ViewSettings {
		temp.ViewSettings[k] = v
	}

	// Apply updates
	if err := update(&temp); err != nil {
		return err
	}

	// Validate
	if err := temp.validate(); err != nil {
		return err
	}

	// Apply validated changes (excluding mutex and callbacks)
	up.configPath = temp.configPath
	up.General = temp.General
	up.Display = temp.Display
	up.Colors = temp.Colors
	up.KeyBindings = temp.KeyBindings
	up.ViewSettings = temp.ViewSettings
	up.Filters = temp.Filters
	up.JobSubmission = temp.JobSubmission
	up.Alerts = temp.Alerts
	up.Performance = temp.Performance
	up.Layouts = temp.Layouts
	up.lastSaved = temp.lastSaved

	// Notify listeners
	for _, fn := range up.onChange {
		go fn()
	}

	// Save without auto-locking since we already hold the lock
	return up.saveWithoutLock()
}

// validate checks if preferences are valid
func (up *UserPreferences) validate() error {
	// Validate refresh interval
	if up.General.AutoRefresh {
		if _, err := time.ParseDuration(up.General.RefreshInterval); err != nil {
			return fmt.Errorf("invalid refresh interval: %w", err)
		}
	}

	// Validate theme
	validThemes := []string{"default", "dark", "light"}
	valid := false
	for _, t := range validThemes {
		if up.General.Theme == t {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid theme: %s", up.General.Theme)
	}

	// Validate performance settings
	if up.Performance.CacheSize < 0 || up.Performance.CacheSize > 1000 {
		return fmt.Errorf("cache size must be between 0 and 1000 MB")
	}

	if up.Performance.MaxConcurrentReq < 1 || up.Performance.MaxConcurrentReq > 20 {
		return fmt.Errorf("max concurrent requests must be between 1 and 20")
	}

	return nil
}

// OnChange registers a callback for preference changes
func (up *UserPreferences) OnChange(fn func()) {
	up.mu.Lock()
	defer up.mu.Unlock()

	up.onChange = append(up.onChange, fn)
}

// GetKeyBinding returns the key binding for an action
func (up *UserPreferences) GetKeyBinding(action string) tcell.Key {
	up.mu.RLock()
	defer up.mu.RUnlock()

	binding, ok := up.KeyBindings[action]
	if !ok {
		return tcell.KeyRune
	}

	// Parse key binding string
	// This is simplified - in production you'd want a proper parser
	switch binding {
	case "Enter":
		return tcell.KeyEnter
	case "Esc":
		return tcell.KeyEsc
	case "Tab":
		return tcell.KeyTab
	case "F1":
		return tcell.KeyF1
	case "F5":
		return tcell.KeyF5
	default:
		return tcell.KeyRune
	}
}

// GetViewSettings returns settings for a specific view
func (up *UserPreferences) GetViewSettings(viewName string) ViewPrefs {
	up.mu.RLock()
	defer up.mu.RUnlock()

	if settings, ok := up.ViewSettings[viewName]; ok {
		return settings
	}

	// Return defaults
	return ViewPrefs{
		SortColumn:  "",
		SortOrder:   "asc",
		ShowDetails: true,
		PageSize:    50,
	}
}

// SetViewSettings updates settings for a specific view
func (up *UserPreferences) SetViewSettings(viewName string, settings ViewPrefs) error {
	return up.Update(func(p *UserPreferences) error {
		p.ViewSettings[viewName] = settings
		return nil
	})
}

// Export exports preferences to a file
func (up *UserPreferences) Export(path string) error {
	up.mu.RLock()
	defer up.mu.RUnlock()

	data, err := json.MarshalIndent(up, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, data, 0644)
}

// Import imports preferences from a file
func (up *UserPreferences) Import(path string) error {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	var prefs UserPreferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return err
	}

	// Validate imported preferences
	if err := prefs.validate(); err != nil {
		return fmt.Errorf("invalid preferences: %w", err)
	}

	up.mu.Lock()
	defer up.mu.Unlock()

	// Keep original path and mutex state
	originalPath := up.configPath
	originalOnChange := up.onChange

	// Copy preferences manually to avoid copying mutex state
	up.General = prefs.General
	up.Display = prefs.Display
	up.Colors = prefs.Colors
	up.KeyBindings = prefs.KeyBindings
	up.ViewSettings = prefs.ViewSettings
	up.Filters = prefs.Filters
	up.JobSubmission = prefs.JobSubmission
	up.Alerts = prefs.Alerts
	up.Performance = prefs.Performance
	up.Layouts = prefs.Layouts
	up.configPath = originalPath
	up.onChange = originalOnChange

	// Save to disk without auto-locking
	return up.saveWithoutLock()
}

// Reset resets preferences to defaults
func (up *UserPreferences) Reset() error {
	up.mu.Lock()
	defer up.mu.Unlock()

	up.setDefaults()

	// Notify listeners
	for _, fn := range up.onChange {
		go fn()
	}

	return up.saveWithoutLock()
}

// saveWithoutLock saves preferences to disk without acquiring locks
func (up *UserPreferences) saveWithoutLock() error {
	// Create directory if needed
	dir := filepath.Dir(up.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(up, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	// Write to temp file first
	tempFile := up.configPath + ".tmp"
	if err := ioutil.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write preferences: %w", err)
	}

	// Rename to actual file
	if err := os.Rename(tempFile, up.configPath); err != nil {
		_ = os.Remove(tempFile)
		return fmt.Errorf("failed to save preferences: %w", err)
	}

	up.lastSaved = time.Now()
	return nil
}
