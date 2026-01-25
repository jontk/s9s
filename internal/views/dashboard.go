package views

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/dao"
	"github.com/rivo/tview"
)

// DashboardView displays a comprehensive cluster overview
type DashboardView struct {
	*BaseView
	client       dao.SlurmClient
	mu           sync.RWMutex
	refreshTimer *time.Timer
	refreshRate  time.Duration
	container    *tview.Flex
	// TODO(lint): Review unused code - field app is unused
	// app          *tview.Application
	pages *tview.Pages

	// Dashboard components
	clusterOverview *tview.TextView
	jobsSummary     *tview.TextView
	nodesSummary    *tview.TextView
	partitionStatus *tview.TextView
	alertsPanel     *tview.TextView
	trendsPanel     *tview.TextView

	// Data cache
	clusterInfo    *dao.ClusterInfo
	clusterMetrics *dao.ClusterMetrics
	jobs           []*dao.Job
	nodes          []*dao.Node
	partitions     []*dao.Partition
	lastUpdate     time.Time
}

// SetPages sets the pages reference for modal handling
func (v *DashboardView) SetPages(pages *tview.Pages) {
	v.pages = pages
}

// NewDashboardView creates a new dashboard view
func NewDashboardView(client dao.SlurmClient) *DashboardView {
	v := &DashboardView{
		BaseView:    NewBaseView("dashboard", "Dashboard"),
		client:      client,
		refreshRate: 10 * time.Second, // More frequent updates for dashboard
	}

	// Create dashboard components
	v.clusterOverview = tview.NewTextView().SetDynamicColors(true)
	v.clusterOverview.SetTitle(" Cluster Overview ").SetBorder(true)

	v.jobsSummary = tview.NewTextView().SetDynamicColors(true)
	v.jobsSummary.SetTitle(" Jobs Summary ").SetBorder(true)

	v.nodesSummary = tview.NewTextView().SetDynamicColors(true)
	v.nodesSummary.SetTitle(" Nodes Summary ").SetBorder(true)

	v.partitionStatus = tview.NewTextView().SetDynamicColors(true)
	v.partitionStatus.SetTitle(" Partition Status ").SetBorder(true)

	v.alertsPanel = tview.NewTextView().SetDynamicColors(true)
	v.alertsPanel.SetTitle(" Alerts & Issues ").SetBorder(true)

	v.trendsPanel = tview.NewTextView().SetDynamicColors(true)
	v.trendsPanel.SetTitle(" Performance Trends ").SetBorder(true)

	// Create layout
	topRow := tview.NewFlex().
		AddItem(v.clusterOverview, 0, 1, false).
		AddItem(v.jobsSummary, 0, 1, false).
		AddItem(v.nodesSummary, 0, 1, false)

	middleRow := tview.NewFlex().
		AddItem(v.partitionStatus, 0, 2, false).
		AddItem(v.alertsPanel, 0, 1, false)

	bottomRow := tview.NewFlex().
		AddItem(v.trendsPanel, 0, 1, false)

	v.container = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(topRow, 0, 1, false).
		AddItem(middleRow, 0, 1, false).
		AddItem(bottomRow, 0, 1, false)

	return v
}

// Init initializes the dashboard view
func (v *DashboardView) Init(ctx context.Context) error {
	_ = v.BaseView.Init(ctx)
	return v.Refresh()
}

// Render returns the view's main component
func (v *DashboardView) Render() tview.Primitive {
	return v.container
}

// Refresh updates all dashboard data
func (v *DashboardView) Refresh() error {
	v.SetRefreshing(true)
	defer v.SetRefreshing(false)

	// Fetch all data concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, 5)

	// Fetch cluster info
	wg.Add(1)
	go func() {
		defer wg.Done()
		if info, err := v.client.ClusterInfo(); err == nil {
			v.mu.Lock()
			v.clusterInfo = info
			v.mu.Unlock()
		} else {
			errChan <- err
		}
	}()

	// Fetch cluster metrics
	wg.Add(1)
	go func() {
		defer wg.Done()
		if infoMgr := v.client.Info(); infoMgr != nil {
			if metrics, err := infoMgr.GetStats(); err == nil {
				v.mu.Lock()
				v.clusterMetrics = metrics
				v.mu.Unlock()
			} else {
				errChan <- err
			}
		}
	}()

	// Fetch jobs
	wg.Add(1)
	go func() {
		defer wg.Done()
		if jobList, err := v.client.Jobs().List(&dao.ListJobsOptions{Limit: 1000}); err == nil {
			v.mu.Lock()
			v.jobs = jobList.Jobs
			v.mu.Unlock()
		} else {
			errChan <- err
		}
	}()

	// Fetch nodes
	wg.Add(1)
	go func() {
		defer wg.Done()
		if nodeList, err := v.client.Nodes().List(&dao.ListNodesOptions{}); err == nil {
			v.mu.Lock()
			v.nodes = nodeList.Nodes
			v.mu.Unlock()
		} else {
			errChan <- err
		}
	}()

	// Fetch partitions
	wg.Add(1)
	go func() {
		defer wg.Done()
		if partitionList, err := v.client.Partitions().List(); err == nil {
			v.mu.Lock()
			v.partitions = partitionList.Partitions
			v.mu.Unlock()
		} else {
			errChan <- err
		}
	}()

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		v.SetLastError(err)
	}

	v.mu.Lock()
	v.lastUpdate = time.Now()
	v.mu.Unlock()

	// Update all dashboard components
	v.updateClusterOverview()
	v.updateJobsSummary()
	v.updateNodesSummary()
	v.updatePartitionStatus()
	v.updateAlertsPanel()
	v.updateTrendsPanel()

	// Schedule next refresh
	v.scheduleRefresh()

	return nil
}

