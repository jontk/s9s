package prometheus

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	// CircuitClosed - Circuit is closed, requests pass through
	CircuitClosed CircuitState = iota
	// CircuitOpen - Circuit is open, requests fail fast
	CircuitOpen
	// CircuitHalfOpen - Circuit is half-open, testing if service recovered
	CircuitHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreakerConfig contains configuration for the circuit breaker
type CircuitBreakerConfig struct {
	// MaxRequests is the maximum number of requests allowed to pass through when the CircuitBreaker is half-open
	MaxRequests uint32
	// Interval is the cyclic period of the closed state for the CircuitBreaker to clear the internal Counts
	Interval time.Duration
	// Timeout is the period of the open state, after which the state of the CircuitBreaker becomes half-open
	Timeout time.Duration
	// ReadyToTrip returns true when the CircuitBreaker should trip and become open
	ReadyToTrip func(counts Counts) bool
	// OnStateChange is called whenever the state of the CircuitBreaker changes
	OnStateChange func(name string, from CircuitState, to CircuitState)
}

// DefaultCircuitBreakerConfig returns a default circuit breaker configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxRequests: 1,
		Interval:    60 * time.Second,
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
		OnStateChange: func(name string, from CircuitState, to CircuitState) {
			// Default: do nothing
		},
	}
}

// Counts holds the numbers of requests and their successes/failures
type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// OnRequest is called on every request
func (c *Counts) OnRequest() {
	c.Requests++
}

// OnSuccess is called on every successful request
func (c *Counts) OnSuccess() {
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0
}

// OnFailure is called on every failed request
func (c *Counts) OnFailure() {
	c.TotalFailures++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0
}

// Clear resets the counts
func (c *Counts) Clear() {
	c.Requests = 0
	c.TotalSuccesses = 0
	c.TotalFailures = 0
	c.ConsecutiveSuccesses = 0
	c.ConsecutiveFailures = 0
}

// CircuitBreaker prevents cascading failures by temporarily stopping calls to a failing service
type CircuitBreaker struct {
	name       string
	config     CircuitBreakerConfig
	mutex      sync.RWMutex
	state      CircuitState
	counts     Counts
	expiry     time.Time
	generation uint32
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, config CircuitBreakerConfig) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:   name,
		config: config,
		state:  CircuitClosed,
		expiry: time.Now().Add(config.Interval),
	}

	return cb
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() CircuitState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	now := time.Now()
	state, _ := cb.currentState(now)
	return state
}

// Counts returns a copy of the current counts
func (cb *CircuitBreaker) Counts() Counts {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()

	return cb.counts
}

// Execute wraps a function call with circuit breaker logic
func (cb *CircuitBreaker) Execute(ctx context.Context, req func() (interface{}, error)) (interface{}, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	defer func() {
		if r := recover(); r != nil {
			cb.afterRequest(generation, false)
			panic(r)
		}
	}()

	result, err := req()
	cb.afterRequest(generation, err == nil)
	return result, err
}

// beforeRequest is called before making a request
func (cb *CircuitBreaker) beforeRequest() (uint32, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == CircuitOpen {
		return generation, fmt.Errorf("circuit breaker is open")
	} else if state == CircuitHalfOpen && cb.counts.Requests >= cb.config.MaxRequests {
		return generation, fmt.Errorf("too many requests in half-open state")
	}

	cb.counts.OnRequest()
	return generation, nil
}

