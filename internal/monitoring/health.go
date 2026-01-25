package monitoring

import (
	"fmt"
	"sync"
	"time"

	"github.com/jontk/s9s/internal/dao"
)

// HealthStatus represents the health status of a cluster component
type HealthStatus string

const (
	// HealthStatusHealthy indicates the component is healthy.
	HealthStatusHealthy HealthStatus = "healthy"
	// HealthStatusWarning indicates the component has warning conditions.
	HealthStatusWarning HealthStatus = "warning"
	// HealthStatusCritical indicates the component is in critical condition.
	HealthStatusCritical HealthStatus = "critical"
	// HealthStatusUnknown indicates the health status is unknown.
	HealthStatusUnknown HealthStatus = "unknown"
)

// HealthCheck represents a single health check
type HealthCheck struct {
	Name        string
	Description string
	Status      HealthStatus
	Message     string
	LastCheck   time.Time
	CheckCount  int
	Threshold   HealthThreshold
}

// HealthThreshold defines warning and critical thresholds for metrics
type HealthThreshold struct {
	WarningMin  *float64
	WarningMax  *float64
	CriticalMin *float64
	CriticalMax *float64
}

// ClusterHealth represents the overall cluster health
type ClusterHealth struct {
	OverallStatus HealthStatus
	Checks        map[string]*HealthCheck
	Issues        []HealthIssue
	LastUpdated   time.Time
	mu            sync.RWMutex
}

// HealthIssue represents a specific health issue
type HealthIssue struct {
	ID          string
	Component   string
	Severity    HealthStatus
	Title       string
	Description string
	FirstSeen   time.Time
	LastSeen    time.Time
	Count       int
	Resolved    bool
}

// HealthMonitor monitors cluster health and generates alerts
type HealthMonitor struct {
	client       dao.SlurmClient
	health       *ClusterHealth
	alertManager *AlertManager
	checks       map[string]HealthCheckFunc
	interval     time.Duration
	running      bool
	mu           sync.RWMutex
}

// HealthCheckFunc defines a function that performs a health check
type HealthCheckFunc func(client dao.SlurmClient) *HealthCheck

// NewHealthMonitor creates a new health monitor
func NewHealthMonitor(client dao.SlurmClient, interval time.Duration) *HealthMonitor {
	hm := &HealthMonitor{
		client:       client,
		interval:     interval,
		alertManager: NewAlertManager(),
		checks:       make(map[string]HealthCheckFunc),
		health: &ClusterHealth{
			OverallStatus: HealthStatusUnknown,
			Checks:        make(map[string]*HealthCheck),
			Issues:        []HealthIssue{},
		},
	}

	// Register default health checks
	hm.registerDefaultChecks()
	return hm
}

// Start begins health monitoring
func (hm *HealthMonitor) Start() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	if hm.running {
		return
	}

	hm.running = true
	go hm.monitorLoop()
}

// Stop stops health monitoring
func (hm *HealthMonitor) Stop() {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.running = false
}

// GetHealth returns the current cluster health
func (hm *HealthMonitor) GetHealth() *ClusterHealth {
	hm.health.mu.RLock()
	defer hm.health.mu.RUnlock()

	// Create a copy to avoid race conditions
	healthCopy := &ClusterHealth{
		OverallStatus: hm.health.OverallStatus,
		Checks:        make(map[string]*HealthCheck),
		Issues:        make([]HealthIssue, len(hm.health.Issues)),
		LastUpdated:   hm.health.LastUpdated,
	}

	for name, check := range hm.health.Checks {
		checkCopy := *check
		healthCopy.Checks[name] = &checkCopy
	}

	copy(healthCopy.Issues, hm.health.Issues)
	return healthCopy
}

// GetAlertManager returns the alert manager
func (hm *HealthMonitor) GetAlertManager() *AlertManager {
	return hm.alertManager
}

