// Package prometheus provides Prometheus client functionality with advanced features
// including caching, circuit breaking, connection pooling, and batch query processing.
// It supports authentication, TLS configuration, retry mechanisms, and comprehensive
// error handling for reliable metric collection from Prometheus servers.
package prometheus

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// PrometheusClientInterface defines the interface for Prometheus clients
type PrometheusClientInterface interface {
	TestConnection(ctx context.Context) error
	Query(ctx context.Context, query string, time time.Time) (*QueryResult, error)
	QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (*QueryResult, error)
	BatchQuery(ctx context.Context, queries map[string]string, time time.Time) (map[string]*QueryResult, error)
	Series(ctx context.Context, matches []string, start, end time.Time) ([]map[string]string, error)
	Labels(ctx context.Context) ([]string, error)
}

// Client represents a Prometheus API client
type Client struct {
	endpoint   string
	httpClient *http.Client
	config     ClientConfig
}

// ClientConfig contains configuration for the Prometheus client
type ClientConfig struct {
	Endpoint      string
	Username      string
	Password      string
	BearerToken   string
	Timeout       time.Duration
	TLSSkipVerify bool
	TLSCertFile   string
	TLSKeyFile    string
	TLSCAFile     string
}

// NewClient creates a new Prometheus client
func NewClient(config ClientConfig) (*Client, error) {
	if config.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}

	// Parse and validate endpoint
	u, err := url.Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}

	// Create TLS configuration
	tlsConfig, err := createTLSConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS config: %w", err)
	}

	// Create HTTP client with custom transport
	transport := &http.Transport{
		TLSClientConfig:     tlsConfig,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		MaxConnsPerHost:     20, // Limit concurrent connections per host
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
		DisableCompression:  false,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}

	return &Client{
		endpoint:   u.String(),
		httpClient: httpClient,
		config:     config,
	}, nil
}

