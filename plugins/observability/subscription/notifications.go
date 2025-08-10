package subscription

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/jontk/s9s/internal/plugin"
)

// NotificationLevel represents the severity level of a notification
type NotificationLevel string

const (
	LevelInfo    NotificationLevel = "info"
	LevelWarning NotificationLevel = "warning"
	LevelError   NotificationLevel = "error"
	LevelCritical NotificationLevel = "critical"
)

// Notification represents a data change notification
type Notification struct {
	ID           string                 `json:"id"`
	SubscriptionID string               `json:"subscription_id"`
	ProviderID   string                 `json:"provider_id"`
	Level        NotificationLevel      `json:"level"`
	Title        string                 `json:"title"`
	Message      string                 `json:"message"`
	Data         interface{}            `json:"data"`
	Metadata     map[string]interface{} `json:"metadata"`
	Timestamp    time.Time              `json:"timestamp"`
	Acknowledged bool                   `json:"acknowledged"`
}

// NotificationManager manages notifications and callbacks
type NotificationManager struct {
	notifications []Notification
	callbacks     map[string][]NotificationCallback
	mu            sync.RWMutex
	maxHistory    int
}

// NotificationCallback is a function that handles notifications
type NotificationCallback func(notification Notification)

// DataChangeDetector detects significant changes in subscription data
type DataChangeDetector struct {
	previousValues map[string]interface{}
	thresholds     map[string]float64
	mu             sync.RWMutex
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager(maxHistory int) *NotificationManager {
	if maxHistory <= 0 {
		maxHistory = 1000
	}

	return &NotificationManager{
		notifications: make([]Notification, 0),
		callbacks:     make(map[string][]NotificationCallback),
		maxHistory:    maxHistory,
	}
}

// RegisterCallback registers a callback for specific subscription types
func (nm *NotificationManager) RegisterCallback(subscriptionType string, callback NotificationCallback) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if nm.callbacks[subscriptionType] == nil {
		nm.callbacks[subscriptionType] = make([]NotificationCallback, 0)
	}

	nm.callbacks[subscriptionType] = append(nm.callbacks[subscriptionType], callback)
}

// UnregisterCallback removes a callback (this is a simplified implementation)
func (nm *NotificationManager) UnregisterCallback(subscriptionType string) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	delete(nm.callbacks, subscriptionType)
}

// SendNotification sends a notification to registered callbacks
func (nm *NotificationManager) SendNotification(notification Notification) {
	nm.mu.Lock()
	
	// Add to history
	nm.notifications = append(nm.notifications, notification)
	
	// Trim history if necessary
	if len(nm.notifications) > nm.maxHistory {
		nm.notifications = nm.notifications[len(nm.notifications)-nm.maxHistory:]
	}
	
	// Get callbacks for this provider
	callbacks := make([]NotificationCallback, 0)
	if cbs, exists := nm.callbacks[notification.ProviderID]; exists {
		callbacks = append(callbacks, cbs...)
	}
	
	// Also get general callbacks
	if cbs, exists := nm.callbacks["*"]; exists {
		callbacks = append(callbacks, cbs...)
	}
	
	nm.mu.Unlock()

	// Call callbacks asynchronously
	for _, callback := range callbacks {
		go func(cb NotificationCallback, notif Notification) {
			defer func() {
				if r := recover(); r != nil {
					// Log callback panic but don't fail
				}
			}()
			cb(notif)
		}(callback, notification)
	}
}

// GetNotifications returns recent notifications
func (nm *NotificationManager) GetNotifications(limit int) []Notification {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if limit <= 0 || limit > len(nm.notifications) {
		limit = len(nm.notifications)
	}

	// Return most recent notifications
	start := len(nm.notifications) - limit
	if start < 0 {
		start = 0
	}

	result := make([]Notification, limit)
	copy(result, nm.notifications[start:])

	return result
}

