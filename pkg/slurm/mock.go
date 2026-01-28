// Package slurm provides SLURM client implementation and mock for testing.
package slurm

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/jontk/s9s/internal/dao"
)

// MockClient implements a mock SLURM client for testing and development
type MockClient struct {
	mu           sync.RWMutex
	jobs         map[string]*dao.Job
	nodes        map[string]*dao.Node
	partitions   map[string]*dao.Partition
	reservations map[string]*dao.Reservation
	qos          map[string]*dao.QoS
	accounts     map[string]*dao.Account
	users        map[string]*dao.User
	clusterInfo  *dao.ClusterInfo
	delay        time.Duration
}

// NewMockClient creates a new mock SLURM client
func NewMockClient() *MockClient {
	client := &MockClient{
		jobs:         make(map[string]*dao.Job),
		nodes:        make(map[string]*dao.Node),
		partitions:   make(map[string]*dao.Partition),
		reservations: make(map[string]*dao.Reservation),
		qos:          make(map[string]*dao.QoS),
		accounts:     make(map[string]*dao.Account),
		users:        make(map[string]*dao.User),
		clusterInfo: &dao.ClusterInfo{
			Name:     "mock-cluster",
			Endpoint: "http://localhost:6820",
			Version:  "23.02.1",
		},
		delay: 1 * time.Millisecond, // Minimal delay for realistic testing (was 100ms)
	}

	// Populate with sample data
	client.populateSampleData()

	return client
}

// NewFastMockClient creates a mock client with no delay for performance tests
func NewFastMockClient() *MockClient {
	client := NewMockClient()
	client.SetDelay(0) // No delay for performance benchmarks
	return client
}

// SetDelay sets the simulated network delay
func (m *MockClient) SetDelay(delay time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.delay = delay
}

// simulateDelay simulates network delay
func (m *MockClient) simulateDelay() {
	if m.delay > 0 {
		time.Sleep(m.delay)
	}
}

// Jobs returns the mock job manager
func (m *MockClient) Jobs() dao.JobManager {
	return &mockJobManager{client: m}
}

// Nodes returns the mock node manager
func (m *MockClient) Nodes() dao.NodeManager {
	return &mockNodeManager{client: m}
}

// Partitions returns the mock partition manager
func (m *MockClient) Partitions() dao.PartitionManager {
	return &mockPartitionManager{client: m}
}

// Reservations returns the mock reservation manager
func (m *MockClient) Reservations() dao.ReservationManager {
	return &mockReservationManager{client: m}
}

// Info returns the mock info manager
func (m *MockClient) Info() dao.InfoManager {
	return &mockInfoManager{client: m}
}

// QoS returns the mock QoS manager
func (m *MockClient) QoS() dao.QoSManager {
	return &mockQoSManager{client: m}
}

// Accounts returns the mock accounts manager
func (m *MockClient) Accounts() dao.AccountManager {
	return &mockAccountManager{client: m}
}

// Users returns the mock users manager
func (m *MockClient) Users() dao.UserManager {
	return &mockUserManager{client: m}
}

// ClusterInfo returns mock cluster information
func (m *MockClient) ClusterInfo() (*dao.ClusterInfo, error) {
	m.simulateDelay()
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.clusterInfo, nil
}

// Close closes the mock client
func (m *MockClient) Close() error {
	return nil
}

// populateSampleData populates the mock client with sample data
func (m *MockClient) populateSampleData() {
	m.populatePartitions()
	m.populateComputeNodes()
	m.populateGPUNodes()
	m.populateJobs()
	m.populateReservations()
	m.populateQoS()
	m.populateAccounts()
	m.populateUsers()
}

func (m *MockClient) populatePartitions() {
	m.partitions["compute"] = &dao.Partition{
		Name:        "compute",
		State:       dao.PartitionStateUp,
		TotalNodes:  100,
		TotalCPUs:   3200,
		DefaultTime: "1:00:00",
		MaxTime:     "7-00:00:00",
		QOS:         []string{"normal", "high", "low"},
		Nodes:       []string{"node[001-100]"},
	}

	m.partitions["gpu"] = &dao.Partition{
		Name:        "gpu",
		State:       dao.PartitionStateUp,
		TotalNodes:  20,
		TotalCPUs:   640,
		DefaultTime: "4:00:00",
		MaxTime:     "2-00:00:00",
		QOS:         []string{"gpu-normal", "gpu-high"},
		Nodes:       []string{"gpu[001-020]"},
	}

	m.partitions["debug"] = &dao.Partition{
		Name:        "debug",
		State:       dao.PartitionStateUp,
		TotalNodes:  10,
		TotalCPUs:   320,
		DefaultTime: "0:30:00",
		MaxTime:     "1:00:00",
		QOS:         []string{"debug"},
		Nodes:       []string{"debug[001-010]"},
	}
}

