package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jontk/s9s/internal/debug"
)

// StaticEndpointDiscoverer implements static endpoint discovery from configuration
type StaticEndpointDiscoverer struct {
	config    DiscoveryConfig
	endpoints map[string][]Endpoint
	balancer  LoadBalancer
}

// NewStaticEndpointDiscoverer creates a new static endpoint discoverer
func NewStaticEndpointDiscoverer() EndpointDiscoverer {
	return &StaticEndpointDiscoverer{
		endpoints: make(map[string][]Endpoint),
		balancer:  NewRoundRobinLoadBalancer(),
	}
}

// GetInfo returns information about this discoverer
func (s *StaticEndpointDiscoverer) GetInfo() DiscovererInfo {
	return DiscovererInfo{
		Name:        "static",
		Version:     "1.0.0",
		Description: "Static endpoint discovery from configuration",
		Author:      "s9s Team",
		Supported:   []string{"static", "manual"},
	}
}

// Initialize initializes the static endpoint discoverer
func (s *StaticEndpointDiscoverer) Initialize(_ context.Context, config DiscoveryConfig) error {
	s.config = config

	// Parse static endpoints from configuration
	if err := s.parseEndpoints(config); err != nil {
		return fmt.Errorf("failed to parse static endpoints: %w", err)
	}

	debug.Logger.Printf("Initialized static endpoint discoverer with %d clusters", len(s.endpoints))
	return nil
}

// parseEndpoints parses the static endpoint configuration
func (s *StaticEndpointDiscoverer) parseEndpoints(config DiscoveryConfig) error {
	endpoints := config.Get("endpoints")
	if endpoints == nil {
		return fmt.Errorf("no endpoints configured")
	}

	endpointsMap, ok := endpoints.(map[string]interface{})
	if !ok {
		return fmt.Errorf("endpoints must be a map of cluster_id -> endpoint list")
	}

	for clusterID, clusterEndpoints := range endpointsMap {
		endpointList, ok := clusterEndpoints.([]interface{})
		if !ok {
			return fmt.Errorf("endpoints for cluster %s must be a list", clusterID)
		}

		var parsedEndpoints []Endpoint
		for i, ep := range endpointList {
			endpoint, err := s.parseEndpoint(clusterID, ep, i)
			if err != nil {
				return fmt.Errorf("failed to parse endpoint %d for cluster %s: %w", i, clusterID, err)
			}
			parsedEndpoints = append(parsedEndpoints, *endpoint)
		}

		s.endpoints[clusterID] = parsedEndpoints
		debug.Logger.Printf("Configured %d endpoints for cluster %s", len(parsedEndpoints), clusterID)
	}

	return nil
}

// parseEndpoint parses a single endpoint configuration
func (s *StaticEndpointDiscoverer) parseEndpoint(clusterID string, ep interface{}, _ int) (*Endpoint, error) {
	switch endpoint := ep.(type) {
	case string:
		return s.parseStringEndpoint(endpoint, clusterID)
	case map[string]interface{}:
		return s.parseMapEndpoint(endpoint, clusterID)
	default:
		return nil, fmt.Errorf("endpoint must be either a string URL or an object with 'url' field")
	}
}

// parseStringEndpoint creates an endpoint from a simple URL string
func (s *StaticEndpointDiscoverer) parseStringEndpoint(url, clusterID string) (*Endpoint, error) {
	return &Endpoint{
		URL:       url,
		ClusterID: clusterID,
		Weight:    100,
		Status:    EndpointStatusUnknown,
		Metadata:  make(map[string]string),
		LastCheck: time.Time{},
	}, nil
}

