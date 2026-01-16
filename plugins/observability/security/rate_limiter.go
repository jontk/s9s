package security

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	mu           sync.RWMutex
	clients      map[string]*ClientLimiter
	globalLimit  int           // Global requests per minute
	cleanupTimer *time.Ticker
	ctx          *interface{} // For future cancellation support
}

// ClientLimiter tracks rate limiting for a specific client
type ClientLimiter struct {
	tokens       int       // Current token count
	lastRefill   time.Time // Last time tokens were refilled
	maxTokens    int       // Maximum tokens (requests per minute)
	refillRate   int       // Tokens added per refill interval
	refillPeriod time.Duration
}

// RateLimitConfig contains rate limiting configuration
type RateLimitConfig struct {
	// Requests per minute allowed per client
	RequestsPerMinute int
	
	// Enable global rate limiting (across all clients)
	EnableGlobalLimit bool
	
	// Global requests per minute limit
	GlobalRequestsPerMinute int
	
	// Burst size (how many requests can be made immediately)
	BurstSize int
	
	// Cleanup interval for removing inactive clients
	CleanupInterval time.Duration
	
	// Client identification method: "ip", "token", "user-agent"
	ClientIDMethod string
	
	// Custom headers to use for client identification
	ClientIDHeader string
}

// DefaultRateLimitConfig returns default rate limiting configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerMinute:       100,
		EnableGlobalLimit:       true,
		GlobalRequestsPerMinute: 1000,
		BurstSize:               10,
		CleanupInterval:         10 * time.Minute,
		ClientIDMethod:          "ip",
		ClientIDHeader:          "",
	}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	if config.RequestsPerMinute <= 0 {
		config.RequestsPerMinute = 100
	}
	if config.BurstSize <= 0 {
		config.BurstSize = config.RequestsPerMinute / 6 // Allow 1/6 of per-minute limit as burst
	}
	if config.CleanupInterval <= 0 {
		config.CleanupInterval = 10 * time.Minute
	}
	
	rl := &RateLimiter{
		clients:      make(map[string]*ClientLimiter),
		globalLimit:  config.GlobalRequestsPerMinute,
		cleanupTimer: time.NewTicker(config.CleanupInterval),
	}
	
	// Start cleanup routine
	go rl.cleanupRoutine(config)
	
	return rl
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(clientID string, config RateLimitConfig) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	
	// Get or create client limiter
	client, exists := rl.clients[clientID]
	if !exists {
		client = &ClientLimiter{
			tokens:       config.BurstSize,
			lastRefill:   now,
			maxTokens:    config.RequestsPerMinute,
			refillRate:   config.RequestsPerMinute,
			refillPeriod: time.Minute,
		}
		rl.clients[clientID] = client
	}
	
	// Refill tokens based on elapsed time
	client.refillTokens(now)
	
	// Check if client has tokens available
	if client.tokens <= 0 {
		return false
	}
	
	// Check global limit if enabled
	if config.EnableGlobalLimit {
		totalRequests := rl.countRecentRequests(now)
		if totalRequests >= rl.globalLimit {
			return false
		}
	}
	
	// Consume a token
	client.tokens--
	return true
}

// refillTokens refills tokens for a client based on elapsed time
func (cl *ClientLimiter) refillTokens(now time.Time) {
	elapsed := now.Sub(cl.lastRefill)
	if elapsed < time.Second {
		return // Don't refill more than once per second
	}
	
	// Calculate tokens to add based on elapsed time
	tokensToAdd := int(elapsed.Minutes() * float64(cl.refillRate))
	if tokensToAdd > 0 {
		cl.tokens += tokensToAdd
		if cl.tokens > cl.maxTokens {
			cl.tokens = cl.maxTokens
		}
		cl.lastRefill = now
	}
}

