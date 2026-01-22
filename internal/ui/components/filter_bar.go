package components

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/ui/filters"
	"github.com/rivo/tview"
)

// FilterBar provides an advanced filtering interface
type FilterBar struct {
	*tview.Flex
	input *tview.InputField
	// TODO(lint): Review unused code - field presetDropdown is unused
	// presetDropdown *tview.List
	helpText      *tview.TextView
	parser        *filters.FilterParser
	presetManager *filters.PresetManager
	currentFilter *filters.Filter
	viewType      string
	// TODO(lint): Review unused code - field isAdvancedMode is unused
	// isAdvancedMode bool
	onFilterChange func(filter *filters.Filter)
	onClose        func()
	app            *tview.Application
	pages          *tview.Pages
}

// NewFilterBar creates a new filter bar component
func NewFilterBar(viewType string, app *tview.Application) *FilterBar {
	fb := &FilterBar{
		Flex:          tview.NewFlex(),
		parser:        filters.NewFilterParser(),
		presetManager: filters.NewPresetManager(),
		viewType:      viewType,
		app:           app,
	}

	// Create input field
	fb.input = tview.NewInputField().
		SetLabel("Filter: ").
		SetFieldWidth(50).
		SetPlaceholder("e.g., state=running user=john memory>4G").
		SetChangedFunc(fb.onInputChange).
		SetDoneFunc(fb.onInputDone)

	// Create help text
	fb.helpText = tview.NewTextView().
		SetDynamicColors(true).
		SetText(fb.getHelpText())

	// Build layout
	fb.SetDirection(tview.FlexRow).
		AddItem(fb.input, 3, 0, true).
		AddItem(fb.helpText, 2, 0, false)

	fb.SetBorder(true).
		SetTitle(" Advanced Filter ").
		SetTitleAlign(tview.AlignLeft)

	return fb
}

// SetPages sets the pages reference for modal handling
func (fb *FilterBar) SetPages(pages *tview.Pages) {
	fb.pages = pages
}

// SetOnFilterChange sets the callback for filter changes
func (fb *FilterBar) SetOnFilterChange(handler func(filter *filters.Filter)) {
	fb.onFilterChange = handler
}

// SetOnClose sets the callback for closing the filter bar
func (fb *FilterBar) SetOnClose(handler func()) {
	fb.onClose = handler
}

// Show displays the filter bar
func (fb *FilterBar) Show() {
	fb.app.SetFocus(fb.input)
}

// Hide hides the filter bar and calls onClose
func (fb *FilterBar) Hide() {
	if fb.onClose != nil {
		fb.onClose()
	}
}

// GetCurrentFilter returns the current filter
func (fb *FilterBar) GetCurrentFilter() *filters.Filter {
	return fb.currentFilter
}

// SetFilter sets the current filter
func (fb *FilterBar) SetFilter(filterStr string) {
	fb.input.SetText(filterStr)
	fb.parseAndApplyFilter(filterStr)
}

// onInputChange handles changes to the filter input
func (fb *FilterBar) onInputChange(text string) {
	// Update help text based on input
	fb.updateHelpText(text)
}

// onInputDone handles completion of filter input
func (fb *FilterBar) onInputDone(key tcell.Key) {
	switch key {
	case tcell.KeyEnter:
		// Apply filter
		fb.parseAndApplyFilter(fb.input.GetText())
	case tcell.KeyEsc:
		// Close filter bar
		fb.Hide()
	case tcell.KeyTab:
		// Show presets
		fb.showPresets()
	}
}

// parseAndApplyFilter parses and applies the filter
func (fb *FilterBar) parseAndApplyFilter(filterStr string) {
	filter, err := fb.parser.Parse(filterStr)
	if err != nil {
		fb.helpText.SetText(fmt.Sprintf("[red]Error: %v[white]", err))
		return
	}

	fb.currentFilter = filter
	if fb.onFilterChange != nil {
		fb.onFilterChange(filter)
	}

	// Update help text
	fb.helpText.SetText(fmt.Sprintf("[green]Filter applied:[white] %d conditions", len(filter.Expressions)))
}

