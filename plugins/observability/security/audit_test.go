package security

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewAuditLogger(t *testing.T) {
	config := DefaultAuditConfig()
	config.LogFile = "" // Use stdout for test

	logger, err := NewAuditLogger(&config)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}

	if logger == nil {
		t.Fatal("NewAuditLogger returned nil")
	}

	if logger.writer == nil {
		t.Error("Expected writer to be initialized")
	}

	_ = logger.Close()
}

func TestNewAuditLoggerWithFile(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "audit.log")

	config := AuditConfig{
		Enabled: true,
		LogFile: logFile,
	}

	logger, err := NewAuditLogger(&config)
	if err != nil {
		t.Fatalf("NewAuditLogger with file failed: %v", err)
	}

	defer func() { _ = logger.Close() }()

	if logger.file == nil {
		t.Error("Expected file to be opened")
	}

	// Verify file was created
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		t.Error("Expected log file to be created")
	}
}

func TestNewAuditLoggerDisabled(t *testing.T) {
	config := AuditConfig{
		Enabled: false,
	}

	logger, err := NewAuditLogger(&config)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}

	// Should still initialize but with discard writer
	if logger == nil {
		t.Fatal("NewAuditLogger returned nil")
	}

	_ = logger.Close()
}

func TestLogEvent(t *testing.T) {
	var buffer bytes.Buffer

	config := DefaultAuditConfig()
	config.LogFile = "" // Use stdout

	logger, err := NewAuditLogger(&config)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Override writer to capture output
	logger.writer = &buffer

	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: EventTypeAPIAccess,
		ClientIP:  "127.0.0.1",
		Method:    "GET",
		Path:      "/api/v1/metrics/query",
		Sensitive: true,
	}

	logger.LogEvent(&event)

	// Verify event was logged
	output := buffer.String()
	if output == "" {
		t.Error("Expected event to be logged")
	}

	// Verify JSON format
	var loggedEvent AuditEvent
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &loggedEvent); err != nil {
		t.Errorf("Logged event is not valid JSON: %v", err)
	}

	if loggedEvent.EventType != EventTypeAPIAccess {
		t.Errorf("Expected event type %s, got %s", EventTypeAPIAccess, loggedEvent.EventType)
	}
}

func TestLogEventSensitiveFiltering(t *testing.T) {
	var buffer bytes.Buffer

	config := AuditConfig{
		Enabled:       true,
		SensitiveOnly: true,
		LogLevel:      "info",
	}

	logger, err := NewAuditLogger(&config)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	logger.writer = &buffer

	// Log non-sensitive event
	nonSensitiveEvent := AuditEvent{
		Timestamp: time.Now(),
		EventType: EventTypeAPIAccess,
		Sensitive: false,
	}
	logger.LogEvent(&nonSensitiveEvent)

	// Should not be logged
	if buffer.Len() > 0 {
		t.Error("Non-sensitive event should not be logged when SensitiveOnly is true")
	}

	// Log sensitive event
	sensitiveEvent := AuditEvent{
		Timestamp: time.Now(),
		EventType: EventTypeSecretAccess,
		Sensitive: true,
	}
	logger.LogEvent(&sensitiveEvent)

	// Should be logged
	if buffer.Len() == 0 {
		t.Error("Sensitive event should be logged")
	}
}

func TestLogAPIRequest(t *testing.T) {
	var buffer bytes.Buffer

	config := DefaultAuditConfig()
	config.LogFile = ""
	config.SensitiveOnly = false // Log all events for this test
	config.LogHeaders = []string{"User-Agent", "X-Test-Header"}

	logger, err := NewAuditLogger(&config)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	logger.writer = &buffer

	// Create test request
	req := httptest.NewRequest("GET", "/api/v1/metrics/query?query=cpu_usage", nil)
	req.Header.Set("User-Agent", "TestClient/1.0")
	req.Header.Set("X-Test-Header", "test-value")
	req.Header.Set("Authorization", "Bearer test-token")
	req.RemoteAddr = "127.0.0.1:12345"

	logger.LogAPIRequest(req, http.StatusOK, 150*time.Millisecond, nil)

	// Verify event was logged
	output := buffer.String()
	if output == "" {
		t.Error("Expected API request to be logged")
	}

	var loggedEvent AuditEvent
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &loggedEvent); err != nil {
		t.Errorf("Logged event is not valid JSON: %v", err)
	}

	// Verify event details
	if loggedEvent.EventType != EventTypeAPIAccess {
		t.Errorf("Expected event type %s, got %s", EventTypeAPIAccess, loggedEvent.EventType)
	}

	if loggedEvent.Method != "GET" {
		t.Errorf("Expected method GET, got %s", loggedEvent.Method)
	}

	if loggedEvent.ClientIP != "127.0.0.1" {
		t.Errorf("Expected client IP 127.0.0.1, got %s", loggedEvent.ClientIP)
	}

	if loggedEvent.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, loggedEvent.StatusCode)
	}

	if loggedEvent.Duration != 150*time.Millisecond {
		t.Errorf("Expected duration 150ms, got %s", loggedEvent.Duration)
	}

	// Check headers were captured
	if userAgent, exists := loggedEvent.Headers["User-Agent"]; !exists || userAgent != "TestClient/1.0" {
		t.Errorf("Expected User-Agent header to be captured")
	}

	if testHeader, exists := loggedEvent.Headers["X-Test-Header"]; !exists || testHeader != "test-value" {
		t.Errorf("Expected X-Test-Header to be captured")
	}
}

