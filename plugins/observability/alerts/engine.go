// Package alerts provides real-time alerting and notification capabilities
// for monitoring system metrics and resource utilization. It supports configurable
// alert rules, multiple severity levels, notification channels, and alert history
// tracking with automatic cleanup and retention management.
package alerts

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jontk/s9s/plugins/observability/config"
	"github.com/jontk/s9s/plugins/observability/models"
	"github.com/jontk/s9s/plugins/observability/prometheus"
)

// Engine manages alert evaluation and state
type Engine struct {
	config       *config.AlertConfig
	client       *prometheus.CachedClient
	queryBuilder *prometheus.QueryBuilder

	// Alert state
	activeAlerts map[string]*Alert
	alertHistory []Alert
	rules        []Rule

	// Collectors for metrics
	nodeCollector *models.NodeMetricsCollector
	jobCollector  *models.JobMetricsCollector

	// Control
	stopChan    chan struct{}
	checkTicker *time.Ticker
	mu          sync.RWMutex

	// Callbacks
	onAlert    func(alert Alert)
	onResolved func(alert Alert)
}

// Alert represents an active or historical alert
type Alert struct {
	ID          string            `json:"id"`
	RuleName    string            `json:"rule_name"`
	Severity    string            `json:"severity"`
	State       AlertState        `json:"state"`
	Message     string            `json:"message"`
	Description string            `json:"description"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Value       float64           `json:"value"`
	Threshold   float64           `json:"threshold"`
	FirstSeen   time.Time         `json:"first_seen"`
	LastSeen    time.Time         `json:"last_seen"`
	ResolvedAt  time.Time         `json:"resolved_at,omitempty"`
	Duration    time.Duration     `json:"duration"`
	Source      string            `json:"source"`
}

// AlertState represents the state of an alert
type AlertState string

const (
	AlertStatePending  AlertState = "pending"
	AlertStateFiring   AlertState = "firing"
	AlertStateResolved AlertState = "resolved"
)

// NewEngine creates a new alert engine
func NewEngine(config *config.AlertConfig, client *prometheus.CachedClient) *Engine {
	qb, _ := prometheus.NewQueryBuilder()

	return &Engine{
		config:        config,
		client:        client,
		queryBuilder:  qb,
		activeAlerts:  make(map[string]*Alert),
		alertHistory:  make([]Alert, 0),
		rules:         make([]Rule, 0),
		nodeCollector: models.NewNodeMetricsCollector("instance"),
		jobCollector:  models.NewJobMetricsCollector("/slurm/uid_.*/job_%s"),
		stopChan:      make(chan struct{}),
	}
}

// Start starts the alert engine
func (e *Engine) Start(ctx context.Context) error {
	if !e.config.Enabled {
		return nil
	}

	// Load rules from configuration
	if err := e.loadRules(); err != nil {
		return fmt.Errorf("failed to load alert rules: %w", err)
	}

	// Start evaluation loop
	e.checkTicker = time.NewTicker(e.config.CheckInterval)
	go e.evaluationLoop(ctx)

	return nil
}

// Stop stops the alert engine
func (e *Engine) Stop() error {
	close(e.stopChan)
	if e.checkTicker != nil {
		e.checkTicker.Stop()
	}
	return nil
}

// evaluationLoop runs periodic alert evaluation
func (e *Engine) evaluationLoop(ctx context.Context) {
	// Initial evaluation
	e.evaluateAll(ctx)

	for {
		select {
		case <-e.checkTicker.C:
			e.evaluateAll(ctx)
		case <-e.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// evaluateAll evaluates all alert rules
func (e *Engine) evaluateAll(ctx context.Context) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Track which alerts are still active
	stillActive := make(map[string]bool)

	// Evaluate each rule
	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}

		alerts := e.evaluateRule(ctx, rule)
		for _, alert := range alerts {
			alertID := e.generateAlertID(alert)
			stillActive[alertID] = true

			// Check if this is a new alert or update existing
			if existing, exists := e.activeAlerts[alertID]; exists {
				// Update existing alert
				existing.LastSeen = time.Now()
				existing.Duration = existing.LastSeen.Sub(existing.FirstSeen)
				existing.Value = alert.Value

				// Check if pending alert should transition to firing
				if existing.State == AlertStatePending {
					if existing.Duration >= rule.Duration {
						existing.State = AlertStateFiring
						e.fireAlert(*existing)
					}
				}
			} else {
				// New alert
				alert.ID = alertID
				alert.FirstSeen = time.Now()
				alert.LastSeen = time.Now()
				alert.State = AlertStatePending

				if rule.Duration == 0 {
					// No duration requirement, fire immediately
					alert.State = AlertStateFiring
					e.fireAlert(alert)
				}

				e.activeAlerts[alertID] = &alert
			}
		}
	}

	// Check for resolved alerts
	for id, alert := range e.activeAlerts {
		if !stillActive[id] {
			// Alert is resolved
			alert.State = AlertStateResolved
			alert.ResolvedAt = time.Now()
			e.resolveAlert(*alert)

			// Add to history
			e.addToHistory(*alert)

			// Remove from active alerts
			delete(e.activeAlerts, id)
		}
	}

	// Clean up old history
	e.cleanupHistory()
}

// evaluateRule evaluates a single alert rule
func (e *Engine) evaluateRule(ctx context.Context, rule Rule) []Alert {
	alerts := []Alert{}

	switch rule.Type {
	case RuleTypeThreshold:
		// Simple threshold rule
		alerts = e.evaluateThresholdRule(ctx, rule)

	case RuleTypeQuery:
		// Custom PromQL query
		alerts = e.evaluateQueryRule(ctx, rule)

	case RuleTypeComposite:
		// Composite rule (multiple conditions)
		alerts = e.evaluateCompositeRule(ctx, rule)
	}

	return alerts
}

// evaluateThresholdRule evaluates a simple threshold rule
func (e *Engine) evaluateThresholdRule(ctx context.Context, rule Rule) []Alert {
	alerts := []Alert{}

	// Get current metrics based on rule target
	switch rule.Target {
	case "node":
		// Evaluate for all nodes
		nodes := e.nodeCollector.GetAllNodes()
		for nodeName, node := range nodes {
			value := e.getNodeMetricValue(node, rule.Metric)
			if value != nil && e.checkThreshold(*value, rule.Operator, rule.Threshold) {
				alert := Alert{
					RuleName:    rule.Name,
					Severity:    rule.Severity,
					Message:     e.formatAlertMessage(rule, nodeName, *value),
					Description: rule.Description,
					Value:       *value,
					Threshold:   rule.Threshold,
					Source:      fmt.Sprintf("node:%s", nodeName),
					Labels: map[string]string{
						"node":   nodeName,
						"metric": rule.Metric,
					},
				}
				alerts = append(alerts, alert)
			}
		}

	case "job":
		// Evaluate for all jobs
		jobs := e.jobCollector.GetActiveJobs()
		for jobID, job := range jobs {
			value := e.getJobMetricValue(job, rule.Metric)
			if value != nil && e.checkThreshold(*value, rule.Operator, rule.Threshold) {
				alert := Alert{
					RuleName:    rule.Name,
					Severity:    rule.Severity,
					Message:     e.formatAlertMessage(rule, jobID, *value),
					Description: rule.Description,
					Value:       *value,
					Threshold:   rule.Threshold,
					Source:      fmt.Sprintf("job:%s", jobID),
					Labels: map[string]string{
						"job":    jobID,
						"user":   job.User,
						"metric": rule.Metric,
					},
				}
				alerts = append(alerts, alert)
			}
		}

	case "cluster":
		// Evaluate cluster-wide metrics
		value := e.getClusterMetricValue(ctx, rule.Metric)
		if value != nil && e.checkThreshold(*value, rule.Operator, rule.Threshold) {
			alert := Alert{
				RuleName:    rule.Name,
				Severity:    rule.Severity,
				Message:     e.formatAlertMessage(rule, "cluster", *value),
				Description: rule.Description,
				Value:       *value,
				Threshold:   rule.Threshold,
				Source:      "cluster",
				Labels: map[string]string{
					"metric": rule.Metric,
				},
			}
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// evaluateQueryRule evaluates a custom PromQL query rule
func (e *Engine) evaluateQueryRule(ctx context.Context, rule Rule) []Alert {
	alerts := []Alert{}

	// Execute the custom query
	result, err := e.client.Query(ctx, rule.Query, time.Now())
	if err != nil {
		// Log error but don't fail
		return alerts
	}

	// Get vector results (most query rules will return vectors)
	vector, err := result.GetVector()
	if err != nil {
		// Try scalar result as fallback
		scalarValue, _, scalarErr := result.GetScalar()
		if scalarErr != nil {
			// Neither vector nor scalar, skip
			return alerts
		}

		// Process scalar result
		if e.checkThreshold(scalarValue, rule.Operator, rule.Threshold) {
			alert := Alert{
				RuleName:    rule.Name,
				Severity:    rule.Severity,
				Message:     e.formatAlertMessage(rule, "", scalarValue),
				Description: rule.Description,
				Value:       scalarValue,
				Threshold:   rule.Threshold,
				Source:      "prometheus",
				Labels:      map[string]string{"metric": rule.Metric},
			}
			alerts = append(alerts, alert)
		}
		return alerts
	}

	// Process vector results
	for _, sample := range vector {
		value := sample.Value.Value()

		if e.checkThreshold(value, rule.Operator, rule.Threshold) {
			// Build labels from metric labels
			labels := make(map[string]string)
			for k, v := range sample.Metric {
				labels[k] = v
			}
			labels["metric"] = rule.Metric

			alert := Alert{
				RuleName:    rule.Name,
				Severity:    rule.Severity,
				Message:     e.formatAlertMessage(rule, "", value),
				Description: rule.Description,
				Value:       value,
				Threshold:   rule.Threshold,
				Source:      "prometheus",
				Labels:      labels,
			}
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// evaluateCompositeRule evaluates a composite rule
func (e *Engine) evaluateCompositeRule(ctx context.Context, rule Rule) []Alert {
	// TODO: Implement composite rule evaluation
	return []Alert{}
}

// getNodeMetricValue extracts a metric value from node metrics
func (e *Engine) getNodeMetricValue(node *models.NodeMetrics, metric string) *float64 {
	var value float64

	switch metric {
	case "cpu_usage":
		value = node.Resources.CPU.Usage
	case "memory_usage":
		value = node.Resources.Memory.Usage
	case "load_1m":
		value = node.Resources.CPU.Load1m
	case "load_per_core":
		if node.Resources.CPU.Cores > 0 {
			value = node.Resources.CPU.Load1m / float64(node.Resources.CPU.Cores)
		}
	case "disk_io_util":
		value = node.Resources.Disk.IOUtilization
	default:
		return nil
	}

	return &value
}

// getJobMetricValue extracts a metric value from job metrics
func (e *Engine) getJobMetricValue(job *models.JobMetrics, metric string) *float64 {
	var value float64

	switch metric {
	case "cpu_usage":
		value = job.Resources.CPU.Usage
	case "memory_usage":
		value = job.Resources.Memory.Usage
	case "memory_usage_percent":
		if job.AllocatedMem > 0 {
			value = float64(job.Resources.Memory.Used) / float64(job.AllocatedMem) * 100
		}
	case "cpu_efficiency":
		value = job.Efficiency.CPUEfficiency
	case "memory_efficiency":
		value = job.Efficiency.MemEfficiency
	case "overall_efficiency":
		value = job.Efficiency.OverallEfficiency
	default:
		return nil
	}

	return &value
}

// getClusterMetricValue gets a cluster-wide metric value
func (e *Engine) getClusterMetricValue(ctx context.Context, metric string) *float64 {
	// Get aggregate metrics
	agg := e.nodeCollector.GetAggregateMetrics()
	if agg == nil {
		return nil
	}

	var value float64

	switch metric {
	case "cpu_usage":
		value = agg.AverageCPUUsage
	case "memory_usage":
		value = agg.MemoryUsagePercent
	case "active_nodes":
		value = float64(agg.ActiveNodes)
	case "total_jobs":
		value = float64(agg.TotalJobs)
	default:
		return nil
	}

	return &value
}

// checkThreshold checks if a value meets the threshold condition
func (e *Engine) checkThreshold(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return value > threshold
	case "<":
		return value < threshold
	case ">=":
		return value >= threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	case "!=":
		return value != threshold
	default:
		return false
	}
}

// formatAlertMessage formats an alert message
func (e *Engine) formatAlertMessage(rule Rule, target string, value float64) string {
	if rule.MessageTemplate != "" {
		// TODO: Implement template rendering
		return rule.MessageTemplate
	}

	// Default message
	if target != "" {
		return fmt.Sprintf("%s: %s %s %s %.2f (threshold: %.2f)",
			target, rule.Metric, rule.Operator, "is", value, rule.Threshold)
	}

	return fmt.Sprintf("%s %s %.2f (threshold: %.2f)",
		rule.Metric, rule.Operator, value, rule.Threshold)
}

// generateAlertID generates a unique ID for an alert
func (e *Engine) generateAlertID(alert Alert) string {
	// Generate ID based on rule name and source
	return fmt.Sprintf("%s:%s", alert.RuleName, alert.Source)
}

// fireAlert is called when an alert transitions to firing state
func (e *Engine) fireAlert(alert Alert) {
	if e.onAlert != nil {
		e.onAlert(alert)
	}
}

// resolveAlert is called when an alert is resolved
func (e *Engine) resolveAlert(alert Alert) {
	if e.onResolved != nil {
		e.onResolved(alert)
	}
}

// addToHistory adds an alert to the history
func (e *Engine) addToHistory(alert Alert) {
	e.alertHistory = append([]Alert{alert}, e.alertHistory...)

	// Limit history size
	maxHistory := 1000
	if len(e.alertHistory) > maxHistory {
		e.alertHistory = e.alertHistory[:maxHistory]
	}
}

// cleanupHistory removes old alerts from history
func (e *Engine) cleanupHistory() {
	if e.config.HistoryRetention == 0 {
		return
	}

	cutoff := time.Now().Add(-e.config.HistoryRetention)
	newHistory := []Alert{}

	for _, alert := range e.alertHistory {
		if alert.ResolvedAt.After(cutoff) || alert.LastSeen.After(cutoff) {
			newHistory = append(newHistory, alert)
		}
	}

	e.alertHistory = newHistory
}

// GetActiveAlerts returns all active alerts
func (e *Engine) GetActiveAlerts() []Alert {
	e.mu.RLock()
	defer e.mu.RUnlock()

	alerts := make([]Alert, 0, len(e.activeAlerts))
	for _, alert := range e.activeAlerts {
		alerts = append(alerts, *alert)
	}

	return alerts
}

// GetAlertHistory returns alert history
func (e *Engine) GetAlertHistory(limit int) []Alert {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if limit <= 0 || limit > len(e.alertHistory) {
		limit = len(e.alertHistory)
	}

	return e.alertHistory[:limit]
}

// SetAlertCallback sets the callback for new alerts
func (e *Engine) SetAlertCallback(fn func(Alert)) {
	e.onAlert = fn
}

// SetResolvedCallback sets the callback for resolved alerts
func (e *Engine) SetResolvedCallback(fn func(Alert)) {
	e.onResolved = fn
}

// loadRules loads alert rules from configuration
func (e *Engine) loadRules() error {
	// Start with predefined rules (if enabled in config)
	if e.config.LoadPredefinedRules {
		for _, rule := range PredefinedRules {
			if rule.Enabled {
				e.rules = append(e.rules, rule)
			}
		}
	}

	// Convert config rules to engine rules
	for _, configRule := range e.config.Rules {
		rule := Rule{
			Name:        configRule.Name,
			Type:        RuleTypeThreshold,
			Target:      "node", // Default to node
			Metric:      configRule.Metric,
			Operator:    configRule.Operator,
			Threshold:   configRule.Threshold,
			Duration:    configRule.Duration,
			Severity:    configRule.Severity,
			Enabled:     configRule.Enabled,
			Description: fmt.Sprintf("Alert when %s %s %.2f", configRule.Metric, configRule.Operator, configRule.Threshold),
		}

		// Determine target from metric name
		if strings.Contains(configRule.Metric, "job_") {
			rule.Target = "job"
		} else if strings.Contains(configRule.Metric, "cluster_") {
			rule.Target = "cluster"
		}

		e.rules = append(e.rules, rule)
	}

	return nil
}

// AddRule adds a new alert rule
func (e *Engine) AddRule(rule Rule) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.rules = append(e.rules, rule)
}

// RemoveRule removes an alert rule by name
func (e *Engine) RemoveRule(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	newRules := []Rule{}
	for _, rule := range e.rules {
		if rule.Name != name {
			newRules = append(newRules, rule)
		}
	}
	e.rules = newRules
}

// GetRules returns all configured rules
func (e *Engine) GetRules() []Rule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	rules := make([]Rule, len(e.rules))
	copy(rules, e.rules)
	return rules
}

// SetNodeCollector updates the node metrics collector
func (e *Engine) SetNodeCollector(collector *models.NodeMetricsCollector) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.nodeCollector = collector
}

// SetJobCollector updates the job metrics collector
func (e *Engine) SetJobCollector(collector *models.JobMetricsCollector) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.jobCollector = collector
}