// countRecentRequests counts requests from all clients in the last minute (for global limiting)
func (rl *RateLimiter) countRecentRequests(now time.Time) int {
	count := 0
	
	for _, client := range rl.clients {
		// Simplified approach: count tokens consumed from initial burst
		// In production, you'd maintain a sliding window of actual requests
		if client.maxTokens > 0 {
			consumed := client.maxTokens - client.tokens
			if consumed > 0 {
				count += consumed
			}
		}
	}
	
	return count
}

// cleanupRoutine removes inactive clients periodically
func (rl *RateLimiter) cleanupRoutine(config RateLimitConfig) {
	for range rl.cleanupTimer.C {
		rl.cleanup(config.CleanupInterval)
	}
}

// cleanup removes clients that haven't been active for the specified duration
func (rl *RateLimiter) cleanup(inactiveDuration time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	cutoff := now.Add(-inactiveDuration)
	
	for clientID, client := range rl.clients {
		if client.lastRefill.Before(cutoff) {
			delete(rl.clients, clientID)
		}
	}
}

// Stop stops the rate limiter and cleanup routine
func (rl *RateLimiter) Stop() {
	if rl.cleanupTimer != nil {
		rl.cleanupTimer.Stop()
	}
}

// GetStats returns rate limiting statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	
	totalClients := len(rl.clients)
	activeClients := 0
	totalTokens := 0
	
	now := time.Now()
	cutoff := now.Add(-5 * time.Minute) // Consider clients active if accessed in last 5 minutes
	
	for _, client := range rl.clients {
		if client.lastRefill.After(cutoff) {
			activeClients++
		}
		totalTokens += client.tokens
	}
	
	return map[string]interface{}{
		"total_clients":  totalClients,
		"active_clients": activeClients,
		"total_tokens":   totalTokens,
		"global_limit":   rl.globalLimit,
	}
}

// RateLimitMiddleware creates HTTP middleware for rate limiting
func RateLimitMiddleware(rateLimiter *RateLimiter, config RateLimitConfig) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			// Extract client identifier
			clientID := extractClientID(r, config)
			
			// Check rate limit
			if !rateLimiter.Allow(clientID, config) {
				writeRateLimitError(w)
				return
			}
			
			// Add rate limit headers
			addRateLimitHeaders(w, config)
			
			// Continue to next handler
			next(w, r)
		}
	}
}

// extractClientID extracts client identifier from request based on configuration
func extractClientID(r *http.Request, config RateLimitConfig) string {
	switch config.ClientIDMethod {
	case "ip":
		// Extract IP from X-Forwarded-For or RemoteAddr
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// Use first IP in X-Forwarded-For chain
			if ips := strings.Split(xff, ","); len(ips) > 0 {
				return strings.TrimSpace(ips[0])
			}
		}
		// Fallback to RemoteAddr (remove port)
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			return host
		}
		return r.RemoteAddr
		
	case "token":
		// Extract from Authorization header
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
		return "anonymous"
		
	case "user-agent":
		userAgent := r.Header.Get("User-Agent")
		if userAgent == "" {
			return "unknown-agent"
		}
		return userAgent
		
	case "header":
		if config.ClientIDHeader != "" {
			headerValue := r.Header.Get(config.ClientIDHeader)
			if headerValue != "" {
				return headerValue
			}
		}
		return "unknown-header"
		
	default:
		// Default to IP-based identification
		return extractClientID(r, RateLimitConfig{ClientIDMethod: "ip"})
	}
}

// writeRateLimitError writes a rate limit exceeded response
func writeRateLimitError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	
	response := map[string]interface{}{
		"status": "error",
		"error":  "Rate limit exceeded",
		"code":   "RATE_LIMIT_EXCEEDED",
	}
	
	json.NewEncoder(w).Encode(response)
}

// addRateLimitHeaders adds rate limiting information to response headers
func addRateLimitHeaders(w http.ResponseWriter, config RateLimitConfig) {
	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", config.RequestsPerMinute))
	w.Header().Set("X-RateLimit-Window", "60") // 60 seconds
}