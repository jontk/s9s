package auth

import (
	"context"
	"time"
)

// Authenticator defines the interface for all authentication plugins
type Authenticator interface {
	// Core authentication methods
	Authenticate(ctx context.Context, config Config) (*Token, error)
	RefreshToken(ctx context.Context, token *Token) (*Token, error)
	ValidateToken(ctx context.Context, token *Token) error
	RevokeToken(ctx context.Context, token *Token) error

	// Plugin metadata
	GetInfo() AuthenticatorInfo
	GetConfigSchema() ConfigSchema

	// Lifecycle management
	Initialize(ctx context.Context, config Config) error
	Cleanup() error
}

// EndpointDiscoverer defines the interface for endpoint discovery plugins
type EndpointDiscoverer interface {
	DiscoverEndpoints(ctx context.Context, clusterID string) ([]Endpoint, error)
	HealthCheck(ctx context.Context, endpoint *Endpoint) error
	GetLoadBalancer() LoadBalancer

	// Plugin metadata
	GetInfo() DiscovererInfo
	Initialize(ctx context.Context, config DiscoveryConfig) error
	Cleanup() error
}

// LoadBalancer defines the interface for load balancing across endpoints
type LoadBalancer interface {
	// SelectEndpoint selects the best endpoint from the available ones
	SelectEndpoint(endpoints []Endpoint) (*Endpoint, error)

	// UpdateEndpointHealth updates the health status of an endpoint
	UpdateEndpointHealth(endpoint *Endpoint, healthy bool)

	// GetStrategy returns the load balancing strategy
	GetStrategy() string
}

// TokenStore defines the interface for secure token storage
type TokenStore interface {
	// Store saves a token securely
	Store(ctx context.Context, clusterID string, token *Token) error

	// Retrieve gets a token from storage
	Retrieve(ctx context.Context, clusterID string) (*Token, error)

	// Delete removes a token from storage
	Delete(ctx context.Context, clusterID string) error

	// List returns all stored cluster IDs
	List(ctx context.Context) ([]string, error)

	// Clear removes all tokens from storage
	Clear(ctx context.Context) error
}

// SecureStore defines the interface for platform-specific secure storage
type SecureStore interface {
	Store(key string, data []byte) error
	Retrieve(key string) ([]byte, error)
	Delete(key string) error
	List() ([]string, error)
	Initialize() error
	Cleanup() error
}

// Manager manages authentication across multiple clusters
type Manager interface {
	// ConfigureCluster sets up authentication for a cluster
	ConfigureCluster(clusterID string, authType string, config Config) error

	// Authenticate authenticates against a specific cluster
	Authenticate(ctx context.Context, clusterID string) (*Token, error)

	// GetToken retrieves a valid token for a cluster
	GetToken(ctx context.Context, clusterID string) (*Token, error)

	// RefreshToken refreshes an expired token
	RefreshToken(ctx context.Context, clusterID string) (*Token, error)

	// RevokeToken revokes a token
	RevokeToken(ctx context.Context, clusterID string) error

	// ListClusters returns all configured clusters
	ListClusters() []string

	// GetClusterInfo returns information about a cluster's auth configuration
	GetClusterInfo(clusterID string) (*ClusterAuthInfo, error)

	// ValidateConfiguration validates auth configuration
	ValidateConfiguration(authType string, config Config) error
}

type AuthManager = Manager

// ClusterAuthInfo contains information about a cluster's authentication setup
type ClusterAuthInfo struct {
	ClusterID     string
	AuthType      string
	IsConfigured  bool
	HasValidToken bool
	TokenExpiry   *time.Time
	LastAuth      *time.Time
	Endpoints     []Endpoint
}

// EndpointManager manages endpoint discovery and selection
type EndpointManager interface {
	// ConfigureCluster sets up endpoint discovery for a cluster
	ConfigureCluster(clusterID string, discoveryType string, config DiscoveryConfig) error

	// DiscoverEndpoints discovers available endpoints for a cluster
	DiscoverEndpoints(ctx context.Context, clusterID string) ([]Endpoint, error)

	// GetEndpoint selects the best endpoint for a cluster
	GetEndpoint(ctx context.Context, clusterID string) (*Endpoint, error)

	// UpdateEndpointHealth updates the health status of an endpoint
	UpdateEndpointHealth(ctx context.Context, endpoint *Endpoint, healthy bool) error

	// GetAllEndpoints returns all known endpoints for a cluster
	GetAllEndpoints(clusterID string) []Endpoint

	// StartHealthChecking starts background health checking
	StartHealthChecking(ctx context.Context, interval time.Duration)

	// StopHealthChecking stops background health checking
	StopHealthChecking()
}
