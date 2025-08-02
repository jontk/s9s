package dao

import (
	"time"
)

// Job represents a SLURM job
type Job struct {
	ID          string
	Name        string
	User        string
	Account     string
	Partition   string
	State       string
	Priority    float64
	QOS         string
	NodeCount   int
	TimeLimit   string
	TimeUsed    string
	SubmitTime  time.Time
	StartTime   *time.Time
	EndTime     *time.Time
	NodeList    string
	Command     string
	WorkingDir  string
	StdOut      string
	StdErr      string
	ExitCode    *int
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

// Node represents a SLURM compute node
type Node struct {
	Name            string
	State           string
	Partitions      []string
	CPUsTotal       int
	CPUsAllocated   int
	CPUsIdle        int
	MemoryTotal     int64 // in MB
	MemoryAllocated int64 // in MB
	MemoryFree      int64 // in MB
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
	Partition    string
	PendingJobs  int
	RunningJobs  int
	TotalJobs    int
	AverageWait  time.Duration
	LongestWait  time.Duration
}

// ClusterMetrics represents overall cluster metrics
type ClusterMetrics struct {
	TotalJobs       int
	RunningJobs     int
	PendingJobs     int
	TotalNodes      int
	ActiveNodes     int
	IdleNodes       int
	DownNodes       int
	CPUUsage        float64 // percentage
	MemoryUsage     float64 // percentage
	LastUpdated     time.Time
}

// JobState constants
const (
	JobStatePending    = "PENDING"
	JobStateRunning    = "RUNNING"
	JobStateCompleted  = "COMPLETED"
	JobStateFailed     = "FAILED"
	JobStateCancelled  = "CANCELLED"
	JobStateSuspended  = "SUSPENDED"
	JobStateCompleting = "COMPLETING"
	JobStateConfiguring = "CONFIGURING"
	JobStatePreempted  = "PREEMPTED"
	JobStateTimeout    = "TIMEOUT"
)

// NodeState constants
const (
	NodeStateIdle       = "IDLE"
	NodeStateAllocated  = "ALLOCATED"
	NodeStateMixed      = "MIXED"
	NodeStateDown       = "DOWN"
	NodeStateDrain      = "DRAIN"
	NodeStateDraining   = "DRAINING"
	NodeStateReserved   = "RESERVED"
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
	switch state {
	case NodeStateIdle:
		return "green"
	case NodeStateAllocated, NodeStateMixed:
		return "blue"
	case NodeStateDown, NodeStateDrain, NodeStateDraining:
		return "red"
	case NodeStateReserved:
		return "yellow"
	case NodeStateMaintenance:
		return "orange"
	default:
		return "white"
	}
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