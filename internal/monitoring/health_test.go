package monitoring

import (
	"errors"
	"testing"
	"time"

	"github.com/jontk/s9s/internal/dao"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSlurmClient implements dao.SlurmClient for testing
type mockSlurmClient struct {
	jobs         *mockJobManager
	nodes        *mockNodeManager
	info         *mockInfoManager
	partitions   *mockPartitionManager
	reservations *mockReservationManager
	qos          *mockQoSManager
	accounts     *mockAccountManager
	users        *mockUserManager
	infoFunc     func() dao.InfoManager
}

func (m *mockSlurmClient) Jobs() dao.JobManager   { return m.jobs }
func (m *mockSlurmClient) Nodes() dao.NodeManager { return m.nodes }
func (m *mockSlurmClient) Info() dao.InfoManager {
	if m.infoFunc != nil {
		return m.infoFunc()
	}
	return m.info
}
func (m *mockSlurmClient) Partitions() dao.PartitionManager     { return m.partitions }
func (m *mockSlurmClient) Reservations() dao.ReservationManager { return m.reservations }
func (m *mockSlurmClient) QoS() dao.QoSManager                  { return m.qos }
func (m *mockSlurmClient) Accounts() dao.AccountManager         { return m.accounts }
func (m *mockSlurmClient) Users() dao.UserManager               { return m.users }
func (m *mockSlurmClient) ClusterInfo() (*dao.ClusterInfo, error) {
	return nil, errors.New("not implemented")
}
func (m *mockSlurmClient) Close() error { return nil }

// mockJobManager implements dao.JobManager for testing
type mockJobManager struct {
	listFunc      func(*dao.ListJobsOptions) (*dao.JobList, error)
	getFunc       func(string) (*dao.Job, error)
	submitFunc    func(*dao.JobSubmission) (string, error)
	cancelFunc    func(string) error
	holdFunc      func(string) error
	releaseFunc   func(string) error
	requeueFunc   func(string) (*dao.Job, error)
	getOutputFunc func(string) (string, error)
	notifyFunc    func(string, string) error
}

func (m *mockJobManager) List(opts *dao.ListJobsOptions) (*dao.JobList, error) {
	if m.listFunc != nil {
		return m.listFunc(opts)
	}
	return &dao.JobList{Jobs: []*dao.Job{}, Total: 0}, nil
}

func (m *mockJobManager) Get(id string) (*dao.Job, error) {
	if m.getFunc != nil {
		return m.getFunc(id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockJobManager) Submit(job *dao.JobSubmission) (string, error) {
	if m.submitFunc != nil {
		return m.submitFunc(job)
	}
	return "", errors.New("not implemented")
}

func (m *mockJobManager) Cancel(id string) error {
	if m.cancelFunc != nil {
		return m.cancelFunc(id)
	}
	return errors.New("not implemented")
}

func (m *mockJobManager) Hold(id string) error {
	if m.holdFunc != nil {
		return m.holdFunc(id)
	}
	return errors.New("not implemented")
}

func (m *mockJobManager) Release(id string) error {
	if m.releaseFunc != nil {
		return m.releaseFunc(id)
	}
	return errors.New("not implemented")
}

func (m *mockJobManager) Requeue(id string) (*dao.Job, error) {
	if m.requeueFunc != nil {
		return m.requeueFunc(id)
	}
	return nil, errors.New("not implemented")
}

func (m *mockJobManager) GetOutput(id string) (string, error) {
	if m.getOutputFunc != nil {
		return m.getOutputFunc(id)
	}
	return "", errors.New("not implemented")
}

func (m *mockJobManager) Notify(id string, message string) error {
	if m.notifyFunc != nil {
		return m.notifyFunc(id, message)
	}
	return errors.New("not implemented")
}

// mockNodeManager implements dao.NodeManager for testing
type mockNodeManager struct {
	listFunc     func(*dao.ListNodesOptions) (*dao.NodeList, error)
	getFunc      func(string) (*dao.Node, error)
	drainFunc    func(string, string) error
	resumeFunc   func(string) error
	setStateFunc func(string, string) error
}

func (m *mockNodeManager) List(opts *dao.ListNodesOptions) (*dao.NodeList, error) {
	if m.listFunc != nil {
		return m.listFunc(opts)
	}
	return &dao.NodeList{Nodes: []*dao.Node{}, Total: 0}, nil
}

func (m *mockNodeManager) Get(name string) (*dao.Node, error) {
	if m.getFunc != nil {
		return m.getFunc(name)
	}
	return nil, errors.New("not implemented")
}

func (m *mockNodeManager) Drain(name string, reason string) error {
	if m.drainFunc != nil {
		return m.drainFunc(name, reason)
	}
	return errors.New("not implemented")
}

func (m *mockNodeManager) Resume(name string) error {
	if m.resumeFunc != nil {
		return m.resumeFunc(name)
	}
	return errors.New("not implemented")
}

func (m *mockNodeManager) SetState(name string, state string) error {
	if m.setStateFunc != nil {
		return m.setStateFunc(name, state)
	}
	return errors.New("not implemented")
}

// mockInfoManager implements dao.InfoManager for testing
type mockInfoManager struct {
	getClusterInfoFunc func() (*dao.ClusterInfo, error)
	getStatsFunc       func() (*dao.ClusterMetrics, error)
}

func (m *mockInfoManager) GetClusterInfo() (*dao.ClusterInfo, error) {
	if m.getClusterInfoFunc != nil {
		return m.getClusterInfoFunc()
	}
	return nil, errors.New("not implemented")
}

func (m *mockInfoManager) GetStats() (*dao.ClusterMetrics, error) {
	if m.getStatsFunc != nil {
		return m.getStatsFunc()
	}
	return nil, errors.New("not implemented")
}

// mockPartitionManager implements dao.PartitionManager for testing
type mockPartitionManager struct {
	listFunc func() (*dao.PartitionList, error)
	getFunc  func(string) (*dao.Partition, error)
}

func (m *mockPartitionManager) List() (*dao.PartitionList, error) {
	if m.listFunc != nil {
		return m.listFunc()
	}
	return &dao.PartitionList{Partitions: []*dao.Partition{}}, nil
}

func (m *mockPartitionManager) Get(name string) (*dao.Partition, error) {
	if m.getFunc != nil {
		return m.getFunc(name)
	}
	return nil, errors.New("not implemented")
}

// mockReservationManager implements dao.ReservationManager for testing
type mockReservationManager struct {
	listFunc func() (*dao.ReservationList, error)
	getFunc  func(string) (*dao.Reservation, error)
}

func (m *mockReservationManager) List() (*dao.ReservationList, error) {
	if m.listFunc != nil {
		return m.listFunc()
	}
	return &dao.ReservationList{Reservations: []*dao.Reservation{}}, nil
}

func (m *mockReservationManager) Get(name string) (*dao.Reservation, error) {
	if m.getFunc != nil {
		return m.getFunc(name)
	}
	return nil, errors.New("not implemented")
}

// mockQoSManager implements dao.QoSManager for testing
type mockQoSManager struct {
	listFunc func() (*dao.QoSList, error)
	getFunc  func(string) (*dao.QoS, error)
}

func (m *mockQoSManager) List() (*dao.QoSList, error) {
	if m.listFunc != nil {
		return m.listFunc()
	}
	return &dao.QoSList{QoS: []*dao.QoS{}}, nil
}

func (m *mockQoSManager) Get(name string) (*dao.QoS, error) {
	if m.getFunc != nil {
		return m.getFunc(name)
	}
	return nil, errors.New("not implemented")
}

// mockAccountManager implements dao.AccountManager for testing
type mockAccountManager struct {
	listFunc func() (*dao.AccountList, error)
	getFunc  func(string) (*dao.Account, error)
}

func (m *mockAccountManager) List() (*dao.AccountList, error) {
	if m.listFunc != nil {
		return m.listFunc()
	}
	return &dao.AccountList{Accounts: []*dao.Account{}}, nil
}

func (m *mockAccountManager) Get(name string) (*dao.Account, error) {
	if m.getFunc != nil {
		return m.getFunc(name)
	}
	return nil, errors.New("not implemented")
}

// mockUserManager implements dao.UserManager for testing
type mockUserManager struct {
	listFunc func() (*dao.UserList, error)
	getFunc  func(string) (*dao.User, error)
}

func (m *mockUserManager) List() (*dao.UserList, error) {
	if m.listFunc != nil {
		return m.listFunc()
	}
	return &dao.UserList{Users: []*dao.User{}}, nil
}

func (m *mockUserManager) Get(name string) (*dao.User, error) {
	if m.getFunc != nil {
		return m.getFunc(name)
	}
	return nil, errors.New("not implemented")
}

// newMockSlurmClient creates a new mock SLURM client with default implementations
func newMockSlurmClient() *mockSlurmClient {
	return &mockSlurmClient{
		jobs:         &mockJobManager{},
		nodes:        &mockNodeManager{},
		info:         &mockInfoManager{},
		partitions:   &mockPartitionManager{},
		reservations: &mockReservationManager{},
		qos:          &mockQoSManager{},
		accounts:     &mockAccountManager{},
		users:        &mockUserManager{},
	}
}

// TestNewHealthMonitor tests the HealthMonitor constructor
func TestNewHealthMonitor(t *testing.T) {
	client := newMockSlurmClient()
	interval := 30 * time.Second

	hm := NewHealthMonitor(client, interval)

	require.NotNil(t, hm, "HealthMonitor should not be nil")
	assert.Equal(t, interval, hm.interval, "Interval should match")
	assert.NotNil(t, hm.health, "ClusterHealth should be initialized")
	assert.Equal(t, HealthStatusUnknown, hm.health.OverallStatus, "Initial status should be unknown")
	assert.NotNil(t, hm.checks, "Checks map should be initialized")
	assert.NotNil(t, hm.alertManager, "AlertManager should be initialized")
	assert.False(t, hm.running, "Should not be running initially")

	// Verify default checks are registered
	assert.Contains(t, hm.checks, "nodes", "Nodes check should be registered")
	assert.Contains(t, hm.checks, "queue", "Queue check should be registered")
	assert.Contains(t, hm.checks, "utilization", "Utilization check should be registered")
}

// TestHealthMonitorStartStop tests the Start and Stop lifecycle
func TestHealthMonitorStartStop(t *testing.T) {
	client := newMockSlurmClient()
	hm := NewHealthMonitor(client, 1*time.Hour) // Long interval to avoid automatic runs

	// Test initial state
	assert.False(t, hm.running, "Should not be running initially")

	// Test start
	hm.Start()
	time.Sleep(10 * time.Millisecond) // Give goroutine time to start
	assert.True(t, hm.running, "Should be running after Start")

	// Test multiple starts (should be idempotent)
	hm.Start()
	assert.True(t, hm.running, "Should still be running after second Start")

	// Test stop
	hm.Stop()
	assert.False(t, hm.running, "Should not be running after Stop")

	// Test multiple stops (should be idempotent)
	hm.Stop()
	assert.False(t, hm.running, "Should still not be running after second Stop")
}

// TestGetHealth tests retrieving cluster health
func TestGetHealth(t *testing.T) {
	client := newMockSlurmClient()
	hm := NewHealthMonitor(client, 30*time.Second)

	// Set some test data in the health monitor
	hm.health.mu.Lock()
	hm.health.OverallStatus = HealthStatusHealthy
	hm.health.Checks["test"] = &HealthCheck{
		Name:       "test",
		Status:     HealthStatusHealthy,
		CheckCount: 5,
	}
	hm.health.Issues = []HealthIssue{
		{ID: "issue1", Component: "test"},
	}
	hm.health.LastUpdated = time.Now()
	hm.health.mu.Unlock()

	// Get health (should return a copy)
	health := hm.GetHealth()

	require.NotNil(t, health, "Health should not be nil")
	assert.Equal(t, HealthStatusHealthy, health.OverallStatus, "Status should match")
	assert.Len(t, health.Checks, 1, "Should have one check")
	assert.Len(t, health.Issues, 1, "Should have one issue")

	// Verify it's a copy by modifying the returned value
	health.OverallStatus = HealthStatusCritical
	assert.Equal(t, HealthStatusHealthy, hm.health.OverallStatus, "Original should not be modified")
}

// TestPerformHealthChecks tests the health check execution
func TestPerformHealthChecks(t *testing.T) {
	client := newMockSlurmClient()

	// Setup mock responses
	client.nodes.listFunc = func(opts *dao.ListNodesOptions) (*dao.NodeList, error) {
		return &dao.NodeList{
			Nodes: []*dao.Node{
				{Name: "node1", State: dao.NodeStateIdle},
				{Name: "node2", State: dao.NodeStateIdle},
			},
			Total: 2,
		}, nil
	}

	client.jobs.listFunc = func(opts *dao.ListJobsOptions) (*dao.JobList, error) {
		return &dao.JobList{
			Jobs:  []*dao.Job{{ID: "1"}, {ID: "2"}},
			Total: 2,
		}, nil
	}

	client.info.getStatsFunc = func() (*dao.ClusterMetrics, error) {
		return &dao.ClusterMetrics{
			CPUUsage:    50.0,
			MemoryUsage: 60.0,
		}, nil
	}

	hm := NewHealthMonitor(client, 30*time.Second)

	// Perform initial check
	hm.performHealthChecks()

	health := hm.GetHealth()
	assert.NotEqual(t, time.Time{}, health.LastUpdated, "LastUpdated should be set")
	assert.Len(t, health.Checks, 3, "Should have 3 checks")
	assert.Contains(t, health.Checks, "nodes", "Should have nodes check")
	assert.Contains(t, health.Checks, "queue", "Should have queue check")
	assert.Contains(t, health.Checks, "utilization", "Should have utilization check")

	// All checks should have count = 1
	for name, check := range health.Checks {
		assert.Equal(t, 1, check.CheckCount, "Check %s should have count 1", name)
	}
}

// TestCounterAccumulation tests that check counts increment properly across runs
func TestCounterAccumulation(t *testing.T) {
	client := newMockSlurmClient()

	// Setup mock responses
	client.nodes.listFunc = func(opts *dao.ListNodesOptions) (*dao.NodeList, error) {
		return &dao.NodeList{
			Nodes: []*dao.Node{{Name: "node1", State: dao.NodeStateIdle}},
			Total: 1,
		}, nil
	}

	client.jobs.listFunc = func(opts *dao.ListJobsOptions) (*dao.JobList, error) {
		return &dao.JobList{Jobs: []*dao.Job{}, Total: 0}, nil
	}

	client.info.getStatsFunc = func() (*dao.ClusterMetrics, error) {
		return &dao.ClusterMetrics{CPUUsage: 50.0, MemoryUsage: 60.0}, nil
	}

	hm := NewHealthMonitor(client, 30*time.Second)

	// Perform multiple health checks
	for i := 1; i <= 5; i++ {
		hm.performHealthChecks()

		health := hm.GetHealth()
		for name, check := range health.Checks {
			assert.Equal(t, i, check.CheckCount, "Check %s should have count %d after run %d", name, i, i)
		}
	}

	// Verify final counts
	health := hm.GetHealth()
	assert.Equal(t, 5, health.Checks["nodes"].CheckCount, "Nodes check should have count 5")
	assert.Equal(t, 5, health.Checks["queue"].CheckCount, "Queue check should have count 5")
	assert.Equal(t, 5, health.Checks["utilization"].CheckCount, "Utilization check should have count 5")
}

// TestUpdateOverallStatus tests overall status aggregation
func TestUpdateOverallStatus(t *testing.T) {
	tests := []struct {
		name           string
		checkStatuses  map[string]HealthStatus
		expectedStatus HealthStatus
	}{
		{
			name: "all healthy",
			checkStatuses: map[string]HealthStatus{
				"check1": HealthStatusHealthy,
				"check2": HealthStatusHealthy,
			},
			expectedStatus: HealthStatusHealthy,
		},
		{
			name: "one warning",
			checkStatuses: map[string]HealthStatus{
				"check1": HealthStatusHealthy,
				"check2": HealthStatusWarning,
			},
			expectedStatus: HealthStatusWarning,
		},
		{
			name: "one critical",
			checkStatuses: map[string]HealthStatus{
				"check1": HealthStatusHealthy,
				"check2": HealthStatusCritical,
			},
			expectedStatus: HealthStatusCritical,
		},
		{
			name: "critical overrides warning",
			checkStatuses: map[string]HealthStatus{
				"check1": HealthStatusWarning,
				"check2": HealthStatusCritical,
			},
			expectedStatus: HealthStatusCritical,
		},
		{
			name: "unknown status",
			checkStatuses: map[string]HealthStatus{
				"check1": HealthStatusHealthy,
				"check2": HealthStatusUnknown,
			},
			expectedStatus: HealthStatusUnknown,
		},
		{
			name: "critical overrides unknown",
			checkStatuses: map[string]HealthStatus{
				"check1": HealthStatusUnknown,
				"check2": HealthStatusCritical,
			},
			expectedStatus: HealthStatusCritical,
		},
		{
			name:           "no checks",
			checkStatuses:  map[string]HealthStatus{},
			expectedStatus: HealthStatusHealthy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newMockSlurmClient()
			hm := NewHealthMonitor(client, 30*time.Second)

			// Set up checks
			hm.health.mu.Lock()
			for name, status := range tt.checkStatuses {
				hm.health.Checks[name] = &HealthCheck{
					Name:   name,
					Status: status,
				}
			}
			hm.updateOverallStatus()
			actualStatus := hm.health.OverallStatus
			hm.health.mu.Unlock()

			assert.Equal(t, tt.expectedStatus, actualStatus, "Overall status should match")
		})
	}
}

// TestCheckNodes tests the node health check
func TestCheckNodes(t *testing.T) {
	tests := []struct {
		name            string
		nodes           []*dao.Node
		err             error
		expectedStatus  HealthStatus
		expectedMessage string
	}{
		{
			name: "all nodes healthy",
			nodes: []*dao.Node{
				{Name: "node1", State: dao.NodeStateIdle},
				{Name: "node2", State: dao.NodeStateAllocated},
				{Name: "node3", State: dao.NodeStateMixed},
			},
			expectedStatus:  HealthStatusHealthy,
			expectedMessage: "All nodes healthy (3 total, 0 down, 0 drain)",
		},
		{
			name: "warning level - 20% unavailable",
			nodes: []*dao.Node{
				{Name: "node1", State: dao.NodeStateIdle},
				{Name: "node2", State: dao.NodeStateIdle},
				{Name: "node3", State: dao.NodeStateIdle},
				{Name: "node4", State: dao.NodeStateIdle},
				{Name: "node5", State: dao.NodeStateIdle},
				{Name: "node6", State: dao.NodeStateIdle},
				{Name: "node7", State: dao.NodeStateIdle},
				{Name: "node8", State: dao.NodeStateIdle},
				{Name: "node9", State: dao.NodeStateIdle},
				{Name: "node10", State: dao.NodeStateIdle},
				{Name: "node11", State: dao.NodeStateIdle},
				{Name: "node12", State: dao.NodeStateIdle},
				{Name: "node13", State: dao.NodeStateIdle},
				{Name: "node14", State: dao.NodeStateIdle},
				{Name: "node15", State: dao.NodeStateIdle},
				{Name: "node16", State: dao.NodeStateIdle},
				{Name: "node17", State: dao.NodeStateDown},
				{Name: "node18", State: dao.NodeStateDown},
				{Name: "node19", State: dao.NodeStateDrain},
				{Name: "node20", State: dao.NodeStateDown},
			},
			expectedStatus:  HealthStatusWarning,
			expectedMessage: "20.0% of nodes unavailable (3 down, 1 drain out of 20 total)",
		},
		{
			name: "critical level - 30% unavailable",
			nodes: []*dao.Node{
				{Name: "node1", State: dao.NodeStateIdle},
				{Name: "node2", State: dao.NodeStateIdle},
				{Name: "node3", State: dao.NodeStateIdle},
				{Name: "node4", State: dao.NodeStateIdle},
				{Name: "node5", State: dao.NodeStateIdle},
				{Name: "node6", State: dao.NodeStateIdle},
				{Name: "node7", State: dao.NodeStateIdle},
				{Name: "node8", State: dao.NodeStateDown},
				{Name: "node9", State: dao.NodeStateDrain},
				{Name: "node10", State: dao.NodeStateDraining},
			},
			expectedStatus:  HealthStatusCritical,
			expectedMessage: "30.0% of nodes unavailable (1 down, 2 drain out of 10 total)",
		},
		{
			name:            "no nodes - critical",
			nodes:           []*dao.Node{},
			expectedStatus:  HealthStatusCritical,
			expectedMessage: "No nodes found in cluster",
		},
		{
			name:            "error retrieving nodes",
			err:             errors.New("connection failed"),
			expectedStatus:  HealthStatusUnknown,
			expectedMessage: "Failed to get node list: connection failed",
		},
		{
			name: "nil node in list",
			nodes: []*dao.Node{
				{Name: "node1", State: dao.NodeStateIdle},
				nil,
				{Name: "node3", State: dao.NodeStateIdle},
			},
			expectedStatus:  HealthStatusHealthy,
			expectedMessage: "All nodes healthy (3 total, 0 down, 0 drain)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newMockSlurmClient()
			client.nodes.listFunc = func(opts *dao.ListNodesOptions) (*dao.NodeList, error) {
				if tt.err != nil {
					return nil, tt.err
				}
				return &dao.NodeList{Nodes: tt.nodes, Total: len(tt.nodes)}, nil
			}

			hm := NewHealthMonitor(client, 30*time.Second)
			check := hm.checkNodes(client)

			require.NotNil(t, check, "Check should not be nil")
			assert.Equal(t, "nodes", check.Name, "Check name should be 'nodes'")
			assert.Equal(t, tt.expectedStatus, check.Status, "Status should match")
			assert.Contains(t, check.Message, tt.expectedMessage, "Message should match")
			assert.NotZero(t, check.LastCheck, "LastCheck should be set")
			assert.NotNil(t, check.Threshold.WarningMax, "Warning threshold should be set")
			assert.NotNil(t, check.Threshold.CriticalMax, "Critical threshold should be set")
		})
	}
}

// TestCheckQueue tests the job queue health check
func TestCheckQueue(t *testing.T) {
	tests := []struct {
		name            string
		pendingJobs     int
		err             error
		expectedStatus  HealthStatus
		expectedMessage string
	}{
		{
			name:            "healthy queue - no pending jobs",
			pendingJobs:     0,
			expectedStatus:  HealthStatusHealthy,
			expectedMessage: "Queue healthy with 0 pending jobs",
		},
		{
			name:            "healthy queue - 50 pending jobs",
			pendingJobs:     50,
			expectedStatus:  HealthStatusHealthy,
			expectedMessage: "Queue healthy with 50 pending jobs",
		},
		{
			name:            "warning level - 150 pending jobs",
			pendingJobs:     150,
			expectedStatus:  HealthStatusWarning,
			expectedMessage: "150 pending jobs (threshold: warning 100, critical 500)",
		},
		{
			name:            "warning level - 200 pending jobs",
			pendingJobs:     200,
			expectedStatus:  HealthStatusWarning,
			expectedMessage: "200 pending jobs (threshold: warning 100, critical 500)",
		},
		{
			name:            "critical level - 600 pending jobs",
			pendingJobs:     600,
			expectedStatus:  HealthStatusCritical,
			expectedMessage: "600 pending jobs (threshold: warning 100, critical 500)",
		},
		{
			name:            "critical level - 1000 pending jobs",
			pendingJobs:     1000,
			expectedStatus:  HealthStatusCritical,
			expectedMessage: "1000 pending jobs (threshold: warning 100, critical 500)",
		},
		{
			name:            "error retrieving jobs",
			err:             errors.New("API error"),
			expectedStatus:  HealthStatusUnknown,
			expectedMessage: "Failed to get job list: API error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newMockSlurmClient()
			client.jobs.listFunc = func(opts *dao.ListJobsOptions) (*dao.JobList, error) {
				if tt.err != nil {
					return nil, tt.err
				}

				// Verify the filter is for pending jobs
				assert.Equal(t, []string{dao.JobStatePending}, opts.States, "Should filter for pending jobs")

				jobs := make([]*dao.Job, tt.pendingJobs)
				for i := 0; i < tt.pendingJobs; i++ {
					jobs[i] = &dao.Job{ID: string(rune(i))}
				}
				return &dao.JobList{Jobs: jobs, Total: tt.pendingJobs}, nil
			}

			hm := NewHealthMonitor(client, 30*time.Second)
			check := hm.checkQueue(client)

			require.NotNil(t, check, "Check should not be nil")
			assert.Equal(t, "queue", check.Name, "Check name should be 'queue'")
			assert.Equal(t, tt.expectedStatus, check.Status, "Status should match")
			assert.Contains(t, check.Message, tt.expectedMessage, "Message should match")
			assert.NotZero(t, check.LastCheck, "LastCheck should be set")
		})
	}
}

