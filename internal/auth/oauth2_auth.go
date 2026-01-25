package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/jontk/s9s/internal/debug"
)

// OAuth2Authenticator implements OAuth2/OIDC authentication
type OAuth2Authenticator struct {
	config     AuthConfig
	httpClient *http.Client
	server     *http.Server
	resultChan chan *oauth2Result
}

// oauth2Result holds the result of OAuth2 flow
type oauth2Result struct {
	token *Token
	err   error
}

// oidcDiscovery represents OIDC discovery document
type oidcDiscovery struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JwksURI               string `json:"jwks_uri"`
	Issuer                string `json:"issuer"`
	RevocationEndpoint    string `json:"revocation_endpoint,omitempty"`
}

// NewOAuth2Authenticator creates a new OAuth2 authenticator
func NewOAuth2Authenticator() Authenticator {
	return &OAuth2Authenticator{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		resultChan: make(chan *oauth2Result, 1),
	}
}

// GetInfo returns information about this authenticator
func (o *OAuth2Authenticator) GetInfo() AuthenticatorInfo {
	return AuthenticatorInfo{
		Name:        "oauth2",
		Version:     "1.0.0",
		Description: "OAuth2/OIDC authentication with support for major providers",
		Author:      "s9s Team",
		Supported:   []string{"oauth2", "oidc", "okta", "azure-ad", "google", "github"},
	}
}

// GetConfigSchema returns the configuration schema for this authenticator
func (o *OAuth2Authenticator) GetConfigSchema() ConfigSchema {
	return ConfigSchema{
		Properties: map[string]PropertySchema{
			"provider": {
				Type:        "string",
				Description: "OAuth2 provider (okta, azure-ad, google, github, custom)",
				Required:    false,
				Default:     "custom",
				Enum:        []string{"okta", "azure-ad", "google", "github", "custom"},
			},
			"client_id": {
				Type:        "string",
				Description: "OAuth2 client ID",
				Required:    true,
				Sensitive:   false,
			},
			"client_secret": {
				Type:        "string",
				Description: "OAuth2 client secret",
				Required:    true,
				Sensitive:   true,
			},
			"discovery_url": {
				Type:        "string",
				Description: "OIDC discovery URL (auto-detected for known providers)",
				Required:    false,
			},
			"authorization_endpoint": {
				Type:        "string",
				Description: "OAuth2 authorization endpoint (required if not using discovery)",
				Required:    false,
			},
			"token_endpoint": {
				Type:        "string",
				Description: "OAuth2 token endpoint (required if not using discovery)",
				Required:    false,
			},
			"redirect_uri": {
				Type:        "string",
				Description: "OAuth2 redirect URI",
				Required:    false,
				Default:     "http://localhost:8080/callback",
			},
			"scopes": {
				Type:        "string",
				Description: "Space-separated list of OAuth2 scopes",
				Required:    false,
				Default:     "openid profile email",
			},
			"timeout": {
				Type:        "integer",
				Description: "Authentication timeout in seconds",
				Required:    false,
				Default:     300,
			},
		},
		Required: []string{"client_id", "client_secret"},
	}
}

// Initialize initializes the OAuth2 authenticator
func (o *OAuth2Authenticator) Initialize(_ context.Context, config AuthConfig) error {
	o.config = config

	// Set custom timeout if provided
	if timeout := config.GetInt("timeout"); timeout > 0 {
		o.httpClient.Timeout = time.Duration(timeout) * time.Second
	}

	// Validate required configuration
	if config.GetString("client_id") == "" {
		return fmt.Errorf("client_id is required")
	}
	if config.GetString("client_secret") == "" {
		return fmt.Errorf("client_secret is required")
	}

	debug.Logger.Printf("Initialized OAuth2 authenticator for provider: %s", config.GetString("provider"))
	return nil
}

