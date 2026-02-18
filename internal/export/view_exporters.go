package export

import (
	"fmt"
	"strings"
	"time"

	"github.com/jontk/s9s/internal/dao"
)

// JobsTableData converts a slice of jobs to TableData for export.
func JobsTableData(jobs []*dao.Job) *TableData {
	headers := []string{
		"ID", "Name", "User", "Account", "State", "Partition",
		"Nodes", "Time Used", "Time Limit", "Priority", "Submit Time",
	}

	rows := make([][]string, len(jobs))
	for i, j := range jobs {
		priority := fmt.Sprintf("%.0f", j.Priority)
		submitTime := ""
		if !j.SubmitTime.IsZero() {
			submitTime = j.SubmitTime.Format("2006-01-02 15:04:05")
		}
		timeUsed := j.TimeUsed
		if timeUsed == "" && j.StartTime != nil {
			timeUsed = formatDuration(time.Since(*j.StartTime))
		}
		rows[i] = []string{
			j.ID, j.Name, j.User, j.Account, j.State, j.Partition,
			fmt.Sprintf("%d", j.NodeCount),
			timeUsed,
			j.TimeLimit,
			priority,
			submitTime,
		}
	}
	return &TableData{Title: "Jobs", Headers: headers, Rows: rows, ExportedAt: time.Now()}
}

// NodesTableData converts a slice of nodes to TableData for export.
func NodesTableData(nodes []*dao.Node) *TableData {
	headers := []string{
		"Name", "State", "Partitions", "CPUs Total", "CPUs Alloc",
		"CPUs Idle", "CPU Load", "Memory Total (MB)", "Memory Alloc (MB)",
		"Memory Free (MB)", "Features", "Reason",
	}

	rows := make([][]string, len(nodes))
	for i, n := range nodes {
		cpuLoad := ""
		if n.CPULoad >= 0 {
			cpuLoad = fmt.Sprintf("%.2f", n.CPULoad)
		}
		rows[i] = []string{
			n.Name,
			n.State,
			strings.Join(n.Partitions, ","),
			fmt.Sprintf("%d", n.CPUsTotal),
			fmt.Sprintf("%d", n.CPUsAllocated),
			fmt.Sprintf("%d", n.CPUsIdle),
			cpuLoad,
			fmt.Sprintf("%d", n.MemoryTotal),
			fmt.Sprintf("%d", n.MemoryAllocated),
			fmt.Sprintf("%d", n.MemoryFree),
			strings.Join(n.Features, ","),
			n.Reason,
		}
	}
	return &TableData{Title: "Nodes", Headers: headers, Rows: rows, ExportedAt: time.Now()}
}

// PartitionsTableData converts a slice of partitions to TableData for export.
func PartitionsTableData(partitions []*dao.Partition) *TableData {
	headers := []string{
		"Name", "State", "Total Nodes", "Total CPUs",
		"Default Time", "Max Time", "QoS", "Nodes",
	}

	rows := make([][]string, len(partitions))
	for i, p := range partitions {
		rows[i] = []string{
			p.Name,
			p.State,
			fmt.Sprintf("%d", p.TotalNodes),
			fmt.Sprintf("%d", p.TotalCPUs),
			p.DefaultTime,
			p.MaxTime,
			strings.Join(p.QOS, ","),
			strings.Join(p.Nodes, ","),
		}
	}
	return &TableData{Title: "Partitions", Headers: headers, Rows: rows, ExportedAt: time.Now()}
}

// ReservationsTableData converts a slice of reservations to TableData for export.
func ReservationsTableData(reservations []*dao.Reservation) *TableData {
	headers := []string{
		"Name", "State", "Start Time", "End Time", "Duration",
		"Nodes", "Node Count", "Core Count", "Users", "Accounts",
	}

	rows := make([][]string, len(reservations))
	for i, r := range reservations {
		startTime := r.StartTime.Format("2006-01-02 15:04:05")
		endTime := r.EndTime.Format("2006-01-02 15:04:05")
		duration := ""
		if r.Duration > 0 {
			duration = r.Duration.String()
		}
		rows[i] = []string{
			r.Name,
			r.State,
			startTime,
			endTime,
			duration,
			strings.Join(r.Nodes, ","),
			fmt.Sprintf("%d", r.NodeCount),
			fmt.Sprintf("%d", r.CoreCount),
			strings.Join(r.Users, ","),
			strings.Join(r.Accounts, ","),
		}
	}
	return &TableData{Title: "Reservations", Headers: headers, Rows: rows, ExportedAt: time.Now()}
}

