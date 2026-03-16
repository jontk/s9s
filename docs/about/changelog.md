# Changelog

This page provides an overview of notable changes to the s9s project. For a complete and detailed changelog, please see the [full changelog](../CHANGELOG.md).

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Recent Changes

### Unreleased

Configurable job submission templates and full SLURM field support:

- **Configurable Templates**: Define custom job templates in config YAML with pre-filled defaults, hidden fields, and restricted dropdown options
- **3-Tier Template Merge**: Built-in, config YAML, and user-saved JSON templates merge with name-based override
- **86 sbatch Fields**: Job submission wizard supports 86 of 117 SLURM fields (was 12) — fields like QoS, GPUs, array, constraints, dependencies now actually reach slurmrestd
- **Smart Field Visibility**: Advanced fields hidden by default, shown automatically when a template sets them
- **CLI Template Management**: `s9s templates list` and `s9s templates export` commands for viewing and exporting templates
- **SubmitRaw Integration**: Bypasses lossy intermediate struct, maps directly to SLURM OpenAPI `JobCreate`
- **QoS Dropdown**: QoS field now fetches from cluster API like partitions and accounts
- **User Defaults**: Account and QoS default to the current SLURM user's defaults
- **Preview Copy**: Ctrl+Y in job script preview copies clean sbatch script to clipboard (OSC 52)
- **Full `--begin` Parsing**: ISO dates, relative times (`now+1hour`), SLURM named times (`midnight`, `fika`, `teatime`)
- **Bug Fixes**: Viper key case normalization, preview directive accuracy, optional dropdown defaults, duplicate shebang/SBATCH removal
- **Removed**: Deprecated `OLDShowJobTemplateSelector` and helpers

### Version 0.6.3 (2026-03-14)

Performance and responsiveness release:

- **Eliminate N+1 API calls**: Partitions view now fetches all jobs in a single API call instead of 2N+1 calls per refresh — fixes multi-second freezes on clusters with 20k+ jobs
- **Lazy view init & background fetch**: Views only initialize when first focused; DAO-level caching reduces redundant API calls; only the active view auto-refreshes
- **Async UI-thread API calls**: 11 blocking API calls (cancel, hold, release, details modals, etc.) moved off the tview draw thread — UI stays responsive during all operations
- **Header improvements**: Remove clock, add navigation key hints (`Tab`, `Enter`, `?`)
- **Per-user temp paths**: Fix multi-user conflicts with shared `/tmp` files
- **Install script fix**: Serve install script directly at `get.s9s.dev` root

### Version 0.6.2 (2026-03-10)

Setup wizard improvements and bug fixes:

- **Setup Wizard Config Fix**: Generated config now uses valid camelCase YAML keys — previously `yaml.Marshal` produced lowercase keys that s9s couldn't load
- **Streamlined Wizard**: Removed questions that didn't affect config; wizard now asks only endpoint, cluster name, and optional JWT token
- **Token Discovery Fix**: Generated config no longer disables token auto-discovery, so `scontrol token` and `SLURM_JWT` fallback works without a configured token
- **Connection Test**: Setup wizard now tests the slurmrestd connection before completing
- **Docs Update**: Lead with zero-configuration auto-discovery (DNS SRV, `scontrol ping`) — s9s works on any SLURM node without running setup

### Version 0.6.1 (2026-03-09)

Bug fix release focusing on metric accuracy:

- **Node Resource Metrics**: Use real SLURM API fields (`AllocCPUs`, `AllocMemory`, `FreeMem`, `CPULoad`) instead of hardcoded 50% estimates
- **Cluster Metrics**: Dashboard, Performance, and Health views now show real memory and CPU usage from node data
- **Job Times**: Fix negative elapsed times for PENDING jobs (SLURM sets StartTime to future estimate)
- Remove fake hardcoded "Job Throughput (24h)" chart from Dashboard

### Version 0.6.0 (2026-03-08)

Key features:

- **Cluster Switcher**: Show active cluster name in header, press `Ctrl+K` to switch clusters at runtime
- **`--cluster` Flag**: Select cluster context from the command line; fix `--config` flag being ignored
- **Table Export**: Press `e` in any view to export data (CSV, JSON, Text, Markdown, HTML)
- **VHS Integration Tests**: Regression testing against real Slurm clusters (24.05, 24.11, 25.11)
- **Breaking**: Config keys renamed `contexts` → `clusters`, `currentContext` → `defaultCluster`

### Version 0.5.0 (2026-02-18)

Key features:

- **Zero-Configuration Auto-Discovery**: Works out-of-the-box on SLURM systems without config files
- **API Version Auto-Detection**: Removed hardcoded API version default
- **Static Binary Builds**: `CGO_ENABLED=0` for cross-distribution compatibility
- **Integration Tests CI**: PR testing with ephemeral k3d cluster and mock SLURM API

### Version 0.4.0 (2026-02-08)

Key features:

- **Performance View**: Real-time cluster-wide metrics (jobs, nodes, CPU%, memory%)
- **Sorting Modal**: Press `S` in any table view for interactive column sorting
- **Command Mode with Arguments**: Vim-style `:cancel`, `:drain`, `:requeue` with Tab completion
- **Reservation Filters**: `a` for active-only, `f` for future-only
- **Job Output File Reading**: Local filesystem and SSH-based remote file access

### Version 0.3.0 (2026-01-30)

Key features:

- **Vim-style view navigation**: `h` and `l` keys for switching between views
- **Global search cross-view navigation**: Search results navigate to the correct view
- **Partition filter syntax**: Use `p:partition_name` to filter by partition
- **Username resolution**: Jobs display actual usernames instead of numeric UIDs
- Centralized version management and automated release processes

### Version 0.1.0 (2026-01-21)

Initial stable release with core features:

- Core TUI Application for SLURM cluster management
- Vim-like navigation and keybindings
- Interactive job and node management
- Multi-cluster support with context switching
- Configuration management and wizard
- CSV, JSON, and Markdown export capabilities
- Observability plugin with Prometheus integration
- Comprehensive test suite and CI/CD

## Viewing the Full Changelog

For a complete list of all changes, features, and fixes across all versions, refer to the [full CHANGELOG.md](../CHANGELOG.md).

## Version History

- [v0.6.3](../CHANGELOG.md#063---2026-03-14) - Latest release
- [v0.6.2](../CHANGELOG.md#062---2026-03-10) - Setup wizard fixes
- [v0.6.1](../CHANGELOG.md#061---2026-03-09) - Node metrics, job times, dashboard fixes
- [v0.6.0](../CHANGELOG.md#060---2026-03-08) - Cluster switcher, export, config rename
- [v0.5.0](../CHANGELOG.md#050---2026-02-18) - Auto-discovery, static builds
- [v0.4.0](../CHANGELOG.md#040---2026-02-08) - Performance view, sorting, command mode
- [v0.3.0](../CHANGELOG.md#030---2026-01-30) - Vim navigation, search, filters
- [v0.1.0](../CHANGELOG.md#010---2026-01-21) - Initial release
- [Unreleased](../CHANGELOG.md#unreleased) - Upcoming changes

## Semantic Versioning

This project follows [Semantic Versioning](https://semver.org/) (MAJOR.MINOR.PATCH):
- **MAJOR**: Incompatible API changes
- **MINOR**: Added functionality in a backwards-compatible manner
- **PATCH**: Backwards-compatible bug fixes

## Contributing

If you've found a bug or have a feature request, please check the changelog to see if it's already been addressed or planned. For more information about contributing, refer to the project's contributing guidelines.
