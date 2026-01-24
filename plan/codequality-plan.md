# Code Quality Improvement Plan: Lint Exclusion Reduction & Enhanced Linting

**Project**: s9s (Go TUI Application)
**Status**: Strategic Planning
**Lead**: Go Lead Developer
**Created**: 2026-01-24

---

## Executive Summary

This document outlines a strategic, phased approach to reduce lint exclusions and progressively enable additional linters to improve code quality across the s9s project. Currently, the project has **8 disabled linters** with approximately **352 pre-existing issues**, and **0 inline nolint directives**. Our goal is to incrementally address these issues and achieve a state where the codebase passes all recommended Go linters with minimal exclusions.

### Current State Analysis

**Enabled Linters (7):**
- errcheck (unchecked errors)
- govet (suspicious constructs)
- ineffassign (ineffective assignments)
- misspell (spelling errors)
- bodyclose (HTTP response body closing)
- errorlint (error wrapping patterns)
- wastedassign (wasted assignments)

**Disabled Linters (8) - Pre-existing Issues:**
- staticcheck: 71 issues (SA9003, S1009, SA1019, QF1008)
- gosec: 145 issues (weak random, file permissions, API misuse)
- gocritic: 51 issues (style, performance, diagnostics)
- unparam: 38 issues (unused parameters)
- prealloc: 16 issues (performance optimizations)
- unconvert: 8 issues (unnecessary type conversions)
- unused: 1 issue
- noctx: 23 issues (missing context in HTTP requests)

**Total Pre-existing Issues**: ~352

---

## Strategic Goals

1. **Phase 1 (Weeks 1-2)**: Reduce high-severity issues from staticcheck and gosec
2. **Phase 2 (Weeks 3-4)**: Address medium-priority issues from remaining disabled linters
3. **Phase 3 (Weeks 5-6)**: Enable all linters with clean status
4. **Phase 4 (Ongoing)**: Maintain code quality and add advanced linters

### Success Metrics

- [ ] Zero nolint directives (inline exclusions)
- [ ] All 15+ recommended Go linters enabled
- [ ] <10 file-level exclusion rules in .golangci.yml
- [ ] 100% pass rate on critical linters (errcheck, govet, staticcheck, gosec)
- [ ] Code coverage maintained at ≥80%
- [ ] Build time for linting <5 seconds

---

## Detailed Phase Breakdown

### Phase 1: High-Severity Linter Issues (staticcheck + security)

**Objective**: Fix 71 staticcheck issues and 145 gosec issues

**Key Issues to Address**:

1. **staticcheck (71 issues)**
   - SA9003: Empty branches in conditionals
   - S1009: Unnecessary nil checks
   - SA1019: Deprecated function usage (strings.Title)
   - QF1008: Unnecessary embedded fields

2. **gosec (145 issues)**
   - G404: Weak random number generation in auth package
   - G301/G306: Insecure file permissions in test/setup contexts
   - G304: File inclusion via variable (application-controlled)

**Approach**:
- Audit each category of issues
- Group by package/component
- Implement targeted fixes (most can be automated or simple refactors)
- Validate security implications don't introduce vulnerabilities

**Deliverables**:
- PR(s) resolving 80%+ of issues
- Updated .golangci.yml with gosec and staticcheck re-enabled
- Security audit report for gosec fixes

---

### Phase 2: Medium-Priority Linter Issues (code quality)

**Objective**: Fix issues from gocritic, unparam, prealloc, unconvert

**Key Issues to Address**:

1. **gocritic (51 issues)**
   - ifElseChain: Simplifiable conditionals
   - singleCaseSwitch: Simplifiable switch statements
   - dupBranchBody: Duplicated branch logic

2. **unparam (38 issues)**
   - Unused function parameters
   - Functions always returning nil errors

3. **prealloc (16 issues)**
   - Slices that should be pre-allocated

4. **unconvert (8 issues)**
   - Unnecessary type conversions

**Approach**:
- Automated fixes where possible (gofmt-like changes)
- Manual review for semantic changes
- Accept style preferences when appropriate
- Create architectural improvements for persistent patterns

**Deliverables**:
- PR(s) implementing fixes with explanation
- Updated gocritic exclusion list (only style preferences)
- Refactored hot paths for performance

---

### Phase 3: Remaining Issues (test context, unused detection)

**Objective**: Fix noctx (23) and unused (1) issues; evaluate unconvert (8)

