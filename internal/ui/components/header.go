package components

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jontk/s9s/internal/dao"
	"github.com/rivo/tview"
)

// Header displays cluster status and navigation information
type Header struct {
	*tview.TextView
	mu            sync.RWMutex // Protects all header fields
	clusterInfo   *dao.ClusterInfo
	metrics       *dao.ClusterMetrics
	currentView   string
	views         []string
	lastUpdate    time.Time
	refreshTicker *time.Ticker
	alertsBadge   *AlertsBadge
}

// NewHeader creates a new header component
func NewHeader() *Header {
	h := &Header{
		TextView:   tview.NewTextView(),
		views:      []string{},
		lastUpdate: time.Now(),
	}

	h.TextView.
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	// Start refresh ticker
	h.refreshTicker = time.NewTicker(5 * time.Second)
	go h.updateLoop()

	return h
}

// SetClusterInfo sets the cluster information
func (h *Header) SetClusterInfo(info *dao.ClusterInfo) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clusterInfo = info
	h.updateDisplay()
}

// SetMetrics sets the cluster metrics
func (h *Header) SetMetrics(metrics *dao.ClusterMetrics) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.metrics = metrics
	h.lastUpdate = time.Now()
	h.updateDisplay()
}

// SetCurrentView sets the current active view
func (h *Header) SetCurrentView(view string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.currentView = view
	h.updateDisplay()
}

// SetViews sets the available views for navigation display
func (h *Header) SetViews(views []string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.views = views
	h.updateDisplay()
}

// SetAlertsBadge sets the alerts badge for display in the header
func (h *Header) SetAlertsBadge(badge *AlertsBadge) {
	h.alertsBadge = badge
	h.updateDisplay()
}

// Stop stops the header update ticker
func (h *Header) Stop() {
	if h.refreshTicker != nil {
		h.refreshTicker.Stop()
	}
}

// updateLoop runs the periodic update loop
func (h *Header) updateLoop() {
	for range h.refreshTicker.C {
		h.mu.Lock()
		h.updateDisplay()
		h.mu.Unlock()
	}
}

// updateDisplay updates the header display
func (h *Header) updateDisplay() {
	var content strings.Builder

	// Build header lines
	h.appendTitleLine(&content)
	h.appendNavigationAndMetricsLine(&content)

	h.SetText(content.String())
}

// appendTitleLine appends the title and cluster info line
func (h *Header) appendTitleLine(content *strings.Builder) {
	content.WriteString("[white::b]S9S - SLURM Terminal UI[white::-]")

	if h.clusterInfo != nil {
		content.WriteString(fmt.Sprintf(" | [cyan]%s[white] (%s)",
			h.clusterInfo.Name, h.clusterInfo.Version))
	}

	// Add current time
	now := time.Now()
	content.WriteString(fmt.Sprintf(" | %s", now.Format("15:04:05")))

	h.appendAlertsBadge(content)
	content.WriteString("\n")
}

// appendAlertsBadge appends the alerts badge if available
func (h *Header) appendAlertsBadge(content *strings.Builder) {
	if h.alertsBadge != nil {
		h.alertsBadge.update()
		badgeText := h.alertsBadge.text.GetText(true)
		if badgeText != "" {
			content.WriteString(" | ")
			content.WriteString(badgeText)
		}
	}
}

// appendNavigationAndMetricsLine appends the navigation and metrics line
func (h *Header) appendNavigationAndMetricsLine(content *strings.Builder) {
	h.appendViewsList(content)
	h.appendMetrics(content)
	h.appendLastUpdateTime(content)
}

// appendViewsList appends the views list
func (h *Header) appendViewsList(content *strings.Builder) {
	if len(h.views) == 0 {
		return
	}

	content.WriteString("[yellow]Views:[white] ")
	for i, view := range h.views {
		if i > 0 {
			content.WriteString(" | ")
		}

		if view == h.currentView {
			content.WriteString(fmt.Sprintf("[black:yellow] %s [white:-:-]", strings.ToUpper(view)))
		} else {
			content.WriteString(fmt.Sprintf("[cyan]%s[white]", strings.ToUpper(view)))
		}
	}
}

// appendMetrics appends cluster metrics if available
func (h *Header) appendMetrics(content *strings.Builder) {
	if h.metrics == nil {
		return
	}

	if len(h.views) > 0 {
		content.WriteString(" | ")
	}
	content.WriteString(h.formatMetrics())
}

// appendLastUpdateTime appends the last update time
func (h *Header) appendLastUpdateTime(content *strings.Builder) {
	age := time.Since(h.lastUpdate)
	if age > time.Minute {
		content.WriteString(fmt.Sprintf(" | [red]Last update: %s ago[white]", formatAge(age)))
	} else {
		content.WriteString(fmt.Sprintf(" | [green]Updated: %s ago[white]", formatAge(age)))
	}
}

// formatMetrics formats cluster metrics for display
func (h *Header) formatMetrics() string {
	m := h.metrics

	// Job statistics
	jobStats := fmt.Sprintf("[green]%d R[white]/[yellow]%d P[white]/[cyan]%d T[white]",
		m.RunningJobs, m.PendingJobs, m.TotalJobs)

	// Node statistics
	nodeStats := fmt.Sprintf("[green]%d A[white]/[blue]%d I[white]/[red]%d D[white]/[gray]%d T[white]",
		m.ActiveNodes, m.IdleNodes, m.DownNodes, m.TotalNodes)

	// Resource utilization
	cpuBar := createMiniBar(m.CPUUsage)
	memBar := createMiniBar(m.MemoryUsage)

	return fmt.Sprintf("Jobs: %s | Nodes: %s | CPU: %s %.1f%% | Mem: %s %.1f%%",
		jobStats, nodeStats, cpuBar, m.CPUUsage, memBar, m.MemoryUsage)
}

// createMiniBar creates a mini usage bar
func createMiniBar(percentage float64) string {
	barLength := 4
	filled := int(percentage / 100.0 * float64(barLength))

	var color string
	switch {
	case percentage < 50:
		color = "green"
	case percentage < 80:
		color = "yellow"
	default:
		color = "red"
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

// formatAge formats a duration as a readable age string
func formatAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
