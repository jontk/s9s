// Package security provides comprehensive security mechanisms for the observability plugin,
// including audit logging, rate limiting, request validation, and secrets management.
// This package implements defense-in-depth security patterns to protect against
// unauthorized access, malicious queries, and resource abuse.
package security

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AuditLogger handles audit logging for sensitive operations
type AuditLogger struct {
	config AuditConfig
	writer io.Writer
	file   *os.File
	mu     sync.Mutex
}

// AuditConfig contains audit logging configuration
type AuditConfig struct {
	// Enable audit logging
	Enabled bool `yaml:"enabled" json:"enabled"`
	
	// Log file path (if empty, logs to stdout)
	LogFile string `yaml:"logFile" json:"logFile"`
	
	// Log level: "info", "warn", "error"
	LogLevel string `yaml:"logLevel" json:"logLevel"`
	
	// Maximum log file size in MB before rotation
	MaxFileSizeMB int `yaml:"maxFileSizeMB" json:"maxFileSizeMB"`
	
	// Maximum number of log files to keep
	MaxFiles int `yaml:"maxFiles" json:"maxFiles"`
	
	// Include request/response bodies in logs (security risk)
	IncludeBodies bool `yaml:"includeBodies" json:"includeBodies"`
	
	// Log sensitive operations only (vs all operations)
	SensitiveOnly bool `yaml:"sensitiveOnly" json:"sensitiveOnly"`
	
	// Additional fields to log from request headers
	LogHeaders []string `yaml:"logHeaders" json:"logHeaders"`
}

// DefaultAuditConfig returns default audit configuration
func DefaultAuditConfig() AuditConfig {
	return AuditConfig{
		Enabled:       true,
		LogFile:       "/var/log/s9s-observability-audit.log",
		LogLevel:      "info",
		MaxFileSizeMB: 100,
		MaxFiles:      5,
		IncludeBodies: false, // Security: don't log bodies by default
		SensitiveOnly: true,  // Only log sensitive operations
		LogHeaders:    []string{"User-Agent", "X-Forwarded-For", "X-Real-IP"},
	}
}