// Authenticate performs OAuth2 authentication flow
func (o *OAuth2Authenticator) Authenticate(ctx context.Context, config AuthConfig) (*Token, error) {
	provider := o.resolveProvider(config)
	debug.Logger.Printf("Starting OAuth2 authentication flow for provider: %s", provider)

	// Get OAuth2 endpoints
	authEndpoint, tokenEndpoint, err := o.getOAuth2Endpoints(config)
	if err != nil {
		return nil, fmt.Errorf("failed to get OAuth2 endpoints: %w", err)
	}

	// Generate authentication parameters
	state, codeVerifier, codeChallenge, err := o.generateAuthParameters()
	if err != nil {
		return nil, err
	}

	// Setup callback server
	redirectURI, callbackPath, err := o.extractRedirectURI(config)
	if err != nil {
		return nil, err
	}

	if err := o.startCallbackServer(redirectURI, state, callbackPath); err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}
	defer o.stopCallbackServer()

	// Build and open authorization URL
	authURL, err := o.buildAuthorizationURL(authEndpoint, config, state, codeChallenge, redirectURI)
	if err != nil {
		return nil, fmt.Errorf("failed to build authorization URL: %w", err)
	}

	o.openBrowserForAuth(authURL)

	// Wait for callback or timeout
	return o.waitForCallback(ctx, tokenEndpoint, codeVerifier, redirectURI, config)
}

// resolveProvider returns the provider name or "custom" as default
func (o *OAuth2Authenticator) resolveProvider(config AuthConfig) string {
	provider := config.GetString("provider")
	if provider == "" {
		return "custom"
	}
	return provider
}

// generateAuthParameters generates state, code verifier, and code challenge
func (o *OAuth2Authenticator) generateAuthParameters() (string, string, string, error) {
	state, err := o.generateRandomString(32)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate state: %w", err)
	}

	codeVerifier, err := o.generateRandomString(43)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate code verifier: %w", err)
	}

	codeChallenge := o.generateCodeChallenge(codeVerifier)
	return state, codeVerifier, codeChallenge, nil
}

// extractRedirectURI extracts the redirect URI and callback path from config
func (o *OAuth2Authenticator) extractRedirectURI(config AuthConfig) (string, string, error) {
	redirectURI := config.GetString("redirect_uri")
	if redirectURI == "" {
		redirectURI = "http://localhost:8080/callback"
	}

	callbackPath := "/callback"
	if u, err := url.Parse(redirectURI); err == nil {
		callbackPath = u.Path
	}

	return redirectURI, callbackPath, nil
}

// openBrowserForAuth prints auth URL and attempts to open browser
func (o *OAuth2Authenticator) openBrowserForAuth(authURL string) {
	fmt.Printf("Opening browser for OAuth2 authentication...\n")
	fmt.Printf("If the browser doesn't open automatically, visit: %s\n", authURL)

	if err := o.openBrowser(authURL); err != nil {
		debug.Logger.Printf("Failed to open browser automatically: %v", err)
	}
}

// waitForCallback waits for the OAuth2 callback with timeout
func (o *OAuth2Authenticator) waitForCallback(ctx context.Context, tokenEndpoint, codeVerifier, redirectURI string, config AuthConfig) (*Token, error) {
	select {
	case result := <-o.resultChan:
		if result.err != nil {
			return nil, result.err
		}

		// Exchange authorization code for token
		authCode := result.token.AccessToken // Temporarily stored in AccessToken
		return o.exchangeCodeForToken(tokenEndpoint, authCode, codeVerifier, redirectURI, config)

	case <-ctx.Done():
		return nil, fmt.Errorf("authentication timeout or cancelled")
	}
}

