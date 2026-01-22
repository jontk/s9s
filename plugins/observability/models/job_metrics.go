package models

import (
	"fmt"
	"strings"
	"time"
)

// JobMetrics represents metrics for a SLURM job
type JobMetrics struct {
	JobID         string            `json:"job_id"`
	JobName       string            `json:"job_name"`
	User          string            `json:"user"`
	State         string            `json:"state"`
	NodeList      []string          `json:"node_list"`
	StartTime     time.Time         `json:"start_time"`
	AllocatedCPUs int               `json:"allocated_cpus"`
	AllocatedMem  uint64            `json:"allocated_mem"` // in bytes
	Resources     ResourceMetrics   `json:"resources"`
	Efficiency    EfficiencyMetrics `json:"efficiency"`
	CgroupPath    string            `json:"cgroup_path"`
	Labels        map[string]string `json:"labels"` // Prometheus labels
	LastUpdate    time.Time         `json:"last_update"`
}

// EfficiencyMetrics tracks resource efficiency
type EfficiencyMetrics struct {
	CPUEfficiency     float64 `json:"cpu_efficiency"`     // Actual vs Allocated CPU %
	MemEfficiency     float64 `json:"mem_efficiency"`     // Actual vs Allocated Memory %
	OverallEfficiency float64 `json:"overall_efficiency"` // Combined efficiency score
	CPUWasted         float64 `json:"cpu_wasted"`         // Wasted CPU cores
	MemWasted         uint64  `json:"mem_wasted"`         // Wasted memory bytes
}

// JobMetricsCollector collects and manages job metrics
type JobMetricsCollector struct {
	jobs          map[string]*JobMetrics
	cgroupPattern string // Pattern for cgroup paths
}

// NewJobMetricsCollector creates a new job metrics collector
func NewJobMetricsCollector(cgroupPattern string) *JobMetricsCollector {
	return &JobMetricsCollector{
		jobs:          make(map[string]*JobMetrics),
		cgroupPattern: cgroupPattern,
	}
}

// UpdateFromPrometheus updates job metrics from Prometheus data
func (jmc *JobMetricsCollector) UpdateFromPrometheus(jobID string, metrics map[string]*TimeSeries) {
	job, exists := jmc.jobs[jobID]
	if !exists {
		job = &JobMetrics{
			JobID:  jobID,
			Labels: make(map[string]string),
		}
		jmc.jobs[jobID] = job
	}

	// Update resource metrics
	job.Resources = jmc.extractResourceMetrics(metrics)
	job.LastUpdate = time.Now()

	// Calculate efficiency if we have allocation info
	if job.AllocatedCPUs > 0 || job.AllocatedMem > 0 {
		job.Efficiency = jmc.calculateEfficiency(job)
	}

	// Extract labels from first metric
	for _, ts := range metrics {
		if len(ts.Labels) > 0 {
			job.Labels = ts.Labels
			break
		}
	}

	// Update cgroup path
	job.CgroupPath = fmt.Sprintf(jmc.cgroupPattern, jobID)
}

// extractResourceMetrics extracts ResourceMetrics from job-specific Prometheus data
func (jmc *JobMetricsCollector) extractResourceMetrics(metrics map[string]*TimeSeries) ResourceMetrics {
	rm := ResourceMetrics{
		Timestamp: time.Now(),
	}

	// CPU metrics
	if cpuUsage, ok := metrics["job_cpu_usage"]; ok && cpuUsage.Latest() != nil {
		rm.CPU.Usage = cpuUsage.Latest().Value
	}
	if cpuThrottled, ok := metrics["job_cpu_throttled"]; ok && cpuThrottled.Latest() != nil {
		rm.CPU.Throttled = cpuThrottled.Latest().Value
	}

	// Memory metrics
	if memUsage, ok := metrics["job_memory_usage"]; ok && memUsage.Latest() != nil {
		rm.Memory.Used = uint64(memUsage.Latest().Value)
	}
	if memLimit, ok := metrics["job_memory_limit"]; ok && memLimit.Latest() != nil {
		// This is the cgroup limit, which might be different from SLURM allocation
		limit := uint64(memLimit.Latest().Value)
		rm.Memory.Limit = limit // Store the limit
		if limit > 0 {
			rm.Memory.Usage = float64(rm.Memory.Used) / float64(limit) * 100
		}
	}
	if memCache, ok := metrics["job_memory_cache"]; ok && memCache.Latest() != nil {
		rm.Memory.Cache = uint64(memCache.Latest().Value)
	}
	if memRSS, ok := metrics["job_memory_rss"]; ok && memRSS.Latest() != nil {
		// RSS is often more accurate for actual memory usage
		rm.Memory.Used = uint64(memRSS.Latest().Value)
	}

	return rm
}

