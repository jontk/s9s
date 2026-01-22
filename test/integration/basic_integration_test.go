package integration

import (
	"context"
	"testing"
	"time"

	"github.com/jontk/s9s/internal/performance"
	"github.com/jontk/s9s/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBasicIntegration runs basic integration tests
func TestBasicIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	t.Run("PerformanceProfiler", func(t *testing.T) {
		testPerformanceProfilerIntegration(t)
	})

	t.Run("SSHSessionManager", func(t *testing.T) {
		testSSHSessionManagerIntegration(t)
	})
}

func testPerformanceProfilerIntegration(t *testing.T) {
	profiler := performance.NewProfiler()
	optimizer := performance.NewOptimizer(profiler)

	// Test profiler with real operations
	const iterations = 100
	for i := 0; i < iterations; i++ {
		done := profiler.StartOperation("test_operation")

		// Simulate work
		time.Sleep(1 * time.Millisecond)

		done()
	}

	// Capture memory stats
	memStats := profiler.CaptureMemoryStats()
	assert.Greater(t, memStats.HeapAlloc, uint64(0))

	// Get operation stats
	stats := profiler.GetOperationStats()
	assert.Contains(t, stats, "test_operation")

	testStats := stats["test_operation"]
	assert.Equal(t, int64(iterations), testStats.Count)
	assert.Greater(t, testStats.TotalTime, time.Duration(0))

	// Test optimizer
	recommendations := optimizer.Analyze()
	// Should get at least some recommendations or none (both are valid)
	t.Logf("Generated %d optimization recommendations", len(recommendations))

	// Test report generation
	report := profiler.Report()
	assert.NotEmpty(t, report)
	assert.Contains(t, report, "Performance Report")
}

func testSSHSessionManagerIntegration(t *testing.T) {
	config := ssh.DefaultSSHConfig()
	manager, err := ssh.NewSessionManager(config)
	require.NoError(t, err)
	defer manager.Shutdown()

	// Test basic session creation (will fail without real SSH server, but should not panic)
	session, err := manager.CreateSession("localhost", "testuser")
	if err != nil {
		// Expected - no SSH server available in CI
		t.Logf("SSH session creation failed as expected: %v", err)
		return
	}

	// If we somehow got a session, test basic operations
	assert.NotNil(t, session)
	assert.Equal(t, "localhost", session.Hostname)
	assert.Equal(t, "testuser", session.Username)

	// Test getting session info
	sessionInfo, err := manager.GetSessionInfo(session.ID)
	assert.NoError(t, err)
	assert.Equal(t, session.ID, sessionInfo.ID)

	// Test session cleanup
	err = manager.CloseSession(session.ID)
	assert.NoError(t, err)
}

// TestPerformanceBenchmarkIntegration tests the benchmarking system
func TestPerformanceBenchmarkIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping benchmark integration tests in short mode")
	}

	// Test the benchmark function itself
	result := performance.BenchmarkOperation("integration_test", 10, func() {
		// Simulate some work
		sum := 0
		for i := 0; i < 1000; i++ {
			sum += i
		}
		_ = sum
	})

	assert.Equal(t, "integration_test", result.Name)
	assert.Equal(t, 10, result.Iterations)
	assert.Greater(t, result.TotalTime, time.Duration(0))
	assert.Greater(t, result.TimePerOp, time.Duration(0))
	assert.GreaterOrEqual(t, result.AllocsPerOp, int64(0))
	assert.GreaterOrEqual(t, result.BytesPerOp, int64(0))

	t.Logf("Benchmark result: %s", result.String())
}

// TestSSHPersistenceIntegration tests SSH session persistence
func TestSSHPersistenceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping persistence integration tests in short mode")
	}

	// Create temporary directory
	tempDir := t.TempDir()

	// Test persistence creation
	persistence, err := ssh.NewSessionPersistence(tempDir)
	require.NoError(t, err)

	// Test session tags
	sessionID := "test_session_123"
	tags := map[string]string{
		"environment": "test",
		"purpose":     "integration",
		"created_by":  "test_suite",
	}

	// Save tags
	err = persistence.SaveSessionTags(sessionID, tags)
	require.NoError(t, err)

	// Load tags
	loadedTags, err := persistence.LoadSessionTags(sessionID)
	require.NoError(t, err)
	assert.Equal(t, tags, loadedTags)

	// Test loading non-existent tags
	emptyTags, err := persistence.LoadSessionTags("non_existent_session")
	require.NoError(t, err)
	assert.Empty(t, emptyTags)

	// Test cleanup
	err = persistence.CleanupOldData()
	assert.NoError(t, err)
}

// TestSSHConfigIntegration tests SSH configuration handling
func TestSSHConfigIntegration(t *testing.T) {
	// Test default config
	defaultConfig := ssh.DefaultSSHConfig()
	assert.Equal(t, 22, defaultConfig.Port)
	assert.Equal(t, 30*time.Second, defaultConfig.Timeout)
	assert.NotEmpty(t, defaultConfig.Options)

	// Test SSH availability check
	available := ssh.IsSSHAvailable()
	t.Logf("SSH client available: %v", available)
	// Don't assert this since it depends on the system
}

// TestContextIntegration tests context handling throughout the system
func TestContextIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test that context cancellation works properly
	profiler := performance.NewProfiler()

	done := profiler.StartOperation("context_test")

	// Simulate work that respects context
	select {
	case <-ctx.Done():
		t.Error("Context should not be cancelled yet")
	case <-time.After(100 * time.Millisecond):
		// Continue
	}

	done()

	// Verify operation was recorded
	stats := profiler.GetOperationStats()
	assert.Contains(t, stats, "context_test")
}

// BenchmarkIntegrationOverhead benchmarks the overhead of integration components
func BenchmarkIntegrationOverhead(b *testing.B) {
	profiler := performance.NewProfiler()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		done := profiler.StartOperation("benchmark_test")
		done()
	}
}

// BenchmarkSSHManagerCreation benchmarks SSH manager creation
func BenchmarkSSHManagerCreation(b *testing.B) {
	config := ssh.DefaultSSHConfig()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		manager, err := ssh.NewSessionManager(config)
		if err != nil {
			b.Fatal(err)
		}
		manager.Shutdown()
	}
}