// Stop stops the view
func (v *DashboardView) Stop() error {
	if v.refreshTimer != nil {
		v.refreshTimer.Stop()
	}
	return nil
}

// Hints returns keyboard hints
func (v *DashboardView) Hints() []string {
	return []string{
		"[yellow]j[white] Jobs View",
		"[yellow]n[white] Nodes View",
		"[yellow]p[white] Partitions View",
		"[yellow]a[white] Analytics",
		"[yellow]R[white] Refresh",
		"[yellow]h[white] Health Check",
	}
}

// OnKey handles keyboard events
func (v *DashboardView) OnKey(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyRune {
		switch event.Rune() {
		case 'R':
			go func() { _ = v.Refresh() }()
			return nil
		case 'j', 'J':
			// TODO: Switch to jobs view
			return nil
		case 'n', 'N':
			// TODO: Switch to nodes view
			return nil
		case 'p', 'P':
			// TODO: Switch to partitions view
			return nil
		case 'a', 'A':
			v.showAdvancedAnalytics()
			return nil
		case 'h', 'H':
			v.showHealthCheck()
			return nil
		}
	}
	return event
}

// OnFocus handles focus events
func (v *DashboardView) OnFocus() error {
	// Refresh when gaining focus
	go func() { _ = v.Refresh() }()
	return nil
}

// OnLoseFocus handles loss of focus
func (v *DashboardView) OnLoseFocus() error {
	return nil
}

// updateClusterOverview updates the cluster overview panel
func (v *DashboardView) updateClusterOverview() {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var content strings.Builder

	// Cluster information
	if v.clusterInfo != nil {
		content.WriteString(fmt.Sprintf("[yellow]Cluster:[white] %s\n", v.clusterInfo.Name))
		content.WriteString(fmt.Sprintf("[yellow]Version:[white] %s\n", v.clusterInfo.Version))
		content.WriteString(fmt.Sprintf("[yellow]Endpoint:[white] %s\n", v.clusterInfo.Endpoint))
	}

	content.WriteString("\n")

	// Overall health
	if v.clusterMetrics != nil {
		healthStatus := v.calculateHealthStatus()
		healthColor := v.getHealthColor(healthStatus)
		content.WriteString(fmt.Sprintf("[yellow]Health:[white] [%s]%s[white]\n", healthColor, healthStatus))

		// Key metrics
		content.WriteString(fmt.Sprintf("[yellow]CPU Usage:[white] %.1f%%\n", v.clusterMetrics.CPUUsage))
		content.WriteString(fmt.Sprintf("[yellow]Memory Usage:[white] %.1f%%\n", v.clusterMetrics.MemoryUsage))

		// Utilization bars
		cpuBar := v.createUtilizationBar(v.clusterMetrics.CPUUsage)
		memBar := v.createUtilizationBar(v.clusterMetrics.MemoryUsage)
		content.WriteString(fmt.Sprintf("[yellow]CPU:[white] %s\n", cpuBar))
		content.WriteString(fmt.Sprintf("[yellow]Memory:[white] %s\n", memBar))
	}

	content.WriteString(fmt.Sprintf("\n[gray]Last Updated: %s[white]", v.lastUpdate.Format("15:04:05")))

	v.clusterOverview.SetText(content.String())
}