func (m *MockClient) populateComputeNodes() {
	for i := 1; i <= 100; i++ {
		state := dao.NodeStateIdle
		if i <= 60 {
			state = dao.NodeStateAllocated
		} else if i > 90 {
			state = dao.NodeStateMixed
		}

		m.nodes[fmt.Sprintf("node%03d", i)] = &dao.Node{
			Name:            fmt.Sprintf("node%03d", i),
			State:           state,
			Partitions:      []string{"compute"},
			CPUsTotal:       32,
			CPUsAllocated:   m.getComputeNodeCPUsAllocated(state),
			CPUsIdle:        m.getComputeNodeCPUsIdle(state),
			MemoryTotal:     128 * 1024, // 128GB
			MemoryAllocated: m.getComputeNodeMemoryAllocated(state),
			MemoryFree:      m.getComputeNodeMemoryFree(state),
			Features:        []string{"avx2", "sse4.2"},
		}
	}

	// Add some down nodes
	m.nodes["node098"].State = dao.NodeStateDown
	m.nodes["node098"].Reason = "Hardware failure"
	m.nodes["node099"].State = dao.NodeStateDrain
	m.nodes["node099"].Reason = "Maintenance"
}

func (m *MockClient) getComputeNodeCPUsAllocated(state string) int {
	switch state {
	case dao.NodeStateAllocated:
		return 32
	case dao.NodeStateMixed:
		return 16
	default:
		return 0
	}
}

func (m *MockClient) getComputeNodeCPUsIdle(state string) int {
	switch state {
	case dao.NodeStateIdle:
		return 32
	case dao.NodeStateMixed:
		return 16
	default:
		return 0
	}
}

func (m *MockClient) getComputeNodeMemoryAllocated(state string) int64 {
	switch state {
	case dao.NodeStateAllocated:
		return 128 * 1024
	case dao.NodeStateMixed:
		return 64 * 1024
	default:
		return 0
	}
}

func (m *MockClient) getComputeNodeMemoryFree(state string) int64 {
	switch state {
	case dao.NodeStateIdle:
		return 128 * 1024
	case dao.NodeStateMixed:
		return 64 * 1024
	default:
		return 0
	}
}

func (m *MockClient) populateGPUNodes() {
	for i := 1; i <= 20; i++ {
		m.nodes[fmt.Sprintf("gpu%03d", i)] = &dao.Node{
			Name:            fmt.Sprintf("gpu%03d", i),
			State:           dao.NodeStateIdle,
			Partitions:      []string{"gpu"},
			CPUsTotal:       32,
			CPUsAllocated:   0,
			CPUsIdle:        32,
			MemoryTotal:     256 * 1024, // 256GB
			MemoryAllocated: 0,
			MemoryFree:      256 * 1024,
			Features:        []string{"gpu", "cuda", "avx2"},
		}
	}
}

func (m *MockClient) populateJobs() {
	jobStates := []string{
		dao.JobStateRunning,
		dao.JobStatePending,
		dao.JobStateCompleted,
		dao.JobStateFailed,
	}

	users := []string{"alice", "bob", "charlie", "david", "eve"}
	accounts := []string{"physics", "chemistry", "biology", "engineering"}

	for i := 1; i <= 200; i++ {
		state := jobStates[rand.Intn(len(jobStates))]
		user := users[rand.Intn(len(users))]
		account := accounts[rand.Intn(len(accounts))]

		job := &dao.Job{
			ID:         fmt.Sprintf("%d", 1000+i),
			Name:       fmt.Sprintf("job_%s_%d", user, i),
			User:       user,
			Account:    account,
			Partition:  "compute",
			State:      state,
			Priority:   rand.Float64() * 1000,
			QOS:        "normal",
			NodeCount:  rand.Intn(10) + 1,
			TimeLimit:  "2:00:00",
			SubmitTime: time.Now().Add(-time.Duration(rand.Intn(72)) * time.Hour),
			WorkingDir: fmt.Sprintf("/home/%s/work", user),
			Command:    "python simulate.py",
		}

		m.setJobStateDetails(job, state)
		m.jobs[job.ID] = job
	}
}