// UpdateJobInfo updates SLURM job information
func (jmc *JobMetricsCollector) UpdateJobInfo(jobID string, info JobInfo) {
	job, exists := jmc.jobs[jobID]
	if !exists {
		job = &JobMetrics{
			JobID:  jobID,
			Labels: make(map[string]string),
		}
		jmc.jobs[jobID] = job
	}

	job.JobName = info.JobName
	job.User = info.User
	job.State = info.State
	job.NodeList = info.NodeList
	job.StartTime = info.StartTime
	job.AllocatedCPUs = info.AllocatedCPUs
	job.AllocatedMem = info.AllocatedMem

	// Recalculate efficiency if we have metrics
	if job.Resources.CPU.Usage > 0 || job.Resources.Memory.Used > 0 {
		job.Efficiency = jmc.calculateEfficiency(job)
	}
}

// JobInfo contains SLURM job information
type JobInfo struct {
	JobName       string
	User          string
	State         string
	NodeList      []string
	StartTime     time.Time
	AllocatedCPUs int
	AllocatedMem  uint64
}

// calculateEfficiency calculates resource efficiency metrics
func (jmc *JobMetricsCollector) calculateEfficiency(job *JobMetrics) EfficiencyMetrics {
	eff := EfficiencyMetrics{}

	// CPU efficiency
	if job.AllocatedCPUs > 0 {
		// Convert CPU usage percentage to cores
		actualCores := job.Resources.CPU.Usage / 100.0
		eff.CPUEfficiency = (actualCores / float64(job.AllocatedCPUs)) * 100
		eff.CPUWasted = float64(job.AllocatedCPUs) - actualCores

		// Cap efficiency at 100%
		if eff.CPUEfficiency > 100 {
			eff.CPUEfficiency = 100
			eff.CPUWasted = 0
		}
	}

	// Memory efficiency
	if job.AllocatedMem > 0 {
		eff.MemEfficiency = (float64(job.Resources.Memory.Used) / float64(job.AllocatedMem)) * 100
		if job.Resources.Memory.Used < job.AllocatedMem {
			eff.MemWasted = job.AllocatedMem - job.Resources.Memory.Used
		}

		// Cap efficiency at 100%
		if eff.MemEfficiency > 100 {
			eff.MemEfficiency = 100
			eff.MemWasted = 0
		}
	}

	// Overall efficiency (weighted average)
	if eff.CPUEfficiency > 0 || eff.MemEfficiency > 0 {
		cpuWeight := 0.6 // CPU is often more important
		memWeight := 0.4
		eff.OverallEfficiency = (eff.CPUEfficiency*cpuWeight + eff.MemEfficiency*memWeight)
	}

	return eff
}

// GetJob returns metrics for a specific job
func (jmc *JobMetricsCollector) GetJob(jobID string) (*JobMetrics, bool) {
	job, exists := jmc.jobs[jobID]
	return job, exists
}

// GetAllJobs returns all job metrics
func (jmc *JobMetricsCollector) GetAllJobs() map[string]*JobMetrics {
	return jmc.jobs
}

// GetActiveJobs returns only running jobs
func (jmc *JobMetricsCollector) GetActiveJobs() map[string]*JobMetrics {
	active := make(map[string]*JobMetrics)
	for id, job := range jmc.jobs {
		if job.State == "RUNNING" || job.State == "R" {
			active[id] = job
		}
	}
	return active
}

// RemoveJob removes a job from tracking
func (jmc *JobMetricsCollector) RemoveJob(jobID string) {
	delete(jmc.jobs, jobID)
}

// GetInefficiientJobs returns jobs with low resource efficiency
func (jmc *JobMetricsCollector) GetInefficiientJobs(threshold float64) []*JobMetrics {
	var inefficient []*JobMetrics

	for _, job := range jmc.jobs {
		if job.State == "RUNNING" && job.Efficiency.OverallEfficiency < threshold {
			inefficient = append(inefficient, job)
		}
	}

	return inefficient
}

// GetJobsSummary returns a summary of jobs by state
func (jmc *JobMetricsCollector) GetJobsSummary() map[string]int {
	summary := make(map[string]int)

	for _, job := range jmc.jobs {
		state := job.State
		if state == "" {
			state = "unknown"
		}
		summary[state]++
	}

	return summary
}

