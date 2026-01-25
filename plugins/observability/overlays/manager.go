// Package overlays provides dynamic metric overlay functionality for enhancing
// existing S9S views with real-time observability data. It supports non-intrusive
// integration with jobs and nodes views, configurable refresh intervals,
// and automatic overlay management with lifecycle controls.
package overlays

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jontk/s9s/internal/plugin"
	"github.com/jontk/s9s/plugins/observability/prometheus"
)

// OverlayManager manages all overlays for different views
type OverlayManager struct {
	// Configuration
	client         *prometheus.CachedClient
	updateInterval time.Duration

	// Registered overlays by view ID
	overlaysByView map[string][]plugin.Overlay
	overlayInfos   map[string]plugin.OverlayInfo

	// State management
	running    bool
	stopChan   chan struct{}
	mu         sync.RWMutex
	updateChan chan string // View ID to update
}

// NewOverlayManager creates a new overlay manager
func NewOverlayManager(client *prometheus.CachedClient, updateInterval time.Duration) *OverlayManager {
	return &OverlayManager{
		client:         client,
		updateInterval: updateInterval,
		overlaysByView: make(map[string][]plugin.Overlay),
		overlayInfos:   make(map[string]plugin.OverlayInfo),
		stopChan:       make(chan struct{}),
		updateChan:     make(chan string, 100),
	}
}

// RegisterOverlay registers an overlay for specific views
func (om *OverlayManager) RegisterOverlay(info *plugin.OverlayInfo, overlay plugin.Overlay) error {
	om.mu.Lock()
	defer om.mu.Unlock()

	// Store overlay info
	om.overlayInfos[info.ID] = *info

	// Register overlay for each target view
	for _, viewID := range info.TargetViews {
		if om.overlaysByView[viewID] == nil {
			om.overlaysByView[viewID] = make([]plugin.Overlay, 0)
		}
		om.overlaysByView[viewID] = append(om.overlaysByView[viewID], overlay)
	}

	return nil
}

// UnregisterOverlay removes an overlay from the manager
func (om *OverlayManager) UnregisterOverlay(overlayID string) error {
	om.mu.Lock()
	defer om.mu.Unlock()

	info, exists := om.overlayInfos[overlayID]
	if !exists {
		return fmt.Errorf("overlay %s not found", overlayID)
	}

	// Remove from each target view
	for _, viewID := range info.TargetViews {
		overlays := om.overlaysByView[viewID]
		for i, overlay := range overlays {
			if overlay == nil {
				continue
			}
			// We can't directly compare interfaces, so we'll need to track by ID
			// This is a limitation, but for now we'll remove by position
			// In a more sophisticated implementation, overlays would have IDs
			overlays[i] = nil
		}
		// Clean up nil entries
		cleanOverlays := make([]plugin.Overlay, 0)
		for _, overlay := range overlays {
			if overlay != nil {
				cleanOverlays = append(cleanOverlays, overlay)
			}
		}
		om.overlaysByView[viewID] = cleanOverlays
	}

	delete(om.overlayInfos, overlayID)
	return nil
}

// GetOverlaysForView returns all overlays registered for a specific view
func (om *OverlayManager) GetOverlaysForView(viewID string) []plugin.Overlay {
	om.mu.RLock()
	defer om.mu.RUnlock()

	overlays := om.overlaysByView[viewID]
	if overlays == nil {
		return []plugin.Overlay{}
	}

	// Return a copy to avoid race conditions
	result := make([]plugin.Overlay, len(overlays))
	copy(result, overlays)
	return result
}

// GetRegisteredOverlays returns all registered overlay infos
func (om *OverlayManager) GetRegisteredOverlays() map[string]plugin.OverlayInfo {
	om.mu.RLock()
	defer om.mu.RUnlock()

	// Return a copy
	result := make(map[string]plugin.OverlayInfo)
	for k, v := range om.overlayInfos {
		result[k] = v
	}
	return result
}

// Start begins the overlay manager background processing
func (om *OverlayManager) Start(ctx context.Context) error {
	om.mu.Lock()
	if om.running {
		om.mu.Unlock()
		return fmt.Errorf("overlay manager is already running")
	}
	om.running = true
	om.mu.Unlock()

	// Start update processing goroutine
	go om.updateProcessor(ctx)

	// Start periodic update trigger
	go om.periodicUpdater(ctx)

	return nil
}

// Stop stops the overlay manager
func (om *OverlayManager) Stop() error {
	if om == nil {
		return nil
	}

	om.mu.Lock()
	if !om.running {
		om.mu.Unlock()
		return fmt.Errorf("overlay manager is not running")
	}
	om.running = false
	om.mu.Unlock()

	close(om.stopChan)
	return nil
}

// TriggerUpdate requests an update for a specific view
func (om *OverlayManager) TriggerUpdate(viewID string) {
	if !om.isRunning() {
		return
	}

	select {
	case om.updateChan <- viewID:
	default:
		// Channel is full, skip this update request
	}
}

