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
	Constraints  string            `json:"constraints,omitempty"`
	NTasks       int               `json:"ntasks,omitempty"`
	NTasksPerNode int              `json:"ntasks_per_node,omitempty"`
	Gres         string            `json:"gres,omitempty"`
	Hold         bool              `json:"hold,omitempty"`
	Reservation  string            `json:"reservation,omitempty"`
	Licenses     string            `json:"licenses,omitempty"`
	Wckey        string            `json:"wckey,omitempty"`
	ExcludeNodes string            `json:"exclude_nodes,omitempty"`
	Priority     int               `json:"priority,omitempty"`
	Nice         int               `json:"nice,omitempty"`
	MemoryPerCPU     string `json:"memory_per_cpu,omitempty"`
	BeginTime        string `json:"begin_time,omitempty"`
	Comment          string `json:"comment,omitempty"`
	Distribution     string `json:"distribution,omitempty"`
	Prefer           string `json:"prefer,omitempty"`
	RequiredNodes    string `json:"required_nodes,omitempty"`
	StandardInput    string `json:"standard_input,omitempty"`
	Container        string `json:"container,omitempty"`
	ThreadsPerCore   int    `json:"threads_per_core,omitempty"`
	TasksPerCore     int    `json:"tasks_per_core,omitempty"`
	TasksPerSocket   int    `json:"tasks_per_socket,omitempty"`
	SocketsPerNode   int    `json:"sockets_per_node,omitempty"`
	MaximumNodes     int    `json:"maximum_nodes,omitempty"`
	MaximumCPUs      int    `json:"maximum_cpus,omitempty"`
	MinimumCPUsPerNode int  `json:"minimum_cpus_per_node,omitempty"`
	TimeMinimum      string `json:"time_minimum,omitempty"`
	Contiguous       bool   `json:"contiguous,omitempty"`
	Overcommit       bool   `json:"overcommit,omitempty"`
	KillOnNodeFail   bool   `json:"kill_on_node_fail,omitempty"`
	WaitAllNodes     bool   `json:"wait_all_nodes,omitempty"`
	OpenMode         string `json:"open_mode,omitempty"`
	TRESPerTask        string `json:"tres_per_task,omitempty"`
	TRESPerSocket      string `json:"tres_per_socket,omitempty"`
	Signal             string `json:"signal,omitempty"`              // --signal=[B:]<sig_num>[@<sig_time>]
	TmpDiskPerNode     int    `json:"tmp_disk_per_node,omitempty"`  // --tmp (MB)
	Deadline           string `json:"deadline,omitempty"`            // --deadline
	NTasksPerTRES      int    `json:"ntasks_per_tres,omitempty"`    // --ntasks-per-gpu
	CPUBinding         string `json:"cpu_binding,omitempty"`         // --cpu-bind
	CPUFrequency       string `json:"cpu_frequency,omitempty"`      // --cpu-freq
	Network            string `json:"network,omitempty"`             // --network
	X11                string `json:"x11,omitempty"`                 // --x11
	Immediate          bool   `json:"immediate,omitempty"`           // --immediate
	BurstBuffer        string `json:"burst_buffer,omitempty"`        // --bb
	BatchFeatures      string `json:"batch_features,omitempty"`      // --batch
	TRESBind           string `json:"tres_bind,omitempty"`           // --tres-bind
	TRESFreq           string `json:"tres_freq,omitempty"`           // --tres-freq
	CoreSpecification  int    `json:"core_specification,omitempty"`  // --core-spec
	ThreadSpecification int   `json:"thread_specification,omitempty"` // --thread-spec
	MemoryBinding      string `json:"memory_binding,omitempty"`      // --mem-bind
	MinimumCPUs         int    `json:"minimum_cpus,omitempty"`         // --mincpus (total CPU floor)
	TRESPerJob          string `json:"tres_per_job,omitempty"`         // --tres-per-job
	CPUsPerTRES         string `json:"cpus_per_tres,omitempty"`        // --cpus-per-gpu
	MemoryPerTRES       string `json:"memory_per_tres,omitempty"`      // --mem-per-gpu
	Argv                string `json:"argv,omitempty"`                 // script arguments (space-separated)
	Flags               string `json:"flags,omitempty"`                // comma-separated job flags (SPREAD_JOB, etc)
	ProfileTypes        string `json:"profile,omitempty"`              // --profile (ENERGY, LUSTRE, NETWORK, TASK)
	CPUBindingFlags     string `json:"cpu_binding_flags,omitempty"`    // --cpu-bind flags (verbose, etc)
	MemoryBindingType   string `json:"memory_binding_type,omitempty"`  // --mem-bind type (local, rank, etc)
	RequiredSwitches    int    `json:"required_switches,omitempty"`    // --switches count
	WaitForSwitch       int    `json:"wait_for_switch,omitempty"`      // --switches timeout (seconds)
	ClusterConstraint   string `json:"cluster_constraint,omitempty"`   // --cluster-constraint (federation)
	Clusters            string `json:"clusters,omitempty"`             // --clusters (federation)
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
	CPULoad         float64 // OS 1-minute load average (SLURM cpu_load / 100)
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
	Partition     string
	PendingJobs   int
	RunningJobs   int
	TotalJobs     int
	AverageWait   time.Duration
	LongestWait   time.Duration
	AllocatedCPUs int
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
	JobStateCancelled   = "CANCELLED" //nolint:misspell // matches SLURM official job state spelling
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
