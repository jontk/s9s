# Linting Standards and Guidelines

This document covers s9s linting standards, configuration, and best practices. Linting is a critical part of our development workflow to ensure code quality, consistency, and reliability.

## Table of Contents

- [Linting Philosophy](#linting-philosophy)
- [Enabled Linters](#enabled-linters)
- [Running Linters](#running-linters)
- [Fixing Lint Issues](#fixing-lint-issues)
- [Pre-commit Hooks](#pre-commit-hooks)
- [Disabled Linters](#disabled-linters)
- [CI/CD Integration](#cicd-integration)
- [Best Practices](#best-practices)

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
- **460 revive violations → 36** (92% reduction)
- All violations fixed except 36 backward-compatible type aliases
- Full backward compatibility maintained

**What was fixed**:
1. **Phase 1 - Quick Wins** (287 violations, 62% reduction)
   - Added 37 package comments (all packages now documented)
   - Fixed 244+ unused parameters → renamed to `_` (118 files)
   - Fixed 6 var-naming violations

2. **Phase 2 - Documentation** (102 violations, 22% reduction)
   - Added godoc comments to 102 exported symbols
   - All public APIs now properly documented (43 files)

3. **Phase 3 - Structural Improvements** (95 violations, 21% reduction)
   - Fixed 31 variables shadowing Go built-ins (max→maxVal, min→minVal, etc.)
   - Refactored 38 stuttering type names with backward-compatible aliases
   - Fixed 7 minor violations (empty-block, indent-error-flow, increment-decrement)

**Remaining 36 Violations** (Acceptable):
- Type aliases for backward compatibility: `type AuthProvider = Provider`
- Maintains existing API while providing cleaner public names
- Standard Go practice for API migration

**Impact**:
- 170+ files modified
- 1000+ lines changed
- Code quality significantly improved
- Code now follows Go idioms and best practices
- All tests passing, clean builds

See PR #29 for complete details: [systematically fix all 460 linter violations through 3-phase approach](https://github.com/jontk/s9s/pull/29)

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

- **staticcheck**: Comprehensive static analysis covering several categories
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
  - See [Disabled Linters](#disabled-linters) for details on exclusions

### Style and Pattern Linters

- **gocritic**: Style and code pattern checks
  - Detects code smells and anti-patterns
  - Diagnostic, performance, and style checks
  - Helps catch subtle bugs in logic
  - Several checks disabled due to style preferences

- **revive**: Go idioms and style enforcement
  - Package and exported symbol documentation
  - Naming conventions (avoids stuttering names)
  - Error handling patterns
  - All 460+ issues fixed in PR #29 (see [Recent Improvements](#recent-improvements---phase-7-pr-29))
  - Only 36 acceptable violations remain (type aliases for backward compatibility)

- **unused**: Finds unused variables, constants, functions, and types
  - Helps clean up dead code
  - Can reveal incomplete refactoring
  - Works across package boundaries

### Advanced Linters

- **nolintlint**: Validates `//nolint` directives
  - Ensures linter suppressions are actually necessary
  - Prevents accumulated technical debt
  - Can detect orphaned nolint comments
  - Requires specific rule names and explanations

## Linter Configuration

The `.golangci.yml` file contains our linter configuration with three main sections:

### run
- **timeout**: 10m - Linting timeout (prevents hanging on large files)
- **tests**: true - Include test files in linting
- **skip-dirs**: Excludes vendor, .git, .github directories

### linters
- **disable-all**: true - Start with all linters disabled, explicitly enable only those we want
- **enable**: List of 15 enabled linters (described above)
- **disable**: List of linters we deliberately don't use

### linters-settings
Specific configuration for each linter:

- **errcheck**: Checks type assertions, allows `_ = expr` for intentional ignores
- **govet**: Enables shadow detection (shadows of parent scope variables)
- **gosec**: Medium severity/confidence threshold with specific rule exclusions
- **gocritic**: Enables diagnostic, performance, and style check tags; disables specific checks
- **errorlint**: Checks errorf with %w, type assertions, and error comparisons
- **misspell**: US English locale
- **gocognit**: Cognitive complexity minimum of 30 (allows complex but understandable code)
- **nolintlint**: Validates nolint directives; disallows unused directives and requires no explanation
- **dupl**: Minimum 150 lines to consider as duplication
- **containedctx**: Prevents context in struct fields (configured to skip tests)

### issues
- **exclude-files**: Excludes generated files (.pb.go, .gen.go)
- **exclude-dirs**: Excludes vendor, third_party, testdata
- **exclude-rules**: Specific exclusions for test files and known patterns

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

### Pre-commit Hooks

Pre-commit hooks automatically run before each commit:

```bash
# Install hooks (one-time setup)
pre-commit install

# Run all hooks manually
pre-commit run --all-files

# Run specific hook
pre-commit run golangci-lint --all-files

# Run hooks on changed files only
pre-commit run

# Skip hooks for emergency commit
git commit --no-verify
```

Configured hooks run in this order:
1. trailing-whitespace - Removes trailing spaces
2. end-of-file-fixer - Ensures files end with newline
3. check-yaml - Validates YAML syntax
4. check-added-large-files - Prevents large files
5. check-merge-conflict - Detects merge conflict markers
6. detect-private-key - Prevents committing secrets
7. mixed-line-ending - Enforces consistent line endings
8. gofumpt - Format with gofumpt
9. goimports - Organize imports
10. go-mod-tidy - Tidy go.mod
11. golangci-lint - Run full linting

## Fixing Lint Issues

### Understanding Linter Messages

Each linter message includes:
- **File path and line number**: Where the issue is
- **Linter name**: Which linter detected it (in parentheses)
- **Issue description**: What the problem is
- **Severity**: Usually implied (errors are critical, style is minor)

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

The `//nolint` directive suppresses specific linter warnings. **Use sparingly** - accumulated nolint directives are technical debt.

Valid reasons for nolint:
- **Intentional patterns**: Code that intentionally violates a linter rule
- **False positives**: Linter incorrectly flags valid code
- **Generated code**: Code created by tools can't be fixed
- **Test requirements**: Testing code that needs different patterns

Invalid reasons:
- **Laziness**: Just to avoid fixing code
- **Disagreement**: Rather than having a team discussion
- **Temporary**: "I'll fix it later" (usually never happens)

#### Proper Nolint Usage

**Philosophy**: Use explicit `//nolint` directives at violation points with justification comments. This is better than project-wide exclusions because:
- Reviewers see the reasoning inline, not buried in config
- Each suppression is intentional and documented
- Easy to track and incrementally fix issues
- Enables nolintlint validation to prevent accumulating technical debt

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

#### Nolint Validation

The **`nolintlint`** linter enforces:
- Specific rule names required (no blanket `//nolint`)
- Explanations required (comments documenting the reason)
- No unused directives (prevents accumulated technical debt)

This ensures suppressions remain intentional and justified.

## Global Exclusions vs. Nolint Directives

We use a **hybrid approach**:

### Project-wide Exclusions (Minimal)

Only used for genuine false positives that apply everywhere:
- **G204 (gosec)**: exec.Command flagged as unsafe when args are separated properly
  - Our code validates command paths and passes args as separate strings
  - This is a known false positive in gosec
  - Single global exclusion prevents noise while still allowing nolint elsewhere

### Nolint Directives (Primary)

Security and linter violations are suppressed with explicit `//nolint` comments:
- **G304 (gosec)**: File path operations - suppressed when paths are app-controlled
- **G404 (gosec)**: Weak randomness - suppressed in specific contexts
- **G101 (gosec)**: Hardcoded credentials - suppressed for test RSA keys
- **Other rules**: Documented inline with justification

**Advantage**: Code reviewers see the reasoning at the violation point, not hidden in config. Each suppression is intentional and can be tracked for removal.

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
1. **Remove trailing whitespace** from all files
2. **Ensure files end with newline**
3. **Fix YAML syntax** issues (if applicable)
4. **Detect large files** being added (prevents accidental commits)
5. **Detect merge conflict** markers
6. **Detect private keys** (prevents credential leaks)
7. **Fix line endings** (enforces LF)
8. **Format code with gofumpt** (modifies files)
9. **Organize imports with goimports** (modifies files)
10. **Tidy go.mod** (modifies files)
11. **Run full golangci-lint** (checks for violations)

If any hook fails, the commit is aborted. You must fix the issues and try again.

### Recommended Workflow

1. Make code changes
2. Run `make fmt` to auto-fix formatting
3. Run `make lint` to check for issues
4. Fix any remaining issues manually
5. Run `git add` to stage changes
6. Run `git commit` - pre-commit hooks run automatically
7. If hooks fail, fix issues and try commit again

## Disabled Linters

Some linters are **disabled** because they require substantial refactoring or don't fit our project patterns. Below is the status of each disabled linter:

### cyclop - Cyclomatic Complexity

**Status**: Disabled (Phase 4)

**Reason**: Configuration issues and high issue count

**Pre-existing issues**: 136 violations across the codebase

**Details**:
- Measures number of decision paths through a function
- High cyclomatic complexity indicates functions that are hard to test and maintain
- 136 issues is substantial and would require significant refactoring
- Configuration in `.golangci.yml` had issues that prevented proper linting

**Path to resolution**:
- Configure complexity thresholds appropriately
- Refactor functions with excessive decision paths
- May re-enable in future phase when complexity is reduced

### gocognit - Cognitive Complexity

**Status**: Enabled (Phase 4)

**Configuration**: `min-complexity: 30` (allows moderate cognitive complexity)

**Details**:
- Measures how difficult code is to understand (different from cyclomatic)
- Fewer pre-existing violations than cyclop
- Threshold of 30 is reasonable for existing codebase
- Helps identify code that's hard to understand

### containedctx - Context in Structs

**Status**: Disabled (Phase 4)

**Reason**: Architectural pattern used throughout codebase

**Pre-existing issues**: 22+ instances of context.Context stored in struct fields

**Details**:
- Recommends passing context.Context as a function parameter instead of storing in structs
- Our application architecture uses context in struct fields for state management
- Changing this would require significant refactoring throughout codebase
- Not worth the cost given the pattern is consistently applied

**Trade-off**:
- Linter rule conflicts with project architecture
- Context handling is consistent, even if it differs from linter recommendation
- Re-enable only if architecture is significantly refactored

### dupl - Code Duplication

**Status**: Enabled (Phase 4)

**Configuration**: `threshold: 150` (only flag large duplications)

**Details**:
- Detects duplicated code blocks of 150+ lines
- Helps identify opportunities for abstraction
- Some code duplication is acceptable (test data, similar patterns)
- Test files are excluded from this check

### gosec - Security Analysis

**Status**: Enabled but heavily configured (Phase 2+)

**Details**:
- Comprehensive security vulnerability detection
- Configured with multiple rule exclusions
- Excluded rules:
  - **G204**: exec.Command with string arguments (false positive with safe arg separation)
  - **G304**: File inclusion via variable (acceptable for app-controlled paths)
  - **G115**: Integer overflow conversions (safe in metrics/test code)
  - **G602**: Slice index out of range (false positives in test code)
  - **G101**: Hardcoded credentials in tests (acceptable for test RSA keys)

**Pre-existing issues**: 87 security issues in various forms

**Current approach**:
- Exclusions documented in `.golangci.yml`
- Security-critical code has been audited
- Remaining issues either false positives or low-risk patterns
- Continued monitoring and incremental fixes

## CI/CD Integration

### GitHub Actions

Linting is automatically run on all pull requests via GitHub Actions:

1. **Linting job**: Runs `golangci-lint run` on all changes
2. **Status check**: PR cannot be merged if linting fails
3. **Details**: Click "Details" next to the check to see lint results

### PR Requirements

All pull requests must:
- ✅ Pass `golangci-lint run` with no new warnings
- ✅ Have code formatted with `gofumpt`
- ✅ Have imports organized with `goimports`
- ✅ Have go.mod tidied with `go mod tidy`
- ✅ Not introduce new linter violations

### Local vs. CI Differences

CI runs the full linter suite, while `--new-from-rev` checks only changed files. This means:

- **Locally**: `golangci-lint run --new-from-rev=HEAD~1` (only changed files)
- **CI**: `golangci-lint run` (all files) - detects if you accidentally reverted fixes

Always run full linting before pushing: `golangci-lint run`

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

### 7. Review Linting Errors Carefully

Don't blindly suppress lint errors:

```go
// Bad - just suppress it
rows.Close() // nolint:errcheck

// Good - understand and handle the error
if err := rows.Close(); err != nil {
    return fmt.Errorf("failed to close rows: %w", err)
}
```

### 8. Involve Team in Linter Decisions

Linting rules affect the whole team:
- Discuss before changing thresholds
- Document decisions in `.golangci.yml` comments
- Include linting changes in PRs with explanations

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

For questions or issues with linting, see [CONTRIBUTING.md](CONTRIBUTING.md#-linting-and-code-quality) or open an issue on GitHub.
