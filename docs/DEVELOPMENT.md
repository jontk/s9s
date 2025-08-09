# s9s Development Guide

This guide covers everything you need to know to develop s9s, from setting up your environment to debugging complex issues.

## Table of Contents

- [Development Environment](#development-environment)
- [Project Structure](#project-structure)
- [Building from Source](#building-from-source)
- [Running Tests](#running-tests)
- [Debugging](#debugging)
- [Mock Mode Development](#mock-mode-development)
- [Adding New Features](#adding-new-features)
- [Performance Optimization](#performance-optimization)
- [Release Process](#release-process)

## Development Environment

### Prerequisites

1. **Go 1.19+**
   ```bash
   # Check version
   go version
   
   # Install/upgrade Go
   # macOS
   brew install go
   
   # Linux
   wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
   sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
   ```

2. **Development Tools**
   ```bash
   # Install required tools
   make install-tools
   
   # Or manually:
   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
   go install mvdan.cc/gofumpt@latest
   go install github.com/goreleaser/goreleaser@latest
   ```

3. **Optional Tools**
   ```bash
   # Delve debugger
   go install github.com/go-delve/delve/cmd/dlv@latest
   
   # Go tools
   go install golang.org/x/tools/cmd/goimports@latest
   go install github.com/kisielk/errcheck@latest
   go install honnef.co/go/tools/cmd/staticcheck@latest
   ```

### IDE Setup

#### VS Code

1. Install Go extension
2. Configure settings:
   ```json
   {
     "go.lintTool": "golangci-lint",
     "go.lintOnSave": "package",
     "go.formatTool": "gofumpt",
     "go.useLanguageServer": true,
     "gopls": {
       "experimentalWorkspaceModule": true
     }
   }
   ```

3. Launch configuration (`.vscode/launch.json`):
   ```json
   {
     "version": "0.2.0",
     "configurations": [
       {
         "name": "Debug s9s",
         "type": "go",
         "request": "launch",
         "mode": "debug",
         "program": "${workspaceFolder}/cmd/s9s",
         "args": ["--mock", "--debug"],
         "env": {
           "S9S_DEBUG": "true"
         }
       }
     ]
   }
   ```

#### GoLand/IntelliJ

1. Open project as Go module
2. Configure Run Configuration:
   - Program arguments: `--mock --debug`
   - Environment: `S9S_DEBUG=true`
3. Enable Go modules support

### Development Workflow

1. **Fork and Clone**
   ```bash
   git clone https://github.com/YOUR_USERNAME/s9s.git
   cd s9s
   git remote add upstream https://github.com/jontk/s9s.git
   ```

2. **Create Branch**
   ```bash
   git checkout -b feature/my-feature
   ```

3. **Make Changes**
   ```bash
   # Edit files
   vim internal/views/jobs.go
   
   # Format code
   make fmt
   
   # Run linter
   make lint
   ```

4. **Test Changes**
   ```bash
   # Run tests
   make test
   
   # Run specific test
   go test -v -run TestJobsView ./internal/views
   
   # Run with coverage
   make coverage
   ```

5. **Commit and Push**
   ```bash
   git add .
   git commit -m "feat: add awesome feature"
   git push origin feature/my-feature
   ```

## Project Structure

```
s9s/
├── cmd/s9s/              # Main application entry
│   └── main.go
├── internal/             # Private application code
│   ├── app/             # Application lifecycle
│   ├── config/          # Configuration management
│   ├── dao/             # Data access objects
│   ├── debug/           # Debug utilities
│   ├── export/          # Export functionality
│   ├── preferences/     # User preferences
│   ├── ssh/             # SSH integration
│   ├── ui/              # UI utilities
│   │   ├── components/  # Reusable UI components
│   │   └── filters/     # Filtering system
│   └── views/           # View implementations
├── pkg/                 # Public packages
│   └── slurm/          # Mock SLURM implementation
├── test/               # Test suites
│   ├── integration/    # Integration tests
│   └── performance/    # Performance benchmarks
├── docs/               # Documentation
├── scripts/            # Build and utility scripts
└── Makefile           # Build automation
```

### Key Directories

- **`internal/views/`**: All UI views (jobs, nodes, etc.)
- **`internal/dao/`**: SLURM client abstraction
- **`internal/ui/components/`**: Reusable UI components
- **`pkg/slurm/`**: Mock SLURM for development/testing

## Building from Source

### Quick Build

```bash
# Build binary
make build

# Install to $GOPATH/bin
make install

# Build for all platforms
make build-all
```

### Manual Build

```bash
# Basic build
go build -o s9s cmd/s9s/main.go

# Build with version info
go build -ldflags "-X main.version=1.0.0 -X main.commit=$(git rev-parse HEAD)" \
  -o s9s cmd/s9s/main.go

# Optimized build
go build -ldflags "-s -w" -o s9s cmd/s9s/main.go
```

### Cross-Compilation

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o s9s-linux-amd64 cmd/s9s/main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o s9s-darwin-amd64 cmd/s9s/main.go
GOOS=darwin GOARCH=arm64 go build -o s9s-darwin-arm64 cmd/s9s/main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o s9s-windows-amd64.exe cmd/s9s/main.go
```

## Running Tests

### Unit Tests

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

### Integration Tests

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

### Benchmarks

```bash
# Run benchmarks
make bench

# Or manually
go test -bench=. ./test/performance

# Run specific benchmark
go test -bench=BenchmarkJobRefresh ./test/performance

# With memory profiling
go test -bench=. -benchmem ./test/performance
```

### Test Coverage

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

### Writing Tests

Example unit test:

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
    assert.Equal(t, dao.JobStateCancelled, job.State)
}
```

Table-driven test:

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

## Debugging

### Debug Mode

```bash
# Enable debug logging
s9s --debug

# Or with environment variable
export S9S_DEBUG=true
s9s

# Debug log location
tail -f ~/.s9s/debug.log
```

### Debug Logging

Add debug statements:

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

### Using Delve

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

### Performance Profiling

```bash
# CPU profiling
go run cmd/s9s/main.go -cpuprofile=cpu.prof --mock
go tool pprof cpu.prof

# Memory profiling
go run cmd/s9s/main.go -memprofile=mem.prof --mock
go tool pprof mem.prof

# View in browser
go tool pprof -http=:8080 cpu.prof
```

### Common Issues

1. **TUI not rendering correctly**
   ```bash
   # Check terminal capabilities
   echo $TERM
   
   # Try with basic terminal
   TERM=xterm-256color s9s
   ```

2. **Colors not working**
   ```bash
   # Force color mode
   export COLORTERM=truecolor
   s9s
   ```

3. **Keyboard input issues**
   ```bash
   # Debug key events
   s9s --debug
   # Check debug.log for key event logs
   ```

## Mock Mode Development

### Running Mock Mode

```bash
# Basic mock mode
s9s --mock

# Customize mock data
s9s --mock --mock-jobs 500 --mock-nodes 200

# With specific scenarios
S9S_MOCK_SCENARIO=high-load s9s --mock
```

### Extending Mock Data

Add mock data in `pkg/slurm/mock.go`:

```go
func (m *MockClient) generateMockJobs(count int) {
    states := []string{
        dao.JobStateRunning,
        dao.JobStatePending,
        dao.JobStateCompleted,
    }
    
    for i := 0; i < count; i++ {
        job := &dao.Job{
            ID:    fmt.Sprintf("%d", 1000+i),
            Name:  fmt.Sprintf("job_%d", i),
            State: states[i%len(states)],
            // ... other fields
        }
        m.jobs[job.ID] = job
    }
}
```

### Mock Scenarios

Create scenario in `pkg/slurm/mock_scenarios.go`:

```go
func (m *MockClient) LoadScenario(name string) {
    switch name {
    case "high-load":
        m.generateMockJobs(10000)
        m.generateMockNodes(500)
        
    case "failing-jobs":
        // Generate jobs that fail
        for i := 0; i < 100; i++ {
            m.AddJob(&dao.Job{
                ID:    fmt.Sprintf("fail_%d", i),
                State: dao.JobStateFailed,
            })
        }
    }
}
```

## Adding New Features

### Adding a New View

1. **Create view file**: `internal/views/myview.go`
   ```go
   package views
   
   type MyView struct {
       *BaseView
       client dao.SlurmClient
       // ... other fields
   }
   
   func NewMyView(client dao.SlurmClient) *MyView {
       return &MyView{
           BaseView: NewBaseView("myview", "My View"),
           client:   client,
       }
   }
   
   func (v *MyView) SetupView() tview.Primitive {
       // Create UI
   }
   
   func (v *MyView) Refresh() error {
       // Refresh data
   }
   ```

2. **Register view**: In `internal/app/app.go`
   ```go
   app.RegisterView(views.NewMyView(client))
   ```

3. **Add key binding**: In `internal/app/keybindings.go`
   ```go
   case 'v':
       app.SwitchToView("myview")
   ```

### Adding a New Command

1. **Define command handler**:
   ```go
   func (v *JobsView) myCommand() {
       // Implementation
   }
   ```

2. **Add key binding**:
   ```go
   case 'x':
       v.myCommand()
       return nil
   ```

3. **Update help text**:
   ```go
   help.AddItem("x", "Execute my command")
   ```

### Adding a New Component

1. **Create component**: `internal/ui/components/mycomponent.go`
   ```go
   package components
   
   type MyComponent struct {
       *tview.Box
       // ... fields
   }
   
   func NewMyComponent() *MyComponent {
       return &MyComponent{
           Box: tview.NewBox(),
       }
   }
   
   func (c *MyComponent) Draw(screen tcell.Screen) {
       // Drawing logic
   }
   ```

2. **Use in views**:
   ```go
   comp := components.NewMyComponent()
   v.layout.AddItem(comp, 0, 1, false)
   ```

## Performance Optimization

### Profiling

```bash
# CPU profile
go test -cpuprofile=cpu.prof -bench=. ./test/performance
go tool pprof -http=:8080 cpu.prof

# Memory profile
go test -memprofile=mem.prof -bench=. ./test/performance
go tool pprof -http=:8080 mem.prof

# Trace
go test -trace=trace.out ./test/performance
go tool trace trace.out
```

### Optimization Techniques

1. **Reduce allocations**:
   ```go
   // Bad
   for _, job := range jobs {
       formatted := fmt.Sprintf("Job: %s", job.Name)
   }
   
   // Good
   var builder strings.Builder
   for _, job := range jobs {
       builder.WriteString("Job: ")
       builder.WriteString(job.Name)
   }
   ```

2. **Cache expensive operations**:
   ```go
   type JobsView struct {
       cache      map[string]*dao.Job
       cacheTime  time.Time
       cacheTTL   time.Duration
   }
   ```

3. **Use goroutines wisely**:
   ```go
   // Parallel fetching
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

## Release Process

### Version Numbering

Follow Semantic Versioning:
- **Major**: Breaking changes
- **Minor**: New features
- **Patch**: Bug fixes

### Release Steps

1. **Update version**:
   ```bash
   # Update version in main.go
   vim cmd/s9s/main.go
   ```

2. **Update changelog**:
   ```bash
   vim CHANGELOG.md
   ```

3. **Create tag**:
   ```bash
   git tag -a v1.2.3 -m "Release v1.2.3"
   git push origin v1.2.3
   ```

4. **Build release**:
   ```bash
   make release
   ```

5. **Create GitHub release**:
   ```bash
   gh release create v1.2.3 \
     --title "Release v1.2.3" \
     --notes-file CHANGELOG.md \
     dist/*.tar.gz
   ```

### Automated Release

Using GoReleaser:

```bash
# Test release
goreleaser release --snapshot --clean

# Create release
export GITHUB_TOKEN=your_token
goreleaser release
```

## Troubleshooting

### Common Development Issues

1. **Module issues**:
   ```bash
   go mod tidy
   go mod download
   go clean -modcache
   ```

2. **Build issues**:
   ```bash
   go clean -cache
   go build -v ./...
   ```

3. **Test failures**:
   ```bash
   # Run with verbose output
   go test -v -run TestName ./package
   
   # Check for race conditions
   go test -race ./...
   ```

### Getting Help

- Check existing issues: [GitHub Issues](https://github.com/jontk/s9s/issues)
- Join Discord: [s9s Discord](https://discord.gg/s9s)
- Read the docs: [Documentation](https://s9s.dev/docs)

---

Happy coding! 🚀