// parseMapEndpoint creates an endpoint from a map with URL and metadata
func (s *StaticEndpointDiscoverer) parseMapEndpoint(endpoint map[string]interface{}, clusterID string) (*Endpoint, error) {
	url, ok := endpoint["url"].(string)
	if !ok {
		return nil, fmt.Errorf("endpoint must have a 'url' field")
	}

	weight := s.extractWeight(endpoint)
	metadata := s.extractMetadata(endpoint)

	return &Endpoint{
		URL:       url,
		ClusterID: clusterID,
		Weight:    weight,
		Status:    EndpointStatusUnknown,
		Metadata:  metadata,
		LastCheck: time.Time{},
	}, nil
}

// extractWeight extracts the weight from endpoint config with fallback to 100
func (s *StaticEndpointDiscoverer) extractWeight(endpoint map[string]interface{}) int {
	if w, ok := endpoint["weight"].(int); ok {
		return w
	}
	if w, ok := endpoint["weight"].(float64); ok {
		return int(w)
	}
	return 100
}

// extractMetadata extracts metadata and tags from endpoint config
func (s *StaticEndpointDiscoverer) extractMetadata(endpoint map[string]interface{}) map[string]string {
	metadata := make(map[string]string)

	// Add explicit metadata
	if meta, ok := endpoint["metadata"].(map[string]interface{}); ok {
		for k, v := range meta {
			if str, ok := v.(string); ok {
				metadata[k] = str
			}
		}
	}

	// Add tags as comma-separated metadata
	if tags := s.extractTags(endpoint); len(tags) > 0 {
		metadata["tags"] = fmt.Sprintf("%v", tags)
	}

	return metadata
}

// extractTags extracts tags from endpoint config as string array
func (s *StaticEndpointDiscoverer) extractTags(endpoint map[string]interface{}) []string {
	if tags, ok := endpoint["tags"].([]interface{}); ok {
		var tagStrings []string
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				tagStrings = append(tagStrings, tagStr)
			}
		}
		return tagStrings
	}
	return nil
}

// DiscoverEndpoints returns the statically configured endpoints for a cluster
func (s *StaticEndpointDiscoverer) DiscoverEndpoints(_ context.Context, clusterID string) ([]Endpoint, error) {
	endpoints, exists := s.endpoints[clusterID]
	if !exists {
		return nil, fmt.Errorf("no endpoints configured for cluster %s", clusterID)
	}

	// Return a copy to avoid external modifications
	result := make([]Endpoint, len(endpoints))
	copy(result, endpoints)

	debug.Logger.Printf("Discovered %d static endpoints for cluster %s", len(result), clusterID)
	return result, nil
}

// HealthCheck performs a health check on an endpoint
func (s *StaticEndpointDiscoverer) HealthCheck(ctx context.Context, endpoint Endpoint) error {
	debug.Logger.Printf("Health checking endpoint: %s", endpoint.URL)

	// Determine health check path
	healthPath := s.config.GetString("health_check_path")
	if healthPath == "" {
		healthPath = "/slurm/v0.0.40/ping" // Default SLURM REST API ping endpoint
	}

	healthURL := endpoint.URL + healthPath

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create health check request
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	req.Header.Set("User-Agent", "s9s-health-check/1.0")

	// Execute health check
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	debug.Logger.Printf("Health check passed for endpoint: %s", endpoint.URL)
	return nil
}

// GetLoadBalancer returns the load balancer for endpoint selection
func (s *StaticEndpointDiscoverer) GetLoadBalancer() LoadBalancer {
	return s.balancer
}

// Cleanup performs any necessary cleanup
func (s *StaticEndpointDiscoverer) Cleanup() error {
	debug.Logger.Printf("Static endpoint discoverer cleanup completed")
	return nil
}

// RoundRobinLoadBalancer implements a simple round-robin load balancer
type RoundRobinLoadBalancer struct {
	current int
}

// NewRoundRobinLoadBalancer creates a new round-robin load balancer
func NewRoundRobinLoadBalancer() LoadBalancer {
	return &RoundRobinLoadBalancer{}
}