// QoSTableData converts a slice of QoS entries to TableData for export.
func QoSTableData(qosList []*dao.QoS) *TableData {
	headers := []string{
		"Name", "Priority", "Preempt Mode", "Flags",
		"Max Jobs/User", "Max Jobs/Account", "Max Submit Jobs/User",
		"Max CPUs/User", "Max Nodes/User", "Max Wall Time (min)",
		"Max Memory/User (MB)", "Min CPUs", "Min Nodes",
	}

	rows := make([][]string, len(qosList))
	for i, q := range qosList {
		rows[i] = []string{
			q.Name,
			fmt.Sprintf("%d", q.Priority),
			q.PreemptMode,
			strings.Join(q.Flags, ","),
			fmt.Sprintf("%d", q.MaxJobsPerUser),
			fmt.Sprintf("%d", q.MaxJobsPerAccount),
			fmt.Sprintf("%d", q.MaxSubmitJobsPerUser),
			fmt.Sprintf("%d", q.MaxCPUsPerUser),
			fmt.Sprintf("%d", q.MaxNodesPerUser),
			fmt.Sprintf("%d", q.MaxWallTime),
			fmt.Sprintf("%d", q.MaxMemoryPerUser),
			fmt.Sprintf("%d", q.MinCPUs),
			fmt.Sprintf("%d", q.MinNodes),
		}
	}
	return &TableData{Title: "QoS", Headers: headers, Rows: rows, ExportedAt: time.Now()}
}

// AccountsTableData converts a slice of accounts to TableData for export.
func AccountsTableData(accounts []*dao.Account) *TableData {
	headers := []string{
		"Name", "Description", "Organization", "Parent",
		"Default QoS", "QoS List", "Coordinators",
		"Max Jobs", "Max Nodes", "Max CPUs", "Max Submit", "Max Wall (min)",
		"Children",
	}

	rows := make([][]string, len(accounts))
	for i, a := range accounts {
		rows[i] = []string{
			a.Name,
			a.Description,
			a.Organization,
			a.Parent,
			a.DefaultQoS,
			strings.Join(a.QoSList, ","),
			strings.Join(a.Coordinators, ","),
			fmt.Sprintf("%d", a.MaxJobs),
			fmt.Sprintf("%d", a.MaxNodes),
			fmt.Sprintf("%d", a.MaxCPUs),
			fmt.Sprintf("%d", a.MaxSubmit),
			fmt.Sprintf("%d", a.MaxWall),
			strings.Join(a.Children, ","),
		}
	}
	return &TableData{Title: "Accounts", Headers: headers, Rows: rows, ExportedAt: time.Now()}
}

// UsersTableData converts a slice of users to TableData for export.
func UsersTableData(users []*dao.User) *TableData {
	headers := []string{
		"Name", "UID", "Default Account", "Accounts",
		"Admin Level", "Default QoS", "QoS List",
		"Max Jobs", "Max Nodes", "Max CPUs", "Max Submit",
	}

	rows := make([][]string, len(users))
	for i, u := range users {
		rows[i] = []string{
			u.Name,
			fmt.Sprintf("%d", u.UID),
			u.DefaultAccount,
			strings.Join(u.Accounts, ","),
			u.AdminLevel,
			u.DefaultQoS,
			strings.Join(u.QoSList, ","),
			fmt.Sprintf("%d", u.MaxJobs),
			fmt.Sprintf("%d", u.MaxNodes),
			fmt.Sprintf("%d", u.MaxCPUs),
			fmt.Sprintf("%d", u.MaxSubmit),
		}
	}
	return &TableData{Title: "Users", Headers: headers, Rows: rows, ExportedAt: time.Now()}
}

// formatDuration formats a duration in HH:MM:SS style (same as the views package).
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}
