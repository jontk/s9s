package views

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// HelpView displays interactive help and keyboard shortcuts
type HelpView struct {
	*BaseView
	container *tview.Flex
	content   *tview.TextView
	pages     *tview.Pages
	app       *tview.Application
}

// NewHelpView creates a new help view
func NewHelpView() *HelpView {
	v := &HelpView{
		BaseView: NewBaseView("help", "Help"),
	}

	v.content = tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetText(v.generateHelpContent())

	v.container = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(v.content, 0, 1, true)

	return v
}

// SetPages sets the pages reference for modal handling
func (v *HelpView) SetPages(pages *tview.Pages) {
	v.pages = pages
}

// SetApp sets the application reference
func (v *HelpView) SetApp(app *tview.Application) {
	v.app = app
}

// Render returns the view's main component
func (v *HelpView) Render() tview.Primitive {
	return v.container
}

// OnKey handles keyboard events
func (v *HelpView) OnKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEsc:
		if v.pages != nil {
			v.pages.RemovePage("help")
		}
		return nil
	case tcell.KeyRune:
		switch event.Rune() {
		case 'q', 'Q':
			if v.pages != nil {
				v.pages.RemovePage("help")
			}
			return nil
		}
	}
	return event
}

// OnFocus handles focus events
func (v *HelpView) OnFocus() error {
	if v.app != nil {
		v.app.SetFocus(v.content)
	}
	return nil
}

// OnLoseFocus handles loss of focus
func (v *HelpView) OnLoseFocus() error {
	return nil
}