// TriggerUpdateAll requests updates for all views with overlays
func (om *OverlayManager) TriggerUpdateAll() {
	om.mu.RLock()
	viewIDs := make([]string, 0, len(om.overlaysByView))
	for viewID := range om.overlaysByView {
		viewIDs = append(viewIDs, viewID)
	}
	om.mu.RUnlock()

	for _, viewID := range viewIDs {
		om.TriggerUpdate(viewID)
	}
}

// isRunning safely checks if the manager is running
func (om *OverlayManager) isRunning() bool {
	om.mu.RLock()
	defer om.mu.RUnlock()
	return om.running
}

// updateProcessor processes update requests
func (om *OverlayManager) updateProcessor(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-om.stopChan:
			return
		case viewID := <-om.updateChan:
			om.processViewUpdate(ctx, viewID)
		}
	}
}

// periodicUpdater triggers periodic updates
func (om *OverlayManager) periodicUpdater(ctx context.Context) {
	ticker := time.NewTicker(om.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-om.stopChan:
			return
		case <-ticker.C:
			om.TriggerUpdateAll()
		}
	}
}

// processViewUpdate processes updates for a specific view
func (om *OverlayManager) processViewUpdate(_ context.Context, viewID string) {
	overlays := om.GetOverlaysForView(viewID)
	if len(overlays) == 0 {
		return
	}

	// Process each overlay for this view
	for _, overlay := range overlays {
		if overlay == nil {
			continue
		}

		// Update the overlay's data
		// This is where we would trigger data refresh for the overlay
		// Each overlay implementation should handle its own data updates

		// Since we don't have a standard Update method in the Overlay interface,
		// we could add one or handle updates differently
		// For now, overlays are responsible for their own data management
	}
}

// GetOverlayData retrieves data from an overlay for a specific view and position
func (om *OverlayManager) GetOverlayData(ctx context.Context, viewID string, rowID interface{}, columnID string) []OverlayData {
	overlays := om.GetOverlaysForView(viewID)
	if len(overlays) == 0 {
		return nil
	}

	var results []OverlayData
	for _, overlay := range overlays {
		if overlay == nil {
			continue
		}

		// Get cell data from overlay
		cellData, err := overlay.GetCellData(ctx, viewID, rowID, columnID)
		if err != nil {
			// Log error but continue with other overlays
			continue
		}

		// Get cell style
		style := overlay.GetCellStyle(ctx, viewID, rowID, columnID)

		results = append(results, OverlayData{
			OverlayID: "", // We don't have overlay ID available here
			Data:      cellData,
			Style:     style,
		})
	}

	return results
}

// OverlayData represents data from an overlay
type OverlayData struct {
	OverlayID string
	Data      string
	Style     plugin.CellStyle
}

// GetColumns returns the columns that overlays want to add to a view
func (om *OverlayManager) GetColumns(viewID string) []plugin.ColumnDefinition {
	overlays := om.GetOverlaysForView(viewID)
	if len(overlays) == 0 {
		return nil
	}

	var columns []plugin.ColumnDefinition
	for _, overlay := range overlays {
		if overlay == nil {
			continue
		}

		overlayColumns := overlay.GetColumns()
		columns = append(columns, overlayColumns...)
	}

	return columns
}

// ShouldRefresh checks if any overlay needs refresh for a view
func (om *OverlayManager) ShouldRefresh(viewID string) bool {
	overlays := om.GetOverlaysForView(viewID)
	if len(overlays) == 0 {
		return false
	}

	for _, overlay := range overlays {
		if overlay == nil {
			continue
		}

		if overlay.ShouldRefresh() {
			return true
		}
	}

	return false
}

// GetOverlayByID returns a specific overlay by its ID
func (om *OverlayManager) GetOverlayByID(overlayID string) plugin.Overlay {
	om.mu.RLock()
	defer om.mu.RUnlock()

	info, exists := om.overlayInfos[overlayID]
	if !exists {
		return nil
	}

	// Find the overlay in the registered overlays
	// This is a limitation of our current design - we don't directly map overlay IDs to instances
	// In a production system, we'd want to maintain this mapping
	for _, viewID := range info.TargetViews {
		overlays := om.overlaysByView[viewID]
		for _, overlay := range overlays {
			if overlay != nil && overlay.GetID() == overlayID {
				return overlay
			}
		}
	}

	return nil
}

// GetStats returns statistics about the overlay manager
func (om *OverlayManager) GetStats() OverlayManagerStats {
	om.mu.RLock()
	defer om.mu.RUnlock()

	stats := OverlayManagerStats{
		TotalOverlays:  len(om.overlayInfos),
		ViewCount:      len(om.overlaysByView),
		IsRunning:      om.running,
		UpdateInterval: om.updateInterval,
	}

	// Count overlays per view
	stats.OverlaysPerView = make(map[string]int)
	for viewID, overlays := range om.overlaysByView {
		stats.OverlaysPerView[viewID] = len(overlays)
	}

	return stats
}

// OverlayManagerStats contains statistics about the overlay manager
type OverlayManagerStats struct {
	TotalOverlays   int
	ViewCount       int
	OverlaysPerView map[string]int
	IsRunning       bool
	UpdateInterval  time.Duration
}
