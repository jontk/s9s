// Package views - help.go renders the full keyboard reference modal (F1).
//
// Content is generated dynamically: the "per-view" sections iterate the
// ViewManager and use each View's Hints() method as the source of truth.
// This means the help screen automatically stays in sync as views evolve —
// there is no hardcoded per-view key list to drift.
//
// The short contextual cheatsheet (bound to '?') still lives in
// internal/app/app_modals.go.
package views

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/dao"
	"github.com/rivo/tview"
)

// ShowFullHelpModal displays the comprehensive keyboard reference as a
// modal dialog. Per-view shortcut sections are generated from each view's
// Hints() method.
func ShowFullHelpModal(pages *tview.Pages, vm *ViewManager) {
	content := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(false).
		SetText(generateFullHelpContent(vm))

	footer := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[yellow]Esc / q[white] close  •  [yellow]j/k[white] scroll")

	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(content, 0, 1, true).
		AddItem(footer, 1, 0, false)

	modal.SetBorder(true).
		SetTitle(" Keyboard Reference ").
		SetTitleAlign(tview.AlignCenter)

	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			pages.RemovePage("fullhelp")
			return nil
		}
		if event.Key() == tcell.KeyRune {
			switch event.Rune() {
			case 'q', 'Q':
				pages.RemovePage("fullhelp")
				return nil
			case 'j':
				// Translate vim-style scroll to tview's native Down.
				return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
			case 'k':
				return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
			}
		}
		return event
	})

	// Centered modal: 80% width, 80% height.
	centered := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(modal, 0, 8, true).
			AddItem(nil, 0, 1, false), 0, 8, true).
		AddItem(nil, 0, 1, false)

	pages.AddPage("fullhelp", centered, true, true)
}

// generateFullHelpContent builds the full-reference help text. The
// per-view sections are driven by each view's Hints() method, so this
// stays in sync with the views themselves.
func generateFullHelpContent(vm *ViewManager) string {
	var b strings.Builder

	b.WriteString("[yellow::b]S9S — Keyboard Reference[white::-]\n\n")

	// Global shortcuts — mirrors internal/app/app_keyboard.go.
	b.WriteString("[teal::b]Global[white::-]\n")
	b.WriteString("  [yellow]1-0[white]           Switch to view (Jobs/Nodes/Partitions/Reservations/QoS/Accounts/Users/Dashboard/Health/Performance)\n")
	b.WriteString("  [yellow]Tab / Shift+Tab[white]  Cycle forward / back between views\n")
	b.WriteString("  [yellow]h / l[white]         Previous / next view\n")
	b.WriteString("  [yellow]F1[white]            Full keyboard reference (this screen)\n")
	b.WriteString("  [yellow]?[white]             Contextual cheatsheet\n")
	b.WriteString("  [yellow]F2[white]            System alerts\n")
	b.WriteString("  [yellow]F5[white]            Refresh current view\n")
	b.WriteString("  [yellow]F6[white]            Pause / resume auto-refresh\n")
	b.WriteString("  [yellow]F10[white]           Configuration\n")
	b.WriteString("  [yellow]:[white]             Command mode\n")
	b.WriteString("  [yellow]Ctrl+K[white]        Switch cluster\n")
	b.WriteString("  [yellow]Esc[white]           Close dialogs / modals\n")
	b.WriteString("  [yellow]q / Ctrl+C[white]    Quit\n\n")

	// Per-view shortcuts — driven by Hints().
	if vm != nil {
		for _, name := range vm.GetViewNames() {
			view, err := vm.GetView(name)
			if err != nil || view == nil {
				continue
			}
			hints := view.Hints()
			if len(hints) == 0 {
				continue
			}
			fmt.Fprintf(&b, "[teal::b]%s[white::-]\n", view.Title())
			for _, h := range hints {
				// Hints already contain tview color tags; just prefix
				// with two-space indent for readability.
				b.WriteString("  ")
				b.WriteString(h)
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}
	}

	// State color legend — driven by dao.Get*StateColor so it can never
	// drift from the actual colors used in the tables.
	b.WriteString("[teal::b]State Colors[white::-]\n")
	writeStateRow(&b, "Jobs      ", []string{
		dao.JobStateRunning, dao.JobStatePending, dao.JobStateCompleted,
		dao.JobStateFailed, dao.JobStateCancelled, dao.JobStateSuspended,
	}, dao.GetJobStateColor)
	writeStateRow(&b, "Nodes     ", []string{
		dao.NodeStateIdle, dao.NodeStateAllocated, dao.NodeStateMixed,
		dao.NodeStateDown, dao.NodeStateDrain, dao.NodeStateReserved,
		dao.NodeStateMaintenance,
	}, dao.GetNodeStateColor)
	writeStateRow(&b, "Partitions", []string{
		dao.PartitionStateUp, dao.PartitionStateDown,
		dao.PartitionStateDrain, dao.PartitionStateInactive,
	}, dao.GetPartitionStateColor)
	b.WriteString("\n")

	// Resource usage bar thresholds — mirrors getUsageColor in
	// performance_view.go. Kept here as a single short reference.
	b.WriteString("[teal::b]Resource Usage Bars[white::-]\n")
	b.WriteString("  [green]green[white]   < 75%   low\n")
	b.WriteString("  [yellow]yellow[white]  75–90%  elevated\n")
	b.WriteString("  [red]red[white]     ≥ 90%   high\n\n")

	b.WriteString("[gray]Tip: press ? from any view for a short contextual cheatsheet.[white]\n")

	return b.String()
}

// writeStateRow renders one line of the State Colors legend by calling
// the real color function for each state, so the legend cannot drift.
func writeStateRow(b *strings.Builder, label string, states []string, colorFn func(string) string) {
	fmt.Fprintf(b, "  %s  ", label)
	for i, s := range states {
		if i > 0 {
			b.WriteString("  ")
		}
		fmt.Fprintf(b, "[%s]%s[white]", colorFn(s), s)
	}
	b.WriteString("\n")
}
