// Package subscription provides real-time data subscription and notification capabilities.
// It supports dynamic subscription management, configurable notification channels,
// persistent subscription storage, and efficient event distribution. The package
// enables real-time monitoring of metrics with customizable alert thresholds.
package subscription

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jontk/s9s/internal/plugin"
	"github.com/jontk/s9s/plugins/observability/prometheus"
)

// SubscriptionManager manages data subscriptions for the observability plugin
type SubscriptionManager struct {
	client        *prometheus.CachedClient
	subscriptions map[string]*Subscription
	callbacks     map[string]plugin.DataCallback
	mu            sync.RWMutex
	running       bool
	stopChan      chan struct{}
}

// Subscription represents an active data subscription
type Subscription struct {
	ID           string                 `json:"id"`
	ProviderID   string                 `json:"provider_id"`
	Params       map[string]interface{} `json:"params"`
	Callback     plugin.DataCallback    `json:"-"`
	CreatedAt    time.Time              `json:"created_at"`
	LastUpdate   time.Time              `json:"last_update"`
	UpdateCount  int64                  `json:"update_count"`
	UpdateInterval time.Duration        `json:"update_interval"`
	Active       bool                   `json:"active"`
	ErrorCount   int                    `json:"error_count"`
	LastError    string                 `json:"last_error,omitempty"`
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager(client *prometheus.CachedClient) *SubscriptionManager {
	return &SubscriptionManager{
		client:        client,
		subscriptions: make(map[string]*Subscription),
		callbacks:     make(map[string]plugin.DataCallback),
		stopChan:      make(chan struct{}),
	}
}

// Start starts the subscription manager
func (sm *SubscriptionManager) Start(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.running {
		return fmt.Errorf("subscription manager is already running")
	}

	sm.running = true
	sm.stopChan = make(chan struct{})

	// Start the update loop in a goroutine
	go sm.updateLoop(ctx)

	return nil
}

// Stop stops the subscription manager
func (sm *SubscriptionManager) Stop() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.running {
		return fmt.Errorf("subscription manager is not running")
	}

	sm.running = false
	close(sm.stopChan)

	return nil
}

// Subscribe creates a new data subscription
func (sm *SubscriptionManager) Subscribe(providerID string, params map[string]interface{}, callback plugin.DataCallback) (plugin.SubscriptionID, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Generate subscription ID
	subscriptionID := fmt.Sprintf("%s_%d", providerID, time.Now().UnixNano())

	// Validate provider ID
	if err := sm.validateProviderID(providerID); err != nil {
		return "", fmt.Errorf("invalid provider ID: %w", err)
	}

	// Determine update interval from params
	updateInterval := 30 * time.Second // Default
	if interval, ok := params["update_interval"]; ok {
		if intervalStr, ok := interval.(string); ok {
			if duration, err := time.ParseDuration(intervalStr); err == nil {
				updateInterval = duration
			}
		}
	}

	// Create subscription
	subscription := &Subscription{
		ID:             subscriptionID,
		ProviderID:     providerID,
		Params:         params,
		Callback:       callback,
		CreatedAt:      time.Now(),
		LastUpdate:     time.Time{},
		UpdateCount:    0,
		UpdateInterval: updateInterval,
		Active:         true,
		ErrorCount:     0,
	}

	sm.subscriptions[subscriptionID] = subscription
	sm.callbacks[subscriptionID] = callback

	return plugin.SubscriptionID(subscriptionID), nil
}

// Unsubscribe removes a data subscription
func (sm *SubscriptionManager) Unsubscribe(subscriptionID plugin.SubscriptionID) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	id := string(subscriptionID)
	if _, exists := sm.subscriptions[id]; !exists {
		return fmt.Errorf("subscription not found: %s", id)
	}

	delete(sm.subscriptions, id)
	delete(sm.callbacks, id)

	return nil
}

