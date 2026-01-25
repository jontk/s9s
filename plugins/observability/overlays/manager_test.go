package overlays

import (
	"context"
	"testing"
	"time"

	"github.com/jontk/s9s/internal/plugin"
	"github.com/jontk/s9s/plugins/observability/prometheus"
)

// Mock overlay for testing
type mockOverlay struct {
	id            string
	shouldRefresh bool
	columns       []plugin.ColumnDefinition
	cellData      map[string]string
	cellStyles    map[string]plugin.CellStyle
}

func (m *mockOverlay) GetID() string {
	return m.id
}

func (m *mockOverlay) GetColumns() []plugin.ColumnDefinition {
	return m.columns
}

func (m *mockOverlay) GetCellData(_ context.Context, viewID string, _ interface{}, columnID string) (string, error) {
	key := viewID + ":" + columnID
	if data, exists := m.cellData[key]; exists {
		return data, nil
	}
	return "", nil
}

func (m *mockOverlay) GetCellStyle(_ context.Context, viewID string, _ interface{}, columnID string) plugin.CellStyle {
	key := viewID + ":" + columnID
	if style, exists := m.cellStyles[key]; exists {
		return style
	}
	return plugin.CellStyle{}
}

func (m *mockOverlay) ShouldRefresh() bool {
	return m.shouldRefresh
}

func TestNewOverlayManager(t *testing.T) {
	var client *prometheus.CachedClient // nil is fine for testing manager logic
	updateInterval := 30 * time.Second

	manager := NewOverlayManager(client, updateInterval)

	if manager == nil {
		t.Fatal("NewOverlayManager returned nil")
	}

	if manager.client != client {
		t.Error("Client not set correctly")
	}

	if manager.updateInterval != updateInterval {
		t.Error("Update interval not set correctly")
	}

	if manager.overlaysByView == nil {
		t.Error("overlaysByView not initialized")
	}

	if manager.overlayInfos == nil {
		t.Error("overlayInfos not initialized")
	}
}

func TestRegisterOverlay(t *testing.T) {
	manager := NewOverlayManager(nil, time.Minute)

	overlay := &mockOverlay{
		id: "test-overlay",
		columns: []plugin.ColumnDefinition{
			{ID: "test-col", Name: "Test Column"},
		},
	}

	info := plugin.OverlayInfo{
		ID:          "test-overlay",
		Name:        "Test Overlay",
		Description: "Test overlay for unit tests",
		TargetViews: []string{"test-view1", "test-view2"},
		Priority:    100,
	}

	err := manager.RegisterOverlay(&info, overlay)
	if err != nil {
		t.Fatalf("RegisterOverlay failed: %v", err)
	}

	// Check overlay is registered for each target view
	for _, viewID := range info.TargetViews {
		overlays := manager.GetOverlaysForView(viewID)
		if len(overlays) != 1 {
			t.Errorf("Expected 1 overlay for view %s, got %d", viewID, len(overlays))
		}
		if overlays[0] != overlay {
			t.Errorf("Wrong overlay registered for view %s", viewID)
		}
	}

	// Check overlay info is stored
	registeredInfos := manager.GetRegisteredOverlays()
	if _, exists := registeredInfos[info.ID]; !exists {
		t.Error("Overlay info not registered")
	}
}

