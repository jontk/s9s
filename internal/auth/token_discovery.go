package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jontk/s9s/internal/debug"
	"github.com/jontk/s9s/internal/security"
)

// TokenDiscovery provides automatic discovery and generation of SLURM JWT tokens
type TokenDiscovery struct {
	enabled       bool
	scontrolPath  string
	timeout       time.Duration
	tokenLifespan int // Token lifespan in seconds

	// Cached token
	mu          sync.RWMutex
	cachedToken *DiscoveredToken
}

// DiscoveredToken represents a discovered/generated SLURM JWT token
type DiscoveredToken struct {
	Token     string            // The JWT token value
	Username  string            // Username the token was generated for
	ExpiresAt time.Time         // Token expiration time
	Source    string            // How the token was obtained (scontrol, env, etc.)
	Metadata  map[string]string // Additional metadata
}

// TokenDiscoveryConfig holds configuration for token discovery
type TokenDiscoveryConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	ScontrolPath  string        `mapstructure:"scontrolPath"`
	Timeout       time.Duration `mapstructure:"timeout"`
	TokenLifespan int           `mapstructure:"tokenLifespan"` // in seconds
}

// DefaultTokenDiscoveryConfig returns the default configuration
func DefaultTokenDiscoveryConfig() TokenDiscoveryConfig {
	return TokenDiscoveryConfig{
		Enabled:       true,
		ScontrolPath:  "scontrol",
		Timeout:       10 * time.Second,
		TokenLifespan: 3600, // 1 hour
	}
}

// NewTokenDiscovery creates a new TokenDiscovery instance with default configuration
func NewTokenDiscovery() *TokenDiscovery {
	cfg := DefaultTokenDiscoveryConfig()
	return NewTokenDiscoveryWithConfig(cfg)
}

// NewTokenDiscoveryWithConfig creates a new TokenDiscovery instance with custom configuration
func NewTokenDiscoveryWithConfig(cfg TokenDiscoveryConfig) *TokenDiscovery {
	// Validate and resolve scontrol command path
	// If validation fails, fall back to the original path (will fail later with clear error)
	validatedPath := cfg.ScontrolPath
	if validated, err := security.ValidateAndResolveCommand(cfg.ScontrolPath, "slurm"); err == nil {
		validatedPath = validated
	}

	return &TokenDiscovery{
		enabled:       cfg.Enabled,
		scontrolPath:  validatedPath,
		timeout:       cfg.Timeout,
		tokenLifespan: cfg.TokenLifespan,
	}
}

// DiscoverToken discovers or generates a SLURM JWT token
// It first checks for an existing SLURM_JWT environment variable,
// then tries to generate a new token using scontrol token
func (td *TokenDiscovery) DiscoverToken(ctx context.Context, clusterName string) (*DiscoveredToken, error) {
	if !td.enabled {
		return nil, fmt.Errorf("token discovery is disabled")
	}

	debug.Logger.Printf("Starting token discovery for cluster: %s", clusterName)

	// Check if we have a valid cached token
	if token := td.getCachedToken(); token != nil {
		debug.Logger.Printf("Returning cached token for user: %s", token.Username)
		return token, nil
	}

	// Check SLURM_JWT environment variable first
	if envToken := os.Getenv("SLURM_JWT"); envToken != "" {
		debug.Logger.Printf("Found SLURM_JWT environment variable")
		token := &DiscoveredToken{
			Token:     envToken,
			Username:  os.Getenv("USER"),
			ExpiresAt: time.Now().Add(time.Hour), // Assume 1 hour if unknown
			Source:    "environment",
			Metadata: map[string]string{
				"env_var": "SLURM_JWT",
			},
		}
		td.cacheToken(token)
		return token, nil
	}

	// Try to generate a new token using scontrol
	token, err := td.generateToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	td.cacheToken(token)
	return token, nil
}

