package prometheus

import (
	"context"
	"errors"
	"testing"
	"time"
)

// MockPrometheusClient implements PrometheusClientInterface for testing
type MockPrometheusClient struct {
	shouldFail  bool
	failCount   int
	callCount   int
	queryResult *QueryResult
	queryError  error
}

func (m *MockPrometheusClient) TestConnection(ctx context.Context) error {
	m.callCount++
	if m.shouldFail {
		return errors.New("connection failed")
	}
	return nil
}

func (m *MockPrometheusClient) Query(ctx context.Context, query string, time time.Time) (*QueryResult, error) {
	m.callCount++
	if m.shouldFail && m.callCount <= m.failCount {
		return nil, errors.New("query failed")
	}
	if m.queryError != nil {
		return nil, m.queryError
	}
	if m.queryResult != nil {
		return m.queryResult, nil
	}
	return &QueryResult{Status: "success", Data: ResultData{ResultType: ResultTypeVector}}, nil
}

func (m *MockPrometheusClient) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (*QueryResult, error) {
	return m.Query(ctx, query, start)
}

func (m *MockPrometheusClient) BatchQuery(ctx context.Context, queries map[string]string, time time.Time) (map[string]*QueryResult, error) {
	m.callCount++
	if m.shouldFail {
		return nil, errors.New("batch query failed")
	}
	result := make(map[string]*QueryResult)
	for name := range queries {
		result[name] = &QueryResult{Status: "success", Data: ResultData{ResultType: ResultTypeVector}}
	}
	return result, nil
}

func (m *MockPrometheusClient) Series(ctx context.Context, matches []string, start, end time.Time) ([]map[string]string, error) {
	m.callCount++
	if m.shouldFail {
		return nil, errors.New("series query failed")
	}
	return []map[string]string{{"__name__": "test_metric"}}, nil
}

func (m *MockPrometheusClient) Labels(ctx context.Context) ([]string, error) {
	m.callCount++
	if m.shouldFail {
		return nil, errors.New("labels query failed")
	}
	return []string{"__name__", "instance", "job"}, nil
}

func TestCircuitBreakerBasicOperation(t *testing.T) {
	mockClient := &MockPrometheusClient{}
	config := DefaultCircuitBreakerConfig()

	cbClient := NewCircuitBreakerClient(mockClient, config)

	ctx := context.Background()

	// Test successful operation
	err := cbClient.TestConnection(ctx)
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	// Verify circuit is closed
	if cbClient.GetState() != CircuitClosed {
		t.Errorf("Expected circuit to be closed, got: %s", cbClient.GetState())
	}
}

func TestCircuitBreakerFailureHandling(t *testing.T) {
	mockClient := &MockPrometheusClient{shouldFail: true}
	config := DefaultCircuitBreakerConfig()
	config.ReadyToTrip = func(counts Counts) bool {
		return counts.ConsecutiveFailures >= 3
	}

	cbClient := NewCircuitBreakerClient(mockClient, config)

	ctx := context.Background()

	// Make requests that should fail
	for i := 0; i < 3; i++ {
		err := cbClient.TestConnection(ctx)
		if err == nil {
			t.Errorf("Expected failure on attempt %d, got success", i+1)
		}
	}

	// Circuit should now be open
	if cbClient.GetState() != CircuitOpen {
		t.Errorf("Expected circuit to be open, got: %s", cbClient.GetState())
	}

	// Next request should fail fast
	err := cbClient.TestConnection(ctx)
	if err == nil || err.Error() != "circuit breaker is open" {
		t.Errorf("Expected 'circuit breaker is open' error, got: %v", err)
	}
}

func TestCircuitBreakerHalfOpenState(t *testing.T) {
	mockClient := &MockPrometheusClient{shouldFail: true}
	config := DefaultCircuitBreakerConfig()
	config.ReadyToTrip = func(counts Counts) bool {
		return counts.ConsecutiveFailures >= 2
	}
	config.Timeout = 10 * time.Millisecond // Short timeout for testing

	cbClient := NewCircuitBreakerClient(mockClient, config)

	ctx := context.Background()

	// Trigger circuit opening
	for i := 0; i < 2; i++ {
		_ = cbClient.TestConnection(ctx)
	}

	// Circuit should be open
	if cbClient.GetState() != CircuitOpen {
		t.Errorf("Expected circuit to be open, got: %s", cbClient.GetState())
	}

	// Wait for timeout
	time.Sleep(15 * time.Millisecond)

	// Next request should put circuit in half-open state
	mockClient.shouldFail = false // Allow success
	err := cbClient.TestConnection(ctx)
	if err != nil {
		t.Errorf("Expected success in half-open state, got: %v", err)
	}

	// Circuit should now be closed
	if cbClient.GetState() != CircuitClosed {
		t.Errorf("Expected circuit to be closed after success, got: %s", cbClient.GetState())
	}
}