// AuditEvent represents an auditable event
type AuditEvent struct {
	Timestamp    time.Time              `json:"timestamp"`
	EventType    string                 `json:"event_type"`
	UserID       string                 `json:"user_id,omitempty"`
	ClientIP     string                 `json:"client_ip"`
	Method       string                 `json:"method"`
	Path         string                 `json:"path"`
	Query        string                 `json:"query,omitempty"`
	StatusCode   int                    `json:"status_code"`
	Duration     time.Duration          `json:"duration"`
	Error        string                 `json:"error,omitempty"`
	Headers      map[string]string      `json:"headers,omitempty"`
	RequestBody  string                 `json:"request_body,omitempty"`
	ResponseBody string                 `json:"response_body,omitempty"`
	Sensitive    bool                   `json:"sensitive"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// EventType constants for different types of auditable events
const (
	EventTypeAPIAccess      = "api_access"
	EventTypeAuthentication = "authentication"
	EventTypeAuthorization  = "authorization"
	EventTypeSecretAccess   = "secret_access"
	EventTypeRateLimit      = "rate_limit"
	EventTypeValidation     = "validation"
	EventTypeError          = "error"
)

// NewAuditLogger creates a new audit logger
func NewAuditLogger(config AuditConfig) (*AuditLogger, error) {
	logger := &AuditLogger{
		config: config,
	}

	if !config.Enabled {
		logger.writer = io.Discard
		return logger, nil
	}

	// Set up writer
	if config.LogFile == "" {
		logger.writer = os.Stdout
	} else {
		// Ensure log directory exists
		logDir := filepath.Dir(config.LogFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		file, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open audit log file: %w", err)
		}
		logger.file = file
		logger.writer = file
	}

	return logger, nil
}

// LogEvent logs an audit event
func (al *AuditLogger) LogEvent(event AuditEvent) {
	if !al.config.Enabled {
		return
	}

	// Filter based on sensitivity settings
	if al.config.SensitiveOnly && !event.Sensitive {
		return
	}

	// Filter based on log level
	if !al.shouldLog(event) {
		return
	}

	al.mu.Lock()
	defer al.mu.Unlock()

	// Check file size and rotate if needed
	if al.file != nil {
		if err := al.rotateIfNeeded(); err != nil {
			// Log rotation failed, continue with current file
		}
	}

	// Serialize and write event
	eventData, err := json.Marshal(event)
	if err != nil {
		return // Skip event if serialization fails
	}

	al.writer.Write(eventData)
	al.writer.Write([]byte("\n"))
}

// LogAPIRequest logs an API request with response details
func (al *AuditLogger) LogAPIRequest(r *http.Request, statusCode int, duration time.Duration, err error) {
	event := AuditEvent{
		Timestamp:  time.Now(),
		EventType:  EventTypeAPIAccess,
		ClientIP:   extractClientIP(r),
		Method:     r.Method,
		Path:       r.URL.Path,
		Query:      r.URL.RawQuery,
		StatusCode: statusCode,
		Duration:   duration,
		Sensitive:  al.isSensitivePath(r.URL.Path),
		Headers:    al.extractHeaders(r),
	}

	if err != nil {
		event.Error = err.Error()
	}

	// Extract user ID from auth token if available
	if auth := r.Header.Get("Authorization"); auth != "" {
		event.UserID = al.extractUserID(auth)
	}

	// Include request body for sensitive operations if configured
	if al.config.IncludeBodies && event.Sensitive {
		if r.Body != nil {
			// Note: This would require buffering the body
			// In practice, you'd use middleware to capture this
		}
	}

	al.LogEvent(event)
}

// LogSecretAccess logs secret access operations
func (al *AuditLogger) LogSecretAccess(secretID, operation, userID string, success bool, err error) {
	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: EventTypeSecretAccess,
		UserID:    userID,
		Sensitive: true,
		Metadata: map[string]interface{}{
			"secret_id": secretID,
			"operation": operation,
			"success":   success,
		},
	}

	if err != nil {
		event.Error = err.Error()
	}

	al.LogEvent(event)
}

// LogAuthenticationAttempt logs authentication attempts
func (al *AuditLogger) LogAuthenticationAttempt(clientIP, userID string, success bool, err error) {
	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: EventTypeAuthentication,
		UserID:    userID,
		ClientIP:  clientIP,
		Sensitive: true,
		Metadata: map[string]interface{}{
			"success": success,
		},
	}

	if err != nil {
		event.Error = err.Error()
	}

	al.LogEvent(event)
}

// LogRateLimit logs rate limiting events
func (al *AuditLogger) LogRateLimit(clientIP, userID string, rateLimitType string) {
	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: EventTypeRateLimit,
		UserID:    userID,
		ClientIP:  clientIP,
		Sensitive: true,
		Metadata: map[string]interface{}{
			"rate_limit_type": rateLimitType,
		},
	}

	al.LogEvent(event)
}

// LogValidationFailure logs request validation failures
func (al *AuditLogger) LogValidationFailure(r *http.Request, validationError error) {
	event := AuditEvent{
		Timestamp: time.Now(),
		EventType: EventTypeValidation,
		ClientIP:  extractClientIP(r),
		Method:    r.Method,
		Path:      r.URL.Path,
		Query:     r.URL.RawQuery,
		Error:     validationError.Error(),
		Sensitive: true,
		Headers:   al.extractHeaders(r),
	}

	al.LogEvent(event)
}

// Close closes the audit logger and any open files
func (al *AuditLogger) Close() error {
	if al == nil {
		return nil
	}
	
	al.mu.Lock()
	defer al.mu.Unlock()

	if al.file != nil {
		return al.file.Close()
	}
	return nil
}

// Helper methods

func (al *AuditLogger) shouldLog(event AuditEvent) bool {
	switch al.config.LogLevel {
	case "error":
		return event.Error != ""
	case "warn":
		return event.Error != "" || event.StatusCode >= 400
	case "info":
		return true
	default:
		return true
	}
}

func (al *AuditLogger) isSensitivePath(path string) bool {
	sensitivePaths := []string{
		"/api/v1/subscriptions",
		"/api/v1/status",
		"/api/v1/analysis",
	}

	for _, sensitivePath := range sensitivePaths {
		if strings.Contains(path, sensitivePath) {
			return true
		}
	}

	// All secret-related paths are sensitive
	if strings.Contains(path, "secret") {
		return true
	}

	return false
}

func (al *AuditLogger) extractHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string)
	
	for _, headerName := range al.config.LogHeaders {
		if value := r.Header.Get(headerName); value != "" {
			headers[headerName] = value
		}
	}
	
	return headers
}

func (al *AuditLogger) extractUserID(authHeader string) string {
	// Extract user ID from bearer token
	// This is a simplified implementation
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		// In practice, you'd decode the JWT or look up the token
		return fmt.Sprintf("token:%s", token[:min(8, len(token))])
	}
	return "unknown"
}

func extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Use first IP in X-Forwarded-For chain
		if ips := strings.Split(xff, ","); len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}
	
	// Check X-Real-IP header
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}
	
	// Fallback to RemoteAddr (remove port)
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	
	return r.RemoteAddr
}

func (al *AuditLogger) rotateIfNeeded() error {
	if al.file == nil || al.config.MaxFileSizeMB <= 0 {
		return nil
	}

	// Get current file size
	stat, err := al.file.Stat()
	if err != nil {
		return err
	}

	maxBytes := int64(al.config.MaxFileSizeMB) * 1024 * 1024
	if stat.Size() < maxBytes {
		return nil // No rotation needed
	}

	// Close current file
	al.file.Close()

	// Rotate files
	if err := al.rotateFiles(); err != nil {
		return err
	}

	// Open new file
	file, err := os.OpenFile(al.config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	al.file = file
	al.writer = file
	return nil
}

func (al *AuditLogger) rotateFiles() error {
	baseName := al.config.LogFile
	
	// Remove oldest file if we've reached the limit
	oldestFile := fmt.Sprintf("%s.%d", baseName, al.config.MaxFiles)
	os.Remove(oldestFile) // Ignore error if file doesn't exist

	// Shift all files
	for i := al.config.MaxFiles - 1; i >= 1; i-- {
		oldFile := fmt.Sprintf("%s.%d", baseName, i)
		newFile := fmt.Sprintf("%s.%d", baseName, i+1)
		os.Rename(oldFile, newFile) // Ignore errors
	}

	// Move current file to .1
	return os.Rename(baseName, baseName+".1")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetStats returns audit logging statistics
func (al *AuditLogger) GetStats() map[string]interface{} {
	al.mu.Lock()
	defer al.mu.Unlock()

	stats := map[string]interface{}{
		"enabled":         al.config.Enabled,
		"log_file":        al.config.LogFile,
		"log_level":       al.config.LogLevel,
		"sensitive_only":  al.config.SensitiveOnly,
		"include_bodies":  al.config.IncludeBodies,
		"max_file_size":   al.config.MaxFileSizeMB,
		"max_files":       al.config.MaxFiles,
	}

	// Add file statistics if logging to file
	if al.file != nil {
		if stat, err := al.file.Stat(); err == nil {
			stats["current_file_size"] = stat.Size()
			stats["last_modified"] = stat.ModTime()
		}
	}

	return stats
}

// AuditMiddleware creates HTTP middleware for audit logging
func AuditMiddleware(logger *AuditLogger) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if logger == nil || !logger.config.Enabled {
				next(w, r)
				return
			}

			start := time.Now()

			// Create response recorder to capture status code
			recorder := &responseRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Call next handler
			var err error
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("handler panic: %v", r)
					recorder.statusCode = http.StatusInternalServerError
					panic(r) // Re-panic after logging
				}
			}()

			next(recorder, r)

			// Log the request
			duration := time.Since(start)
			logger.LogAPIRequest(r, recorder.statusCode, duration, err)
		}
	}
}

// responseRecorder captures response details for audit logging
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.statusCode = code
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(data []byte) (int, error) {
	rr.body = append(rr.body, data...)
	return rr.ResponseWriter.Write(data)
}