// updateJobsSummary updates the jobs summary panel
func (v *DashboardView) updateJobsSummary() {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var content strings.Builder

	if len(v.jobs) == 0 {
		content.WriteString("[gray]No job data available[white]")
		v.jobsSummary.SetText(content.String())
		return
	}

	// Job state counts
	jobStats := make(map[string]int)
	totalJobs := len(v.jobs)
	var totalWaitTime time.Duration
	waitingJobs := 0
	now := time.Now()

	for _, job := range v.jobs {
		jobStats[job.State]++
		if job.State == dao.JobStatePending {
			waitTime := now.Sub(job.SubmitTime)
			totalWaitTime += waitTime
			waitingJobs++
		}
	}

	content.WriteString(fmt.Sprintf("[yellow]Total Jobs:[white] %d\n\n", totalJobs))

	// Job states with colors
	content.WriteString(fmt.Sprintf("[green]Running:[white] %d\n", jobStats[dao.JobStateRunning]))
	content.WriteString(fmt.Sprintf("[yellow]Pending:[white] %d\n", jobStats[dao.JobStatePending]))
	content.WriteString(fmt.Sprintf("[cyan]Completed:[white] %d\n", jobStats[dao.JobStateCompleted]))
	content.WriteString(fmt.Sprintf("[red]Failed:[white] %d\n", jobStats[dao.JobStateFailed]))
	content.WriteString(fmt.Sprintf("[orange]Cancelled:[white] %d\n", jobStats[dao.JobStateCancelled]))

	// Job state visualization
	if totalJobs > 0 {
		content.WriteString("\n[yellow]Distribution:[white]\n")
		runningPct := float64(jobStats[dao.JobStateRunning]) * 100.0 / float64(totalJobs)
		pendingPct := float64(jobStats[dao.JobStatePending]) * 100.0 / float64(totalJobs)
		content.WriteString(fmt.Sprintf("Running: %s %.1f%%\n", v.createMiniBar(runningPct, "green"), runningPct))
		content.WriteString(fmt.Sprintf("Pending: %s %.1f%%\n", v.createMiniBar(pendingPct, "yellow"), pendingPct))
	}

	// Average wait time
	if waitingJobs > 0 {
		avgWait := totalWaitTime / time.Duration(waitingJobs)
		content.WriteString(fmt.Sprintf("\n[yellow]Avg Wait:[white] %s", FormatDurationDetailed(avgWait)))
	}

	v.jobsSummary.SetText(content.String())
}

// updateNodesSummary updates the nodes summary panel
func (v *DashboardView) updateNodesSummary() {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var content strings.Builder

	if len(v.nodes) == 0 {
		content.WriteString("[gray]No node data available[white]")
		v.nodesSummary.SetText(content.String())
		return
	}

	// Node state counts
	nodeStats := make(map[string]int)
	totalNodes := len(v.nodes)
	totalCPUs := 0
	allocatedCPUs := 0
	totalMemory := int64(0)
	allocatedMemory := int64(0)

	for _, node := range v.nodes {
		nodeStats[node.State]++
		totalCPUs += node.CPUsTotal
		allocatedCPUs += node.CPUsAllocated
		totalMemory += node.MemoryTotal
		allocatedMemory += node.MemoryAllocated
	}

	content.WriteString(fmt.Sprintf("[yellow]Total Nodes:[white] %d\n\n", totalNodes))

	// Node states with colors
	content.WriteString(fmt.Sprintf("[green]Idle:[white] %d\n", nodeStats[dao.NodeStateIdle]))
	content.WriteString(fmt.Sprintf("[blue]Allocated:[white] %d\n", nodeStats[dao.NodeStateAllocated]))
	content.WriteString(fmt.Sprintf("[yellow]Mixed:[white] %d\n", nodeStats[dao.NodeStateMixed]))
	content.WriteString(fmt.Sprintf("[red]Down:[white] %d\n", nodeStats[dao.NodeStateDown]))
	content.WriteString(fmt.Sprintf("[orange]Drain:[white] %d\n", nodeStats[dao.NodeStateDrain]))

	// Resource utilization
	if totalCPUs > 0 {
		cpuUtilization := float64(allocatedCPUs) * 100.0 / float64(totalCPUs)
		content.WriteString(fmt.Sprintf("\n[yellow]CPU Util:[white] %s %.1f%%\n",
			v.createMiniBar(cpuUtilization, getUtilizationColor(cpuUtilization)), cpuUtilization))
	}

	if totalMemory > 0 {
		memUtilization := float64(allocatedMemory) * 100.0 / float64(totalMemory)
		content.WriteString(fmt.Sprintf("[yellow]Mem Util:[white] %s %.1f%%\n",
			v.createMiniBar(memUtilization, getUtilizationColor(memUtilization)), memUtilization))
	}

	// Availability
	availableNodes := nodeStats[dao.NodeStateIdle] + nodeStats[dao.NodeStateMixed]
	if totalNodes > 0 {
		availabilityPct := float64(availableNodes) * 100.0 / float64(totalNodes)
		content.WriteString(fmt.Sprintf("[yellow]Available:[white] %d (%.1f%%)", availableNodes, availabilityPct))
	}

	v.nodesSummary.SetText(content.String())
}

// updatePartitionStatus updates the partition status panel
func (v *DashboardView) updatePartitionStatus() {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var content strings.Builder

	if len(v.partitions) == 0 {
		content.WriteString("[gray]No partition data available[white]")
		v.partitionStatus.SetText(content.String())
		return
	}

	content.WriteString(fmt.Sprintf("[yellow]Total Partitions:[white] %d\n\n", len(v.partitions)))

	// Sort partitions by node count for display
	sortedPartitions := make([]*dao.Partition, len(v.partitions))
	copy(sortedPartitions, v.partitions)
	sort.Slice(sortedPartitions, func(i, j int) bool {
		return sortedPartitions[i].TotalNodes > sortedPartitions[j].TotalNodes
	})

	// Show top partitions
	maxDisplay := 8
	if len(sortedPartitions) < maxDisplay {
		maxDisplay = len(sortedPartitions)
	}

	content.WriteString("[yellow]Partition         Nodes   CPUs    State[white]\n")
	content.WriteString("──────────────────────────────────────────\n")

	for i := 0; i < maxDisplay; i++ {
		partition := sortedPartitions[i]
		stateColor := dao.GetPartitionStateColor(partition.State)

		name := partition.Name
		if len(name) > 16 {
			name = name[:13] + "..."
		}
		name = fmt.Sprintf("%-16s", name)

		content.WriteString(fmt.Sprintf("%s %5d %7d [%s]%s[white]\n",
			name, partition.TotalNodes, partition.TotalCPUs, stateColor, partition.State))
	}

	if len(sortedPartitions) > maxDisplay {
		content.WriteString(fmt.Sprintf("... and %d more partitions\n", len(sortedPartitions)-maxDisplay))
	}

	v.partitionStatus.SetText(content.String())
}