// AcknowledgeNotification marks a notification as acknowledged
func (nm *NotificationManager) AcknowledgeNotification(notificationID string) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	for i := range nm.notifications {
		if nm.notifications[i].ID == notificationID {
			nm.notifications[i].Acknowledged = true
			return nil
		}
	}

	return fmt.Errorf("notification not found: %s", notificationID)
}

// GetStats returns notification statistics
func (nm *NotificationManager) GetStats() map[string]interface{} {
	nm.mu.RLock()
	defer nm.mu.Unlock()

	levelCounts := make(map[string]int)
	acknowledgedCount := 0
	unacknowledgedCount := 0

	for _, notification := range nm.notifications {
		levelCounts[string(notification.Level)]++
		if notification.Acknowledged {
			acknowledgedCount++
		} else {
			unacknowledgedCount++
		}
	}

	return map[string]interface{}{
		"total_notifications":    len(nm.notifications),
		"acknowledged":           acknowledgedCount,
		"unacknowledged":        unacknowledgedCount,
		"level_counts":          levelCounts,
		"registered_callbacks":  len(nm.callbacks),
	}
}

// NewDataChangeDetector creates a new data change detector
func NewDataChangeDetector() *DataChangeDetector {
	return &DataChangeDetector{
		previousValues: make(map[string]interface{}),
		thresholds:     make(map[string]float64),
	}
}

// SetThreshold sets a change threshold for a specific metric
func (dcd *DataChangeDetector) SetThreshold(metricName string, threshold float64) {
	dcd.mu.Lock()
	defer dcd.mu.Unlock()

	dcd.thresholds[metricName] = threshold
}

// DetectChanges analyzes data for significant changes
func (dcd *DataChangeDetector) DetectChanges(subscriptionID, providerID string, currentData interface{}) []Notification {
	dcd.mu.Lock()
	defer dcd.mu.Unlock()

	notifications := make([]Notification, 0)
	key := fmt.Sprintf("%s_%s", subscriptionID, providerID)
	
	previousData, exists := dcd.previousValues[key]
	dcd.previousValues[key] = currentData

	if !exists {
		// First time seeing this data, no change to detect
		return notifications
	}

	// Convert data to map for analysis
	currentMap, err := dataToMap(currentData)
	if err != nil {
		return notifications
	}

	previousMap, err := dataToMap(previousData)
	if err != nil {
		return notifications
	}

	// Detect changes based on data type
	switch providerID {
	case "prometheus-metrics", "node-metrics", "job-metrics":
		notifications = append(notifications, dcd.detectMetricChanges(subscriptionID, providerID, previousMap, currentMap)...)
	case "alerts":
		notifications = append(notifications, dcd.detectAlertChanges(subscriptionID, providerID, previousMap, currentMap)...)
	}

	return notifications
}

// detectMetricChanges detects changes in metric data
func (dcd *DataChangeDetector) detectMetricChanges(subscriptionID, providerID string, previous, current map[string]interface{}) []Notification {
	notifications := make([]Notification, 0)

	// Check for metrics that have changed significantly
	for metricName, currentValue := range current {
		if previousValue, exists := previous[metricName]; exists {
			if change := dcd.calculateChange(previousValue, currentValue); change != nil {
				threshold := dcd.thresholds[metricName]
				if threshold == 0 {
					threshold = 10.0 // Default 10% threshold
				}

				if *change >= threshold {
					level := LevelWarning
					if *change >= 50.0 {
						level = LevelError
					}

					notification := Notification{
						ID:             fmt.Sprintf("%s_%s_%d", subscriptionID, metricName, time.Now().UnixNano()),
						SubscriptionID: subscriptionID,
						ProviderID:     providerID,
						Level:          level,
						Title:          fmt.Sprintf("Significant change in %s", metricName),
						Message:        fmt.Sprintf("Metric %s changed by %.2f%%", metricName, *change),
						Data: map[string]interface{}{
							"metric_name":    metricName,
							"previous_value": previousValue,
							"current_value":  currentValue,
							"change_percent": *change,
						},
						Timestamp: time.Now(),
					}

					notifications = append(notifications, notification)
				}
			}
		}
	}

	return notifications
}

