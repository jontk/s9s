# Code Quality Specification: Lint System Enhancement

**Document Type**: Technical Specification
**Version**: 1.0
**Last Updated**: 2026-01-24
**Audience**: Developers, Maintainers, QA

---

## 1. Objectives & Requirements

### 1.1 Primary Objectives

1. **Reduce Exclusions**: Eliminate all nolint directives and minimize file-level exclusions
2. **Increase Coverage**: Re-enable all disabled linters systematically
3. **Improve Signals**: Catch more issues early (developer experience)
4. **Maintain Quality**: Ensure no performance or security regressions
5. **Establish Standards**: Create sustainable linting culture

### 1.2 Requirements Specification

#### Functional Requirements (FR)

| ID | Requirement | Priority | Acceptance Criteria |
|----|-------------|----------|------------------|
| FR-1 | Enable staticcheck linter | P0 | 71 issues fixed, linter enabled, no new issues |
| FR-2 | Enable gosec security linter | P0 | 145 issues fixed, security review passed |
| FR-3 | Enable gocritic linter | P1 | 51 issues addressed, style exclusions documented |
| FR-4 | Enable unparam linter | P1 | 38 issues fixed, test exclusions applied |
| FR-5 | Enable prealloc linter | P2 | 16 issues fixed, hot paths optimized |
| FR-6 | Enable unconvert linter | P2 | 8 issues fixed, conversion patterns improved |
| FR-7 | Enable unused linter | P2 | 1 issue fixed, dead code removed |
| FR-8 | Enable noctx linter | P2 | Test exclusions applied, production code fixed |
| FR-9 | Zero nolint directives | P0 | Codebase audit shows 0 inline exclusions |
| FR-10 | CI/CD linting gate | P1 | Linting required before merge, documentation updated |

#### Non-Functional Requirements (NFR)

| ID | Requirement | Target | Measurement |
|----|-------------|--------|-------------|
| NFR-1 | Lint execution time | <5 seconds | golangci-lint run timing |
| NFR-2 | Code coverage maintenance | ≥80% | go test -cover results |
| NFR-3 | No performance regression | <5% | Benchmarks before/after |
| NFR-4 | Security audit completion | 100% | Gosec fix review checklist |
| NFR-5 | Documentation completeness | 100% | CONTRIBUTING.md updated |

---

## 2. Current State Assessment

### 2.1 Inventory of Issues

```
Total Pre-existing Linter Issues: 352

By Severity:
  - CRITICAL (Security):     145 (gosec)
  - HIGH (Correctness):       71 (staticcheck)
  - MEDIUM (Code Quality):   110 (gocritic, unparam, prealloc, unconvert)
  - LOW (Style/Test):         26 (unused, noctx)

By Category:
  - Security:                145 gosec issues
  - Logic Errors:             71 staticcheck issues
  - Style/Performance:       110 issues
  - Test Infrastructure:      26 issues

By Package:
  Priority 1 (Security):
    - internal/auth/:        ~40 issues
    - internal/config/:      ~35 issues
    - cmd/:                  ~30 issues

  Priority 2 (Core):
    - internal/app/:         ~50 issues
    - internal/dao/:         ~40 issues
    - internal/discovery/:   ~35 issues

  Priority 3 (UI/Utils):
    - internal/views/:       ~30 issues
    - internal/ui/:          ~25 issues
    - pkg/:                  ~22 issues
```

### 2.2 Current Configuration

**Location**: `.golangci.yml` (v2)

**Enabled Linters**: 7
- errcheck, govet, ineffassign, misspell, bodyclose, errorlint, wastedassign

**Disabled Linters**: 8
- staticcheck, gosec, gocritic, unparam, prealloc, unconvert, unused, noctx

**Inline Exclusions**: 0 (none currently)

**File-level Exclusions**: 15 rules in `.golangci.yml`

---

## 3. Target State Definition

### 3.1 Linter Configuration Target

#### Phase 3 Target (All Issues Resolved)

```yaml
linters:
  disable-all: false
  enable:
    # Critical correctness linters
    - errcheck
    - govet
    - staticcheck
    - gosec
    - errorlint

    # Code quality linters
    - gocritic
    - unparam
    - ineffassign
    - unconvert
    - wastedassign

    # Performance linters
    - prealloc

    # Spelling & formatting
    - misspell
    - bodyclose
    - unused

    # Test context checking
    - noctx
```

#### Phase 4 Target (Enhanced Linting)

