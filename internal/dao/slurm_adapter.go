package dao

import (
	"context"
	"fmt"
	"os"
	osuser "os/user"
	"strings"
	"time"

	"github.com/jontk/s9s/internal/config"
	"github.com/jontk/s9s/internal/debug"
	"github.com/jontk/s9s/internal/errs"
	slurm "github.com/jontk/slurm-client"
	slurmconfig "github.com/jontk/slurm-client/pkg/config"
)

// SlurmAdapter wraps the slurm-client library and provides version abstraction
type SlurmAdapter struct {
	client slurm.SlurmClient
	config *config.ClusterConfig
	ctx    context.Context
	cache  *DAOCache
}

// NewSlurmAdapter creates a new SLURM adapter instance
func NewSlurmAdapter(ctx context.Context, cfg *config.ClusterConfig) (*SlurmAdapter, error) {
	if cfg == nil {
		return nil, errs.Config("cluster config is required")
	}

	// Parse timeout - use shorter timeout for database operations to fail fast
	timeout := 5 * time.Second // Reduced from 30s to 5s for faster failover
	if cfg.Timeout != "" {
		if t, err := time.ParseDuration(cfg.Timeout); err == nil {
			timeout = t
		}
	}

	// Create slurm-client config
	slurmCfg := &slurmconfig.Config{
		BaseURL:    cfg.Endpoint,
		Timeout:    timeout,
		MaxRetries: 0, // Disable retries to avoid delays
	}

	// Create client options
	opts := []slurm.ClientOption{
		slurm.WithConfig(slurmCfg),
		// Adapter implementation is now the default (WithUseAdapters removed in v0.3+)
	}

	// Add authentication if token is provided
	if cfg.Token != "" {
		// Get current username for X-SLURM-USER-NAME header
		username := "root" // default
		if currentUser, err := osuser.Current(); err == nil && currentUser.Username != "" {
			username = currentUser.Username
		}

		// Use WithUserToken to set both X-SLURM-USER-NAME and X-SLURM-USER-TOKEN headers
		// This is required for slurmrestd authentication
		opts = append(opts, slurm.WithUserToken(username, cfg.Token))
	}

	// Create the client
	var slurmClient slurm.SlurmClient
	var err error

	if cfg.APIVersion != "" {
		debug.Logger.Printf("Creating SLURM client with explicit version: %s", cfg.APIVersion)
		slurmClient, err = slurm.NewClientWithVersion(ctx, cfg.APIVersion, opts...)
	} else {
		// Auto-detect version
		debug.Logger.Printf("Creating SLURM client with auto-detected version")
		slurmClient, err = slurm.NewClient(ctx, opts...)
	}

	if err != nil {
		debug.Logger.Printf("Failed to create SLURM client: %v", err)
		return nil, errs.Wrap(err, errs.ErrorTypeNetwork, "failed to create SLURM client").
			WithContext("component", "dao").
			WithContext("endpoint", cfg.Endpoint).
			WithContext("timeout", timeout.String())
	}

	debug.Logger.Printf("SLURM client created successfully")

	return &SlurmAdapter{
		client: slurmClient,
		config: cfg,
		ctx:    ctx,
		cache:  NewDAOCache(10*time.Second, 50),
	}, nil
}

// Close closes the adapter and releases resources
func (s *SlurmAdapter) Close() error {
	return nil
}

// Jobs returns the jobs manager with caching
func (s *SlurmAdapter) Jobs() JobManager {
	return &cachedJobManager{
		inner: &jobManager{
			client: s.client.Jobs(),
			ctx:    s.ctx,
		},
		cache: s.cache,
	}
}

// Nodes returns the nodes manager with caching
func (s *SlurmAdapter) Nodes() NodeManager {
	return &cachedNodeManager{
		inner: &nodeManager{
			client: s.client.Nodes(),
			ctx:    s.ctx,
		},
		cache: s.cache,
	}
}

// Partitions returns the partitions manager with caching
func (s *SlurmAdapter) Partitions() PartitionManager {
	return &cachedPartitionManager{
		inner: &partitionManager{
			client: s.client.Partitions(),
			ctx:    s.ctx,
		},
		cache: s.cache,
	}
}

// Reservations returns the reservations manager
func (s *SlurmAdapter) Reservations() ReservationManager {
	return &reservationManager{
		client: s.client.Reservations(),
		ctx:    s.ctx,
	}
}

// QoS returns the QoS manager
func (s *SlurmAdapter) QoS() QoSManager {
	return &qosManager{
		client: s.client.QoS(),
		ctx:    s.ctx,
	}
}

// Accounts returns the accounts manager
func (s *SlurmAdapter) Accounts() AccountManager {
	return &accountManager{
		client: s.client.Accounts(),
		ctx:    s.ctx,
	}
}

// Users returns the users manager
func (s *SlurmAdapter) Users() UserManager {
	return &userManager{
		client: s.client.Users(),
		ctx:    s.ctx,
	}
}

// Info returns the info manager for cluster information
func (s *SlurmAdapter) Info() InfoManager {
	return &infoManager{
		client:     s.client.Info(),
		nodeClient: s.client.Nodes(),
		ctx:        s.ctx,
	}
}

// ClusterInfo returns cluster information
func (s *SlurmAdapter) ClusterInfo() (*ClusterInfo, error) {
	info, err := s.client.Info().Get(s.ctx)
	if err != nil {
		return nil, errs.SlurmAPI("get cluster info", err)
	}

	return &ClusterInfo{
		Name:     info.ClusterName,
		Endpoint: s.config.Endpoint,
		Version:  info.Version,
	}, nil
}

// jobManager implements JobManager
type jobManager struct {
	client slurm.JobManager
	ctx    context.Context
}

