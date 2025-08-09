package auth

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/jontk/s9s/internal/debug"
)

// AdvancedLoadBalancer provides sophisticated load balancing with health checking
type AdvancedLoadBalancer struct {
	strategy      string
	endpoints     map[string]*ManagedEndpoint
	mutex         sync.RWMutex
	healthChecker *HealthChecker
	config        LoadBalancerConfig
}

// ManagedEndpoint wraps an Endpoint with additional load balancing metadata
type ManagedEndpoint struct {
	Endpoint
	consecutiveFailures int
	lastSuccess         time.Time
	lastFailure         time.Time
	responseTime        time.Duration
	activeConnections   int32
	circuitBreakerOpen  bool
	circuitBreakerUntil time.Time
}

// LoadBalancerConfig configures the load balancer behavior
type LoadBalancerConfig struct {
	Strategy                string
	MaxConsecutiveFailures  int
	CircuitBreakerTimeout   time.Duration
	HealthCheckInterval     time.Duration
	HealthCheckTimeout      time.Duration
	HealthCheckPath         string
	ResponseTimeWeight      float64
	WeightUpdateInterval    time.Duration
}

// HealthChecker performs periodic health checks on endpoints
type HealthChecker struct {
	client      *http.Client
	config      LoadBalancerConfig
	balancer    *AdvancedLoadBalancer
	stopChan    chan struct{}
	wg          sync.WaitGroup
	mutex       sync.RWMutex
}

// NewAdvancedLoadBalancer creates a new advanced load balancer
func NewAdvancedLoadBalancer(config LoadBalancerConfig) LoadBalancer {
	if config.Strategy == "" {
		config.Strategy = "weighted-least-connections"
	}
	if config.MaxConsecutiveFailures == 0 {
		config.MaxConsecutiveFailures = 3
	}
	if config.CircuitBreakerTimeout == 0 {
		config.CircuitBreakerTimeout = 30 * time.Second
	}
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 30 * time.Second
	}
	if config.HealthCheckTimeout == 0 {
		config.HealthCheckTimeout = 10 * time.Second
	}
	if config.HealthCheckPath == "" {
		config.HealthCheckPath = "/slurm/v0.0.40/ping"
	}
	if config.ResponseTimeWeight == 0 {
		config.ResponseTimeWeight = 0.3
	}

	balancer := &AdvancedLoadBalancer{
		strategy:  config.Strategy,
		endpoints: make(map[string]*ManagedEndpoint),
		config:    config,
	}

	balancer.healthChecker = &HealthChecker{
		client: &http.Client{
			Timeout: config.HealthCheckTimeout,
		},
		config:   config,
		balancer: balancer,
		stopChan: make(chan struct{}),
	}

	return balancer
}

// SelectEndpoint selects the best endpoint using the configured strategy
func (a *AdvancedLoadBalancer) SelectEndpoint(endpoints []Endpoint) (*Endpoint, error) {
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints available")
	}

	// Update managed endpoints
	a.updateManagedEndpoints(endpoints)

	// Filter healthy endpoints
	candidates := a.getHealthyEndpoints()
	if len(candidates) == 0 {
		debug.Logger.Printf("No healthy endpoints available, falling back to all endpoints")
		candidates = a.getAllManagedEndpoints()
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no endpoints available after filtering")
	}

	// Select endpoint based on strategy
	var selected *ManagedEndpoint
	switch a.config.Strategy {
	case "round-robin":
		selected = a.selectRoundRobin(candidates)
	case "weighted-round-robin":
		selected = a.selectWeightedRoundRobin(candidates)
	case "least-connections":
		selected = a.selectLeastConnections(candidates)
	case "weighted-least-connections":
		selected = a.selectWeightedLeastConnections(candidates)
	case "response-time":
		selected = a.selectByResponseTime(candidates)
	case "random":
		selected = a.selectRandom(candidates)
	default:
		selected = a.selectWeightedLeastConnections(candidates)
	}

	if selected == nil {
		return nil, fmt.Errorf("failed to select endpoint")
	}

	debug.Logger.Printf("Selected endpoint: %s (strategy: %s, weight: %d, connections: %d, response_time: %v)",
		selected.URL, a.config.Strategy, selected.Weight, selected.activeConnections, selected.responseTime)

	return &selected.Endpoint, nil
}

