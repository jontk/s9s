package streaming

import (
	"sync"
)

// EventBus manages event distribution for streaming operations
type EventBus struct {
	subscribers map[string][]chan<- StreamEvent // key: "jobID:outputType"
	mu          sync.RWMutex
}

// NewEventBus creates a new event bus
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]chan<- StreamEvent),
	}
}

// Subscribe adds a subscriber for events from a specific job and output type
func (eb *EventBus) Subscribe(jobID, outputType string, ch chan<- StreamEvent) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	key := eb.makeKey(jobID, outputType)
	eb.subscribers[key] = append(eb.subscribers[key], ch)
}

// Unsubscribe removes a subscriber for a specific job and output type
func (eb *EventBus) Unsubscribe(jobID, outputType string, ch chan<- StreamEvent) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	key := eb.makeKey(jobID, outputType)
	subscribers := eb.subscribers[key]

	// Find and remove the channel
	for i, sub := range subscribers {
		if sub == ch {
			// Remove by swapping with last element and truncating
			subscribers[i] = subscribers[len(subscribers)-1]
			eb.subscribers[key] = subscribers[:len(subscribers)-1]
			break
		}
	}

	// Clean up empty subscriber lists
	if len(eb.subscribers[key]) == 0 {
		delete(eb.subscribers, key)
	}
}

// UnsubscribeAll removes all subscribers for a specific job and output type
func (eb *EventBus) UnsubscribeAll(jobID, outputType string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	key := eb.makeKey(jobID, outputType)

	// Close all channels to signal unsubscription
	for _, ch := range eb.subscribers[key] {
		close(ch)
	}

	delete(eb.subscribers, key)
}

// Publish sends an event to all subscribers of a specific job and output type
func (eb *EventBus) Publish(event StreamEvent) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	key := eb.makeKey(event.JobID, event.OutputType)
	subscribers := eb.subscribers[key]

	// Send event to all subscribers (non-blocking)
	for _, ch := range subscribers {
		select {
		case ch <- event:
			// Event sent successfully
		default:
			// Channel full or closed, skip this subscriber
			// In a production system, we might want to log this
		}
	}
}

// PublishError sends an error event to all subscribers of a specific job and output type
func (eb *EventBus) PublishError(jobID, outputType string, err error) {
	event := StreamEvent{
		JobID:      jobID,
		OutputType: outputType,
		EventType:  StreamEventError,
		Error:      err,
		Timestamp:  GetCurrentTime(),
	}
	eb.Publish(event)
}

// GetSubscriberCount returns the number of subscribers for a specific job and output type
func (eb *EventBus) GetSubscriberCount(jobID, outputType string) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	key := eb.makeKey(jobID, outputType)
	return len(eb.subscribers[key])
}

// GetAllSubscriptions returns information about all active subscriptions
func (eb *EventBus) GetAllSubscriptions() map[string]int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	result := make(map[string]int)
	for key, subscribers := range eb.subscribers {
		result[key] = len(subscribers)
	}

	return result
}

// HasSubscribers returns true if there are any subscribers for a specific job and output type
func (eb *EventBus) HasSubscribers(jobID, outputType string) bool {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	key := eb.makeKey(jobID, outputType)
	return len(eb.subscribers[key]) > 0
}

// Clear removes all subscribers and closes all channels
func (eb *EventBus) Clear() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	// Close all channels
	for _, subscribers := range eb.subscribers {
		for _, ch := range subscribers {
			close(ch)
		}
	}

	// Clear the map
	eb.subscribers = make(map[string][]chan<- StreamEvent)
}

// makeKey creates a unique key for job and output type combination
func (eb *EventBus) makeKey(jobID, outputType string) string {
	return jobID + ":" + outputType
}

// SubscriptionInfo contains information about a subscription
type SubscriptionInfo struct {
	JobID       string `json:"job_id"`
	OutputType  string `json:"output_type"`
	Subscribers int    `json:"subscribers"`
}

// GetSubscriptionInfo returns detailed information about all subscriptions
func (eb *EventBus) GetSubscriptionInfo() []SubscriptionInfo {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	info := make([]SubscriptionInfo, 0, len(eb.subscribers))

	for key, subscribers := range eb.subscribers {
		jobID, outputType := eb.parseKey(key)
		info = append(info, SubscriptionInfo{
			JobID:       jobID,
			OutputType:  outputType,
			Subscribers: len(subscribers),
		})
	}

	return info
}

// parseKey extracts job ID and output type from a subscription key
func (eb *EventBus) parseKey(key string) (jobID, outputType string) {
	// Split on the last colon to handle job IDs that might contain colons
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == ':' {
			return key[:i], key[i+1:]
		}
	}
	// Fallback if no colon found (shouldn't happen with valid keys)
	return key, ""
}