// generateHelpContent generates the help content
func (v *HelpView) generateHelpContent() string {
	var help strings.Builder

	help.WriteString("[yellow::b]S9S - SLURM Terminal UI - Interactive Help[white::-]\n\n")

	// Global shortcuts
	help.WriteString("[teal::b]Global Shortcuts:[white::-]\n")
	help.WriteString("  [yellow]Tab[white]          Navigate between views\n")
	help.WriteString("  [yellow]1-9,0[white]        Quick switch to views (Jobs/Nodes/Partitions/etc)\n")
	help.WriteString("  [yellow]Ctrl+C[white]       Exit application\n")
	help.WriteString("  [yellow]F1[white]           Show this help\n")
	help.WriteString("  [yellow]F2[white]           Show system alerts\n")
	help.WriteString("  [yellow]F5[white]           Refresh current view\n")
	help.WriteString("  [yellow]Esc[white]          Close dialogs/modals\n\n")

	// Vim navigation
	help.WriteString("[teal::b]Vim Navigation:[white::-]\n")
	help.WriteString("  [yellow]j[white]            Move down\n")
	help.WriteString("  [yellow]k[white]            Move up\n")
	help.WriteString("  [yellow]g[white]            Go to top\n")
	help.WriteString("  [yellow]G[white]            Go to bottom\n\n")

	// Jobs view
	help.WriteString("[teal::b]Jobs View:[white::-]\n")
	help.WriteString("  [yellow]Enter[white]        Show job details\n")
	help.WriteString("  [yellow]k[white]            Kill selected job\n")
	help.WriteString("  [yellow]H[white]            Hold/release job\n")
	help.WriteString("  [yellow]c[white]            Cancel job\n")
	help.WriteString("  [yellow]s[white]            Submit new job\n")
	help.WriteString("  [yellow]/[white]            Filter jobs\n")
	help.WriteString("  [yellow]1-9[white]          Sort by column\n")
	help.WriteString("  [yellow]R[white]            Refresh view\n")
	help.WriteString("  [yellow]a[white]            Show analytics\n")
	help.WriteString("  [yellow]d[white]            Show dependencies\n")
	help.WriteString("  [yellow]t[white]            Show templates\n\n")

	// Nodes view
	help.WriteString("[teal::b]Nodes View:[white::-]\n")
	help.WriteString("  [yellow]Enter[white]        Show node details\n")
	help.WriteString("  [yellow]d[white]            Drain selected node\n")
	help.WriteString("  [yellow]r[white]            Resume drained node\n")
	help.WriteString("  [yellow]s[white]            SSH to node\n")
	help.WriteString("  [yellow]/[white]            Filter nodes\n")
	help.WriteString("  [yellow]1-9[white]          Sort by column\n")
	help.WriteString("  [yellow]R[white]            Refresh view\n")
	help.WriteString("  [yellow]p[white]            Filter by partition\n")
	help.WriteString("  [yellow]a[white]            Show all states\n")
	help.WriteString("  [yellow]i[white]            Filter idle nodes\n")
	help.WriteString("  [yellow]m[white]            Filter mixed nodes\n")
	help.WriteString("  [yellow]g[white]            Group nodes by attribute\n")
	help.WriteString("  [yellow]Space[white]        Toggle group expansion\n\n")

	// Partitions view
	help.WriteString("[teal::b]Partitions View:[white::-]\n")
	help.WriteString("  [yellow]Enter[white]        Show partition details\n")
	help.WriteString("  [yellow]/[white]            Filter partitions\n")
	help.WriteString("  [yellow]1-9[white]          Sort by column\n")
	help.WriteString("  [yellow]R[white]            Refresh view\n")
	help.WriteString("  [yellow]a[white]            Show analytics modal\n")
	help.WriteString("  [yellow]w[white]            Show wait time analytics\n\n")

	// QoS view
	help.WriteString("[teal::b]QoS View:[white::-]\n")
	help.WriteString("  [yellow]Enter[white]        Show QoS details\n")
	help.WriteString("  [yellow]/[white]            Filter QoS policies\n")
	help.WriteString("  [yellow]1-9[white]          Sort by column\n")
	help.WriteString("  [yellow]R[white]            Refresh view\n\n")

	// Accounts view
	help.WriteString("[teal::b]Accounts View:[white::-]\n")
	help.WriteString("  [yellow]Enter[white]        Show account details\n")
	help.WriteString("  [yellow]/[white]            Filter accounts\n")
	help.WriteString("  [yellow]1-9[white]          Sort by column\n")
	help.WriteString("  [yellow]R[white]            Refresh view\n")
	help.WriteString("  [yellow]h[white]            Show account hierarchy\n\n")

	// Users view
	help.WriteString("[teal::b]Users View:[white::-]\n")
	help.WriteString("  [yellow]Enter[white]        Show user details\n")
	help.WriteString("  [yellow]/[white]            Filter users\n")
	help.WriteString("  [yellow]1-9[white]          Sort by column\n")
	help.WriteString("  [yellow]R[white]            Refresh view\n")
	help.WriteString("  [yellow]a[white]            Show admin users\n\n")

	// Reservations view
	help.WriteString("[teal::b]Reservations View:[white::-]\n")
	help.WriteString("  [yellow]Enter[white]        Show reservation details\n")
	help.WriteString("  [yellow]/[white]            Filter reservations\n")
	help.WriteString("  [yellow]1-9[white]          Sort by column\n")
	help.WriteString("  [yellow]R[white]            Refresh view\n\n")

	// Dashboard
	help.WriteString("[teal::b]Dashboard:[white::-]\n")
	help.WriteString("  [yellow]a[white]            Show advanced analytics\n")
	help.WriteString("  [yellow]h[white]            Show health check\n")
	help.WriteString("  [yellow]R[white]            Refresh all panels\n\n")

	// Features and tips
	help.WriteString("[teal::b]Features & Tips:[white::-]\n")
	help.WriteString("  • [cyan]Real-time updates[white] - Data refreshes automatically every 5-30 seconds\n")
	help.WriteString("  • [cyan]ASCII visualizations[white] - Resource usage shown with colored progress bars\n")
	help.WriteString("  • [cyan]Advanced filtering[white] - Use '/' to filter data in any view\n")
	help.WriteString("  • [cyan]Sortable columns[white] - Click column headers or use number keys\n")
	help.WriteString("  • [cyan]Detailed analytics[white] - Press 'a' in most views for deep insights\n")
	help.WriteString("  • [cyan]Node grouping[white] - Group nodes by partition, state, or features\n")
	help.WriteString("  • [cyan]Wait time analysis[white] - Predictive queue analytics in partitions\n")
	help.WriteString("  • [cyan]Health monitoring[white] - Cluster health scoring and alerts\n")
	help.WriteString("  • [cyan]Hierarchical views[white] - Account hierarchy and dependency trees\n\n")

	// Color coding
	help.WriteString("[teal::b]Color Coding:[white::-]\n")
	help.WriteString("  [green]Green[white]         - Running, Available, Healthy states\n")
	help.WriteString("  [yellow]Yellow[white]        - Pending, Mixed, Warning states\n")
	help.WriteString("  [red]Red[white]           - Failed, Down, Error states\n")
	help.WriteString("  [blue]Blue[white]          - Allocated, Active states\n")
	help.WriteString("  [cyan]Cyan[white]          - Information, Special states\n")
	help.WriteString("  [orange]Orange[white]        - Drain, Maintenance states\n")
	help.WriteString("  [gray]Gray[white]          - Unknown, Inactive states\n\n")

	// Resource bars
	help.WriteString("[teal::b]Resource Usage Bars:[white::-]\n")
	help.WriteString("  ▰▰▰▰▱▱▱▱ - Visual representation of resource utilization\n")
	help.WriteString("  [green]Green bars[white]   - Low usage (< 50%)\n")
	help.WriteString("  [yellow]Yellow bars[white]  - Medium usage (50-80%)\n")
	help.WriteString("  [red]Red bars[white]     - High usage (> 80%)\n\n")

	help.WriteString("[teal::b]Press ESC or Q to close this help[white::-]\n")

	return help.String()
}

// ShowHelpModal shows the help as a modal dialog
func ShowHelpModal(pages *tview.Pages) {
	helpView := NewHelpView()
	helpView.SetPages(pages)

	// Create modal layout
	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(helpView.content, 0, 1, true).
		AddItem(tview.NewTextView().SetText("[yellow]Press ESC or Q to close[white]"), 1, 0, false)

	modal.SetBorder(true).
		SetTitle(" S9S Help & Keyboard Shortcuts ").
		SetTitleAlign(tview.AlignCenter)

	// Handle key events
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc ||
			(event.Key() == tcell.KeyRune && (event.Rune() == 'q' || event.Rune() == 'Q')) {
			pages.RemovePage("help")
			return nil
		}
		return event
	})

	// Create centered modal layout
	centeredModal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(modal, 0, 8, true).
			AddItem(nil, 0, 1, false), 0, 8, true).
		AddItem(nil, 0, 1, false)

	pages.AddPage("help", centeredModal, true, true)
}
