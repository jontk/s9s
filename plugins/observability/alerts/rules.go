package alerts

import (
	"fmt"
	"time"
)

// RuleType defines the type of alert rule
type RuleType string

const (
	// RuleTypeThreshold is the threshold rule type for alerts.
	RuleTypeThreshold RuleType = "threshold"
	// RuleTypeQuery is the query rule type for alerts.
	RuleTypeQuery RuleType = "query"
	// RuleTypeComposite is the composite rule type for alerts.
	RuleTypeComposite RuleType = "composite"
)

// Rule defines an alert rule
type Rule struct {
	Name            string            `json:"name"`
	Type            RuleType          `json:"type"`
	Target          string            `json:"target"` // "node", "job", "cluster"
	Metric          string            `json:"metric"`
	Operator        string            `json:"operator"`
	Threshold       float64           `json:"threshold"`
	Duration        time.Duration     `json:"duration"`
	Query           string            `json:"query,omitempty"`      // For query type rules
	Conditions      []RuleCondition   `json:"conditions,omitempty"` // For composite rules
	Severity        string            `json:"severity"`
	Enabled         bool              `json:"enabled"`
	Description     string            `json:"description"`
	MessageTemplate string            `json:"message_template,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	Annotations     map[string]string `json:"annotations,omitempty"`
}

// RuleCondition defines a condition for composite rules
type RuleCondition struct {
	Metric    string  `json:"metric"`
	Operator  string  `json:"operator"`
	Threshold float64 `json:"threshold"`
	Weight    float64 `json:"weight"` // For weighted conditions
}

// PredefinedRules contains commonly used alert rules
var PredefinedRules = []Rule{
	// Node alerts
	{
		Name:        "high_node_cpu_usage",
		Type:        RuleTypeThreshold,
		Target:      "node",
		Metric:      "cpu_usage",
		Operator:    ">",
		Threshold:   90.0,
		Duration:    5 * time.Minute,
		Severity:    "warning",
		Enabled:     true,
		Description: "Node CPU usage is above 90%",
	},
	{
		Name:        "critical_node_cpu_usage",
		Type:        RuleTypeThreshold,
		Target:      "node",
		Metric:      "cpu_usage",
		Operator:    ">",
		Threshold:   95.0,
		Duration:    5 * time.Minute,
		Severity:    "critical",
		Enabled:     true,
		Description: "Node CPU usage is critically high",
	},
	{
		Name:        "high_node_memory_usage",
		Type:        RuleTypeThreshold,
		Target:      "node",
		Metric:      "memory_usage",
		Operator:    ">",
		Threshold:   90.0,
		Duration:    5 * time.Minute,
		Severity:    "warning",
		Enabled:     true,
		Description: "Node memory usage is above 90%",
	},
	{
		Name:        "critical_node_memory_usage",
		Type:        RuleTypeThreshold,
		Target:      "node",
		Metric:      "memory_usage",
		Operator:    ">",
		Threshold:   95.0,
		Duration:    5 * time.Minute,
		Severity:    "critical",
		Enabled:     true,
		Description: "Node memory usage is critically high",
	},
	{
		Name:        "high_load_average",
		Type:        RuleTypeThreshold,
		Target:      "node",
		Metric:      "load_per_core",
		Operator:    ">",
		Threshold:   2.0,
		Duration:    10 * time.Minute,
		Severity:    "warning",
		Enabled:     true,
		Description: "Load average per core is high",
	},
	{
		Name:        "disk_io_saturation",
		Type:        RuleTypeThreshold,
		Target:      "node",
		Metric:      "disk_io_util",
		Operator:    ">",
		Threshold:   90.0,
		Duration:    5 * time.Minute,
		Severity:    "warning",
		Enabled:     true,
		Description: "Disk I/O utilization is saturated",
	},

	// Job alerts
	{
		Name:        "job_memory_limit",
		Type:        RuleTypeThreshold,
		Target:      "job",
		Metric:      "memory_usage_percent",
		Operator:    ">",
		Threshold:   95.0,
		Duration:    5 * time.Minute,
		Severity:    "critical",
		Enabled:     true,
		Description: "Job is approaching memory limit",
	},
	{
		Name:        "job_inefficient",
		Type:        RuleTypeThreshold,
		Target:      "job",
		Metric:      "overall_efficiency",
		Operator:    "<",
		Threshold:   20.0,
		Duration:    30 * time.Minute,
		Severity:    "warning",
		Enabled:     true,
		Description: "Job is using less than 20% of allocated resources",
	},
	{
		Name:        "job_cpu_throttled",
		Type:        RuleTypeQuery,
		Target:      "job",
		Query:       `rate(container_cpu_throttled_seconds_total[5m]) > 0.1`,
		Operator:    ">",
		Threshold:   0.1,
		Duration:    5 * time.Minute,
		Severity:    "warning",
		Enabled:     false,
		Description: "Job CPU is being throttled",
	},

	// Cluster alerts
	{
		Name:        "cluster_cpu_high",
		Type:        RuleTypeThreshold,
		Target:      "cluster",
		Metric:      "cpu_usage",
		Operator:    ">",
		Threshold:   80.0,
		Duration:    15 * time.Minute,
		Severity:    "warning",
		Enabled:     true,
		Description: "Cluster-wide CPU usage is high",
	},
	{
		Name:        "cluster_memory_high",
		Type:        RuleTypeThreshold,
		Target:      "cluster",
		Metric:      "memory_usage",
		Operator:    ">",
		Threshold:   85.0,
		Duration:    15 * time.Minute,
		Severity:    "warning",
		Enabled:     true,
		Description: "Cluster-wide memory usage is high",
	},
	{
		Name:        "nodes_down",
		Type:        RuleTypeQuery,
		Target:      "cluster",
		Query:       `count(up{job="node-exporter"} == 0) > 0`,
		Operator:    ">",
		Threshold:   0,
		Duration:    5 * time.Minute,
		Severity:    "critical",
		Enabled:     false,
		Description: "One or more nodes are down",
	},
}

// RuleTemplates provides templates for creating custom rules
var RuleTemplates = map[string]Rule{
	"node_metric_threshold": {
		Name:     "node_${metric}_threshold",
		Type:     RuleTypeThreshold,
		Target:   "node",
		Metric:   "${metric}",
		Operator: ">",
		Severity: "warning",
		Enabled:  true,
	},
	"job_metric_threshold": {
		Name:     "job_${metric}_threshold",
		Type:     RuleTypeThreshold,
		Target:   "job",
		Metric:   "${metric}",
		Operator: ">",
		Severity: "warning",
		Enabled:  true,
	},
	"custom_query": {
		Name:     "custom_query_alert",
		Type:     RuleTypeQuery,
		Target:   "cluster",
		Query:    "${query}",
		Operator: ">",
		Severity: "warning",
		Enabled:  true,
	},
}

// ValidateRule validates an alert rule
func ValidateRule(rule *Rule) error {
	// Validate basic fields
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}

	if rule.Type == "" {
		return fmt.Errorf("rule type is required")
	}

	// Validate rule type-specific requirements
	if err := validateRuleType(rule); err != nil {
		return err
	}

	// Validate severity
	if !isValidSeverity(rule.Severity) {
		return fmt.Errorf("invalid severity: %s", rule.Severity)
	}

	return nil
}

// validateRuleType validates requirements specific to each rule type
func validateRuleType(rule *Rule) error {
	switch rule.Type {
	case RuleTypeThreshold:
		return validateThresholdRule(rule)
	case RuleTypeQuery:
		return validateQueryRule(rule)
	case RuleTypeComposite:
		return validateCompositeRule(rule)
	default:
		return fmt.Errorf("invalid rule type: %s", rule.Type)
	}
}

// validateThresholdRule validates threshold rule requirements
func validateThresholdRule(rule *Rule) error {
	if rule.Metric == "" {
		return fmt.Errorf("metric is required for threshold rules")
	}
	if rule.Operator == "" {
		return fmt.Errorf("operator is required for threshold rules")
	}
	if !isValidOperator(rule.Operator) {
		return fmt.Errorf("invalid operator: %s", rule.Operator)
	}
	return nil
}

// validateQueryRule validates query rule requirements
func validateQueryRule(rule *Rule) error {
	if rule.Query == "" {
		return fmt.Errorf("query is required for query rules")
	}
	return nil
}

// validateCompositeRule validates composite rule requirements
func validateCompositeRule(rule *Rule) error {
	if len(rule.Conditions) == 0 {
		return fmt.Errorf("conditions are required for composite rules")
	}

	for i, cond := range rule.Conditions {
		if cond.Metric == "" {
			return fmt.Errorf("condition %d: metric is required", i)
		}
		if !isValidOperator(cond.Operator) {
			return fmt.Errorf("condition %d: invalid operator: %s", i, cond.Operator)
		}
	}
	return nil
}

// isValidOperator checks if an operator is valid
func isValidOperator(op string) bool {
	validOps := map[string]bool{
		">": true, "<": true, ">=": true, "<=": true, "==": true, "!=": true,
	}
	return validOps[op]
}

// isValidSeverity checks if a severity is valid
func isValidSeverity(severity string) bool {
	validSeverities := map[string]bool{
		"info": true, "warning": true, "critical": true,
	}
	return validSeverities[severity]
}

// RuleBuilder helps build alert rules programmatically
type RuleBuilder struct {
	rule Rule
}

// NewRuleBuilder creates a new rule builder
func NewRuleBuilder(name string) *RuleBuilder {
	return &RuleBuilder{
		rule: Rule{
			Name:    name,
			Type:    RuleTypeThreshold,
			Enabled: true,
		},
	}
}

// WithType sets the rule type
func (rb *RuleBuilder) WithType(ruleType RuleType) *RuleBuilder {
	rb.rule.Type = ruleType
	return rb
}

// WithTarget sets the target
func (rb *RuleBuilder) WithTarget(target string) *RuleBuilder {
	rb.rule.Target = target
	return rb
}

// WithMetric sets the metric
func (rb *RuleBuilder) WithMetric(metric string) *RuleBuilder {
	rb.rule.Metric = metric
	return rb
}

// WithThreshold sets the threshold and operator
func (rb *RuleBuilder) WithThreshold(operator string, threshold float64) *RuleBuilder {
	rb.rule.Operator = operator
	rb.rule.Threshold = threshold
	return rb
}

// WithQuery sets the PromQL query
func (rb *RuleBuilder) WithQuery(query string) *RuleBuilder {
	rb.rule.Query = query
	rb.rule.Type = RuleTypeQuery
	return rb
}

// WithDuration sets the duration
func (rb *RuleBuilder) WithDuration(duration time.Duration) *RuleBuilder {
	rb.rule.Duration = duration
	return rb
}

// WithSeverity sets the severity
func (rb *RuleBuilder) WithSeverity(severity string) *RuleBuilder {
	rb.rule.Severity = severity
	return rb
}

// WithDescription sets the description
func (rb *RuleBuilder) WithDescription(description string) *RuleBuilder {
	rb.rule.Description = description
	return rb
}

// WithMessageTemplate sets the message template
func (rb *RuleBuilder) WithMessageTemplate(template string) *RuleBuilder {
	rb.rule.MessageTemplate = template
	return rb
}

// WithLabel adds a label
func (rb *RuleBuilder) WithLabel(key, value string) *RuleBuilder {
	if rb.rule.Labels == nil {
		rb.rule.Labels = make(map[string]string)
	}
	rb.rule.Labels[key] = value
	return rb
}

// WithAnnotation adds an annotation
func (rb *RuleBuilder) WithAnnotation(key, value string) *RuleBuilder {
	if rb.rule.Annotations == nil {
		rb.rule.Annotations = make(map[string]string)
	}
	rb.rule.Annotations[key] = value
	return rb
}

// Build validates and returns the rule
func (rb *RuleBuilder) Build() (Rule, error) {
	if err := ValidateRule(&rb.rule); err != nil {
		return Rule{}, err
	}
	return rb.rule, nil
}