// updateAlertsPanel updates the alerts and issues panel
func (v *DashboardView) updateAlertsPanel() {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var content strings.Builder
	alerts := v.generateAlerts()

	if len(alerts) == 0 {
		content.WriteString("[green]✓ No issues detected[white]\n")
		content.WriteString("\n[gray]System operating normally[white]")
	} else {
		content.WriteString(fmt.Sprintf("[yellow]%d issue(s) detected:[white]\n\n", len(alerts)))

		for _, alert := range alerts {
			content.WriteString(fmt.Sprintf("[%s]%s %s[white]\n", alert.Color, alert.Icon, alert.Message))
		}
	}

	v.alertsPanel.SetText(content.String())
}

// updateTrendsPanel updates the performance trends panel
func (v *DashboardView) updateTrendsPanel() {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var content strings.Builder

	content.WriteString("[yellow]Performance Overview[white]\n\n")

	// Job throughput trends (mock data for now)
	content.WriteString("[teal]Job Throughput (24h):[white]\n")
	content.WriteString("Jobs/Hour: █▇▆▅▄▃▂▁▂▃▄▅▆▇█ (Trend: ↗)\n")

	content.WriteString("\n[teal]Resource Efficiency:[white]\n")
	if v.clusterMetrics != nil {
		efficiency := (v.clusterMetrics.CPUUsage + v.clusterMetrics.MemoryUsage) / 2
		efficiencyBar := v.createMiniBar(efficiency, getUtilizationColor(efficiency))
		content.WriteString(fmt.Sprintf("Overall: %s %.1f%%\n", efficiencyBar, efficiency))
	}

	content.WriteString("\n[teal]System Health Score:[white]\n")
	healthScore := v.calculateHealthScore()
	scoreColor := v.getHealthColor(fmt.Sprintf("%.0f%%", healthScore))
	scoreBar := v.createMiniBar(healthScore, scoreColor)
	content.WriteString(fmt.Sprintf("Score: %s %.1f%%", scoreBar, healthScore))

	v.trendsPanel.SetText(content.String())
}

// Helper functions

func (v *DashboardView) scheduleRefresh() {
	// Remove automatic refresh scheduling to prevent memory leak
	// Refresh will be handled by the main app refresh timer
}

func (v *DashboardView) calculateHealthStatus() string {
	score := v.calculateHealthScore()

	switch {
	case score >= 90:
		return "EXCELLENT"
	case score >= 75:
		return "GOOD"
	case score >= 60:
		return "FAIR"
	case score >= 40:
		return "POOR"
	default:
		return "CRITICAL"
	}
}

func (v *DashboardView) calculateHealthScore() float64 {
	score := 100.0

	score -= v.calculateNodeHealthDeduction()
	score -= v.calculateJobHealthDeduction()
	score -= v.calculateResourceHealthDeduction()

	if score < 0 {
		score = 0
	}

	return score
}

// calculateNodeHealthDeduction calculates health score deduction for down nodes
func (v *DashboardView) calculateNodeHealthDeduction() float64 {
	if len(v.nodes) == 0 {
		return 0
	}

	downNodes := 0
	for _, node := range v.nodes {
		if node.State == dao.NodeStateDown {
			downNodes++
		}
	}

	downPercent := float64(downNodes) * 100.0 / float64(len(v.nodes))
	return downPercent * 2 // Each down node % costs 2 points
}

// calculateJobHealthDeduction calculates health score deduction for failed jobs
func (v *DashboardView) calculateJobHealthDeduction() float64 {
	if len(v.jobs) == 0 {
		return 0
	}

	failedJobs := 0
	for _, job := range v.jobs {
		if job.State == dao.JobStateFailed {
			failedJobs++
		}
	}

	failedPercent := float64(failedJobs) * 100.0 / float64(len(v.jobs))
	return failedPercent // Each failed job % costs 1 point
}

// calculateResourceHealthDeduction calculates health score deduction for high resource utilization
func (v *DashboardView) calculateResourceHealthDeduction() float64 {
	if v.clusterMetrics == nil {
		return 0
	}

	deduction := 0.0
	if v.clusterMetrics.CPUUsage > 95 {
		deduction += 10
	}
	if v.clusterMetrics.MemoryUsage > 95 {
		deduction += 10
	}

	return deduction
}

func (v *DashboardView) getHealthColor(status string) string {
	switch status {
	case "EXCELLENT":
		return "green"
	case "GOOD":
		return "cyan"
	case "FAIR":
		return "yellow"
	case "POOR":
		return "orange"
	case "CRITICAL":
		return "red"
	default:
		return "white"
	}
}