// getOAuth2Endpoints gets the OAuth2 endpoints from discovery or configuration
func (o *OAuth2Authenticator) getOAuth2Endpoints(config AuthConfig) (string, string, error) {
	// Check if discovery URL is provided
	discoveryURL := config.GetString("discovery_url")

	// Auto-detect discovery URL for known providers
	if discoveryURL == "" {
		provider := config.GetString("provider")
		switch provider {
		case "google":
			discoveryURL = "https://accounts.google.com/.well-known/openid_configuration"
		case "okta":
			// Okta discovery URL should be provided in config
			return "", "", fmt.Errorf("okta provider requires discovery_url in config")
		case "azure-ad":
			// Azure AD discovery URL should be provided in config
			return "", "", fmt.Errorf("azure-ad provider requires discovery_url in config")
		}
	}

	// Use discovery if URL is available
	if discoveryURL != "" {
		return o.discoverEndpoints(discoveryURL)
	}

	// Fall back to manual configuration
	authEndpoint := config.GetString("authorization_endpoint")
	tokenEndpoint := config.GetString("token_endpoint")

	if authEndpoint == "" || tokenEndpoint == "" {
		return "", "", fmt.Errorf("authorization_endpoint and token_endpoint are required when not using discovery")
	}

	return authEndpoint, tokenEndpoint, nil
}

// discoverEndpoints discovers OAuth2 endpoints using OIDC discovery
func (o *OAuth2Authenticator) discoverEndpoints(discoveryURL string) (string, string, error) {
	debug.Logger.Printf("Discovering OAuth2 endpoints from: %s", discoveryURL)

	req, err := http.NewRequestWithContext(context.Background(), "GET", discoveryURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("failed to create discovery request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "s9s/1.0")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("discovery request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("discovery failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("failed to read discovery response: %w", err)
	}

	var discovery oidcDiscovery
	if err := json.Unmarshal(body, &discovery); err != nil {
		return "", "", fmt.Errorf("failed to parse discovery response: %w", err)
	}

	if discovery.AuthorizationEndpoint == "" || discovery.TokenEndpoint == "" {
		return "", "", fmt.Errorf("invalid discovery response: missing required endpoints")
	}

	debug.Logger.Printf("Discovered endpoints - Auth: %s, Token: %s", discovery.AuthorizationEndpoint, discovery.TokenEndpoint)
	return discovery.AuthorizationEndpoint, discovery.TokenEndpoint, nil
}

// startCallbackServer starts the local HTTP server for OAuth2 callback
func (o *OAuth2Authenticator) startCallbackServer(redirectURI, state, callbackPath string) error {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return fmt.Errorf("invalid redirect URI: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(callbackPath, func(w http.ResponseWriter, r *http.Request) {
		o.handleCallback(w, r, state)
	})

	o.server = &http.Server{
		Addr:              u.Host,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := o.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			o.resultChan <- &oauth2Result{err: fmt.Errorf("callback server error: %w", err)}
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)
	debug.Logger.Printf("Started OAuth2 callback server on: %s", u.Host)
	return nil
}

// stopCallbackServer stops the local callback server
func (o *OAuth2Authenticator) stopCallbackServer() {
	if o.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = o.server.Shutdown(ctx)
		o.server = nil
	}
}

// handleCallback handles the OAuth2 callback
func (o *OAuth2Authenticator) handleCallback(w http.ResponseWriter, r *http.Request, expectedState string) {
	// Check for errors
	if errMsg := r.URL.Query().Get("error"); errMsg != "" {
		errorDesc := r.URL.Query().Get("error_description")
		o.resultChan <- &oauth2Result{err: fmt.Errorf("OAuth2 error: %s - %s", errMsg, errorDesc)}

		http.Error(w, fmt.Sprintf("Authentication failed: %s", errMsg), http.StatusBadRequest)
		return
	}

	// Verify state
	state := r.URL.Query().Get("state")
	if state != expectedState {
		o.resultChan <- &oauth2Result{err: fmt.Errorf("invalid state parameter")}

		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	// Get authorization code
	code := r.URL.Query().Get("code")
	if code == "" {
		o.resultChan <- &oauth2Result{err: fmt.Errorf("missing authorization code")}

		http.Error(w, "Missing authorization code", http.StatusBadRequest)
		return
	}

	// Success response
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`
		<html>
		<body>
			<h2>Authentication Successful</h2>
			<p>You can now close this browser window.</p>
			<script>window.close();</script>
		</body>
		</html>
	`))

	// Send result (code temporarily stored in AccessToken)
	o.resultChan <- &oauth2Result{
		token: &Token{AccessToken: code},
	}
}

