package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zalando/go-keyring"
)

const (
	// KeyringService is the service name for storing tokens in the system keyring
	KeyringService = "s9s"

	// TokenCacheFile is the filename for the cached tokens
	TokenCacheFile = "tokens.json"

	// DefaultTokenExpiry is the default token expiration time
	DefaultTokenExpiry = 24 * time.Hour
)

// SimpleToken represents a cached authentication token (for backward compatibility)
type SimpleToken struct {
	Value      string    `json:"value"`
	ExpiresAt  time.Time `json:"expires_at"`
	ClusterURL string    `json:"cluster_url"`
	Username   string    `json:"username"`
}

// IsExpired checks if the token has expired
func (t *SimpleToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// TokenManager manages authentication tokens for multiple clusters
type TokenManager struct {
	mu         sync.RWMutex
	tokens     map[string]*Token // key is cluster name
	cacheDir   string
	useKeyring bool
}

// NewTokenManager creates a new token manager
func NewTokenManager(cacheDir string, useKeyring bool) *TokenManager {
	return &TokenManager{
		tokens:     make(map[string]*Token),
		cacheDir:   cacheDir,
		useKeyring: useKeyring,
	}
}

// GetToken retrieves a token for the specified cluster
func (tm *TokenManager) GetToken(clusterName string) (*Token, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// Check in-memory cache first
	if token, exists := tm.tokens[clusterName]; exists && !token.IsExpired() {
		return token, nil
	}

	// Try to load from keyring if enabled
	if tm.useKeyring {
		token, err := tm.loadFromKeyring(clusterName)
		if err == nil && !token.IsExpired() {
			tm.mu.RUnlock()
			tm.mu.Lock()
			tm.tokens[clusterName] = token
			tm.mu.Unlock()
			tm.mu.RLock()
			return token, nil
		}
	}

	// Try to load from file cache
	token, err := tm.loadFromFile(clusterName)
	if err == nil && !token.IsExpired() {
		tm.mu.RUnlock()
		tm.mu.Lock()
		tm.tokens[clusterName] = token
		tm.mu.Unlock()
		tm.mu.RLock()
		return token, nil
	}

	return nil, fmt.Errorf("no valid token found for cluster %s", clusterName)
}

// SetToken stores a token for the specified cluster
func (tm *TokenManager) SetToken(clusterName string, token *Token) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Store in memory
	tm.tokens[clusterName] = token

	// Store in keyring if enabled
	if tm.useKeyring {
		if err := tm.saveToKeyring(clusterName, token); err != nil {
			// Log error but don't fail if keyring storage fails
			_, _ = fmt.Fprintf(os.Stderr, "Failed to save token to keyring: %v\n", err)
		}
	}

	// Always store in file cache as backup
	return tm.saveToFile(clusterName, token)
}

// RemoveToken removes a token for the specified cluster
func (tm *TokenManager) RemoveToken(clusterName string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Remove from memory
	delete(tm.tokens, clusterName)

	// Remove from keyring if enabled
	if tm.useKeyring {
		_ = keyring.Delete(KeyringService, clusterName)
	}

	// Remove from file cache
	return tm.removeFromFile(clusterName)
}

// ListClusters returns a list of clusters with stored tokens
func (tm *TokenManager) ListClusters() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	clusters := make([]string, 0, len(tm.tokens))
	for cluster := range tm.tokens {
		clusters = append(clusters, cluster)
	}
	return clusters
}

// loadFromKeyring loads a token from the system keyring
func (tm *TokenManager) loadFromKeyring(clusterName string) (*Token, error) {
	secret, err := keyring.Get(KeyringService, clusterName)
	if err != nil {
		return nil, err
	}

	var token Token
	if err := json.Unmarshal([]byte(secret), &token); err != nil {
		return nil, err
	}

	return &token, nil
}

// saveToKeyring saves a token to the system keyring
func (tm *TokenManager) saveToKeyring(clusterName string, token *Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}

	return keyring.Set(KeyringService, clusterName, string(data))
}

// loadFromFile loads a token from the file cache
func (tm *TokenManager) loadFromFile(clusterName string) (*Token, error) {
	cachePath := filepath.Join(tm.cacheDir, TokenCacheFile)

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("token cache not found")
		}
		return nil, err
	}

	var cache map[string]*Token
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	token, exists := cache[clusterName]
	if !exists {
		return nil, fmt.Errorf("token not found in cache")
	}

	return token, nil
}

// saveToFile saves a token to the file cache
func (tm *TokenManager) saveToFile(clusterName string, token *Token) error {
	cachePath := filepath.Join(tm.cacheDir, TokenCacheFile)

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(tm.cacheDir, 0700); err != nil {
		return err
	}

	// Load existing cache
	cache := make(map[string]*Token)
	if data, err := os.ReadFile(cachePath); err == nil {
		_ = json.Unmarshal(data, &cache)
	}

	// Update cache
	cache[clusterName] = token

	// Save cache
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0600)
}

// removeFromFile removes a token from the file cache
func (tm *TokenManager) removeFromFile(clusterName string) error {
	cachePath := filepath.Join(tm.cacheDir, TokenCacheFile)

	// Load existing cache
	cache := make(map[string]*Token)
	if data, err := os.ReadFile(cachePath); err == nil {
		_ = json.Unmarshal(data, &cache)
	}

	// Remove from cache
	delete(cache, clusterName)

	// Save cache
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(cachePath, data, 0600)
}

// ValidateJWT validates a JWT token and extracts claims
func ValidateJWT(tokenString string) (*jwt.RegisteredClaims, error) {
	// Parse the token
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		// This is where you would verify the signing key
		// For now, we'll accept any valid JWT structure
		// In production, you should verify against the cluster's public key
		return []byte("secret"), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// CreateToken creates a new token with the specified expiration
func CreateToken(username, clusterURL string, expiry time.Duration) (*Token, error) {
	expiresAt := time.Now().Add(expiry)

	// Create JWT claims
	claims := &jwt.RegisteredClaims{
		Subject:   username,
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "s9s",
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token
	// In production, this should use a proper signing key
	tokenString, err := token.SignedString([]byte("secret"))
	if err != nil {
		return nil, err
	}

	return &Token{
		AccessToken: tokenString,
		ExpiresAt:   expiresAt,
		TokenType:   "Bearer",
		ClusterID:   clusterURL,
		Metadata: map[string]string{
			"username": username,
		},
	}, nil
}