// updateHelpText updates the help text based on current input
func (fb *FilterBar) updateHelpText(input string) {
	if input == "" {
		fb.helpText.SetText(fb.getHelpText())
		return
	}

	// Try to parse current input
	_, err := fb.parser.Parse(input)
	if err != nil {
		fb.helpText.SetText(fmt.Sprintf("[yellow]%v[white] | Press [yellow]Tab[white] for presets", err))
	} else {
		fb.helpText.SetText("[green]Valid filter[white] | Press [yellow]Enter[white] to apply | [yellow]Tab[white] for presets")
	}
}

// getHelpText returns the default help text
func (fb *FilterBar) getHelpText() string {
	examples := fb.getExamplesForView()
	return fmt.Sprintf("[yellow]Examples:[white] %s | [yellow]Tab[white] Presets | [yellow]F1[white] Help", examples)
}

// getExamplesForView returns example filters for the current view
func (fb *FilterBar) getExamplesForView() string {
	switch fb.viewType {
	case "jobs":
		return "state=running | memory>4G | time>2:00:00 | submittime=today"
	case "nodes":
		return "state=idle | memory>256G | features~gpu | name=~node[0-9]+"
	case "partitions":
		return "state=up | nodes>10 | maxmemory>1TB | qos~normal"
	default:
		return "field=value | field>4G | field=~pattern | time>1:30:00"
	}
}

// showPresets shows the filter presets menu
func (fb *FilterBar) showPresets() {
	if fb.pages == nil {
		return
	}

	presets := fb.presetManager.GetPresets(fb.viewType)
	if len(presets) == 0 {
		fb.helpText.SetText("[yellow]No presets available for this view[white]")
		return
	}

	// Create preset list
	list := tview.NewList()
	list.SetBorder(true).
		SetTitle(" Filter Presets ").
		SetTitleAlign(tview.AlignCenter)

	// Add presets
	for _, preset := range presets {
		p := preset // Capture for closure
		list.AddItem(
			p.Name,
			p.Description+" ("+p.FilterStr+")",
			0,
			func() {
				fb.SetFilter(p.FilterStr)
				fb.pages.RemovePage("filter-presets")
			},
		)
	}

	// Add options
	list.AddItem("──────────", "", 0, nil)

	list.AddItem("Save Current Filter", "Save current filter as preset", 's', func() {
		fb.pages.RemovePage("filter-presets")
		fb.showSavePresetDialog()
	})

	list.AddItem("Manage Presets", "Edit or delete existing presets", 'm', func() {
		fb.pages.RemovePage("filter-presets")
		fb.showManagePresetsDialog()
	})

	list.AddItem("Cancel", "Close without selecting", 'q', func() {
		fb.pages.RemovePage("filter-presets")
	})

	// Handle ESC
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			fb.pages.RemovePage("filter-presets")
			return nil
		}
		return event
	})

	// Show as modal
	modal := createCenteredModal(list, 60, 20)
	fb.pages.AddPage("filter-presets", modal, true, true)
}

// showSavePresetDialog shows dialog to save current filter as preset
func (fb *FilterBar) showSavePresetDialog() {
	form := tview.NewForm()

	var name, description string
	var isGlobal bool

	form.AddInputField("Name", "", 30, nil, func(text string) {
		name = text
	})

	form.AddInputField("Description", "", 50, nil, func(text string) {
		description = text
	})

	form.AddCheckbox("Global (all views)", false, func(checked bool) {
		isGlobal = checked
	})

	form.AddButton("Save", func() {
		if name == "" {
			return
		}

		preset := filters.FilterPreset{
			Name:        name,
			Description: description,
			ViewType:    fb.viewType,
			FilterStr:   fb.input.GetText(),
			IsGlobal:    isGlobal,
		}

		if isGlobal {
			preset.ViewType = "all"
		}

		_ = fb.presetManager.AddPreset(preset)
		fb.pages.RemovePage("save-preset")
		fb.helpText.SetText(fmt.Sprintf("[green]Preset '%s' saved[white]", name))
	})

	form.AddButton("Cancel", func() {
		fb.pages.RemovePage("save-preset")
	})

	form.SetBorder(true).
		SetTitle(" Save Filter Preset ").
		SetTitleAlign(tview.AlignCenter)

	// Handle ESC
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			fb.pages.RemovePage("save-preset")
			return nil
		}
		return event
	})

	modal := createCenteredModal(form, 60, 15)
	fb.pages.AddPage("save-preset", modal, true, true)
}

