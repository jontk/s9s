// Package auth provides authentication and authorization functionality for s9s.
package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jontk/s9s/internal/debug"
)

// APIAuthenticator implements authentication against a configurable API endpoint
type APIAuthenticator struct {
	config     AuthConfig
	httpClient *http.Client
}

// NewAPIAuthenticator creates a new API authenticator
func NewAPIAuthenticator() Authenticator {
	return &APIAuthenticator{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetInfo returns information about this authenticator
func (a *APIAuthenticator) GetInfo() AuthenticatorInfo {
	return AuthenticatorInfo{
		Name:        "api-auth",
		Version:     "1.0.0",
		Description: "API endpoint authentication with JWT token support",
		Author:      "s9s Team",
		Supported:   []string{"api-auth", "jwt", "rest-api"},
	}
}

// GetConfigSchema returns the configuration schema for this authenticator
func (a *APIAuthenticator) GetConfigSchema() ConfigSchema {
	return ConfigSchema{
		Properties: map[string]PropertySchema{
			"endpoint": {
				Type:        "string",
				Description: "Authentication API endpoint URL",
				Required:    true,
			},
			"method": {
				Type:        "string",
				Description: "HTTP method for authentication",
				Required:    false,
				Default:     "POST",
				Enum:        []string{"POST", "GET"},
			},
			"username": {
				Type:        "string",
				Description: "Username for authentication",
				Required:    true,
				Sensitive:   false,
			},
			"password": {
				Type:        "string",
				Description: "Password for authentication",
				Required:    true,
				Sensitive:   true,
			},
			"token_path": {
				Type:        "string",
				Description: "JSON path to access token in response",
				Required:    false,
				Default:     "access_token",
			},
			"refresh_token_path": {
				Type:        "string",
				Description: "JSON path to refresh token in response",
				Required:    false,
				Default:     "refresh_token",
			},
			"expiry_path": {
				Type:        "string",
				Description: "JSON path to token expiry in response",
				Required:    false,
				Default:     "expires_in",
			},
			"timeout": {
				Type:        "integer",
				Description: "HTTP request timeout in seconds",
				Required:    false,
				Default:     30,
			},
		},
		Required: []string{"endpoint", "username", "password"},
	}
}

// Initialize initializes the API authenticator
func (a *APIAuthenticator) Initialize(_ context.Context, config AuthConfig) error {
	a.config = config

	// Set custom timeout if provided
	if timeout := config.GetInt("timeout"); timeout > 0 {
		a.httpClient.Timeout = time.Duration(timeout) * time.Second
	}

	endpoint := config.GetString("endpoint")
	if endpoint == "" {
		return fmt.Errorf("endpoint is required")
	}

	debug.Logger.Printf("Initialized API authenticator for endpoint: %s", endpoint)
	return nil
}

// Authenticate authenticates against the API endpoint
func (a *APIAuthenticator) Authenticate(ctx context.Context, config AuthConfig) (*Token, error) {
	endpoint := config.GetString("endpoint")
	method := a.getHTTPMethod(config)

	debug.Logger.Printf("Authenticating with API endpoint: %s", endpoint)

	// Prepare request payload
	payloadBytes, err := a.preparePayload(config)
	if err != nil {
		return nil, err
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set request headers
	a.setRequestHeaders(req, config)

	// Execute request
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle response
	responseData, err := a.handleResponse(resp)
	if err != nil {
		return nil, err
	}

	// Extract token information
	token, err := a.extractToken(responseData, config)
	if err != nil {
		return nil, fmt.Errorf("failed to extract token from response: %w", err)
	}

	debug.Logger.Printf("Successfully authenticated via API, token expires at: %v", token.ExpiresAt)
	return token, nil
}

// getHTTPMethod returns the HTTP method, defaulting to POST
func (a *APIAuthenticator) getHTTPMethod(config AuthConfig) string {
	method := config.GetString("method")
	if method == "" {
		return "POST"
	}
	return method
}

// preparePayload creates the request payload
func (a *APIAuthenticator) preparePayload(config AuthConfig) ([]byte, error) {
	payload := map[string]interface{}{
		"username": config.GetString("username"),
		"password": config.GetString("password"),
	}

	// Add client_id if present
	if clientID := config.GetString("client_id"); clientID != "" {
		payload["client_id"] = clientID
	}

	// Add grant_type (default to "password")
	if grantType := config.GetString("grant_type"); grantType != "" {
		payload["grant_type"] = grantType
	} else {
		payload["grant_type"] = "password"
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request payload: %w", err)
	}
	return payloadBytes, nil
}

// setRequestHeaders sets required and custom headers on the request
func (a *APIAuthenticator) setRequestHeaders(req *http.Request, config AuthConfig) {
	// Set standard headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "s9s/1.0")

	// Add custom headers from config
	if headers := config.Get("headers"); headers != nil {
		if headerMap, ok := headers.(map[string]interface{}); ok {
			for key, value := range headerMap {
				if strValue, ok := value.(string); ok {
					req.Header.Set(key, strValue)
				}
			}
		}
	}
}

// handleResponse reads and validates the HTTP response
func (a *APIAuthenticator) handleResponse(resp *http.Response) (map[string]interface{}, error) {
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		debug.Logger.Printf("API authentication failed with status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("authentication failed with status %d", resp.StatusCode)
	}

	// Parse JSON response
	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return responseData, nil
}

// extractToken extracts token information from the API response
func (a *APIAuthenticator) extractToken(data map[string]interface{}, config AuthConfig) (*Token, error) {
	// Get configured paths or use defaults
	tokenPath := a.getPathOrDefault(config, "token_path", "access_token")
	refreshTokenPath := a.getPathOrDefault(config, "refresh_token_path", "refresh_token")
	expiryPath := a.getPathOrDefault(config, "expiry_path", "expires_in")

	// Extract access token (required)
	accessTokenStr, err := a.extractAccessToken(data, tokenPath)
	if err != nil {
		return nil, err
	}

	// Extract optional fields
	refreshToken := a.extractRefreshToken(data, refreshTokenPath)
	expiresAt := a.extractExpiry(data, expiryPath)
	tokenType := a.extractTokenType(data)

	return &Token{
		AccessToken:  accessTokenStr,
		RefreshToken: refreshToken,
		TokenType:    tokenType,
		ExpiresAt:    expiresAt,
		ClusterID:    config.GetString("cluster_id"),
		Metadata: map[string]string{
			"auth_method": "api-auth",
			"endpoint":    config.GetString("endpoint"),
		},
	}, nil
}

// getPathOrDefault returns a path from config or the default value
func (a *APIAuthenticator) getPathOrDefault(config AuthConfig, key, defaultValue string) string {
	if val := config.GetString(key); val != "" {
		return val
	}
	return defaultValue
}

// extractAccessToken extracts and validates the access token
func (a *APIAuthenticator) extractAccessToken(data map[string]interface{}, tokenPath string) (string, error) {
	accessToken, err := a.extractValueFromPath(data, tokenPath)
	if err != nil {
		return "", fmt.Errorf("failed to extract access token: %w", err)
	}

	accessTokenStr, ok := accessToken.(string)
	if !ok {
		return "", fmt.Errorf("access token is not a string")
	}
	return accessTokenStr, nil
}

// extractRefreshToken extracts the refresh token if available
func (a *APIAuthenticator) extractRefreshToken(data map[string]interface{}, refreshTokenPath string) string {
	if refreshTokenVal, err := a.extractValueFromPath(data, refreshTokenPath); err == nil {
		if refreshTokenStr, ok := refreshTokenVal.(string); ok {
			return refreshTokenStr
		}
	}
	return ""
}

// extractExpiry extracts the token expiry time with fallback to 1 hour
func (a *APIAuthenticator) extractExpiry(data map[string]interface{}, expiryPath string) time.Time {
	if expiryVal, err := a.extractValueFromPath(data, expiryPath); err == nil {
		return a.parseExpiryValue(expiryVal)
	}
	return time.Now().Add(1 * time.Hour)
}

// parseExpiryValue parses an expiry value that can be float64, int, or string
func (a *APIAuthenticator) parseExpiryValue(val interface{}) time.Time {
	switch exp := val.(type) {
	case float64:
		return time.Now().Add(time.Duration(exp) * time.Second)
	case int:
		return time.Now().Add(time.Duration(exp) * time.Second)
	case string:
		// Try to parse as RFC3339 timestamp
		if parsedTime, err := time.Parse(time.RFC3339, exp); err == nil {
			return parsedTime
		}
	}
	// Default to 1 hour
	return time.Now().Add(1 * time.Hour)
}

// extractTokenType extracts the token type with fallback to "Bearer"
func (a *APIAuthenticator) extractTokenType(data map[string]interface{}) string {
	if tokenTypeVal, err := a.extractValueFromPath(data, "token_type"); err == nil {
		if tokenTypeStr, ok := tokenTypeVal.(string); ok {
			return tokenTypeStr
		}
	}
	return "Bearer"
}

// extractValueFromPath extracts a value from a nested map using a simple path
func (a *APIAuthenticator) extractValueFromPath(data map[string]interface{}, path string) (interface{}, error) {
	// Handle simple paths (no nesting for now)
	if value, exists := data[path]; exists {
		return value, nil
	}

	// Handle JSON path-like syntax (simple dot notation)
	if strings.Contains(path, ".") {
		parts := strings.Split(path, ".")
		current := data
		for _, part := range parts {
			if val, exists := current[part]; exists {
				if nextMap, ok := val.(map[string]interface{}); ok {
					current = nextMap
				} else {
					return val, nil
				}
			} else {
				return nil, fmt.Errorf("path %s not found", path)
			}
		}
	}

	return nil, fmt.Errorf("path %s not found in response", path)
}

// RefreshToken refreshes an expired token using the refresh token
func (a *APIAuthenticator) RefreshToken(ctx context.Context, token *Token) (*Token, error) {
	if token.RefreshToken == "" {
		debug.Logger.Printf("No refresh token available, re-authenticating")
		return a.Authenticate(ctx, a.config)
	}

	debug.Logger.Printf("Refreshing token using refresh token")

	// Determine endpoint
	endpoint := a.resolveRefreshEndpoint()

	// Prepare and send refresh request
	payloadBytes, err := a.prepareRefreshPayload(token)
	if err != nil {
		return nil, err
	}

	statusCode, body, err := a.sendAPIRefreshRequest(ctx, endpoint, payloadBytes)
	if err != nil {
		return nil, err
	}

	// Check response status
	if statusCode < 200 || statusCode >= 300 {
		debug.Logger.Printf("Token refresh failed with status %d, re-authenticating", statusCode)
		return a.Authenticate(ctx, a.config)
	}

	// Parse response and extract token
	var responseData map[string]interface{}
	if err := json.Unmarshal(body, &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	newToken, err := a.extractToken(responseData, a.config)
	if err != nil {
		return nil, fmt.Errorf("failed to extract refreshed token: %w", err)
	}

	debug.Logger.Printf("Successfully refreshed token, expires at: %v", newToken.ExpiresAt)
	return newToken, nil
}

// resolveRefreshEndpoint returns the refresh endpoint with fallback to main endpoint
func (a *APIAuthenticator) resolveRefreshEndpoint() string {
	if endpoint := a.config.GetString("refresh_endpoint"); endpoint != "" {
		return endpoint
	}
	return a.config.GetString("endpoint")
}

// prepareRefreshPayload builds the refresh token request payload
func (a *APIAuthenticator) prepareRefreshPayload(token *Token) ([]byte, error) {
	payload := map[string]interface{}{
		"grant_type":    "refresh_token",
		"refresh_token": token.RefreshToken,
	}

	if clientID := a.config.GetString("client_id"); clientID != "" {
		payload["client_id"] = clientID
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal refresh payload: %w", err)
	}
	return payloadBytes, nil
}

// sendAPIRefreshRequest sends the refresh request to the API endpoint
func (a *APIAuthenticator) sendAPIRefreshRequest(ctx context.Context, endpoint string, payloadBytes []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "s9s/1.0")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to read refresh response: %w", err)
	}

	return resp.StatusCode, body, nil
}

// ValidateToken validates a token by checking its structure and expiration
func (a *APIAuthenticator) ValidateToken(ctx context.Context, token *Token) error {
	if token == nil {
		return fmt.Errorf("token is nil")
	}

	if token.AccessToken == "" {
		return fmt.Errorf("access token is empty")
	}

	if token.IsExpired() {
		return fmt.Errorf("token is expired")
	}

	// Optionally validate against API endpoint
	if validateEndpoint := a.config.GetString("validate_endpoint"); validateEndpoint != "" {
		return a.validateTokenWithAPI(ctx, token, validateEndpoint)
	}

	debug.Logger.Printf("Token is valid, expires in: %v", token.ExpiresIn())
	return nil
}

// validateTokenWithAPI validates a token by calling the API validation endpoint
func (a *APIAuthenticator) validateTokenWithAPI(ctx context.Context, token *Token, endpoint string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create validation request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.AccessToken))
	req.Header.Set("User-Agent", "s9s/1.0")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("token validation request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token validation failed with status %d", resp.StatusCode)
	}

	return nil
}

// RevokeToken revokes a token if the API supports it
func (a *APIAuthenticator) RevokeToken(ctx context.Context, token *Token) error {
	revokeEndpoint := a.config.GetString("revoke_endpoint")
	if revokeEndpoint == "" {
		debug.Logger.Printf("No revoke endpoint configured, token will expire naturally")
		return nil
	}

	debug.Logger.Printf("Revoking token via API endpoint: %s", revokeEndpoint)

	payload := map[string]interface{}{
		"token": token.AccessToken,
	}

	if token.RefreshToken != "" {
		payload["refresh_token"] = token.RefreshToken
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal revoke payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", revokeEndpoint, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create revoke request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.AccessToken))
	req.Header.Set("User-Agent", "s9s/1.0")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("token revocation request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("token revocation failed with status %d", resp.StatusCode)
	}

	debug.Logger.Printf("Successfully revoked token")
	return nil
}

// Cleanup performs any necessary cleanup
func (a *APIAuthenticator) Cleanup() error {
	debug.Logger.Printf("API authenticator cleanup completed")
	return nil
}