// GetSubscription returns subscription details
func (sm *SubscriptionManager) GetSubscription(subscriptionID plugin.SubscriptionID) (*Subscription, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	id := string(subscriptionID)
	subscription, exists := sm.subscriptions[id]
	if !exists {
		return nil, fmt.Errorf("subscription not found: %s", id)
	}

	// Return a copy to avoid race conditions
	subscriptionCopy := *subscription
	subscriptionCopy.Callback = nil // Don't expose callback in returned data
	return &subscriptionCopy, nil
}

// ListSubscriptions returns all active subscriptions
func (sm *SubscriptionManager) ListSubscriptions() map[string]*Subscription {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	result := make(map[string]*Subscription)
	for id, subscription := range sm.subscriptions {
		subscriptionCopy := *subscription
		subscriptionCopy.Callback = nil // Don't expose callback
		result[id] = &subscriptionCopy
	}

	return result
}

// GetStats returns subscription manager statistics
func (sm *SubscriptionManager) GetStats() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	totalSubscriptions := len(sm.subscriptions)
	activeSubscriptions := 0
	totalUpdates := int64(0)
	totalErrors := 0

	providerCounts := make(map[string]int)

	for _, subscription := range sm.subscriptions {
		if subscription.Active {
			activeSubscriptions++
		}
		totalUpdates += subscription.UpdateCount
		totalErrors += subscription.ErrorCount
		providerCounts[subscription.ProviderID]++
	}

	return map[string]interface{}{
		"total_subscriptions":  totalSubscriptions,
		"active_subscriptions": activeSubscriptions,
		"total_updates":        totalUpdates,
		"total_errors":         totalErrors,
		"provider_counts":      providerCounts,
		"running":              sm.running,
	}
}

// updateLoop runs the subscription update process
func (sm *SubscriptionManager) updateLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second) // Check every second
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sm.stopChan:
			return
		case <-ticker.C:
			sm.processUpdates(ctx)
		}
	}
}

// processUpdates processes subscription updates
func (sm *SubscriptionManager) processUpdates(ctx context.Context) {
	sm.mu.RLock()
	subscriptions := make([]*Subscription, 0, len(sm.subscriptions))
	for _, sub := range sm.subscriptions {
		if sub.Active {
			subscriptions = append(subscriptions, sub)
		}
	}
	sm.mu.RUnlock()

	for _, subscription := range subscriptions {
		// Check if it's time to update this subscription
		if time.Since(subscription.LastUpdate) >= subscription.UpdateInterval {
			go sm.updateSubscription(ctx, subscription)
		}
	}
}

// updateSubscription updates a specific subscription
func (sm *SubscriptionManager) updateSubscription(ctx context.Context, subscription *Subscription) {
	defer func() {
		if r := recover(); r != nil {
			sm.incrementErrorCount(subscription.ID, fmt.Sprintf("panic: %v", r))
		}
	}()

	// Get data based on provider ID
	data, err := sm.getData(ctx, subscription.ProviderID, subscription.Params)
	if err != nil {
		sm.incrementErrorCount(subscription.ID, err.Error())
		return
	}

	// Update subscription metadata
	sm.mu.Lock()
	subscription.LastUpdate = time.Now()
	subscription.UpdateCount++
	sm.mu.Unlock()

	// Call the callback
	if subscription.Callback != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					sm.incrementErrorCount(subscription.ID, fmt.Sprintf("callback panic: %v", r))
				}
			}()

			subscription.Callback(data, nil)
		}()
	}
}

// GetData retrieves data for a subscription (public method)
func (sm *SubscriptionManager) GetData(ctx context.Context, providerID string, params map[string]interface{}) (interface{}, error) {
	return sm.getData(ctx, providerID, params)
}

// getData retrieves data for a subscription (internal method)
func (sm *SubscriptionManager) getData(ctx context.Context, providerID string, params map[string]interface{}) (interface{}, error) {
	switch providerID {
	case "prometheus-metrics":
		return sm.getPrometheusMetrics(ctx, params)
	case "alerts":
		return sm.getAlerts(ctx, params)
	case "node-metrics":
		return sm.getNodeMetrics(ctx, params)
	case "job-metrics":
		return sm.getJobMetrics(ctx, params)
	default:
		return nil, fmt.Errorf("unknown provider ID: %s", providerID)
	}
}

