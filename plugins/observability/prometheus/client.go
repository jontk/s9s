package prometheus

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

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

	// Create HTTP client with custom transport
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: config.TLSSkipVerify,
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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

// BatchQuery executes multiple queries in parallel
func (c *Client) BatchQuery(ctx context.Context, queries map[string]string, time time.Time) (map[string]*QueryResult, error) {
	results := make(map[string]*QueryResult)
	resultsChan := make(chan struct {
		name   string
		result *QueryResult
		err    error
	}, len(queries))

	// Execute queries in parallel
	for name, query := range queries {
		go func(n, q string) {
			result, err := c.Query(ctx, q, time)
			resultsChan <- struct {
				name   string
				result *QueryResult
				err    error
			}{name: n, result: result, err: err}
		}(name, query)
	}

	// Collect results
	var firstError error
	for i := 0; i < len(queries); i++ {
		r := <-resultsChan
		if r.err != nil && firstError == nil {
			firstError = r.err
		}
		if r.result != nil {
			results[r.name] = r.result
		}
	}

	if firstError != nil {
		return results, fmt.Errorf("batch query partially failed: %w", firstError)
	}

	return results, nil
}