// SelectEndpoint selects the next endpoint using round-robin
func (r *RoundRobinLoadBalancer) SelectEndpoint(endpoints []Endpoint) (*Endpoint, error) {
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints available")
	}

	// Filter out unhealthy endpoints
	healthy := make([]Endpoint, 0)
	for _, ep := range endpoints {
		if ep.Status != EndpointStatusUnhealthy {
			healthy = append(healthy, ep)
		}
	}

	// If no healthy endpoints, use all endpoints as fallback
	if len(healthy) == 0 {
		healthy = endpoints
	}

	// Simple round-robin selection
	selected := &healthy[r.current%len(healthy)]
	r.current++

	debug.Logger.Printf("Selected endpoint: %s (weight: %d, status: %s)", selected.URL, selected.Weight, selected.Status.String())
	return selected, nil
}

// UpdateEndpointHealth updates the health status of an endpoint
func (r *RoundRobinLoadBalancer) UpdateEndpointHealth(endpoint *Endpoint, healthy bool) {
	if healthy {
		endpoint.Status = EndpointStatusHealthy
	} else {
		endpoint.Status = EndpointStatusUnhealthy
	}
	endpoint.LastCheck = time.Now()

	debug.Logger.Printf("Updated endpoint %s health status to: %s", endpoint.URL, endpoint.Status.String())
}

// GetStrategy returns the load balancing strategy name
func (r *RoundRobinLoadBalancer) GetStrategy() string {
	return "round-robin"
}

// WeightedRoundRobinLoadBalancer implements weighted round-robin load balancing
type WeightedRoundRobinLoadBalancer struct {
	weights map[string]int
	current map[string]int
}

// NewWeightedRoundRobinLoadBalancer creates a new weighted round-robin load balancer
func NewWeightedRoundRobinLoadBalancer() LoadBalancer {
	return &WeightedRoundRobinLoadBalancer{
		weights: make(map[string]int),
		current: make(map[string]int),
	}
}

// SelectEndpoint selects an endpoint using weighted round-robin
func (w *WeightedRoundRobinLoadBalancer) SelectEndpoint(endpoints []Endpoint) (*Endpoint, error) {
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("no endpoints available")
	}

	// Filter healthy endpoints
	healthy := make([]Endpoint, 0)
	totalWeight := 0
	for _, ep := range endpoints {
		if ep.Status != EndpointStatusUnhealthy {
			healthy = append(healthy, ep)
			totalWeight += ep.Weight
		}
	}

	if len(healthy) == 0 {
		healthy = endpoints
		for _, ep := range healthy {
			totalWeight += ep.Weight
		}
	}

	// Select based on weights
	if totalWeight == 0 {
		// Equal weights fallback
		selected := &healthy[0]
		debug.Logger.Printf("Selected endpoint (equal weights): %s", selected.URL)
		return selected, nil
	}

	// Simple weighted selection (can be improved with proper weighted round-robin algorithm)
	maxWeight := 0
	var selected *Endpoint
	for i := range healthy {
		ep := &healthy[i]
		if ep.Weight > maxWeight {
			maxWeight = ep.Weight
			selected = ep
		}
	}

	if selected == nil {
		selected = &healthy[0]
	}

	debug.Logger.Printf("Selected endpoint (weighted): %s (weight: %d)", selected.URL, selected.Weight)
	return selected, nil
}

// UpdateEndpointHealth updates the health status of an endpoint
func (w *WeightedRoundRobinLoadBalancer) UpdateEndpointHealth(endpoint *Endpoint, healthy bool) {
	if healthy {
		endpoint.Status = EndpointStatusHealthy
	} else {
		endpoint.Status = EndpointStatusUnhealthy
	}
	endpoint.LastCheck = time.Now()

	debug.Logger.Printf("Updated endpoint %s health status to: %s", endpoint.URL, endpoint.Status.String())
}

// GetStrategy returns the load balancing strategy name
func (w *WeightedRoundRobinLoadBalancer) GetStrategy() string {
	return "weighted-round-robin"
}
