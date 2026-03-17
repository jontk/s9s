# s9s Development Setup Guide

This guide covers everything you need to set up your development environment for s9s.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Development Tools](#development-tools)
- [IDE Setup](#ide-setup)
- [Development Workflow](#development-workflow)

## Prerequisites

### Go Installation

1. **Go 1.24 or higher**
   ```bash
   # Check version
   go version

   # Install/upgrade Go
   # macOS
   brew install go

   # Linux
   wget https://go.dev/dl/go1.24.4.linux-amd64.tar.gz
   sudo tar -C /usr/local -xzf go1.24.4.linux-amd64.tar.gz
   ```

### Development Tools

Install required tools:

```bash
# Install required tools
make install-tools

# Or manually:
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install mvdan.cc/gofumpt@latest
go install github.com/goreleaser/goreleaser@latest
```

### Optional Tools

```bash
# Delve debugger
go install github.com/go-delve/delve/cmd/dlv@latest

# Go tools
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/kisielk/errcheck@latest
go install honnef.co/go/tools/cmd/staticcheck@latest
```

## IDE Setup

### VS Code

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
         "args": ["--mock", "--debug"]
       }
     ]
   }
   ```

### GoLand/IntelliJ

1. Open project as Go module
2. Configure Run Configuration:
   - Program arguments: `--mock --debug`
   - Program arguments include `--debug`
3. Enable Go modules support

## Development Workflow

### Fork and Clone

```bash
git clone https://github.com/YOUR_USERNAME/s9s.git
cd s9s
git remote add upstream https://github.com/jontk/s9s.git
```

### Create Feature Branch

```bash
git checkout -b feature/my-feature
```

### Make Changes

```bash
# Edit files
vim internal/views/jobs.go

# Format code
make fmt

# Run linter
make lint
```

For detailed linting standards and configuration, see [Linting Standards](linting.md).

### Test Changes

```bash
# Run tests
make test

# Run specific test
go test -v -run TestJobsView ./internal/views

# Run with coverage
make coverage
```

See [Testing Guide](testing.md) for comprehensive testing documentation.

### Commit and Push

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

# Build for all platforms (Linux, macOS, Windows)
make ci-build
```

### Manual Build

```bash
# Basic build
go build -o s9s cmd/s9s/main.go

# Build with version info
go build -ldflags "-X github.com/jontk/s9s/internal/version.Version=1.0.0 -X github.com/jontk/s9s/internal/version.Commit=$(git rev-parse HEAD)" \
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

## Mock Mode Development

### Running Mock Mode

```bash
# Mock mode (requires S9S_ENABLE_MOCK=1)
export S9S_ENABLE_MOCK=1
s9s --mock
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

### Mock Data

Mock data is generated in `pkg/slurm/mock.go`. The mock client simulates a SLURM cluster with realistic job and node data for development and testing.

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

## Getting Help

- Check existing issues: [GitHub Issues](https://github.com/jontk/s9s/issues)
- Join Discord: [s9s Discord](https://discord.gg/s9s)
- Read the docs: [Documentation](https://s9s.dev/docs)

## Next Steps

- See [Architecture Guide](architecture.md) for system design
- See [Testing Guide](testing.md) for test strategies
- See [Contributing Guide](contributing.md) for contribution process
- See [Linting Standards](linting.md) for code quality requirements