// updateManagedEndpoints synchronizes the managed endpoints with the provided list
func (a *AdvancedLoadBalancer) updateManagedEndpoints(endpoints []Endpoint) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Create a map of current endpoints
	current := make(map[string]bool)
	for _, ep := range endpoints {
		current[ep.URL] = true

		// Update or create managed endpoint
		if managed, exists := a.endpoints[ep.URL]; exists {
			// Update existing endpoint
			managed.Endpoint = ep
		} else {
			// Create new managed endpoint
			a.endpoints[ep.URL] = &ManagedEndpoint{
				Endpoint:    ep,
				lastSuccess: time.Now(),
			}
		}
	}

	// Remove endpoints that are no longer in the list
	for url := range a.endpoints {
		if !current[url] {
			delete(a.endpoints, url)
			debug.Logger.Printf("Removed endpoint from load balancer: %s", url)
		}
	}
}

// getHealthyEndpoints returns endpoints that are healthy and not circuit broken
func (a *AdvancedLoadBalancer) getHealthyEndpoints() []*ManagedEndpoint {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	var healthy []*ManagedEndpoint
	now := time.Now()

	for _, managed := range a.endpoints {
		// Skip unhealthy endpoints
		if managed.Status == EndpointStatusUnhealthy {
			continue
		}

		// Check circuit breaker
		if managed.circuitBreakerOpen && now.Before(managed.circuitBreakerUntil) {
			continue
		}

		// Reset circuit breaker if timeout has passed
		if managed.circuitBreakerOpen && now.After(managed.circuitBreakerUntil) {
			managed.circuitBreakerOpen = false
			managed.consecutiveFailures = 0
			debug.Logger.Printf("Circuit breaker reset for endpoint: %s", managed.URL)
		}

		healthy = append(healthy, managed)
	}

	return healthy
}

// getAllManagedEndpoints returns all managed endpoints
func (a *AdvancedLoadBalancer) getAllManagedEndpoints() []*ManagedEndpoint {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	var all []*ManagedEndpoint
	for _, managed := range a.endpoints {
		all = append(all, managed)
	}
	return all
}

// Selection strategies

// selectRoundRobin implements simple round-robin selection
func (a *AdvancedLoadBalancer) selectRoundRobin(candidates []*ManagedEndpoint) *ManagedEndpoint {
	// Simple implementation - in production would maintain counter
	index := int(time.Now().UnixNano()) % len(candidates)
	return candidates[index]
}

// selectWeightedRoundRobin implements weighted round-robin selection
func (a *AdvancedLoadBalancer) selectWeightedRoundRobin(candidates []*ManagedEndpoint) *ManagedEndpoint {
	totalWeight := 0
	for _, candidate := range candidates {
		totalWeight += candidate.Weight
	}

	if totalWeight == 0 {
		return a.selectRoundRobin(candidates)
	}

	// Weighted random selection
	target := rand.Intn(totalWeight)
	current := 0

	for _, candidate := range candidates {
		current += candidate.Weight
		if current > target {
			return candidate
		}
	}

	return candidates[0] // Fallback
}

// selectLeastConnections selects the endpoint with fewest active connections
func (a *AdvancedLoadBalancer) selectLeastConnections(candidates []*ManagedEndpoint) *ManagedEndpoint {
	var best *ManagedEndpoint
	minConnections := int32(1<<31 - 1) // Max int32

	for _, candidate := range candidates {
		if candidate.activeConnections < minConnections {
			minConnections = candidate.activeConnections
			best = candidate
		}
	}

	return best
}