func (m *MockClient) setJobStateDetails(job *dao.Job, state string) {
	switch state {
	case dao.JobStateRunning:
		startTime := job.SubmitTime.Add(time.Duration(rand.Intn(60)) * time.Minute)
		job.StartTime = &startTime
		job.TimeUsed = fmt.Sprintf("%d:%02d:%02d", rand.Intn(2), rand.Intn(60), rand.Intn(60))
		job.NodeList = fmt.Sprintf("node[%03d-%03d]", rand.Intn(90)+1, rand.Intn(90)+10)
	case dao.JobStateCompleted, dao.JobStateFailed:
		startTime := job.SubmitTime.Add(time.Duration(rand.Intn(60)) * time.Minute)
		endTime := startTime.Add(time.Duration(rand.Intn(120)) * time.Minute)
		job.StartTime = &startTime
		job.EndTime = &endTime
		exitCode := 0
		if state == dao.JobStateFailed {
			exitCode = rand.Intn(255) + 1
		}
		job.ExitCode = &exitCode
	}
}

func (m *MockClient) populateReservations() {
	m.reservations["maint-001"] = &dao.Reservation{
		Name:      "maint-001",
		State:     "ACTIVE",
		StartTime: time.Now().Add(24 * time.Hour),
		EndTime:   time.Now().Add(26 * time.Hour),
		Duration:  2 * time.Hour,
		Nodes:     []string{"node[090-100]"},
		NodeCount: 11,
		CoreCount: 352,
		Users:     []string{"admin"},
		Accounts:  []string{"maintenance"},
	}
}

func (m *MockClient) populateQoS() {
	m.qos["normal"] = &dao.QoS{
		Name:                 "normal",
		Priority:             100,
		PreemptMode:          "off",
		Flags:                []string{},
		GraceTime:            30,
		MaxJobsPerUser:       50,
		MaxJobsPerAccount:    200,
		MaxSubmitJobsPerUser: 100,
		MaxCPUsPerUser:       1000,
		MaxNodesPerUser:      32,
		MaxWallTime:          10080, // 7 days in minutes
		MaxMemoryPerUser:     0,     // unlimited
		MinCPUs:              1,
		MinNodes:             1,
	}

	m.qos["high"] = &dao.QoS{
		Name:                 "high",
		Priority:             500,
		PreemptMode:          "suspend",
		Flags:                []string{"DenyOnLimit"},
		GraceTime:            15,
		MaxJobsPerUser:       10,
		MaxJobsPerAccount:    50,
		MaxSubmitJobsPerUser: 20,
		MaxCPUsPerUser:       500,
		MaxNodesPerUser:      16,
		MaxWallTime:          4320, // 3 days in minutes
		MaxMemoryPerUser:     0,    // unlimited
		MinCPUs:              1,
		MinNodes:             1,
	}

	m.qos["low"] = &dao.QoS{
		Name:                 "low",
		Priority:             10,
		PreemptMode:          "off",
		Flags:                []string{},
		GraceTime:            60,
		MaxJobsPerUser:       100,
		MaxJobsPerAccount:    500,
		MaxSubmitJobsPerUser: 200,
		MaxCPUsPerUser:       2000,
		MaxNodesPerUser:      64,
		MaxWallTime:          20160, // 14 days in minutes
		MaxMemoryPerUser:     0,     // unlimited
		MinCPUs:              1,
		MinNodes:             1,
	}
}

