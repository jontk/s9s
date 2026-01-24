package auth

import (
	"context"
	"fmt"
	"time"
)

// Provider defines the interface for authentication providers
type Provider interface {
	// Authenticate performs authentication and returns a token
	Authenticate(ctx context.Context, username, password string) (*Token, error)

	// RefreshToken refreshes an existing token
	RefreshToken(ctx context.Context, token *Token) (*Token, error)

	// ValidateToken validates if a token is still valid
	ValidateToken(ctx context.Context, token *Token) error

	// Logout invalidates a token
	Logout(ctx context.Context, token *Token) error
}

// AuthProvider is an alias for backward compatibility
type AuthProvider = Provider

// SlurmAuthProvider implements authentication against SLURM REST API
type SlurmAuthProvider struct {
	baseURL string
	timeout time.Duration
}

// NewSlurmAuthProvider creates a new SLURM authentication provider
func NewSlurmAuthProvider(baseURL string, timeout time.Duration) *SlurmAuthProvider {
	return &SlurmAuthProvider{
		baseURL: baseURL,
		timeout: timeout,
	}
}

// Authenticate performs authentication against SLURM REST API
func (s *SlurmAuthProvider) Authenticate(_ context.Context, username, _ string) (*Token, error) {
	// In a real implementation, this would:
	// 1. Make an HTTP POST request to the SLURM REST API auth endpoint
	// 2. Parse the response to extract the JWT token
	// 3. Create and return a Token object

	// For now, create a mock token
	return CreateToken(username, s.baseURL, DefaultTokenExpiry)
}

// RefreshToken refreshes an existing token
func (s *SlurmAuthProvider) RefreshToken(_ context.Context, token *Token) (*Token, error) {
	// In a real implementation, this would use the refresh endpoint
	// For now, just create a new token
	// Extract username and cluster from token metadata
	username := token.Metadata["username"]
	if username == "" {
		username = "unknown"
	}
	return CreateToken(username, token.ClusterID, DefaultTokenExpiry)
}

// ValidateToken validates if a token is still valid
func (s *SlurmAuthProvider) ValidateToken(_ context.Context, token *Token) error {
	if token.IsExpired() {
		return ErrTokenExpired
	}

	// In a real implementation, this would make a request to validate the token
	// For now, just check JWT validity
	_, err := ValidateJWT(token.AccessToken)
	return err
}

// Logout invalidates a token
func (s *SlurmAuthProvider) Logout(_ context.Context, _ *Token) error {
	// In a real implementation, this would call the logout endpoint
	// to invalidate the token server-side
	return nil
}

// Common authentication errors
var (
	ErrTokenExpired = fmt.Errorf("token has expired")
	ErrInvalidToken = fmt.Errorf("invalid token")
	ErrAuthFailed   = fmt.Errorf("authentication failed")
)
