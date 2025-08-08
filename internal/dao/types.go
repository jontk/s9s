package dao

import (
	"strings"
	"time"
)

// Job represents a SLURM job
type Job struct {
	ID         string
	Name       string
	User       string
	Account    string
	Partition  string
	State      string
	Priority   float64
	QOS        string
	NodeCount  int
	TimeLimit  string
	TimeUsed   string
	SubmitTime time.Time
	StartTime  *time.Time
	EndTime    *time.Time
	NodeList   string
	Command    string
	WorkingDir string
	StdOut     string
	StdErr     string
	ExitCode   *int
}

// JobList represents a list of jobs
type JobList struct {
	Jobs  []*Job
	Total int
}

// ListJobsOptions contains options for listing jobs
type ListJobsOptions struct {
	States     []string
	Users      []string
	Partitions []string
	Accounts   []string
	Limit      int
	Offset     int
}

// JobSubmission represents a job submission request
type JobSubmission struct {
	Name         string            `json:"name"`
	Script       string            `json:"script,omitempty"`
	Command      string            `json:"command,omitempty"`
	Partition    string            `json:"partition,omitempty"`
	Account      string            `json:"account,omitempty"`
	QoS          string            `json:"qos,omitempty"`
	Nodes        int               `json:"nodes,omitempty"`
	CPUs         int               `json:"cpus,omitempty"` // Simplified from CPUsPerNode
	CPUsPerNode  int               `json:"cpus_per_node,omitempty"`
	Memory       string            `json:"memory,omitempty"`
	GPUs         int               `json:"gpus,omitempty"`
	TimeLimit    string            `json:"time_limit,omitempty"`
	WorkingDir   string            `json:"working_directory,omitempty"`
	OutputFile   string            `json:"output_file,omitempty"`
	ErrorFile    string            `json:"error_file,omitempty"`
	StdOut       string            `json:"stdout,omitempty"`
	StdErr       string            `json:"stderr,omitempty"`
	EmailNotify  bool              `json:"email_notify,omitempty"`
	Email        string            `json:"email,omitempty"`
	Environment  map[string]string `json:"environment,omitempty"`
	Dependencies []string          `json:"dependencies,omitempty"`
	ArraySpec    string            `json:"array,omitempty"`
	Exclusive    bool              `json:"exclusive,omitempty"`
	Requeue      bool              `json:"requeue,omitempty"`
}

// JobTemplate represents a predefined job template
type JobTemplate struct {
	Name          string
	Description   string
	JobSubmission JobSubmission
}

// Node represents a SLURM compute node
type Node struct {
	Name            string
	State           string
	Partitions      []string
	CPUsTotal       int
	CPUsAllocated   int
	CPUsIdle        int
	CPULoad         float64 // CPU load average
	MemoryTotal     int64   // in MB
	MemoryAllocated int64   // in MB
	MemoryFree      int64   // in MB
	Features        []string
	Reason          string
	ReasonTime      *time.Time
	AllocatedJobs   []string
}

// NodeList represents a list of nodes
type NodeList struct {
	Nodes []*Node
	Total int
}

// ListNodesOptions contains options for listing nodes
type ListNodesOptions struct {
	States     []string
	Partitions []string
	Features   []string
}

// Partition represents a SLURM partition
type Partition struct {
	Name        string
	State       string
	TotalNodes  int
	TotalCPUs   int
	DefaultTime string
	MaxTime     string
	QOS         []string
	Nodes       []string
}

// PartitionList represents a list of partitions
type PartitionList struct {
	Partitions []*Partition
}

// Reservation represents a SLURM reservation
type Reservation struct {
	Name      string
	State     string
	StartTime time.Time
	EndTime   time.Time
	Duration  time.Duration
	Nodes     []string
	NodeCount int
	CoreCount int
	Users     []string
	Accounts  []string
}

// ReservationList represents a list of reservations
type ReservationList struct {
	Reservations []*Reservation
}

// QueueInfo represents queue information for a partition
type QueueInfo struct {
	Partition   string
	PendingJobs int
	RunningJobs int
	TotalJobs   int
	AverageWait time.Duration
	LongestWait time.Duration
}

// ClusterMetrics represents overall cluster metrics
type ClusterMetrics struct {
	TotalJobs   int
	RunningJobs int
	PendingJobs int
	TotalNodes  int
	ActiveNodes int
	IdleNodes   int
	DownNodes   int
	CPUUsage    float64 // percentage
	MemoryUsage float64 // percentage
	LastUpdated time.Time
}

