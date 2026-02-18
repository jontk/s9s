# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

### Changed

### Fixed

### Removed

## [0.5.0] - 2026-02-18

### Added

- **Zero-Configuration Auto-Discovery** (#81)
  - s9s now works out-of-the-box on SLURM systems without requiring a configuration file
  - Automatic discovery of slurmrestd endpoint and JWT token
  - API version auto-detection from slurmrestd (removed hardcoded `v0.0.43` default)
  - When a config file exists with explicit contexts, those values take precedence over discovery

- **Dark Mode Logo Support** (#76)
  - Logo adapts to dark and light terminal themes

- **Integration Tests CI Workflow** (#80)
  - PR testing with ephemeral k3d cluster and mock SLURM API
  - Main branch testing against real SLURM cluster in brokkr-prod
  - Race detection, coverage reporting (Codecov), and automatic resource cleanup
  - Debug output on failure (pod logs, cluster events)

### Changed

- **Static Binary Builds** (#81)
  - Added `CGO_ENABLED=0` to Makefile for cross-distribution compatibility (NixOS, Rocky, Ubuntu, etc.)

- **SLURM Authentication** (#81)
  - Changed from `auth.NewTokenAuth()` to `slurm.WithUserToken()` to set both `X-SLURM-USER-NAME` and `X-SLURM-USER-TOKEN` headers
  - Fixes 401 Unauthorized errors with slurmrestd

- **API Version Handling** (#81)
  - Removed hardcoded `v0.0.43` default; API version is now auto-detected from the slurmrestd endpoint
  - Ensures s9s always uses the latest version supported by the cluster

### Fixed

- **Discovery Order** (#81)
  - Fixed auto-discovery to discover endpoint → token → create context (previously the context was created before the token was discovered, losing it)

- **Config Fallback** (#81)
  - Fixed `findClusterConfig()` to fall back to `cfg.Cluster` when no contexts exist, preventing discovered tokens from being lost during initialization

- **Debug System** (#81)
  - Rewrote debug logging to use `io.Discard` by default, only writing to log file when `--debug` is passed
  - Fixes debug messages overlaying the TUI display

### Documentation

- Added slurm-client Go library to API reference (#78)
- Updated installation instructions and default paths (#77)

## [0.4.0] - 2026-02-08

### Added

- **Performance View with Cluster Metrics** (#72, #73)
  - New cluster-focused Performance view accessible via key `9`
  - Real-time cluster-wide metrics: Jobs (total/running/pending), Nodes (total/active/idle/down), Resources (CPU%/Memory%)
  - Color-coded resource utilization bars (green: 0-75%, yellow: 75-90%, red: 90-100%)
  - Auto-refresh every 5 seconds with manual refresh support (`R` to toggle, `F5` for immediate refresh)
  - Comprehensive documentation with interpretation guide and example scenarios
  - Integrated demo GIF showing cluster metrics and auto-refresh

- **App Diagnostics View** (#72)
  - Debug-only view for monitoring s9s application performance (hidden by default)
  - Enables via `features.appDiagnostics: true` in config
  - Shows app-level metrics: memory, goroutines, CPU, network, operations
  - Useful for debugging s9s performance issues

- **Sorting Modal for All Table Views** (#71)
  - Press `S` key to open interactive sorting modal in all table-based views
  - Visual sort indicators in table headers showing current sort column and direction
  - Sort indicators preserved during table refresh operations
  - Added sorting demonstrations to jobs, nodes, and partitions demos

- **Reservation Time-Based Filters** (#63)
  - Press `a` to toggle "Active Only" filter (shows reservations currently running)
  - Press `f` to toggle "Future Only" filter (shows reservations not yet started)
  - Visual feedback with green checkmarks (✓) in status hints when filters active
  - Filters use OR logic when both enabled (excludes only past reservations)

- **Job Output File Reading** (#62)
  - Core infrastructure for reading SLURM job output files (stdout/stderr)
  - Supports local filesystem reading (NFS, Lustre, GPFS)
  - Remote file access via SSH for distributed clusters
  - Efficient tail/head operations for large files with configurable limits
  - Context-aware cancellation and thread-safe operations
  - New `internal/output/` package with comprehensive test coverage

- **Command Mode with Arguments** (#59)
  - Vim-style command mode with argument support
  - Intelligent Tab completion for commands and arguments
  - Job operations: `:cancel JOBID`, `:hold JOBID`, `:release JOBID`, `:requeue JOBID`
  - Node operations: `:drain NODE [REASON]`, `:resume NODE`
  - Autocomplete suggests job IDs and node names from current view

- **Demo GIFs with Git LFS** (#60)
  - 12 comprehensive demo GIFs (~17MB total) for documentation
  - Git LFS integration to prevent repository bloat
  - Auto-regeneration workflow via GitHub Actions
  - Covers all major views and features (accounts, dashboard, health, jobs, nodes, partitions, QoS, reservations, users, search, job submission, overview)

- **VHS Demo Recordings** (#56)
  - Automated demo recording infrastructure using VHS
  - Terminal recordings for documentation and website integration
  - Consistent theme and timing across all demos

### Changed

- **Performance View Purpose** (#72)
  - Changed from app diagnostics to cluster performance monitoring
  - Focus on cluster health assessment and capacity planning
  - App profiling tools moved to separate App Diagnostics view

- **Dependencies** (#74, #64)
  - Updated `slurm-client` to v0.3.0 (from v0.2.5)
    - SDK architectural refactor with adapter pattern for improved extensibility
    - Multiple bug fixes and lint improvements
  - Previous update to latest slurm-client API version (#64)

### Fixed

- **Critical Performance Dashboard Deadlock** (#72)
  - Fixed race condition causing s9s to freeze when accessing Performance view
  - Root cause: Background goroutine holding mutex while calling `QueueUpdateDraw()` blocked by main thread trying to acquire same mutex
  - Solution: Lock-free callbacks with copied data and prevention of concurrent updates
  - Removed 93 lines of unused update methods after refactor

- **Architectural Quality & Correctness** (#67)
  - Fixed UI threading race conditions from background goroutines updating UI directly
  - Fixed channel panic on PerformanceDashboard stop from closed channel
  - Added nil check in job output fallback preventing crash
  - Stabilized flaky performance test with `testing.Short()` guard
  - Precompiled 7 regexes in cache key generator (3.5x speedup: 28µs → 8µs)
  - Added 4,100+ lines of comprehensive tests (performance dashboard: 100%, health monitoring: 96.6%, views: 95%+)
  - Replaced `interface{}` with typed interfaces eliminating runtime assertions

- **Performance Monitoring Runtime Controls** (#66)
  - Improved lifecycle management of performance monitoring components
  - Better resource cleanup and goroutine management

- **Observability Production Hardening** (#61, #69)
  - Fixed goroutine leaks in observability plugin
  - Added lifecycle tests for proper start/stop behavior
  - Prevented information disclosure in validation errors

- **Demo Recording Improvements** (#70)
  - Comprehensive fixes to demo recordings for accuracy
  - Removed non-functional sorting/grouping demonstrations
  - Fixed misleading sort hints across all views
  - Corrected accounts view sorting hint and demo search term
  - Removed redundant view jumping from dashboard demo

- **Sort Indicators in MultiSelectTable** (#71)
  - Fixed sort indicators being lost during table refresh
  - Sort column and direction now properly preserved

- **Nodes View 'S' Key Binding** (#71)
  - Fixed 'S' key not opening sort modal in nodes view
  - Now consistent with all other table-based views

### Documentation

- **Complete Documentation Restructure** (#57, #58, #73)
  - Per-view guides with comprehensive keyboard shortcuts and features
  - Removed all unimplemented features from documentation
  - Added Performance view documentation with interpretation guide
  - Demo GIF integration throughout documentation
  - Example scenarios for cluster health assessment

- **Updated README** (#73)
  - Changed Performance feature description to accurately reflect cluster monitoring
  - Updated feature list to match current capabilities

### Internal Improvements

- **Test Coverage** (#67)
  - Strengthened coverage thresholds to prevent regression
  - Added cross-package coverage tracking with `-coverpkg` flag
  - Per-package coverage enforcement in CI (dao: 70%, monitoring: 80%, config: 80%)
  - Performance dashboard: 100% coverage on metric calculations
  - Health monitoring: 96.6% package coverage
  - View calculations: 95%+ coverage on critical functions

- **Code Quality** (#67)
  - Removed empty test/unit package references from Makefile
  - Type safety improvements throughout codebase

## [0.3.0] - 2026-01-30

### Added
- **Job submission working directory default**: Working directory now defaults to the current directory where s9s is started
- **Vim-style view navigation**: `h` and `l` keys for switching between views (left/right)
- **Global search cross-view navigation**: Search results now navigate to the correct view and focus on the selected item
- **Partition filter syntax**: Use `p:partition_name` in filter fields to filter by partition (Jobs and Nodes views)
- **Users view admin filter**: Press `a` to toggle showing only administrators/operators
- **Status bar feedback**: Users view shows status messages when toggling admin filter
- **Job display improvements**: Global search shows jobs with username, state, and partition
- **Username resolution**: Jobs display actual usernames instead of numeric UIDs (with system lookup fallback)
- Centralized version management in `internal/version` package
- GoReleaser configuration for automated releases
- CHANGELOG.md following Keep a Changelog format
- GitHub Actions release workflow
- Version-aware build targets in Makefile

### Changed
- **Keyboard shortcuts changed to uppercase** to avoid vim navigation conflicts:
  - `h` → `H` for Hold (Jobs), Show Hierarchy (Accounts), Health Details (Health)
  - `j` → `J` for Jobs view (Dashboard, Partitions)
  - `n` → `N` for Nodes view (Dashboard, Partitions)
  - `p` → `P` for Partitions view (Dashboard)
  - `a` → `A` for Analytics (Dashboard, Partitions)
  - `w` → `W` for Wait Times (Partitions)
- **Dependency upgrade**: Updated slurm-client to upstream v0.2.4 (from v0.2.3)
- **Filter performance**: Added debouncing (300ms) and immediate client-side filtering for better UX
- **Modal consistency**: All modals now use the `pages` API for proper lifecycle management
- **Linting configuration**: Use file-specific cyclop exclusions (max-complexity: 10) instead of global threshold increase

### Fixed
- **Linting issues**: Resolved all errcheck, noctx, staticcheck, and unused linter violations
- **Modal focus issues**: Input fields in drain and partition filter modals now maintain focus properly
- **Drain operation**: Fixed drain showing success but not actually draining nodes
- **Resume operation**: Fixed modal double-removal causing operation to fail
- **Node state display**: Fixed state truncation - now shows `IDLE+DRAIN` instead of just `IDLE`
- **Filter input hijacking**: Global shortcuts no longer steal focus from filter input fields
- **Search deadlocks**: Removed QueueUpdateDraw calls from event handlers that caused freezes
- **Configuration**: Environment variables now properly override config file values
- **Case sensitivity**: Admin level comparison in Users view is now case-insensitive

### Removed
- **Loading modals**: Removed intrusive "Loading..." modals from Accounts, QoS, Reservations, and Users views
- **Header update time**: Removed misleading "Updated: X ago" display (was tracking app startup, not data refresh)
- **Local slurm-client replace directive**: Now using published upstream version

## [0.1.0] - 2026-01-21

### Added
- **Core TUI Application**: Terminal-based user interface for SLURM cluster management
  - Vim-like navigation and keybindings
  - Real-time job and node monitoring
  - Interactive job management (submit, cancel, hold, release)
  - Multiple view modes (jobs, nodes, partitions, accounts, QOS, reservations, users)
  - SSH terminal integration for direct node access
  - Plugin system for extensibility

- **Configuration Management**:
  - Multi-cluster support with context switching
  - Configuration wizard (`s9s setup`) with auto-discovery
  - YAML-based configuration
  - Environment variable support
  - Mock mode for testing and development

- **Export Capabilities**:
  - CSV, JSON, and Markdown export formats
  - Job output streaming and filtering
  - Performance metrics export

- **Observability Plugin**:
  - Prometheus integration for metrics collection
  - Historical data collection and analysis
  - Performance dashboards
  - Resource efficiency analysis
  - Security audit logging
  - Rate limiting and circuit breaking

- **Testing & Quality**:
  - Comprehensive test suite
  - Mock SLURM client for testing
  - Integration tests
  - CI/CD with GitHub Actions

- **Code Quality**:
  - golangci-lint v2 integration
  - All errcheck, ineffassign, and govet issues resolved
  - Race condition fixes with proper synchronization
  - Thread-safe components

### Changed
- Upgraded to Go 1.24
- Updated slurm-client to upstream version v0.0.0-20260120203936
- Improved CI reliability with non-blocking security scans

### Fixed
- Race conditions in StatusBar, PerformanceDashboard, and MetricCache
- Deadlock in HistoricalDataCollector Stop() method
- Test isolation issues in key manager tests
- Config parser now supports both flat dotted keys and nested maps

### Security
- Security audit logging in observability plugin
- Rate limiting for API requests
- Circuit breaker pattern for fault tolerance
- Secrets management with encryption

## [Initial Development] - 2026-01-17 to 2026-01-20

### Added
- Initial project structure
- SLURM client integration
- Basic TUI components
- Plugin architecture
- Authentication and authorization
- SSH integration
- Export functionality

---

**Note**: Versions prior to 0.1.0 were in active development and did not follow semantic versioning.

[Unreleased]: https://github.com/jontk/s9s/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/jontk/s9s/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/jontk/s9s/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/jontk/s9s/compare/v0.1.0...v0.3.0
[0.1.0]: https://github.com/jontk/s9s/releases/tag/v0.1.0
