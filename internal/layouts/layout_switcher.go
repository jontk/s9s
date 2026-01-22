package layouts

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// LayoutSwitcher provides a UI for switching between layouts
type LayoutSwitcher struct {
	manager  *LayoutManager
	app      *tview.Application
	pages    *tview.Pages
	modal    *tview.Flex
	list     *tview.List
	preview  *tview.TextView
	onSwitch func(layoutID string)
	onCancel func()
}

// NewLayoutSwitcher creates a new layout switcher
func NewLayoutSwitcher(manager *LayoutManager, app *tview.Application, pages *tview.Pages) *LayoutSwitcher {
	switcher := &LayoutSwitcher{
		manager: manager,
		app:     app,
		pages:   pages,
	}

	switcher.buildUI()
	return switcher
}

// buildUI builds the layout switcher interface
func (ls *LayoutSwitcher) buildUI() {
	// Create layout list
	ls.list = tview.NewList()
	ls.list.SetBorder(true)
	ls.list.SetTitle(" Available Layouts ")
	ls.list.SetTitleAlign(tview.AlignCenter)

	// Create preview panel
	ls.preview = tview.NewTextView()
	ls.preview.SetDynamicColors(true)
	ls.preview.SetWrap(true)
	ls.preview.SetBorder(true)
	ls.preview.SetTitle(" Layout Preview ")
	ls.preview.SetTitleAlign(tview.AlignCenter)

	// Create main layout
	content := tview.NewFlex()
	content.AddItem(ls.list, 0, 1, true)
	content.AddItem(ls.preview, 0, 2, false)

	// Create help text
	helpText := tview.NewTextView()
	helpText.SetDynamicColors(true)
	helpText.SetText("[yellow]Keys:[white] Enter=Select Space=Preview Esc=Cancel")
	helpText.SetTextAlign(tview.AlignCenter)

	// Create modal container
	ls.modal = tview.NewFlex()
	ls.modal.SetDirection(tview.FlexRow)
	ls.modal.AddItem(nil, 0, 1, false)
	ls.modal.AddItem(tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(content, 0, 1, true).
			AddItem(helpText, 1, 0, false), 80, 0, true).
		AddItem(nil, 0, 1, false), 0, 3, true)
	ls.modal.AddItem(nil, 0, 1, false)

	ls.modal.SetBorder(true)
	ls.modal.SetTitle(" Layout Switcher ")
	ls.modal.SetTitleAlign(tview.AlignCenter)

	// Setup event handlers
	ls.setupEventHandlers()

	// Populate layouts
	ls.populateLayouts()
}

// setupEventHandlers configures keyboard and selection handlers
func (ls *LayoutSwitcher) setupEventHandlers() {
	// Handle list selection
	ls.list.SetSelectedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		layoutID := ls.getLayoutIDFromIndex(index)
		if layoutID != "" {
			ls.switchToLayout(layoutID)
		}
	})

	// Handle list change for preview
	ls.list.SetChangedFunc(func(index int, mainText, secondaryText string, shortcut rune) {
		layoutID := ls.getLayoutIDFromIndex(index)
		if layoutID != "" {
			ls.showPreview(layoutID)
		}
	})

	// Handle keyboard shortcuts
	ls.modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			ls.close()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case ' ':
				// Preview current selection
				currentIndex := ls.list.GetCurrentItem()
				layoutID := ls.getLayoutIDFromIndex(currentIndex)
				if layoutID != "" {
					ls.showPreview(layoutID)
				}
				return nil
			}
		}
		return event
	})
}

// populateLayouts fills the list with available layouts
func (ls *LayoutSwitcher) populateLayouts() {
	layouts := ls.manager.ListLayouts()
	current := ls.manager.GetCurrentLayout()

	index := 0
	for _, layout := range layouts {
		text := layout.Name
		description := layout.Description

		// Mark current layout
		if current != nil && layout.ID == current.ID {
			text = "[green]● " + text + " (Current)[white]"
		}

		// Add template indicator
		if layout.Template != "" {
			text += fmt.Sprintf(" [blue](%s)[white]", layout.Template)
		}

		ls.list.AddItem(text, description, rune('1'+index), nil)
		index++
	}

	// Select first item and show preview
	if ls.list.GetItemCount() > 0 {
		ls.list.SetCurrentItem(0)
		layoutID := ls.getLayoutIDFromIndex(0)
		if layoutID != "" {
			ls.showPreview(layoutID)
		}
	}
}

// getLayoutIDFromIndex gets layout ID from list index
func (ls *LayoutSwitcher) getLayoutIDFromIndex(index int) string {
	layouts := ls.manager.ListLayouts()
	i := 0
	for id := range layouts {
		if i == index {
			return id
		}
		i++
	}
	return ""
}

// showPreview displays a preview of the selected layout
func (ls *LayoutSwitcher) showPreview(layoutID string) {
	layout, err := ls.manager.GetLayout(layoutID)
	if err != nil {
		ls.preview.SetText(fmt.Sprintf("[red]Error: %v[white]", err))
		return
	}

	preview := ls.generateLayoutPreview(layout)
	ls.preview.SetText(preview)
}