// TestCheckUtilization tests the resource utilization health check
func TestCheckUtilization(t *testing.T) {
	tests := []struct {
		name            string
		cpuUsage        float64
		memoryUsage     float64
		err             error
		nilInfoMgr      bool
		expectedStatus  HealthStatus
		expectedMessage string
	}{
		{
			name:            "healthy utilization",
			cpuUsage:        50.0,
			memoryUsage:     60.0,
			expectedStatus:  HealthStatusHealthy,
			expectedMessage: "Resource utilization healthy: CPU 50.0%, Memory 60.0%",
		},
		{
			name:            "warning level - CPU at 92%",
			cpuUsage:        92.0,
			memoryUsage:     50.0,
			expectedStatus:  HealthStatusWarning,
			expectedMessage: "High resource utilization: CPU 92.0%, Memory 50.0% (max 92.0%)",
		},
		{
			name:            "warning level - Memory at 91%",
			cpuUsage:        50.0,
			memoryUsage:     91.0,
			expectedStatus:  HealthStatusWarning,
			expectedMessage: "High resource utilization: CPU 50.0%, Memory 91.0% (max 91.0%)",
		},
		{
			name:            "critical level - CPU at 97%",
			cpuUsage:        97.0,
			memoryUsage:     50.0,
			expectedStatus:  HealthStatusCritical,
			expectedMessage: "High resource utilization: CPU 97.0%, Memory 50.0% (max 97.0%)",
		},
		{
			name:            "critical level - Memory at 96%",
			cpuUsage:        50.0,
			memoryUsage:     96.0,
			expectedStatus:  HealthStatusCritical,
			expectedMessage: "High resource utilization: CPU 50.0%, Memory 96.0% (max 96.0%)",
		},
		{
			name:            "memory unavailable - healthy CPU",
			cpuUsage:        50.0,
			memoryUsage:     -1.0,
			expectedStatus:  HealthStatusHealthy,
			expectedMessage: "CPU utilization healthy: 50.0% (memory data unavailable)",
		},
		{
			name:            "memory unavailable - critical CPU",
			cpuUsage:        96.0,
			memoryUsage:     -1.0,
			expectedStatus:  HealthStatusCritical,
			expectedMessage: "High CPU utilization: 96.0% (memory data unavailable)",
		},
		{
			name:            "nil info manager",
			nilInfoMgr:      true,
			expectedStatus:  HealthStatusUnknown,
			expectedMessage: "Cluster metrics not available",
		},
		{
			name:            "error retrieving metrics",
			err:             errors.New("stats error"),
			expectedStatus:  HealthStatusUnknown,
			expectedMessage: "Failed to get cluster metrics: stats error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newMockSlurmClient()

			if tt.nilInfoMgr {
				// Override Info() to return nil
				client.infoFunc = func() dao.InfoManager {
					return nil
				}
			} else {
				client.info.getStatsFunc = func() (*dao.ClusterMetrics, error) {
					if tt.err != nil {
						return nil, tt.err
					}
					return &dao.ClusterMetrics{
						CPUUsage:    tt.cpuUsage,
						MemoryUsage: tt.memoryUsage,
					}, nil
				}
			}

			hm := NewHealthMonitor(client, 30*time.Second)
			check := hm.checkUtilization(client)

			require.NotNil(t, check, "Check should not be nil")
			assert.Equal(t, "utilization", check.Name, "Check name should be 'utilization'")
			assert.Equal(t, tt.expectedStatus, check.Status, "Status should match")
			assert.Contains(t, check.Message, tt.expectedMessage, "Message should match")
			assert.NotZero(t, check.LastCheck, "LastCheck should be set")
		})
	}
}