func TestLogSecretAccess(t *testing.T) {
	var buffer bytes.Buffer

	config := DefaultAuditConfig()
	config.LogFile = ""

	logger, err := NewAuditLogger(&config)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	logger.writer = &buffer

	logger.LogSecretAccess("secret-123", "read", "user-456", true, nil)

	// Verify event was logged
	output := buffer.String()
	if output == "" {
		t.Error("Expected secret access to be logged")
	}

	var loggedEvent AuditEvent
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &loggedEvent); err != nil {
		t.Errorf("Logged event is not valid JSON: %v", err)
	}

	if loggedEvent.EventType != EventTypeSecretAccess {
		t.Errorf("Expected event type %s, got %s", EventTypeSecretAccess, loggedEvent.EventType)
	}

	if loggedEvent.UserID != "user-456" {
		t.Errorf("Expected user ID 'user-456', got '%s'", loggedEvent.UserID)
	}

	if !loggedEvent.Sensitive {
		t.Error("Secret access should be marked as sensitive")
	}

	// Check metadata
	if metadata, ok := loggedEvent.Metadata["secret_id"].(string); !ok || metadata != "secret-123" {
		t.Error("Expected secret_id in metadata")
	}

	if operation, ok := loggedEvent.Metadata["operation"].(string); !ok || operation != "read" {
		t.Error("Expected operation in metadata")
	}
}

func TestAuditMiddleware(t *testing.T) {
	var buffer bytes.Buffer

	config := AuditConfig{
		Enabled:       true,
		SensitiveOnly: false, // Log all events for test
		LogLevel:      "info",
	}

	logger, err := NewAuditLogger(&config)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	logger.writer = &buffer

	// Create test handler
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("success"))
	}

	// Wrap with audit middleware
	middleware := AuditMiddleware(logger)
	wrappedHandler := middleware(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/api/v1/metrics/query", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()

	// Execute request
	wrappedHandler(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}

	// Verify audit log
	output := buffer.String()
	if output == "" {
		t.Error("Expected request to be audited")
	}

	var loggedEvent AuditEvent
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &loggedEvent); err != nil {
		t.Errorf("Logged event is not valid JSON: %v", err)
	}

	if loggedEvent.Path != "/api/v1/metrics/query" {
		t.Errorf("Expected path '/api/v1/metrics/query', got '%s'", loggedEvent.Path)
	}
}

func TestAuditMiddlewareDisabled(t *testing.T) {
	config := AuditConfig{
		Enabled: false,
	}

	logger, err := NewAuditLogger(&config)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	// Create test handler
	handler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	// Wrap with audit middleware
	middleware := AuditMiddleware(logger)
	wrappedHandler := middleware(handler)

	// Execute request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	wrappedHandler(w, req)

	// Should complete normally even with disabled audit logging
	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK even with disabled audit, got %d", w.Code)
	}
}

func TestExtractClientIPFunction(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		headers    map[string]string
		expected   string
	}{
		{
			name:       "RemoteAddr only",
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For single IP",
			remoteAddr: "127.0.0.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1"},
			expected:   "10.0.0.1",
		},
		{
			name:       "X-Forwarded-For multiple IPs",
			remoteAddr: "127.0.0.1:12345",
			headers:    map[string]string{"X-Forwarded-For": "10.0.0.1, 192.168.1.1"},
			expected:   "10.0.0.1",
		},
		{
			name:       "X-Real-IP header",
			remoteAddr: "127.0.0.1:12345",
			headers:    map[string]string{"X-Real-IP": "203.0.113.1"},
			expected:   "203.0.113.1",
		},
		{
			name:       "X-Forwarded-For takes precedence over X-Real-IP",
			remoteAddr: "127.0.0.1:12345",
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1",
				"X-Real-IP":       "203.0.113.1",
			},
			expected: "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr

			for header, value := range tt.headers {
				req.Header.Set(header, value)
			}

			clientIP := extractClientIP(req)
			if clientIP != tt.expected {
				t.Errorf("Expected client IP '%s', got '%s'", tt.expected, clientIP)
			}
		})
	}
}

