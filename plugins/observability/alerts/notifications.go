package alerts

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// NotificationManager handles alert notifications
type NotificationManager struct {
	handlers      map[string]NotificationHandler
	subscriptions map[string][]AlertSubscription
	history       []NotificationEvent
	mu            sync.RWMutex

	// Configuration
	maxHistorySize int
	rateLimiter    *RateLimiter
}

// NotificationHandler defines the interface for notification handlers
type NotificationHandler interface {
	GetID() string
	GetName() string
	Send(ctx context.Context, notification Notification) error
	IsEnabled() bool
}

// Notification represents an alert notification
type Notification struct {
	Alert      Alert             `json:"alert"`
	Type       NotificationType  `json:"type"`
	Timestamp  time.Time         `json:"timestamp"`
	Recipients []string          `json:"recipients,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// NotificationType defines the type of notification
type NotificationType string

const (
	NotificationTypeAlert    NotificationType = "alert"
	NotificationTypeResolved NotificationType = "resolved"
	NotificationTypeTest     NotificationType = "test"
)

// AlertSubscription defines a subscription to alert notifications
type AlertSubscription struct {
	ID           string
	Filter       AlertFilter
	HandlerIDs   []string
	CreatedAt    time.Time
	LastNotified time.Time
}

// AlertFilter defines criteria for filtering alerts
type AlertFilter struct {
	Severities []string
	Sources    []string
	Rules      []string
	Labels     map[string]string
}

// NotificationEvent represents a notification event in history
type NotificationEvent struct {
	ID           string
	Notification Notification
	HandlerID    string
	Status       string
	Error        string
	SentAt       time.Time
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager() *NotificationManager {
	return &NotificationManager{
		handlers:       make(map[string]NotificationHandler),
		subscriptions:  make(map[string][]AlertSubscription),
		history:        make([]NotificationEvent, 0),
		maxHistorySize: 1000,
		rateLimiter:    NewRateLimiter(10, time.Minute), // 10 notifications per minute by default
	}
}

// RegisterHandler registers a notification handler
func (nm *NotificationManager) RegisterHandler(handler NotificationHandler) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	id := handler.GetID()
	if _, exists := nm.handlers[id]; exists {
		return fmt.Errorf("handler with ID %s already registered", id)
	}

	nm.handlers[id] = handler
	return nil
}

// UnregisterHandler unregisters a notification handler
func (nm *NotificationManager) UnregisterHandler(handlerID string) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	delete(nm.handlers, handlerID)
}

// Subscribe creates a subscription for alert notifications
func (nm *NotificationManager) Subscribe(subscription AlertSubscription) error {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	if subscription.ID == "" {
		subscription.ID = generateSubscriptionID()
	}

	subscription.CreatedAt = time.Now()

	// Group subscriptions by handler
	for _, handlerID := range subscription.HandlerIDs {
		if _, exists := nm.handlers[handlerID]; !exists {
			return fmt.Errorf("handler %s not found", handlerID)
		}

		nm.subscriptions[handlerID] = append(nm.subscriptions[handlerID], subscription)
	}

	return nil
}

// Unsubscribe removes a subscription
func (nm *NotificationManager) Unsubscribe(subscriptionID string) {
	nm.mu.Lock()
	defer nm.mu.Unlock()

	for handlerID, subs := range nm.subscriptions {
		newSubs := []AlertSubscription{}
		for _, sub := range subs {
			if sub.ID != subscriptionID {
				newSubs = append(newSubs, sub)
			}
		}
		nm.subscriptions[handlerID] = newSubs
	}
}

// NotifyAlert sends notifications for a new alert
func (nm *NotificationManager) NotifyAlert(ctx context.Context, alert Alert) error {
	notification := Notification{
		Alert:     alert,
		Type:      NotificationTypeAlert,
		Timestamp: time.Now(),
	}

	return nm.sendNotification(ctx, notification)
}

// NotifyResolved sends notifications for a resolved alert
func (nm *NotificationManager) NotifyResolved(ctx context.Context, alert Alert) error {
	notification := Notification{
		Alert:     alert,
		Type:      NotificationTypeResolved,
		Timestamp: time.Now(),
	}

	return nm.sendNotification(ctx, notification)
}

// sendNotification sends a notification to all matching subscriptions
func (nm *NotificationManager) sendNotification(ctx context.Context, notification Notification) error {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	var lastError error

	// Check each handler's subscriptions
	for handlerID, handler := range nm.handlers {
		if !handler.IsEnabled() {
			continue
		}

		// Check if any subscription matches
		for _, sub := range nm.subscriptions[handlerID] {
			if nm.matchesFilter(notification.Alert, sub.Filter) {
				// Apply rate limiting
				if !nm.rateLimiter.Allow(fmt.Sprintf("%s:%s", handlerID, sub.ID)) {
					continue
				}

				// Send notification
				event := NotificationEvent{
					ID:           generateEventID(),
					Notification: notification,
					HandlerID:    handlerID,
					Status:       "pending",
					SentAt:       time.Now(),
				}

				if err := handler.Send(ctx, notification); err != nil {
					event.Status = "failed"
					event.Error = err.Error()
					lastError = err
				} else {
					event.Status = "sent"
					sub.LastNotified = time.Now()
				}

				nm.addToHistory(event)
			}
		}
	}

	return lastError
}

// matchesFilter checks if an alert matches a subscription filter
func (nm *NotificationManager) matchesFilter(alert Alert, filter AlertFilter) bool {
	// Check severity
	if len(filter.Severities) > 0 {
		found := false
		for _, sev := range filter.Severities {
			if alert.Severity == sev {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check source
	if len(filter.Sources) > 0 {
		found := false
		for _, src := range filter.Sources {
			if alert.Source == src {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check rule
	if len(filter.Rules) > 0 {
		found := false
		for _, rule := range filter.Rules {
			if alert.RuleName == rule {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check labels
	for key, value := range filter.Labels {
		if alertValue, ok := alert.Labels[key]; !ok || alertValue != value {
			return false
		}
	}

	return true
}

// addToHistory adds an event to the notification history
func (nm *NotificationManager) addToHistory(event NotificationEvent) {
	nm.history = append([]NotificationEvent{event}, nm.history...)

	// Limit history size
	if len(nm.history) > nm.maxHistorySize {
		nm.history = nm.history[:nm.maxHistorySize]
	}
}

// GetHistory returns notification history
func (nm *NotificationManager) GetHistory(limit int) []NotificationEvent {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	if limit <= 0 || limit > len(nm.history) {
		limit = len(nm.history)
	}

	return nm.history[:limit]
}

// GetHandlers returns all registered handlers
func (nm *NotificationManager) GetHandlers() map[string]NotificationHandler {
	nm.mu.RLock()
	defer nm.mu.RUnlock()

	handlers := make(map[string]NotificationHandler)
	for id, handler := range nm.handlers {
		handlers[id] = handler
	}
	return handlers
}

// TestNotification sends a test notification to a specific handler
func (nm *NotificationManager) TestNotification(ctx context.Context, handlerID string) error {
	nm.mu.RLock()
	handler, exists := nm.handlers[handlerID]
	nm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("handler %s not found", handlerID)
	}

	testAlert := Alert{
		RuleName:    "test_alert",
		Severity:    "info",
		State:       AlertStateFiring,
		Message:     "This is a test alert notification",
		Description: "Testing notification handler connectivity",
		FirstSeen:   time.Now(),
		LastSeen:    time.Now(),
		Source:      "test",
	}

	notification := Notification{
		Alert:     testAlert,
		Type:      NotificationTypeTest,
		Timestamp: time.Now(),
	}

	return handler.Send(ctx, notification)
}

// Built-in notification handlers

// InAppHandler handles in-app notifications
type InAppHandler struct {
	enabled       bool
	notifications []Notification
	mu            sync.RWMutex
	maxSize       int
}

// NewInAppHandler creates a new in-app notification handler
func NewInAppHandler() *InAppHandler {
	return &InAppHandler{
		enabled:       true,
		notifications: make([]Notification, 0),
		maxSize:       100,
	}
}

func (h *InAppHandler) GetID() string {
	return "in-app"
}

func (h *InAppHandler) GetName() string {
	return "In-App Notifications"
}

func (h *InAppHandler) Send(ctx context.Context, notification Notification) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.notifications = append([]Notification{notification}, h.notifications...)

	// Limit size
	if len(h.notifications) > h.maxSize {
		h.notifications = h.notifications[:h.maxSize]
	}

	return nil
}

func (h *InAppHandler) IsEnabled() bool {
	return h.enabled
}

// GetNotifications returns in-app notifications
func (h *InAppHandler) GetNotifications(limit int) []Notification {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if limit <= 0 || limit > len(h.notifications) {
		limit = len(h.notifications)
	}

	return h.notifications[:limit]
}

// ClearNotifications clears all in-app notifications
func (h *InAppHandler) ClearNotifications() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.notifications = make([]Notification, 0)
}

// LogHandler logs notifications
type LogHandler struct {
	enabled bool
	logger  func(string)
}

// NewLogHandler creates a new log handler
func NewLogHandler(logger func(string)) *LogHandler {
	return &LogHandler{
		enabled: true,
		logger:  logger,
	}
}

func (h *LogHandler) GetID() string {
	return "log"
}

func (h *LogHandler) GetName() string {
	return "Log Handler"
}

func (h *LogHandler) Send(ctx context.Context, notification Notification) error {
	if h.logger == nil {
		return fmt.Errorf("logger not configured")
	}

	message := fmt.Sprintf("[ALERT] %s: %s - %s (severity: %s, source: %s)",
		notification.Type,
		notification.Alert.RuleName,
		notification.Alert.Message,
		notification.Alert.Severity,
		notification.Alert.Source,
	)

	h.logger(message)
	return nil
}

func (h *LogHandler) IsEnabled() bool {
	return h.enabled
}

// Helper functions

func generateSubscriptionID() string {
	return fmt.Sprintf("sub_%d", time.Now().UnixNano())
}

func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}

// RateLimiter provides simple rate limiting
type RateLimiter struct {
	rate     int
	period   time.Duration
	requests map[string][]time.Time
	mu       sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate int, period time.Duration) *RateLimiter {
	return &RateLimiter{
		rate:     rate,
		period:   period,
		requests: make(map[string][]time.Time),
	}
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.period)

	// Clean old requests
	validRequests := []time.Time{}
	for _, t := range rl.requests[key] {
		if t.After(cutoff) {
			validRequests = append(validRequests, t)
		}
	}

	// Check rate
	if len(validRequests) >= rl.rate {
		rl.requests[key] = validRequests
		return false
	}

	// Allow and record
	rl.requests[key] = append(validRequests, now)
	return true
}