// generateToken generates a new SLURM JWT token using scontrol token
func (td *TokenDiscovery) generateToken(ctx context.Context) (*DiscoveredToken, error) {
	username := os.Getenv("USER")
	if username == "" {
		return nil, fmt.Errorf("USER environment variable not set")
	}

	debug.Logger.Printf("Generating SLURM token for user: %s with lifespan: %d seconds", username, td.tokenLifespan)

	// Create context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, td.timeout)
	defer cancel()

	// Build the scontrol token command
	// nolint:gosec // G204: Command path is validated during initialization via security.ValidateAndResolveCommand
	cmd := exec.CommandContext(ctxWithTimeout, td.scontrolPath, "token",
		fmt.Sprintf("username=%s", username),
		fmt.Sprintf("lifespan=%d", td.tokenLifespan))

	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("scontrol token failed: %w (stderr: %s)", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("scontrol token failed: %w", err)
	}

	// Parse the token from output
	token, err := td.parseTokenOutput(string(output))
	if err != nil {
		return nil, err
	}

	token.Username = username
	token.ExpiresAt = time.Now().Add(time.Duration(td.tokenLifespan) * time.Second)
	token.Source = "scontrol"
	token.Metadata = map[string]string{
		"generated_by": "scontrol token",
		"lifespan":     fmt.Sprintf("%d", td.tokenLifespan),
	}

	debug.Logger.Printf("Successfully generated SLURM token, expires at: %v", token.ExpiresAt)
	return token, nil
}

// parseTokenOutput parses the output of scontrol token command
// Expected format: SLURM_JWT=eyJ...
func (td *TokenDiscovery) parseTokenOutput(output string) (*DiscoveredToken, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Primary format: SLURM_JWT=token
		if strings.HasPrefix(line, "SLURM_JWT=") {
			token := strings.TrimPrefix(line, "SLURM_JWT=")
			if token != "" {
				return &DiscoveredToken{
					Token: token,
				}, nil
			}
		}
	}

	// Try alternative regex patterns
	patterns := []*regexp.Regexp{
		// Pattern: SLURM_JWT=token
		regexp.MustCompile(`SLURM_JWT=([A-Za-z0-9\-._~+/]+=*)`),
		// Pattern: token: value
		regexp.MustCompile(`(?i)token[:\s]*([A-Za-z0-9\-._~+/]+=*)`),
	}

	for _, pattern := range patterns {
		matches := pattern.FindStringSubmatch(output)
		if len(matches) > 1 && matches[1] != "" {
			return &DiscoveredToken{
				Token: matches[1],
			}, nil
		}
	}

	return nil, fmt.Errorf("could not find token in scontrol output: %s", output)
}

// getCachedToken returns the cached token if it's still valid
func (td *TokenDiscovery) getCachedToken() *DiscoveredToken {
	td.mu.RLock()
	defer td.mu.RUnlock()

	if td.cachedToken == nil {
		return nil
	}

	// Check if token is expired (with 5 minute buffer)
	if time.Now().Add(5 * time.Minute).After(td.cachedToken.ExpiresAt) {
		return nil
	}

	return td.cachedToken
}

// cacheToken caches the token for reuse
func (td *TokenDiscovery) cacheToken(token *DiscoveredToken) {
	td.mu.Lock()
	defer td.mu.Unlock()
	td.cachedToken = token
}

// ClearCache clears the cached token
func (td *TokenDiscovery) ClearCache() {
	td.mu.Lock()
	defer td.mu.Unlock()
	td.cachedToken = nil
}

// SetEnabled enables or disables token discovery
func (td *TokenDiscovery) SetEnabled(enabled bool) {
	td.enabled = enabled
}

// IsEnabled returns whether token discovery is enabled
func (td *TokenDiscovery) IsEnabled() bool {
	return td.enabled
}

// RefreshToken forces generation of a new token
func (td *TokenDiscovery) RefreshToken(ctx context.Context) (*DiscoveredToken, error) {
	td.ClearCache()

	// Don't use environment variable for refresh, always generate new
	token, err := td.generateToken(ctx)
	if err != nil {
		return nil, err
	}

	td.cacheToken(token)
	return token, nil
}

// IsTokenExpired checks if the cached token is expired
func (td *TokenDiscovery) IsTokenExpired() bool {
	td.mu.RLock()
	defer td.mu.RUnlock()

	if td.cachedToken == nil {
		return true
	}

	return time.Now().After(td.cachedToken.ExpiresAt)
}

// GetTokenExpiresIn returns the duration until the cached token expires
func (td *TokenDiscovery) GetTokenExpiresIn() time.Duration {
	td.mu.RLock()
	defer td.mu.RUnlock()

	if td.cachedToken == nil {
		return 0
	}

	return time.Until(td.cachedToken.ExpiresAt)
}

// TokenToAuthToken converts a DiscoveredToken to an auth.Token
func (dt *DiscoveredToken) ToAuthToken(clusterID string) *Token {
	return &Token{
		AccessToken: dt.Token,
		TokenType:   "Bearer",
		ExpiresAt:   dt.ExpiresAt,
		ClusterID:   clusterID,
		Metadata: map[string]string{
			"username":    dt.Username,
			"source":      dt.Source,
			"auth_method": "auto-discovery",
		},
	}
}