func TestUnregisterOverlay(t *testing.T) {
	manager := NewOverlayManager(nil, time.Minute)

	overlay := &mockOverlay{id: "test-overlay"}
	info := plugin.OverlayInfo{
		ID:          "test-overlay",
		Name:        "Test Overlay",
		TargetViews: []string{"test-view"},
		Priority:    100,
	}

	// Register first
	err := manager.RegisterOverlay(&info, overlay)
	if err != nil {
		t.Fatalf("RegisterOverlay failed: %v", err)
	}

	// Verify it's registered
	overlays := manager.GetOverlaysForView("test-view")
	if len(overlays) != 1 {
		t.Fatal("Overlay not registered correctly")
	}

	// Unregister
	err = manager.UnregisterOverlay("test-overlay")
	if err != nil {
		t.Fatalf("UnregisterOverlay failed: %v", err)
	}

	// Verify it's unregistered
	overlays = manager.GetOverlaysForView("test-view")
	if len(overlays) != 0 {
		t.Error("Overlay not unregistered from view")
	}

	registeredInfos := manager.GetRegisteredOverlays()
	if _, exists := registeredInfos["test-overlay"]; exists {
		t.Error("Overlay info not unregistered")
	}

	// Test unregistering non-existent overlay
	err = manager.UnregisterOverlay("non-existent")
	if err == nil {
		t.Error("Expected error when unregistering non-existent overlay")
	}
}

func TestGetOverlaysForView(t *testing.T) {
	manager := NewOverlayManager(nil, time.Minute)

	overlay1 := &mockOverlay{id: "overlay1"}
	overlay2 := &mockOverlay{id: "overlay2"}

	info1 := plugin.OverlayInfo{
		ID:          "overlay1",
		TargetViews: []string{"view1", "view2"},
		Priority:    100,
	}

	info2 := plugin.OverlayInfo{
		ID:          "overlay2",
		TargetViews: []string{"view2", "view3"},
		Priority:    200,
	}

	// Register overlays
	_ = manager.RegisterOverlay(&info1, overlay1)
	_ = manager.RegisterOverlay(&info2, overlay2)

	// Test view1 (should have overlay1)
	overlays := manager.GetOverlaysForView("view1")
	if len(overlays) != 1 {
		t.Errorf("Expected 1 overlay for view1, got %d", len(overlays))
	}

	// Test view2 (should have both overlays)
	overlays = manager.GetOverlaysForView("view2")
	if len(overlays) != 2 {
		t.Errorf("Expected 2 overlays for view2, got %d", len(overlays))
	}

	// Test view3 (should have overlay2)
	overlays = manager.GetOverlaysForView("view3")
	if len(overlays) != 1 {
		t.Errorf("Expected 1 overlay for view3, got %d", len(overlays))
	}

	// Test non-existent view
	overlays = manager.GetOverlaysForView("non-existent")
	if len(overlays) != 0 {
		t.Errorf("Expected 0 overlays for non-existent view, got %d", len(overlays))
	}
}

func TestStartStopManager(t *testing.T) {
	manager := NewOverlayManager(nil, 100*time.Millisecond)
	ctx := context.Background()

	// Test starting
	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !manager.isRunning() {
		t.Error("Manager should be running after Start")
	}

	// Test starting again (should fail)
	err = manager.Start(ctx)
	if err == nil {
		t.Error("Expected error when starting already running manager")
	}

	// Test stopping
	err = manager.Stop()
	if err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	if manager.isRunning() {
		t.Error("Manager should not be running after Stop")
	}

	// Test stopping again (should fail)
	err = manager.Stop()
	if err == nil {
		t.Error("Expected error when stopping non-running manager")
	}
}

func TestTriggerUpdate(t *testing.T) {
	manager := NewOverlayManager(nil, time.Hour) // Long interval to avoid automatic updates
	ctx := context.Background()

	// Start manager
	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer func() { _ = manager.Stop() }()

	// Register an overlay
	overlay := &mockOverlay{id: "test-overlay"}
	info := plugin.OverlayInfo{
		ID:          "test-overlay",
		TargetViews: []string{"test-view"},
		Priority:    100,
	}
	_ = manager.RegisterOverlay(&info, overlay)

	// Trigger update for specific view
	manager.TriggerUpdate("test-view")

	// Give some time for processing
	time.Sleep(10 * time.Millisecond)

	// Trigger update for all views
	manager.TriggerUpdateAll()

	// Give some time for processing
	time.Sleep(10 * time.Millisecond)

	// Test triggering update on stopped manager
	_ = manager.Stop()
	manager.TriggerUpdate("test-view") // Should not panic or block
}

