package layouts

import (
	"testing"

	"github.com/rivo/tview"
)

func TestFindRowBands_SingleRow(t *testing.T) {
	placements := []WidgetPlacement{
		{WidgetID: "a", Row: 0, RowSpan: 1},
		{WidgetID: "b", Row: 0, RowSpan: 1},
	}
	bands := (&LayoutManager{}).findRowBands(1, placements)
	if len(bands) != 1 {
		t.Fatalf("expected 1 band, got %d", len(bands))
	}
	if bands[0].startRow != 0 || bands[0].endRow != 1 {
		t.Errorf("expected band {0,1}, got {%d,%d}", bands[0].startRow, bands[0].endRow)
	}
}

func TestFindRowBands_NoSpanning(t *testing.T) {
	placements := []WidgetPlacement{
		{WidgetID: "a", Row: 0, RowSpan: 1},
		{WidgetID: "b", Row: 1, RowSpan: 1},
		{WidgetID: "c", Row: 2, RowSpan: 1},
	}
	bands := (&LayoutManager{}).findRowBands(3, placements)
	if len(bands) != 3 {
		t.Fatalf("expected 3 bands, got %d", len(bands))
	}
	for i, b := range bands {
		if b.startRow != i || b.endRow != i+1 {
			t.Errorf("band %d: expected {%d,%d}, got {%d,%d}", i, i, i+1, b.startRow, b.endRow)
		}
	}
}

func TestFindRowBands_SpanningMergesRows(t *testing.T) {
	// Widget "a" spans rows 0-1, linking them into one band. Row 2 is separate.
	placements := []WidgetPlacement{
		{WidgetID: "a", Row: 0, RowSpan: 2, Column: 0},
		{WidgetID: "b", Row: 0, RowSpan: 1, Column: 1},
		{WidgetID: "c", Row: 1, RowSpan: 1, Column: 1},
		{WidgetID: "d", Row: 2, RowSpan: 1, Column: 0},
	}
	bands := (&LayoutManager{}).findRowBands(3, placements)
	if len(bands) != 2 {
		t.Fatalf("expected 2 bands, got %d", len(bands))
	}
	if bands[0].startRow != 0 || bands[0].endRow != 2 {
		t.Errorf("band 0: expected {0,2}, got {%d,%d}", bands[0].startRow, bands[0].endRow)
	}
	if bands[1].startRow != 2 || bands[1].endRow != 3 {
		t.Errorf("band 1: expected {2,3}, got {%d,%d}", bands[1].startRow, bands[1].endRow)
	}
}

func TestFindRowBands_ChainedSpanning(t *testing.T) {
	// Widget "a" spans rows 0-1, widget "b" spans rows 1-2. All three rows merge.
	placements := []WidgetPlacement{
		{WidgetID: "a", Row: 0, RowSpan: 2},
		{WidgetID: "b", Row: 1, RowSpan: 2},
	}
	bands := (&LayoutManager{}).findRowBands(3, placements)
	if len(bands) != 1 {
		t.Fatalf("expected 1 band, got %d", len(bands))
	}
	if bands[0].startRow != 0 || bands[0].endRow != 3 {
		t.Errorf("expected band {0,3}, got {%d,%d}", bands[0].startRow, bands[0].endRow)
	}
}

func TestGroupByColumn(t *testing.T) {
	placements := []WidgetPlacement{
		{WidgetID: "a", Column: 2},
		{WidgetID: "b", Column: 0},
		{WidgetID: "c", Column: 2},
		{WidgetID: "d", Column: 1},
	}
	lm := &LayoutManager{}
	groups, cols := lm.groupByColumn(placements)

	// Columns should be sorted
	expected := []int{0, 1, 2}
	if len(cols) != len(expected) {
		t.Fatalf("expected %d columns, got %d", len(expected), len(cols))
	}
	for i, c := range cols {
		if c != expected[i] {
			t.Errorf("cols[%d]: expected %d, got %d", i, expected[i], c)
		}
	}

	// Column 2 should have 2 widgets
	if len(groups[2]) != 2 {
		t.Errorf("expected 2 widgets in column 2, got %d", len(groups[2]))
	}
	// Columns 0 and 1 should have 1 each
	if len(groups[0]) != 1 || len(groups[1]) != 1 {
		t.Errorf("expected 1 widget each in columns 0 and 1")
	}
}

// stubWidget implements the Widget interface for testing.
type stubWidget struct {
	id   string
	prim tview.Primitive
}

func newStubWidget(id string) *stubWidget {
	return &stubWidget{id: id, prim: tview.NewBox()}
}