// JobState constants
const (
	JobStatePending     = "PENDING"
	JobStateRunning     = "RUNNING"
	JobStateCompleted   = "COMPLETED"
	JobStateFailed      = "FAILED"
	JobStateCancelled   = "CANCELLED"
	JobStateSuspended   = "SUSPENDED"
	JobStateCompleting  = "COMPLETING"
	JobStateConfiguring = "CONFIGURING"
	JobStatePreempted   = "PREEMPTED"
	JobStateTimeout     = "TIMEOUT"
)

// NodeState constants
const (
	NodeStateIdle        = "IDLE"
	NodeStateAllocated   = "ALLOCATED"
	NodeStateMixed       = "MIXED"
	NodeStateDown        = "DOWN"
	NodeStateDrain       = "DRAIN"
	NodeStateDraining    = "DRAINING"
	NodeStateReserved    = "RESERVED"
	NodeStateMaintenance = "MAINTENANCE"
)

// PartitionState constants
const (
	PartitionStateUp       = "UP"
	PartitionStateDown     = "DOWN"
	PartitionStateDrain    = "DRAIN"
	PartitionStateInactive = "INACTIVE"
)

// GetJobStateColor returns the color for a job state
func GetJobStateColor(state string) string {
	switch state {
	case JobStateRunning:
		return "green"
	case JobStatePending:
		return "yellow"
	case JobStateCompleted:
		return "cyan"
	case JobStateFailed:
		return "red"
	case JobStateCancelled:
		return "gray"
	case JobStateSuspended:
		return "orange"
	default:
		return "white"
	}
}

// GetNodeStateColor returns the color for a node state
func GetNodeStateColor(state string) string {
	// Handle compound states like "IDLE+DRAIN" - prioritize DRAIN over IDLE
	if strings.Contains(state, NodeStateDrain) || strings.Contains(state, NodeStateDraining) {
		return "red"
	}
	if strings.Contains(state, NodeStateDown) {
		return "red"
	}
	if strings.Contains(state, NodeStateReserved) {
		return "yellow"
	}
	if strings.Contains(state, NodeStateMaintenance) {
		return "orange"
	}
	if strings.Contains(state, NodeStateAllocated) || strings.Contains(state, NodeStateMixed) {
		return "blue"
	}
	if strings.Contains(state, NodeStateIdle) {
		return "green"
	}
	return "white"
}

// IsJobActive returns true if the job is in an active state
func IsJobActive(state string) bool {
	switch state {
	case JobStatePending, JobStateRunning, JobStateSuspended, JobStateConfiguring:
		return true
	default:
		return false
	}
}

// IsNodeAvailable returns true if the node is available for jobs
func IsNodeAvailable(state string) bool {
	switch state {
	case NodeStateIdle, NodeStateMixed:
		return true
	default:
		return false
	}
}

// GetPartitionStateColor returns the color for a partition state
func GetPartitionStateColor(state string) string {
	switch state {
	case PartitionStateUp:
		return "green"
	case PartitionStateDown:
		return "red"
	case PartitionStateDrain:
		return "orange"
	case PartitionStateInactive:
		return "gray"
	default:
		return "white"
	}
}

// QoS represents a SLURM Quality of Service
type QoS struct {
	Name                 string
	Priority             int
	PreemptMode          string
	Flags                []string
	GraceTime            int // in minutes
	MaxJobsPerUser       int
	MaxJobsPerAccount    int
	MaxSubmitJobsPerUser int
	MaxCPUsPerUser       int
	MaxNodesPerUser      int
	MaxWallTime          int   // in minutes
	MaxMemoryPerUser     int64 // in MB
	MinCPUs              int
	MinNodes             int
}

// QoSList represents a list of QoS entries
type QoSList struct {
	QoS   []*QoS
	Total int
}

// Account represents a SLURM account
type Account struct {
	Name         string
	Description  string
	Organization string
	Coordinators []string
	DefaultQoS   string
	QoSList      []string
	MaxJobs      int
	MaxNodes     int
	MaxCPUs      int
	MaxSubmit    int
	MaxWall      int // in minutes
	Parent       string
	Children     []string
}

// AccountList represents a list of accounts
type AccountList struct {
	Accounts []*Account
	Total    int
}

// User represents a SLURM user
type User struct {
	Name           string
	UID            int
	DefaultAccount string
	Accounts       []string
	AdminLevel     string
	DefaultQoS     string
	QoSList        []string
	MaxJobs        int
	MaxNodes       int
	MaxCPUs        int
	MaxSubmit      int
}

// UserList represents a list of users
type UserList struct {
	Users []*User
	Total int
}
