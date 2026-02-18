package widgets

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/export"
	"github.com/jontk/s9s/internal/ui/styles"
	"github.com/rivo/tview"
)

// TableExportDialog is a modal dialog for exporting table data.
// It is modal-safe: call Show/Hide and it manages its own page on tview.Pages.
type TableExportDialog struct {
	*tview.Flex
	form           *tview.Form
	pages          *tview.Pages
	app            *tview.Application
	selectedFormat export.ExportFormat
	customPath     string
	viewName       string // e.g. "Jobs", "Nodes"
	recordCount    int
	onExport       func(format export.ExportFormat, path string)
	onCancel       func()
}

const tableExportPageName = "table_export_dialog"

// NewTableExportDialog creates a new export dialog for the given view.
func NewTableExportDialog(viewName string, recordCount int, pages *tview.Pages, app *tview.Application) *TableExportDialog {
	homeDir, _ := os.UserHomeDir()
	defaultPath := filepath.Join(homeDir, "slurm_exports")

	d := &TableExportDialog{
		Flex:           tview.NewFlex(),
		pages:          pages,
		app:            app,
		viewName:       viewName,
		recordCount:    recordCount,
		selectedFormat: export.FormatCSV,
		customPath:     defaultPath,
	}
	d.build()
	return d
}

func (d *TableExportDialog) build() {
	d.form = styles.StyleForm(tview.NewForm())
	d.form.SetBorder(true)
	d.form.SetBorderPadding(1, 1, 2, 2)
	d.form.SetTitle(fmt.Sprintf(" Export %s (%d records) ", d.viewName, d.recordCount))
	d.form.SetTitleAlign(tview.AlignCenter)
	d.form.SetBorderColor(tcell.ColorTeal)

	// Format dropdown — default CSV (index 1)
	formats := []string{"Text", "CSV", "JSON", "Markdown", "HTML"}
	d.form.AddDropDown("Format:", formats, 1, func(option string, _ int) {
		d.selectedFormat = parseFormat(option)
	})

	// Path input
	d.form.AddInputField("Save to:", d.customPath, 50, nil, func(text string) {
		d.customPath = text
	})

	// Buttons
	d.form.AddButton("Export", func() {
		if d.onExport != nil {
			d.onExport(d.selectedFormat, d.customPath)
		}
	})
	d.form.AddButton("Cancel", func() {
		d.Hide()
		if d.onCancel != nil {
			d.onCancel()
		}
	})

	// Help line
	help := tview.NewTextView().
		SetText("[Tab] Navigate  [Enter] Select  [Esc] Cancel").
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorGray)

	// Outer flex — centers the form vertically and horizontally
	d.SetDirection(tview.FlexRow).
		AddItem(tview.NewBox(), 0, 1, false).
		AddItem(
			tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(tview.NewBox(), 0, 1, false).
				AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
					AddItem(d.form, 11, 0, true).
					AddItem(help, 1, 0, false),
					60, 0, true).
				AddItem(tview.NewBox(), 0, 1, false),
			11, 0, true).
		AddItem(tview.NewBox(), 0, 1, false)
}

// SetExportHandler registers the callback invoked when the user clicks Export.
func (d *TableExportDialog) SetExportHandler(fn func(format export.ExportFormat, path string)) {
	d.onExport = fn
}

// SetCancelHandler registers the callback invoked when the dialog is canceled.
func (d *TableExportDialog) SetCancelHandler(fn func()) {
	d.onCancel = fn
}

// Show adds the dialog as a modal page.
func (d *TableExportDialog) Show() {
	if d.pages == nil {
		return
	}
	d.pages.AddPage(tableExportPageName, d, true, true)
	if d.app != nil {
		d.app.SetFocus(d.form)
	}
}

// Hide removes the dialog page.
func (d *TableExportDialog) Hide() {
	if d.pages == nil {
		return
	}
	d.pages.RemovePage(tableExportPageName)
}

// InputHandler intercepts Escape to cancel.
func (d *TableExportDialog) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return d.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		if event.Key() == tcell.KeyEsc {
			d.Hide()
			if d.onCancel != nil {
				d.onCancel()
			}
			return
		}
		d.Flex.InputHandler()(event, setFocus)
	})
}

// UpdateRecordCount updates the displayed record count (e.g. after filtering).
func (d *TableExportDialog) UpdateRecordCount(count int) {
	d.recordCount = count
	d.form.SetTitle(fmt.Sprintf(" Export %s (%d records) ", d.viewName, count))
}

// parseFormat maps the dropdown label to an ExportFormat.
func parseFormat(option string) export.ExportFormat {
	switch option {
	case "Text":
		return export.FormatText
	case "CSV":
		return export.FormatCSV
	case "JSON":
		return export.FormatJSON
	case "Markdown":
		return export.FormatMarkdown
	case "HTML":
		return export.FormatHTML
	}
	return export.FormatCSV
}