Add additional linters:
- cyclop (cyclomatic complexity limit: 20)
- gocognit (cognitive complexity limit: 30)
- revive (rule engine for advanced checks)
- nolintlint (detect superfluous nolint)
- dupl (code duplication)
- wsl (whitespace standards)
- containedctx (context in structs)

### 3.2 Exclusion Rules Target

**Current**: 15 rules
**Target (Phase 3)**: <10 rules
**Target (Phase 4)**: <5 rules

**Allowable Exclusions by Type**:
```
1. Generated code patterns (*.pb.go, *.gen.go)
2. Third-party and vendor code
3. Test files (specific rules only):
   - noctx for _test.go (HTTP context not critical in tests)
   - unparam for _test.go (test helper signatures may be generic)
4. Exceptional cases (must be documented):
   - Strategic weak randomness (if any)
   - File permission quirks (Windows compatibility)
```

### 3.3 Inline Directives Target

**Target**: 0 nolint directives
**Strategy**: Fix issues rather than exclude them

**Why**:
- Exclusions hide code quality issues
- Maintenance burden increases over time
- False security of "known issues"
- Testing reveals actual problems

---

## 4. Linter-by-Linter Specification

### 4.1 PHASE 1: staticcheck

**Current State**: Disabled, 71 issues

**Issues Breakdown**:
- SA9003 (empty branch): ~20 issues
- S1009 (nil check): ~15 issues
- SA1019 (deprecated): ~20 issues
- QF1008 (embedded field): ~16 issues

**Fix Strategy**:

1. **SA9003 - Empty Branches**
   ```go
   // Bad
   if condition {
   } else {
       doSomething()
   }

   // Good
   if !condition {
       doSomething()
   }
   ```
   - Action: Invert condition or use early return
   - Effort: Low (automated in most cases)

2. **S1009 - Unnecessary nil Check**
   ```go
   // Bad
   if x != nil && x.DoSomething() { }

   // Good (if guaranteed x != nil)
   if x.DoSomething() { }
   ```
   - Action: Remove redundant nil check
   - Effort: Low-Medium (needs context review)

3. **SA1019 - Deprecated Function**
   ```go
   // Bad
   title := strings.Title(name)

   // Good
   title := cases.Title(language.Und).String(name)
   ```
   - Action: Replace with non-deprecated alternative
   - Effort: Medium (requires Go 1.18+ compatibility check)

4. **QF1008 - Embedded Field**
   ```go
   // Review for necessity of embedding
   type Server struct {
       *http.Server  // Can often be composition instead
   }
   ```
   - Action: Evaluate embedding vs composition
   - Effort: Medium (API implications)

**Acceptance Criteria**:
- [ ] 0 SA9003 issues remaining
- [ ] 0 S1009 issues remaining
- [ ] 0 SA1019 issues remaining (or documented compatibility reasons)
- [ ] 0 QF1008 issues remaining (or API reviewed)
- [ ] All tests passing
- [ ] Code coverage ≥80%
- [ ] Security review passed

---

### 4.2 PHASE 1: gosec (Security)

**Current State**: Disabled, 145 issues

**Issues Breakdown**:
- G404 (weak random): ~40 issues
- G301/G306 (file perms): ~50 issues
- G304 (file inclusion): ~35 issues
- Other: ~20 issues

**Fix Strategy**:

1. **G404 - Weak Random**

   **Affected Package**: `internal/auth/` (load balancer, session tokens)

   ```go
   // Bad
   rand.Intn(max)  // Weak for crypto

   // Good (if cryptographic)
   crypto/rand.Intn(max)

   // Or exclude if legitimate weak randomness
   // (e.g., load balancing, non-security shuffle)
   ```
   - Action: Use crypto/rand for security-critical, evaluate business need
   - Audit: Distinguish between security vs non-security use cases
   - Exclusion: G404 in specific contexts may be acceptable
   - Effort: Medium

2. **G301/G306 - File Permissions**

   **Affected Areas**: Tests, setup wizards, exports

   ```go
   // Context: Test data, setup, or UI export
   os.Mkdir(path, 0777)      // OK in test/setup context

   // Production code should use restrictive perms
   os.Mkdir(path, 0700)      // Better
   ```
   - Action: Add file permission context awareness
   - Exclusion: Safe to exclude for test/setup paths
   - Effort: Low

