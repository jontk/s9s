package prometheus

import (
	"fmt"
	"strings"
	"text/template"
)

// QueryTemplates contains standard PromQL query templates
var QueryTemplates = map[string]string{
	// Node-level metrics (node-exporter)
	"node_cpu_usage": `100 - (avg by ({{.NodeLabel}}) (rate(node_cpu_seconds_total{mode="idle", {{.NodeLabel}}=~"{{.NodePattern}}"}[{{.Range}}])) * 100)`,
	
	"node_cpu_cores": `count by ({{.NodeLabel}}) (node_cpu_seconds_total{mode="idle", {{.NodeLabel}}=~"{{.NodePattern}}"})`,
	
	"node_memory_usage": `(1 - (node_memory_MemAvailable_bytes{{{.NodeLabel}}=~"{{.NodePattern}}"} / node_memory_MemTotal_bytes{{{.NodeLabel}}=~"{{.NodePattern}}"})) * 100`,
	
	"node_memory_total": `node_memory_MemTotal_bytes{{{.NodeLabel}}=~"{{.NodePattern}}"}`,
	
	"node_memory_available": `node_memory_MemAvailable_bytes{{{.NodeLabel}}=~"{{.NodePattern}}"}`,
	
	"node_load_1m": `node_load1{{{.NodeLabel}}=~"{{.NodePattern}}"}`,
	
	"node_load_5m": `node_load5{{{.NodeLabel}}=~"{{.NodePattern}}"}`,
	
	"node_load_15m": `node_load15{{{.NodeLabel}}=~"{{.NodePattern}}"}`,
	
	"node_disk_read_bytes": `rate(node_disk_read_bytes_total{{{.NodeLabel}}=~"{{.NodePattern}}", device!~"dm-.*"}[{{.Range}}])`,
	
	"node_disk_write_bytes": `rate(node_disk_write_bytes_total{{{.NodeLabel}}=~"{{.NodePattern}}", device!~"dm-.*"}[{{.Range}}])`,
	
	"node_network_receive_bytes": `rate(node_network_receive_bytes_total{{{.NodeLabel}}=~"{{.NodePattern}}", device!~"lo"}[{{.Range}}])`,
	
	"node_network_transmit_bytes": `rate(node_network_transmit_bytes_total{{{.NodeLabel}}=~"{{.NodePattern}}", device!~"lo"}[{{.Range}}])`,
	
	"node_filesystem_usage": `(1 - (node_filesystem_avail_bytes{{{.NodeLabel}}=~"{{.NodePattern}}", fstype!~"tmpfs|fuse.lxcfs"} / node_filesystem_size_bytes{{{.NodeLabel}}=~"{{.NodePattern}}", fstype!~"tmpfs|fuse.lxcfs"})) * 100`,
	
	// Job-level metrics (cgroup-exporter)
	"job_cpu_usage": `rate(container_cpu_usage_seconds_total{id=~"/slurm/uid_.*/job_{{.JobID}}"}[{{.Range}}]) * 100`,
	
	"job_cpu_throttled": `rate(container_cpu_throttled_seconds_total{id=~"/slurm/uid_.*/job_{{.JobID}}"}[{{.Range}}])`,
	
	"job_memory_usage": `container_memory_usage_bytes{id=~"/slurm/uid_.*/job_{{.JobID}}"}`,
	
	"job_memory_limit": `container_spec_memory_limit_bytes{id=~"/slurm/uid_.*/job_{{.JobID}}"}`,
	
	"job_memory_cache": `container_memory_cache{id=~"/slurm/uid_.*/job_{{.JobID}}"}`,
	
	"job_memory_rss": `container_memory_rss{id=~"/slurm/uid_.*/job_{{.JobID}}"}`,
	
	// Cluster-level aggregations
	"cluster_cpu_usage": `100 - (avg(rate(node_cpu_seconds_total{mode="idle"}[{{.Range}}])) * 100)`,
	
	"cluster_memory_usage": `(1 - (sum(node_memory_MemAvailable_bytes) / sum(node_memory_MemTotal_bytes))) * 100`,
	
	"cluster_nodes_up": `count(up{job="node-exporter"} == 1)`,
	
	"cluster_nodes_down": `count(up{job="node-exporter"} == 0)`,
}

// QueryBuilder helps build PromQL queries with parameters
type QueryBuilder struct {
	templates map[string]*template.Template
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder() (*QueryBuilder, error) {
	qb := &QueryBuilder{
		templates: make(map[string]*template.Template),
	}
	
	// Parse all query templates
	for name, tmpl := range QueryTemplates {
		t, err := template.New(name).Parse(tmpl)
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %s: %w", name, err)
		}
		qb.templates[name] = t
	}
	
	return qb, nil
}

