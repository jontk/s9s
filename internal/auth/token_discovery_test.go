package auth

import (
	"testing"
	"time"
)

func TestDefaultTokenDiscoveryConfig(t *testing.T) {
	cfg := DefaultTokenDiscoveryConfig()

	if !cfg.Enabled {
		t.Errorf("expected Enabled=true, got false")
	}

	if cfg.ScontrolPath != "scontrol" {
		t.Errorf("expected ScontrolPath='scontrol', got %q", cfg.ScontrolPath)
	}

	if cfg.Timeout != 10*time.Second {
		t.Errorf("expected Timeout=10s, got %v", cfg.Timeout)
	}

	if cfg.TokenLifespan != 3600 {
		t.Errorf("expected TokenLifespan=3600, got %d", cfg.TokenLifespan)
	}
}

func TestNewTokenDiscovery(t *testing.T) {
	td := NewTokenDiscovery()

	if td == nil {
		t.Fatalf("expected non-nil TokenDiscovery")
	}

	if !td.IsEnabled() {
		t.Errorf("expected token discovery to be enabled by default")
	}
}

func TestNewTokenDiscoveryWithConfig(t *testing.T) {
	cfg := TokenDiscoveryConfig{
		Enabled:       false,
		ScontrolPath:  "/usr/bin/scontrol",
		Timeout:       30 * time.Second,
		TokenLifespan: 7200,
	}

	td := NewTokenDiscoveryWithConfig(cfg)

	if td.IsEnabled() {
		t.Errorf("expected token discovery to be disabled")
	}

	if td.scontrolPath != "/usr/bin/scontrol" {
		t.Errorf("expected scontrolPath='/usr/bin/scontrol', got %q", td.scontrolPath)
	}

	if td.timeout != 30*time.Second {
		t.Errorf("expected timeout=30s, got %v", td.timeout)
	}

	if td.tokenLifespan != 7200 {
		t.Errorf("expected tokenLifespan=7200, got %d", td.tokenLifespan)
	}
}

func TestSetEnabled(t *testing.T) {
	td := NewTokenDiscovery()

	// Default should be enabled
	if !td.IsEnabled() {
		t.Errorf("expected enabled=true by default")
	}

	// Disable
	td.SetEnabled(false)
	if td.IsEnabled() {
		t.Errorf("expected enabled=false after SetEnabled(false)")
	}

	// Re-enable
	td.SetEnabled(true)
	if !td.IsEnabled() {
		t.Errorf("expected enabled=true after SetEnabled(true)")
	}
}

func TestParseTokenOutput(t *testing.T) {
	td := &TokenDiscovery{}

	tests := []struct {
		name        string
		input       string
		expectToken string
		expectError bool
	}{
		{
			name:        "standard format",
			input:       "SLURM_JWT=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test",
			expectToken: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test",
			expectError: false,
		},
		{
			name:        "with whitespace",
			input:       "  SLURM_JWT=eyJtoken123  \n",
			expectToken: "eyJtoken123",
			expectError: false,
		},
		{
			name: "multiple lines",
			input: `Some output
SLURM_JWT=eyJtoken456
More output`,
			expectToken: "eyJtoken456",
			expectError: false,
		},
		{
			name:        "empty output",
			input:       "",
			expectToken: "",
			expectError: true,
		},
		{
			name:        "no token",
			input:       "Some random output without token",
			expectToken: "",
			expectError: true,
		},
		{
			name:        "empty token value",
			input:       "SLURM_JWT=",
			expectToken: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := td.parseTokenOutput(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if token.Token != tt.expectToken {
				t.Errorf("expected token %q, got %q", tt.expectToken, token.Token)
			}
		})
	}
}

func TestClearCache(t *testing.T) {
	td := NewTokenDiscovery()

	// Manually cache a token
	token := &DiscoveredToken{
		Token:     "test-token",
		Username:  "testuser",
		ExpiresAt: time.Now().Add(time.Hour),
		Source:    "test",
	}
	td.cacheToken(token)

	// Verify it's cached
	if cached := td.getCachedToken(); cached == nil {
		t.Errorf("expected cached token, got nil")
	}

	// Clear cache
	td.ClearCache()

	// Verify it's cleared
	if cached := td.getCachedToken(); cached != nil {
		t.Errorf("expected nil after ClearCache, got %+v", cached)
	}
}