// monitorLoop runs the health monitoring loop
func (hm *HealthMonitor) monitorLoop() {
	ticker := time.NewTicker(hm.interval)
	defer ticker.Stop()

	// Initial check
	hm.performHealthChecks()

	for range ticker.C {
		hm.mu.RLock()
		running := hm.running
		hm.mu.RUnlock()

		if !running {
			return
		}

		hm.performHealthChecks()
	}
}

// performHealthChecks runs all registered health checks
func (hm *HealthMonitor) performHealthChecks() {
	hm.health.mu.Lock()
	defer hm.health.mu.Unlock()

	// Run all health checks
	for name, checkFunc := range hm.checks {
		check := checkFunc(hm.client)
		if check != nil {
			check.CheckCount++
			hm.health.Checks[name] = check

			// Generate alerts for critical/warning status
			if check.Status == HealthStatusCritical || check.Status == HealthStatusWarning {
				hm.generateAlert(check)
			}
		}
	}

	// Update overall status
	hm.updateOverallStatus()
	hm.health.LastUpdated = time.Now()
}

// updateOverallStatus calculates the overall cluster health status
func (hm *HealthMonitor) updateOverallStatus() {
	hasCritical := false
	hasWarning := false
	hasUnknown := false

	for _, check := range hm.health.Checks {
		switch check.Status {
		case HealthStatusCritical:
			hasCritical = true
		case HealthStatusWarning:
			hasWarning = true
		case HealthStatusUnknown:
			hasUnknown = true
		}
	}

	switch {
	case hasCritical:
		hm.health.OverallStatus = HealthStatusCritical
	case hasWarning:
		hm.health.OverallStatus = HealthStatusWarning
	case hasUnknown:
		hm.health.OverallStatus = HealthStatusUnknown
	default:
		hm.health.OverallStatus = HealthStatusHealthy
	}
}

// generateAlert creates an alert for a health check issue
func (hm *HealthMonitor) generateAlert(check *HealthCheck) {
	alert := &Alert{
		ID:           fmt.Sprintf("%s-%d", check.Name, time.Now().Unix()),
		Type:         AlertTypeHealth,
		Severity:     AlertSeverity(check.Status),
		Title:        fmt.Sprintf("Health Check Alert: %s", check.Name),
		Message:      check.Message,
		Component:    check.Name,
		Timestamp:    time.Now(),
		Acknowledged: false,
	}

	hm.alertManager.AddAlert(alert)
}

// registerDefaultChecks registers the default set of health checks
func (hm *HealthMonitor) registerDefaultChecks() {
	hm.checks["nodes"] = hm.checkNodes
	hm.checks["queue"] = hm.checkQueue
	hm.checks["utilization"] = hm.checkUtilization
}

// checkNodes performs a node availability health check
func (hm *HealthMonitor) checkNodes(client dao.SlurmClient) *HealthCheck {
	check := &HealthCheck{
		Name:        "nodes",
		Description: "Monitor node availability and health",
		LastCheck:   time.Now(),
		Threshold: HealthThreshold{
			WarningMax:  floatPtr(10.0),
			CriticalMax: floatPtr(25.0),
		},
	}

	nodeList, err := client.Nodes().List(&dao.ListNodesOptions{})
	if err != nil {
		check.Status = HealthStatusUnknown
		check.Message = fmt.Sprintf("Failed to get node list: %v", err)
		return check
	}

	total := len(nodeList.Nodes)
	if total == 0 {
		check.Status = HealthStatusCritical
		check.Message = "No nodes found in cluster"
		return check
	}

	downNodes, drainNodes := countNodeStates(nodeList.Nodes)
	unavailablePercent := float64(downNodes+drainNodes) * 100.0 / float64(total)

	hm.setCheckStatus(check, unavailablePercent, func(_ HealthStatus) string {
		return fmt.Sprintf("%.1f%% of nodes unavailable (%d down, %d drain out of %d total)",
			unavailablePercent, downNodes, drainNodes, total)
	}, func() string {
		return fmt.Sprintf("All nodes healthy (%d total, %d down, %d drain)", total, downNodes, drainNodes)
	})

	return check
}

