package auth

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/jontk/s9s/internal/debug"
)

// SlurmTokenAuthenticator implements authentication using SLURM's native token system
type SlurmTokenAuthenticator struct {
	config AuthConfig
	user   string
}

// NewSlurmTokenAuthenticator creates a new SLURM token authenticator
func NewSlurmTokenAuthenticator() Authenticator {
	return &SlurmTokenAuthenticator{}
}

// GetInfo returns information about this authenticator
func (s *SlurmTokenAuthenticator) GetInfo() AuthenticatorInfo {
	return AuthenticatorInfo{
		Name:        "slurm-token",
		Version:     "1.0.0",
		Description: "SLURM native token authentication using scontrol token",
		Author:      "s9s Team",
		Supported:   []string{"slurm-token"},
	}
}

// GetConfigSchema returns the configuration schema for this authenticator
func (s *SlurmTokenAuthenticator) GetConfigSchema() ConfigSchema {
	return ConfigSchema{
		Properties: map[string]PropertySchema{
			"username": {
				Type:        "string",
				Description: "Username for token generation (defaults to $USER)",
				Required:    false,
				Default:     "${USER}",
			},
			"slurm_config_path": {
				Type:        "string",
				Description: "Path to SLURM configuration file",
				Required:    false,
				Default:     "/etc/slurm/slurm.conf",
			},
			"token_lifetime": {
				Type:        "integer",
				Description: "Token lifetime in seconds",
				Required:    false,
				Default:     3600,
			},
			"scontrol_path": {
				Type:        "string",
				Description: "Path to scontrol binary",
				Required:    false,
				Default:     "scontrol",
			},
		},
		Required: []string{},
	}
}

// Initialize initializes the SLURM token authenticator
func (s *SlurmTokenAuthenticator) Initialize(ctx context.Context, config AuthConfig) error {
	s.config = config

	// Get username from config or environment
	s.user = config.GetString("username")
	if s.user == "" || s.user == "${USER}" {
		s.user = os.Getenv("USER")
	}
	if s.user == "" {
		return fmt.Errorf("username not specified and $USER environment variable not set")
	}

	debug.Logger.Printf("Initialized SLURM token authenticator for user: %s", s.user)
	return nil
}

// Authenticate generates a new SLURM token
func (s *SlurmTokenAuthenticator) Authenticate(ctx context.Context, config AuthConfig) (*Token, error) {
	debug.Logger.Printf("Authenticating with SLURM token for user: %s", s.user)

	// Get configuration values
	scontrolPath := config.GetString("scontrol_path")
	if scontrolPath == "" {
		scontrolPath = "scontrol"
	}

	tokenLifetime := config.GetInt("token_lifetime")
	if tokenLifetime == 0 {
		tokenLifetime = 3600 // Default to 1 hour
	}

	// Execute scontrol token command
	cmd := exec.CommandContext(ctx, scontrolPath, "token", fmt.Sprintf("username=%s", s.user), fmt.Sprintf("lifespan=%d", tokenLifetime))

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to generate SLURM token: %w", err)
	}

	// Parse the token from output
	token, expiresAt, err := s.parseTokenOutput(string(output), tokenLifetime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token output: %w", err)
	}

	debug.Logger.Printf("Successfully generated SLURM token, expires at: %v", expiresAt)

	return &Token{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		ClusterID:   config.GetString("cluster_id"),
		Metadata: map[string]string{
			"username":     s.user,
			"auth_method":  "slurm-token",
			"generated_by": "scontrol",
		},
	}, nil
}

// parseTokenOutput extracts the token from scontrol output
func (s *SlurmTokenAuthenticator) parseTokenOutput(output string, lifetime int) (string, time.Time, error) {
	// scontrol token output format:
	// SLURM_JWT=eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9...

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SLURM_JWT=") {
			token := strings.TrimPrefix(line, "SLURM_JWT=")
			if token != "" {
				// Calculate expiration time based on lifetime
				expiresAt := time.Now().Add(time.Duration(lifetime) * time.Second)
				return token, expiresAt, nil
			}
		}
	}

	// Alternative format parsing - some versions might output differently
	tokenRegex := regexp.MustCompile(`(?i)token[:\s]*([A-Za-z0-9\-._~+/]+=*)`)
	matches := tokenRegex.FindStringSubmatch(output)
	if len(matches) > 1 && matches[1] != "" {
		expiresAt := time.Now().Add(time.Duration(lifetime) * time.Second)
		return matches[1], expiresAt, nil
	}

	return "", time.Time{}, fmt.Errorf("could not find token in scontrol output: %s", output)
}

// RefreshToken generates a new token (SLURM tokens cannot be refreshed, only regenerated)
func (s *SlurmTokenAuthenticator) RefreshToken(ctx context.Context, token *Token) (*Token, error) {
	debug.Logger.Printf("Refreshing SLURM token (regenerating)")

	// For SLURM tokens, refresh means generating a new token
	return s.Authenticate(ctx, s.config)
}

// ValidateToken validates a SLURM token by checking expiration
func (s *SlurmTokenAuthenticator) ValidateToken(ctx context.Context, token *Token) error {
	if token == nil {
		return fmt.Errorf("token is nil")
	}

	if token.AccessToken == "" {
		return fmt.Errorf("token is empty")
	}

	if token.IsExpired() {
		return fmt.Errorf("token is expired")
	}

	debug.Logger.Printf("SLURM token is valid, expires in: %v", token.ExpiresIn())
	return nil
}

// RevokeToken revokes a SLURM token (not supported by SLURM, token will expire naturally)
func (s *SlurmTokenAuthenticator) RevokeToken(ctx context.Context, token *Token) error {
	debug.Logger.Printf("SLURM token revocation requested - tokens will expire naturally")

	// SLURM doesn't support token revocation, tokens expire based on their lifetime
	// We could potentially try to generate a very short-lived token to "revoke" the current one
	// but that's not really revocation. For now, we'll just log and return success.
	return nil
}

// Cleanup performs any necessary cleanup
func (s *SlurmTokenAuthenticator) Cleanup() error {
	debug.Logger.Printf("SLURM token authenticator cleanup completed")
	return nil
}