// getPrometheusMetrics retrieves metrics from Prometheus
func (sm *SubscriptionManager) getPrometheusMetrics(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if sm.client == nil {
		return nil, fmt.Errorf("prometheus client not available")
	}

	// Extract query from parameters
	query, ok := params["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query parameter is required and must be a string")
	}

	// Execute the query
	result, err := sm.client.Query(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("prometheus query failed: %w", err)
	}

	return result, nil
}

// getAlerts retrieves active alerts
func (sm *SubscriptionManager) getAlerts(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// For now, return mock alert data
	// TODO: Implement actual alert retrieval from Prometheus Alertmanager
	return map[string]interface{}{
		"alerts": []map[string]interface{}{
			{
				"name":        "HighCPUUsage",
				"status":      "firing",
				"severity":    "warning",
				"instance":    "node1",
				"value":       "85%",
				"description": "CPU usage is above 80%",
				"timestamp":   time.Now().Unix(),
			},
		},
		"total": 1,
	}, nil
}

// getNodeMetrics retrieves node-specific metrics
func (sm *SubscriptionManager) getNodeMetrics(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if sm.client == nil {
		return nil, fmt.Errorf("prometheus client not available")
	}

	nodeID, ok := params["node_id"].(string)
	if !ok {
		return nil, fmt.Errorf("node_id parameter is required")
	}

	// Build queries for node metrics
	queries := map[string]string{
		"cpu":    fmt.Sprintf(`100 - (avg(irate(node_cpu_seconds_total{mode="idle",instance="%s"}[5m])) * 100)`, nodeID),
		"memory": fmt.Sprintf(`(1 - (node_memory_MemAvailable_bytes{instance="%s"} / node_memory_MemTotal_bytes{instance="%s"})) * 100`, nodeID, nodeID),
		"load":   fmt.Sprintf(`node_load1{instance="%s"}`, nodeID),
	}

	// Execute batch query
	results, err := sm.client.BatchQuery(ctx, queries, time.Now())
	if err != nil {
		return nil, fmt.Errorf("batch query failed: %w", err)
	}

	return map[string]interface{}{
		"node_id": nodeID,
		"metrics": results,
		"timestamp": time.Now().Unix(),
	}, nil
}

// getJobMetrics retrieves job-specific metrics
func (sm *SubscriptionManager) getJobMetrics(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if sm.client == nil {
		return nil, fmt.Errorf("prometheus client not available")
	}

	jobID, ok := params["job_id"].(string)
	if !ok {
		return nil, fmt.Errorf("job_id parameter is required")
	}

	// Build queries for job metrics
	queries := map[string]string{
		"cpu":    fmt.Sprintf(`rate(container_cpu_usage_seconds_total{name=~".*%s.*"}[5m]) * 100`, jobID),
		"memory": fmt.Sprintf(`container_memory_usage_bytes{name=~".*%s.*"} / 1024 / 1024`, jobID),
	}

	// Execute batch query
	results, err := sm.client.BatchQuery(ctx, queries, time.Now())
	if err != nil {
		return nil, fmt.Errorf("batch query failed: %w", err)
	}

	return map[string]interface{}{
		"job_id":    jobID,
		"metrics":   results,
		"timestamp": time.Now().Unix(),
	}, nil
}

// validateProviderID validates a provider ID
func (sm *SubscriptionManager) validateProviderID(providerID string) error {
	validProviders := []string{
		"prometheus-metrics",
		"alerts",
		"node-metrics",
		"job-metrics",
	}

	for _, valid := range validProviders {
		if providerID == valid {
			return nil
		}
	}

	return fmt.Errorf("unsupported provider ID: %s", providerID)
}

// incrementErrorCount increments the error count for a subscription
func (sm *SubscriptionManager) incrementErrorCount(subscriptionID, errorMsg string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if subscription, exists := sm.subscriptions[subscriptionID]; exists {
		subscription.ErrorCount++
		subscription.LastError = errorMsg

		// Disable subscription if too many errors
		if subscription.ErrorCount >= 10 {
			subscription.Active = false
		}
	}
}