func (j *jobManager) List(opts *ListJobsOptions) (*JobList, error) {
	debug.Logger.Printf("Jobs List() called at %s", time.Now().Format("15:04:05.000"))
	// Convert options to slurm-client format
	// Note: slurm-client's ListJobsOptions only supports: UserID, States, Partition, Limit, Offset
	// For other filters (Users, Partitions, Accounts), the adapter will apply client-side filtering
	clientOpts := &slurm.ListJobsOptions{}
	if opts != nil {
		clientOpts.States = opts.States
		clientOpts.Limit = opts.Limit
		clientOpts.Offset = opts.Offset
		// Note: If we need per-user filtering, convert first user to UserID
		// For now, we pass empty and let adapter handle in applyClientSideFilters()
		debug.Logger.Printf("List options - States: %v, Limit: %d, Offset: %d", clientOpts.States, clientOpts.Limit, clientOpts.Offset)
	}

	// DEBUG: Log what we're passing to the adapter
	debug.Logger.Printf("DEBUG: Calling adapter with clientOpts: States=%v, Limit=%d, Offset=%d",
		clientOpts.States, clientOpts.Limit, clientOpts.Offset)

	// Call the client
	result, err := j.client.List(j.ctx, clientOpts)
	debug.Logger.Printf("Jobs List() returned at %s", time.Now().Format("15:04:05.000"))
	if err != nil {
		debug.Logger.Printf("Jobs List() error: %v", err)
		return nil, errs.SlurmAPI("list jobs", err)
	}

	if result == nil {
		debug.Logger.Printf("Jobs List() returned nil result")
		return &JobList{Jobs: []*Job{}, Total: 0}, nil
	}

	debug.Logger.Printf("DEBUG: Result object - type: %T, Jobs length: %d, Total: %d", result, len(result.Jobs), result.Total)
	debug.Logger.Printf("Jobs List() returned %d jobs, total: %d", len(result.Jobs), result.Total)

	// Convert to our types
	jobs := make([]*Job, 0, len(result.Jobs))
	for _, job := range result.Jobs {
		converted := convertJob(&job)

		// Apply client-side partition filter if specified
		if opts != nil && len(opts.Partitions) > 0 {
			matched := false
			for _, filterPart := range opts.Partitions {
				if converted.Partition == filterPart {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}

		jobs = append(jobs, converted)
	}

	return &JobList{
		Jobs:  jobs,
		Total: len(jobs),
	}, nil
}

func (j *jobManager) Get(id string) (*Job, error) {
	debug.Logger.Printf("JobManager.Get() called for job %s", id)
	job, err := j.client.Get(j.ctx, id)
	if err != nil {
		debug.Logger.Printf("JobManager.Get() error: %v", err)
		return nil, errs.DAOError("get", "job", err).WithContext("job_id", id)
	}
	if job == nil {
		debug.Logger.Printf("JobManager.Get() returned nil job for %s", id)
		return nil, nil
	}
	// Log job details (handling pointer fields)
	jobID := "unknown"
	if job.JobID != nil {
		jobID = fmt.Sprintf("%d", *job.JobID)
	}
	jobState := "unknown"
	if len(job.JobState) > 0 {
		jobState = string(job.JobState[0])
	}
	debug.Logger.Printf("JobManager.Get() returned job: ID=%s, State=%s", jobID, jobState)
	return convertJob(job), nil
}

func (j *jobManager) Submit(job *JobSubmission) (string, error) {
	// Convert our JobSubmission directly to slurm-client JobCreate,
	// bypassing the lossy JobSubmission intermediate struct.
	jobCreate := convertJobSubmissionToJobCreate(job)

	result, err := j.client.SubmitRaw(j.ctx, jobCreate)
	if err != nil {
		return "", errs.SlurmAPI("submit job", err)
	}

	return fmt.Sprintf("%d", result.JobId), nil
}

// convertJobSubmissionToJobCreate converts our JobSubmission directly to slurm-client's
// JobCreate struct (the full OpenAPI type with 90+ fields). This bypasses the deprecated
// slurm.JobSubmission which only supports 12 fields.
func convertJobSubmissionToJobCreate(job *JobSubmission) *slurm.JobCreate {
	// Convert time limit from string to uint32 minutes
	var timeLimit uint32
	if job.TimeLimit != "" {
		var hours, minutes, seconds int
		if strings.Contains(job.TimeLimit, "-") {
			// D-HH:MM:SS format
			var days int
			parts := strings.SplitN(job.TimeLimit, "-", 2)
			fmt.Sscanf(parts[0], "%d", &days)
			fmt.Sscanf(parts[1], "%d:%d:%d", &hours, &minutes, &seconds)
			timeLimit = uint32(days*24*60 + hours*60 + minutes)
		} else if _, err := fmt.Sscanf(job.TimeLimit, "%d:%d:%d", &hours, &minutes, &seconds); err == nil {
			timeLimit = uint32(hours*60 + minutes)
		} else {
			var mins int
			if _, err := fmt.Sscanf(job.TimeLimit, "%d", &mins); err == nil {
				timeLimit = uint32(mins)
			} else {
				timeLimit = 60 // default
			}
		}
	}

	// Convert memory from string to uint64 MB
	var memory uint64
	if job.Memory != "" {
		var num int
		if _, err := fmt.Sscanf(job.Memory, "%d", &num); err == nil {
			suffix := strings.ToUpper(job.Memory[len(job.Memory)-1:])
			switch suffix {
			case "G":
				memory = uint64(num) * 1024
			default: // "M" or bare number
				memory = uint64(num)
			}
		}
	}

	// Ensure the script has a shebang; SLURM requires it for batch submission.
	script := job.Script
	if script != "" && !strings.HasPrefix(script, "#!") {
		script = "#!/bin/bash\n" + script
	}

	// Like sbatch/srun, export the caller's environment by default.
	// slurmctld requires at least one env var; an empty set causes
	// "Batch job submission failed".
	// Filter to POSIX-valid names (letters/digits/underscore) to exclude
	// bash-exported functions (e.g. BASH_FUNC_foo%%) which cause slurmrestd
	// to reject the submission.
	var envList []string
	if len(job.Environment) > 0 {
		for k, v := range job.Environment {
			envList = append(envList, k+"="+v)
		}
	} else {
		for _, e := range os.Environ() {
			k, _, ok := strings.Cut(e, "=")
			if ok && isPosixEnvKey(k) {
				envList = append(envList, e)
			}
		}
	}

	jc := &slurm.JobCreate{
		Name:                    ptrString(job.Name),
		Script:                  ptrString(script),
		Partition:               ptrString(job.Partition),
		Account:                 ptrString(job.Account),
		CPUsPerTask:             ptrInt32(int32(job.CPUs)),
		TimeLimit:               ptrUint32(timeLimit),
		CurrentWorkingDirectory: ptrString(job.WorkingDir),
		Environment:             envList,
		MinimumNodes:            ptrInt32(int32(job.Nodes)),
	}

	// Memory
	if memory > 0 {
		jc.MemoryPerNode = ptrUint64(memory)
	}

	// QoS
	if job.QoS != "" {
		jc.QoS = ptrString(job.QoS)
	}

	// GPUs → TRESPerNode
	if job.GPUs > 0 {
		jc.TRESPerNode = ptrString(fmt.Sprintf("gres/gpu:%d", job.GPUs))
	}

	// Output/Error files
	if job.OutputFile != "" {
		jc.StandardOutput = ptrString(job.OutputFile)
	}
	if job.ErrorFile != "" {
		jc.StandardError = ptrString(job.ErrorFile)
	}

	// Email notifications
	if job.EmailNotify {
		jc.MailType = []slurm.MailTypeValue{"ALL"}
		if job.Email != "" {
			jc.MailUser = ptrString(job.Email)
		}
	}

	// Array job
	if job.ArraySpec != "" {
		jc.Array = ptrString(job.ArraySpec)
	}

	// Exclusive → Shared
	if job.Exclusive {
		jc.Shared = []slurm.SharedValue{"EXCLUSIVE"}
	}

	// Requeue
	if job.Requeue {
		jc.Requeue = ptrBool(true)
	}

	// Dependencies → "afterok:id1:id2"
	if len(job.Dependencies) > 0 {
		jc.Dependency = ptrString("afterok:" + strings.Join(job.Dependencies, ":"))
	}

	// Constraints (node features)
	if job.Constraints != "" {
		jc.Constraints = ptrString(job.Constraints)
	}

	// NTasks
	if job.NTasks > 0 {
		jc.Tasks = ptrInt32(int32(job.NTasks))
	}

	// NTasks per node
	if job.NTasksPerNode > 0 {
		jc.TasksPerNode = ptrInt32(int32(job.NTasksPerNode))
	}

	// Gres (generic resources, overrides GPUs if both set)
	if job.Gres != "" {
		jc.TRESPerNode = ptrString(job.Gres)
	}

	// Hold
	if job.Hold {
		jc.Hold = ptrBool(true)
	}

	// Reservation
	if job.Reservation != "" {
		jc.Reservation = ptrString(job.Reservation)
	}

	// Licenses
	if job.Licenses != "" {
		jc.Licenses = ptrString(job.Licenses)
	}

	// Wckey
	if job.Wckey != "" {
		jc.Wckey = ptrString(job.Wckey)
	}

	// Exclude nodes
	if job.ExcludeNodes != "" {
		jc.ExcludedNodes = []string{job.ExcludeNodes}
	}

	// Priority
	if job.Priority > 0 {
		jc.Priority = ptrUint32(uint32(job.Priority))
	}

	// Nice
	if job.Nice != 0 {
		jc.Nice = ptrInt32(int32(job.Nice))
	}

	// Memory per CPU (alternative to memory per node)
	if job.MemoryPerCPU != "" {
		var num int
		if _, err := fmt.Sscanf(job.MemoryPerCPU, "%d", &num); err == nil {
			suffix := strings.ToUpper(job.MemoryPerCPU[len(job.MemoryPerCPU)-1:])
			memMB := uint64(num)
			if suffix == "G" {
				memMB = uint64(num) * 1024
			}
			jc.MemoryPerCPU = ptrUint64(memMB)
		}
	}

	// Begin time (deferred start) — parse to unix timestamp
	if job.BeginTime != "" {
		if bt, err := parseBeginTime(job.BeginTime); err == nil {
			jc.BeginTime = ptrUint64(bt)
		}
	}

	// Comment
	if job.Comment != "" {
		jc.Comment = ptrString(job.Comment)
	}

	// Distribution
	if job.Distribution != "" {
		jc.Distribution = ptrString(job.Distribution)
	}

	// Prefer (preferred features)
	if job.Prefer != "" {
		jc.Prefer = ptrString(job.Prefer)
	}

	// Required nodes
	if job.RequiredNodes != "" {
		jc.RequiredNodes = []string{job.RequiredNodes}
	}

	// Standard input
	if job.StandardInput != "" {
		jc.StandardInput = ptrString(job.StandardInput)
	}

	// Container
	if job.Container != "" {
		jc.Container = ptrString(job.Container)
	}

	// Threads per core
	if job.ThreadsPerCore > 0 {
		jc.ThreadsPerCore = ptrInt32(int32(job.ThreadsPerCore))
	}

	// Tasks per core
	if job.TasksPerCore > 0 {
		jc.TasksPerCore = ptrInt32(int32(job.TasksPerCore))
	}

	// Tasks per socket
	if job.TasksPerSocket > 0 {
		jc.TasksPerSocket = ptrInt32(int32(job.TasksPerSocket))
	}

	// Sockets per node
	if job.SocketsPerNode > 0 {
		jc.SocketsPerNode = ptrInt32(int32(job.SocketsPerNode))
	}

	// Maximum nodes
	if job.MaximumNodes > 0 {
		jc.MaximumNodes = ptrInt32(int32(job.MaximumNodes))
	}

	// Maximum CPUs
	if job.MaximumCPUs > 0 {
		jc.MaximumCPUs = ptrInt32(int32(job.MaximumCPUs))
	}

	// Minimum CPUs per node
	if job.MinimumCPUsPerNode > 0 {
		jc.MinimumCPUsPerNode = ptrInt32(int32(job.MinimumCPUsPerNode))
	}

	// Time minimum (for backfill scheduling)
	if job.TimeMinimum != "" {
		var hours, minutes, seconds int
		if _, err := fmt.Sscanf(job.TimeMinimum, "%d:%d:%d", &hours, &minutes, &seconds); err == nil {
			jc.TimeMinimum = ptrUint32(uint32(hours*60 + minutes))
		}
	}

	// Contiguous
	if job.Contiguous {
		jc.Contiguous = ptrBool(true)
	}

	// Overcommit
	if job.Overcommit {
		jc.Overcommit = ptrBool(true)
	}

	// Kill on node fail
	if job.KillOnNodeFail {
		jc.KillOnNodeFail = ptrBool(true)
	}

	// Wait all nodes
	if job.WaitAllNodes {
		jc.WaitAllNodes = ptrBool(true)
	}

	// Open mode
	if job.OpenMode != "" {
		jc.OpenMode = []slurm.OpenModeValue{slurm.OpenModeValue(strings.ToUpper(job.OpenMode))}
	}

	// TRES per task
	if job.TRESPerTask != "" {
		jc.TRESPerTask = ptrString(job.TRESPerTask)
	}

	// TRES per socket
	if job.TRESPerSocket != "" {
		jc.TRESPerSocket = ptrString(job.TRESPerSocket)
	}

	// Signal (kill warning) — parse "[B:|R:]<sig>[@<time>]" format
	if job.Signal != "" {
		sig := job.Signal
		var delay int
		if idx := strings.LastIndex(sig, "@"); idx >= 0 {
			fmt.Sscanf(sig[idx+1:], "%d", &delay)
			sig = sig[:idx]
		}
		// Handle B: (batch only) and R: (reservation overlap) prefixes
		if strings.HasPrefix(sig, "B:") {
			jc.KillWarningFlags = []slurm.KillWarningFlagsValue{"BATCH_JOB"}
			sig = sig[2:]
		} else if strings.HasPrefix(sig, "R:") {
			sig = sig[2:]
		}
		jc.KillWarningSignal = ptrString(sig)
		if delay > 0 {
			d := uint16(delay)
			jc.KillWarningDelay = &d
		}
	}

	// Temporary disk per node (MB)
	if job.TmpDiskPerNode > 0 {
		jc.TemporaryDiskPerNode = ptrInt32(int32(job.TmpDiskPerNode))
	}

	// Deadline
	if job.Deadline != "" {
		if dl, err := parseBeginTime(job.Deadline); err == nil {
			jc.Deadline = ptrInt64(int64(dl))
		}
	}

	// NTasks per TRES (--ntasks-per-gpu)
	if job.NTasksPerTRES > 0 {
		jc.NtasksPerTRES = ptrInt32(int32(job.NTasksPerTRES))
	}

	// CPU binding
	if job.CPUBinding != "" {
		jc.CPUBinding = ptrString(job.CPUBinding)
	}

	// CPU frequency
	if job.CPUFrequency != "" {
		jc.CPUFrequency = ptrString(job.CPUFrequency)
	}

	// Network
	if job.Network != "" {
		jc.Network = ptrString(job.Network)
	}

	// X11 forwarding
	if job.X11 != "" {
		jc.X11 = []slurm.X11Value{slurm.X11Value(strings.ToUpper(job.X11))}
	}

	// Immediate
	if job.Immediate {
		jc.Immediate = ptrBool(true)
	}

	// Burst buffer
	if job.BurstBuffer != "" {
		jc.BurstBuffer = ptrString(job.BurstBuffer)
	}

	// Batch features
	if job.BatchFeatures != "" {
		jc.BatchFeatures = ptrString(job.BatchFeatures)
	}

	// TRES bind
	if job.TRESBind != "" {
		jc.TRESBind = ptrString(job.TRESBind)
	}

	// TRES freq
	if job.TRESFreq != "" {
		jc.TRESFreq = ptrString(job.TRESFreq)
	}

	// Core specification
	if job.CoreSpecification > 0 {
		jc.CoreSpecification = ptrInt32(int32(job.CoreSpecification))
	}

	// Thread specification
	if job.ThreadSpecification > 0 {
		jc.ThreadSpecification = ptrInt32(int32(job.ThreadSpecification))
	}

	// Memory binding
	if job.MemoryBinding != "" {
		jc.MemoryBinding = ptrString(job.MemoryBinding)
	}

	// Minimum CPUs (total floor, different from CPUsPerTask)
	if job.MinimumCPUs > 0 {
		jc.MinimumCPUs = ptrInt32(int32(job.MinimumCPUs))
	}

	// TRES per job
	if job.TRESPerJob != "" {
		jc.TRESPerJob = ptrString(job.TRESPerJob)
	}

	// CPUs per TRES (--cpus-per-gpu)
	if job.CPUsPerTRES != "" {
		jc.CPUsPerTRES = ptrString(job.CPUsPerTRES)
	}

	// Memory per TRES (--mem-per-gpu)
	if job.MemoryPerTRES != "" {
		jc.MemoryPerTRES = ptrString(job.MemoryPerTRES)
	}

	// Script arguments
	if job.Argv != "" {
		jc.Argv = strings.Fields(job.Argv)
	}

	// Job flags (SPREAD_JOB, KILL_INVALID_DEPENDENCY, etc.)
	if job.Flags != "" {
		var flags []slurm.FlagsValue
		for _, f := range strings.Split(job.Flags, ",") {
			f = strings.TrimSpace(f)
			if f != "" {
				flags = append(flags, slurm.FlagsValue(strings.ToUpper(f)))
			}
		}
		jc.Flags = flags
	}

	// Profile (ENERGY, LUSTRE, NETWORK, TASK)
	if job.ProfileTypes != "" {
		var profiles []slurm.ProfileValue
		for _, p := range strings.Split(job.ProfileTypes, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				profiles = append(profiles, slurm.ProfileValue(strings.ToUpper(p)))
			}
		}
		jc.Profile = profiles
	}

	// CPU binding flags
	if job.CPUBindingFlags != "" {
		var flags []slurm.CPUBindingFlagsValue
		for _, f := range strings.Split(job.CPUBindingFlags, ",") {
			f = strings.TrimSpace(f)
			if f != "" {
				flags = append(flags, slurm.CPUBindingFlagsValue(strings.ToUpper(f)))
			}
		}
		jc.CPUBindingFlags = flags
	}

	// Memory binding type
	if job.MemoryBindingType != "" {
		var types []slurm.MemoryBindingTypeValue
		for _, t := range strings.Split(job.MemoryBindingType, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				types = append(types, slurm.MemoryBindingTypeValue(strings.ToUpper(t)))
			}
		}
		jc.MemoryBindingType = types
	}

	// Required switches
	if job.RequiredSwitches > 0 {
		jc.RequiredSwitches = ptrUint32(uint32(job.RequiredSwitches))
	}

	// Wait for switch (seconds)
	if job.WaitForSwitch > 0 {
		jc.WaitForSwitch = ptrInt32(int32(job.WaitForSwitch))
	}

	// Cluster constraint (federation)
	if job.ClusterConstraint != "" {
		jc.ClusterConstraint = ptrString(job.ClusterConstraint)
	}

	// Clusters (federation)
	if job.Clusters != "" {
		jc.Clusters = ptrString(job.Clusters)
	}

	return jc
}