// TestSetCheckStatus tests status threshold logic
func TestSetCheckStatus(t *testing.T) {
	tests := []struct {
		name           string
		value          float64
		threshold      HealthThreshold
		expectedStatus HealthStatus
		expectCritical bool
		expectHealthy  bool
	}{
		{
			name:  "below warning threshold",
			value: 5.0,
			threshold: HealthThreshold{
				WarningMax:  floatPtr(10.0),
				CriticalMax: floatPtr(20.0),
			},
			expectedStatus: HealthStatusHealthy,
			expectHealthy:  true,
		},
		{
			name:  "just above warning threshold",
			value: 10.1,
			threshold: HealthThreshold{
				WarningMax:  floatPtr(10.0),
				CriticalMax: floatPtr(20.0),
			},
			expectedStatus: HealthStatusWarning,
			expectCritical: true,
		},
		{
			name:  "above warning threshold",
			value: 15.0,
			threshold: HealthThreshold{
				WarningMax:  floatPtr(10.0),
				CriticalMax: floatPtr(20.0),
			},
			expectedStatus: HealthStatusWarning,
			expectCritical: true,
		},
		{
			name:  "just above critical threshold",
			value: 20.1,
			threshold: HealthThreshold{
				WarningMax:  floatPtr(10.0),
				CriticalMax: floatPtr(20.0),
			},
			expectedStatus: HealthStatusCritical,
			expectCritical: true,
		},
		{
			name:  "above critical threshold",
			value: 25.0,
			threshold: HealthThreshold{
				WarningMax:  floatPtr(10.0),
				CriticalMax: floatPtr(20.0),
			},
			expectedStatus: HealthStatusCritical,
			expectCritical: true,
		},
		{
			name:  "no thresholds set",
			value: 100.0,
			threshold: HealthThreshold{
				WarningMax:  nil,
				CriticalMax: nil,
			},
			expectedStatus: HealthStatusHealthy,
			expectHealthy:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newMockSlurmClient()
			hm := NewHealthMonitor(client, 30*time.Second)

			check := &HealthCheck{
				Name:      "test",
				Threshold: tt.threshold,
			}

			criticalMsgCalled := false
			healthyMsgCalled := false

			criticalMsg := func(status HealthStatus) string {
				criticalMsgCalled = true
				return "critical message"
			}

			healthyMsg := func() string {
				healthyMsgCalled = true
				return "healthy message"
			}

			hm.setCheckStatus(check, tt.value, criticalMsg, healthyMsg)

			assert.Equal(t, tt.expectedStatus, check.Status, "Status should match")
			assert.NotEmpty(t, check.Message, "Message should be set")

			if tt.expectCritical {
				assert.True(t, criticalMsgCalled, "Critical message function should be called")
				assert.False(t, healthyMsgCalled, "Healthy message function should not be called")
			}

			if tt.expectHealthy {
				assert.False(t, criticalMsgCalled, "Critical message function should not be called")
				assert.True(t, healthyMsgCalled, "Healthy message function should be called")
			}
		})
	}
}

