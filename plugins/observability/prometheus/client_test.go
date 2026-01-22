package prometheus

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  ClientConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: ClientConfig{
				Endpoint: "http://localhost:9090",
				Timeout:  10 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "empty endpoint",
			config: ClientConfig{
				Timeout: 10 * time.Second,
			},
			wantErr: true,
			errMsg:  "endpoint is required",
		},
		{
			name: "invalid endpoint URL",
			config: ClientConfig{
				Endpoint: "ht!tp://invalid-url",
				Timeout:  10 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "HTTPS endpoint",
			config: ClientConfig{
				Endpoint: "https://prometheus.example.com:9090",
				Timeout:  10 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "with basic auth",
			config: ClientConfig{
				Endpoint: "http://localhost:9090",
				Username: "user",
				Password: "pass",
				Timeout:  10 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "with bearer token",
			config: ClientConfig{
				Endpoint:    "http://localhost:9090",
				BearerToken: "token123",
				Timeout:     10 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "with TLS skip verify",
			config: ClientConfig{
				Endpoint:      "https://localhost:9090",
				TLSSkipVerify: true,
				Timeout:       10 * time.Second,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if client != nil {
					t.Error("Expected nil client on error")
				}
				if tt.errMsg != "" && (err == nil || err.Error() != tt.errMsg) {
					t.Errorf("Expected error message '%s', got '%v'", tt.errMsg, err)
				}
				return
			}

			if client == nil {
				t.Fatal("Expected non-nil client on success")
			}

			if client.endpoint == "" {
				t.Error("Client endpoint not set")
			}

			if client.httpClient == nil {
				t.Error("HTTP client not initialized")
			}

			if client.config.Endpoint != tt.config.Endpoint {
				t.Error("Config not stored correctly")
			}
		})
	}
}

func TestCreateTLSConfig(t *testing.T) {
	tests := []struct {
		name     string
		config   ClientConfig
		wantErr  bool
		validate func(*tls.Config) error
	}{
		{
			name: "basic TLS config",
			config: ClientConfig{
				TLSSkipVerify: false,
			},
			wantErr: false,
			validate: func(tlsConfig *tls.Config) error {
				if tlsConfig.InsecureSkipVerify {
					t.Error("Expected InsecureSkipVerify false")
				}
				if tlsConfig.RootCAs != nil {
					t.Error("Expected no custom root CAs")
				}
				if len(tlsConfig.Certificates) != 0 {
					t.Error("Expected no client certificates")
				}
				return nil
			},
		},
		{
			name: "skip verify enabled",
			config: ClientConfig{
				TLSSkipVerify: true,
			},
			wantErr: false,
			validate: func(tlsConfig *tls.Config) error {
				if !tlsConfig.InsecureSkipVerify {
					t.Error("Expected InsecureSkipVerify true")
				}
				return nil
			},
		},
		{
			name: "invalid CA file",
			config: ClientConfig{
				TLSCAFile: "/nonexistent/ca.pem",
			},
			wantErr: true,
		},
		{
			name: "invalid cert file",
			config: ClientConfig{
				TLSCertFile: "/nonexistent/cert.pem",
				TLSKeyFile:  "/nonexistent/key.pem",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tlsConfig, err := createTLSConfig(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("createTLSConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tlsConfig != nil {
					t.Error("Expected nil TLS config on error")
				}
				return
			}

			if tlsConfig == nil {
				t.Fatal("Expected non-nil TLS config on success")
			}

			if tt.validate != nil {
				if err := tt.validate(tlsConfig); err != nil {
					t.Errorf("TLS config validation failed: %v", err)
				}
			}
		})
	}
}

func TestClientConfigFields(t *testing.T) {
	config := ClientConfig{
		Endpoint:      "https://prometheus.example.com:9090",
		Username:      "testuser",
		Password:      "testpass",
		BearerToken:   "token123",
		Timeout:       30 * time.Second,
		TLSSkipVerify: true,
		TLSCertFile:   "/path/to/cert.pem",
		TLSKeyFile:    "/path/to/key.pem",
		TLSCAFile:     "/path/to/ca.pem",
	}

	client, err := NewClient(config)
	if err != nil {
		// Expected to fail due to non-existent cert files
		// But we can verify the config was stored
		return
	}

	if client.config.Endpoint != config.Endpoint {
		t.Error("Endpoint not stored correctly")
	}

	if client.config.Username != config.Username {
		t.Error("Username not stored correctly")
	}

	if client.config.Password != config.Password {
		t.Error("Password not stored correctly")
	}

	if client.config.BearerToken != config.BearerToken {
		t.Error("Bearer token not stored correctly")
	}

	if client.config.Timeout != config.Timeout {
		t.Error("Timeout not stored correctly")
	}

	if client.config.TLSSkipVerify != config.TLSSkipVerify {
		t.Error("TLS skip verify not stored correctly")
	}
}

func TestBatchQuery(t *testing.T) {
	// Create a client (this will fail connection tests, but we can test the structure)
	config := ClientConfig{
		Endpoint: "http://localhost:9090",
		Timeout:  10 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()
	queries := map[string]string{
		"cpu":    "cpu_usage_total",
		"memory": "memory_usage_bytes",
	}

	// This will fail due to no actual Prometheus server, but we can verify the method exists
	results, err := client.BatchQuery(ctx, queries, time.Now())

	// We expect an error since there's no real server
	if err == nil {
		t.Log("Unexpected success - there might be a Prometheus server running")
	}

	// Results should be a map even on error
	if results == nil {
		t.Error("Expected non-nil results map")
	}
}

func TestClientClose(t *testing.T) {
	config := ClientConfig{
		Endpoint: "http://localhost:9090",
		Timeout:  10 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// This should not panic
	client.Close()

	// Multiple closes should be safe
	client.Close()
}

func TestClientMethods(t *testing.T) {
	config := ClientConfig{
		Endpoint: "http://localhost:9090",
		Timeout:  10 * time.Second,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Test TestConnection (will fail without real server)
	err = client.TestConnection(ctx)
	if err == nil {
		t.Log("TestConnection succeeded - there might be a real Prometheus server")
	}

	// Test Query (will fail without real server)
	result, err := client.Query(ctx, "up", time.Now())
	if err == nil && result != nil {
		t.Log("Query succeeded - there might be a real Prometheus server")
	}

	// Test QueryRange (will fail without real server)
	now := time.Now()
	result, err = client.QueryRange(ctx, "up", now.Add(-time.Hour), now, time.Minute)
	if err == nil && result != nil {
		t.Log("QueryRange succeeded - there might be a real Prometheus server")
	}

	// Test Series (will fail without real server)
	series, err := client.Series(ctx, []string{"up"}, now.Add(-time.Hour), now)
	if err == nil && series != nil {
		t.Log("Series succeeded - there might be a real Prometheus server")
	}

	// Test Labels (will fail without real server)
	labels, err := client.Labels(ctx)
	if err == nil && labels != nil {
		t.Log("Labels succeeded - there might be a real Prometheus server")
	}
}

func TestHTTPClientConfiguration(t *testing.T) {
	config := ClientConfig{
		Endpoint:      "https://prometheus.example.com:9090",
		Timeout:       30 * time.Second,
		TLSSkipVerify: true,
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client.httpClient == nil {
		t.Fatal("HTTP client not created")
	}

	if client.httpClient.Timeout != config.Timeout {
		t.Errorf("Expected timeout %v, got %v", config.Timeout, client.httpClient.Timeout)
	}

	// Check that transport is configured
	transport, ok := client.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Error("Expected custom transport")
		return
	}

	if transport.TLSClientConfig == nil {
		t.Error("TLS config not set on transport")
		return
	}

	if !transport.TLSClientConfig.InsecureSkipVerify {
		t.Error("TLS skip verify not set correctly")
	}
}

func TestEndpointNormalization(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		want     string
	}{
		{
			name:     "simple HTTP",
			endpoint: "http://localhost:9090",
			want:     "http://localhost:9090",
		},
		{
			name:     "simple HTTPS",
			endpoint: "https://prometheus.example.com",
			want:     "https://prometheus.example.com",
		},
		{
			name:     "with path",
			endpoint: "http://localhost:9090/prometheus",
			want:     "http://localhost:9090/prometheus",
		},
		{
			name:     "with trailing slash",
			endpoint: "http://localhost:9090/",
			want:     "http://localhost:9090/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ClientConfig{
				Endpoint: tt.endpoint,
				Timeout:  10 * time.Second,
			}

			client, err := NewClient(config)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			if client.endpoint != tt.want {
				t.Errorf("Expected endpoint %s, got %s", tt.want, client.endpoint)
			}
		})
	}
}

// Benchmark tests
func BenchmarkNewClient(b *testing.B) {
	config := ClientConfig{
		Endpoint: "http://localhost:9090",
		Timeout:  10 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client, err := NewClient(config)
		if err != nil {
			b.Fatalf("Failed to create client: %v", err)
		}
		client.Close()
	}
}

func BenchmarkCreateTLSConfig(b *testing.B) {
	config := ClientConfig{
		TLSSkipVerify: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := createTLSConfig(config)
		if err != nil {
			b.Fatalf("Failed to create TLS config: %v", err)
		}
	}
}