// buildAuthorizationURL builds the OAuth2 authorization URL
func (o *OAuth2Authenticator) buildAuthorizationURL(authEndpoint string, config AuthConfig, state, codeChallenge, redirectURI string) (string, error) {
	u, err := url.Parse(authEndpoint)
	if err != nil {
		return "", fmt.Errorf("invalid authorization endpoint: %w", err)
	}

	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", config.GetString("client_id"))
	params.Set("redirect_uri", redirectURI)
	params.Set("state", state)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")

	scopes := config.GetString("scopes")
	if scopes == "" {
		scopes = "openid profile email"
	}
	params.Set("scope", scopes)

	// Add provider-specific parameters
	provider := config.GetString("provider")
	switch provider {
	case "okta":
		params.Set("response_mode", "query")
	case "azure-ad":
		params.Set("response_mode", "query")
		// Azure AD specific handling could be added here
		_ = config.GetString("tenant")
	}

	u.RawQuery = params.Encode()
	return u.String(), nil
}

// exchangeCodeForToken exchanges the authorization code for an access token
func (o *OAuth2Authenticator) exchangeCodeForToken(tokenEndpoint, code, codeVerifier, redirectURI string, config AuthConfig) (*Token, error) {
	debug.Logger.Printf("Exchanging authorization code for access token")

	// Prepare and send token exchange request
	req := o.prepareTokenExchangeRequest(tokenEndpoint, code, codeVerifier, redirectURI, config)

	body, err := o.sendTokenRequest(req)
	if err != nil {
		return nil, err
	}

	// Parse token response
	var tokenResponse map[string]interface{}
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	// Extract token from response
	token, err := o.extractExchangeTokenResponse(tokenResponse, config)
	if err != nil {
		return nil, err
	}

	debug.Logger.Printf("Successfully obtained OAuth2 token, expires at: %v", token.ExpiresAt)
	return token, nil
}

// prepareTokenExchangeRequest builds the token exchange request
func (o *OAuth2Authenticator) prepareTokenExchangeRequest(tokenEndpoint, code, codeVerifier, redirectURI string, config AuthConfig) *http.Request {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("client_id", config.GetString("client_id"))
	data.Set("client_secret", config.GetString("client_secret"))
	data.Set("code_verifier", codeVerifier)

	// Note: We can safely ignore the error here as we control all inputs
	req, _ := http.NewRequestWithContext(context.Background(), "POST", tokenEndpoint, strings.NewReader(data.Encode())) //nolint:errcheck
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "s9s/1.0")

	return req
}

// sendTokenRequest executes the token request and validates the response
func (o *OAuth2Authenticator) sendTokenRequest(req *http.Request) ([]byte, error) {
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		debug.Logger.Printf("Token request failed with status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("token request failed with status %d", resp.StatusCode)
	}

	return body, nil
}

// extractExchangeTokenResponse extracts all token fields from exchange response
func (o *OAuth2Authenticator) extractExchangeTokenResponse(response map[string]interface{}, config AuthConfig) (*Token, error) {
	// Extract and validate required access token
	accessToken, err := o.extractRequiredAccessToken(response)
	if err != nil {
		return nil, err
	}

	// Extract optional fields
	refreshToken := o.extractRefreshToken(response)
	expiresAt := o.extractExpiryTime(response)
	tokenType := o.extractTokenType(response)
	scopes := o.extractScopes(response)

	return &Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    tokenType,
		ExpiresAt:    expiresAt,
		Scopes:       scopes,
		ClusterID:    config.GetString("cluster_id"),
		Metadata: map[string]string{
			"auth_method": "oauth2",
			"provider":    config.GetString("provider"),
		},
	}, nil
}

