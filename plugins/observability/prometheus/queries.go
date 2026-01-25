package prometheus

import (
	"fmt"
	"strings"
)

// QueryTemplates contains standard PromQL query templates
// Using a function to build the string to avoid template parsing issues
var QueryTemplates = map[string]string{
	// Node-level metrics (node-exporter)
	"node_cpu_usage": `100 - (avg by ({{.NodeLabel}}) (rate(node_cpu_seconds_total` + `{mode="idle", {{.NodeLabel}}=~"{{.NodePattern}}"}[{{.Range}}])) * 100)`,

	"node_cpu_cores": `count by ({{.NodeLabel}}) (node_cpu_seconds_total` + `{mode="idle", {{.NodeLabel}}=~"{{.NodePattern}}"})`,

	"node_memory_usage": `(1 - (node_memory_MemAvailable_bytes` + `{{{.NodeLabel}}=~"{{.NodePattern}}"} / node_memory_MemTotal_bytes` + `{{{.NodeLabel}}=~"{{.NodePattern}}"})) * 100`,

	"node_memory_total": `node_memory_MemTotal_bytes` + `{{{.NodeLabel}}=~"{{.NodePattern}}"}`,

	"node_memory_available": `node_memory_MemAvailable_bytes` + `{{{.NodeLabel}}=~"{{.NodePattern}}"}`,

	"node_load_1m": `node_load1` + `{{{.NodeLabel}}=~"{{.NodePattern}}"}`,

	"node_load_5m": `node_load5` + `{{{.NodeLabel}}=~"{{.NodePattern}}"}`,

	"node_load_15m": `node_load15` + `{{{.NodeLabel}}=~"{{.NodePattern}}"}`,

	"node_disk_read_bytes": `rate(node_disk_read_bytes_total` + `{{{.NodeLabel}}=~"{{.NodePattern}}", device!~"dm-.*"}[{{.Range}}])`,

	"node_disk_write_bytes": `rate(node_disk_write_bytes_total` + `{{{.NodeLabel}}=~"{{.NodePattern}}", device!~"dm-.*"}[{{.Range}}])`,

	"node_network_receive_bytes": `rate(node_network_receive_bytes_total` + `{{{.NodeLabel}}=~"{{.NodePattern}}", device!~"lo"}[{{.Range}}])`,

	"node_network_transmit_bytes": `rate(node_network_transmit_bytes_total` + `{{{.NodeLabel}}=~"{{.NodePattern}}", device!~"lo"}[{{.Range}}])`,

	"node_filesystem_usage": `(1 - (node_filesystem_avail_bytes` + `{{{.NodeLabel}}=~"{{.NodePattern}}", fstype!~"tmpfs|fuse.lxcfs"} / node_filesystem_size_bytes` + `{{{.NodeLabel}}=~"{{.NodePattern}}", fstype!~"tmpfs|fuse.lxcfs"})) * 100`,

	// Job-level metrics (cgroup-exporter/cAdvisor)
	"job_cpu_usage": `rate(container_cpu_usage_seconds_total` + `{cpu="total",id=~"/system.slice/.*slurmstepd.scope/job_{{.JobID}}"}[{{.Range}}]) * 100`,

	"job_cpu_throttled": `rate(container_cpu_throttled_seconds_total` + `{id=~"/system.slice/.*slurmstepd.scope/job_{{.JobID}}"}[{{.Range}}])`,

	"job_memory_usage": `container_memory_usage_bytes` + `{id=~"/system.slice/.*slurmstepd.scope/job_{{.JobID}}"}`,

	"job_memory_limit": `container_spec_memory_limit_bytes` + `{id=~"/system.slice/.*slurmstepd.scope/job_{{.JobID}}"}`,

	"job_memory_cache": `container_memory_cache` + `{id=~"/system.slice/.*slurmstepd.scope/job_{{.JobID}}"}`,

	"job_memory_rss": `container_memory_rss` + `{id=~"/system.slice/.*slurmstepd.scope/job_{{.JobID}}"}`,

	// Cluster-level aggregations
	"cluster_cpu_usage": `100 - (avg(rate(node_cpu_seconds_total` + `{mode="idle"}[{{.Range}}])) * 100)`,

	"cluster_memory_usage": `(1 - (sum(node_memory_MemAvailable_bytes) / sum(node_memory_MemTotal_bytes))) * 100`,

	"cluster_nodes_up": `count(up` + `{job="node-exporter"} == 1)`,

	"cluster_nodes_down": `count(up` + `{job="node-exporter"} == 0)`,
}