// parseBeginTime parses a time string into a unix timestamp.
// Supports formats: "2024-01-01T15:00:00", "2024-01-01T15:00", "2024-01-01",
// "now+1hour", "now+30minutes"
// parseBeginTime parses a SLURM-style time string into a unix timestamp.
// Supports all formats accepted by sbatch --begin:
//   - Named times: "now", "today", "tomorrow", "midnight", "noon", "elevenses", "fika", "teatime"
//   - Relative: "now+1hour", "now+30minutes", "now+60" (seconds default)
//   - ISO dates: "2024-01-01", "2024-01-01T15:00", "2024-01-01T15:00:00"
//   - US dates: "01/02/24", "010224"
//   - Time of day: "16:00", "4:00PM"
//   - RFC3339: "2024-01-01T15:00:00Z"
func parseBeginTime(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty time string")
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Named times
	switch strings.ToLower(s) {
	case "now":
		return uint64(now.Unix()), nil
	case "today":
		return uint64(today.Unix()), nil
	case "tomorrow":
		return uint64(today.Add(24 * time.Hour).Unix()), nil
	case "midnight":
		next := today.Add(24 * time.Hour)
		if now.Before(today.Add(1 * time.Second)) {
			next = today // if it's exactly midnight, use now
		}
		return uint64(next.Unix()), nil
	case "noon":
		noon := today.Add(12 * time.Hour)
		if now.After(noon) {
			noon = noon.Add(24 * time.Hour) // next day's noon
		}
		return uint64(noon.Unix()), nil
	case "elevenses":
		t := today.Add(11 * time.Hour)
		if now.After(t) {
			t = t.Add(24 * time.Hour)
		}
		return uint64(t.Unix()), nil
	case "fika":
		t := today.Add(15 * time.Hour)
		if now.After(t) {
			t = t.Add(24 * time.Hour)
		}
		return uint64(t.Unix()), nil
	case "teatime":
		t := today.Add(16 * time.Hour)
		if now.After(t) {
			t = t.Add(24 * time.Hour)
		}
		return uint64(t.Unix()), nil
	}

	// Relative format: now+Nunit (default unit = seconds)
	if strings.HasPrefix(strings.ToLower(s), "now+") {
		rest := s[4:]
		var n int
		var unit string
		if _, err := fmt.Sscanf(rest, "%d%s", &n, &unit); err == nil {
			d := parseDurationUnit(n, unit)
			if d > 0 {
				return uint64(now.Add(d).Unix()), nil
			}
		} else if _, err := fmt.Sscanf(rest, "%d", &n); err == nil {
			// bare number = seconds
			return uint64(now.Add(time.Duration(n) * time.Second).Unix()), nil
		}
	}

	// ISO and date-time formats
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return uint64(t.Unix()), nil
		}
	}

	// US date formats: MM/DD/YY, MMDDYY
	usLayouts := []string{
		"01/02/06",
		"010206",
	}
	for _, layout := range usLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return uint64(t.Unix()), nil
		}
	}

	// Time of day: HH:MM[:SS] — schedule for today or tomorrow
	todayLayouts := []string{
		"15:04:05",
		"15:04",
	}
	for _, layout := range todayLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			target := time.Date(now.Year(), now.Month(), now.Day(),
				t.Hour(), t.Minute(), t.Second(), 0, now.Location())
			if now.After(target) {
				target = target.Add(24 * time.Hour) // next day
			}
			return uint64(target.Unix()), nil
		}
	}

	// AM/PM time: "4:00PM", "4PM"
	ampmLayouts := []string{
		"3:04PM",
		"3:04pm",
		"3PM",
		"3pm",
	}
	for _, layout := range ampmLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			target := time.Date(now.Year(), now.Month(), now.Day(),
				t.Hour(), t.Minute(), 0, 0, now.Location())
			if now.After(target) {
				target = target.Add(24 * time.Hour)
			}
			return uint64(target.Unix()), nil
		}
	}

	return 0, fmt.Errorf("cannot parse begin time: %s", s)
}