// afterRequest is called after making a request
func (cb *CircuitBreaker) afterRequest(before uint32, success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

// onSuccess is called when a request succeeds
func (cb *CircuitBreaker) onSuccess(state CircuitState, now time.Time) {
	cb.counts.OnSuccess()

	if state == CircuitHalfOpen {
		cb.setState(CircuitClosed, now)
	}
}

// onFailure is called when a request fails
func (cb *CircuitBreaker) onFailure(state CircuitState, now time.Time) {
	cb.counts.OnFailure()

	switch state {
	case CircuitClosed:
		if cb.config.ReadyToTrip(cb.counts) {
			cb.setState(CircuitOpen, now)
		}
	case CircuitHalfOpen:
		cb.setState(CircuitOpen, now)
	}
}

// currentState returns the current state and generation
func (cb *CircuitBreaker) currentState(now time.Time) (CircuitState, uint32) {
	switch cb.state {
	case CircuitClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case CircuitOpen:
		if cb.expiry.Before(now) {
			cb.setState(CircuitHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

// setState changes the state of the circuit breaker
func (cb *CircuitBreaker) setState(state CircuitState, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	cb.toNewGeneration(now)

	if cb.config.OnStateChange != nil {
		cb.config.OnStateChange(cb.name, prev, state)
	}
}

// toNewGeneration resets the counts and sets the expiry time
func (cb *CircuitBreaker) toNewGeneration(now time.Time) {
	cb.counts.Clear()
	cb.generation++

	var zero time.Time
	switch cb.state {
	case CircuitClosed:
		if cb.config.Interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.config.Interval)
		}
	case CircuitOpen:
		cb.expiry = now.Add(cb.config.Timeout)
	default: // CircuitHalfOpen
		cb.expiry = zero
	}
}

// CircuitBreakerClient wraps a Prometheus client with circuit breaker functionality
type CircuitBreakerClient struct {
	client  PrometheusClientInterface
	breaker *CircuitBreaker
}

// NewCircuitBreakerClient creates a new circuit breaker client
func NewCircuitBreakerClient(client PrometheusClientInterface, config CircuitBreakerConfig) *CircuitBreakerClient {
	if config.MaxRequests == 0 {
		config = DefaultCircuitBreakerConfig()
	}

	breaker := NewCircuitBreaker("prometheus-client", config)

	return &CircuitBreakerClient{
		client:  client,
		breaker: breaker,
	}
}

// TestConnection tests the connection to Prometheus with circuit breaker protection
func (cb *CircuitBreakerClient) TestConnection(ctx context.Context) error {
	_, err := cb.breaker.Execute(ctx, func() (interface{}, error) {
		return nil, cb.client.TestConnection(ctx)
	})
	return err
}

// Query executes a Prometheus query with circuit breaker protection
func (cb *CircuitBreakerClient) Query(ctx context.Context, query string, time time.Time) (*QueryResult, error) {
	result, err := cb.breaker.Execute(ctx, func() (interface{}, error) {
		return cb.client.Query(ctx, query, time)
	})
	if err != nil {
		return nil, err
	}
	return result.(*QueryResult), nil
}

// QueryRange executes a Prometheus range query with circuit breaker protection
func (cb *CircuitBreakerClient) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (*QueryResult, error) {
	result, err := cb.breaker.Execute(ctx, func() (interface{}, error) {
		return cb.client.QueryRange(ctx, query, start, end, step)
	})
	if err != nil {
		return nil, err
	}
	return result.(*QueryResult), nil
}

// BatchQuery executes multiple queries with circuit breaker protection
func (cb *CircuitBreakerClient) BatchQuery(ctx context.Context, queries map[string]string, queryTime time.Time) (map[string]*QueryResult, error) {
	result, err := cb.breaker.Execute(ctx, func() (interface{}, error) {
		return cb.client.BatchQuery(ctx, queries, queryTime)
	})
	if err != nil {
		return nil, err
	}
	return result.(map[string]*QueryResult), nil
}

// BatchQueryWithConfig executes multiple queries with custom configuration and circuit breaker protection
func (cb *CircuitBreakerClient) BatchQueryWithConfig(ctx context.Context, queries map[string]string, queryTime time.Time, config BatchQueryConfig) (map[string]*QueryResult, error) {
	result, err := cb.breaker.Execute(ctx, func() (interface{}, error) {
		// Check if the underlying client supports BatchQueryWithConfig
		if clientWithConfig, ok := cb.client.(interface {
			BatchQueryWithConfig(context.Context, map[string]string, time.Time, BatchQueryConfig) (map[string]*QueryResult, error)
		}); ok {
			return clientWithConfig.BatchQueryWithConfig(ctx, queries, queryTime, config)
		}
		// Fallback to regular BatchQuery
		return cb.client.BatchQuery(ctx, queries, queryTime)
	})
	if err != nil {
		return nil, err
	}
	return result.(map[string]*QueryResult), nil
}

// Series returns the list of time series that match a label set with circuit breaker protection
func (cb *CircuitBreakerClient) Series(ctx context.Context, matches []string, start, end time.Time) ([]map[string]string, error) {
	result, err := cb.breaker.Execute(ctx, func() (interface{}, error) {
		return cb.client.Series(ctx, matches, start, end)
	})
	if err != nil {
		return nil, err
	}
	return result.([]map[string]string), nil
}

// Labels returns the list of label names with circuit breaker protection
func (cb *CircuitBreakerClient) Labels(ctx context.Context) ([]string, error) {
	result, err := cb.breaker.Execute(ctx, func() (interface{}, error) {
		return cb.client.Labels(ctx)
	})
	if err != nil {
		return nil, err
	}
	return result.([]string), nil
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreakerClient) GetState() CircuitState {
	return cb.breaker.State()
}

// GetCounts returns the current circuit breaker counts
func (cb *CircuitBreakerClient) GetCounts() Counts {
	return cb.breaker.Counts()
}