**Key Issues to Address**:

1. **noctx (23 issues)** - Missing context in HTTP requests
   - Mostly in test files (acceptable to exclude)
   - Some in actual code paths (need fixing)

2. **unused (1 issue)** - Trivial
   - Single unused variable or function

**Approach**:
- Context additions to actual code paths
- Exclude noctx for test files in .golangci.yml
- Remove unused code or add justification

**Deliverables**:
- All linters enabled with minimal exclusions
- Clean .golangci.yml without test-specific broad exclusions

---

### Phase 4: Advanced Linting & Maintenance

**Objective**: Add additional linters and maintain quality

**New Linters to Consider**:
- **cyclop**: Cyclomatic complexity checking
- **gocognit**: Cognitive complexity analysis
- **revive**: Configurable rule engine for Go
- **nolintlint**: Detect superfluous nolint directives
- **dupl**: Find code duplication
- **wsl**: Whitespace linting
- **containedctx**: Prevent storing context in structs

**Approach**:
- Gradually introduce linters with conservative thresholds
- Build supporting tooling (lint violations dashboard)
- Establish team lint standards
- Integrate with pre-commit hooks

**Deliverables**:
- Enhanced .golangci.yml with 15+ linters
- Pre-commit hook configuration
- Lint violations CI/CD gate

---

## Implementation Strategy

### Code Organization by Package

```
Priority 1 (High): Foundational & Security
├── internal/auth/          (25+ gosec issues)
├── internal/config/        (20+ gosec issues)
└── cmd/                    (security-adjacent)

Priority 2 (Medium): Core Logic
├── internal/app/
├── internal/dao/
├── internal/discovery/
└── internal/ssh/

Priority 3 (Low): UI/Utilities
├── internal/views/
├── internal/ui/
├── internal/layouts/
└── pkg/
```

### Workflow Process

1. **Issue Analysis**: Run linters individually, categorize issues
2. **Impact Assessment**: Identify high-risk, high-frequency issues
3. **Fix Implementation**: Create targeted PRs by package
4. **Validation**: Ensure no test coverage regression
5. **Documentation**: Update CONTRIBUTING.md with lint standards
6. **Enforcement**: CI/CD integration and pre-commit hooks

---

## Risk Mitigation

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaking changes in fixes | Medium | Comprehensive test coverage, staged rollout |
| Security regressions | High | Security-focused code review, gosec re-validation |
| Performance regression | Medium | Benchmarking before/after critical paths |
| Schedule slip | Medium | Break into smaller PRs, clear priorities |
| False positive handling | Low | Maintain minimal exclusion rules |

---

## Timeline & Milestones

```
Week 1 (Jan 24-31):
  - Audit all 352 issues
  - Phase 1 fixes (staticcheck SA9003, SA1019)
  - Phase 1 PRs submitted

Week 2 (Jan 31-Feb 7):
  - Phase 1 complete (staticcheck + gosec security)
  - Phase 2 begins (gocritic, unparam)

Week 3 (Feb 7-14):
  - Phase 2 in progress
  - Refactoring complex patterns

Week 4 (Feb 14-21):
  - Phase 2 complete
  - Phase 3 begins (remaining issues)

Week 5 (Feb 21-28):
  - All phases complete
  - Linters fully enabled

Week 6+ (Mar onwards):
  - Phase 4: Additional linters
  - Maintenance & enforcement
```

---

## Success Criteria

- ✅ All disabled linters re-enabled and passing
- ✅ Zero nolint inline directives in codebase
- ✅ Minimal file-level exclusions (<10 rules)
- ✅ All tests passing (coverage maintained)
- ✅ Security review completed for gosec fixes
- ✅ Documentation updated
- ✅ CI/CD linting gate active

---

## Dependencies & Prerequisites

- [ ] Go 1.21+
- [ ] golangci-lint 1.54+
- [ ] Team alignment on linting standards
- [ ] Code review process established
- [ ] Test suite running in CI/CD

---

## Appendix: Reference Materials

### Linter Documentation
- [staticcheck](https://staticcheck.dev/) - Lint checker for Go
- [gosec](https://github.com/securego/gosec) - Security scanner
- [gocritic](https://github.com/go-critic/go-critic) - Opinionated linter
- [golangci-lint](https://golangci-lint.run/) - Meta-linter framework

### Best Practices
- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [CWE Top 25](https://cwe.mitre.org/top25/) - Security patterns to avoid