// parseDurationUnit converts a number and unit string to a time.Duration.
func parseDurationUnit(n int, unit string) time.Duration {
	switch strings.ToLower(unit) {
	case "s", "sec", "second", "seconds":
		return time.Duration(n) * time.Second
	case "m", "min", "minute", "minutes":
		return time.Duration(n) * time.Minute
	case "h", "hour", "hours":
		return time.Duration(n) * time.Hour
	case "d", "day", "days":
		return time.Duration(n) * 24 * time.Hour
	case "w", "week", "weeks":
		return time.Duration(n) * 7 * 24 * time.Hour
	default:
		return 0
	}
}

// pointer helpers for JobCreate fields
func ptrString(s string) *string { return &s }
func ptrInt32(i int32) *int32    { return &i }
func ptrUint32(i uint32) *uint32 { return &i }
func ptrUint64(i uint64) *uint64 { return &i }
func ptrInt64(i int64) *int64    { return &i }
func ptrBool(b bool) *bool       { return &b }

func (j *jobManager) Cancel(id string) error {
	debug.Logger.Printf("Cancel job %s", id)
	err := j.client.Cancel(j.ctx, id)
	if err != nil {
		debug.Logger.Printf("Cancel failed for job %s: %v", id, err)
	} else {
		debug.Logger.Printf("Cancel successful for job %s", id)
	}
	return err
}

func (j *jobManager) Hold(id string) error {
	debug.Logger.Printf("Hold job %s", id)
	err := j.client.Hold(j.ctx, id)
	if err != nil {
		debug.Logger.Printf("Hold failed for job %s: %v", id, err)
	} else {
		debug.Logger.Printf("Hold successful for job %s", id)
	}
	return err
}

func (j *jobManager) Release(id string) error {
	debug.Logger.Printf("Release job %s", id)
	err := j.client.Release(j.ctx, id)
	if err != nil {
		debug.Logger.Printf("Release failed for job %s: %v", id, err)
	} else {
		debug.Logger.Printf("Release successful for job %s", id)
	}
	return err
}

func (j *jobManager) Requeue(id string) (*Job, error) {
	// Call slurm-client Requeue directly
	err := j.client.Requeue(j.ctx, id)
	if err != nil {
		return nil, errs.SlurmAPI("requeue job", err).WithContext("job_id", id)
	}

	// Get the updated job details
	return j.Get(id)
}

func (j *jobManager) GetOutput(id string) (string, error) {
	// Check if the slurm-client supports getting job output
	if outputGetter, ok := j.client.(interface {
		GetOutput(ctx context.Context, id string) (string, error)
	}); ok {
		return outputGetter.GetOutput(j.ctx, id)
	}

	// Fallback: simulate job output for testing/demo purposes
	return j.simulateJobOutput(id)
}

// simulateJobOutput simulates job output for testing/demo purposes
func (j *jobManager) simulateJobOutput(id string) (string, error) {
	// Get job details to create realistic output
	job, err := j.Get(id)
	if err != nil {
		return "", errs.DAOError("get", "job", err).WithContext("job_id", id).WithContext("operation", "get_output")
	}
	if job == nil {
		return "", errs.NotFound("job").WithContext("job_id", id).WithContext("operation", "get_output")
	}

	// Build output header with job info
	output := j.buildJobOutputHeader(job)

	// Add state-specific output
	output += j.buildStateOutput(job)

	output += "\n=== End of Output ===\n"
	return output, nil
}

// buildJobOutputHeader creates the header section of job output
func (j *jobManager) buildJobOutputHeader(job *Job) string {
	output := fmt.Sprintf("=== Job %s (%s) Output ===\n", job.ID, job.Name)
	output += fmt.Sprintf("User: %s\n", job.User)
	output += fmt.Sprintf("Partition: %s\n", job.Partition)
	output += fmt.Sprintf("Submit Time: %s\n", job.SubmitTime.Format("2006-01-02 15:04:05"))

	if job.StartTime != nil {
		output += fmt.Sprintf("Start Time: %s\n", job.StartTime.Format("2006-01-02 15:04:05"))
	}

	output += fmt.Sprintf("State: %s\n", job.State)
	output += "\n=== Command Output ===\n"

	return output
}

// buildStateOutput creates output based on job state
func (j *jobManager) buildStateOutput(job *Job) string {
	switch job.State {
	case "PENDING":
		return j.addPendingOutput()
	case "RUNNING":
		return j.addRunningOutput()
	case "COMPLETED":
		return j.addCompletedOutput(job)
	case "FAILED":
		return j.addFailedOutput(job)
	case "CANCELLED": //nolint:misspell // matches SLURM official job state spelling
		return j.addCancelledOutput()
	default:
		return fmt.Sprintf("Job state: %s\n", job.State)
	}
}

// addPendingOutput returns output for pending jobs
func (j *jobManager) addPendingOutput() string {
	return "Job is waiting in queue...\n"
}

// addRunningOutput returns output for running jobs
func (j *jobManager) addRunningOutput() string {
	output := "Job is currently running...\n"
	output += "Processing data...\n"
	output += "[Step 1/3] Initializing...\n"
	output += "[Step 2/3] Computing...\n"
	output += "[Step 3/3] Finalizing...\n"
	return output
}

// addCompletedOutput returns output for completed jobs
func (j *jobManager) addCompletedOutput(job *Job) string {
	output := "Job completed successfully!\n"
	output += "Processing completed in " + job.TimeUsed + "\n"
	output += "Results saved to output files.\n"
	if job.ExitCode != nil {
		output += fmt.Sprintf("Exit Code: %d\n", *job.ExitCode)
	}
	return output
}