func (v *DashboardView) createUtilizationBar(percentage float64) string {
	return v.createMiniBar(percentage, getUtilizationColor(percentage))
}

func (v *DashboardView) createMiniBar(percentage float64, color string) string {
	barLength := 10
	filled := int(percentage / 100.0 * float64(barLength))

	if filled > barLength {
		filled = barLength
	}

	var bar strings.Builder
	bar.WriteString(fmt.Sprintf("[%s]", color))

	for i := 0; i < filled; i++ {
		bar.WriteString("▰")
	}

	bar.WriteString("[gray]")
	for i := filled; i < barLength; i++ {
		bar.WriteString("▱")
	}

	bar.WriteString("[white]")
	return bar.String()
}

func getUtilizationColor(percentage float64) string {
	switch {
	case percentage < 50:
		return "green"
	case percentage < 80:
		return "yellow"
	default:
		return "red"
	}
}

// Alert represents a system alert
type Alert struct {
	Level   string
	Icon    string
	Color   string
	Message string
}

func (v *DashboardView) generateAlerts() []Alert {
	var alerts []Alert

	// Check each alert condition and collect non-empty alerts
	if alert := v.checkDownNodesAlert(); alert != nil {
		alerts = append(alerts, *alert)
	}

	if alert := v.checkResourceUtilizationAlert(); alert != nil {
		alerts = append(alerts, *alert)
	}

	if alert := v.checkCPUAlertIfHigh(); alert != nil {
		alerts = append(alerts, *alert)
	}

	if alert := v.checkLongWaitingJobsAlert(); alert != nil {
		alerts = append(alerts, *alert)
	}

	if alert := v.checkFailedJobsAlert(); alert != nil {
		alerts = append(alerts, *alert)
	}

	return alerts
}

func (v *DashboardView) checkDownNodesAlert() *Alert {
	if len(v.nodes) == 0 {
		return nil
	}

	downNodes := 0
	for _, node := range v.nodes {
		if node.State == dao.NodeStateDown {
			downNodes++
		}
	}

	if downNodes == 0 {
		return nil
	}

	return &Alert{
		Level:   "WARNING",
		Icon:    "⚠",
		Color:   "yellow",
		Message: fmt.Sprintf("%d node(s) are down", downNodes),
	}
}

func (v *DashboardView) checkResourceUtilizationAlert() *Alert {
	if v.clusterMetrics == nil || v.clusterMetrics.MemoryUsage <= 90 {
		return nil
	}

	return &Alert{
		Level:   "WARNING",
		Icon:    "⚠",
		Color:   "orange",
		Message: fmt.Sprintf("High memory utilization: %.1f%%", v.clusterMetrics.MemoryUsage),
	}
}

func (v *DashboardView) checkCPUAlertIfHigh() *Alert {
	if v.clusterMetrics == nil || v.clusterMetrics.CPUUsage <= 90 {
		return nil
	}

	return &Alert{
		Level:   "WARNING",
		Icon:    "⚠",
		Color:   "orange",
		Message: fmt.Sprintf("High CPU utilization: %.1f%%", v.clusterMetrics.CPUUsage),
	}
}

func (v *DashboardView) checkLongWaitingJobsAlert() *Alert {
	if len(v.jobs) == 0 {
		return nil
	}

	longWaitingJobs := 0
	now := time.Now()
	for _, job := range v.jobs {
		if job.State == dao.JobStatePending && now.Sub(job.SubmitTime) > 24*time.Hour {
			longWaitingJobs++
		}
	}

	if longWaitingJobs == 0 {
		return nil
	}

	return &Alert{
		Level:   "INFO",
		Icon:    "ℹ",
		Color:   "cyan",
		Message: fmt.Sprintf("%d job(s) waiting >24h", longWaitingJobs),
	}
}

func (v *DashboardView) checkFailedJobsAlert() *Alert {
	if len(v.jobs) == 0 {
		return nil
	}

	failedJobs := 0
	for _, job := range v.jobs {
		if job.State == dao.JobStateFailed {
			failedJobs++
		}
	}

	if failedJobs <= 10 {
		return nil
	}

	return &Alert{
		Level:   "ERROR",
		Icon:    "✗",
		Color:   "red",
		Message: fmt.Sprintf("%d failed jobs detected", failedJobs),
	}
}

// showAdvancedAnalytics shows advanced analytics modal
func (v *DashboardView) showAdvancedAnalytics() {
	// Create analytics content
	analytics := v.generateAdvancedAnalytics()

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(analytics).
		SetScrollable(true)

	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, true).
		AddItem(tview.NewTextView().SetText("Press ESC to close | R to refresh"), 1, 0, false)

	modal.SetBorder(true).
		SetTitle(" Advanced Cluster Analytics ").
		SetTitleAlign(tview.AlignCenter)

	// Create centered modal layout
	centeredModal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(modal, 0, 8, true).
			AddItem(nil, 0, 1, false), 0, 8, true).
		AddItem(nil, 0, 1, false)

	// Handle keys
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			if v.pages != nil {
				v.pages.RemovePage("advanced-analytics")
			}
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'R' || event.Rune() == 'r' {
				// Refresh and update the display
				go func() {
					_ = v.Refresh()
					newAnalytics := v.generateAdvancedAnalytics()
					textView.SetText(newAnalytics)
				}()
				return nil
			}
		}
		return event
	})

	if v.pages != nil {
		v.pages.AddPage("advanced-analytics", centeredModal, true, true)
	}
}

