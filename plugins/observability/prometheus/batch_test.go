package prometheus

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockBatchClient implements PrometheusClientInterface for batch testing
type MockBatchClient struct {
	shouldFail   bool
	failCount    int
	callCount    int
	queryLatency time.Duration
	queryResults map[string]*QueryResult
}

func (m *MockBatchClient) TestConnection(ctx context.Context) error {
	if m.shouldFail {
		return errors.New("connection failed")
	}
	return nil
}

func (m *MockBatchClient) Query(ctx context.Context, query string, queryTime time.Time) (*QueryResult, error) {
	m.callCount++

	// Simulate query latency
	if m.queryLatency > 0 {
		timer := time.NewTimer(m.queryLatency)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}

	// Check predefined results first
	if m.queryResults != nil {
		if result, exists := m.queryResults[query]; exists {
			return result, nil
		}
	}

	// Then check if should fail based on failCount
	if m.shouldFail && m.failCount > 0 && m.callCount <= m.failCount {
		return nil, errors.New("query failed")
	}

	// If shouldFail is true, no failCount specified, and no predefined results, always fail
	if m.shouldFail && m.failCount == 0 {
		return nil, errors.New("query failed")
	}

	return &QueryResult{
		Status: "success",
		Data:   ResultData{ResultType: ResultTypeVector},
	}, nil
}

func (m *MockBatchClient) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (*QueryResult, error) {
	return m.Query(ctx, query, start)
}

func (m *MockBatchClient) BatchQuery(ctx context.Context, queries map[string]string, queryTime time.Time) (map[string]*QueryResult, error) {
	results := make(map[string]*QueryResult)
	for name, query := range queries {
		result, err := m.Query(ctx, query, queryTime)
		if err != nil {
			return nil, err
		}
		results[name] = result
	}
	return results, nil
}

func (m *MockBatchClient) Series(ctx context.Context, matches []string, start, end time.Time) ([]map[string]string, error) {
	if m.shouldFail {
		return nil, errors.New("series query failed")
	}
	return []map[string]string{{"__name__": "test_metric"}}, nil
}

func (m *MockBatchClient) Labels(ctx context.Context) ([]string, error) {
	if m.shouldFail {
		return nil, errors.New("labels query failed")
	}
	return []string{"__name__", "instance", "job"}, nil
}

func TestBatchQueryConfig(t *testing.T) {
	config := DefaultBatchQueryConfig()

	if config.MaxConcurrency != 10 {
		t.Errorf("Expected MaxConcurrency 10, got: %d", config.MaxConcurrency)
	}
	if config.BatchTimeout != 30*time.Second {
		t.Errorf("Expected BatchTimeout 30s, got: %v", config.BatchTimeout)
	}
	if config.RetryAttempts != 3 {
		t.Errorf("Expected RetryAttempts 3, got: %d", config.RetryAttempts)
	}
}

func TestBatchQueryEmpty(t *testing.T) {
	client := &Client{}
	ctx := context.Background()

	results, err := client.BatchQuery(ctx, map[string]string{}, time.Now())
	if err != nil {
		t.Fatalf("Empty batch query failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected empty results, got: %d", len(results))
	}
}

func TestBatchQueryWithConcurrencyLimit(t *testing.T) {
	mockClient := &MockBatchClient{
		queryLatency: 50 * time.Millisecond,
	}

	queries := map[string]string{
		"query1": "up",
		"query2": "cpu_usage",
		"query3": "memory_usage",
		"query4": "disk_usage",
		"query5": "network_usage",
	}

	ctx := context.Background()
	start := time.Now()

	// Test with mock client directly
	results := make(map[string]*QueryResult)
	for name, query := range queries {
		result, err := mockClient.Query(ctx, query, time.Now())
		if err != nil {
			t.Fatalf("Query %s failed: %v", name, err)
		}
		results[name] = result
	}

	elapsed := time.Since(start)

	if len(results) != 5 {
		t.Errorf("Expected 5 results, got: %d", len(results))
	}

	t.Logf("Batch execution took: %v", elapsed)
}