// extractRequiredAccessToken extracts the required access token from response
func (o *OAuth2Authenticator) extractRequiredAccessToken(response map[string]interface{}) (string, error) {
	accessToken, ok := response["access_token"].(string)
	if !ok || accessToken == "" {
		return "", fmt.Errorf("missing or invalid access_token in response")
	}
	return accessToken, nil
}

// extractRefreshToken extracts the optional refresh token from response
func (o *OAuth2Authenticator) extractRefreshToken(response map[string]interface{}) string {
	if rt, ok := response["refresh_token"].(string); ok {
		return rt
	}
	return ""
}

// extractExpiryTime extracts the token expiry time from response
func (o *OAuth2Authenticator) extractExpiryTime(response map[string]interface{}) time.Time {
	if expiresIn, ok := response["expires_in"].(float64); ok {
		return time.Now().Add(time.Duration(expiresIn) * time.Second)
	}
	return time.Now().Add(1 * time.Hour)
}

// extractTokenType extracts the token type from response with default
func (o *OAuth2Authenticator) extractTokenType(response map[string]interface{}) string {
	if tt, ok := response["token_type"].(string); ok {
		return tt
	}
	return "Bearer"
}

// extractScopes extracts scopes from response
func (o *OAuth2Authenticator) extractScopes(response map[string]interface{}) []string {
	if scopeStr, ok := response["scope"].(string); ok {
		return strings.Fields(scopeStr)
	}
	return nil
}

// RefreshToken refreshes an expired token using the refresh token
func (o *OAuth2Authenticator) RefreshToken(ctx context.Context, token *Token) (*Token, error) {
	if token.RefreshToken == "" {
		debug.Logger.Printf("No refresh token available, re-authenticating")
		return o.Authenticate(ctx, o.config)
	}

	debug.Logger.Printf("Refreshing OAuth2 token")

	// Get token endpoint
	_, tokenEndpoint, err := o.getOAuth2Endpoints(o.config)
	if err != nil {
		return nil, fmt.Errorf("failed to get token endpoint: %w", err)
	}

	// Prepare and send refresh request
	req, err := o.prepareRefreshRequest(tokenEndpoint, token)
	if err != nil {
		return nil, err
	}

	statusCode, body, err := o.sendRefreshRequest(req)
	if err != nil {
		return nil, err
	}

	// Check response status - if not OK, re-authenticate
	if statusCode != http.StatusOK {
		debug.Logger.Printf("Token refresh failed with status %d, re-authenticating", statusCode)
		return o.Authenticate(ctx, o.config)
	}

	// Parse refresh response
	var tokenResponse map[string]interface{}
	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return nil, fmt.Errorf("failed to parse refresh response: %w", err)
	}

	// Extract token from response
	newToken, err := o.extractRefreshTokenResponse(token, tokenResponse)
	if err != nil {
		return nil, err
	}

	debug.Logger.Printf("Successfully refreshed OAuth2 token, expires at: %v", newToken.ExpiresAt)
	return newToken, nil
}

// prepareRefreshRequest prepares the refresh token request
func (o *OAuth2Authenticator) prepareRefreshRequest(tokenEndpoint string, token *Token) (*http.Request, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", token.RefreshToken)
	data.Set("client_id", o.config.GetString("client_id"))
	data.Set("client_secret", o.config.GetString("client_secret"))

	req, err := http.NewRequestWithContext(context.Background(), "POST", tokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "s9s/1.0")

	return req, nil
}

