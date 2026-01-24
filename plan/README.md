# Code Quality Improvement Initiative - Planning Documents

**Status**: Planning Complete ✓
**Created**: 2026-01-24
**Lead**: Go Lead Developer
**Audience**: Development Team, Maintainers

---

## 📋 Overview

This directory contains the comprehensive strategic and implementation plan for reducing lint exclusions and enhancing code quality across the s9s project through systematic linter enablement.

### Deliverables in This Directory

1. **codequality-plan.md** (283 lines)
   - Executive summary and strategic vision
   - Current state assessment (352 pre-existing issues)
   - Phased approach across 4 phases
   - Timeline, milestones, and success criteria
   - Risk mitigation strategies

2. **codequality-spec.md** (789 lines)
   - Detailed technical specification
   - Complete requirements breakdown
   - Linter-by-linter remediation strategies
   - Testing and validation procedures
   - Success metrics and KPIs

---

## 🎯 Current State Summary

**Total Pre-existing Issues**: 352

| Category | Count | Severity |
|----------|-------|----------|
| gosec (security) | 145 | CRITICAL |
| staticcheck (logic) | 71 | HIGH |
| gocritic (style/perf) | 51 | MEDIUM |
| unparam (params) | 38 | MEDIUM |
| prealloc (optimization) | 16 | LOW |
| unconvert (type safety) | 8 | LOW |
| noctx (context) | 23 | LOW |
| unused (dead code) | 1 | LOW |
| **TOTAL** | **352** | |

---

## 📊 Work Breakdown Structure

### Created Artifacts

- **4 Epics**: Phase 1-4 organizational structure
- **18 Main Tasks**: Core work items by linter/category
- **6 Child Tasks**: Detailed breakdowns for Phase 1 audits
- **Total Issues**: 30 (open, ready for work)

### Phase 1: High-Severity Issues (P0)
**Status**: Ready to Start
**Epics**: 1 (s9s-qki)
**Tasks**: 8 primary + 6 subtasks

```
✓ Audit: Staticcheck Issues by Category (s9s-27x)
  ├─ Scan: golangci-lint staticcheck capture (s9s-27x.1)
  ├─ Categorize: Group by SA9003, S1009, SA1019, QF1008 (s9s-27x.2)
  └─ Plan: Document fix complexity (s9s-27x.3)

✓ Fix: Staticcheck SA9003 - Empty Branches (s9s-w3f)
✓ Fix: Staticcheck S1009 - Unnecessary nil Checks (s9s-tur)
✓ Fix: Staticcheck SA1019 - Deprecated Functions (s9s-vuf)
✓ Fix: Staticcheck QF1008 - Embedded Fields (s9s-0tx)

✓ Security Audit: Gosec G404 - Weak Random (s9s-7n8)
  ├─ Audit: Identify G404 uses in internal/auth/ (s9s-7n8.1)
  ├─ Security: Distinguish crypto vs non-crypto (s9s-7n8.2)
  └─ Fix/Exclude: Apply remediation (s9s-7n8.3)

✓ Fix: Gosec G301/G306 - File Permissions (s9s-uey)
✓ Fix: Gosec G304 - Variable File Inclusion (s9s-14w)
✓ Security Review: Gosec Fixes & Re-enable (s9s-9o8)
✓ Test & Validate: Phase 1 Staticcheck Fixes (s9s-232)
```

**Blocked by**: None (22 issues ready to work)
**Blocks**: Phase 2

### Phase 2: Code Quality Issues (P1)
**Status**: Awaiting Phase 1 Completion
**Epics**: 1 (s9s-5j8)
**Tasks**: 4 primary

```
✓ Fix: Gocritic Issues - Style & Performance (s9s-oee)
✓ Fix: Unparam - Unused Parameters & Error Returns (s9s-dna)
✓ Optimize: Prealloc - Slice Pre-allocation (s9s-rwx)
✓ Fix: Unconvert - Remove Type Conversions (s9s-m3h)
```

**Blocked by**: Phase 1 (s9s-qki)
**Blocks**: Phase 3

### Phase 3: Remaining Issues (P2)
**Status**: Awaiting Phase 2 Completion
**Epics**: 1 (s9s-2qz)
**Tasks**: 2 primary

```
✓ Fix: Unused - Remove Dead Code (s9s-dty)
✓ Fix: Noctx - Context in HTTP Requests (s9s-c88)
```

**Blocked by**: Phase 2 (s9s-5j8)
**Blocks**: Phase 4

### Phase 4: Advanced Linters & Maintenance (P3)
**Status**: Future Work
**Epics**: 1 (s9s-790)
**Tasks**: 1 primary

```
✓ Phase 4: Add Advanced Linters (s9s-im1)
  Includes: cyclop, gocognit, revive, nolintlint, dupl, wsl, containedctx
```

