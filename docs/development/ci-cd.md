# CI/CD Setup & Linting Gate Configuration

This document explains the s9s CI/CD pipeline, linting enforcement, and how to configure branch protection rules to maintain code quality.

## Table of Contents

- [Overview](#overview)
- [GitHub Actions Workflow](#github-actions-workflow)
- [Linting Gate](#linting-gate)
- [Branch Protection Rules](#branch-protection-rules)
- [Local Testing Before Push](#local-testing-before-push)
- [Troubleshooting](#troubleshooting)
- [Best Practices](#best-practices)

## Overview

The s9s project uses GitHub Actions to enforce code quality standards automatically. Every pull request is checked for:

1. **Code Style & Quality** - golangci-lint validates 15+ code quality rules
2. **Unit Tests** - Tested on Go 1.23 and 1.24
3. **Build Verification** - Ensures code compiles for 6 platform combinations
4. **Security** - Trivy and gosec security scanners detect vulnerabilities

All checks must pass before code can be merged to main.

## GitHub Actions Workflow

### Pipeline Overview

```
┌─────────────┐
│   Trigger   │ (push to main/develop, PR to main)
└──────┬──────┘
       │
       ├─> ┌──────────────────────┐
       │   │  Lint Job (5-8 min)  │ <-- BLOCKS BUILD IF FAILS
       │   │ - golangci-lint      │
       │   │ - 15 linters         │
       │   └──────────┬───────────┘
       │              │
       ├─> ┌──────────v────────────┐
       │   │ Test Job (8-15 min)   │ <-- BLOCKS BUILD IF FAILS
       │   │ - Go 1.23 & 1.24      │
       │   │ - Race detector       │
       │   │ - Coverage upload     │
       │   └──────────┬────────────┘
       │              │
       │   ┌──────────v────────────────────────┐
       │   │ Build Job (5-10 min) - DEPENDS ON │
       │   │ - Linux x86_64, arm64             │ <-- ONLY RUNS IF
       │   │ - macOS x86_64, arm64             │     LINT & TEST PASS
       │   │ - Windows x86_64                  │
       │   └──────────┬────────────────────────┘
       │              │
       └─> ┌──────────v─────────────┐
           │ Security Job (3-5 min) │
           │ - Trivy scanner        │
           │ - Gosec scanner        │
           └────────────────────────┘
```

### Key Configuration

**File**: `.github/workflows/ci.yml`

#### Lint Job
```yaml
jobs:
  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v5
      with:
        go-version: '1.24'
    - name: Install golangci-lint
      run: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.3.0
    - name: Run golangci-lint
      run: golangci-lint run --timeout 10m
```

**What this does**:
- Checks out your code
- Sets up Go 1.24 environment
- Installs golangci-lint v2.3.0
- Runs all 15 enabled linters with 10-minute timeout
- Fails the job if ANY linting violation is found

#### Build Job Dependencies
```yaml
jobs:
  build:
    name: Build
    runs-on: ubuntu-latest
    needs: [lint, test]  # CRITICAL: Requires both to pass
```

The `needs:` directive ensures the build only runs if both lint and test jobs complete successfully.

### Linters Enabled

The CI enforces the same linters as `.golangci.yml`:

**Core Linters**:
- `errcheck` - Unchecked errors (critical)
- `govet` - Suspicious constructs
- `ineffassign` - Ineffective assignments
- `staticcheck` - Static analysis

**Quality Linters**:
- `misspell` - Spelling errors
- `bodyclose` - HTTP response bodies not closed
- `errorlint` - Error wrapping violations
- `wastedassign` - Wasted assignments

**Style & Patterns**:
- `gocritic` - Code patterns and style
- `unused` - Dead code detection
- `nolintlint` - Validate nolint directives
- `revive` - Go idioms enforcement

**Advanced**:
- `gocognit` - Cognitive complexity (threshold: 50)
- `dupl` - Code duplication (threshold: 150 lines)

See `.golangci.yml` and [Linting Standards](linting.md) for complete configuration.

## Linting Gate

### What Is It?

A **linting gate** is an automated enforcement mechanism that prevents code merges unless all linting checks pass. The s9s project implements this through:

1. **GitHub Actions Jobs** - The lint job in `.github/workflows/ci.yml`
2. **Job Dependencies** - Build job depends on lint job passing
3. **Branch Protection Rules** - GitHub setting to require lint job to pass before merge

### How It Works

1. **Developer pushes code to PR**
   ```bash
   git push origin feature/my-feature
   ```

2. **GitHub Actions runs automatically**
   - Lint job runs: `golangci-lint run --timeout 10m`
   - If lint fails: Red X shown on PR
   - If lint passes: Green checkmark shown on PR

3. **PR cannot be merged until lint passes**
   - With branch protection enabled (see below)
   - GitHub shows: "This branch has 1 failing check"
   - Merge button is disabled

4. **Developer fixes issues locally**
   ```bash
   golangci-lint run     # Identify issues
   make fmt              # Fix formatting
   git add . && git commit -m "fix: resolve linting violations"
   git push origin feature/my-feature
   ```

5. **Lint job runs again**
   - All checks now pass
   - Merge button becomes enabled
   - Code can be merged

### Current Status

The linting gate is **ALREADY IMPLEMENTED** in `.github/workflows/ci.yml`:

- Lint job runs on every push and PR
- Build job depends on lint job passing
- Build cannot run if lint fails

**Next Step**: Configure branch protection rules to formally require the lint check (see below).

## Branch Protection Rules

### What Are They?

**Branch protection rules** are GitHub repository settings that enforce policies on branches. For the main branch, we enforce:

1. **Require status checks to pass before merging**
   - Lint check must pass
   - Test check must pass
   - Security check must pass
2. **Require pull request reviews before merging**
3. **Require branches to be up to date before merging**
4. **Require signed commits**

### Configuration Steps

#### Via GitHub Web UI

1. **Go to Repository Settings**
   - Navigate to: https://github.com/jontk/s9s/settings/branches
   - Or: Click "Settings" > "Branches" in left sidebar

2. **Add Rule for `main` Branch**
   - Click "Add rule"
   - Branch name pattern: `main`
   - Click "Create"

3. **Configure Required Status Checks**
   - Scroll to "Require status checks to pass before merging"
   - Enable: "Require branches to be up to date before merging"
   - Search and select required checks:
     - Lint
     - Test
     - Security Scan (optional but recommended)

4. **Require Pull Request Reviews** (recommended)
   - Enable: "Require pull request reviews before merging"
   - Approvals required: 1
   - Enable: "Dismiss stale pull request approvals when new commits are pushed"

5. **Additional Security Options** (recommended)
   - Enable: "Require signed commits"
   - Enable: "Require status checks to pass before merging"

6. **Save Rules**
   - Click "Save changes" button

### Verify Configuration

After enabling branch protection:

1. **Create a test PR with intentional linting error**
   ```bash
   git checkout -b test/lint-gate
   echo "var unusedVar int" >> cmd/s9s/main.go
   git add . && git commit -m "test: intentional linting error"
   git push origin test/lint-gate
   ```

2. **Check GitHub PR page**
   - Should show: "1 failing check" (Lint job)
   - Merge button should be disabled
   - Message: "Status checks failing"

3. **Fix the error**
   ```bash
   git checkout test/lint-gate
   git revert HEAD  # Undo the change
   git push origin test/lint-gate
   ```

4. **Verify merge becomes available**
   - All checks now pass
   - Merge button becomes enabled
   - Can now merge the PR

## Local Testing Before Push

### Pre-commit Hooks (Recommended)

Install pre-commit hooks to catch linting issues before pushing:

```bash
# One-time setup
pre-commit install

# Now before every commit, hooks automatically run:
# - gofumpt (formatting)
# - goimports (import organization)
# - golangci-lint (full linting)
```

See [docs/PRE_COMMIT_SETUP.md](../../docs/PRE_COMMIT_SETUP.md) for detailed setup guide.

### Manual Testing

**Run linting locally before pushing**:

```bash
# Check for violations
golangci-lint run

# Fix formatting issues
make fmt

# Run all checks that CI will run
make test
make lint
make build
```

**Recommended workflow**:

```bash
# 1. Make your changes
vim cmd/s9s/main.go

# 2. Format and lint locally
make fmt
golangci-lint run

# 3. Fix any issues shown
# ... edit files ...

# 4. Run tests
make test

# 5. Build verification
make build

# 6. Only then commit and push
git add .
git commit -m "feat: add new feature"
git push origin feature/my-feature
```

### Common Linting Violations

**Unused variable**:
```go
// Wrong - golangci-lint will flag as 'unused'
var unusedVar string

// Right - Use underscore for intentionally unused
_ = unusedVar
```

**Unused function parameter**:
```go
// Wrong - parameter 'ctx' not used
func processJob(ctx context.Context) error {
    return nil
}

// Right - rename to underscore
func processJob(_ context.Context) error {
    return nil
}
```

**Missing error check**:
```go
// Wrong - error returned but not checked
file.Close()

// Right - check error
if err := file.Close(); err != nil {
    return fmt.Errorf("failed to close file: %w", err)
}
```

**Unclosed HTTP body**:
```go
// Wrong - response body not closed
resp, _ := http.Get(url)
data := resp.Body  // bodyclose violation

// Right - defer close
resp, err := http.Get(url)
if err != nil {
    return err
}
defer resp.Body.Close()  // Ensures body is closed
```

**Error wrapping**:
```go
// Wrong - old error wrapping format
return errors.New(fmt.Sprintf("error: %s", err.Error()))

// Right - use %w for proper error chaining
return fmt.Errorf("failed operation: %w", err)
```

## Troubleshooting

### Lint Job Fails on PR

**Problem**: PR shows red X on "Lint" check

**Solution**:

1. **View the CI logs**
   - Click on the red X next to "Lint"
   - Click "Details" to see full error messages

2. **Identify the violation**
   - Log shows: `cmd/s9s/main.go:23:5: unused-parameter: parameter 'cmd' is unused [unused]`
   - This means variable `cmd` in file `cmd/s9s/main.go` line 23 is not used

3. **Fix locally and test**
   ```bash
   git fetch origin
   git checkout feature/my-feature
   golangci-lint run  # See the same error locally

   # Fix: rename parameter to underscore
   vim cmd/s9s/main.go  # Change `cmd` to `_`
   golangci-lint run  # Verify fixed
   ```

4. **Push the fix**
   ```bash
   git add cmd/s9s/main.go
   git commit -m "fix: remove unused parameter"
   git push origin feature/my-feature
   ```

### Different Linting Results Locally vs CI

**Problem**: `golangci-lint run` passes locally but fails in CI

**Causes**:
- Different golangci-lint version (CI uses v2.3.0)
- Different Go version (CI uses 1.24, you might have 1.23)
- Incomplete module cache

**Solution**:

```bash
# 1. Check your local version
golangci-lint version

# 2. If different from v2.3.0, install correct version
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.3.0

# 3. Update Go modules
go mod tidy

# 4. Clear build cache
go clean -cache

# 5. Run again
golangci-lint run
```

### How to Skip a Linting Check

**Use `//nolint` directive** (when absolutely necessary):

```go
// Only for legitimate exceptions, with explanation:
var shadowedErr error
if err != nil {
    shadowedErr = err  // nolint:shadowed // intentional for deferred handling
}
```

**Requirements**:
- Must be specific: `//nolint:rulename` not just `//nolint`
- Must have explanation: `// nolint:rulename // explanation`
- nolintlint linter validates these directives

See [Linting Standards](linting.md#using-nolint-directives) for complete guidelines.

### Merge Blocked by Status Checks

**Problem**: "This branch has 1 failing check" message, merge button disabled

**Causes**:
1. Lint job is still running (wait for completion)
2. Lint job failed (see "Lint Job Fails on PR" above)
3. Test job failed (check test logs)
4. Branch is out of date with main

**Solution**:

1. **If branch is out of date**:
   ```bash
   git fetch origin
   git rebase origin/main
   # Fix any conflicts
   git push origin feature/my-feature --force-with-lease
   ```

2. **If tests are failing**:
   - Click on red X next to "Test"
   - Click "Details" to see test output
   - Fix the failing test locally

3. **Wait for all checks to complete**
   - GitHub shows progress: "X of 3 checks passing"
   - Once all show checkmark, merge button becomes enabled

## Best Practices

### For Developers

1. **Install pre-commit hooks** (first-time setup)
   ```bash
   pre-commit install
   ```

2. **Run checks before pushing**
   ```bash
   make fmt
   make lint
   make test
   ```

3. **Write meaningful commit messages**
   - Follow Conventional Commits format (feat:, fix:, docs:, etc.)
   - Reference issue numbers when appropriate

4. **Respond to CI feedback quickly**
   - Check failing checks immediately
   - Fix and push updated commits same day
   - Don't let PRs accumulate unaddressed CI failures

### For Code Reviews

1. **Check CI status before reviewing**
   - All status checks should show checkmark
   - Don't approve PRs with failing checks

2. **Verify linting and tests**
   - Ensure "Lint" and "Test" are green
   - Don't merge if CI red

3. **Require updates before merge**
   - If branch becomes out of date with main
   - Have author rebase and push: `git rebase origin/main && git push --force-with-lease`

### For Maintainers

1. **Monitor CI performance**
   - Lint should take 5-8 minutes
   - If slower, investigate golangci-lint timeout issues

2. **Keep dependencies current**
   - Update golangci-lint version quarterly
   - Update Go version in `.github/workflows/ci.yml`
   - Keep linter rule set updated

3. **Archive old build artifacts**
   - Builds are uploaded as artifacts
   - Archive old artifacts to save storage

## Related Documentation

- [CONTRIBUTING.md](contributing.md) - Contribution guidelines and setup
- [LINTING.md](linting.md) - Linting standards and rules
- [PRE_COMMIT_SETUP.md](../../docs/PRE_COMMIT_SETUP.md) - Pre-commit hook configuration
- [.golangci.yml](../../.golangci.yml) - Linting configuration
- [.github/workflows/ci.yml](../../.github/workflows/ci.yml) - CI/CD workflow

## Questions?

- Check existing [GitHub Issues](https://github.com/jontk/s9s/issues)
- Review [CI workflow logs](https://github.com/jontk/s9s/actions) for recent runs
- Read [Contributing Guide](contributing.md#linting-and-code-quality)
