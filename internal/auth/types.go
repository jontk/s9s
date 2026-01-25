package auth

import (
	"time"
)

// Token represents an authentication token
type Token struct {
	AccessToken  string            `json:"access_token"`
	RefreshToken string            `json:"refresh_token,omitempty"`
	TokenType    string            `json:"token_type"`
	ExpiresAt    time.Time         `json:"expires_at"`
	Scopes       []string          `json:"scopes,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	ClusterID    string            `json:"cluster_id"`
}

// IsValid checks if the token is still valid
func (t *Token) IsValid() bool {
	if t.AccessToken == "" {
		return false
	}
	return time.Now().Before(t.ExpiresAt)
}

// IsExpired checks if the token is expired
func (t *Token) IsExpired() bool {
	return !t.IsValid()
}

// ExpiresIn returns the duration until the token expires
func (t *Token) ExpiresIn() time.Duration {
	return time.Until(t.ExpiresAt)
}

// Config holds configuration for authenticators
type Config map[string]interface{}

// Get retrieves a configuration value by key
func (c Config) Get(key string) interface{} {
	return c[key]
}

// GetString retrieves a string configuration value
func (c Config) GetString(key string) string {
	if val, ok := c[key].(string); ok {
		return val
	}
	return ""
}

// GetInt retrieves an integer configuration value
func (c Config) GetInt(key string) int {
	if val, ok := c[key].(int); ok {
		return val
	}
	return 0
}

// GetBool retrieves a boolean configuration value
func (c Config) GetBool(key string) bool {
	if val, ok := c[key].(bool); ok {
		return val
	}
	return false
}

//nolint:revive // type alias for backward compatibility
type AuthConfig = Config

// AuthenticatorInfo contains metadata about an authenticator
type AuthenticatorInfo struct {
	Name        string
	Version     string
	Description string
	Author      string
	Supported   []string // List of supported authentication methods
}

// ConfigSchema defines the configuration schema for an authenticator
type ConfigSchema struct {
	Properties map[string]PropertySchema `json:"properties"`
	Required   []string                  `json:"required"`
}

// PropertySchema defines a single property in the configuration schema
type PropertySchema struct {
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Sensitive   bool        `json:"sensitive"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
}

// Endpoint represents a slurmrestd API endpoint
type Endpoint struct {
	URL       string            `json:"url"`
	ClusterID string            `json:"cluster_id"`
	Weight    int               `json:"weight"`
	Metadata  map[string]string `json:"metadata"`
	Status    EndpointStatus    `json:"status"`
	LastCheck time.Time         `json:"last_check"`
}

// EndpointStatus represents the health status of an endpoint
type EndpointStatus int

const (
	// EndpointStatusUnknown is the unknown endpoint status.
	EndpointStatusUnknown EndpointStatus = iota
	// EndpointStatusHealthy is the healthy endpoint status.
	EndpointStatusHealthy
	// EndpointStatusUnhealthy is the unhealthy endpoint status.
	EndpointStatusUnhealthy
	// EndpointStatusMaintenance is the maintenance endpoint status.
	EndpointStatusMaintenance
)

// String returns the string representation of the endpoint status
func (s EndpointStatus) String() string {
	switch s {
	case EndpointStatusHealthy:
		return "healthy"
	case EndpointStatusUnhealthy:
		return "unhealthy"
	case EndpointStatusMaintenance:
		return "maintenance"
	default:
		return "unknown"
	}
}

// DiscoveryConfig holds configuration for endpoint discovery
type DiscoveryConfig map[string]interface{}

// Get retrieves a configuration value by key
func (c DiscoveryConfig) Get(key string) interface{} {
	return c[key]
}

// GetString retrieves a string configuration value
func (c DiscoveryConfig) GetString(key string) string {
	if val, ok := c[key].(string); ok {
		return val
	}
	return ""
}

// GetInt retrieves an integer configuration value
func (c DiscoveryConfig) GetInt(key string) int {
	if val, ok := c[key].(int); ok {
		return val
	}
	return 0
}

// DiscovererInfo contains metadata about an endpoint discoverer
type DiscovererInfo struct {
	Name        string
	Version     string
	Description string
	Author      string
	Supported   []string // List of supported discovery methods
}