// TestCountNodeStates tests the node state counting helper
func TestCountNodeStates(t *testing.T) {
	tests := []struct {
		name          string
		nodes         []*dao.Node
		expectedDown  int
		expectedDrain int
	}{
		{
			name:          "empty node list",
			nodes:         []*dao.Node{},
			expectedDown:  0,
			expectedDrain: 0,
		},
		{
			name: "all healthy nodes",
			nodes: []*dao.Node{
				{State: dao.NodeStateIdle},
				{State: dao.NodeStateAllocated},
				{State: dao.NodeStateMixed},
			},
			expectedDown:  0,
			expectedDrain: 0,
		},
		{
			name: "down nodes only",
			nodes: []*dao.Node{
				{State: dao.NodeStateDown},
				{State: dao.NodeStateDown},
				{State: dao.NodeStateIdle},
			},
			expectedDown:  2,
			expectedDrain: 0,
		},
		{
			name: "drain nodes only",
			nodes: []*dao.Node{
				{State: dao.NodeStateDrain},
				{State: dao.NodeStateDraining},
				{State: dao.NodeStateIdle},
			},
			expectedDown:  0,
			expectedDrain: 2,
		},
		{
			name: "mixed down and drain",
			nodes: []*dao.Node{
				{State: dao.NodeStateDown},
				{State: dao.NodeStateDrain},
				{State: dao.NodeStateDraining},
				{State: dao.NodeStateIdle},
			},
			expectedDown:  1,
			expectedDrain: 2,
		},
		{
			name: "nil node in list",
			nodes: []*dao.Node{
				{State: dao.NodeStateDown},
				nil,
				{State: dao.NodeStateDrain},
			},
			expectedDown:  1,
			expectedDrain: 1,
		},
		{
			name: "all nil nodes",
			nodes: []*dao.Node{
				nil,
				nil,
				nil,
			},
			expectedDown:  0,
			expectedDrain: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			down, drain := countNodeStates(tt.nodes)
			assert.Equal(t, tt.expectedDown, down, "Down count should match")
			assert.Equal(t, tt.expectedDrain, drain, "Drain count should match")
		})
	}
}