func (w *stubWidget) ID() string                  { return w.id }
func (w *stubWidget) Name() string                { return w.id }
func (w *stubWidget) Description() string          { return "" }
func (w *stubWidget) Type() WidgetType             { return "test" }
func (w *stubWidget) Render() tview.Primitive      { return w.prim }
func (w *stubWidget) Update() error                { return nil }
func (w *stubWidget) Configure() error             { return nil }
func (w *stubWidget) MinSize() (int, int)          { return 0, 0 }
func (w *stubWidget) MaxSize() (int, int)          { return 0, 0 }
func (w *stubWidget) OnResize(_, _ int)            {}
func (w *stubWidget) OnFocus(_ bool)               {}

func TestBuildBandFlex_SingleWidget(t *testing.T) {
	lm := NewLayoutManager(tview.NewApplication())
	_ = lm.RegisterWidget(newStubWidget("a"))

	placements := []WidgetPlacement{
		{WidgetID: "a", Row: 0, Column: 0, RowSpan: 1, ColSpan: 1, Width: 100, Visible: true},
	}
	band := rowBand{startRow: 0, endRow: 1}

	lm.mu.Lock()
	flex := lm.buildBandFlex(band, placements)
	lm.mu.Unlock()

	if flex == nil {
		t.Fatal("expected non-nil flex")
	}
	if flex.GetItemCount() != 1 {
		t.Errorf("expected 1 item, got %d", flex.GetItemCount())
	}
}

func TestBuildBandFlex_StackedWidgets(t *testing.T) {
	lm := NewLayoutManager(tview.NewApplication())
	_ = lm.RegisterWidget(newStubWidget("a"))
	_ = lm.RegisterWidget(newStubWidget("b"))

	// Two widgets in the same column, different rows
	placements := []WidgetPlacement{
		{WidgetID: "a", Row: 0, Column: 0, RowSpan: 1, ColSpan: 1, Width: 50, Visible: true},
		{WidgetID: "b", Row: 1, Column: 0, RowSpan: 1, ColSpan: 1, Width: 50, Visible: true},
	}
	band := rowBand{startRow: 0, endRow: 2}

	lm.mu.Lock()
	flex := lm.buildBandFlex(band, placements)
	lm.mu.Unlock()

	if flex == nil {
		t.Fatal("expected non-nil flex")
	}
	// One column group → one item in the horizontal flex (a vertical stack)
	if flex.GetItemCount() != 1 {
		t.Errorf("expected 1 column group, got %d", flex.GetItemCount())
	}
}

func TestBuildBandFlex_MultipleColumns(t *testing.T) {
	lm := NewLayoutManager(tview.NewApplication())
	_ = lm.RegisterWidget(newStubWidget("a"))
	_ = lm.RegisterWidget(newStubWidget("b"))
	_ = lm.RegisterWidget(newStubWidget("c"))

	// "a" spans 2 rows in col 0, "b" and "c" stacked in col 1
	placements := []WidgetPlacement{
		{WidgetID: "a", Row: 0, Column: 0, RowSpan: 2, ColSpan: 1, Width: 60, Visible: true},
		{WidgetID: "b", Row: 0, Column: 1, RowSpan: 1, ColSpan: 1, Width: 40, Visible: true},
		{WidgetID: "c", Row: 1, Column: 1, RowSpan: 1, ColSpan: 1, Width: 40, Visible: true},
	}
	band := rowBand{startRow: 0, endRow: 2}

	lm.mu.Lock()
	flex := lm.buildBandFlex(band, placements)
	lm.mu.Unlock()

	if flex == nil {
		t.Fatal("expected non-nil flex")
	}
	// Two column groups: col 0 (single widget) and col 1 (vertical stack)
	if flex.GetItemCount() != 2 {
		t.Errorf("expected 2 column groups, got %d", flex.GetItemCount())
	}
}

func TestApplyGridLayout_EmptyPlacements(t *testing.T) {
	lm := NewLayoutManager(tview.NewApplication())

	layout := &Layout{
		ID:   "test",
		Name: "Test",
		Grid: GridConfig{Rows: 2, Columns: 2, Orientation: "grid"},
		Widgets: []WidgetPlacement{
			{WidgetID: "missing", Row: 0, Column: 0, RowSpan: 1, ColSpan: 1, Width: 100, Visible: true},
		},
	}

	lm.mu.Lock()
	err := lm.applyGridLayout(layout)
	lm.mu.Unlock()

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	// Missing widgets are skipped, container should be empty
	if lm.container.GetItemCount() != 0 {
		t.Errorf("expected 0 items in container, got %d", lm.container.GetItemCount())
	}
}