func TestShouldRefresh(t *testing.T) {
	manager := NewOverlayManager(nil, time.Minute)

	// Test with no overlays
	shouldRefresh := manager.ShouldRefresh("test-view")
	if shouldRefresh {
		t.Error("Should not refresh when no overlays exist")
	}

	// Register overlay that needs refresh
	overlay1 := &mockOverlay{id: "overlay1", shouldRefresh: true}
	info1 := plugin.OverlayInfo{
		ID:          "overlay1",
		TargetViews: []string{"test-view"},
		Priority:    100,
	}
	_ = manager.RegisterOverlay(&info1, overlay1)

	shouldRefresh = manager.ShouldRefresh("test-view")
	if !shouldRefresh {
		t.Error("Should refresh when overlay needs refresh")
	}

	// Register overlay that doesn't need refresh
	overlay2 := &mockOverlay{id: "overlay2", shouldRefresh: false}
	info2 := plugin.OverlayInfo{
		ID:          "overlay2",
		TargetViews: []string{"test-view"},
		Priority:    200,
	}
	_ = manager.RegisterOverlay(&info2, overlay2)

	shouldRefresh = manager.ShouldRefresh("test-view")
	if !shouldRefresh {
		t.Error("Should still refresh when at least one overlay needs refresh")
	}

	// Update first overlay to not need refresh
	overlay1.shouldRefresh = false
	shouldRefresh = manager.ShouldRefresh("test-view")
	if shouldRefresh {
		t.Error("Should not refresh when no overlays need refresh")
	}
}

func TestGetColumns(t *testing.T) {
	manager := NewOverlayManager(nil, time.Minute)

	// Test with no overlays
	columns := manager.GetColumns("test-view")
	if len(columns) != 0 {
		t.Errorf("Expected 0 columns with no overlays, got %d", len(columns))
	}

	// Register overlay with columns
	overlay := &mockOverlay{
		id: "test-overlay",
		columns: []plugin.ColumnDefinition{
			{ID: "col1", Name: "Column 1"},
			{ID: "col2", Name: "Column 2"},
		},
	}
	info := plugin.OverlayInfo{
		ID:          "test-overlay",
		TargetViews: []string{"test-view"},
		Priority:    100,
	}
	_ = manager.RegisterOverlay(&info, overlay)

	columns = manager.GetColumns("test-view")
	if len(columns) != 2 {
		t.Errorf("Expected 2 columns, got %d", len(columns))
	}

	if columns[0].ID != "col1" || columns[1].ID != "col2" {
		t.Error("Column IDs don't match expected values")
	}
}

func TestGetOverlayData(t *testing.T) {
	manager := NewOverlayManager(nil, time.Minute)
	ctx := context.Background()

	// Register overlay with data
	overlay := &mockOverlay{
		id: "test-overlay",
		cellData: map[string]string{
			"test-view:col1": "test-data",
		},
		cellStyles: map[string]plugin.CellStyle{
			"test-view:col1": {Foreground: "red", Bold: true},
		},
	}
	info := plugin.OverlayInfo{
		ID:          "test-overlay",
		TargetViews: []string{"test-view"},
		Priority:    100,
	}
	_ = manager.RegisterOverlay(&info, overlay)

	// Get overlay data
	data := manager.GetOverlayData(ctx, "test-view", "row1", "col1")
	if len(data) != 1 {
		t.Errorf("Expected 1 data item, got %d", len(data))
	}

	if data[0].Data != "test-data" {
		t.Errorf("Expected data 'test-data', got '%s'", data[0].Data)
	}

	if data[0].Style.Foreground != "red" || !data[0].Style.Bold {
		t.Error("Cell style not returned correctly")
	}
}