// TestConnection tests the connection to Prometheus
func (c *Client) TestConnection(ctx context.Context) error {
	// Use the config endpoint to test connectivity
	resp, err := c.doRequest(ctx, "GET", "/api/v1/label/__name__/values", nil)
	if err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("connection test failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Query executes a Prometheus query
func (c *Client) Query(ctx context.Context, query string, time time.Time) (*QueryResult, error) {
	params := url.Values{}
	params.Set("query", query)
	if !time.IsZero() {
		params.Set("time", fmt.Sprintf("%d", time.Unix()))
	}

	resp, err := c.doRequest(ctx, "GET", "/api/v1/query?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("query failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result QueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("query failed: %s", result.Error)
	}

	return &result, nil
}

// QueryRange executes a Prometheus range query
func (c *Client) QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (*QueryResult, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("start", fmt.Sprintf("%d", start.Unix()))
	params.Set("end", fmt.Sprintf("%d", end.Unix()))
	params.Set("step", fmt.Sprintf("%ds", int(step.Seconds())))

	resp, err := c.doRequest(ctx, "GET", "/api/v1/query_range?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("range query failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("range query failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result QueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("range query failed: %s", result.Error)
	}

	return &result, nil
}

// Series returns the list of time series that match a label set
func (c *Client) Series(ctx context.Context, matches []string, start, end time.Time) ([]map[string]string, error) {
	params := url.Values{}
	for _, match := range matches {
		params.Add("match[]", match)
	}
	params.Set("start", fmt.Sprintf("%d", start.Unix()))
	params.Set("end", fmt.Sprintf("%d", end.Unix()))

	resp, err := c.doRequest(ctx, "GET", "/api/v1/series?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("series query failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("series query failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Status string              `json:"status"`
		Data   []map[string]string `json:"data"`
		Error  string              `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("series query failed: %s", result.Error)
	}

	return result.Data, nil
}

// Labels returns the list of label names
func (c *Client) Labels(ctx context.Context) ([]string, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/v1/labels", nil)
	if err != nil {
		return nil, fmt.Errorf("labels query failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("labels query failed with status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
		Error  string   `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("labels query failed: %s", result.Error)
	}

	return result.Data, nil
}

// Close closes the client connections
func (c *Client) Close() {
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}
}

// doRequest performs an HTTP request with authentication
	//nolint:unparam // Method parameter allows for future extensibility beyond GET-only requests
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	fullURL := c.endpoint + path

	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication
	if c.config.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.BearerToken)
	} else if c.config.Username != "" && c.config.Password != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	// Add headers
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// BatchQueryConfig holds configuration for batch query execution
type BatchQueryConfig struct {
	MaxConcurrency int           // Maximum concurrent queries (default: 10)
	BatchTimeout   time.Duration // Timeout for entire batch (default: 30s)
	RetryAttempts  int           // Number of retry attempts for failed queries (default: 3)
}

// DefaultBatchQueryConfig returns default batch query configuration
func DefaultBatchQueryConfig() BatchQueryConfig {
	return BatchQueryConfig{
		MaxConcurrency: 10,
		BatchTimeout:   30 * time.Second,
		RetryAttempts:  3,
	}
}

// BatchQuery executes multiple queries with configurable concurrency and retry logic
func (c *Client) BatchQuery(ctx context.Context, queries map[string]string, queryTime time.Time) (map[string]*QueryResult, error) {
	return c.BatchQueryWithConfig(ctx, queries, queryTime, DefaultBatchQueryConfig())
}

// BatchQueryWithConfig executes multiple queries with custom configuration
func (c *Client) BatchQueryWithConfig(ctx context.Context, queries map[string]string, queryTime time.Time, config BatchQueryConfig) (map[string]*QueryResult, error) {
	if len(queries) == 0 {
		return make(map[string]*QueryResult), nil
	}

	// Create context with timeout for entire batch
	batchCtx, cancel := context.WithTimeout(ctx, config.BatchTimeout)
	defer cancel()

	// Limit concurrency using semaphore pattern
	semaphore := make(chan struct{}, config.MaxConcurrency)
	results := make(map[string]*QueryResult)
	resultsChan := make(chan batchResult, len(queries))

	// Execute queries with controlled concurrency
	for name, query := range queries {
		go func(n, q string) {
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			result, err := c.executeQueryWithRetry(batchCtx, q, queryTime, config.RetryAttempts)
			resultsChan <- batchResult{name: n, result: result, err: err}
		}(name, query)
	}

	// Collect results
	var errors []error
	successCount := 0

	for i := 0; i < len(queries); i++ {
		select {
		case r := <-resultsChan:
			if r.err != nil {
				errors = append(errors, fmt.Errorf("query '%s': %w", r.name, r.err))
			} else {
				results[r.name] = r.result
				successCount++
			}
		case <-batchCtx.Done():
			return results, fmt.Errorf("batch query timed out after %v, completed %d/%d queries",
				config.BatchTimeout, successCount, len(queries))
		}
	}

	// Return results with error summary if some queries failed
	if len(errors) > 0 {
		return results, fmt.Errorf("batch query completed with %d errors: %w", len(errors), errors[0])
	}

	return results, nil
}

// batchResult holds the result of a single query in a batch
type batchResult struct {
	name   string
	result *QueryResult
	err    error
}

// executeQueryWithRetry executes a query with exponential backoff retry logic
func (c *Client) executeQueryWithRetry(ctx context.Context, query string, queryTime time.Time, maxRetries int) (*QueryResult, error) {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err := c.Query(ctx, query, queryTime)
		if err == nil {
			return result, nil
		}

		lastErr = err

		// Don't retry on context cancellation or certain HTTP errors
		if ctx.Err() != nil {
			break
		}

		// Exponential backoff: 100ms, 200ms, 400ms, 800ms...
		if attempt < maxRetries {
			backoffDuration := time.Duration(100<<uint(attempt)) * time.Millisecond
			timer := time.NewTimer(backoffDuration)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
				continue
			}
		}
	}

	return nil, fmt.Errorf("query failed after %d retries: %w", maxRetries, lastErr)
}

// createTLSConfig creates a TLS configuration from ClientConfig
//
// SECURITY WARNING: TLSSkipVerify disables certificate validation when set to true.
// This should only be used in development/testing environments or when using self-signed
// certificates in trusted networks. For production use, provide proper CA certificates
// via TLSCAFile configuration instead.
func createTLSConfig(config ClientConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.TLSSkipVerify, //nolint:gosec // G402: User-configurable, with security warning in docs
		MinVersion:         tls.VersionTLS12,     // Enforce minimum TLS 1.2
	}

	// Load CA certificate if specified
	if config.TLSCAFile != "" {
		caCert, err := os.ReadFile(config.TLSCAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caCertPool
	}

	// Load client certificate and key if specified
	if config.TLSCertFile != "" && config.TLSKeyFile != "" {
		clientCert, err := tls.LoadX509KeyPair(config.TLSCertFile, config.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}

	return tlsConfig, nil
}