// addFailedOutput returns output for failed jobs
func (j *jobManager) addFailedOutput(job *Job) string {
	output := "Job failed during execution.\n"
	output += "Error: Simulation failure for demonstration\n"
	if job.ExitCode != nil {
		output += fmt.Sprintf("Exit Code: %d\n", *job.ExitCode)
	}
	return output
}

// addCancelledOutput returns output for cancelled jobs
//
//nolint:misspell // "cancelled" matches SLURM official job state spelling
func (j *jobManager) addCancelledOutput() string {
	return "Job was cancelled by user.\n"
}

func (j *jobManager) Notify(id, message string) error {
	return j.client.Notify(j.ctx, id, message)
}

// nodeManager implements NodeManager
type nodeManager struct {
	client slurm.NodeManager
	ctx    context.Context
}

func (n *nodeManager) List(opts *ListNodesOptions) (*NodeList, error) {
	// Convert options to slurm-client format
	clientOpts := &slurm.ListNodesOptions{}
	if opts != nil {
		clientOpts.States = opts.States
	}

	result, err := n.client.List(n.ctx, clientOpts)
	if err != nil {
		return nil, errs.SlurmAPI("list nodes", err)
	}

	// Convert to our types with client-side partition filtering
	nodes := make([]*Node, 0, len(result.Nodes))
	for _, node := range result.Nodes {
		converted := convertNode(&node)

		// Apply client-side partition filter if specified
		if opts != nil && len(opts.Partitions) > 0 {
			matched := false
			for _, filterPart := range opts.Partitions {
				for _, nodePart := range converted.Partitions {
					if nodePart == filterPart {
						matched = true
						break
					}
				}
				if matched {
					break
				}
			}
			if !matched {
				continue
			}
		}

		nodes = append(nodes, converted)
	}

	return &NodeList{
		Nodes: nodes,
		Total: len(nodes),
	}, nil
}

func (n *nodeManager) Get(name string) (*Node, error) {
	node, err := n.client.Get(n.ctx, name)
	if err != nil {
		return nil, errs.DAOError("get", "node", err).WithContext("node_name", name)
	}
	return convertNode(node), nil
}

func (n *nodeManager) Drain(name, reason string) error {
	debug.Logger.Printf("Drain node %s with reason: %s", name, reason)
	err := n.client.Drain(n.ctx, name, reason)
	if err != nil {
		debug.Logger.Printf("Drain failed for node %s: %v", name, err)
	} else {
		debug.Logger.Printf("Drain successful for node %s", name)
	}
	return err
}

func (n *nodeManager) Resume(name string) error {
	debug.Logger.Printf("Resume node %s", name)
	err := n.client.Resume(n.ctx, name)
	if err != nil {
		debug.Logger.Printf("Resume failed for node %s: %v", name, err)
	} else {
		debug.Logger.Printf("Resume successful for node %s", name)
	}
	return err
}

func (n *nodeManager) SetState(_, _ string) error {
	// Note: May need to implement if slurm-client supports it
	return errs.Internal("set state operation not supported by slurm-client")
}

// partitionManager implements PartitionManager
type partitionManager struct {
	client slurm.PartitionManager
	ctx    context.Context
}

func (p *partitionManager) List() (*PartitionList, error) {
	result, err := p.client.List(p.ctx, nil)
	if err != nil {
		return nil, errs.SlurmAPI("list partitions", err)
	}

	// Convert to our types
	partitions := make([]*Partition, len(result.Partitions))
	for i, partition := range result.Partitions {
		partitions[i] = convertPartition(&partition)
	}

	return &PartitionList{
		Partitions: partitions,
	}, nil
}

func (p *partitionManager) Get(name string) (*Partition, error) {
	partition, err := p.client.Get(p.ctx, name)
	if err != nil {
		return nil, errs.DAOError("get", "partition", err).WithContext("partition_name", name)
	}
	return convertPartition(partition), nil
}

// reservationManager implements ReservationManager
type reservationManager struct {
	client slurm.ReservationManager
	ctx    context.Context
}

func (r *reservationManager) List() (*ReservationList, error) {
	result, err := r.client.List(r.ctx, nil)
	if err != nil {
		return nil, errs.SlurmAPI("list reservations", err)
	}

	// Convert to our types
	reservations := make([]*Reservation, len(result.Reservations))
	for i, res := range result.Reservations {
		reservations[i] = convertReservation(&res)
	}

	return &ReservationList{
		Reservations: reservations,
	}, nil
}

/*
TODO(lint): Review unused code - func (*reservationManager).getMockReservationList is unused

getMockReservationList returns mock reservation data as fallback
func (r *reservationManager) getMockReservationList() *ReservationList {
	now := time.Now()
	mockReservations := []*Reservation{
		{
			Name:      "maintenance-window",
			State:     "ACTIVE",
			StartTime: now.Add(-2 * time.Hour),
			EndTime:   now.Add(2 * time.Hour),
			Duration:  4 * time.Hour,
			Nodes:     []string{"node[050-055]"},
			NodeCount: 6,
			CoreCount: 192,
			Users:     []string{"admin"},
			Accounts:  []string{"maintenance"},
		},
		{
			Name:      "gpu-exclusive",
			State:     "ACTIVE",
			StartTime: now.Add(-1 * time.Hour),
			EndTime:   now.Add(23 * time.Hour),
			Duration:  24 * time.Hour,
			Nodes:     []string{"gpu[001-010]"},
			NodeCount: 10,
			CoreCount: 320,
			Users:     []string{"research-team"},
			Accounts:  []string{"gpu-research"},
		},
		{
			Name:      "scheduled-downtime",
			State:     "INACTIVE",
			StartTime: now.Add(48 * time.Hour),
			EndTime:   now.Add(56 * time.Hour),
			Duration:  8 * time.Hour,
			Nodes:     []string{"node[001-100]"},
			NodeCount: 100,
			CoreCount: 3200,
			Users:     []string{"admin"},
			Accounts:  []string{"system"},
		},
	}

	return &ReservationList{
		Reservations: mockReservations,
	}
}
*/

func (r *reservationManager) Get(name string) (*Reservation, error) {
	reservation, err := r.client.Get(r.ctx, name)
	if err != nil {
		return nil, errs.DAOError("get", "reservation", err).WithContext("reservation_name", name)
	}
	return convertReservation(reservation), nil
}

// infoManager implements InfoManager
type infoManager struct {
	client     slurm.InfoManager
	nodeClient slurm.NodeManager
	ctx        context.Context
}

func (i *infoManager) GetClusterInfo() (*ClusterInfo, error) {
	info, err := i.client.Get(i.ctx)
	if err != nil {
		return nil, errs.SlurmAPI("get cluster info", err)
	}

	return &ClusterInfo{
		Name:     info.ClusterName,
		Endpoint: "",
		Version:  info.Version,
	}, nil
}

func (i *infoManager) GetStats() (*ClusterMetrics, error) {
	stats, err := i.client.Stats(i.ctx)
	if err != nil {
		return nil, errs.SlurmAPI("get cluster stats", err)
	}

	// Calculate CPU usage with divide-by-zero guard
	cpuUsage := 0.0
	if stats.TotalCPUs > 0 {
		cpuUsage = float64(stats.AllocatedCPUs) / float64(stats.TotalCPUs) * 100
	}

	metrics := &ClusterMetrics{
		TotalJobs:   stats.TotalJobs,
		RunningJobs: stats.RunningJobs,
		PendingJobs: stats.PendingJobs,
		TotalNodes:  stats.TotalNodes,
		ActiveNodes: stats.AllocatedNodes,
		IdleNodes:   stats.IdleNodes,
		DownNodes:   0,
		CPUUsage:    cpuUsage,
		MemoryUsage: -1.0, // Will be enriched from node data below
		LastUpdated: time.Now(),
	}

	// Enrich with node-level data for metrics the Stats API doesn't provide
	if i.nodeClient != nil {
		if nodeList, err := i.nodeClient.List(i.ctx, nil); err == nil {
			var totalMem, allocMem int64
			var totalCPUs, allocCPUs int
			for _, node := range nodeList.Nodes {
				if node.RealMemory != nil {
					totalMem += *node.RealMemory
				}
				if node.AllocMemory != nil {
					allocMem += *node.AllocMemory
				}
				if node.CPUs != nil {
					totalCPUs += int(*node.CPUs)
				}
				if node.AllocCPUs != nil {
					allocCPUs += int(*node.AllocCPUs)
				}
			}
			if totalMem > 0 {
				metrics.MemoryUsage = float64(allocMem) / float64(totalMem) * 100
			}
			if metrics.CPUUsage == 0 && totalCPUs > 0 && allocCPUs > 0 {
				metrics.CPUUsage = float64(allocCPUs) / float64(totalCPUs) * 100
			}
		}
	}

	return metrics, nil
}