**Blocked by**: Phase 3 (s9s-2qz)

### Cross-Phase Tasks
**Status**: Awaiting Core Completion
**Tasks**: 2

```
✓ Enable All Linters & Finalize Configuration (s9s-y4v)
✓ Documentation: Update CONTRIBUTING.md (s9s-wk4)
✓ CI/CD: Integrate Linting Gate & Pre-commit (s9s-e13)
```

**Blocked by**: Phase 1+ (s9s-232)

---

## 🚀 How to Get Started

### 1. Review the Plans
```bash
# Read strategic overview
cat plan/codequality-plan.md

# Read technical specification
cat plan/codequality-spec.md
```

### 2. View Available Work
```bash
# See all ready issues (no blockers)
bd ready

# See specific phase details
bd show s9s-qki   # Phase 1 epic
bd show s9s-27x   # Staticcheck audit task
bd show s9s-7n8   # Gosec security audit task
```

### 3. Claim Work
```bash
# Start working on a task
bd update s9s-27x --status=in_progress

# View your active work
bd list --status=in_progress
```

### 4. Track Progress
```bash
# Check project health
bd stats

# View blocked issues waiting for dependencies
bd blocked

# Show dependencies for a specific task
bd show s9s-5j8
```

### 5. Sync & Complete Work
```bash
# Before leaving session: sync beads changes
bd sync

# When work is done
bd close s9s-27x --reason="Completed staticcheck audit"
```

---

## 📈 Success Metrics

### Phase Completion Goals

| Phase | Issues | Blockers | Success Criteria |
|-------|--------|----------|-----------------|
| **1** | 8 + 6 subtasks | None | staticcheck + gosec enabled, 216/352 issues resolved |
| **2** | 4 | Phase 1 done | gocritic + unparam + prealloc enabled, 280/352 resolved |
| **3** | 2 | Phase 2 done | unused + noctx enabled, 304/352 resolved |
| **4** | 1 | Phase 3 done | Advanced linters enabled, ongoing maintenance |

### Final State Target (Phase 3 Complete)
- ✅ **0** nolint directives in codebase
- ✅ **<10** file-level exclusion rules in .golangci.yml
- ✅ **15+** linters enabled and passing
- ✅ **100%** test coverage maintained (≥80%)
- ✅ **0** pre-existing linter issues remaining
- ✅ **<5** seconds linting time in CI/CD

---

## 📚 Implementation References

### Key Files to Know
- `.golangci.yml` - Current linter configuration
- `plan/codequality-plan.md` - Strategic planning document
- `plan/codequality-spec.md` - Technical specification
- `CONTRIBUTING.md` - (Will be updated with linting standards)

### External Resources
- [staticcheck Documentation](https://staticcheck.dev/)
- [gosec Security Scanner](https://github.com/securego/gosec)
- [golangci-lint](https://golangci-lint.run/)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)

---

## ⚠️ Important Notes

### Phase Ordering
Tasks must be completed in phase order due to dependencies. Phase 1 must complete before Phase 2 can begin, etc.

### Documentation First
Both planning documents provide the complete context for implementation. Refer to them for:
- Detailed linter-by-linter fix strategies
- Security implications of gosec decisions
- Testing and validation procedures
- Risk mitigation approaches

### Git Workflow
```bash
# Before session end, ALWAYS:
1. git status              (check what changed)
2. git add <files>         (stage code changes)
3. bd sync                 (commit beads changes)
4. git commit -m "..."     (commit code changes)
5. git push                (push to remote)
```

---

## 📋 Quick Command Reference

```bash
# View planning
bd ready                           # See available work
bd show <id>                       # View issue details
bd list --status=in_progress       # Your active work

# Update work
bd update <id> --status=in_progress # Start working
bd update <id> --status=completed  # Finish task

# Manage
bd sync                            # Sync with git
bd blocked                         # Show blocked work
bd stats                           # Project statistics
bd doctor                          # Check for issues
```

---

## 🎓 Learning Path for Team

1. **Day 1**: Read `codequality-plan.md` (strategic overview)
2. **Day 2**: Read `codequality-spec.md` (technical details)
3. **Day 3**: Review Phase 1 tasks, understand dependencies
4. **Day 4+**: Begin Phase 1 implementation

---

## 📞 Questions & Discussion

For questions about:
- **Strategy**: See `plan/codequality-plan.md`
- **Technical Details**: See `plan/codequality-spec.md`
- **Task Progress**: Run `bd show <issue-id>`
- **Linter Fixes**: See section in spec corresponding to linter
- **Dependencies**: Run `bd blocked` or `bd show <id>`

---

**Last Updated**: 2026-01-24
**Next Review**: After Phase 1 completion