3. **G304 - File Inclusion via Variable**

   **Context**: Application-controlled paths (config, filters)

   ```go
   // Application config, not user input
   path := config.ExportPath  // Safe - controlled by app
   os.Open(path)              // OK to exclude
   ```
   - Action: Validate app-controlled vs user-supplied distinction
   - Exclusion: Can exclude for config/filters paths
   - Effort: Low-Medium

**Acceptance Criteria**:
- [ ] Security audit completed for all G404 usages
- [ ] Weak random usages documented (business justification)
- [ ] File permission context verified
- [ ] File inclusion paths confirmed app-controlled
- [ ] <5 exclusion rules remain
- [ ] No new security vulnerabilities introduced
- [ ] Passes external security review

---

### 4.3 PHASE 2: gocritic

**Current State**: Disabled, 51 issues

**Disabled Checks** (intentional style preferences):
- ifElseChain
- singleCaseSwitch
- dupBranchBody

**Strategy**:
- Accept style preference exclusions for above 3 categories
- Fix other diagnostics and performance issues
- Enabled checks: diagnostic, performance (non-style)

**Target exclusions**: Keep only style preference exclusions

**Acceptance Criteria**:
- [ ] All non-style issues fixed
- [ ] Performance issues addressed in hot paths
- [ ] Style exclusions documented and justified
- [ ] No code duplication remaining

---

### 4.4 PHASE 2: unparam

**Current State**: Disabled, 38 issues

**Issue Categories**:
- Unused parameters in public APIs (30 issues)
- Functions always returning nil errors (8 issues)

**Strategy**:

1. **API Compatibility**:
   - Public functions: May need parameters for API contract/extensibility
   - Rationale: Document for future use vs remove
   - Approach: Exclude test files, review public APIs

2. **Error Returns**:
   - Some functions designed to return error for extensibility
   - OK to exclude with documentation

**Exclusion Rules**:
```yaml
- linters:
    - unparam
  path: "_test\\.go"  # Test helpers often have generic signatures
```

**Acceptance Criteria**:
- [ ] Test files excluded
- [ ] Public API decisions documented
- [ ] No unused parameters in production code (except documented)
- [ ] <3 exclusion rules

---

### 4.5 PHASE 2: prealloc

**Current State**: Disabled, 16 issues

**Strategy**: Hot path optimization

```go
// Before (no prealloc)
var items []Item
for _, v := range input {
    items = append(items, process(v))
}

// After (preallocated)
items := make([]Item, 0, len(input))
for _, v := range input {
    items = append(items, process(v))
}
```

**Targeting**:
- Focus on loops in performance-critical paths
- Measure impact on hot paths only
- Skip low-frequency operations

**Acceptance Criteria**:
- [ ] Hot paths identified and optimized
- [ ] Benchmarks show <5% performance improvement
- [ ] Memory allocation reduced for large datasets
- [ ] Readability maintained

---

### 4.6 PHASE 2: unconvert

**Current State**: Disabled, 8 issues

**Strategy**: Remove unnecessary type conversions

```go
// Bad
var x int = 5
y := int(x)  // Unnecessary conversion

// Good
y := x
```

**Acceptance Criteria**:
- [ ] All unnecessary conversions removed
- [ ] Type consistency verified
- [ ] No behavior changes

---

### 4.7 PHASE 3: unused

**Current State**: Disabled, 1 issue

**Strategy**: Remove dead code

**Acceptance Criteria**:
- [ ] Dead code identified and removed
- [ ] API breakage assessed
- [ ] Tests still passing

---

### 4.8 PHASE 3: noctx

**Current State**: Disabled, 23 issues

**Mostly Test Files**: ~20 issues acceptable in test context

**Production Code**: ~3 issues that need fixing

**Strategy**:

```yaml
# Exclude from test files
- linters:
    - noctx
  path: "_test\\.go"

# Fix production code contexts
- Must add context to actual HTTP calls where appropriate
```

**Acceptance Criteria**:
- [ ] Test files excluded from noctx
- [ ] Production code HTTP calls use context
- [ ] Context propagation traced through call stack
- [ ] Timeout configuration documented

---

## 5. Implementation Roadmap

### 5.1 Phase 1 Implementation Tasks

```
1.1 - Audit staticcheck issues by category
1.2 - Fix SA9003 empty branches (automated where possible)
1.3 - Fix S1009 unnecessary nil checks
1.4 - Fix SA1019 deprecated functions
1.5 - Review and fix QF1008 embedded fields
1.6 - Test and validate Phase 1
1.7 - Re-enable staticcheck in .golangci.yml

2.1 - Security audit for gosec issues
2.2 - Fix G404 weak random uses
2.3 - Document/exclude G301/G306 file permission cases
2.4 - Validate G304 file inclusion contexts
2.5 - Address remaining gosec issues
2.6 - Test and security review Phase 1
2.7 - Re-enable gosec in .golangci.yml
```