// TestHealthyMessageGeneration verifies that healthy states have explicit messages
func TestHealthyMessageGeneration(t *testing.T) {
	client := newMockSlurmClient()

	// Setup healthy responses
	client.nodes.listFunc = func(opts *dao.ListNodesOptions) (*dao.NodeList, error) {
		return &dao.NodeList{
			Nodes: []*dao.Node{
				{Name: "node1", State: dao.NodeStateIdle},
				{Name: "node2", State: dao.NodeStateIdle},
			},
			Total: 2,
		}, nil
	}

	client.jobs.listFunc = func(opts *dao.ListJobsOptions) (*dao.JobList, error) {
		return &dao.JobList{
			Jobs:  []*dao.Job{{ID: "1"}},
			Total: 1,
		}, nil
	}

	client.info.getStatsFunc = func() (*dao.ClusterMetrics, error) {
		return &dao.ClusterMetrics{
			CPUUsage:    45.0,
			MemoryUsage: 55.0,
		}, nil
	}

	hm := NewHealthMonitor(client, 30*time.Second)
	hm.performHealthChecks()

	health := hm.GetHealth()

	// Verify all checks are healthy
	for name, check := range health.Checks {
		assert.Equal(t, HealthStatusHealthy, check.Status, "Check %s should be healthy", name)
		assert.NotEmpty(t, check.Message, "Check %s should have a message", name)
		assert.NotEqual(t, "", check.Message, "Check %s message should not be empty string", name)
	}

	// Verify specific messages
	assert.Contains(t, health.Checks["nodes"].Message, "All nodes healthy", "Nodes check should have healthy message")
	assert.Contains(t, health.Checks["queue"].Message, "Queue healthy", "Queue check should have healthy message")
	assert.Contains(t, health.Checks["utilization"].Message, "Resource utilization healthy", "Utilization check should have healthy message")
}

