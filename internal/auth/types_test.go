package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTokenIsValid(t *testing.T) {
	tests := []struct {
		name     string
		token    *Token
		expected bool
	}{
		{
			name: "valid token with future expiry",
			token: &Token{
				AccessToken: "test-token",
				ExpiresAt:   time.Now().Add(1 * time.Hour),
			},
			expected: true,
		},
		{
			name: "expired token",
			token: &Token{
				AccessToken: "test-token",
				ExpiresAt:   time.Now().Add(-1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "token expiring in 1 second (still valid)",
			token: &Token{
				AccessToken: "test-token",
				ExpiresAt:   time.Now().Add(1 * time.Second),
			},
			expected: true,
		},
		{
			name: "token with empty access token",
			token: &Token{
				AccessToken: "",
				ExpiresAt:   time.Now().Add(1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "token exactly at expiry boundary",
			token: &Token{
				AccessToken: "test-token",
				ExpiresAt:   time.Now(),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.token.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTokenIsExpired(t *testing.T) {
	tests := []struct {
		name     string
		token    *Token
		expected bool
	}{
		{
			name: "future expiry is not expired",
			token: &Token{
				AccessToken: "test-token",
				ExpiresAt:   time.Now().Add(1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "past expiry is expired",
			token: &Token{
				AccessToken: "test-token",
				ExpiresAt:   time.Now().Add(-1 * time.Hour),
			},
			expected: true,
		},
		{
			name: "empty access token is expired",
			token: &Token{
				AccessToken: "",
				ExpiresAt:   time.Now().Add(1 * time.Hour),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.token.IsExpired()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTokenExpiresIn(t *testing.T) {
	tests := []struct {
		name         string
		token        *Token
		expectedMin  time.Duration
		expectedMax  time.Duration
		checkNegative bool
	}{
		{
			name: "expires in 1 hour",
			token: &Token{
				AccessToken: "test-token",
				ExpiresAt:   time.Now().Add(1 * time.Hour),
			},
			expectedMin: 59 * time.Minute,  // Allow some tolerance
			expectedMax: 61 * time.Minute,
		},
		{
			name: "expired 1 hour ago",
			token: &Token{
				AccessToken: "test-token",
				ExpiresAt:   time.Now().Add(-1 * time.Hour),
			},
			checkNegative: true,
		},
		{
			name: "expires in 10 seconds",
			token: &Token{
				AccessToken: "test-token",
				ExpiresAt:   time.Now().Add(10 * time.Second),
			},
			expectedMin: 9 * time.Second,
			expectedMax: 11 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.token.ExpiresIn()
			if tt.checkNegative {
				assert.True(t, result < 0, "Expected negative duration for expired token")
			} else {
				assert.GreaterOrEqual(t, result, tt.expectedMin)
				assert.LessOrEqual(t, result, tt.expectedMax)
			}
		})
	}
}

func TestAuthConfigGet(t *testing.T) {
	config := AuthConfig{
		"string_key": "test-value",
		"int_key":    42,
		"bool_key":   true,
	}

	assert.Equal(t, "test-value", config.Get("string_key"))
	assert.Equal(t, 42, config.Get("int_key"))
	assert.Equal(t, true, config.Get("bool_key"))
	assert.Nil(t, config.Get("nonexistent_key"))
}

func TestAuthConfigGetString(t *testing.T) {
	config := AuthConfig{
		"string_key": "test-value",
		"int_key":    42,
		"bool_key":   true,
	}

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "get existing string value",
			key:      "string_key",
			expected: "test-value",
		},
		{
			name:     "get non-string value returns empty string",
			key:      "int_key",
			expected: "",
		},
		{
			name:     "get nonexistent key returns empty string",
			key:      "nonexistent",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.GetString(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuthConfigGetInt(t *testing.T) {
	config := AuthConfig{
		"string_key": "test-value",
		"int_key":    42,
		"bool_key":   true,
	}

	tests := []struct {
		name     string
		key      string
		expected int
	}{
		{
			name:     "get existing int value",
			key:      "int_key",
			expected: 42,
		},
		{
			name:     "get non-int value returns zero",
			key:      "string_key",
			expected: 0,
		},
		{
			name:     "get nonexistent key returns zero",
			key:      "nonexistent",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.GetInt(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAuthConfigGetBool(t *testing.T) {
	config := AuthConfig{
		"string_key": "test-value",
		"int_key":    42,
		"bool_key":   true,
		"false_key":  false,
	}

	tests := []struct {
		name     string
		key      string
		expected bool
	}{
		{
			name:     "get existing true bool value",
			key:      "bool_key",
			expected: true,
		},
		{
			name:     "get existing false bool value",
			key:      "false_key",
			expected: false,
		},
		{
			name:     "get non-bool value returns false",
			key:      "string_key",
			expected: false,
		},
		{
			name:     "get nonexistent key returns false",
			key:      "nonexistent",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.GetBool(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDiscoveryConfigGet(t *testing.T) {
	config := DiscoveryConfig{
		"string_key": "test-value",
		"int_key":    42,
	}

	assert.Equal(t, "test-value", config.Get("string_key"))
	assert.Equal(t, 42, config.Get("int_key"))
	assert.Nil(t, config.Get("nonexistent_key"))
}

func TestDiscoveryConfigGetString(t *testing.T) {
	config := DiscoveryConfig{
		"string_key": "test-value",
		"int_key":    42,
	}

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{
			name:     "get existing string value",
			key:      "string_key",
			expected: "test-value",
		},
		{
			name:     "get non-string value returns empty string",
			key:      "int_key",
			expected: "",
		},
		{
			name:     "get nonexistent key returns empty string",
			key:      "nonexistent",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.GetString(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDiscoveryConfigGetInt(t *testing.T) {
	config := DiscoveryConfig{
		"string_key": "test-value",
		"int_key":    42,
	}

	tests := []struct {
		name     string
		key      string
		expected int
	}{
		{
			name:     "get existing int value",
			key:      "int_key",
			expected: 42,
		},
		{
			name:     "get non-int value returns zero",
			key:      "string_key",
			expected: 0,
		},
		{
			name:     "get nonexistent key returns zero",
			key:      "nonexistent",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.GetInt(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEndpointStatusString(t *testing.T) {
	tests := []struct {
		name     string
		status   EndpointStatus
		expected string
	}{
		{
			name:     "healthy status",
			status:   EndpointStatusHealthy,
			expected: "healthy",
		},
		{
			name:     "unhealthy status",
			status:   EndpointStatusUnhealthy,
			expected: "unhealthy",
		},
		{
			name:     "maintenance status",
			status:   EndpointStatusMaintenance,
			expected: "maintenance",
		},
		{
			name:     "unknown status",
			status:   EndpointStatusUnknown,
			expected: "unknown",
		},
		{
			name:     "invalid status value defaults to unknown",
			status:   EndpointStatus(999),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.status.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEndpointStatusConstants(t *testing.T) {
	// Verify the iota enumeration is correct
	assert.Equal(t, EndpointStatus(0), EndpointStatusUnknown)
	assert.Equal(t, EndpointStatus(1), EndpointStatusHealthy)
	assert.Equal(t, EndpointStatus(2), EndpointStatusUnhealthy)
	assert.Equal(t, EndpointStatus(3), EndpointStatusMaintenance)
}