// generateLayoutPreview creates a text preview of the layout
func (ls *LayoutSwitcher) generateLayoutPreview(layout *Layout) string {
	var preview strings.Builder

	// Layout info
	preview.WriteString(fmt.Sprintf("[yellow]%s[white]\n", layout.Name))
	preview.WriteString(fmt.Sprintf("%s\n\n", layout.Description))

	// Grid information
	preview.WriteString(fmt.Sprintf("[blue]Grid:[white] %dx%d (%s)\n",
		layout.Grid.Rows, layout.Grid.Columns, layout.Grid.Orientation))

	if layout.Responsive {
		preview.WriteString("[green]✓ Responsive layout[white]\n")
	}

	preview.WriteString("\n[yellow]Widgets:[white]\n")

	// Widget list
	if len(layout.Widgets) == 0 {
		preview.WriteString("  No widgets configured\n")
	} else {
		for _, widget := range layout.Widgets {
			if !widget.Visible {
				continue
			}

			preview.WriteString(fmt.Sprintf("  • %s\n", widget.WidgetID))
			preview.WriteString(fmt.Sprintf("    Position: Row %d, Col %d\n",
				widget.Row, widget.Column))
			preview.WriteString(fmt.Sprintf("    Size: %d×%d (span: %d×%d)\n",
				widget.Width, widget.Height, widget.ColSpan, widget.RowSpan))

			if widget.Priority > 0 {
				preview.WriteString(fmt.Sprintf("    Priority: %d\n", widget.Priority))
			}

			preview.WriteString("\n")
		}
	}

	// ASCII layout diagram
	preview.WriteString("\n[yellow]Layout Diagram:[white]\n")
	diagram := ls.generateLayoutDiagram(layout)
	preview.WriteString(diagram)

	return preview.String()
}

// generateLayoutDiagram creates an ASCII diagram of the layout
func (ls *LayoutSwitcher) generateLayoutDiagram(layout *Layout) string {
	if layout.Grid.Rows == 0 || layout.Grid.Columns == 0 {
		return "Invalid grid dimensions"
	}

	// Create grid
	grid := make([][]string, layout.Grid.Rows)
	for i := range grid {
		grid[i] = make([]string, layout.Grid.Columns)
		for j := range grid[i] {
			grid[i][j] = "."
		}
	}

	// Place widgets
	widgetChars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	widgetIndex := 0

	for _, widget := range layout.Widgets {
		if !widget.Visible {
			continue
		}

		char := "?"
		if widgetIndex < len(widgetChars) {
			char = string(widgetChars[widgetIndex])
		}

		// Fill widget area
		for r := widget.Row; r < widget.Row+widget.RowSpan && r < layout.Grid.Rows; r++ {
			for c := widget.Column; c < widget.Column+widget.ColSpan && c < layout.Grid.Columns; c++ {
				grid[r][c] = char
			}
		}

		widgetIndex++
	}

	// Generate ASCII
	var diagram strings.Builder
	diagram.WriteString("┌")
	for c := 0; c < layout.Grid.Columns; c++ {
		diagram.WriteString("─")
	}
	diagram.WriteString("┐\n")

	for r := 0; r < layout.Grid.Rows; r++ {
		diagram.WriteString("│")
		for c := 0; c < layout.Grid.Columns; c++ {
			diagram.WriteString(grid[r][c])
		}
		diagram.WriteString("│\n")
	}

	diagram.WriteString("└")
	for c := 0; c < layout.Grid.Columns; c++ {
		diagram.WriteString("─")
	}
	diagram.WriteString("┘\n")

	// Add legend
	diagram.WriteString("\n[blue]Legend:[white]\n")
	widgetIndex = 0
	for _, widget := range layout.Widgets {
		if !widget.Visible {
			continue
		}

		char := "?"
		if widgetIndex < len(widgetChars) {
			char = string(widgetChars[widgetIndex])
		}

		diagram.WriteString(fmt.Sprintf("  %s = %s\n", char, widget.WidgetID))
		widgetIndex++
	}

	return diagram.String()
}

// switchToLayout switches to the selected layout
func (ls *LayoutSwitcher) switchToLayout(layoutID string) {
	err := ls.manager.SetCurrentLayout(layoutID)
	if err != nil {
		// Show error modal
		ls.showError(fmt.Sprintf("Failed to switch layout: %v", err))
		return
	}

	// Call callback
	if ls.onSwitch != nil {
		ls.onSwitch(layoutID)
	}

	// Close switcher
	ls.close()
}

// showError displays an error message
func (ls *LayoutSwitcher) showError(message string) {
	modal := tview.NewModal()
	modal.SetText(message)
	modal.AddButtons([]string{"OK"})
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		ls.pages.RemovePage("error")
	})

	ls.pages.AddPage("error", modal, true, true)
}

// close closes the layout switcher
func (ls *LayoutSwitcher) close() {
	if ls.onCancel != nil {
		ls.onCancel()
	}

	ls.pages.RemovePage("layout-switcher")
}

// Show displays the layout switcher
func (ls *LayoutSwitcher) Show() {
	ls.pages.AddPage("layout-switcher", ls.modal, true, true)
	ls.app.SetFocus(ls.list)
}

// SetOnSwitch sets the callback for layout switching
func (ls *LayoutSwitcher) SetOnSwitch(callback func(layoutID string)) {
	ls.onSwitch = callback
}

// SetOnCancel sets the callback for canceling
func (ls *LayoutSwitcher) SetOnCancel(callback func()) {
	ls.onCancel = callback
}

// ShowLayoutSwitcher displays the layout switcher modal
func ShowLayoutSwitcher(manager *LayoutManager, app *tview.Application, pages *tview.Pages, onSwitch func(string)) {
	switcher := NewLayoutSwitcher(manager, app, pages)
	switcher.SetOnSwitch(onSwitch)
	switcher.SetOnCancel(func() {
		// Just close, no additional action needed
	})
	switcher.Show()
}