// selectWeightedLeastConnections combines least connections with weights
func (a *AdvancedLoadBalancer) selectWeightedLeastConnections(candidates []*ManagedEndpoint) *ManagedEndpoint {
	var best *ManagedEndpoint
	var bestScore float64 = -1

	for _, candidate := range candidates {
		// Calculate score: weight / (connections + 1)
		score := float64(candidate.Weight) / float64(candidate.activeConnections+1)

		// Factor in response time if available
		if candidate.responseTime > 0 {
			responseTimeFactor := 1.0 / (1.0 + candidate.responseTime.Seconds()*a.config.ResponseTimeWeight)
			score *= responseTimeFactor
		}

		if score > bestScore {
			bestScore = score
			best = candidate
		}
	}

	return best
}

// selectByResponseTime selects the endpoint with best response time
func (a *AdvancedLoadBalancer) selectByResponseTime(candidates []*ManagedEndpoint) *ManagedEndpoint {
	var best *ManagedEndpoint
	var bestTime time.Duration = time.Hour // Start with a large value

	for _, candidate := range candidates {
		if candidate.responseTime > 0 && candidate.responseTime < bestTime {
			bestTime = candidate.responseTime
			best = candidate
		}
	}

	// If no response times available, fall back to least connections
	if best == nil {
		return a.selectLeastConnections(candidates)
	}

	return best
}

// selectRandom selects a random endpoint
func (a *AdvancedLoadBalancer) selectRandom(candidates []*ManagedEndpoint) *ManagedEndpoint {
	index := rand.Intn(len(candidates))
	return candidates[index]
}

// UpdateEndpointHealth updates the health status and metrics of an endpoint
func (a *AdvancedLoadBalancer) UpdateEndpointHealth(endpoint *Endpoint, healthy bool) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	managed, exists := a.endpoints[endpoint.URL]
	if !exists {
		return
	}

	now := time.Now()
	managed.LastCheck = now

	if healthy {
		managed.Status = EndpointStatusHealthy
		managed.lastSuccess = now
		managed.consecutiveFailures = 0

		// Reset circuit breaker on successful health check
		if managed.circuitBreakerOpen {
			managed.circuitBreakerOpen = false
			debug.Logger.Printf("Circuit breaker reset after successful health check: %s", managed.URL)
		}
	} else {
		managed.Status = EndpointStatusUnhealthy
		managed.lastFailure = now
		managed.consecutiveFailures++

		// Open circuit breaker if too many consecutive failures
		if managed.consecutiveFailures >= a.config.MaxConsecutiveFailures && !managed.circuitBreakerOpen {
			managed.circuitBreakerOpen = true
			managed.circuitBreakerUntil = now.Add(a.config.CircuitBreakerTimeout)
			debug.Logger.Printf("Circuit breaker opened for endpoint %s after %d consecutive failures",
				managed.URL, managed.consecutiveFailures)
		}
	}

	// Update the original endpoint status
	endpoint.Status = managed.Status
	endpoint.LastCheck = managed.LastCheck

	debug.Logger.Printf("Updated endpoint %s health status: %s (failures: %d, circuit_open: %v)",
		managed.URL, managed.Status.String(), managed.consecutiveFailures, managed.circuitBreakerOpen)
}

// RecordResponseTime records the response time for an endpoint
func (a *AdvancedLoadBalancer) RecordResponseTime(endpoint *Endpoint, responseTime time.Duration) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	managed, exists := a.endpoints[endpoint.URL]
	if !exists {
		return
	}

	// Update response time with exponential smoothing
	if managed.responseTime == 0 {
		managed.responseTime = responseTime
	} else {
		// Exponential moving average (alpha = 0.3)
		alpha := 0.3
		managed.responseTime = time.Duration(float64(managed.responseTime)*(1-alpha) + float64(responseTime)*alpha)
	}

	debug.Logger.Printf("Updated response time for endpoint %s: %v", managed.URL, managed.responseTime)
}

// IncrementActiveConnections increments the active connection count
func (a *AdvancedLoadBalancer) IncrementActiveConnections(endpoint *Endpoint) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if managed, exists := a.endpoints[endpoint.URL]; exists {
		managed.activeConnections++
	}
}

