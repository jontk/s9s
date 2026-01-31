# s9s Development Documentation

Welcome to the s9s development documentation. This directory contains comprehensive guides for developers working on the s9s project.

## Quick Navigation

### Getting Started
- **[Development Setup](setup.md)** - Set up your development environment, prerequisites, IDE configuration, and build instructions
- **[Quick Start](setup.md#development-workflow)** - Fork, clone, and make your first change

### Core Development Guides
- **[Architecture Guide](architecture.md)** - Understand the system design, core components, and design patterns
- **[Testing Guide](testing.md)** - Learn about unit tests, integration tests, benchmarks, and test coverage
- **[Contributing Guide](contributing.md)** - Follow the contribution process and understand code standards
- **[Linting Standards](linting.md)** - Code quality requirements and linting configuration
- **[CI/CD Setup](ci-cd.md)** - Continuous integration pipeline, linting gate, and branch protection rules

## Documentation by Role

### New Contributors
1. Read [Development Setup](setup.md) to set up your environment
2. Read [Contributing Guide](contributing.md) for the contribution process
3. Pick an issue labeled "good first issue"
4. Follow the [Development Process](contributing.md#development-process) workflow
5. Submit a pull request

### Feature Developers
1. Review [Architecture Guide](architecture.md) to understand the system
2. Read [Testing Guide](testing.md) to write comprehensive tests
3. Check [Linting Standards](linting.md) for code quality
4. Follow the [Code Style](contributing.md#code-style) guidelines
5. Ensure [CI/CD Setup](ci-cd.md) checks pass before merge

### Code Reviewers
1. Understand the [Architecture](architecture.md)
2. Verify [Testing](testing.md) is comprehensive
3. Check [Code Style](contributing.md#code-style) compliance
4. Ensure [CI/CD](ci-cd.md) checks are green
5. Use [Linting Standards](linting.md) for quality assessment

### Maintainers
1. Monitor [CI/CD](ci-cd.md) performance and health
2. Review and approve PRs following [Contributing Guide](contributing.md)
3. Manage [Linting Standards](linting.md) and linter updates
4. Monitor [Architecture](architecture.md) decisions and evolution

## Common Tasks

### Set Up Development Environment
```bash
# Follow the setup guide
# File: docs/development/setup.md
make install-tools
pre-commit install
```

### Make a Code Change
```bash
git checkout -b feature/my-feature
# Make changes following code style in contributing.md
make fmt
make lint
make test
git push origin feature/my-feature
```

### Run Tests
```bash
# See testing.md for comprehensive testing guide
make test              # Unit tests
make test-integration  # Integration tests
make bench            # Benchmarks
make coverage         # Coverage report
```

### Check Code Quality
```bash
# See linting.md for complete linting standards
make lint
make fmt
golangci-lint run     # Detailed linting
```

### Debug Issues
```bash
# See architecture.md#debugging-and-diagnostics
s9s --debug           # Enable debug logging
tail -f ~/.s9s/debug.log
```

## Document Overview

### Setup Guide
**File**: `docs/development/setup.md`

Covers:
- Go installation and version management
- Development tools installation
- IDE setup (VS Code, GoLand/IntelliJ)
- Project structure overview
- Building from source
- Cross-compilation
- Mock mode development
- Troubleshooting common issues

### Testing Guide
**File**: `docs/development/testing.md`

Covers:
- Unit tests, integration tests, benchmarks
- Test coverage measurement and goals
- Writing unit tests and table-driven tests
- Running tests with various options
- Performance profiling (CPU, memory, trace)
- Optimization techniques
- Debug logging and Delve debugger
- Test organization and best practices

### Architecture Guide
**File**: `docs/development/architecture.md`

Covers:
- System architecture and layered design
- Core components (views, DAO, SLURM adapter, mock)
- Design patterns (interface segregation, dependency injection, etc.)
- Data flow and configuration management
- State management and concurrency model
- Error handling and security considerations
- Performance optimization strategies
- Testing architecture and extension points

### Contributing Guide
**File**: `docs/development/contributing.md`

Covers:
- Code of conduct and community values
- Getting started with fork and setup
- Development workflow and branching
- Commit message conventions (Conventional Commits)
- Code style standards (Go idioms, error handling, comments)
- UI/view guidelines
- Linting and code quality enforcement
- Testing requirements and test organization
- Pull request process and review requirements
- Issue reporting (bugs and features)
- Security reporting guidelines

### Linting Standards
**File**: `docs/development/linting.md`

Covers:
- Linting philosophy and goals
- 15 enabled linters with explanations
- Linter configuration in `.golangci.yml`
- Running linters (make lint, manual execution)
- Fixing lint issues by category
- Proper use of `//nolint` directives
- Pre-commit hooks and automation
- Disabled linters and their status
- CI/CD integration and requirements
- Best practices for linting

### CI/CD Setup
**File**: `docs/development/ci-cd.md`

Covers:
- GitHub Actions workflow overview
- Lint job, test job, build job, security job
- Linting gate implementation and enforcement
- Branch protection rules configuration
- Local testing before pushing
- Troubleshooting CI failures
- Common linting violations and fixes
- Best practices for different roles

## Key Files Referenced

### Configuration Files
- `.golangci.yml` - Linter configuration
- `.github/workflows/ci.yml` - CI/CD workflow definition
- `.pre-commit-config.yaml` - Pre-commit hooks configuration
- `Makefile` - Build and development automation
- `go.mod` / `go.sum` - Go module dependencies

### Test Organization
- `internal/*/` - Package code with `*_test.go` files
- `test/integration/` - Integration tests
- `test/performance/` - Performance benchmarks
- `pkg/slurm/mock.go` - Mock SLURM client for testing

## Development Standards

### Code Quality Requirements
- 80%+ test coverage for new code
- All tests must pass before merge
- Code must pass all 15 linters
- Code must be formatted with gofumpt
- Imports must be organized with goimports
- go.mod must be tidy

### Commit and PR Standards
- Follow Conventional Commits format
- PRs should be focused and reasonably sized
- Include tests for new functionality
- Update documentation when needed
- Address reviewer feedback promptly

### Security Standards
- Never commit credentials
- Validate all user input
- Follow principle of least privilege
- No credentials in logs or debug output
- Use environment variables for sensitive data

## Useful Commands

```bash
# Setup
make install-tools         # Install development tools
pre-commit install        # Install pre-commit hooks

# Development
make fmt                  # Format code
make lint                 # Run linter
make build               # Build application
go run cmd/s9s/main.go --mock  # Run in mock mode

# Testing
make test                # Run unit tests
make test-integration    # Run integration tests
make bench               # Run benchmarks
make coverage            # Generate coverage report

# Debugging
s9s --debug              # Enable debug mode
dlv debug cmd/s9s/main.go -- --mock  # Use Delve debugger
go tool pprof cpu.prof   # Analyze profiles

# CI/CD
golangci-lint run        # Full linting (as CI runs it)
pre-commit run --all-files # Run all pre-commit hooks
```

## Getting Help

If you have questions:
1. Check the relevant documentation file above
2. Search [existing GitHub issues](https://github.com/jontk/s9s/issues)
3. Review [PR discussions](https://github.com/jontk/s9s/pulls) for examples
4. Join the [s9s Discord](https://discord.gg/s9s) community
5. Reach out to maintainers

## Contributing Workflow Checklist

- [ ] Read [Development Setup](setup.md) and configure environment
- [ ] Read [Contributing Guide](contributing.md)
- [ ] Understand [Architecture](architecture.md)
- [ ] Create feature branch (`git checkout -b feature/my-feature`)
- [ ] Make changes following [Code Style](contributing.md#code-style)
- [ ] Write tests following [Testing Guide](testing.md)
- [ ] Format code (`make fmt`)
- [ ] Run linter (`make lint`)
- [ ] Run tests (`make test`)
- [ ] Commit with [Conventional Commits](contributing.md#commit-message-guidelines)
- [ ] Push and create PR
- [ ] Ensure all [CI/CD](ci-cd.md) checks pass
- [ ] Address reviewer feedback
- [ ] Merge when approved and all checks pass

## Related Resources

- [Main README](../../README.md) - Project overview
- [GitHub Issues](https://github.com/jontk/s9s/issues) - Bug reports and feature requests
- [GitHub Discussions](https://github.com/jontk/s9s/discussions) - Community discussions
- [s9s Discord](https://discord.gg/s9s) - Real-time community chat
- [Go Documentation](https://golang.org/doc/) - Go language reference
- [tview Documentation](https://github.com/rivo/tview/wiki) - Terminal UI framework
- [SLURM Documentation](https://slurm.schedmd.com/documentation.html) - SLURM reference

---

Happy coding! We look forward to your contributions to s9s.