// checkQueue performs a job queue depth health check
func (hm *HealthMonitor) checkQueue(client dao.SlurmClient) *HealthCheck {
	check := &HealthCheck{
		Name:        "queue",
		Description: "Monitor job queue depth and wait times",
		LastCheck:   time.Now(),
		Threshold: HealthThreshold{
			WarningMax:  floatPtr(100.0),
			CriticalMax: floatPtr(500.0),
		},
	}

	jobList, err := client.Jobs().List(&dao.ListJobsOptions{
		States: []string{dao.JobStatePending},
	})
	if err != nil {
		check.Status = HealthStatusUnknown
		check.Message = fmt.Sprintf("Failed to get job list: %v", err)
		return check
	}

	pendingJobs := float64(len(jobList.Jobs))

	hm.setCheckStatus(check, pendingJobs, func(status HealthStatus) string {
		if status != HealthStatusHealthy {
			return fmt.Sprintf("%.0f pending jobs (threshold: %.0f)",
				pendingJobs, *check.Threshold.CriticalMax)
		}
		return fmt.Sprintf("Queue healthy with %.0f pending jobs", pendingJobs)
	}, nil)

	return check
}

// checkUtilization performs a resource utilization health check
func (hm *HealthMonitor) checkUtilization(client dao.SlurmClient) *HealthCheck {
	check := &HealthCheck{
		Name:        "utilization",
		Description: "Monitor cluster resource utilization",
		LastCheck:   time.Now(),
		Threshold: HealthThreshold{
			WarningMax:  floatPtr(90.0),
			CriticalMax: floatPtr(95.0),
		},
	}

	infoMgr := client.Info()
	if infoMgr == nil {
		check.Status = HealthStatusUnknown
		check.Message = "Cluster metrics not available"
		return check
	}

	metrics, err := infoMgr.GetStats()
	if err != nil {
		check.Status = HealthStatusUnknown
		check.Message = fmt.Sprintf("Failed to get cluster metrics: %v", err)
		return check
	}

	cpuUtil := metrics.CPUUsage
	memUtil := metrics.MemoryUsage
	maxUtil := cpuUtil
	if memUtil > maxUtil {
		maxUtil = memUtil
	}

	hm.setCheckStatus(check, maxUtil, func(_ HealthStatus) string {
		return fmt.Sprintf("Resource utilization: CPU %.1f%%, Memory %.1f%%", cpuUtil, memUtil)
	}, nil)

	return check
}

// setCheckStatus sets the check status based on threshold comparison
func (hm *HealthMonitor) setCheckStatus(check *HealthCheck, value float64, criticalMsg func(HealthStatus) string, healthyMsg func() string) {
	switch {
	case check.Threshold.CriticalMax != nil && value > *check.Threshold.CriticalMax:
		check.Status = HealthStatusCritical
		check.Message = criticalMsg(HealthStatusCritical)
	case check.Threshold.WarningMax != nil && value > *check.Threshold.WarningMax:
		check.Status = HealthStatusWarning
		check.Message = criticalMsg(HealthStatusWarning)
	default:
		check.Status = HealthStatusHealthy
		if healthyMsg != nil {
			check.Message = healthyMsg()
		}
	}
}

// countNodeStates counts nodes in down and drain states
func countNodeStates(nodes []*dao.Node) (down, drain int) {
	for _, node := range nodes {
		if node == nil {
			continue
		}
		switch node.State {
		case dao.NodeStateDown:
			down++
		case dao.NodeStateDrain, dao.NodeStateDraining:
			drain++
		}
	}
	return down, drain
}

// floatPtr returns a pointer to a float64 value
func floatPtr(f float64) *float64 {
	return &f
}
