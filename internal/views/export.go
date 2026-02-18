// Package views provides TUI views for the s9s application.
// This file implements export functionality shared across all views.
package views

import (
	"fmt"

	"github.com/jontk/s9s/internal/export"
	"github.com/jontk/s9s/internal/ui/widgets"
	"github.com/rivo/tview"
)

// showTableExportDialog is a helper that shows the export dialog, performs the export,
// and displays the result to the user via a modal message.
// getData must return (headers []string, rows [][]string, title string).
func showTableExportDialog(pages *tview.Pages, app *tview.Application, title string, getData func() *export.TableData) {
	if pages == nil || app == nil {
		return
	}

	td := getData()
	dialog := widgets.NewTableExportDialog(title, len(td.Rows), pages, app)

	dialog.SetExportHandler(func(format export.ExportFormat, path string) {
		dialog.Hide()

		// Perform export in background
		go func() {
			exporter := export.NewTableExporter(path)
			result, err := exporter.Export(td, format, "")

			app.QueueUpdateDraw(func() {
				var msg string
				if err != nil {
					msg = fmt.Sprintf("[red]Export failed: %v[white]", err)
				} else {
					msg = fmt.Sprintf("[green]Exported %d records to:[white]\n%s", len(td.Rows), result.FilePath)
				}
				showExportResultModal(pages, app, msg)
			})
		}()
	})

	dialog.SetCancelHandler(func() {
		dialog.Hide()
	})

	dialog.Show()
}

// showExportResultModal shows a brief result message after export.
func showExportResultModal(pages *tview.Pages, app *tview.Application, msg string) {
	const pageName = "export_result"

	modal := tview.NewModal().
		SetText(msg).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(_ int, _ string) {
			pages.RemovePage(pageName)
		})

	pages.AddPage(pageName, modal, true, true)
	app.SetFocus(modal)
}