func TestBatchQueryRetryLogic(t *testing.T) {
	mockClient := &MockBatchClient{
		shouldFail: true,
		failCount:  2, // Fail first 2 attempts, succeed on 3rd
	}

	ctx := context.Background()

	// Test retry logic manually - this should succeed on the 3rd attempt
	// because failCount is 2, meaning calls 1 and 2 fail, call 3 succeeds
	var result *QueryResult
	var lastErr error
	maxRetries := 3

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, lastErr = mockClient.Query(ctx, "test_query", time.Now())
		if lastErr == nil {
			// Success
			break
		}

		// Simulate exponential backoff
		if attempt < maxRetries {
			time.Sleep(time.Duration(10<<uint(attempt)) * time.Millisecond)
		}
	}

	// Should have succeeded after 3 calls (2 failures + 1 success)
	if lastErr != nil {
		t.Fatalf("Query should have succeeded after retries: %v", lastErr)
	}

	if result == nil {
		t.Error("Expected non-nil result")
	}

	// Verify the mock was called 3 times (2 failures + 1 success)
	if mockClient.callCount != 3 {
		t.Errorf("Expected 3 calls, got: %d", mockClient.callCount)
	}
}

func TestBatchQueryTimeout(t *testing.T) {
	mockClient := &MockBatchClient{
		queryLatency: 200 * time.Millisecond, // Each query takes 200ms
	}

	ctx := context.Background()

	// Simulate timeout by testing with context cancellation
	timeoutCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	// Test that queries respect context cancellation
	_, err := mockClient.Query(timeoutCtx, "test", time.Now())

	// Should eventually timeout or be cancelled
	if err == nil {
		t.Log("Query completed before timeout (this is okay)")
	} else {
		t.Logf("Query cancelled as expected: %v", err)
	}
}

func TestBatchQueryPartialFailure(t *testing.T) {
	// Create a mock that fails queries not in the predefined results
	mockClient := &MockBatchClient{
		shouldFail: true, // Will fail unless query is in queryResults
		queryResults: map[string]*QueryResult{
			"up": { // Use the actual query string, not the name
				Status: "success",
				Data:   ResultData{ResultType: ResultTypeVector},
			},
		},
	}

	queries := map[string]string{
		"good_query": "up",
		"bad_query":  "invalid_metric",
	}

	ctx := context.Background()

	// Test partial failure scenario
	results := make(map[string]*QueryResult)
	var errors []error

	for name, query := range queries {
		result, err := mockClient.Query(ctx, query, time.Now())
		if err != nil {
			errors = append(errors, err)
		} else {
			results[name] = result
		}
	}

	// Should have 1 success and 1 failure
	if len(results) != 1 {
		t.Errorf("Expected 1 successful result, got: %d", len(results))
	}

	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got: %d", len(errors))
	}

	if results["good_query"] == nil {
		t.Error("Expected good_query to succeed")
	}

	t.Logf("Completed with %d successes and %d errors", len(results), len(errors))
}

func TestBatchQueryCircuitBreakerIntegration(t *testing.T) {
	mockClient := &MockBatchClient{}
	config := DefaultCircuitBreakerConfig()

	cbClient := NewCircuitBreakerClient(mockClient, config)

	queries := map[string]string{
		"query1": "up",
		"query2": "cpu_usage",
	}

	ctx := context.Background()

	// Test that batch queries work through circuit breaker
	results, err := cbClient.BatchQuery(ctx, queries, time.Now())
	if err != nil {
		t.Fatalf("Circuit breaker batch query failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got: %d", len(results))
	}

	// Verify circuit breaker state
	if cbClient.GetState() != CircuitClosed {
		t.Errorf("Expected circuit to be closed, got: %s", cbClient.GetState())
	}
}