// generateAdvancedAnalytics generates advanced analytics content
func (v *DashboardView) generateAdvancedAnalytics() string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var analytics strings.Builder

	analytics.WriteString("[yellow]Advanced Cluster Analytics[white]\n\n")

	v.appendEfficiencyAnalysis(&analytics)
	v.appendJobAnalysis(&analytics)
	v.appendNodeAnalysis(&analytics)
	v.appendRecommendations(&analytics)
	analytics.WriteString(fmt.Sprintf("\n[gray]Generated: %s[white]", time.Now().Format("2006-01-02 15:04:05")))

	return analytics.String()
}

// appendEfficiencyAnalysis appends resource efficiency analysis section
func (v *DashboardView) appendEfficiencyAnalysis(w *strings.Builder) {
	w.WriteString("[teal]Resource Efficiency Analysis:[white]\n")
	if v.clusterMetrics == nil {
		return
	}

	cpuEfficiency := v.clusterMetrics.CPUUsage
	memEfficiency := v.clusterMetrics.MemoryUsage
	overallEfficiency := (cpuEfficiency + memEfficiency) / 2

	w.WriteString(fmt.Sprintf("[yellow]  CPU Efficiency:[white] %.1f%% %s\n", cpuEfficiency, v.getEfficiencyAssessment(cpuEfficiency)))
	w.WriteString(fmt.Sprintf("[yellow]  Memory Efficiency:[white] %.1f%% %s\n", memEfficiency, v.getEfficiencyAssessment(memEfficiency)))
	w.WriteString(fmt.Sprintf("[yellow]  Overall Efficiency:[white] %.1f%% %s\n", overallEfficiency, v.getEfficiencyAssessment(overallEfficiency)))
}

// appendJobAnalysis appends job analysis section
func (v *DashboardView) appendJobAnalysis(w *strings.Builder) {
	if len(v.jobs) == 0 {
		return
	}

	w.WriteString("\n[teal]Job Analysis:[white]\n")
	v.appendJobStateDistribution(w)
	v.appendWaitTimeAnalysis(w)
}

// appendJobStateDistribution appends job state distribution analysis
func (v *DashboardView) appendJobStateDistribution(w *strings.Builder) {
	stateStats := make(map[string]int)
	totalJobs := len(v.jobs)

	for _, job := range v.jobs {
		stateStats[job.State]++
	}

	for state, count := range stateStats {
		percentage := float64(count) * 100.0 / float64(totalJobs)
		color := dao.GetJobStateColor(state)
		w.WriteString(fmt.Sprintf("[yellow]  %s:[white] [%s]%d[white] (%.1f%%)\n", state, color, count, percentage))
	}
}

// appendWaitTimeAnalysis appends wait time analysis section
func (v *DashboardView) appendWaitTimeAnalysis(w *strings.Builder) {
	now := time.Now()
	var waitTimes []time.Duration
	for _, job := range v.jobs {
		if job.State == dao.JobStatePending {
			waitTimes = append(waitTimes, now.Sub(job.SubmitTime))
		}
	}

	if len(waitTimes) == 0 {
		return
	}

	w.WriteString("\n[teal]Wait Time Analysis:[white]\n")

	var totalWait, maxWait time.Duration
	for _, wait := range waitTimes {
		totalWait += wait
		if wait > maxWait {
			maxWait = wait
		}
	}
	avgWait := totalWait / time.Duration(len(waitTimes))

	w.WriteString(fmt.Sprintf("[yellow]  Average Wait Time:[white] %s\n", FormatDurationDetailed(avgWait)))
	w.WriteString(fmt.Sprintf("[yellow]  Maximum Wait Time:[white] %s\n", FormatDurationDetailed(maxWait)))
	w.WriteString(fmt.Sprintf("[yellow]  Jobs Waiting >1h:[white] %d\n", v.countJobsWaitingLongerThan(time.Hour)))
	w.WriteString(fmt.Sprintf("[yellow]  Jobs Waiting >24h:[white] %d\n", v.countJobsWaitingLongerThan(24*time.Hour)))
}