// TestCriticalMessageGeneration verifies critical/warning states have appropriate messages
func TestCriticalMessageGeneration(t *testing.T) {
	client := newMockSlurmClient()

	// Setup critical responses
	client.nodes.listFunc = func(opts *dao.ListNodesOptions) (*dao.NodeList, error) {
		return &dao.NodeList{
			Nodes: []*dao.Node{
				{Name: "node1", State: dao.NodeStateDown},
				{Name: "node2", State: dao.NodeStateDown},
				{Name: "node3", State: dao.NodeStateDown},
			},
			Total: 3,
		}, nil
	}

	client.jobs.listFunc = func(opts *dao.ListJobsOptions) (*dao.JobList, error) {
		jobs := make([]*dao.Job, 600)
		for i := 0; i < 600; i++ {
			jobs[i] = &dao.Job{ID: string(rune(i))}
		}
		return &dao.JobList{Jobs: jobs, Total: 600}, nil
	}

	client.info.getStatsFunc = func() (*dao.ClusterMetrics, error) {
		return &dao.ClusterMetrics{
			CPUUsage:    98.0,
			MemoryUsage: 97.0,
		}, nil
	}

	hm := NewHealthMonitor(client, 30*time.Second)
	hm.performHealthChecks()

	health := hm.GetHealth()

	// Verify all checks are critical
	for name, check := range health.Checks {
		assert.Equal(t, HealthStatusCritical, check.Status, "Check %s should be critical", name)
		assert.NotEmpty(t, check.Message, "Check %s should have a message", name)
	}

	// Verify specific critical messages contain expected text
	assert.Contains(t, health.Checks["nodes"].Message, "100.0% of nodes unavailable", "Nodes check should mention unavailability")
	assert.Contains(t, health.Checks["queue"].Message, "600 pending jobs", "Queue check should mention pending count")
	assert.Contains(t, health.Checks["utilization"].Message, "High resource utilization", "Utilization check should mention high usage")
}

