package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jontk/s9s/internal/ui/components"
)

// WebhookConfig holds configuration for webhook notifications
type WebhookConfig struct {
	Enabled       bool              `json:"enabled"`
	URL           string            `json:"url"`
	MinAlertLevel int               `json:"min_alert_level"`
	Headers       map[string]string `json:"headers"`
	Timeout       int               `json:"timeout"` // Request timeout in seconds
	RetryCount    int               `json:"retry_count"`
}

// WebhookChannel implements webhook notifications
type WebhookChannel struct {
	config WebhookConfig
	client *http.Client
}

// WebhookPayload represents the payload sent to the webhook
type WebhookPayload struct {
	Timestamp   time.Time `json:"timestamp"`
	Level       string    `json:"level"`
	LevelInt    int       `json:"level_int"`
	Title       string    `json:"title"`
	Message     string    `json:"message"`
	Source      string    `json:"source"`
	AlertID     string    `json:"alert_id"`
	ClusterName string    `json:"cluster_name,omitempty"`
}

// NewWebhookChannel creates a new webhook notification channel
func NewWebhookChannel(config WebhookConfig) *WebhookChannel {
	// Set defaults
	if config.Timeout <= 0 {
		config.Timeout = 30
	}
	if config.RetryCount <= 0 {
		config.RetryCount = 3
	}

	return &WebhookChannel{
		config: config,
		client: &http.Client{
			Timeout: time.Duration(config.Timeout) * time.Second,
		},
	}
}

// Name returns the channel name
func (w *WebhookChannel) Name() string {
	return "webhook"
}

// IsEnabled returns whether the channel is enabled
func (w *WebhookChannel) IsEnabled() bool {
	return w.config.Enabled && w.config.URL != ""
}

// Notify sends a webhook notification
func (w *WebhookChannel) Notify(alert *components.Alert) error {
	// Check alert level threshold
	if int(alert.Level) < w.config.MinAlertLevel {
		return nil
	}

	// Create payload
	payload := WebhookPayload{
		Timestamp:   alert.Timestamp,
		Level:       getLevelString(alert.Level),
		LevelInt:    int(alert.Level),
		Title:       alert.Title,
		Message:     alert.Message,
		Source:      alert.Source,
		AlertID:     alert.ID,
		ClusterName: "S9S", // Could be made configurable
	}

	// Marshal payload
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	// Send with retries
	var lastError error
	for i := 0; i < w.config.RetryCount; i++ {
		if err := w.sendRequest(jsonData); err != nil {
			lastError = err
			// Wait before retry (exponential backoff)
			if i < w.config.RetryCount-1 {
				time.Sleep(time.Duration(i+1) * time.Second)
			}
		} else {
			return nil // Success
		}
	}

	return fmt.Errorf("webhook notification failed after %d retries: %w", w.config.RetryCount, lastError)
}

// sendRequest sends the actual HTTP request
func (w *WebhookChannel) sendRequest(jsonData []byte) error {
	req, err := http.NewRequest("POST", w.config.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "S9S/1.0")

	// Add custom headers
	for key, value := range w.config.Headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// Configure updates channel configuration
func (w *WebhookChannel) Configure(config map[string]interface{}) error {
	if enabled, ok := config["enabled"].(bool); ok {
		w.config.Enabled = enabled
	}

	if url, ok := config["url"].(string); ok {
		w.config.URL = url
	}

	if minLevel, ok := config["min_alert_level"].(int); ok {
		w.config.MinAlertLevel = minLevel
	}

	if headers, ok := config["headers"].(map[string]string); ok {
		w.config.Headers = headers
	}

	if timeout, ok := config["timeout"].(int); ok {
		w.config.Timeout = timeout
		w.client.Timeout = time.Duration(timeout) * time.Second
	}

	if retryCount, ok := config["retry_count"].(int); ok {
		w.config.RetryCount = retryCount
	}

	return nil
}