// Conversion functions
func convertJob(job *slurm.Job) *Job {
	// Handle ExitCode - now a struct with ReturnCode field
	var exitCode *int
	if job.ExitCode != nil && job.ExitCode.ReturnCode != nil {
		code := int(*job.ExitCode.ReturnCode)
		if code != 0 {
			exitCode = &code
		}
	}

	// Prefer UserName if available, fall back to UserID with system lookup
	username := ""
	if job.UserName != nil && *job.UserName != "" {
		username = *job.UserName
	} else if job.UserID != nil {
		// Try to resolve numeric UID to username via system call
		userIDStr := fmt.Sprintf("%d", *job.UserID)
		if u, err := osuser.LookupId(userIDStr); err == nil {
			username = u.Username
		} else {
			username = userIDStr // Fall back to raw ID
		}
	}

	// Convert JobID from *int32 to string
	jobID := ""
	if job.JobID != nil {
		jobID = fmt.Sprintf("%d", *job.JobID)
	}

	// Convert JobState slice to string (take first state if available)
	state := ""
	if len(job.JobState) > 0 {
		state = string(job.JobState[0])
	}

	// Handle pointer fields with safe dereferencing
	name := ""
	if job.Name != nil {
		name = *job.Name
	}

	partition := ""
	if job.Partition != nil {
		partition = *job.Partition
	}

	priority := 0.0
	if job.Priority != nil {
		priority = float64(*job.Priority)
	}

	// TimeLimit is *uint32 in OpenAPI
	timeLimit := "0"
	if job.TimeLimit != nil {
		timeLimit = fmt.Sprintf("%d", *job.TimeLimit)
	}

	// Command is *string
	command := ""
	if job.Command != nil {
		command = *job.Command
	}

	// WorkingDir is *string (CurrentWorkingDirectory in OpenAPI)
	workingDir := ""
	if job.CurrentWorkingDirectory != nil {
		workingDir = *job.CurrentWorkingDirectory
	}

	// Nodes is *string (comma-separated list)
	nodeList := ""
	nodeCount := 0
	if job.Nodes != nil {
		nodeList = *job.Nodes
		if nodeList != "" {
			// Count nodes by splitting on comma
			nodeCount = len(strings.Split(nodeList, ","))
		}
	}

	// Convert time.Time to *time.Time for StartTime and EndTime
	var startTime *time.Time
	if !job.StartTime.IsZero() {
		startTime = &job.StartTime
	}
	var endTime *time.Time
	if !job.EndTime.IsZero() {
		endTime = &job.EndTime
	}

	return &Job{
		ID:         jobID,
		Name:       name,
		User:       username,
		Account:    "", // Not available in basic Job struct
		Partition:  partition,
		State:      state,
		Priority:   priority,
		QOS:        "", // Not available in basic Job struct
		NodeCount:  nodeCount,
		TimeLimit:  timeLimit,
		TimeUsed:   "", // Not available in basic Job struct
		SubmitTime: job.SubmitTime,
		StartTime:  startTime,
		EndTime:    endTime,
		NodeList:   nodeList,
		Command:    command,
		WorkingDir: workingDir,
		StdOut:     "", // Not available in basic Job struct
		StdErr:     "", // Not available in basic Job struct
		ExitCode:   exitCode,
	}
}

func convertNode(node *slurm.Node) *Node {
	// Handle pointer fields with safe dereferencing
	nodeName := ""
	if node.Name != nil {
		nodeName = *node.Name
	}

	// Get state as string (take first if multiple)
	stateStr := ""
	if len(node.State) > 0 {
		stateStr = string(node.State[0])
	}

	// CPUs is *int32
	cpusTotal := 0
	if node.CPUs != nil {
		cpusTotal = int(*node.CPUs)
	}

	// RealMemory is *int64 (Memory field was renamed)
	memoryTotalMB := int64(0)
	if node.RealMemory != nil {
		memoryTotalMB = *node.RealMemory
	}

	// Use real allocation data from SLURM API
	allocCPUs := 0
	if node.AllocCPUs != nil {
		allocCPUs = int(*node.AllocCPUs)
	}
	allocMemory := int64(0)
	if node.AllocMemory != nil {
		allocMemory = *node.AllocMemory
	}
	idleCPUs := safeSubtract(cpusTotal, allocCPUs)
	var freeMemory int64
	if node.FreeMem != nil {
		freeMemory = int64(*node.FreeMem)
	} else {
		freeMemory = safeSubtract64(memoryTotalMB, allocMemory)
	}
	// SLURM cpu_load is the 1-minute OS load average * 100
	cpuLoad := float64(-1)
	if node.CPULoad != nil {
		cpuLoad = float64(*node.CPULoad) / 100.0
	}

	// Reason is *string
	reason := ""
	if node.Reason != nil {
		reason = *node.Reason
	}

	// LastBusy is time.Time, ReasonTime expects *time.Time
	var reasonTime *time.Time
	if !node.LastBusy.IsZero() {
		reasonTime = &node.LastBusy
	}

	debug.Logger.Printf("convertNode: %s state='%s' CPULoad=%.2f AllocCPUs=%d AllocMem=%dMB MemTotal=%dMB FreeMem=%dMB",
		nodeName, stateStr, cpuLoad, allocCPUs, allocMemory, memoryTotalMB, freeMemory)

	return &Node{
		Name:            nodeName,
		State:           stateStr,
		Partitions:      node.Partitions, // []string, safe to use directly
		CPUsTotal:       cpusTotal,
		CPUsAllocated:   allocCPUs,
		CPUsIdle:        idleCPUs,
		CPULoad:         cpuLoad,
		MemoryTotal:     memoryTotalMB,
		MemoryAllocated: allocMemory,   // Memory allocated by SLURM to jobs
		MemoryFree:      freeMemory,    // Actual free memory on the system
		Features:        node.Features, // []string, safe to use directly
		Reason:          reason,
		ReasonTime:      reasonTime,
		AllocatedJobs:   []string{}, // Would need to query jobs for this node
	}
}

// safeSubtract subtracts two integers and returns 0 if result is negative
func safeSubtract(a, b int) int {
	result := a - b
	if result < 0 {
		return 0
	}
	return result
}

// safeSubtract64 subtracts two int64 values and returns 0 if result is negative
func safeSubtract64(a, b int64) int64 {
	result := a - b
	if result < 0 {
		return 0
	}
	return result
}

func convertPartition(partition *slurm.Partition) *Partition {
	// Handle pointer fields
	name := ""
	if partition.Name != nil {
		name = *partition.Name
	}

	// State is in partition.Partition.State (nested struct)
	state := ""
	if partition.Partition != nil && len(partition.Partition.State) > 0 {
		state = string(partition.Partition.State[0])
	}

	// TotalCPUs is in partition.CPUs.Total
	totalCPUs := 0
	if partition.CPUs != nil && partition.CPUs.Total != nil {
		totalCPUs = int(*partition.CPUs.Total)
	}

	// TotalNodes is in partition.Nodes.Total
	totalNodes := 0
	if partition.Nodes != nil && partition.Nodes.Total != nil {
		totalNodes = int(*partition.Nodes.Total)
	}

	// DefaultTime is in partition.Defaults.Time
	defaultTime := "0"
	if partition.Defaults != nil && partition.Defaults.Time != nil {
		defaultTime = fmt.Sprintf("%d", *partition.Defaults.Time)
	}

	// MaxTime is in partition.Maximums.Time
	maxTime := "0"
	if partition.Maximums != nil && partition.Maximums.Time != nil {
		maxTime = fmt.Sprintf("%d", *partition.Maximums.Time)
	}

	return &Partition{
		Name:        name,
		State:       state,
		TotalNodes:  totalNodes,
		TotalCPUs:   totalCPUs,
		DefaultTime: defaultTime,
		MaxTime:     maxTime,
		QOS:         []string{}, // Not available in basic Partition struct
		Nodes:       []string{}, // Not available in basic Partition struct
	}
}

func convertReservation(res *slurm.Reservation) *Reservation {
	// Handle pointer fields
	name := ""
	if res.Name != nil {
		name = *res.Name
	}

	// State field doesn't exist - use Flags instead
	state := ""
	if len(res.Flags) > 0 {
		state = string(res.Flags[0])
	}

	// NodeList is *string (comma-separated)
	nodeList := []string{}
	if res.NodeList != nil && *res.NodeList != "" {
		nodeList = strings.Split(*res.NodeList, ",")
	}

	// NodeCount is *int32
	nodeCount := 0
	if res.NodeCount != nil {
		nodeCount = int(*res.NodeCount)
	}

	// CoreCount is *int32
	coreCount := 0
	if res.CoreCount != nil {
		coreCount = int(*res.CoreCount)
	}

	// Users is *string (comma-separated)
	users := []string{}
	if res.Users != nil && *res.Users != "" {
		users = strings.Split(*res.Users, ",")
	}

	// Accounts is *string (comma-separated)
	accounts := []string{}
	if res.Accounts != nil && *res.Accounts != "" {
		accounts = strings.Split(*res.Accounts, ",")
	}

	return &Reservation{
		Name:      name,
		State:     state,
		StartTime: res.StartTime,
		EndTime:   res.EndTime,
		Duration:  res.EndTime.Sub(res.StartTime),
		Nodes:     nodeList,
		NodeCount: nodeCount,
		CoreCount: coreCount,
		Users:     users,
		Accounts:  accounts,
	}
}