### 5.2 Phase 2 Implementation Tasks

```
3.1 - Review gocritic issues by package
3.2 - Fix diagnostic and performance issues
3.3 - Document style preference exclusions
3.4 - Test and validate Phase 2
3.5 - Re-enable gocritic in .golangci.yml

4.1 - Analyze unparam issues
4.2 - Review test file impact
4.3 - Evaluate public API signatures
4.4 - Apply exclusion rules
4.5 - Test and validate
4.6 - Re-enable unparam in .golangci.yml

5.1 - Identify hot paths for prealloc
5.2 - Implement slice preallocation
5.3 - Benchmark improvements
5.4 - Re-enable prealloc in .golangci.yml

6.1 - Fix unconvert issues
6.2 - Test and validate
6.3 - Re-enable unconvert in .golangci.yml
```

### 5.3 Phase 3 & 4 Tasks

```
7.1 - Fix unused code (1 issue)
8.1 - Exclude noctx for test files
8.2 - Fix production code context issues
9.1 - Enable all linters
9.2 - Document standards in CONTRIBUTING.md
10.1 - Add Phase 4 linters (cyclop, gocognit, etc.)
10.2 - Set quality thresholds
10.3 - Integrate linting gate in CI/CD
```

---

## 6. Testing & Validation Strategy

### 6.1 Pre-Fix Testing

1. Run full test suite: `go test ./...`
2. Generate coverage baseline: `go test -cover ./...`
3. Run benchmarks on hot paths: `go test -bench=. ./...`
4. Document baseline metrics

### 6.2 Post-Fix Validation

1. **Functional Testing**:
   - All tests pass
   - Coverage maintained ≥80%
   - Integration tests pass

2. **Lint Validation**:
   - Linter-specific tests pass
   - No new issues introduced
   - Exclusion count reduced

3. **Performance Testing**:
   - Benchmarks stable or improved
   - Memory allocation reduced (prealloc)
   - Linting time <5 seconds

4. **Security Review**:
   - gosec fixes reviewed
   - No new vulnerabilities
   - File permissions verified
   - Randomness usage justified

### 6.3 Regression Testing

- Pre-commit hook catches issues
- CI/CD linting gate enforces standards
- Manual code review for security/performance
- Canary testing before full rollout

---

## 7. Documentation & Maintenance

### 7.1 Documentation to Update

- [ ] `CONTRIBUTING.md` - Linting standards
- [ ] `.golangci.yml` - Add inline documentation
- [ ] `README.md` - Code quality section
- [ ] GitHub wiki - Development setup guide
- [ ] PR template - Linting checklist

### 7.2 Developer Guidelines

**Linting Standards Document** (to be created):
```
## Code Quality Standards

### Linting Requirements
- All code must pass `golangci-lint run`
- No inline nolint directives (fix issues instead)
- File-level exclusions require justification

### Pre-commit Checks
1. gofmt: Code formatting
2. goimports: Import organization
3. golangci-lint: Lint checks

### Common Issues & Fixes
- errcheck: Always check errors
- govet: Use shadow for variable consistency
- staticcheck: Avoid deprecated functions
- gosec: Review security implications
```

### 7.3 Maintenance Process

```yaml
Quarterly Reviews:
  - Run full linter audit
  - Review new linter versions
  - Update .golangci.yml with latest best practices
  - Measure metrics (build time, issue counts)

Monthly Check-ins:
  - Review nolint directives (should be zero)
  - Check linting time trend
  - Resolve CI/CD lint failures within 24 hours

Continuous:
  - Lint checks in pre-commit hooks
  - CI/CD gate enforcing linting
  - Code review includes lint considerations
```

---

## 8. Success Metrics & KPIs

| Metric | Current | Target (Phase 3) | Target (Phase 4) | Measurement |
|--------|---------|------------------|------------------|-------------|
| Pre-existing Issues | 352 | 0 | 0 | golangci-lint output |
| Enabled Linters | 7 | 15+ | 20+ | .golangci.yml config |
| Nolint Directives | 0 | 0 | 0 | grep nolint count |
| Exclusion Rules | 15 | <10 | <5 | .golangci.yml rules |
| Lint Pass Rate | ~65% | 100% | 100% | CI/CD gate |
| Linting Time | N/A | <5s | <5s | CI/CD logs |
| Code Coverage | ≥80% | ≥80% | ≥80% | go test coverage |
| Security Issues | ~145 | 0 | 0 | gosec scan |
| Developer Satisfaction | N/A | High | High | Survey/feedback |

