package security

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	config := DefaultRateLimitConfig()
	rl := NewRateLimiter(config)

	if rl == nil {
		t.Fatal("NewRateLimiter returned nil")
	}

	if rl.clients == nil {
		t.Error("Expected clients map to be initialized")
	}

	if rl.globalLimit != config.GlobalRequestsPerMinute {
		t.Errorf("Expected global limit %d, got %d", config.GlobalRequestsPerMinute, rl.globalLimit)
	}

	if rl.cleanupTimer == nil {
		t.Error("Expected cleanup timer to be initialized")
	}

	// Clean up
	rl.Stop()
}

func TestRateLimiterAllow(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerMinute:       10,
		EnableGlobalLimit:       false,
		GlobalRequestsPerMinute: 100,
		BurstSize:               5,
		CleanupInterval:         time.Minute,
		ClientIDMethod:          "ip",
	}

	rl := NewRateLimiter(config)
	defer rl.Stop()

	clientID := "127.0.0.1"

	// Should allow initial requests up to burst size
	for i := 0; i < config.BurstSize; i++ {
		if !rl.Allow(clientID, config) {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	// Should deny next request (burst exhausted)
	if rl.Allow(clientID, config) {
		t.Error("Request beyond burst size should be denied")
	}
}

func TestRateLimiterTokenRefill(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerMinute:       60, // 1 per second
		EnableGlobalLimit:       false,
		BurstSize:               1,
		CleanupInterval:         time.Minute,
		ClientIDMethod:          "ip",
	}

	rl := NewRateLimiter(config)
	defer rl.Stop()

	clientID := "127.0.0.1"

	// Use the burst token
	if !rl.Allow(clientID, config) {
		t.Error("First request should be allowed")
	}

	// Should be denied immediately after
	if rl.Allow(clientID, config) {
		t.Error("Second request should be denied")
	}

	// Wait a bit for tokens to refill (simplified test)
	time.Sleep(2 * time.Second)

	// Should be allowed again after refill
	if !rl.Allow(clientID, config) {
		t.Error("Request after refill should be allowed")
	}
}

func TestRateLimiterGlobalLimit(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerMinute:       100,
		EnableGlobalLimit:       true,
		GlobalRequestsPerMinute: 3, // Very low global limit for testing
		BurstSize:               2,  // Small burst size
		CleanupInterval:         time.Minute,
		ClientIDMethod:          "ip",
	}

	rl := NewRateLimiter(config)
	defer rl.Stop()

	// Use multiple clients, each making 1 request to stay under individual limits
	// but exceed global limit
	allowedCount := 0
	for i := 0; i < 5; i++ {
		clientID := fmt.Sprintf("127.0.0.%d", i+1)
		if rl.Allow(clientID, config) {
			allowedCount++
		}
	}

	// Should allow exactly the global limit number of requests
	if allowedCount > config.GlobalRequestsPerMinute {
		t.Errorf("Expected at most %d requests to be allowed, got %d", config.GlobalRequestsPerMinute, allowedCount)
	}
}

func TestRateLimiterCleanup(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerMinute:       100,
		BurstSize:               10,
		CleanupInterval:         100 * time.Millisecond, // Fast cleanup for testing
		ClientIDMethod:          "ip",
	}

	rl := NewRateLimiter(config)
	defer rl.Stop()

	// Make request to create client entry
	clientID := "127.0.0.1"
	rl.Allow(clientID, config)

	// Verify client exists
	rl.mu.RLock()
	initialCount := len(rl.clients)
	rl.mu.RUnlock()
	
	if initialCount == 0 {
		t.Error("Client should exist after request")
	}

	// Manually trigger cleanup with very short duration to remove all clients
	rl.cleanup(time.Nanosecond)

	// Client should be removed now
	rl.mu.RLock()
	finalCount := len(rl.clients)
	rl.mu.RUnlock()
	
	if finalCount != 0 {
		t.Errorf("All clients should be removed after cleanup, got %d clients", finalCount)
	}
}