func (m *MockClient) populateAccounts() {
	m.accounts["physics"] = &dao.Account{
		Name:         "physics",
		Description:  "Physics Department Research",
		Organization: "University",
		Coordinators: []string{"physics_admin"},
		DefaultQoS:   "normal",
		QoSList:      []string{"normal", "high", "low"},
		MaxJobs:      200,
		MaxNodes:     32,
		MaxCPUs:      1000,
		MaxSubmit:    400,
		MaxWall:      10080, // 7 days
		Parent:       "",
		Children:     []string{"physics_theory", "physics_experimental"},
	}

	m.accounts["physics_theory"] = &dao.Account{
		Name:         "physics_theory",
		Description:  "Theoretical Physics Group",
		Organization: "University",
		Coordinators: []string{"theory_admin"},
		DefaultQoS:   "normal",
		QoSList:      []string{"normal", "high"},
		MaxJobs:      100,
		MaxNodes:     16,
		MaxCPUs:      500,
		MaxSubmit:    200,
		MaxWall:      7200, // 5 days
		Parent:       "physics",
		Children:     []string{},
	}

	m.accounts["physics_experimental"] = &dao.Account{
		Name:         "physics_experimental",
		Description:  "Experimental Physics Group",
		Organization: "University",
		Coordinators: []string{"exp_admin"},
		DefaultQoS:   "normal",
		QoSList:      []string{"normal", "high"},
		MaxJobs:      100,
		MaxNodes:     16,
		MaxCPUs:      500,
		MaxSubmit:    200,
		MaxWall:      7200, // 5 days
		Parent:       "physics",
		Children:     []string{},
	}

	m.accounts["chemistry"] = &dao.Account{
		Name:         "chemistry",
		Description:  "Chemistry Department Research",
		Organization: "University",
		Coordinators: []string{"chem_admin"},
		DefaultQoS:   "normal",
		QoSList:      []string{"normal", "low"},
		MaxJobs:      150,
		MaxNodes:     24,
		MaxCPUs:      800,
		MaxSubmit:    300,
		MaxWall:      14400, // 10 days
		Parent:       "",
		Children:     []string{},
	}

	m.accounts["biology"] = &dao.Account{
		Name:         "biology",
		Description:  "Biology Department Research",
		Organization: "University",
		Coordinators: []string{"bio_admin"},
		DefaultQoS:   "low",
		QoSList:      []string{"normal", "low"},
		MaxJobs:      100,
		MaxNodes:     16,
		MaxCPUs:      600,
		MaxSubmit:    200,
		MaxWall:      20160, // 14 days
		Parent:       "",
		Children:     []string{},
	}

	m.accounts["engineering"] = &dao.Account{
		Name:         "engineering",
		Description:  "Engineering Department Research",
		Organization: "University",
		Coordinators: []string{"eng_admin"},
		DefaultQoS:   "normal",
		QoSList:      []string{"normal", "high"},
		MaxJobs:      300,
		MaxNodes:     48,
		MaxCPUs:      1500,
		MaxSubmit:    600,
		MaxWall:      7200, // 5 days
		Parent:       "",
		Children:     []string{},
	}
}

func (m *MockClient) populateUsers() {
	m.users["alice"] = &dao.User{
		Name:           "alice",
		UID:            1001,
		DefaultAccount: "physics",
		Accounts:       []string{"physics", "physics_theory"},
		AdminLevel:     "None",
		DefaultQoS:     "normal",
		QoSList:        []string{"normal", "high"},
		MaxJobs:        50,
		MaxNodes:       16,
		MaxCPUs:        500,
		MaxSubmit:      100,
	}

	m.users["bob"] = &dao.User{
		Name:           "bob",
		UID:            1002,
		DefaultAccount: "chemistry",
		Accounts:       []string{"chemistry"},
		AdminLevel:     "None",
		DefaultQoS:     "normal",
		QoSList:        []string{"normal", "low"},
		MaxJobs:        30,
		MaxNodes:       8,
		MaxCPUs:        200,
		MaxSubmit:      60,
	}

	m.users["charlie"] = &dao.User{
		Name:           "charlie",
		UID:            1003,
		DefaultAccount: "engineering",
		Accounts:       []string{"engineering"},
		AdminLevel:     "Operator",
		DefaultQoS:     "high",
		QoSList:        []string{"normal", "high"},
		MaxJobs:        100,
		MaxNodes:       32,
		MaxCPUs:        1000,
		MaxSubmit:      200,
	}

	m.users["david"] = &dao.User{
		Name:           "david",
		UID:            1004,
		DefaultAccount: "biology",
		Accounts:       []string{"biology"},
		AdminLevel:     "None",
		DefaultQoS:     "low",
		QoSList:        []string{"normal", "low"},
		MaxJobs:        25,
		MaxNodes:       4,
		MaxCPUs:        100,
		MaxSubmit:      50,
	}

	m.users["eve"] = &dao.User{
		Name:           "eve",
		UID:            1005,
		DefaultAccount: "physics_experimental",
		Accounts:       []string{"physics", "physics_experimental"},
		AdminLevel:     "None",
		DefaultQoS:     "normal",
		QoSList:        []string{"normal", "high"},
		MaxJobs:        40,
		MaxNodes:       12,
		MaxCPUs:        400,
		MaxSubmit:      80,
	}

	m.users["admin"] = &dao.User{
		Name:           "admin",
		UID:            0,
		DefaultAccount: "root",
		Accounts:       []string{"root", "physics", "chemistry", "biology", "engineering"},
		AdminLevel:     "Administrator",
		DefaultQoS:     "high",
		QoSList:        []string{"normal", "high", "low"},
		MaxJobs:        1000,
		MaxNodes:       100,
		MaxCPUs:        5000,
		MaxSubmit:      2000,
	}
}

