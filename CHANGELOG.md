# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

### Changed

### Fixed

### Removed

## [0.7.1] - 2026-03-21

### Added

- **Configurable SLURM username per cluster** — new `user` field in cluster config overrides the `X-SLURM-USER-NAME` header, for environments where the local OS user doesn't match a SLURM user (e.g., laptops, CI runners, containers, or any host outside the cluster's shared user directory) (#143, #145)
- **`SLURM_USER_NAME` environment variable** — overrides all other username resolution methods (#144)
- **`config.ResolveSlurmUser()`** — shared helper for consistent username resolution across auth and wizard

### Changed

- **Username resolution chain** — `SLURM_USER_NAME` env > config `cluster.user` > `USER` env > OS current user. No silent `root` fallback; fails explicitly if no username can be determined
- **Dev clusters script** — writes `user: root` directly in config instead of requiring env var wrappers

### Fixed

- **Job submission fails when local user differs from SLURM user** — `X-SLURM-USER-NAME` header sent the local OS username instead of the SLURM user the JWT token was generated for, causing `slurm_submit_batch_job()` failures (#143)
- **Node drain/resume fails when local user lacks SLURM admin** — same username mismatch caused `_update_node` permission errors
- **Account dropdown empty when local user not in SLURM** — wizard looked up the local OS username in SLURM's user database instead of the configured user

## [0.7.0] - 2026-03-16

### Added

- **Configurable job submission templates** — define custom templates with pre-filled values, global form defaults, hidden fields, and restricted dropdown options via `views.jobs.submission` config section
- **3-tier template merge** — templates from built-in (8), config YAML, and user-saved JSON (`~/.s9s/templates/`) merge with name-based override; controlled by `templateSources` config
- **84 SLURM sbatch fields** — job submission wizard supports 84 of 117 SLURM OpenAPI JobCreate fields (was 12); fields organized in visibility tiers (7 always visible, 7 default, 70 hidden — shown via templates)
- **`s9s templates list`** — CLI command to list all templates from all sources with source indicator (builtin/config/saved)
- **`s9s templates export`** — CLI command to export built-in and config templates to `~/.s9s/templates/` as editable JSON files; supports `--force`, `--dir`, and single template name filter
- **`SubmitRaw` integration** — job submission bypasses the lossy 12-field `slurm.JobSubmission` intermediate and maps directly to `*slurm.JobCreate` via `SubmitRaw`; fields previously silently dropped (QoS, GPUs, output/error files, mail, array, exclusive, requeue, dependencies, constraints, and 50+ more) now reach slurmrestd
- **Full `--begin` time parsing** — supports ISO dates, US dates, relative times (`now+1hour`), and SLURM named times (`midnight`, `noon`, `fika`, `teatime`, `tomorrow`)
- **`--signal` parsing** — `B:USR1@300` correctly sets KillWarningSignal, KillWarningDelay, and KillWarningFlags
- **QoS dropdown from cluster API** — QoS field now fetches available values from the cluster (like partitions and accounts), filtered by `fieldOptions`
- **User default account and QoS** — form defaults to the current SLURM user's `DefaultAccount` and `DefaultQoS`; falls back to username as account if no default is set
- **Smart field visibility** — advanced fields (70+) are hidden by default and only shown when a template or `formDefaults` sets a non-zero value for them
- **Ctrl+Y copy-to-clipboard in preview** — copies clean plain-text sbatch script via OSC 52 escape sequence

### Changed

- **Single source of truth for built-in templates** — `createDefaultTemplates()` now delegates to `BuiltinTemplates()` instead of maintaining a separate list
- **Legacy field normalization** — saved templates using `command`/`cpus_per_node` (old format) are automatically normalized to `script`/`cpus` on load
- **`showBuiltinTemplates` superseded** — replaced by `templateSources` config option; old flag still works for backward compatibility
- **`templateSources` validation** — invalid source names are silently filtered; all-invalid falls back to default
- **Form modal widened** — 80x35 → 100x40 to accommodate additional fields; long labels shortened
- **Viper key normalization** — config map keys are normalized to lowercase before lookup, fixing Viper's automatic lowercasing of camelCase YAML keys (e.g., `timeLimit` → `timelimit`)

### Fixed

- **Job submission fields silently dropped** — QoS, GPUs, output/error files, email notifications, array spec, exclusive, requeue, dependencies, and constraints were collected by the form but never sent to slurmrestd
- **Saved template field mismatch** — templates created by the old form flow used `command`/`cpus_per_node` fields that the wizard didn't read
- **Built-in template scripts had duplicate #SBATCH directives** — scripts now contain only commands; `#SBATCH` directives are generated from form field values
- **Preview duplicate shebang** — `#!/bin/bash` no longer appears twice when script body includes it
- **Preview sbatch directive accuracy** — corrected `--no-kill` (was `--kill-on-invalid-dep`), `--tres-per-socket` (was missing `#SBATCH` prefix), `--nodes` range format, and `--mincpus` duplication
- **Optional dropdown defaults** — account and QoS dropdowns now default to empty ("not set") instead of silently selecting the first cluster value

### Removed

- **Deprecated `OLDShowJobTemplateSelector`** — removed along with `addTemplatesToList`, `setupTemplateSelectorInput`, `showJobSubmissionFormWithTemplate`, `extractJobSubmissionFromForm`, and `saveFormAsTemplate` helpers

## [0.6.3] - 2026-03-14

### Performance

- **Eliminate N+1 API calls in Partitions view** (#107)
  - Replace per-partition `calculateAllocatedCPUs()` (called on UI draw thread) and `calculateQueueInfo()` with a single bulk `Jobs().List()` call and `buildAllQueueInfo()` helper
  - Reduces API calls from 2N+1 to 2 per refresh cycle (N = number of partitions)
  - Fixes multi-second UI freezes on clusters with 20k+ jobs

- **Lazy view initialization and background fetch** (#106)
  - Views are only initialized and fetched when first focused, not all at startup
  - Background data fetching with DAO-level caching to reduce redundant API calls
  - Only the active view refreshes on timer; inactive views skip refresh cycles

- **Async UI-thread API calls** (#107)
  - Move 11 blocking API calls off the tview draw thread into goroutines with `QueueUpdateDraw`
  - Affected: job cancel/hold/release/requeue/details, node details, partition details/analytics, job dependencies, job templates
  - UI remains responsive during all API operations

### Fixed

- **Per-user temp file paths** (#103)
  - Use per-user paths instead of shared `/tmp` files to prevent multi-user conflicts

- **Install script serving** (#104)
  - Serve install script directly at `get.s9s.dev` root

- **Mock mode validation** (#105)
  - Simplify mock mode validation logic

### Changed

- **Header bar** (#108)
  - Remove clock display from header (added no value, wasted space)
  - Add navigation hints (`Tab:Switch Views  Enter:Details  ?:Help`)

## [0.6.2] - 2026-03-10

### Fixed

- **Setup Wizard Config Generation** (#99)
  - Replace `yaml.Marshal` with template-based rendering to produce valid camelCase YAML keys (`defaultCluster`, `apiVersion`, not `defaultcluster`, `apiversion`)
  - Config structs only have `mapstructure` tags, so `yaml.Marshal` lowercased all keys and s9s failed to load the generated config

- **Setup Wizard Token Discovery** (#99)
  - Generated config now writes `enableEndpoint: false` instead of `enabled: false`, so `scontrol token` and `SLURM_JWT` fallback still works when no token is entered during setup

- **Lint Fixes** (#97)
  - Use `http.NoBody` instead of `nil` for HTTP request bodies (gocritic)

### Changed

- **Setup Wizard Streamlined** (#95, #99)
  - Remove wizard steps that collected data but never wrote to config (name, organization, auth method details, storage preferences, caching, logging, plugins)
  - Wizard now asks only: endpoint, cluster name, and optional JWT token
  - Add connection test to slurmrestd during setup (#95)
  - Update docs to lead with zero-configuration auto-discovery (DNS SRV, `scontrol ping`, `scontrol token`) — s9s works on any SLURM node without running setup

- **Code Formatting** (#98)
  - Run `gofmt` across entire codebase

## [0.6.1] - 2026-03-09

### Fixed

- **Node Resource Metrics** (#92)
  - Replace hardcoded 50% CPU/memory estimates with real SLURM API fields (`AllocCPUs`, `AllocMemory`, `FreeMem`, `CPULoad`)
  - Nodes now display actual allocation and real OS load average instead of fabricated values
  - CPU Load label corrected to "OS 1-min load avg" (previously misleadingly said "1-minute load average" while showing allocation percentage)

- **Cluster Metrics in Dashboard, Performance, and Health Views** (#92)
  - Enrich `GetStats()` with node-level data for memory and CPU usage
  - Memory usage now shows real allocation percentage (was displaying `-1.0%`)
  - CPU usage falls back to node-level aggregation when Stats API returns 0

- **Negative Job Times for PENDING Jobs** (#92)
  - SLURM sets `StartTime` on pending jobs to a future estimated time, causing `time.Since()` to produce negative durations (e.g. `-13:-39:-58`)
  - Elapsed time is now only computed for RUNNING jobs; PENDING jobs show an empty Time column

### Removed

- Fake hardcoded "Job Throughput (24h)" chart from Dashboard — was purely decorative with no real data

## [0.6.0] - 2026-03-08

### Added

- **Cluster Switcher** (#89)
  - Show the active cluster name in the header when multiple clusters are configured
  - Press `Ctrl+K` to open an interactive cluster picker modal
  - Switch between configured clusters at runtime without restarting

- **`--cluster` Flag and Config Rename** (#87)
  - Add `--cluster` flag to select a cluster context from the command line
  - Fix `--config` flag being ignored (always loaded default config path)
  - **Breaking**: Rename config keys `contexts` → `clusters` and `currentContext` → `defaultCluster`
  - Upgrade slurm-client to v0.3.3 (fixes empty Nodes/Partitions/Reservations/Users views on v0.0.41 API)

- **Table Export for All Views** (#85)
  - Press `e` in any view to export the current data to a file
  - Supported views: Jobs, Nodes, Partitions, Reservations, QoS, Accounts, Users
  - Supported formats: CSV, JSON, Text (ASCII table), Markdown, HTML
  - Export dialog shows record count, format picker, and configurable output path
  - Files are written to `~/slurm_exports/` by default with timestamped filenames (e.g. `jobs_20260218_143022.csv`)
  - JSON exports include a structured envelope with `title`, `exported_at`, `total`, and `records` keyed by column name
  - Export paths are validated to prevent writes outside the user's home directory

- **VHS Integration Tests** (#88)
  - VHS-based regression testing against 4 real Slurm clusters (24.05, 24.11, 25.05, 25.11)
  - Normalized ASCII golden file comparison for structural/behavioral change detection
  - Cluster seeding, JWT token generation, and output normalization scripts
  - `scripts/dev-clusters.sh` for generating multi-cluster dev configs
  - Makefile targets: `test-vhs`, `test-vhs-update`, `test-vhs-{2405,2411,2505,2511}`

### Changed

- Config terminology: `contexts` → `clusters`, `currentContext` → `defaultCluster` for consistency

### Fixed

- `--config` flag was completely ignored — always loaded `~/.s9s/config.yaml` (#87)
- Job submission and slurm-client v0.3.1 upgrade (#86)
- Local-first file reading with SSH fallback for job output (#86)
- v0.0.41 API converters returning empty data for Nodes, Partitions, Reservations, and Users (slurm-client v0.3.3)

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

[Unreleased]: https://github.com/jontk/s9s/compare/v0.7.0...HEAD
[0.7.0]: https://github.com/jontk/s9s/compare/v0.6.3...v0.7.0
[0.6.3]: https://github.com/jontk/s9s/compare/v0.6.2...v0.6.3
[0.6.2]: https://github.com/jontk/s9s/compare/v0.6.1...v0.6.2
[0.6.1]: https://github.com/jontk/s9s/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/jontk/s9s/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/jontk/s9s/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/jontk/s9s/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/jontk/s9s/compare/v0.1.0...v0.3.0
[0.1.0]: https://github.com/jontk/s9s/releases/tag/v0.1.0