// appendNodeAnalysis appends node analysis section
func (v *DashboardView) appendNodeAnalysis(w *strings.Builder) {
	if len(v.nodes) == 0 {
		return
	}

	w.WriteString("\n[teal]Node Analysis:[white]\n")

	stateStats := make(map[string]int)
	totalCPUs, allocatedCPUs := 0, 0
	var totalMemory, allocatedMemory int64

	for _, node := range v.nodes {
		stateStats[node.State]++
		totalCPUs += node.CPUsTotal
		allocatedCPUs += node.CPUsAllocated
		totalMemory += node.MemoryTotal
		allocatedMemory += node.MemoryAllocated
	}

	for state, count := range stateStats {
		percentage := float64(count) * 100.0 / float64(len(v.nodes))
		color := dao.GetNodeStateColor(state)
		w.WriteString(fmt.Sprintf("[yellow]  %s:[white] [%s]%d[white] (%.1f%%)\n", state, color, count, percentage))
	}

	if totalCPUs > 0 {
		cpuUtilization := float64(allocatedCPUs) * 100.0 / float64(totalCPUs)
		w.WriteString(fmt.Sprintf("[yellow]  CPU Utilization:[white] %.1f%% (%d/%d cores)\n", cpuUtilization, allocatedCPUs, totalCPUs))
	}

	if totalMemory > 0 {
		memUtilization := float64(allocatedMemory) * 100.0 / float64(totalMemory)
		w.WriteString(fmt.Sprintf("[yellow]  Memory Utilization:[white] %.1f%% (%s/%s)\n",
			memUtilization, FormatMemory(allocatedMemory), FormatMemory(totalMemory)))
	}
}

// appendRecommendations appends recommendations section
func (v *DashboardView) appendRecommendations(w *strings.Builder) {
	w.WriteString("\n[teal]Recommendations:[white]\n")
	recommendations := v.generateRecommendations()
	for _, rec := range recommendations {
		w.WriteString(fmt.Sprintf("[yellow]  •[white] %s\n", rec))
	}
}

// showHealthCheck shows health check modal
func (v *DashboardView) showHealthCheck() {
	// Create health check content
	healthCheck := v.generateHealthCheck()

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(healthCheck).
		SetScrollable(true)

	modal := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 0, 1, true).
		AddItem(tview.NewTextView().SetText("Press ESC to close | R to refresh"), 1, 0, false)

	modal.SetBorder(true).
		SetTitle(" Cluster Health Check ").
		SetTitleAlign(tview.AlignCenter)

	// Create centered modal layout
	centeredModal := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(modal, 0, 8, true).
			AddItem(nil, 0, 1, false), 0, 8, true).
		AddItem(nil, 0, 1, false)

	// Handle keys
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			if v.pages != nil {
				v.pages.RemovePage("health-check")
			}
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'R' || event.Rune() == 'r' {
				// Refresh and update the display
				go func() {
					_ = v.Refresh()
					newHealthCheck := v.generateHealthCheck()
					textView.SetText(newHealthCheck)
				}()
				return nil
			}
		}
		return event
	})

	if v.pages != nil {
		v.pages.AddPage("health-check", centeredModal, true, true)
	}
}

// generateHealthCheck generates health check content
func (v *DashboardView) generateHealthCheck() string {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var health strings.Builder

	health.WriteString("[yellow]Cluster Health Check Report[white]\n\n")

	healthScore := v.calculateHealthScore()
	healthStatus := v.calculateHealthStatus()
	healthColor := v.getHealthColor(healthStatus)

	health.WriteString(fmt.Sprintf("[yellow]Overall Health Score:[white] [%s]%.1f/100 (%s)[white]\n\n",
		healthColor, healthScore, healthStatus))

	// Detailed checks
	health.WriteString("[teal]Component Health Checks:[white]\n")
	health.WriteString(v.generateNodeHealthSection())
	health.WriteString(v.generateJobQueueHealthSection())
	health.WriteString(v.generateResourceUtilizationSection())
	health.WriteString(v.generatePartitionHealthSection())

	health.WriteString(fmt.Sprintf("\n[gray]Health check completed: %s[white]", time.Now().Format("2006-01-02 15:04:05")))

	return health.String()
}

func (v *DashboardView) generateNodeHealthSection() string {
	if len(v.nodes) == 0 {
		return ""
	}

	var output strings.Builder
	downNodes := 0
	drainNodes := 0

	for _, node := range v.nodes {
		switch node.State {
		case dao.NodeStateDown:
			downNodes++
		case dao.NodeStateDrain, dao.NodeStateDraining:
			drainNodes++
		}
	}

	output.WriteString("[yellow]Nodes:[white]\n")
	if downNodes == 0 {
		output.WriteString("  [green]✓[white] All nodes operational\n")
	} else {
		output.WriteString(fmt.Sprintf("  [red]✗[white] %d node(s) down\n", downNodes))
	}

	if drainNodes > 0 {
		output.WriteString(fmt.Sprintf("  [yellow]⚠[white] %d node(s) draining\n", drainNodes))
	}

	return output.String()
}

func (v *DashboardView) generateJobQueueHealthSection() string {
	if len(v.jobs) == 0 {
		return ""
	}

	var output strings.Builder
	failedJobs := 0
	stuckJobs := 0
	now := time.Now()

	for _, job := range v.jobs {
		if job.State == dao.JobStateFailed {
			failedJobs++
		} else if job.State == dao.JobStatePending && now.Sub(job.SubmitTime) > 24*time.Hour {
			stuckJobs++
		}
	}

	output.WriteString("\n[yellow]Job Queue:[white]\n")
	if failedJobs == 0 {
		output.WriteString("  [green]✓[white] No failed jobs detected\n")
	} else {
		output.WriteString(fmt.Sprintf("  [red]✗[white] %d failed job(s)\n", failedJobs))
	}

	if stuckJobs == 0 {
		output.WriteString("  [green]✓[white] No stuck jobs detected\n")
	} else {
		output.WriteString(fmt.Sprintf("  [yellow]⚠[white] %d job(s) waiting >24h\n", stuckJobs))
	}

	return output.String()
}