// DecrementActiveConnections decrements the active connection count
func (a *AdvancedLoadBalancer) DecrementActiveConnections(endpoint *Endpoint) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if managed, exists := a.endpoints[endpoint.URL]; exists && managed.activeConnections > 0 {
		managed.activeConnections--
	}
}

// GetStrategy returns the load balancing strategy name
func (a *AdvancedLoadBalancer) GetStrategy() string {
	return a.config.Strategy
}

// GetEndpointStats returns statistics for all endpoints
func (a *AdvancedLoadBalancer) GetEndpointStats() map[string]EndpointStats {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	stats := make(map[string]EndpointStats)
	for url, managed := range a.endpoints {
		stats[url] = EndpointStats{
			URL:                  url,
			Status:               managed.Status.String(),
			Weight:               managed.Weight,
			ActiveConnections:    managed.activeConnections,
			ConsecutiveFailures:  managed.consecutiveFailures,
			ResponseTime:         managed.responseTime,
			LastSuccess:          managed.lastSuccess,
			LastFailure:          managed.lastFailure,
			CircuitBreakerOpen:   managed.circuitBreakerOpen,
			CircuitBreakerUntil:  managed.circuitBreakerUntil,
		}
	}

	return stats
}

// EndpointStats contains statistics for an endpoint
type EndpointStats struct {
	URL                 string
	Status              string
	Weight              int
	ActiveConnections   int32
	ConsecutiveFailures int
	ResponseTime        time.Duration
	LastSuccess         time.Time
	LastFailure         time.Time
	CircuitBreakerOpen  bool
	CircuitBreakerUntil time.Time
}

// Health Checker implementation

// StartHealthChecking starts periodic health checks
func (h *HealthChecker) Start(ctx context.Context) {
	h.wg.Add(1)
	go h.healthCheckLoop(ctx)
	debug.Logger.Printf("Started health checker with interval: %v", h.config.HealthCheckInterval)
}

// Stop stops the health checker
func (h *HealthChecker) Stop() {
	close(h.stopChan)
	h.wg.Wait()
	debug.Logger.Printf("Stopped health checker")
}

// healthCheckLoop runs the periodic health check loop
func (h *HealthChecker) healthCheckLoop(ctx context.Context) {
	defer h.wg.Done()

	ticker := time.NewTicker(h.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.performHealthChecks(ctx)
		case <-h.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// performHealthChecks performs health checks on all endpoints
func (h *HealthChecker) performHealthChecks(ctx context.Context) {
	endpoints := h.balancer.getAllManagedEndpoints()

	// Check endpoints in parallel
	var wg sync.WaitGroup
	for _, endpoint := range endpoints {
		wg.Add(1)
		go func(ep *ManagedEndpoint) {
			defer wg.Done()
			h.checkEndpointHealth(ctx, &ep.Endpoint)
		}(endpoint)
	}
	wg.Wait()
}

// checkEndpointHealth performs a health check on a single endpoint
func (h *HealthChecker) checkEndpointHealth(ctx context.Context, endpoint *Endpoint) {
	healthURL := endpoint.URL + h.config.HealthCheckPath

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		debug.Logger.Printf("Health check request creation failed for %s: %v", endpoint.URL, err)
		h.balancer.UpdateEndpointHealth(endpoint, false)
		return
	}

	req.Header.Set("User-Agent", "s9s-health-check/1.0")

	resp, err := h.client.Do(req)
	responseTime := time.Since(start)

	if err != nil {
		debug.Logger.Printf("Health check failed for %s: %v", endpoint.URL, err)
		h.balancer.UpdateEndpointHealth(endpoint, false)
		return
	}
	defer resp.Body.Close()

	healthy := resp.StatusCode >= 200 && resp.StatusCode < 400
	h.balancer.UpdateEndpointHealth(endpoint, healthy)

	// Record response time for healthy endpoints
	if healthy {
		h.balancer.RecordResponseTime(endpoint, responseTime)
	}

	debug.Logger.Printf("Health check completed for %s: status=%d, healthy=%v, response_time=%v",
		endpoint.URL, resp.StatusCode, healthy, responseTime)
}