// mockJobManager implements dao.JobManager
type mockJobManager struct {
	client *MockClient
}

func (m *mockJobManager) List(opts *dao.ListJobsOptions) (*dao.JobList, error) {
	m.client.simulateDelay()
	m.client.mu.RLock()
	defer m.client.mu.RUnlock()

	jobs := m.filterJobs(opts)
	start, end := m.calculateLimitOffset(opts, len(jobs))

	return &dao.JobList{
		Jobs:  jobs[start:end],
		Total: len(jobs),
	}, nil
}

func (m *mockJobManager) filterJobs(opts *dao.ListJobsOptions) []*dao.Job {
	jobs := make([]*dao.Job, 0, len(m.client.jobs))
	for _, job := range m.client.jobs {
		if m.jobMatchesFilters(job, opts) {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

func (m *mockJobManager) jobMatchesFilters(job *dao.Job, opts *dao.ListJobsOptions) bool {
	if opts == nil {
		return true
	}
	if len(opts.States) > 0 && !contains(opts.States, job.State) {
		return false
	}
	if len(opts.Users) > 0 && !contains(opts.Users, job.User) {
		return false
	}
	if len(opts.Partitions) > 0 && !contains(opts.Partitions, job.Partition) {
		return false
	}
	if len(opts.Accounts) > 0 && !contains(opts.Accounts, job.Account) {
		return false
	}
	return true
}

func (m *mockJobManager) calculateLimitOffset(opts *dao.ListJobsOptions, jobCount int) (int, int) {
	start := 0
	end := jobCount
	if opts != nil {
		if opts.Offset > 0 && opts.Offset < jobCount {
			start = opts.Offset
		}
		if opts.Limit > 0 && start+opts.Limit < end {
			end = start + opts.Limit
		}
	}
	return start, end
}

func (m *mockJobManager) Get(id string) (*dao.Job, error) {
	m.client.simulateDelay()
	m.client.mu.RLock()
	defer m.client.mu.RUnlock()

	job, exists := m.client.jobs[id]
	if !exists {
		return nil, fmt.Errorf("job %s not found", id)
	}
	return job, nil
}

func (m *mockJobManager) Submit(jobSub *dao.JobSubmission) (string, error) {
	m.client.simulateDelay()
	m.client.mu.Lock()
	defer m.client.mu.Unlock()

	// Generate new job ID
	jobID := fmt.Sprintf("%d", 2000+len(m.client.jobs))

	// Create new job from submission
	job := &dao.Job{
		ID:         jobID,
		Name:       jobSub.Name,
		User:       "current-user", // In real implementation, would get from auth context
		Account:    jobSub.Account,
		Partition:  jobSub.Partition,
		State:      dao.JobStatePending,
		Priority:   100.0, // Default priority
		QOS:        jobSub.QoS,
		NodeCount:  jobSub.Nodes,
		TimeLimit:  jobSub.TimeLimit,
		SubmitTime: time.Now(),
		WorkingDir: jobSub.WorkingDir,
		Command:    jobSub.Command,
		StdOut:     jobSub.StdOut,
		StdErr:     jobSub.StdErr,
	}

	// Set defaults if not provided
	if job.Account == "" {
		job.Account = "default"
	}
	if job.Partition == "" {
		job.Partition = "compute"
	}
	if job.QOS == "" {
		job.QOS = "normal"
	}
	if job.NodeCount == 0 {
		job.NodeCount = 1
	}
	if job.TimeLimit == "" {
		job.TimeLimit = "1:00:00"
	}
	if job.WorkingDir == "" {
		job.WorkingDir = "/tmp"
	}

	// Store the job
	m.client.jobs[jobID] = job

	return job.ID, nil
}

func (m *mockJobManager) Cancel(id string) error {
	m.client.simulateDelay()
	m.client.mu.Lock()
	defer m.client.mu.Unlock()

	job, exists := m.client.jobs[id]
	if !exists {
		return fmt.Errorf("job %s not found", id)
	}

	if job.State == dao.JobStateRunning || job.State == dao.JobStatePending {
		job.State = dao.JobStateCancelled
		now := time.Now()
		job.EndTime = &now
	}
	return nil
}

func (m *mockJobManager) Hold(id string) error {
	m.client.simulateDelay()
	m.client.mu.Lock()
	defer m.client.mu.Unlock()

	job, exists := m.client.jobs[id]
	if !exists {
		return fmt.Errorf("job %s not found", id)
	}

	if job.State == dao.JobStatePending {
		job.State = dao.JobStateSuspended
	}
	return nil
}

func (m *mockJobManager) Release(id string) error {
	m.client.simulateDelay()
	m.client.mu.Lock()
	defer m.client.mu.Unlock()

	job, exists := m.client.jobs[id]
	if !exists {
		return fmt.Errorf("job %s not found", id)
	}

	if job.State == dao.JobStateSuspended {
		job.State = dao.JobStatePending
	}
	return nil
}

func (m *mockJobManager) Requeue(id string) (*dao.Job, error) {
	m.client.simulateDelay()
	m.client.mu.Lock()
	defer m.client.mu.Unlock()

	originalJob, exists := m.client.jobs[id]
	if !exists {
		return nil, fmt.Errorf("job %s not found", id)
	}

	// Check if job can be requeued
	if originalJob.State != dao.JobStateCompleted && originalJob.State != dao.JobStateFailed && originalJob.State != dao.JobStateCancelled {
		return nil, fmt.Errorf("job %s cannot be requeued (current state: %s)", id, originalJob.State)
	}

	// Generate new job ID
	newJobID := fmt.Sprintf("%d", 3000+len(m.client.jobs))

	// Create new job based on original
	newJob := &dao.Job{
		ID:         newJobID,
		Name:       originalJob.Name + "_requeued",
		User:       originalJob.User,
		Account:    originalJob.Account,
		Partition:  originalJob.Partition,
		State:      dao.JobStatePending,
		Priority:   originalJob.Priority,
		QOS:        originalJob.QOS,
		NodeCount:  originalJob.NodeCount,
		TimeLimit:  originalJob.TimeLimit,
		SubmitTime: time.Now(),
		WorkingDir: originalJob.WorkingDir,
		Command:    originalJob.Command,
		StdOut:     originalJob.StdOut,
		StdErr:     originalJob.StdErr,
	}

	// Store the new job
	m.client.jobs[newJobID] = newJob

	return newJob, nil
}

func (m *mockJobManager) GetOutput(id string) (string, error) {
	m.client.simulateDelay()
	m.client.mu.RLock()
	defer m.client.mu.RUnlock()

	_, exists := m.client.jobs[id]
	if !exists {
		return "", fmt.Errorf("job %s not found", id)
	}

	// Return mock output
	return fmt.Sprintf("Mock output for job %s\nLine 1: Starting simulation...\nLine 2: Processing data...\nLine 3: Simulation complete.", id), nil
}

func (m *mockJobManager) Notify(id, _ string) error {
	m.client.simulateDelay()
	m.client.mu.RLock()
	defer m.client.mu.RUnlock()

	_, exists := m.client.jobs[id]
	if !exists {
		return fmt.Errorf("job %s not found", id)
	}

	// In a real implementation, this would send a notification to the job
	// For the mock, we just return success
	return nil
}

// mockNodeManager implements dao.NodeManager
type mockNodeManager struct {
	client *MockClient
}

func (m *mockNodeManager) List(opts *dao.ListNodesOptions) (*dao.NodeList, error) {
	m.client.simulateDelay()
	m.client.mu.RLock()
	defer m.client.mu.RUnlock()

	nodes := make([]*dao.Node, 0, len(m.client.nodes))
	for _, node := range m.client.nodes {
		// Apply filters
		if opts != nil {
			if len(opts.States) > 0 && !contains(opts.States, node.State) {
				continue
			}
			if len(opts.Partitions) > 0 && !hasCommonElement(opts.Partitions, node.Partitions) {
				continue
			}
		}
		nodes = append(nodes, node)
	}

	return &dao.NodeList{
		Nodes: nodes,
		Total: len(nodes),
	}, nil
}

func (m *mockNodeManager) Get(name string) (*dao.Node, error) {
	m.client.simulateDelay()
	m.client.mu.RLock()
	defer m.client.mu.RUnlock()

	node, exists := m.client.nodes[name]
	if !exists {
		return nil, fmt.Errorf("node %s not found", name)
	}
	return node, nil
}

func (m *mockNodeManager) Drain(name, reason string) error {
	m.client.simulateDelay()
	m.client.mu.Lock()
	defer m.client.mu.Unlock()

	node, exists := m.client.nodes[name]
	if !exists {
		return fmt.Errorf("node %s not found", name)
	}

	node.State = dao.NodeStateDrain
	node.Reason = reason
	now := time.Now()
	node.ReasonTime = &now
	return nil
}

func (m *mockNodeManager) Resume(name string) error {
	m.client.simulateDelay()
	m.client.mu.Lock()
	defer m.client.mu.Unlock()

	node, exists := m.client.nodes[name]
	if !exists {
		return fmt.Errorf("node %s not found", name)
	}

	if node.State == dao.NodeStateDrain || node.State == dao.NodeStateDraining {
		node.State = dao.NodeStateIdle
		node.Reason = ""
		node.ReasonTime = nil
	}
	return nil
}

func (m *mockNodeManager) SetState(name, state string) error {
	m.client.simulateDelay()
	m.client.mu.Lock()
	defer m.client.mu.Unlock()

	node, exists := m.client.nodes[name]
	if !exists {
		return fmt.Errorf("node %s not found", name)
	}

	node.State = state
	return nil
}

// mockPartitionManager implements dao.PartitionManager
type mockPartitionManager struct {
	client *MockClient
}

func (m *mockPartitionManager) List() (*dao.PartitionList, error) {
	m.client.simulateDelay()
	m.client.mu.RLock()
	defer m.client.mu.RUnlock()

	partitions := make([]*dao.Partition, 0, len(m.client.partitions))
	for _, partition := range m.client.partitions {
		partitions = append(partitions, partition)
	}

	return &dao.PartitionList{
		Partitions: partitions,
	}, nil
}

func (m *mockPartitionManager) Get(name string) (*dao.Partition, error) {
	m.client.simulateDelay()
	m.client.mu.RLock()
	defer m.client.mu.RUnlock()

	partition, exists := m.client.partitions[name]
	if !exists {
		return nil, fmt.Errorf("partition %s not found", name)
	}
	return partition, nil
}

// mockReservationManager implements dao.ReservationManager
type mockReservationManager struct {
	client *MockClient
}

func (m *mockReservationManager) List() (*dao.ReservationList, error) {
	m.client.simulateDelay()
	m.client.mu.RLock()
	defer m.client.mu.RUnlock()

	reservations := make([]*dao.Reservation, 0, len(m.client.reservations))
	for _, reservation := range m.client.reservations {
		reservations = append(reservations, reservation)
	}

	return &dao.ReservationList{
		Reservations: reservations,
	}, nil
}

func (m *mockReservationManager) Get(name string) (*dao.Reservation, error) {
	m.client.simulateDelay()
	m.client.mu.RLock()
	defer m.client.mu.RUnlock()

	reservation, exists := m.client.reservations[name]
	if !exists {
		return nil, fmt.Errorf("reservation %s not found", name)
	}
	return reservation, nil
}

// mockInfoManager implements dao.InfoManager
type mockInfoManager struct {
	client *MockClient
}

func (m *mockInfoManager) GetClusterInfo() (*dao.ClusterInfo, error) {
	return m.client.ClusterInfo()
}

func (m *mockInfoManager) GetStats() (*dao.ClusterMetrics, error) {
	m.client.simulateDelay()
	m.client.mu.RLock()
	defer m.client.mu.RUnlock()

	// Calculate metrics
	totalJobs := len(m.client.jobs)
	runningJobs := 0
	pendingJobs := 0

	for _, job := range m.client.jobs {
		switch job.State {
		case dao.JobStateRunning:
			runningJobs++
		case dao.JobStatePending:
			pendingJobs++
		}
	}

	totalNodes := len(m.client.nodes)
	activeNodes := 0
	idleNodes := 0
	downNodes := 0
	totalCPUs := 0
	usedCPUs := 0
	totalMemory := int64(0)
	usedMemory := int64(0)

	for _, node := range m.client.nodes {
		totalCPUs += node.CPUsTotal
		usedCPUs += node.CPUsAllocated
		totalMemory += node.MemoryTotal
		usedMemory += node.MemoryAllocated

		switch node.State {
		case dao.NodeStateAllocated, dao.NodeStateMixed:
			activeNodes++
		case dao.NodeStateIdle:
			idleNodes++
		case dao.NodeStateDown, dao.NodeStateDrain, dao.NodeStateDraining:
			downNodes++
		}
	}

	cpuUsage := 0.0
	if totalCPUs > 0 {
		cpuUsage = float64(usedCPUs) * 100 / float64(totalCPUs)
	}

	memoryUsage := 0.0
	if totalMemory > 0 {
		memoryUsage = float64(usedMemory) * 100 / float64(totalMemory)
	}

	return &dao.ClusterMetrics{
		TotalJobs:   totalJobs,
		RunningJobs: runningJobs,
		PendingJobs: pendingJobs,
		TotalNodes:  totalNodes,
		ActiveNodes: activeNodes,
		IdleNodes:   idleNodes,
		DownNodes:   downNodes,
		CPUUsage:    cpuUsage,
		MemoryUsage: memoryUsage,
		LastUpdated: time.Now(),
	}, nil
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func hasCommonElement(slice1, slice2 []string) bool {
	for _, s1 := range slice1 {
		for _, s2 := range slice2 {
			if s1 == s2 {
				return true
			}
		}
	}
	return false
}

// mockQoSManager implements dao.QoSManager
type mockQoSManager struct {
	client *MockClient
}

func (m *mockQoSManager) List() (*dao.QoSList, error) {
	m.client.simulateDelay()
	m.client.mu.RLock()
	defer m.client.mu.RUnlock()

	qosList := make([]*dao.QoS, 0, len(m.client.qos))
	for _, qos := range m.client.qos {
		qosList = append(qosList, qos)
	}

	return &dao.QoSList{
		QoS:   qosList,
		Total: len(qosList),
	}, nil
}

func (m *mockQoSManager) Get(name string) (*dao.QoS, error) {
	m.client.simulateDelay()
	m.client.mu.RLock()
	defer m.client.mu.RUnlock()

	qos, exists := m.client.qos[name]
	if !exists {
		return nil, fmt.Errorf("QoS %s not found", name)
	}
	return qos, nil
}

// mockAccountManager implements dao.AccountManager
type mockAccountManager struct {
	client *MockClient
}

func (m *mockAccountManager) List() (*dao.AccountList, error) {
	m.client.simulateDelay()
	m.client.mu.RLock()
	defer m.client.mu.RUnlock()

	accounts := make([]*dao.Account, 0, len(m.client.accounts))
	for _, account := range m.client.accounts {
		accounts = append(accounts, account)
	}

	return &dao.AccountList{
		Accounts: accounts,
		Total:    len(accounts),
	}, nil
}

func (m *mockAccountManager) Get(name string) (*dao.Account, error) {
	m.client.simulateDelay()
	m.client.mu.RLock()
	defer m.client.mu.RUnlock()

	account, exists := m.client.accounts[name]
	if !exists {
		return nil, fmt.Errorf("account %s not found", name)
	}
	return account, nil
}

// mockUserManager implements dao.UserManager
type mockUserManager struct {
	client *MockClient
}

func (m *mockUserManager) List() (*dao.UserList, error) {
	m.client.simulateDelay()
	m.client.mu.RLock()
	defer m.client.mu.RUnlock()

	users := make([]*dao.User, 0, len(m.client.users))
	for _, user := range m.client.users {
		users = append(users, user)
	}

	return &dao.UserList{
		Users: users,
		Total: len(users),
	}, nil
}

func (m *mockUserManager) Get(name string) (*dao.User, error) {
	m.client.simulateDelay()
	m.client.mu.RLock()
	defer m.client.mu.RUnlock()

	user, exists := m.client.users[name]
	if !exists {
		return nil, fmt.Errorf("user %s not found", name)
	}
	return user, nil
}