func TestRateLimiterGetStats(t *testing.T) {
	config := DefaultRateLimitConfig()
	rl := NewRateLimiter(config)
	defer rl.Stop()

	// Make some requests
	rl.Allow("127.0.0.1", config)
	rl.Allow("127.0.0.2", config)

	stats := rl.GetStats()

	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	totalClients, ok := stats["total_clients"].(int)
	if !ok || totalClients != 2 {
		t.Errorf("Expected 2 total clients, got %v", stats["total_clients"])
	}

	activeClients, ok := stats["active_clients"].(int)
	if !ok || activeClients != 2 {
		t.Errorf("Expected 2 active clients, got %v", stats["active_clients"])
	}

	globalLimit, ok := stats["global_limit"].(int)
	if !ok || globalLimit != config.GlobalRequestsPerMinute {
		t.Errorf("Expected global limit %d, got %v", config.GlobalRequestsPerMinute, stats["global_limit"])
	}
}

func TestExtractClientID(t *testing.T) {
	tests := []struct {
		name           string
		config         RateLimitConfig
		setupRequest   func(*http.Request)
		expectedPrefix string
	}{
		{
			name: "IP extraction from RemoteAddr",
			config: RateLimitConfig{
				ClientIDMethod: "ip",
			},
			setupRequest: func(r *http.Request) {
				r.RemoteAddr = "192.168.1.1:12345"
			},
			expectedPrefix: "192.168.1.1",
		},
		{
			name: "IP extraction from X-Forwarded-For",
			config: RateLimitConfig{
				ClientIDMethod: "ip",
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "10.0.0.1, 192.168.1.1")
				r.RemoteAddr = "192.168.1.1:12345"
			},
			expectedPrefix: "10.0.0.1",
		},
		{
			name: "Token extraction from Authorization header",
			config: RateLimitConfig{
				ClientIDMethod: "token",
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer test-token-123")
			},
			expectedPrefix: "test-token-123",
		},
		{
			name: "User-Agent extraction",
			config: RateLimitConfig{
				ClientIDMethod: "user-agent",
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("User-Agent", "TestClient/1.0")
			},
			expectedPrefix: "TestClient/1.0",
		},
		{
			name: "Custom header extraction",
			config: RateLimitConfig{
				ClientIDMethod: "header",
				ClientIDHeader: "X-Client-ID",
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-Client-ID", "custom-client-123")
			},
			expectedPrefix: "custom-client-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			tt.setupRequest(req)

			clientID := extractClientID(req, tt.config)
			if clientID != tt.expectedPrefix {
				t.Errorf("Expected client ID '%s', got '%s'", tt.expectedPrefix, clientID)
			}
		})
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerMinute: 2,
		BurstSize:         1,
		ClientIDMethod:    "ip",
	}

	rl := NewRateLimiter(config)
	defer rl.Stop()

	// Create a test handler
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}

	// Wrap with rate limiting middleware
	middleware := RateLimitMiddleware(rl, config)
	wrappedHandler := middleware(handler)

	// Test first request - should succeed
	req1 := httptest.NewRequest("GET", "/test", nil)
	req1.RemoteAddr = "127.0.0.1:12345"
	w1 := httptest.NewRecorder()
	wrappedHandler(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("First request should succeed, got status %d", w1.Code)
	}

	// Test second request - should be rate limited
	req2 := httptest.NewRequest("GET", "/test", nil)
	req2.RemoteAddr = "127.0.0.1:12345"
	w2 := httptest.NewRecorder()
	wrappedHandler(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("Second request should be rate limited, got status %d", w2.Code)
	}

	// Verify rate limit headers are present
	if w1.Header().Get("X-RateLimit-Limit") == "" {
		t.Error("Expected X-RateLimit-Limit header in successful response")
	}

	if w1.Header().Get("X-RateLimit-Window") != "60" {
		t.Error("Expected X-RateLimit-Window header to be '60'")
	}
}

func TestRateLimitErrorResponse(t *testing.T) {
	w := httptest.NewRecorder()
	writeRateLimitError(w)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected status %d, got %d", http.StatusTooManyRequests, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	// Check response body contains expected fields
	body := w.Body.String()
	if !strings.Contains(body, "Rate limit exceeded") {
		t.Error("Response should contain rate limit error message")
	}
}

func TestDefaultRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()

	if config.RequestsPerMinute <= 0 {
		t.Error("RequestsPerMinute should be positive")
	}

	if config.BurstSize <= 0 {
		t.Error("BurstSize should be positive")
	}

	if !config.EnableGlobalLimit {
		t.Error("EnableGlobalLimit should be true by default")
	}

	if config.GlobalRequestsPerMinute <= 0 {
		t.Error("GlobalRequestsPerMinute should be positive")
	}

	if config.CleanupInterval <= 0 {
		t.Error("CleanupInterval should be positive")
	}

	if config.ClientIDMethod != "ip" {
		t.Errorf("Expected default ClientIDMethod 'ip', got '%s'", config.ClientIDMethod)
	}
}

func TestRateLimiterConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config RateLimitConfig
		valid  bool
	}{
		{
			name:   "valid default config",
			config: DefaultRateLimitConfig(),
			valid:  true,
		},
		{
			name: "zero requests per minute gets default",
			config: RateLimitConfig{
				RequestsPerMinute: 0,
				BurstSize:         10,
			},
			valid: true,
		},
		{
			name: "zero burst size gets calculated default",
			config: RateLimitConfig{
				RequestsPerMinute: 60,
				BurstSize:         0,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter(tt.config)
			defer rl.Stop()

			if rl == nil {
				if tt.valid {
					t.Error("Expected valid config to create rate limiter")
				}
			} else {
				if !tt.valid {
					t.Error("Expected invalid config to fail")
				}
			}
		})
	}
}

func TestClientLimiterRefillTokens(t *testing.T) {
	client := &ClientLimiter{
		tokens:       0,
		lastRefill:   time.Now().Add(-2 * time.Minute), // 2 minutes ago
		maxTokens:    100,
		refillRate:   100, // 100 tokens per minute
		refillPeriod: time.Minute,
	}

	now := time.Now()
	client.refillTokens(now)

	// Should have refilled tokens based on elapsed time (2 minutes = 200 tokens, capped at maxTokens)
	if client.tokens != client.maxTokens {
		t.Errorf("Expected tokens to be capped at maxTokens (%d), got %d", client.maxTokens, client.tokens)
	}

	if !client.lastRefill.Equal(now) {
		t.Error("lastRefill should be updated to current time")
	}
}

func TestClientLimiterRefillTokensRecentRefill(t *testing.T) {
	now := time.Now()
	client := &ClientLimiter{
		tokens:       5,
		lastRefill:   now.Add(-500 * time.Millisecond), // Less than 1 second ago
		maxTokens:    100,
		refillRate:   100,
		refillPeriod: time.Minute,
	}

	originalTokens := client.tokens
	originalRefill := client.lastRefill

	client.refillTokens(now)

	// Should not refill tokens if less than 1 second has passed
	if client.tokens != originalTokens {
		t.Errorf("Tokens should not be refilled, expected %d, got %d", originalTokens, client.tokens)
	}

	if !client.lastRefill.Equal(originalRefill) {
		t.Error("lastRefill should not be updated for recent refills")
	}
}

func TestRateLimiterMultipleClients(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerMinute: 10,
		BurstSize:         2,
		ClientIDMethod:    "ip",
	}

	rl := NewRateLimiter(config)
	defer rl.Stop()

	client1 := "127.0.0.1"
	client2 := "127.0.0.2"

	// Both clients should be able to make burst requests
	for i := 0; i < config.BurstSize; i++ {
		if !rl.Allow(client1, config) {
			t.Errorf("Client1 request %d should be allowed", i+1)
		}
		if !rl.Allow(client2, config) {
			t.Errorf("Client2 request %d should be allowed", i+1)
		}
	}

	// Both should be rate limited now
	if rl.Allow(client1, config) {
		t.Error("Client1 should be rate limited")
	}
	if rl.Allow(client2, config) {
		t.Error("Client2 should be rate limited")
	}
}