// detectAlertChanges detects changes in alert data
func (dcd *DataChangeDetector) detectAlertChanges(subscriptionID, providerID string, previous, current map[string]interface{}) []Notification {
	notifications := make([]Notification, 0)

	// Extract alert arrays
	currentAlerts, _ := current["alerts"].([]interface{})
	previousAlerts, _ := previous["alerts"].([]interface{})

	// Simple change detection - compare counts
	if len(currentAlerts) != len(previousAlerts) {
		level := LevelInfo
		message := fmt.Sprintf("Alert count changed from %d to %d", len(previousAlerts), len(currentAlerts))

		if len(currentAlerts) > len(previousAlerts) {
			level = LevelWarning
			message = fmt.Sprintf("New alerts detected: %d alerts (was %d)", len(currentAlerts), len(previousAlerts))
		}

		notification := Notification{
			ID:             fmt.Sprintf("%s_alerts_%d", subscriptionID, time.Now().UnixNano()),
			SubscriptionID: subscriptionID,
			ProviderID:     providerID,
			Level:          level,
			Title:          "Alert status changed",
			Message:        message,
			Data: map[string]interface{}{
				"previous_count": len(previousAlerts),
				"current_count":  len(currentAlerts),
				"alerts":         currentAlerts,
			},
			Timestamp: time.Now(),
		}

		notifications = append(notifications, notification)
	}

	return notifications
}

// calculateChange calculates percentage change between two values
func (dcd *DataChangeDetector) calculateChange(previous, current interface{}) *float64 {
	prevFloat, ok1 := convertToFloat(previous)
	currFloat, ok2 := convertToFloat(current)

	if !ok1 || !ok2 || prevFloat == 0 {
		return nil
	}

	change := ((currFloat - prevFloat) / prevFloat) * 100
	if change < 0 {
		change = -change // Return absolute change
	}

	return &change
}

// convertToFloat attempts to convert an interface{} to float64
func convertToFloat(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case string:
		// Try to parse string as number
		if f, err := fmt.Sscanf(v, "%f", new(float64)); err == nil && f == 1 {
			var result float64
			fmt.Sscanf(v, "%f", &result)
			return result, true
		}
	}
	return 0, false
}

// dataToMap converts arbitrary data to map[string]interface{}
func dataToMap(data interface{}) (map[string]interface{}, error) {
	// If it's already a map, return it
	if m, ok := data.(map[string]interface{}); ok {
		return m, nil
	}

	// Try to convert via JSON marshaling/unmarshaling
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return result, nil
}

// Enhanced subscription callback wrapper
type EnhancedSubscriptionCallback struct {
	callback           plugin.DataCallback
	changeDetector     *DataChangeDetector
	notificationMgr    *NotificationManager
	subscriptionID     string
	providerID         string
}

// SetSubscriptionID sets the subscription ID after the subscription is created
func (esc *EnhancedSubscriptionCallback) SetSubscriptionID(id string) {
	esc.subscriptionID = id
}

// NewEnhancedSubscriptionCallback creates an enhanced callback with change detection
func NewEnhancedSubscriptionCallback(
	callback plugin.DataCallback,
	subscriptionID, providerID string,
	notificationMgr *NotificationManager,
) *EnhancedSubscriptionCallback {
	return &EnhancedSubscriptionCallback{
		callback:        callback,
		changeDetector:  NewDataChangeDetector(),
		notificationMgr: notificationMgr,
		subscriptionID:  subscriptionID,
		providerID:      providerID,
	}
}

// Call invokes the callback with change detection and notifications
func (esc *EnhancedSubscriptionCallback) Call(data interface{}, err error) {
	// Call original callback
	esc.callback(data, err)

	// Detect changes and send notifications if data is successful
	if err == nil && data != nil {
		notifications := esc.changeDetector.DetectChanges(esc.subscriptionID, esc.providerID, data)
		for _, notification := range notifications {
			esc.notificationMgr.SendNotification(notification)
		}
	}
}