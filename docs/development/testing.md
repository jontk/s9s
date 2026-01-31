# s9s Testing Guide

Comprehensive guide to testing in s9s, covering unit tests, integration tests, benchmarks, and test coverage.

## Table of Contents

- [Unit Tests](#unit-tests)
- [Integration Tests](#integration-tests)
- [Benchmarks](#benchmarks)
- [Test Coverage](#test-coverage)
- [Writing Tests](#writing-tests)
- [Running Tests](#running-tests)
- [Performance Profiling](#performance-profiling)

## Unit Tests

Unit tests verify the behavior of individual components in isolation.

### Running Unit Tests

```bash
# Run all tests
make test

# Run with verbose output
go test -v ./...

# Run specific package
go test -v ./internal/views

# Run specific test
go test -v -run TestJobsView_Cancel ./internal/views

# Run with race detector
go test -race ./...
```

### Race Detector

The race detector helps identify concurrent access issues:

```bash
# Enable race detection
go test -race ./...

# Recommended: run in CI to catch data races
```

## Integration Tests

Integration tests verify component interactions and full workflows.

### Running Integration Tests

```bash
# Run integration tests
make test-integration

# Or manually
go test -tags=integration ./test/integration

# With specific SLURM cluster
SLURM_URL=https://test.example.com \
SLURM_TOKEN=token123 \
go test -tags=integration ./test/integration
```

## Benchmarks

Performance benchmarks measure and track code performance over time.

### Running Benchmarks

```bash
# Run all benchmarks
make bench

# Or manually
go test -bench=. ./test/performance

# Run specific benchmark
go test -bench=BenchmarkJobRefresh ./test/performance

# With memory profiling
go test -bench=. -benchmem ./test/performance
```

## Test Coverage

Test coverage measures what percentage of code is exercised by tests.

### Generate Coverage Report

```bash
# Generate coverage report
make coverage

# Or manually
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
open coverage.html

# Check coverage by package
go test -cover ./...
```

### Coverage Goals

- Target: 80% code coverage minimum
- Focus on critical paths first
- Avoid covering trivial code (getters, simple setters)
- Prioritize error handling paths

## Writing Tests

### Unit Test Example

```go
func TestJobsView_CancelJob(t *testing.T) {
    // Setup
    mockClient := slurm.NewMockClient()
    mockClient.AddJob(&dao.Job{
        ID:    "123",
        State: dao.JobStateRunning,
    })

    view := NewJobsView(mockClient)

    // Test
    err := view.cancelJob("123")

    // Assert
    assert.NoError(t, err)

    job, _ := mockClient.Jobs().Get("123")
    assert.Equal(t, dao.JobStateCanceled, job.State)
}
```

### Table-Driven Tests

Table-driven tests are efficient for testing multiple scenarios:

```go
func TestParseState(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name:     "colored state",
            input:    "[green]RUNNING[white]",
            expected: "RUNNING",
        },
        {
            name:     "plain state",
            input:    "PENDING",
            expected: "PENDING",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := parseState(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Test Organization

```
internal/views/jobs_test.go      # Unit tests for views
pkg/slurm/mock_test.go          # Mock implementation tests
test/integration/               # Integration tests
test/performance/              # Performance benchmarks
```

### Best Practices

1. **Use descriptive test names**
   - Format: `Test<Function>_<Scenario>`
   - Example: `TestJobsView_CancelJob`, `TestFilter_WithEmptyString`

2. **Follow Arrange-Act-Assert pattern**
   ```go
   // Arrange - set up test data
   mockClient := slurm.NewMockClient()

   // Act - perform the action
   err := view.cancelJob("123")

   // Assert - verify the result
   assert.NoError(t, err)
   ```

3. **Mock external dependencies**
   - Use mock SLURM client for testing
   - Mock HTTP requests
   - Mock file I/O

4. **Test error conditions**
   ```go
   func TestJobsView_CancelJob_NotFound(t *testing.T) {
       mockClient := slurm.NewMockClient()
       view := NewJobsView(mockClient)

       err := view.cancelJob("nonexistent")

       assert.Error(t, err)
       assert.Contains(t, err.Error(), "not found")
   }
   ```

5. **Avoid test interdependencies**
   - Each test should be independent
   - Tests should not rely on execution order
   - Use t.Parallel() for parallel test execution

## Running Tests

### Common Test Commands

```bash
# Run all tests with summary
go test ./...

# Run with detailed output
go test -v ./...

# Run and show coverage for each function
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out

# Run with short timeout (useful for quick feedback)
go test -short ./...

# Run tests matching pattern
go test -run "TestJobs" ./...

# Stop after first failure
go test -failfast ./...
```

### Test Filters

```bash
# Run tests in specific package
go test ./internal/views

# Run tests matching a pattern
go test -run "TestJobsView" ./...

# Run specific test function
go test -run "^TestJobsView_CancelJob$" ./...

# Exclude tests with short flag
go test -short ./...
```

## Performance Profiling

### CPU Profiling

```bash
# Run with CPU profiling
go run cmd/s9s/main.go -cpuprofile=cpu.prof --mock
go tool pprof cpu.prof

# Or from tests
go test -cpuprofile=cpu.prof -bench=. ./test/performance
go tool pprof -http=:8080 cpu.prof
```

### Memory Profiling

```bash
# Run with memory profiling
go run cmd/s9s/main.go -memprofile=mem.prof --mock
go tool pprof mem.prof

# Or from tests
go test -memprofile=mem.prof -bench=. ./test/performance
go tool pprof -http=:8080 mem.prof
```

### Trace Analysis

```bash
# Generate trace
go test -trace=trace.out ./test/performance

# Analyze trace (opens browser)
go tool trace trace.out
```

### Common Profiling Commands

```bash
# View pprof in browser
go tool pprof -http=:8080 cpu.prof

# Get top functions by CPU time
go tool pprof cpu.prof
(pprof) top

# View callgraph
(pprof) png > callgraph.png

# List specific function
(pprof) list functionName
```

## Optimization Techniques

### Reduce Allocations

```go
// Bad - creates new string on each iteration
for _, job := range jobs {
    formatted := fmt.Sprintf("Job: %s", job.Name)
}

// Good - reuse builder
var builder strings.Builder
for _, job := range jobs {
    builder.WriteString("Job: ")
    builder.WriteString(job.Name)
}
```

### Cache Expensive Operations

```go
type JobsView struct {
    cache      map[string]*dao.Job
    cacheTime  time.Time
    cacheTTL   time.Duration
}
```

### Use Goroutines Wisely

```go
// Parallel fetching with limited concurrency
var wg sync.WaitGroup
results := make(chan *dao.Job, len(jobIDs))

for _, id := range jobIDs {
    wg.Add(1)
    go func(jobID string) {
        defer wg.Done()
        job, _ := client.Jobs().Get(jobID)
        results <- job
    }(id)
}
```

## Debug Logging

### Enable Debug Mode

```bash
# Enable debug logging
s9s --debug

# Or with environment variable
export S9S_DEBUG=true
s9s

# Check debug log location
tail -f ~/.s9s/debug.log
```

### Add Debug Statements

```go
import "github.com/jontk/s9s/internal/debug"

func (v *JobsView) Refresh() error {
    debug.Logger.Printf("Refresh started at %s", time.Now())

    jobs, err := v.client.Jobs().List(nil)
    if err != nil {
        debug.Logger.Printf("Failed to list jobs: %v", err)
        return err
    }

    debug.Logger.Printf("Retrieved %d jobs", len(jobs.Jobs))
    return nil
}
```

## Using Delve Debugger

### Basic Debugging

```bash
# Debug the application
dlv debug cmd/s9s/main.go -- --mock

# Set breakpoint
(dlv) break internal/views/jobs.go:123
(dlv) continue

# Examine variables
(dlv) print job
(dlv) locals

# Step through code
(dlv) next
(dlv) step
(dlv) stepout
```

### Delve Commands

```
break (b)        - Set breakpoint
continue (c)     - Continue execution
next (n)         - Step over
step (s)         - Step into
stepout (so)     - Step out of function
print (p)        - Print variable
locals           - Show local variables
args             - Show function arguments
stack            - Show call stack
quit (q)         - Quit debugger
```

## Troubleshooting Tests

### TUI not rendering correctly

```bash
# Check terminal capabilities
echo $TERM

# Try with basic terminal
TERM=xterm-256color s9s --mock
```

### Colors not working

```bash
# Force color mode
export COLORTERM=truecolor
s9s --mock
```

### Keyboard input issues

```bash
# Debug key events
s9s --debug
# Check debug.log for key event logs
```

## CI/CD Testing

All tests must pass in CI before merging:

```bash
# Run tests as CI would
make test
make test-integration
make lint
make build
```

See [CI/CD Setup](ci-cd.md) for detailed CI/CD configuration.

## Best Practices Summary

1. Write tests as you code (test-driven development)
2. Aim for 80%+ code coverage
3. Test error conditions and edge cases
4. Use table-driven tests for multiple scenarios
5. Mock external dependencies
6. Keep tests independent and idempotent
7. Use descriptive test names
8. Follow Arrange-Act-Assert pattern
9. Run tests frequently during development
10. Profile before optimizing

## Related Documentation

- [Setup Guide](setup.md) - Development environment setup
- [Contributing Guide](contributing.md) - Contribution process
- [Linting Standards](linting.md) - Code quality requirements
- [CI/CD Setup](ci-cd.md) - Continuous integration configuration