// TestGenerateAlert verifies alert generation for critical/warning checks
func TestGenerateAlert(t *testing.T) {
	client := newMockSlurmClient()
	hm := NewHealthMonitor(client, 30*time.Second)

	check := &HealthCheck{
		Name:    "test-check",
		Status:  HealthStatusCritical,
		Message: "Test critical message",
	}

	hm.generateAlert(check)

	alerts := hm.alertManager.GetActiveAlerts()
	assert.Len(t, alerts, 1, "Should have generated one alert")

	alert := alerts[0]
	assert.Equal(t, AlertTypeHealth, alert.Type, "Alert type should be health")
	assert.Equal(t, AlertSeverity(HealthStatusCritical), alert.Severity, "Severity should match check status")
	assert.Contains(t, alert.Title, "test-check", "Alert title should contain check name")
	assert.Equal(t, "Test critical message", alert.Message, "Alert message should match check message")
	assert.Equal(t, "test-check", alert.Component, "Component should match check name")
	assert.False(t, alert.Acknowledged, "Alert should not be acknowledged")
}

// TestFloatPtr tests the float pointer helper function
func TestFloatPtr(t *testing.T) {
	val := floatPtr(42.5)
	require.NotNil(t, val, "Pointer should not be nil")
	assert.Equal(t, 42.5, *val, "Value should match")

	// Verify it creates distinct pointers
	val1 := floatPtr(10.0)
	val2 := floatPtr(10.0)
	assert.NotSame(t, val1, val2, "Should create distinct pointers")
	assert.Equal(t, *val1, *val2, "Values should be equal")
}