func (v *DashboardView) generateResourceUtilizationSection() string {
	if v.clusterMetrics == nil {
		return ""
	}

	var output strings.Builder
	output.WriteString("\n[yellow]Resource Utilization:[white]\n")

	if v.clusterMetrics.CPUUsage < 95 {
		output.WriteString(fmt.Sprintf("  [green]✓[white] CPU utilization normal (%.1f%%)\n", v.clusterMetrics.CPUUsage))
	} else {
		output.WriteString(fmt.Sprintf("  [red]✗[white] CPU utilization critical (%.1f%%)\n", v.clusterMetrics.CPUUsage))
	}

	if v.clusterMetrics.MemoryUsage < 95 {
		output.WriteString(fmt.Sprintf("  [green]✓[white] Memory utilization normal (%.1f%%)\n", v.clusterMetrics.MemoryUsage))
	} else {
		output.WriteString(fmt.Sprintf("  [red]✗[white] Memory utilization critical (%.1f%%)\n", v.clusterMetrics.MemoryUsage))
	}

	return output.String()
}

func (v *DashboardView) generatePartitionHealthSection() string {
	if len(v.partitions) == 0 {
		return ""
	}

	var output strings.Builder
	downPartitions := 0

	for _, partition := range v.partitions {
		if partition.State == dao.PartitionStateDown {
			downPartitions++
		}
	}

	output.WriteString("\n[yellow]Partitions:[white]\n")
	if downPartitions == 0 {
		output.WriteString("  [green]✓[white] All partitions operational\n")
	} else {
		output.WriteString(fmt.Sprintf("  [red]✗[white] %d partition(s) down\n", downPartitions))
	}

	return output.String()
}

// Helper functions for analytics

func (v *DashboardView) getEfficiencyAssessment(percentage float64) string {
	switch {
	case percentage < 30:
		return "[red](Under-utilized)[white]"
	case percentage < 70:
		return "[yellow](Moderate)[white]"
	case percentage < 95:
		return "[green](Well-utilized)[white]"
	default:
		return "[red](Over-utilized)[white]"
	}
}

func (v *DashboardView) countJobsWaitingLongerThan(duration time.Duration) int {
	count := 0
	now := time.Now()
	for _, job := range v.jobs {
		if job.State == dao.JobStatePending && now.Sub(job.SubmitTime) > duration {
			count++
		}
	}
	return count
}

func (v *DashboardView) generateRecommendations() []string {
	var recommendations []string

	v.addResourceUtilizationRecommendations(&recommendations)
	v.addJobQueueRecommendations(&recommendations)
	v.addNodeHealthRecommendations(&recommendations)

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "System is operating optimally - no immediate action required")
	}

	return recommendations
}

func (v *DashboardView) addResourceUtilizationRecommendations(recommendations *[]string) {
	if v.clusterMetrics == nil {
		return
	}

	if v.clusterMetrics.CPUUsage < 30 {
		*recommendations = append(*recommendations,
			"Consider consolidating workloads or reducing cluster size due to low CPU utilization")
	} else if v.clusterMetrics.CPUUsage > 90 {
		*recommendations = append(*recommendations,
			"Consider expanding CPU capacity due to high utilization")
	}

	if v.clusterMetrics.MemoryUsage > 90 {
		*recommendations = append(*recommendations,
			"Consider expanding memory capacity due to high utilization")
	}
}

func (v *DashboardView) addJobQueueRecommendations(recommendations *[]string) {
	if len(v.jobs) == 0 {
		return
	}

	longWaitingJobs := v.countJobsWaitingLongerThan(24 * time.Hour)
	if longWaitingJobs > 10 {
		*recommendations = append(*recommendations,
			"High number of jobs waiting >24h - review job priorities and resource allocation")
	}

	failedJobs := v.countFailedJobs()
	if failedJobs > 5 {
		*recommendations = append(*recommendations,
			"Multiple failed jobs detected - review job scripts and resource requirements")
	}
}

func (v *DashboardView) addNodeHealthRecommendations(recommendations *[]string) {
	if len(v.nodes) == 0 {
		return
	}

	downNodes := v.countDownNodes()
	if downNodes > 0 {
		*recommendations = append(*recommendations,
			"Down nodes detected - investigate hardware issues and consider maintenance")
	}
}

func (v *DashboardView) countDownNodes() int {
	count := 0
	for _, node := range v.nodes {
		if node.State == dao.NodeStateDown {
			count++
		}
	}
	return count
}

func (v *DashboardView) countFailedJobs() int {
	count := 0
	for _, job := range v.jobs {
		if job.State == dao.JobStateFailed {
			count++
		}
	}
	return count
}
