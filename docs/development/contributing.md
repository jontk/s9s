# Contributing to s9s

Thank you for your interest in contributing to s9s! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Process](#development-process)
- [Code Style](#code-style)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Reporting Issues](#reporting-issues)
- [Security](#security)
- [Resources](#resources)

## Code of Conduct

By participating in this project, you agree to abide by our Code of Conduct:

- Be respectful and inclusive
- Welcome newcomers and help them get started
- Focus on constructive criticism
- Accept feedback gracefully
- Prioritize the community's best interests

## Getting Started

### Prerequisites

- Go 1.19 or higher
- Git
- Make (optional but recommended)
- golangci-lint (for linting)

### Setting Up Your Development Environment

1. **Fork the repository**
   ```bash
   # Click the "Fork" button on GitHub, then:
   git clone https://github.com/YOUR_USERNAME/s9s.git
   cd s9s
   ```

2. **Add upstream remote**
   ```bash
   git remote add upstream https://github.com/jontk/s9s.git
   ```

3. **Install dependencies**
   ```bash
   go mod download
   ```

4. **Install development tools**
   ```bash
   # Install golangci-lint
   curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

   # Install other tools
   go install github.com/goreleaser/goreleaser@latest
   go install mvdan.cc/gofumpt@latest
   ```

5. **Run the application in mock mode**
   ```bash
   go run cmd/s9s/main.go --mock
   ```

## Development Process

### Workflow

1. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   # or
   git checkout -b fix/issue-description
   ```

2. **Make your changes**
   - Write code following our style guide
   - Add tests for new functionality
   - Update documentation as needed

3. **Test your changes**
   ```bash
   # Run all tests
   go test ./...

   # Run specific package tests
   go test ./internal/views/...

   # Run with coverage
   go test -coverprofile=coverage.out ./...
   go tool cover -html=coverage.out
   ```

4. **Lint your code**
   ```bash
   golangci-lint run

   # Fix formatting
   gofumpt -w .
   ```

5. **Commit your changes**
   ```bash
   git add .
   git commit -m "feat: add amazing feature"
   ```

### Commit Message Guidelines

We follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `style:` Code style changes (formatting, etc.)
- `refactor:` Code refactoring
- `perf:` Performance improvements
- `test:` Test additions or changes
- `chore:` Build process or auxiliary tool changes

Examples:
```
feat: add batch job cancellation support
fix: resolve node state parsing for compound states
docs: update README with new configuration options
refactor: simplify job filtering logic
```

## Code Style

### Go Code Style

1. **Follow standard Go conventions**
   - Use `gofmt` or `gofumpt` for formatting
   - Follow [Effective Go](https://golang.org/doc/effective_go.html)
   - Use meaningful variable and function names

2. **Package Organization**
   ```go
   package views

   import (
       "standard library packages"

       "third party packages"

       "internal packages"
   )
   ```

3. **Error Handling**
   ```go
   // Good
   if err != nil {
       return fmt.Errorf("failed to process job %s: %w", jobID, err)
   }

   // Avoid
   if err != nil {
       return err
   }
   ```

4. **Comments and Documentation**
   ```go
   // JobsView displays the jobs list and provides job management functionality.
   // It implements the View interface and supports real-time updates.
   type JobsView struct {
       // ...
   }

   // Refresh updates the jobs list with latest data from the SLURM cluster.
   // It returns an error if the refresh operation fails.
   func (v *JobsView) Refresh() error {
       // Implementation
   }
   ```

5. **Interface Design**
   ```go
   // Interfaces should be small and focused
   type JobManager interface {
       List(opts *ListJobsOptions) (*JobList, error)
       Get(id string) (*Job, error)
       Cancel(id string) error
   }
   ```

### UI/View Guidelines

1. **Consistent Key Bindings**
   - Follow vim-like conventions where appropriate
   - Document all key bindings in help text
   - Avoid conflicts with global bindings

2. **Status Messages**
   - Use status bar for user feedback
   - Provide clear error messages
   - Show progress for long operations

3. **Color Usage**
   - Follow the established color scheme
   - Ensure readability on dark and light terminals
   - Use colors consistently across views

## Linting and Code Quality

We use **golangci-lint** with 15 enabled linters to maintain code quality and consistency throughout the project.

### Quick Start

```bash
# Run linter
make lint

# Fix formatting issues
make fmt

# Install pre-commit hooks (recommended)
pre-commit install
```

### Linting Gate (Automated Enforcement)

All pull requests are automatically checked by our CI/CD pipeline. Linting failures block merges to ensure code quality.

- Lint job runs on every push and PR
- Build depends on lint passing
- Branch protection requires lint check
- Pre-commit hooks catch issues before push

See [CI/CD Setup](ci-cd.md) for complete CI/CD documentation.

### Enabled Linters

The linters are organized by category:

- **Core**: errcheck, govet, ineffassign, staticcheck
  - Catch critical errors that shouldn't be ignored
  - Detect suspicious constructs and bad patterns

- **Quality**: misspell, bodyclose, errorlint, wastedassign
  - Improve code quality and catch common mistakes
  - Ensure HTTP resources are properly closed

- **Security**: gosec (configured with exclusions)
  - Security-focused static analysis
  - Detects vulnerabilities and unsafe patterns

- **Style & Patterns**: gocritic, unused, nolintlint, revive
  - Code style and pattern checks
  - Identifies dead code and unused suppressions
  - Enforces Go idioms

- **Advanced**: gocognit, dupl
  - Cognitive complexity checking
  - Code duplication detection

See [Linting Standards](linting.md) for complete linting standards and configuration details.

### Pre-commit Hooks

Install pre-commit hooks to automatically check code before committing:

```bash
# Install hooks (one-time setup)
pre-commit install

# Run manually on all files
pre-commit run --all-files

# Hooks will automatically run before git commit
git commit -m "feat: add amazing feature"
```

The hooks automatically:
- Remove trailing whitespace
- Ensure files end with newline
- Format code with gofumpt
- Organize imports with goimports
- Tidy go.mod and go.sum
- Run golangci-lint to check for violations

See [PRE_COMMIT_SETUP.md](../../docs/PRE_COMMIT_SETUP.md) for detailed pre-commit hook setup.

### CI Requirements

All pull requests must:
- Pass `golangci-lint run` with no new warnings
- Have code formatted with `gofumpt`
- Have imports organized with `goimports`
- Have go.mod tidied with `go mod tidy`

These checks are enforced in GitHub Actions and must pass before merge.

### Fixing Lint Issues

1. **Auto-fixable issues**: Use `make fmt` to fix formatting
2. **Manual fixes**: Review linter messages and fix code
3. **Suppressions**: Only use `//nolint` when absolutely necessary, with justification

Example of proper suppression:
```go
// nolint:gochecknoglobals // needed for initialization
var globalState int
```

## Testing

### Test Organization

```
internal/views/jobs_test.go      # Unit tests for views
pkg/slurm/mock_test.go          # Mock implementation tests
test/integration/               # Integration tests
test/performance/              # Performance benchmarks
```

### Writing Tests

1. **Unit Tests**
   ```go
   func TestJobsView_CancelJob(t *testing.T) {
       // Arrange
       mockClient := slurm.NewMockClient()
       view := NewJobsView(mockClient)

       // Act
       err := view.cancelJob("job123")

       // Assert
       assert.NoError(t, err)
       assert.Equal(t, JobStateCanceled, mockClient.GetJob("job123").State)
   }
   ```

2. **Table-Driven Tests**
   ```go
   func TestParseJobState(t *testing.T) {
       tests := []struct {
           name     string
           input    string
           expected string
       }{
           {"running state", "[green]RUNNING[white]", "RUNNING"},
           {"pending state", "[yellow]PENDING[white]", "PENDING"},
       }

       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               result := parseJobState(tt.input)
               assert.Equal(t, tt.expected, result)
           })
       }
   }
   ```

3. **Mock Usage**
   - Use the built-in mock SLURM client for testing
   - Mock external dependencies
   - Test error conditions

### Running Tests

```bash
# Run all tests
make test

# Run with race detector
go test -race ./...

# Run specific test
go test -run TestJobsView_CancelJob ./internal/views

# Benchmark
go test -bench=. ./test/performance
```

See [Testing Guide](testing.md) for comprehensive testing documentation.

## Submitting Changes

### Pull Request Process

1. **Update your fork**
   ```bash
   git fetch upstream
   git checkout main
   git merge upstream/main
   ```

2. **Rebase your feature branch**
   ```bash
   git checkout feature/your-feature
   git rebase main
   ```

3. **Push to your fork**
   ```bash
   git push origin feature/your-feature
   ```

4. **Create Pull Request**
   - Use a clear, descriptive title
   - Reference any related issues
   - Provide a detailed description
   - Include screenshots for UI changes

### PR Template

```markdown
## Description
Brief description of the changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Comments added for complex code
- [ ] Documentation updated
- [ ] No new warnings
```

### Review Process

1. All PRs require at least one review
2. Address reviewer feedback promptly
3. Keep PRs focused and reasonably sized
4. Ensure CI checks pass

## Reporting Issues

### Bug Reports

Please include:
- s9s version (`s9s --version`)
- SLURM version
- Operating system
- Terminal emulator
- Steps to reproduce
- Expected vs actual behavior
- Error messages/logs

### Feature Requests

Please include:
- Use case description
- Proposed solution
- Alternative solutions considered
- Additional context

## Security

### Reporting Security Issues

**DO NOT** report security vulnerabilities in public issues.

Instead:
1. Email security@s9s.dev
2. Include detailed description
3. Wait for acknowledgment
4. Follow responsible disclosure

### Security Best Practices

- Never commit credentials
- Use environment variables for sensitive data
- Validate all user input
- Follow principle of least privilege

## Areas for Contribution

### Good First Issues

Look for issues labeled:
- `good first issue`
- `help wanted`
- `documentation`

### Priority Areas

1. **Test Coverage**
   - Increase unit test coverage
   - Add integration tests
   - Improve mock implementations

2. **Documentation**
   - Improve inline documentation
   - Add examples
   - Create tutorials

3. **Performance**
   - Optimize refresh operations
   - Reduce memory usage
   - Improve startup time

4. **Features**
   - Plugin system
   - Additional export formats
   - Enhanced filtering

## Resources

- [Go Documentation](https://golang.org/doc/)
- [tview Documentation](https://github.com/rivo/tview/wiki)
- [SLURM Documentation](https://slurm.schedmd.com/documentation.html)
- [s9s Architecture](architecture.md)
- [CI/CD Setup & Linting Gate](ci-cd.md)
- [Linting Standards](linting.md)

## Getting Help

- Open an issue for bugs or features
- Join our [Discord](https://discord.gg/s9s) for discussions
- Check existing issues and PRs
- Read the documentation

---

Thank you for contributing to s9s! Your efforts help make HPC cluster management better for everyone.