// showManagePresetsDialog shows dialog to manage existing presets
func (fb *FilterBar) showManagePresetsDialog() {
	if fb.pages == nil {
		return
	}

	presetManager := NewPresetManagerUI(fb.app, fb.presetManager, fb.viewType)
	presetManager.Show(fb.pages, func() {
		// Refresh help text when done
		fb.helpText.SetText(fb.getHelpText())
	})
}

// ShowFilterHelp shows detailed filter help
func (fb *FilterBar) ShowFilterHelp() {
	if fb.pages == nil {
		return
	}

	helpText := `[yellow]Advanced Filter Syntax[white]

[teal]Basic Operators:[white]
  =    Equals               state=running
  !=   Not equals           state!=failed
  ~    Contains             name~test
  !~   Not contains         name!~debug
  >    Greater than         priority>1000
  <    Less than            cpus<8
  >=   Greater or equal     memory>=4G
  <=   Less or equal        nodes<=10
  =~   Regex match          name=~^job_\d+
  in   In list              state in (running,pending)

[teal]Memory & Size Units:[white]
  memory>4G                 Memory greater than 4 gigabytes
  memory>=1024M             Memory at least 1024 megabytes
  memory<512MB              Memory less than 512 megabytes
  size!=2TB                 Size not equal to 2 terabytes

[teal]Time & Duration:[white]
  time>2:30:00              Time greater than 2h 30m
  time<=1-12:00:00          Time up to 1 day 12 hours
  elapsed>90m               Elapsed time over 90 minutes
  runtime>=2h30m            Runtime at least 2.5 hours

[teal]Date Ranges:[white]
  submittime=today          Jobs submitted today
  starttime=yesterday       Jobs started yesterday
  endtime="last week"       Jobs ended last week
  created="last 7 days"     Created in last 7 days
  submittime=2024-01-01..2024-01-31   Date range

[teal]Advanced Regex:[white]
  name=~^test_\d+$          Job names starting with test_
  user=~john|jane           User john or jane
  partition=~gpu.*compute   Partition containing gpu and compute
  node=~node[0-9]{2,3}      Node names like node01, node123

[teal]Field Names:[white]
  Jobs:       id, name, user, state, partition, priority, cpus, memory, time,
              submittime, starttime, endtime, elapsed, runtime
  Nodes:      name, state, partition, cpus, memory, features, uptime
  Partitions: name, state, nodes, cpus, qos, maxmemory, maxtime

[teal]Examples:[white]
  state=running memory>4G           Running jobs with >4GB RAM
  submittime=today user=john        Today's jobs by john
  memory>=256G features~gpu         High-mem GPU nodes
  time>2:00:00 partition!=debug     Long jobs not in debug
  name=~^job_\d+ state=pending      Pending jobs matching pattern
  endtime="last 24h" state=failed   Failed jobs in last day

[teal]Tips:[white]
  • Memory units: B, KB, MB, GB, TB (e.g., 4G, 1024M)
  • Time formats: HH:MM:SS or D-HH:MM:SS (e.g., 2:30:00, 1-12:00:00)
  • Date ranges: today, yesterday, "last week", "last N days"
  • Regex patterns: Use =~ for pattern matching
  • Multiple conditions are AND'ed by default
  • Field names are case-insensitive
  • Press Tab to see and select from saved presets

Press ESC to close`

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(helpText).
		SetScrollable(true)

	textView.SetBorder(true).
		SetTitle(" Filter Help ").
		SetTitleAlign(tview.AlignCenter)

	// Handle ESC
	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			fb.pages.RemovePage("filter-help")
			return nil
		}
		return event
	})

	modal := createCenteredModal(textView, 70, 30)
	fb.pages.AddPage("filter-help", modal, true, true)
}

// createCenteredModal helper function (duplicate from other files, should be in common utils)
func createCenteredModal(content tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(content, height, 1, true).
			AddItem(nil, 0, 1, false), width, 1, true).
		AddItem(nil, 0, 1, false)
}
