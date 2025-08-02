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
		clusterInfo: &dao.ClusterInfo{
			Name:     "mock-cluster",
			Endpoint: "http://localhost:6820",
			Version:  "23.02.1",
		},
		delay: 100 * time.Millisecond, // Simulate network delay
	}

	// Populate with sample data
	client.populateSampleData()

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
	// Add sample partitions
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

	// Add sample nodes
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
			CPUsAllocated:   func() int { if state == dao.NodeStateAllocated { return 32 } else if state == dao.NodeStateMixed { return 16 } else { return 0 } }(),
			CPUsIdle:        func() int { if state == dao.NodeStateIdle { return 32 } else if state == dao.NodeStateMixed { return 16 } else { return 0 } }(),
			MemoryTotal:     128 * 1024, // 128GB
			MemoryAllocated: func() int64 { if state == dao.NodeStateAllocated { return 128 * 1024 } else if state == dao.NodeStateMixed { return 64 * 1024 } else { return 0 } }(),
			MemoryFree:      func() int64 { if state == dao.NodeStateIdle { return 128 * 1024 } else if state == dao.NodeStateMixed { return 64 * 1024 } else { return 0 } }(),
			Features:        []string{"avx2", "sse4.2"},
		}
	}

	// Add some down nodes
	m.nodes["node098"].State = dao.NodeStateDown
	m.nodes["node098"].Reason = "Hardware failure"
	m.nodes["node099"].State = dao.NodeStateDrain
	m.nodes["node099"].Reason = "Maintenance"

	// Add GPU nodes
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

	// Add sample jobs
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

		if state == dao.JobStateRunning {
			startTime := job.SubmitTime.Add(time.Duration(rand.Intn(60)) * time.Minute)
			job.StartTime = &startTime
			job.TimeUsed = fmt.Sprintf("%d:%02d:%02d", rand.Intn(2), rand.Intn(60), rand.Intn(60))
			job.NodeList = fmt.Sprintf("node[%03d-%03d]", rand.Intn(90)+1, rand.Intn(90)+10)
		} else if state == dao.JobStateCompleted || state == dao.JobStateFailed {
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

		m.jobs[job.ID] = job
	}

	// Add sample reservations
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

// mockJobManager implements dao.JobManager
type mockJobManager struct {
	client *MockClient
}

func (m *mockJobManager) List(opts *dao.ListJobsOptions) (*dao.JobList, error) {
	m.client.simulateDelay()
	m.client.mu.RLock()
	defer m.client.mu.RUnlock()

	jobs := make([]*dao.Job, 0, len(m.client.jobs))
	for _, job := range m.client.jobs {
		// Apply filters
		if opts != nil {
			if len(opts.States) > 0 && !contains(opts.States, job.State) {
				continue
			}
			if len(opts.Users) > 0 && !contains(opts.Users, job.User) {
				continue
			}
			if len(opts.Partitions) > 0 && !contains(opts.Partitions, job.Partition) {
				continue
			}
			if len(opts.Accounts) > 0 && !contains(opts.Accounts, job.Account) {
				continue
			}
		}
		jobs = append(jobs, job)
	}

	// Apply limit and offset
	start := 0
	end := len(jobs)
	if opts != nil {
		if opts.Offset > 0 && opts.Offset < len(jobs) {
			start = opts.Offset
		}
		if opts.Limit > 0 && start+opts.Limit < end {
			end = start + opts.Limit
		}
	}

	return &dao.JobList{
		Jobs:  jobs[start:end],
		Total: len(jobs),
	}, nil
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

func (m *mockNodeManager) Drain(name string, reason string) error {
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

func (m *mockNodeManager) SetState(name string, state string) error {
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