// qosManager implements QoSManager
type qosManager struct {
	client slurm.QoSManager
	ctx    context.Context
}

func (q *qosManager) List() (*QoSList, error) {
	debug.Logger.Printf("QoS List() called at %s", time.Now().Format("15:04:05.000"))
	result, err := q.client.List(q.ctx, nil)
	debug.Logger.Printf("QoS List() returned at %s", time.Now().Format("15:04:05.000"))
	if err != nil {
		return nil, errs.SlurmAPI("list QoS", err)
	}

	debug.Logger.Printf("QoS converting %d items at %s", len(result.QoS), time.Now().Format("15:04:05.000"))
	// Convert to our types
	qosList := make([]*QoS, len(result.QoS))
	for i, qos := range result.QoS {
		qosList[i] = convertQoS(&qos)
	}
	debug.Logger.Printf("QoS conversion done at %s", time.Now().Format("15:04:05.000"))

	return &QoSList{
		QoS:   qosList,
		Total: len(qosList),
	}, nil
}

/*
TODO(lint): Review unused code - func (*qosManager).getMockQoSList is unused

getMockQoSList returns mock QoS data as fallback
func (q *qosManager) getMockQoSList() *QoSList {
	mockQosList := []*QoS{
		{
			Name:                 "normal",
			Priority:             100,
			PreemptMode:          "cluster",
			Flags:                []string{""},
			GraceTime:            60,
			MaxJobsPerUser:       500,
			MaxJobsPerAccount:    1000,
			MaxSubmitJobsPerUser: 1000,
			MaxCPUsPerUser:       10000,
			MaxNodesPerUser:      100,
			MaxWallTime:          10080, // 7 days
			MaxMemoryPerUser:     1024000,
			MinCPUs:              1,
			MinNodes:             1,
		},
		{
			Name:                 "high",
			Priority:             200,
			PreemptMode:          "suspend",
			Flags:                []string{""},
			GraceTime:            120,
			MaxJobsPerUser:       100,
			MaxJobsPerAccount:    200,
			MaxSubmitJobsPerUser: 200,
			MaxCPUsPerUser:       5000,
			MaxNodesPerUser:      50,
			MaxWallTime:          2880, // 2 days
			MaxMemoryPerUser:     512000,
			MinCPUs:              1,
			MinNodes:             1,
		},
		{
			Name:                 "gpu",
			Priority:             150,
			PreemptMode:          "cluster",
			Flags:                []string{""},
			GraceTime:            60,
			MaxJobsPerUser:       50,
			MaxJobsPerAccount:    100,
			MaxSubmitJobsPerUser: 100,
			MaxCPUsPerUser:       1000,
			MaxNodesPerUser:      10,
			MaxWallTime:          1440, // 1 day
			MaxMemoryPerUser:     256000,
			MinCPUs:              1,
			MinNodes:             1,
		},
	}

	return &QoSList{
		QoS:   mockQosList,
		Total: len(mockQosList),
	}
}
*/

func (q *qosManager) Get(name string) (*QoS, error) {
	qos, err := q.client.Get(q.ctx, name)
	if err != nil {
		return nil, errs.DAOError("get", "QoS", err).WithContext("qos_name", name)
	}
	return convertQoS(qos), nil
}

// accountManager implements AccountManager
type accountManager struct {
	client slurm.AccountManager
	ctx    context.Context
}

func (a *accountManager) List() (*AccountList, error) {
	result, err := a.client.List(a.ctx, nil)
	if err != nil {
		return nil, errs.SlurmAPI("list accounts", err)
	}

	// Convert to our types
	accounts := make([]*Account, len(result.Accounts))
	for i, acc := range result.Accounts {
		accounts[i] = convertAccount(&acc)
	}

	return &AccountList{
		Accounts: accounts,
		Total:    len(accounts),
	}, nil
}

/*
TODO(lint): Review unused code - func (*accountManager).getMockAccountList is unused

getMockAccountList returns mock account data as fallback
func (a *accountManager) getMockAccountList() *AccountList {
	mockAccounts := []*Account{
		{
			Name:         "root",
			Description:  "Root account for system administration",
			Organization: "IT Department",
			Coordinators: []string{"admin", "sysadmin"},
			DefaultQoS:   "normal",
			QoSList:      []string{"normal", "high", "low"},
			MaxJobs:      1000,
			MaxNodes:     100,
			MaxCPUs:      10000,
			MaxSubmit:    2000,
			MaxWall:      10080, // 7 days
			Parent:       "",
			Children:     []string{"research", "engineering", "finance"},
		},
		{
			Name:         "research",
			Description:  "Research group accounts",
			Organization: "Research Division",
			Coordinators: []string{"research-lead"},
			DefaultQoS:   "normal",
			QoSList:      []string{"normal", "high", "gpu"},
			MaxJobs:      500,
			MaxNodes:     50,
			MaxCPUs:      5000,
			MaxSubmit:    1000,
			MaxWall:      4320, // 3 days
			Parent:       "root",
			Children:     []string{"ml-research", "physics"},
		},
		{
			Name:         "engineering",
			Description:  "Engineering team accounts",
			Organization: "Engineering Division",
			Coordinators: []string{"eng-manager"},
			DefaultQoS:   "normal",
			QoSList:      []string{"normal", "high"},
			MaxJobs:      300,
			MaxNodes:     30,
			MaxCPUs:      3000,
			MaxSubmit:    600,
			MaxWall:      2880, // 2 days
			Parent:       "root",
			Children:     []string{},
		},
		{
			Name:         "finance",
			Description:  "Finance department computational resources",
			Organization: "Finance Division",
			Coordinators: []string{"finance-lead"},
			DefaultQoS:   "low",
			QoSList:      []string{"low", "normal"},
			MaxJobs:      100,
			MaxNodes:     10,
			MaxCPUs:      1000,
			MaxSubmit:    200,
			MaxWall:      1440, // 1 day
			Parent:       "root",
			Children:     []string{},
		},
	}

	return &AccountList{
		Accounts: mockAccounts,
		Total:    len(mockAccounts),
	}
}
*/

func (a *accountManager) Get(name string) (*Account, error) {
	account, err := a.client.Get(a.ctx, name)
	if err != nil {
		return nil, errs.DAOError("get", "account", err).WithContext("account_name", name)
	}
	return convertAccount(account), nil
}

// userManager implements UserManager
type userManager struct {
	client slurm.UserManager
	ctx    context.Context
}

func (u *userManager) List() (*UserList, error) {
	result, err := u.client.List(u.ctx, nil)
	if err != nil {
		return nil, errs.SlurmAPI("list users", err)
	}

	// Convert to our types
	users := make([]*User, len(result.Users))
	for i, user := range result.Users {
		users[i] = convertUser(&user)
	}

	return &UserList{
		Users: users,
		Total: len(users),
	}, nil
}

/*
TODO(lint): Review unused code - func (*userManager).getMockUserList is unused

getMockUserList returns mock user data as fallback
func (u *userManager) getMockUserList() *UserList {
	mockUsers := []*User{
		{
			Name:           "alice",
			UID:            1001,
			DefaultAccount: "research",
			Accounts:       []string{"research", "ml-research"},
			AdminLevel:     "None",
			DefaultQoS:     "normal",
			QoSList:        []string{"normal", "high", "gpu"},
			MaxJobs:        200,
			MaxNodes:       20,
			MaxCPUs:        2000,
			MaxSubmit:      400,
		},
		{
			Name:           "bob",
			UID:            1002,
			DefaultAccount: "engineering",
			Accounts:       []string{"engineering"},
			AdminLevel:     "None",
			DefaultQoS:     "normal",
			QoSList:        []string{"normal", "high"},
			MaxJobs:        150,
			MaxNodes:       15,
			MaxCPUs:        1500,
			MaxSubmit:      300,
		},
		{
			Name:           "charlie",
			UID:            1003,
			DefaultAccount: "finance",
			Accounts:       []string{"finance"},
			AdminLevel:     "None",
			DefaultQoS:     "low",
			QoSList:        []string{"low", "normal"},
			MaxJobs:        50,
			MaxNodes:       5,
			MaxCPUs:        500,
			MaxSubmit:      100,
		},
		{
			Name:           "admin",
			UID:            0,
			DefaultAccount: "root",
			Accounts:       []string{"root", "research", "engineering", "finance"},
			AdminLevel:     "Administrator",
			DefaultQoS:     "high",
			QoSList:        []string{"normal", "high", "low", "gpu"},
			MaxJobs:        1000,
			MaxNodes:       100,
			MaxCPUs:        10000,
			MaxSubmit:      2000,
		},
		{
			Name:           "researcher1",
			UID:            1004,
			DefaultAccount: "ml-research",
			Accounts:       []string{"ml-research", "research"},
			AdminLevel:     "None",
			DefaultQoS:     "gpu",
			QoSList:        []string{"normal", "high", "gpu"},
			MaxJobs:        100,
			MaxNodes:       10,
			MaxCPUs:        1000,
			MaxSubmit:      200,
		},
	}

	return &UserList{
		Users: mockUsers,
		Total: len(mockUsers),
	}
}
*/