func TestIsSensitivePath(t *testing.T) {
	config := DefaultAuditConfig()
	config.LogFile = "" // Use stdout for test
	logger, err := NewAuditLogger(&config)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	tests := []struct {
		path      string
		sensitive bool
	}{
		{"/api/v1/metrics/query", false},
		{"/api/v1/subscriptions", true},
		{"/api/v1/status", true},
		{"/api/v1/analysis/trend", true},
		{"/api/v1/secret/get", true},
		{"/health", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := logger.isSensitivePath(tt.path)
			if result != tt.sensitive {
				t.Errorf("Path '%s' sensitivity: expected %v, got %v", tt.path, tt.sensitive, result)
			}
		})
	}
}

func TestLogLevelFiltering(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  string
		event     AuditEvent
		shouldLog bool
	}{
		{
			name:      "info level logs everything",
			logLevel:  "info",
			event:     AuditEvent{StatusCode: 200},
			shouldLog: true,
		},
		{
			name:      "warn level logs 4xx errors",
			logLevel:  "warn",
			event:     AuditEvent{StatusCode: 400},
			shouldLog: true,
		},
		{
			name:      "warn level skips 2xx",
			logLevel:  "warn",
			event:     AuditEvent{StatusCode: 200},
			shouldLog: false,
		},
		{
			name:      "error level logs only errors",
			logLevel:  "error",
			event:     AuditEvent{Error: "test error"},
			shouldLog: true,
		},
		{
			name:      "error level skips success",
			logLevel:  "error",
			event:     AuditEvent{StatusCode: 200},
			shouldLog: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := AuditConfig{
				Enabled:       true,
				LogLevel:      tt.logLevel,
				SensitiveOnly: false,
			}

			logger, _ := NewAuditLogger(&config)
			defer func() { _ = logger.Close() }()

			result := logger.shouldLog(&tt.event)
			if result != tt.shouldLog {
				t.Errorf("Log level '%s' with event: expected shouldLog=%v, got %v",
					tt.logLevel, tt.shouldLog, result)
			}
		})
	}
}

func TestExtractUserID(t *testing.T) {
	config := DefaultAuditConfig()
	logger, _ := NewAuditLogger(&config)
	defer func() { _ = logger.Close() }()

	tests := []struct {
		name       string
		authHeader string
		expected   string
	}{
		{
			name:       "Bearer token",
			authHeader: "Bearer abcd1234567890",
			expected:   "token:abcd1234",
		},
		{
			name:       "Short bearer token",
			authHeader: "Bearer abc",
			expected:   "token:abc",
		},
		{
			name:       "No bearer prefix",
			authHeader: "Basic dXNlcjpwYXNz",
			expected:   "unknown",
		},
		{
			name:       "Empty header",
			authHeader: "",
			expected:   "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := logger.extractUserID(tt.authHeader)
			if result != tt.expected {
				t.Errorf("Expected user ID '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestAuditGetStats(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "audit.log")

	config := AuditConfig{
		Enabled:       true,
		LogFile:       logFile,
		LogLevel:      "info",
		MaxFileSizeMB: 50,
		MaxFiles:      3,
		SensitiveOnly: true,
		IncludeBodies: false,
	}

	logger, err := NewAuditLogger(&config)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}
	defer func() { _ = logger.Close() }()

	stats := logger.GetStats()

	expectedFields := []string{
		"enabled",
		"log_file",
		"log_level",
		"sensitive_only",
		"include_bodies",
		"max_file_size",
		"max_files",
	}

	for _, field := range expectedFields {
		if _, exists := stats[field]; !exists {
			t.Errorf("Missing expected field in stats: %s", field)
		}
	}

	if enabled, ok := stats["enabled"].(bool); !ok || !enabled {
		t.Error("Expected enabled to be true")
	}

	if logLevel, ok := stats["log_level"].(string); !ok || logLevel != "info" {
		t.Errorf("Expected log_level 'info', got '%v'", stats["log_level"])
	}
}

func TestResponseRecorder(t *testing.T) {
	w := httptest.NewRecorder()
	recorder := &responseRecorder{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	// Test WriteHeader
	recorder.WriteHeader(http.StatusCreated)
	if recorder.statusCode != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, recorder.statusCode)
	}

	// Test Write
	data := []byte("test response")
	n, err := recorder.Write(data)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	if !bytes.Equal(recorder.body, data) {
		t.Errorf("Expected body '%s', got '%s'", string(data), string(recorder.body))
	}
}