// BuildQueryString builds the actual query string by replacing template variables
// This avoids the template parsing issues with curly braces
func BuildQueryString(template string, params map[string]interface{}) string {
	result := template

	// Set defaults
	if params == nil {
		params = make(map[string]interface{})
	}
	if _, ok := params["Range"]; !ok {
		params["Range"] = "5m"
	}
	if _, ok := params["NodeLabel"]; !ok {
		params["NodeLabel"] = "instance"
	}
	if _, ok := params["NodePattern"]; !ok {
		params["NodePattern"] = ".*"
	}

	// Simple string replacement to avoid template issues
	for key, value := range params {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}

	// Now handle the triple braces by replacing them with single braces
	result = strings.ReplaceAll(result, "{{{", "{")
	result = strings.ReplaceAll(result, "}}}", "}")

	return result
}

// QueryBuilder helps build PromQL queries with parameters
type QueryBuilder struct {
	// We'll skip templates and just use string replacement
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder() (*QueryBuilder, error) {
	return &QueryBuilder{}, nil
}

// BuildQuery builds a query from a template with parameters
func (qb *QueryBuilder) BuildQuery(queryName string, params map[string]interface{}) (string, error) {
	tmpl, exists := QueryTemplates[queryName]
	if !exists {
		return "", fmt.Errorf("unknown query template: %s", queryName)
	}

	return BuildQueryString(tmpl, params), nil
}

// GetNodeQueries returns all node-level queries for a specific node
func (qb *QueryBuilder) GetNodeQueries(nodeName, labelName string) (map[string]string, error) {
	queries := make(map[string]string)

	nodeQueries := []string{
		"node_cpu_usage",
		"node_cpu_cores",
		"node_memory_usage",
		"node_memory_total",
		"node_memory_available",
		"node_load_1m",
		"node_load_5m",
		"node_load_15m",
		"node_disk_read_bytes",
		"node_disk_write_bytes",
		"node_network_receive_bytes",
		"node_network_transmit_bytes",
	}

	// Don't escape the pattern - Prometheus expects the exact instance name
	params := map[string]interface{}{
		"NodeLabel":   labelName,
		"NodePattern": nodeName, // Use exact name, no escaping
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

/*
TODO(lint): Review unused code - func escapeRegex is unused

escapeRegex escapes special regex characters in a string
func escapeRegex(s string) string {
	// Escape common regex special characters
	replacer := strings.NewReplacer(
		".", `\.`,
		"*", `\*`,
		"+", `\+`,
		"?", `\?`,
		"(", `\(`,
		")", `\)`,
		"[", `\[`,
		"]", `\]`,
		"{", `\{`,
		"}", `\}`,
		"^", `\^`,
		"$", `\$`,
		"|", `\|`,
		"\\", `\\`,
	)
	return replacer.Replace(s)
}
*/

// TestTemplateOutput is a helper function for testing query template output.
func TestTemplateOutput() {
	qb, _ := NewQueryBuilder()

	// Test node query
	params := map[string]interface{}{
		"NodeLabel":   "instance",
		"NodePattern": "node001",
	}

	query, _ := qb.BuildQuery("node_cpu_usage", params)
	fmt.Printf("Generated query: %s\n", query)
}
