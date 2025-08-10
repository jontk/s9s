package subscription

import (
	"context"
	"testing"
	"time"
)

func TestNewSubscriptionManager(t *testing.T) {
	manager := NewSubscriptionManager(nil)
	if manager == nil {
		t.Fatal("NewSubscriptionManager returned nil")
	}

	if manager.subscriptions == nil {
		t.Error("subscriptions map not initialized")
	}

	if manager.callbacks == nil {
		t.Error("callbacks map not initialized")
	}
}

func TestSubscriptionManagerStartStop(t *testing.T) {
	manager := NewSubscriptionManager(nil)
	ctx := context.Background()

	// Test starting
	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !manager.running {
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

	if manager.running {
		t.Error("Manager should not be running after Stop")
	}

	// Test stopping again (should fail)
	err = manager.Stop()
	if err == nil {
		t.Error("Expected error when stopping non-running manager")
	}
}

func TestSubscribe(t *testing.T) {
	manager := NewSubscriptionManager(nil)
	
	callback := func(data interface{}, err error) {
		// Callback for testing
	}

	params := map[string]interface{}{
		"update_interval": "30s",
	}

	subscriptionID, err := manager.Subscribe("prometheus-metrics", params, callback)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	if subscriptionID == "" {
		t.Error("Subscribe returned empty subscription ID")
	}

	// Check subscription was stored
	subscription, err := manager.GetSubscription(subscriptionID)
	if err != nil {
		t.Fatalf("GetSubscription failed: %v", err)
	}

	if subscription.ID != string(subscriptionID) {
		t.Error("Subscription ID mismatch")
	}

	if subscription.ProviderID != "prometheus-metrics" {
		t.Error("Provider ID mismatch")
	}

	if !subscription.Active {
		t.Error("Subscription should be active")
	}
}

func TestUnsubscribe(t *testing.T) {
	manager := NewSubscriptionManager(nil)
	
	callback := func(data interface{}, err error) {}
	params := map[string]interface{}{}

	subscriptionID, err := manager.Subscribe("prometheus-metrics", params, callback)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Verify subscription exists
	_, err = manager.GetSubscription(subscriptionID)
	if err != nil {
		t.Fatalf("GetSubscription failed: %v", err)
	}

	// Unsubscribe
	err = manager.Unsubscribe(subscriptionID)
	if err != nil {
		t.Fatalf("Unsubscribe failed: %v", err)
	}

	// Verify subscription is removed
	_, err = manager.GetSubscription(subscriptionID)
	if err == nil {
		t.Error("Expected error when getting unsubscribed subscription")
	}

	// Test unsubscribing non-existent subscription
	err = manager.Unsubscribe("non-existent")
	if err == nil {
		t.Error("Expected error when unsubscribing non-existent subscription")
	}
}

func TestListSubscriptions(t *testing.T) {
	manager := NewSubscriptionManager(nil)
	callback := func(data interface{}, err error) {}

	// Initially should be empty
	subscriptions := manager.ListSubscriptions()
	if len(subscriptions) != 0 {
		t.Errorf("Expected 0 subscriptions, got %d", len(subscriptions))
	}

	// Add subscriptions
	params := map[string]interface{}{}
	subscriptionID1, _ := manager.Subscribe("prometheus-metrics", params, callback)
	subscriptionID2, _ := manager.Subscribe("alerts", params, callback)

	subscriptions = manager.ListSubscriptions()
	if len(subscriptions) != 2 {
		t.Errorf("Expected 2 subscriptions, got %d", len(subscriptions))
	}

	// Check subscription IDs are present
	found1, found2 := false, false
	for id := range subscriptions {
		if id == string(subscriptionID1) {
			found1 = true
		}
		if id == string(subscriptionID2) {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Error("Not all subscriptions found in list")
	}
}

func TestGetStats(t *testing.T) {
	manager := NewSubscriptionManager(nil)
	callback := func(data interface{}, err error) {}

	// Initial stats
	stats := manager.GetStats()
	if stats["total_subscriptions"] != 0 {
		t.Error("Expected 0 total subscriptions initially")
	}

	if stats["active_subscriptions"] != 0 {
		t.Error("Expected 0 active subscriptions initially")
	}

	if stats["running"] != false {
		t.Error("Expected manager not running initially")
	}

	// Add subscriptions
	params := map[string]interface{}{}
	manager.Subscribe("prometheus-metrics", params, callback)
	manager.Subscribe("alerts", params, callback)

	stats = manager.GetStats()
	if stats["total_subscriptions"] != 2 {
		t.Errorf("Expected 2 total subscriptions, got %v", stats["total_subscriptions"])
	}

	if stats["active_subscriptions"] != 2 {
		t.Errorf("Expected 2 active subscriptions, got %v", stats["active_subscriptions"])
	}

	// Check provider counts
	providerCounts, ok := stats["provider_counts"].(map[string]int)
	if !ok {
		t.Error("Expected provider_counts to be map[string]int")
	} else {
		if providerCounts["prometheus-metrics"] != 1 {
			t.Error("Expected 1 prometheus-metrics subscription")
		}
		if providerCounts["alerts"] != 1 {
			t.Error("Expected 1 alerts subscription")
		}
	}
}

func TestValidateProviderID(t *testing.T) {
	manager := NewSubscriptionManager(nil)

	validProviders := []string{
		"prometheus-metrics",
		"alerts",
		"node-metrics",
		"job-metrics",
	}

	for _, providerID := range validProviders {
		if err := manager.validateProviderID(providerID); err != nil {
			t.Errorf("Valid provider %s was rejected: %v", providerID, err)
		}
	}

	invalidProviders := []string{
		"invalid",
		"unknown-provider",
		"",
	}

	for _, providerID := range invalidProviders {
		if err := manager.validateProviderID(providerID); err == nil {
			t.Errorf("Invalid provider %s was accepted", providerID)
		}
	}
}

// func TestSubscriptionWithMockClient(t *testing.T) {
// 	// Create a mock client for testing
// 	mockClient := &MockCachedClient{}
// 	manager := NewSubscriptionManager(mockClient)

// 	callback := func(data interface{}, err error) {
// 		if err != nil {
// 			t.Errorf("Callback received error: %v", err)
// 		}
// 	}

// 	params := map[string]interface{}{
// 		"query": "up",
// 	}

// 	subscriptionID, err := manager.Subscribe("prometheus-metrics", params, callback)
// 	if err != nil {
// 		t.Fatalf("Subscribe failed: %v", err)
// 	}

// 	// Test data retrieval
// 	ctx := context.Background()
// 	data, err := manager.GetData(ctx, "prometheus-metrics", params)
// 	if err != nil {
// 		t.Fatalf("GetData failed: %v", err)
// 	}

// 	if data == nil {
// 		t.Error("Expected data, got nil")
// 	}

// 	// Clean up
// 	manager.Unsubscribe(subscriptionID)
// }

func TestIncrementErrorCount(t *testing.T) {
	manager := NewSubscriptionManager(nil)
	callback := func(data interface{}, err error) {}

	subscriptionID, err := manager.Subscribe("prometheus-metrics", map[string]interface{}{}, callback)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	// Increment error count
	manager.incrementErrorCount(string(subscriptionID), "test error")

	subscription, err := manager.GetSubscription(subscriptionID)
	if err != nil {
		t.Fatalf("GetSubscription failed: %v", err)
	}

	if subscription.ErrorCount != 1 {
		t.Errorf("Expected error count 1, got %d", subscription.ErrorCount)
	}

	if subscription.LastError != "test error" {
		t.Errorf("Expected last error 'test error', got '%s'", subscription.LastError)
	}

	// Increment many times to trigger deactivation
	for i := 0; i < 10; i++ {
		manager.incrementErrorCount(string(subscriptionID), "repeated error")
	}

	subscription, err = manager.GetSubscription(subscriptionID)
	if err != nil {
		t.Fatalf("GetSubscription failed: %v", err)
	}

	if subscription.Active {
		t.Error("Subscription should be deactivated after many errors")
	}
}

// MockCachedClient for testing
type MockCachedClient struct{}

func (m *MockCachedClient) Query(ctx context.Context, query string, ts time.Time) (interface{}, error) {
	return map[string]interface{}{
		"query":  query,
		"result": "mock_result",
	}, nil
}

func (m *MockCachedClient) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (interface{}, error) {
	return map[string]interface{}{
		"query": query,
		"range": "mock_range_result",
	}, nil
}

func (m *MockCachedClient) BatchQuery(ctx context.Context, queries map[string]string, ts time.Time) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	for key, query := range queries {
		results[key] = map[string]interface{}{
			"query":  query,
			"result": "mock_batch_result",
		}
	}
	return results, nil
}

func (m *MockCachedClient) CacheStats() map[string]interface{} {
	return map[string]interface{}{
		"hits":   100,
		"misses": 10,
		"size":   50,
	}
}

func TestConcurrentSubscriptions(t *testing.T) {
	manager := NewSubscriptionManager(nil)
	ctx := context.Background()

	err := manager.Start(ctx)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer manager.Stop()

	// Create subscriptions concurrently
	done := make(chan bool)
	numWorkers := 10

	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer func() { done <- true }()

			callback := func(data interface{}, err error) {}
			params := map[string]interface{}{
				"worker_id": workerID,
			}

			subscriptionID, err := manager.Subscribe("prometheus-metrics", params, callback)
			if err != nil {
				t.Errorf("Worker %d: Subscribe failed: %v", workerID, err)
				return
			}

			// Access subscription data
			_, err = manager.GetSubscription(subscriptionID)
			if err != nil {
				t.Errorf("Worker %d: GetSubscription failed: %v", workerID, err)
			}

			// Unsubscribe
			err = manager.Unsubscribe(subscriptionID)
			if err != nil {
				t.Errorf("Worker %d: Unsubscribe failed: %v", workerID, err)
			}
		}(i)
	}

	// Wait for all workers
	for i := 0; i < numWorkers; i++ {
		<-done
	}

	// Verify final state
	subscriptions := manager.ListSubscriptions()
	if len(subscriptions) != 0 {
		t.Errorf("Expected 0 subscriptions after concurrent test, got %d", len(subscriptions))
	}
}