func TestCircuitBreakerCounts(t *testing.T) {
	mockClient := &MockPrometheusClient{}
	config := DefaultCircuitBreakerConfig()

	cbClient := NewCircuitBreakerClient(mockClient, config)

	ctx := context.Background()

	// Make some successful requests
	for i := 0; i < 3; i++ {
		err := cbClient.TestConnection(ctx)
		if err != nil {
			t.Errorf("Unexpected error on request %d: %v", i+1, err)
		}
	}

	counts := cbClient.GetCounts()
	if counts.Requests != 3 {
		t.Errorf("Expected 3 requests, got: %d", counts.Requests)
	}
	if counts.TotalSuccesses != 3 {
		t.Errorf("Expected 3 successes, got: %d", counts.TotalSuccesses)
	}
	if counts.ConsecutiveSuccesses != 3 {
		t.Errorf("Expected 3 consecutive successes, got: %d", counts.ConsecutiveSuccesses)
	}

	// Make a failing request
	mockClient.shouldFail = true
	_ = cbClient.TestConnection(ctx)

	counts = cbClient.GetCounts()
	if counts.Requests != 4 {
		t.Errorf("Expected 4 requests, got: %d", counts.Requests)
	}
	if counts.TotalFailures != 1 {
		t.Errorf("Expected 1 failure, got: %d", counts.TotalFailures)
	}
	if counts.ConsecutiveFailures != 1 {
		t.Errorf("Expected 1 consecutive failure, got: %d", counts.ConsecutiveFailures)
	}
	if counts.ConsecutiveSuccesses != 0 {
		t.Errorf("Expected 0 consecutive successes after failure, got: %d", counts.ConsecutiveSuccesses)
	}
}

func TestCircuitBreakerQueryOperations(t *testing.T) {
	mockClient := &MockPrometheusClient{}
	config := DefaultCircuitBreakerConfig()

	cbClient := NewCircuitBreakerClient(mockClient, config)

	ctx := context.Background()
	testTime := time.Now()

	// Test Query
	result, err := cbClient.Query(ctx, "test_query", testTime)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("Expected success status, got: %s", result.Status)
	}

	// Test QueryRange
	result, err = cbClient.QueryRange(ctx, "test_query", testTime, testTime.Add(time.Hour), time.Minute)
	if err != nil {
		t.Fatalf("QueryRange failed: %v", err)
	}

	// Test BatchQuery
	queries := map[string]string{"query1": "test1", "query2": "test2"}
	batchResult, err := cbClient.BatchQuery(ctx, queries, testTime)
	if err != nil {
		t.Fatalf("BatchQuery failed: %v", err)
	}
	if len(batchResult) != 2 {
		t.Errorf("Expected 2 results, got: %d", len(batchResult))
	}

	// Test Series
	series, err := cbClient.Series(ctx, []string{"test_metric"}, testTime, testTime.Add(time.Hour))
	if err != nil {
		t.Fatalf("Series failed: %v", err)
	}
	if len(series) != 1 {
		t.Errorf("Expected 1 series, got: %d", len(series))
	}

	// Test Labels
	labels, err := cbClient.Labels(ctx)
	if err != nil {
		t.Fatalf("Labels failed: %v", err)
	}
	if len(labels) != 3 {
		t.Errorf("Expected 3 labels, got: %d", len(labels))
	}
}

func TestCircuitBreakerRecovery(t *testing.T) {
	mockClient := &MockPrometheusClient{shouldFail: true, failCount: 5}
	config := DefaultCircuitBreakerConfig()
	config.ReadyToTrip = func(counts Counts) bool {
		return counts.ConsecutiveFailures >= 3
	}
	config.Timeout = 10 * time.Millisecond

	cbClient := NewCircuitBreakerClient(mockClient, config)

	ctx := context.Background()

	// Fail enough times to open the circuit
	for i := 0; i < 5; i++ {
		_ = cbClient.TestConnection(ctx)
	}

	if cbClient.GetState() != CircuitOpen {
		t.Errorf("Expected circuit to be open, got: %s", cbClient.GetState())
	}

	// Wait for timeout
	time.Sleep(15 * time.Millisecond)

	// Make a request that should succeed (mockClient allows success after failCount)
	mockClient.shouldFail = false
	err := cbClient.TestConnection(ctx)
	if err != nil {
		t.Errorf("Expected recovery success, got: %v", err)
	}

	// Circuit should be closed now
	if cbClient.GetState() != CircuitClosed {
		t.Errorf("Expected circuit to be closed after recovery, got: %s", cbClient.GetState())
	}
}