// sendRefreshRequest executes the refresh request and reads the response
func (o *OAuth2Authenticator) sendRefreshRequest(req *http.Request) (int, []byte, error) {
	resp, err := o.httpClient.Do(req)
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

// extractRefreshTokenResponse extracts token information from refresh response
func (o *OAuth2Authenticator) extractRefreshTokenResponse(oldToken *Token, response map[string]interface{}) (*Token, error) {
	accessToken, ok := response["access_token"].(string)
	if !ok || accessToken == "" {
		return nil, fmt.Errorf("missing or invalid access_token in refresh response")
	}

	// Use existing refresh token if new one not provided
	refreshToken := oldToken.RefreshToken
	if rt, ok := response["refresh_token"].(string); ok && rt != "" {
		refreshToken = rt
	}

	// Extract expiry with fallback
	var expiresAt time.Time
	if expiresIn, ok := response["expires_in"].(float64); ok {
		expiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	} else {
		expiresAt = time.Now().Add(1 * time.Hour)
	}

	return &Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    oldToken.TokenType,
		ExpiresAt:    expiresAt,
		Scopes:       oldToken.Scopes,
		ClusterID:    oldToken.ClusterID,
		Metadata:     oldToken.Metadata,
	}, nil
}

// ValidateToken validates an OAuth2 token
func (o *OAuth2Authenticator) ValidateToken(_ context.Context, token *Token) error {
	if token == nil {
		return fmt.Errorf("token is nil")
	}

	if token.AccessToken == "" {
		return fmt.Errorf("access token is empty")
	}

	if token.IsExpired() {
		return fmt.Errorf("token is expired")
	}

	debug.Logger.Printf("OAuth2 token is valid, expires in: %v", token.ExpiresIn())
	return nil
}

// RevokeToken revokes an OAuth2 token
func (o *OAuth2Authenticator) RevokeToken(_ context.Context, token *Token) error {
	// Try to discover revocation endpoint
	discoveryURL := o.config.GetString("discovery_url")
	var revocationEndpoint string

	if discoveryURL != "" {
		if discovery, err := o.getDiscoveryDocument(discoveryURL); err == nil {
			revocationEndpoint = discovery.RevocationEndpoint
		}
	}

	// Fall back to manual configuration
	if revocationEndpoint == "" {
		revocationEndpoint = o.config.GetString("revocation_endpoint")
	}

	if revocationEndpoint == "" {
		debug.Logger.Printf("No revocation endpoint available, token will expire naturally")
		return nil
	}

	debug.Logger.Printf("Revoking OAuth2 token")

	// Prepare revocation request
	data := url.Values{}
	data.Set("token", token.AccessToken)
	data.Set("client_id", o.config.GetString("client_id"))
	data.Set("client_secret", o.config.GetString("client_secret"))

	req, err := http.NewRequestWithContext(context.Background(), "POST", revocationEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create revocation request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "s9s/1.0")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("revocation request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token revocation failed with status %d", resp.StatusCode)
	}

	debug.Logger.Printf("Successfully revoked OAuth2 token")
	return nil
}

// getDiscoveryDocument fetches and parses OIDC discovery document
func (o *OAuth2Authenticator) getDiscoveryDocument(discoveryURL string) (*oidcDiscovery, error) {
	req, err := http.NewRequestWithContext(context.Background(), "GET", discoveryURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "s9s/1.0")

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discovery request failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var discovery oidcDiscovery
	if err := json.Unmarshal(body, &discovery); err != nil {
		return nil, err
	}

	return &discovery, nil
}

// Utility functions

// generateRandomString generates a cryptographically secure random string
func (o *OAuth2Authenticator) generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes)[:length], nil
}

// generateCodeChallenge generates PKCE code challenge from verifier
func (o *OAuth2Authenticator) generateCodeChallenge(verifier string) string {
	// For simplicity, we'll use plain method
	// In production, should use S256 method with SHA256 hash
	return verifier
}

// openBrowser opens the default browser with the given URL
func (o *OAuth2Authenticator) openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.CommandContext(context.Background(), "xdg-open", url)
	case "darwin":
		cmd = exec.CommandContext(context.Background(), "open", url)
	case "windows":
		cmd = exec.CommandContext(context.Background(), "rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	return cmd.Start()
}

// Cleanup performs any necessary cleanup
func (o *OAuth2Authenticator) Cleanup() error {
	o.stopCallbackServer()
	debug.Logger.Printf("OAuth2 authenticator cleanup completed")
	return nil
}