// GetAggregateMetrics returns aggregate metrics across all jobs
func (jmc *JobMetricsCollector) GetAggregateMetrics() *AggregateJobMetrics {
	agg := &AggregateJobMetrics{
		Timestamp: time.Now(),
	}

	runningJobs := 0

	for _, job := range jmc.jobs {
		if job.State != "RUNNING" && job.State != "R" {
			continue
		}

		runningJobs++

		// Sum allocations
		agg.TotalAllocatedCPUs += job.AllocatedCPUs
		agg.TotalAllocatedMem += job.AllocatedMem

		// Sum actual usage
		actualCores := job.Resources.CPU.Usage / 100.0
		agg.TotalUsedCPUs += actualCores
		agg.TotalUsedMem += job.Resources.Memory.Used

		// Sum wasted resources
		agg.TotalWastedCPUs += job.Efficiency.CPUWasted
		agg.TotalWastedMem += job.Efficiency.MemWasted

		// Track efficiency
		agg.TotalEfficiencyScore += job.Efficiency.OverallEfficiency
	}

	// Calculate averages
	if runningJobs > 0 {
		agg.RunningJobs = runningJobs
		agg.AverageEfficiency = agg.TotalEfficiencyScore / float64(runningJobs)

		if agg.TotalAllocatedCPUs > 0 {
			agg.CPUUtilization = (agg.TotalUsedCPUs / float64(agg.TotalAllocatedCPUs)) * 100
		}
		if agg.TotalAllocatedMem > 0 {
			agg.MemUtilization = (float64(agg.TotalUsedMem) / float64(agg.TotalAllocatedMem)) * 100
		}
	}

	return agg
}

// AggregateJobMetrics represents cluster-wide job aggregate metrics
type AggregateJobMetrics struct {
	Timestamp            time.Time `json:"timestamp"`
	RunningJobs          int       `json:"running_jobs"`
	TotalAllocatedCPUs   int       `json:"total_allocated_cpus"`
	TotalAllocatedMem    uint64    `json:"total_allocated_mem"`
	TotalUsedCPUs        float64   `json:"total_used_cpus"`
	TotalUsedMem         uint64    `json:"total_used_mem"`
	TotalWastedCPUs      float64   `json:"total_wasted_cpus"`
	TotalWastedMem       uint64    `json:"total_wasted_mem"`
	CPUUtilization       float64   `json:"cpu_utilization"`
	MemUtilization       float64   `json:"mem_utilization"`
	AverageEfficiency    float64   `json:"average_efficiency"`
	TotalEfficiencyScore float64   `json:"total_efficiency_score"`
}

// FormatJobMetrics formats job metrics for display
func FormatJobMetrics(job *JobMetrics) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("Job: %s", job.JobID))
	if job.JobName != "" {
		parts = append(parts, fmt.Sprintf("Name: %s", job.JobName))
	}
	parts = append(parts, fmt.Sprintf("User: %s", job.User))
	parts = append(parts, fmt.Sprintf("State: %s", job.State))

	if job.AllocatedCPUs > 0 {
		parts = append(parts, fmt.Sprintf("CPUs: %.1f/%d (%.1f%% eff)",
			job.Resources.CPU.Usage/100.0,
			job.AllocatedCPUs,
			job.Efficiency.CPUEfficiency))
	}

	if job.AllocatedMem > 0 {
		parts = append(parts, fmt.Sprintf("Memory: %s/%s (%.1f%% eff)",
			FormatValue(float64(job.Resources.Memory.Used), "bytes"),
			FormatValue(float64(job.AllocatedMem), "bytes"),
			job.Efficiency.MemEfficiency))
	}

	if job.Efficiency.OverallEfficiency > 0 {
		parts = append(parts, fmt.Sprintf("Overall Efficiency: %.1f%%",
			job.Efficiency.OverallEfficiency))
	}

	return strings.Join(parts, " | ")
}

// GetEfficiencyColor returns a color based on efficiency percentage
func GetEfficiencyColor(efficiency float64) string {
	switch {
	case efficiency >= 80:
		return "green"
	case efficiency >= 50:
		return "yellow"
	case efficiency >= 20:
		return "orange"
	default:
		return "red"
	}
}

// IsEfficient returns whether a job is using resources efficiently
func (j *JobMetrics) IsEfficient(threshold float64) bool {
	return j.Efficiency.OverallEfficiency >= threshold
}