func (u *userManager) Get(name string) (*User, error) {
	user, err := u.client.Get(u.ctx, name)
	if err != nil {
		return nil, errs.DAOError("get", "user", err).WithContext("user_name", name)
	}
	return convertUser(user), nil
}

// Conversion functions for new types
func convertQoS(qos *slurm.QoS) *QoS {
	// Handle pointer fields
	name := ""
	if qos.Name != nil {
		name = *qos.Name
	}

	priority := 0
	if qos.Priority != nil {
		priority = int(*qos.Priority)
	}

	// Flags is now []QoSFlagsValue - convert to strings
	flags := []string{}
	for _, flag := range qos.Flags {
		flags = append(flags, string(flag))
	}

	// PreemptMode is in qos.Preempt.Mode
	preemptMode := ""
	if qos.Preempt != nil && len(qos.Preempt.Mode) > 0 {
		preemptMode = string(qos.Preempt.Mode[0])
	}

	// GraceTime is in qos.Limits.GraceTime (*int32 seconds)
	graceTime := 0
	if qos.Limits != nil && qos.Limits.GraceTime != nil {
		graceTime = int(*qos.Limits.GraceTime) / 60 // Convert seconds to minutes
	}

	// Navigate deeply nested Limits structure for max/min fields
	var maxJobsPerUser, maxJobsPerAccount, maxSubmitJobsPerUser int
	var maxCPUsPerUser, maxNodesPerUser int
	var maxWallTime int
	var maxMemoryPerUser int64
	var minCPUs, minNodes int

	if qos.Limits != nil && qos.Limits.Max != nil {
		// MaxJobsPerUser: Limits.Max.Jobs.ActiveJobs.Per.User
		if qos.Limits.Max.Jobs != nil && qos.Limits.Max.Jobs.ActiveJobs != nil &&
			qos.Limits.Max.Jobs.ActiveJobs.Per != nil && qos.Limits.Max.Jobs.ActiveJobs.Per.User != nil {
			maxJobsPerUser = int(*qos.Limits.Max.Jobs.ActiveJobs.Per.User)
		}

		// MaxJobsPerAccount: Limits.Max.Jobs.ActiveJobs.Per.Account
		if qos.Limits.Max.Jobs != nil && qos.Limits.Max.Jobs.ActiveJobs != nil &&
			qos.Limits.Max.Jobs.ActiveJobs.Per != nil && qos.Limits.Max.Jobs.ActiveJobs.Per.Account != nil {
			maxJobsPerAccount = int(*qos.Limits.Max.Jobs.ActiveJobs.Per.Account)
		}

		// MaxSubmitJobsPerUser: Limits.Max.Jobs.Per.User
		if qos.Limits.Max.Jobs != nil && qos.Limits.Max.Jobs.Per != nil &&
			qos.Limits.Max.Jobs.Per.User != nil {
			maxSubmitJobsPerUser = int(*qos.Limits.Max.Jobs.Per.User)
		}

		// MaxWallTime: Limits.Max.WallClock.Per.Job (in minutes)
		if qos.Limits.Max.WallClock != nil && qos.Limits.Max.WallClock.Per != nil &&
			qos.Limits.Max.WallClock.Per.Job != nil {
			maxWallTime = int(*qos.Limits.Max.WallClock.Per.Job)
		}

		// MaxCPUsPerUser and MaxNodesPerUser from TRES.Per.User
		if qos.Limits.Max.TRES != nil && qos.Limits.Max.TRES.Per != nil {
			for _, tres := range qos.Limits.Max.TRES.Per.User {
				switch tres.Type {
				case "cpu":
					if tres.Count != nil {
						maxCPUsPerUser = int(*tres.Count)
					}
				case "node":
					if tres.Count != nil {
						maxNodesPerUser = int(*tres.Count)
					}
				case "mem":
					if tres.Count != nil {
						maxMemoryPerUser = int64(*tres.Count)
					}
				}
			}
		}
	}

	// MinCPUs and MinNodes from Limits.Min.TRES.Per.Job
	if qos.Limits != nil && qos.Limits.Min != nil && qos.Limits.Min.TRES != nil &&
		qos.Limits.Min.TRES.Per != nil {
		for _, tres := range qos.Limits.Min.TRES.Per.Job {
			switch tres.Type {
			case "cpu":
				if tres.Count != nil {
					minCPUs = int(*tres.Count)
				}
			case "node":
				if tres.Count != nil {
					minNodes = int(*tres.Count)
				}
			}
		}
	}

	return &QoS{
		Name:                 name,
		Priority:             priority,
		PreemptMode:          preemptMode,
		Flags:                flags,
		GraceTime:            graceTime,
		MaxJobsPerUser:       maxJobsPerUser,
		MaxJobsPerAccount:    maxJobsPerAccount,
		MaxSubmitJobsPerUser: maxSubmitJobsPerUser,
		MaxCPUsPerUser:       maxCPUsPerUser,
		MaxNodesPerUser:      maxNodesPerUser,
		MaxWallTime:          maxWallTime,
		MaxMemoryPerUser:     maxMemoryPerUser,
		MinCPUs:              minCPUs,
		MinNodes:             minNodes,
	}
}

func convertAccount(acc *slurm.Account) *Account {
	// Extract coordinator names from Coord structs
	coordinators := []string{}
	for _, coord := range acc.Coordinators {
		coordinators = append(coordinators, coord.Name)
	}

	// Most limit fields are no longer available in the simplified Account struct
	// These would need to be queried from associations or other endpoints
	return &Account{
		Name:         acc.Name,
		Description:  acc.Description,
		Organization: acc.Organization,
		Coordinators: coordinators,
		DefaultQoS:   "",         // Not available in base Account
		QoSList:      []string{}, // Not available in base Account
		MaxJobs:      0,          // Not available in base Account
		MaxNodes:     0,          // Not available in base Account
		MaxCPUs:      0,          // Not available in base Account
		MaxSubmit:    0,          // Not available in base Account
		MaxWall:      0,          // Not available in base Account
		Parent:       "",         // Not available in base Account
		Children:     []string{}, // Not available in base Account
	}
}

func convertUser(user *slurm.User) *User {
	// Extract account names from Associations
	accountNames := []string{}
	for _, assoc := range user.Associations {
		if assoc.Account != nil {
			accountNames = append(accountNames, *assoc.Account)
		}
	}

	// Get default account from Default.Account
	defaultAccount := ""
	if user.Default != nil && user.Default.Account != nil {
		defaultAccount = *user.Default.Account
	}

	// Get admin level (take first if multiple)
	adminLevel := ""
	if len(user.AdministratorLevel) > 0 {
		adminLevel = string(user.AdministratorLevel[0])
	}

	// Most limit fields are not available in the simplified User struct
	// These would need to be queried from associations or other endpoints
	return &User{
		Name:           user.Name,
		UID:            0, // UID not available in User struct
		DefaultAccount: defaultAccount,
		Accounts:       accountNames,
		AdminLevel:     adminLevel,
		DefaultQoS:     "",         // Not available in base User
		QoSList:        []string{}, // Not available in base User
		MaxJobs:        0,          // Not available in base User
		MaxNodes:       0,          // Not available in base User
		MaxCPUs:        0,          // Not available in base User
		MaxSubmit:      0,          // Not available in base User
	}
}

// isPosixEnvKey returns true if k is a valid POSIX environment variable name
// (only letters, digits, and underscores). This filters out bash-exported
// function names like BASH_FUNC_foo%% which cause slurmrestd to reject jobs.
func isPosixEnvKey(k string) bool {
	if len(k) == 0 {
		return false
	}
	for _, c := range k {
		if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}
	return true
}