// BuildQuery builds a query from a template with parameters
func (qb *QueryBuilder) BuildQuery(queryName string, params map[string]interface{}) (string, error) {
	tmpl, exists := qb.templates[queryName]
	if !exists {
		return "", fmt.Errorf("unknown query template: %s", queryName)
	}
	
	// Set defaults
	if params == nil {
		params = make(map[string]interface{})
	}
	
	// Default parameters
	if _, ok := params["Range"]; !ok {
		params["Range"] = "5m"
	}
	if _, ok := params["NodeLabel"]; !ok {
		params["NodeLabel"] = "instance"
	}
	if _, ok := params["NodePattern"]; !ok {
		params["NodePattern"] = ".*"
	}
	
	var result strings.Builder
	if err := tmpl.Execute(&result, params); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	
	return result.String(), nil
}

// GetNodeQueries returns all node-level queries for a specific node
func (qb *QueryBuilder) GetNodeQueries(nodeName string, labelName string) (map[string]string, error) {
	queries := make(map[string]string)
	
	nodeQueries := []string{
		"node_cpu_usage",
		"node_memory_usage",
		"node_load_1m",
		"node_disk_read_bytes",
		"node_disk_write_bytes",
		"node_network_receive_bytes",
		"node_network_transmit_bytes",
	}
	
	params := map[string]interface{}{
		"NodeLabel":   labelName,
		"NodePattern": escapeRegex(nodeName),
	}
	
	for _, queryName := range nodeQueries {
		query, err := qb.BuildQuery(queryName, params)
		if err != nil {
			return nil, fmt.Errorf("failed to build query %s: %w", queryName, err)
		}
		queries[queryName] = query
	}
	
	return queries, nil
}

// GetJobQueries returns all job-level queries for a specific job
func (qb *QueryBuilder) GetJobQueries(jobID string) (map[string]string, error) {
	queries := make(map[string]string)
	
	jobQueries := []string{
		"job_cpu_usage",
		"job_cpu_throttled",
		"job_memory_usage",
		"job_memory_limit",
		"job_memory_cache",
		"job_memory_rss",
	}
	
	params := map[string]interface{}{
		"JobID": jobID,
	}
	
	for _, queryName := range jobQueries {
		query, err := qb.BuildQuery(queryName, params)
		if err != nil {
			return nil, fmt.Errorf("failed to build query %s: %w", queryName, err)
		}
		queries[queryName] = query
	}
	
	return queries, nil
}

// GetClusterQueries returns cluster-level aggregation queries
func (qb *QueryBuilder) GetClusterQueries() (map[string]string, error) {
	queries := make(map[string]string)
	
	clusterQueries := []string{
		"cluster_cpu_usage",
		"cluster_memory_usage",
		"cluster_nodes_up",
		"cluster_nodes_down",
	}
	
	for _, queryName := range clusterQueries {
		query, err := qb.BuildQuery(queryName, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to build query %s: %w", queryName, err)
		}
		queries[queryName] = query
	}
	
	return queries, nil
}

// escapeRegex escapes special regex characters in a string
func escapeRegex(s string) string {
	special := []string{".", "+", "*", "?", "^", "$", "(", ")", "[", "]", "{", "}", "|", "\\"}
	result := s
	for _, char := range special {
		result = strings.ReplaceAll(result, char, "\\"+char)
	}
	return result
}

// StandardQueries provides pre-built queries for common use cases
type StandardQueries struct {
	qb *QueryBuilder
}

// NewStandardQueries creates a new standard queries helper
func NewStandardQueries() (*StandardQueries, error) {
	qb, err := NewQueryBuilder()
	if err != nil {
		return nil, err
	}
	
	return &StandardQueries{qb: qb}, nil
}

// GetNodeResourceQuery returns a query for node resource usage
func (sq *StandardQueries) GetNodeResourceQuery(nodeName, resource, labelName string) (string, error) {
	var queryName string
	switch resource {
	case "cpu":
		queryName = "node_cpu_usage"
	case "memory":
		queryName = "node_memory_usage"
	case "load":
		queryName = "node_load_1m"
	case "disk_read":
		queryName = "node_disk_read_bytes"
	case "disk_write":
		queryName = "node_disk_write_bytes"
	case "network_rx":
		queryName = "node_network_receive_bytes"
	case "network_tx":
		queryName = "node_network_transmit_bytes"
	default:
		return "", fmt.Errorf("unknown resource type: %s", resource)
	}
	
	return sq.qb.BuildQuery(queryName, map[string]interface{}{
		"NodeLabel":   labelName,
		"NodePattern": escapeRegex(nodeName),
	})
}

// GetJobResourceQuery returns a query for job resource usage
func (sq *StandardQueries) GetJobResourceQuery(jobID, resource string) (string, error) {
	var queryName string
	switch resource {
	case "cpu":
		queryName = "job_cpu_usage"
	case "memory":
		queryName = "job_memory_usage"
	case "memory_limit":
		queryName = "job_memory_limit"
	case "cpu_throttled":
		queryName = "job_cpu_throttled"
	default:
		return "", fmt.Errorf("unknown resource type: %s", resource)
	}
	
	return sq.qb.BuildQuery(queryName, map[string]interface{}{
		"JobID": jobID,
	})
}