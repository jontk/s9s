# Linting Standards and Guidelines

This document covers s9s linting standards, configuration, and best practices. Linting is a critical part of our development workflow to ensure code quality, consistency, and reliability.

## Table of Contents

- [Linting Philosophy](#linting-philosophy)
- [Enabled Linters](#enabled-linters)
- [Linter Configuration](#linter-configuration)
- [Running Linters](#running-linters)
- [Fixing Lint Issues](#fixing-lint-issues)
- [Pre-commit Hooks](#pre-commit-hooks)
- [Disabled Linters](#disabled-linters)
- [CI/CD Integration](#cicd-integration)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Linting Philosophy

We use linting to:

- **Catch errors early**: Find bugs before runtime (e.g., unchecked errors, unused variables)
- **Enforce consistency**: Maintain consistent code style across the project
- **Improve readability**: Ensure code is easy to understand and maintain
- **Prevent issues**: Stop common mistakes and anti-patterns

We configure linters conservatively - we enable only linters that provide clear value and can be maintained. This avoids false positives and linter fatigue.

## Recent Improvements - Phase 7 (PR #29)

### Revive Linter Fixes

As of **PR #29**, we systematically fixed all revive linter violations through a comprehensive 3-phase approach:

**Summary**:
- **460 revive violations reduced to 36** (92% reduction)
- All violations fixed except 36 backward-compatible type aliases
- Full backward compatibility maintained

**What was fixed**:
1. **Phase 1 - Quick Wins** (287 violations, 62% reduction)
   - Added 37 package comments (all packages now documented)
   - Fixed 244+ unused parameters (renamed to `_`)
   - Fixed 6 var-naming violations

2. **Phase 2 - Documentation** (102 violations, 22% reduction)
   - Added godoc comments to 102 exported symbols
   - All public APIs now properly documented

3. **Phase 3 - Structural Improvements** (95 violations, 21% reduction)
   - Fixed 31 variables shadowing Go built-ins
   - Refactored 38 stuttering type names with backward-compatible aliases
   - Fixed 7 minor violations

**Impact**:
- 170+ files modified
- 1000+ lines changed
- Code quality significantly improved
- All tests passing, clean builds

## Enabled Linters

s9s uses **15 enabled linters** configured in `.golangci.yml`. They are organized by category:

### Core Linters

These catch critical errors that should never be ignored:

- **errcheck**: Ensures all error returns are checked
  - Essential for reliability
  - Helps prevent silent failures
  - Example: `rows.Close()` must have error checked

- **govet**: Detects suspicious constructs and potential bugs
  - Reports shadowed variables, incorrect printf usage, etc.
  - Built into Go's standard tooling
  - High signal-to-noise ratio

- **ineffassign**: Finds ineffective assignments
  - Variables assigned but never used
  - Dead code elimination
  - Example: `x = 5; x = 10` (first assignment wasted)

- **staticcheck**: Comprehensive static analysis
  - Unused code detection
  - Unreachable code
  - Incorrect API usage
  - Type conversion issues

### Code Quality Linters

These improve code quality and catch common issues:

- **misspell**: Detects commonly misspelled English words
  - Checks comments and strings
  - Prevents typos in documentation
  - Uses US English conventions

- **bodyclose**: Ensures HTTP response bodies are closed
  - Prevents resource leaks
  - Critical for long-running services
  - Example: `resp, _ := http.Get(url); defer resp.Body.Close()`

- **errorlint**: Ensures proper error handling patterns
  - Checks for %w verb usage in fmt.Errorf (Go 1.13+)
  - Validates error type assertions
  - Prevents error wrapping anti-patterns

- **wastedassign**: Detects wasted assignments
  - Variables assigned but used only in control flow
  - Similar to ineffassign but for different patterns
  - Helps identify unnecessary variables

### Security Linters

- **gosec** (configured): Security-focused static analysis
  - Detects security vulnerabilities
  - Checks for weak randomness, SQL injection, path traversal, etc.
  - Configured with exclusions for accepted patterns

### Style and Pattern Linters

- **gocritic**: Style and code pattern checks
  - Detects code smells and anti-patterns
  - Diagnostic, performance, and style checks
  - Helps catch subtle bugs in logic

- **revive**: Go idioms and style enforcement
  - Package and exported symbol documentation
  - Naming conventions (avoids stuttering names)
  - Error handling patterns
  - All 460+ issues fixed in PR #29
  - Only 36 acceptable violations remain

- **unused**: Finds unused variables, constants, functions, and types
  - Helps clean up dead code
  - Can reveal incomplete refactoring
  - Works across package boundaries

- **nolintlint**: Validates `//nolint` directives
  - Ensures linter suppressions are actually necessary
  - Prevents accumulated technical debt
  - Requires specific rule names and explanations

### Advanced Linters

- **gocognit**: Cognitive complexity checking
  - Measures how difficult code is to understand
  - Threshold: 30 (allows moderate complexity)
  - Helps identify hard-to-understand code

- **dupl**: Code duplication detection
  - Detects duplicated code blocks
  - Threshold: 150 lines
  - Helps identify abstraction opportunities

## Linter Configuration

The `.golangci.yml` file contains our linter configuration with three main sections:

### run

- **timeout**: 10m - Linting timeout (prevents hanging on large files)
- **tests**: true - Include test files in linting
- **skip-dirs**: Excludes vendor, .git, .github directories

### linters

- **disable-all**: true - Start with all linters disabled, explicitly enable only those we want
- **enable**: List of 15 enabled linters
- **disable**: List of linters we deliberately don't use

### linters-settings

Specific configuration for each linter:

- **errcheck**: Checks type assertions, allows `_ = expr` for intentional ignores
- **govet**: Enables shadow detection
- **gosec**: Medium severity/confidence threshold with specific rule exclusions
- **gocritic**: Enables diagnostic, performance, and style check tags
- **errorlint**: Checks errorf with %w, type assertions, and error comparisons
- **misspell**: US English locale
- **gocognit**: Cognitive complexity minimum of 30
- **nolintlint**: Validates nolint directives
- **dupl**: Minimum 150 lines to consider as duplication
- **containedctx**: Prevents context in struct fields

## Running Linters

### Using Make

```bash
# Run golangci-lint (the primary linter)
make lint

# Fix formatting issues (gofumpt and goimports)
make fmt

# Run all checks (lint + fmt)
make check
```

### Manual golangci-lint Execution

```bash
# Run linter on all Go files
golangci-lint run

# Run linter with verbose output
golangci-lint run -v

# Run linter on specific package
golangci-lint run ./internal/views

# Run linter on specific file
golangci-lint run ./internal/views/jobs.go

# Check only files changed in last commit
golangci-lint run --new-from-rev=HEAD~1

# Check only files changed against main branch
golangci-lint run --new-from-rev=origin/main

# Check new issues only (faster, useful during development)
golangci-lint run --new
```

## Fixing Lint Issues

### Understanding Linter Messages

Each linter message includes:
- **File path and line number**: Where the issue is
- **Linter name**: Which linter detected it (in parentheses)
- **Issue description**: What the problem is

Example:
```
internal/views/jobs.go:123:5: `foo` is unused (unused)
internal/views/jobs.go:456:2: Error return value of `rows.Close()` is not checked (errcheck)
```

### Common Fixes by Linter

**errcheck** - Unchecked error:
```go
// Bad
rows, _ := db.Query("SELECT * FROM jobs")

// Good
rows, err := db.Query("SELECT * FROM jobs")
if err != nil {
    return fmt.Errorf("failed to query jobs: %w", err)
}
defer rows.Close()
```

**unused** - Unused variable:
```go
// Bad
func ProcessJob(job *Job, extra string) {
    fmt.Println(job.ID)
}

// Good (remove parameter if not used)
func ProcessJob(job *Job) {
    fmt.Println(job.ID)
}

// Or if parameter is required by interface
func ProcessJob(job *Job, _ string) {
    fmt.Println(job.ID)
}
```

**bodyclose** - Response body not closed:
```go
// Bad
resp, err := http.Get(url)
if err != nil {
    return err
}
data, _ := io.ReadAll(resp.Body)

// Good
resp, err := http.Get(url)
if err != nil {
    return err
}
defer resp.Body.Close()
data, err := io.ReadAll(resp.Body)
```

**errorlint** - Missing %w in error wrapping:
```go
// Bad
return fmt.Errorf("failed: %v", err)

// Good
return fmt.Errorf("failed: %w", err)
```

### When to Use `//nolint` Directives

The `//nolint` directive suppresses specific linter warnings. Use sparingly - accumulated nolint directives are technical debt.

Valid reasons for nolint:
- **Intentional patterns**: Code that intentionally violates a linter rule
- **False positives**: Linter incorrectly flags valid code
- **Generated code**: Code created by tools can't be fixed
- **Test requirements**: Testing code that needs different patterns

Invalid reasons:
- **Laziness**: Just to avoid fixing code
- **Disagreement**: Rather than having a team discussion
- **Temporary**: "I'll fix it later"

#### Proper Nolint Usage

Always include:
1. Specific linter name(s)
2. Justification comment explaining WHY the violation is acceptable

```go
// Bad - suppress all linters, no explanation
// nolint

// Bad - suppress all linters for this line, no explanation
var globalState int // nolint

// Good - specific linter with justification
var globalState int // nolint:gochecknoglobals // state needed for initialization

// Good - security issue with validation explanation
//nolint:gosec // G304: path is application-controlled (from cacheDir config), not user input
data, err := os.ReadFile(cachePath)

// Good - command with validated arguments
//nolint:gosec // G204: ssh-keygen is a well-known system command, args are controlled
cmd := exec.CommandContext(ctx, "ssh-keygen", args...)
```

## Pre-commit Hooks

### Installation

Pre-commit hooks automatically run linting before each commit, preventing bad code from being committed:

```bash
# Install hooks
pre-commit install

# Verify installation
cat .git/hooks/pre-commit

# Update hooks to latest versions
pre-commit autoupdate
```

### Usage

```bash
# Hooks run automatically on `git commit`
git commit -m "feat: add job filtering"

# Run hooks manually on all files
pre-commit run --all-files

# Run specific hook
pre-commit run gofumpt --all-files

# Skip hooks (emergency only!)
git commit --no-verify
```

### What Hooks Do

The hooks automatically:
1. Remove trailing whitespace from all files
2. Ensure files end with newline
3. Fix YAML syntax issues
4. Detect large files being added
5. Detect merge conflict markers
6. Detect private keys (prevents credential leaks)
7. Fix line endings (enforces LF)
8. Format code with gofumpt
9. Organize imports with goimports
10. Tidy go.mod
11. Run full golangci-lint

If any hook fails, the commit is aborted. You must fix the issues and try again.

## Disabled Linters

Some linters are **disabled** because they require substantial refactoring or don't fit our project patterns.

### cyclop - Cyclomatic Complexity

**Status**: Disabled

**Reason**: Configuration issues and high issue count (136 violations)

### containedctx - Context in Structs

**Status**: Disabled

**Reason**: Architectural pattern used throughout codebase (22+ instances)

### gosec - Security Analysis

**Status**: Enabled but heavily configured

**Excluded rules**:
- **G204**: exec.Command flagged as unsafe (false positive with safe arg separation)
- **G304**: File inclusion via variable (acceptable for app-controlled paths)
- **G115**: Integer overflow conversions (safe in metrics/test code)

## CI/CD Integration

### GitHub Actions

Linting is automatically run on all pull requests via GitHub Actions:

1. **Linting job**: Runs `golangci-lint run` on all changes
2. **Status check**: PR cannot be merged if linting fails
3. **Details**: Click "Details" next to the check to see lint results

### PR Requirements

All pull requests must:
- Pass `golangci-lint run` with no new warnings
- Have code formatted with `gofumpt`
- Have imports organized with `goimports`
- Have go.mod tidied with `go mod tidy`
- Not introduce new linter violations

## Best Practices

### 1. Run Linting Frequently

```bash
# During development
make lint

# Before committing
pre-commit run --all-files

# After pulling from upstream
make lint
```

### 2. Fix Issues Incrementally

Don't wait until the end of development to fix lint issues:

```bash
# After each feature
make fmt    # Fix formatting
make lint   # Check for issues
# Fix any issues immediately
```

### 3. Never Disable Linters Casually

If you feel the need to disable a linter:
1. Discuss with the team first
2. Document the reason in `.golangci.yml`
3. Plan for re-enabling in a future phase

### 4. Document Suppressions

When using `//nolint`, always explain why:

```go
// Bad
x := 5 // nolint

// Good
// nolint:unused // x is used by external system via reflection
x := 5
```

### 5. Keep the Linter Configuration Updated

Periodically:
- Review disabled linters to see if they can be enabled
- Update linter versions when new versions become stable
- Adjust thresholds based on code quality trends

### 6. Use Pre-commit Hooks

Install and use pre-commit hooks:

```bash
pre-commit install

# Hooks now run before every commit
git commit -m "feat: add feature"
```

## Troubleshooting

### Linting Fails Locally but Passes in CI

Usually due to:
1. Different golangci-lint versions - update locally
2. Cached results - run with `--no-cache`
3. Environment differences - check Go version

```bash
# Force fresh lint
golangci-lint run --no-cache

# Check versions
golangci-lint version
go version
```

### Linting Takes Too Long

The 10-minute timeout might be exceeded for large codebases:

```bash
# Check which linters are slow
golangci-lint run --no-cache -v

# Run specific fast linters
golangci-lint run --linters=errcheck,staticcheck
```

### Pre-commit Hook Fails but Need to Commit

**Never ignore linting issues**, but if you absolutely must:

```bash
# Emergency only - skip hooks
git commit --no-verify

# But then fix the issues immediately
make fmt
make lint
git add .
git commit -m "fix: address linting issues"
```

## Summary

Linting is essential for maintaining code quality in s9s. Our 15-linter configuration balances comprehensiveness with practicality. The key principles are:

1. **Automatic enforcement**: Pre-commit hooks catch issues before they're committed
2. **Team consistency**: All developers use the same linting rules
3. **Pragmatic approach**: Disabled linters are disabled for good reasons
4. **Continuous improvement**: Strive to resolve violations incrementally

For questions or issues with linting, see [CONTRIBUTING.md](contributing.md) or open an issue on GitHub.