func TestGetOverlayByID(t *testing.T) {
	manager := NewOverlayManager(nil, time.Minute)

	// Test with non-existent overlay
	overlay := manager.GetOverlayByID("non-existent")
	if overlay != nil {
		t.Error("Expected nil for non-existent overlay")
	}

	// Register overlay
	testOverlay := &mockOverlay{id: "test-overlay"}
	info := plugin.OverlayInfo{
		ID:          "test-overlay",
		TargetViews: []string{"test-view"},
		Priority:    100,
	}
	_ = manager.RegisterOverlay(&info, testOverlay)

	// Get overlay by ID
	overlay = manager.GetOverlayByID("test-overlay")
	if overlay == nil {
		t.Error("Expected overlay, got nil")
	}

	if overlay != testOverlay {
		t.Error("Got different overlay than expected")
	}
}

func TestGetStats(t *testing.T) {
	manager := NewOverlayManager(nil, time.Minute)

	// Initial stats
	stats := manager.GetStats()
	if stats.TotalOverlays != 0 {
		t.Errorf("Expected 0 total overlays, got %d", stats.TotalOverlays)
	}
	if stats.ViewCount != 0 {
		t.Errorf("Expected 0 view count, got %d", stats.ViewCount)
	}
	if stats.IsRunning {
		t.Error("Expected manager not running")
	}

	// Register overlays and check stats
	overlay1 := &mockOverlay{id: "overlay1"}
	overlay2 := &mockOverlay{id: "overlay2"}

	info1 := plugin.OverlayInfo{
		ID:          "overlay1",
		TargetViews: []string{"view1", "view2"},
		Priority:    100,
	}

	info2 := plugin.OverlayInfo{
		ID:          "overlay2",
		TargetViews: []string{"view2"},
		Priority:    200,
	}

	_ = manager.RegisterOverlay(&info1, overlay1)
	_ = manager.RegisterOverlay(&info2, overlay2)

	stats = manager.GetStats()
	if stats.TotalOverlays != 2 {
		t.Errorf("Expected 2 total overlays, got %d", stats.TotalOverlays)
	}
	if stats.ViewCount != 2 {
		t.Errorf("Expected 2 views, got %d", stats.ViewCount)
	}

	// Check per-view counts
	if stats.OverlaysPerView["view1"] != 1 {
		t.Errorf("Expected 1 overlay for view1, got %d", stats.OverlaysPerView["view1"])
	}
	if stats.OverlaysPerView["view2"] != 2 {
		t.Errorf("Expected 2 overlays for view2, got %d", stats.OverlaysPerView["view2"])
	}

	// Test with running manager
	ctx := context.Background()
	_ = manager.Start(ctx)
	defer func() { _ = manager.Stop() }()

	stats = manager.GetStats()
	if !stats.IsRunning {
		t.Error("Expected manager running")
	}
}

func TestConcurrentAccess(t *testing.T) {
	manager := NewOverlayManager(nil, time.Minute)
	ctx := context.Background()

	// Start manager
	_ = manager.Start(ctx)
	defer func() { _ = manager.Stop() }()

	// Test concurrent registration and access
	done := make(chan bool)

	// Register overlays concurrently
	go func() {
		for i := 0; i < 10; i++ {
			overlay := &mockOverlay{id: "overlay" + string(rune('0'+i))}
			info := plugin.OverlayInfo{
				ID:          overlay.id,
				TargetViews: []string{"test-view"},
				Priority:    100 + i,
			}
			_ = manager.RegisterOverlay(&info, overlay)
		}
		done <- true
	}()

	// Access overlays concurrently
	go func() {
		for i := 0; i < 50; i++ {
			manager.GetOverlaysForView("test-view")
			manager.ShouldRefresh("test-view")
			manager.GetColumns("test-view")
			manager.TriggerUpdate("test-view")
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Verify final state
	overlays := manager.GetOverlaysForView("test-view")
	if len(overlays) != 10 {
		t.Errorf("Expected 10 overlays, got %d", len(overlays))
	}
}