func TestTokenExpiry(t *testing.T) {
	td := NewTokenDiscovery()

	// Test with no cached token
	if !td.IsTokenExpired() {
		t.Errorf("expected IsTokenExpired=true when no token cached")
	}

	// Cache an expired token
	expiredToken := &DiscoveredToken{
		Token:     "expired-token",
		Username:  "testuser",
		ExpiresAt: time.Now().Add(-time.Hour),
		Source:    "test",
	}
	td.cacheToken(expiredToken)

	if !td.IsTokenExpired() {
		t.Errorf("expected IsTokenExpired=true for expired token")
	}

	// Cache a valid token
	validToken := &DiscoveredToken{
		Token:     "valid-token",
		Username:  "testuser",
		ExpiresAt: time.Now().Add(time.Hour),
		Source:    "test",
	}
	td.cacheToken(validToken)

	if td.IsTokenExpired() {
		t.Errorf("expected IsTokenExpired=false for valid token")
	}
}

func TestGetTokenExpiresIn(t *testing.T) {
	td := NewTokenDiscovery()

	// Test with no cached token
	if td.GetTokenExpiresIn() != 0 {
		t.Errorf("expected ExpiresIn=0 when no token cached")
	}

	// Cache a token expiring in 1 hour
	token := &DiscoveredToken{
		Token:     "test-token",
		Username:  "testuser",
		ExpiresAt: time.Now().Add(time.Hour),
		Source:    "test",
	}
	td.cacheToken(token)

	expiresIn := td.GetTokenExpiresIn()
	if expiresIn < 59*time.Minute || expiresIn > time.Hour {
		t.Errorf("expected ExpiresIn ~1h, got %v", expiresIn)
	}
}

func TestDiscoveredTokenToAuthToken(t *testing.T) {
	discoveredToken := &DiscoveredToken{
		Token:     "test-jwt-token",
		Username:  "testuser",
		ExpiresAt: time.Now().Add(time.Hour),
		Source:    "scontrol",
		Metadata: map[string]string{
			"generated_by": "scontrol token",
		},
	}

	authToken := discoveredToken.ToAuthToken("test-cluster")

	if authToken.AccessToken != "test-jwt-token" {
		t.Errorf("expected AccessToken='test-jwt-token', got %q", authToken.AccessToken)
	}

	if authToken.TokenType != "Bearer" {
		t.Errorf("expected TokenType='Bearer', got %q", authToken.TokenType)
	}

	if authToken.ClusterID != "test-cluster" {
		t.Errorf("expected ClusterID='test-cluster', got %q", authToken.ClusterID)
	}

	if authToken.Metadata["username"] != "testuser" {
		t.Errorf("expected metadata username='testuser', got %q", authToken.Metadata["username"])
	}

	if authToken.Metadata["source"] != "scontrol" {
		t.Errorf("expected metadata source='scontrol', got %q", authToken.Metadata["source"])
	}

	if authToken.Metadata["auth_method"] != "auto-discovery" {
		t.Errorf("expected metadata auth_method='auto-discovery', got %q", authToken.Metadata["auth_method"])
	}
}

func TestCacheExpiryBuffer(t *testing.T) {
	td := NewTokenDiscovery()

	// Cache a token that will expire in 4 minutes (less than the 5-minute buffer)
	token := &DiscoveredToken{
		Token:     "soon-to-expire-token",
		Username:  "testuser",
		ExpiresAt: time.Now().Add(4 * time.Minute),
		Source:    "test",
	}
	td.cacheToken(token)

	// Should return nil because of the 5-minute buffer
	if cached := td.getCachedToken(); cached != nil {
		t.Errorf("expected nil due to expiry buffer, got %+v", cached)
	}

	// Cache a token that will expire in 10 minutes (more than the 5-minute buffer)
	token2 := &DiscoveredToken{
		Token:     "valid-token",
		Username:  "testuser",
		ExpiresAt: time.Now().Add(10 * time.Minute),
		Source:    "test",
	}
	td.cacheToken(token2)

	// Should return the token
	if cached := td.getCachedToken(); cached == nil {
		t.Errorf("expected cached token, got nil")
	}
}