// TestGetAlertManager verifies the alert manager getter
func TestGetAlertManager(t *testing.T) {
	client := newMockSlurmClient()
	hm := NewHealthMonitor(client, 30*time.Second)

	am := hm.GetAlertManager()
	require.NotNil(t, am, "AlertManager should not be nil")
	assert.Same(t, hm.alertManager, am, "Should return the same alert manager instance")
}

// TestHealthCheckThresholds verifies threshold values are set correctly
func TestHealthCheckThresholds(t *testing.T) {
	client := newMockSlurmClient()
	hm := NewHealthMonitor(client, 30*time.Second)

	// Check nodes thresholds
	client.nodes.listFunc = func(opts *dao.ListNodesOptions) (*dao.NodeList, error) {
		return &dao.NodeList{Nodes: []*dao.Node{{State: dao.NodeStateIdle}}, Total: 1}, nil
	}
	nodesCheck := hm.checkNodes(client)
	assert.NotNil(t, nodesCheck.Threshold.WarningMax, "Nodes warning threshold should be set")
	assert.NotNil(t, nodesCheck.Threshold.CriticalMax, "Nodes critical threshold should be set")
	assert.Equal(t, 10.0, *nodesCheck.Threshold.WarningMax, "Nodes warning threshold should be 10%")
	assert.Equal(t, 25.0, *nodesCheck.Threshold.CriticalMax, "Nodes critical threshold should be 25%")

	// Check queue thresholds
	client.jobs.listFunc = func(opts *dao.ListJobsOptions) (*dao.JobList, error) {
		return &dao.JobList{Jobs: []*dao.Job{}, Total: 0}, nil
	}
	queueCheck := hm.checkQueue(client)
	assert.NotNil(t, queueCheck.Threshold.WarningMax, "Queue warning threshold should be set")
	assert.NotNil(t, queueCheck.Threshold.CriticalMax, "Queue critical threshold should be set")
	assert.Equal(t, 100.0, *queueCheck.Threshold.WarningMax, "Queue warning threshold should be 100")
	assert.Equal(t, 500.0, *queueCheck.Threshold.CriticalMax, "Queue critical threshold should be 500")

	// Check utilization thresholds
	client.info.getStatsFunc = func() (*dao.ClusterMetrics, error) {
		return &dao.ClusterMetrics{CPUUsage: 50.0, MemoryUsage: 50.0}, nil
	}
	utilCheck := hm.checkUtilization(client)
	assert.NotNil(t, utilCheck.Threshold.WarningMax, "Utilization warning threshold should be set")
	assert.NotNil(t, utilCheck.Threshold.CriticalMax, "Utilization critical threshold should be set")
	assert.Equal(t, 90.0, *utilCheck.Threshold.WarningMax, "Utilization warning threshold should be 90%")
	assert.Equal(t, 95.0, *utilCheck.Threshold.CriticalMax, "Utilization critical threshold should be 95%")
}

// TestEdgeCases tests various edge cases
func TestEdgeCases(t *testing.T) {
	t.Run("zero values", func(t *testing.T) {
		client := newMockSlurmClient()
		client.info.getStatsFunc = func() (*dao.ClusterMetrics, error) {
			return &dao.ClusterMetrics{CPUUsage: 0.0, MemoryUsage: 0.0}, nil
		}

		hm := NewHealthMonitor(client, 30*time.Second)
		check := hm.checkUtilization(client)
		assert.Equal(t, HealthStatusHealthy, check.Status, "Zero utilization should be healthy")
	})

	t.Run("exact threshold boundaries", func(t *testing.T) {
		client := newMockSlurmClient()

		// Create exactly 9 nodes with 1 down (11.11% - just above warning threshold of 10%)
		nodes := make([]*dao.Node, 9)
		for i := 0; i < 8; i++ {
			nodes[i] = &dao.Node{State: dao.NodeStateIdle}
		}
		nodes[8] = &dao.Node{State: dao.NodeStateDown}

		client.nodes.listFunc = func(opts *dao.ListNodesOptions) (*dao.NodeList, error) {
			return &dao.NodeList{Nodes: nodes, Total: 9}, nil
		}

		hm := NewHealthMonitor(client, 30*time.Second)
		check := hm.checkNodes(client)
		assert.Equal(t, HealthStatusWarning, check.Status, "Just above threshold should be warning")
	})

	t.Run("negative memory usage", func(t *testing.T) {
		client := newMockSlurmClient()
		client.info.getStatsFunc = func() (*dao.ClusterMetrics, error) {
			return &dao.ClusterMetrics{CPUUsage: 50.0, MemoryUsage: -1.0}, nil
		}

		hm := NewHealthMonitor(client, 30*time.Second)
		check := hm.checkUtilization(client)
		assert.Contains(t, check.Message, "memory data unavailable", "Should indicate memory is unavailable")
	})
}