---

## 9. Risk Assessment & Mitigation

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|-----------|
| Breaking API changes | Medium | High | Test-first approach, staged rollout |
| Security regression | Low | Critical | External security review, gosec validation |
| Performance degradation | Low | Medium | Benchmarking, prealloc optimization |
| Schedule delays | Medium | Medium | Break into smaller PRs, parallel work |
| False positives in exclusions | Low | Low | Thorough review, limit exclusions |
| Incomplete issues | Medium | Medium | Issue tracking in beads, dependency links |

---

## 10. Approval & Sign-Off

**Document Prepared By**: Go Lead Developer
**Date**: 2026-01-24
**Status**: Ready for Review

**Approvals Required**:
- [ ] Tech Lead Review
- [ ] Project Manager
- [ ] Security Team (for gosec decisions)
- [ ] Maintainer Consensus

---

## Appendix A: Issue Categorization Details

### A.1 By Priority & Category

**Security-Critical (P0)**:
- gosec G404: Weak randomness (40 issues) → Crypto audit needed
- gosec G301/G306: File permissions (50 issues) → Context-aware
- gosec G304: Variable paths (35 issues) → Validation needed

**High Priority (P1)**:
- staticcheck: Logic errors (71 issues) → Straightforward fixes
- gocritic: Performance (20 issues) → Hot path optimization

**Medium Priority (P2)**:
- unparam: Unused params (38 issues) → API review needed
- prealloc: Slice optimization (16 issues) → Performance tuning
- unconvert: Type conversions (8 issues) → Simple cleanup

**Low Priority (P3)**:
- unused: Dead code (1 issue) → Straightforward removal
- noctx: Context in tests (23 issues) → Test exclusion sufficient

### A.2 Package Impact Analysis

```
Highest Impact Packages:
  1. internal/auth/       - ~40 issues (security + logic)
  2. internal/config/     - ~35 issues (security + style)
  3. cmd/                 - ~30 issues (security + quality)
  4. internal/app/        - ~50 issues (quality + performance)
  5. internal/dao/        - ~40 issues (quality + logic)

Medium Impact:
  6. internal/discovery/  - ~35 issues
  7. internal/ssh/        - ~20 issues
  8. internal/views/      - ~30 issues

Lower Impact:
  9. internal/ui/         - ~25 issues
  10. pkg/                - ~22 issues
```

---

## Appendix B: Configuration Templates

### B.1 Phase 3 Target .golangci.yml Section

```yaml
linters:
  disable-all: false
  enable:
    - errcheck
    - govet
    - staticcheck
    - gosec
    - errorlint
    - gocritic
    - unparam
    - ineffassign
    - unconvert
    - wastedassign
    - prealloc
    - misspell
    - bodyclose
    - unused
    - noctx

linters-settings:
  gocritic:
    disabled-checks:
      - ifElseChain      # Style preference
      - singleCaseSwitch # Style preference
      - dupBranchBody    # Acceptable patterns

issues:
  exclude-files:
    - ".*\\.pb\\.go$"
    - ".*\\.gen\\.go$"

  exclude-dirs:
    - vendor
    - third_party
    - testdata

  exclude-rules:
    # noctx: Test HTTP requests don't need context
    - linters:
        - noctx
      path: "_test\\.go"

    # Minimal production exclusions (well-justified only)
```

---

## Appendix C: Phase 4 Linters Evaluation

### Advanced Linters to Consider

| Linter | Purpose | Threshold | Recommendation |
|--------|---------|-----------|-----------------|
| **cyclop** | Cyclomatic complexity | 20 | High priority - early detection |
| **gocognit** | Cognitive complexity | 30 | High priority - code readability |
| **revive** | Rule engine | Custom | Medium - flexible rules |
| **nolintlint** | Detect superfluous nolint | Any | High - enforce cleanliness |
| **dupl** | Code duplication | 3 tokens | Medium - refactoring opportunities |
| **wsl** | Whitespace standards | Strict | Low - cosmetic |
| **containedctx** | Context in structs | Any | High - prevent context leaks |
| **funlen** | Function length | 80 lines | Medium - cyclomatic correlation |

---
