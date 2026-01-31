# Changelog

This page provides an overview of notable changes to the s9s project. For a complete and detailed changelog, please see the [full changelog](../CHANGELOG.md).

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Recent Changes

### Version 0.3.0 (2026-01-30)

Key improvements and new features:

- **Job submission working directory default**: Working directory now defaults to the current directory where s9s is started
- **Vim-style view navigation**: `h` and `l` keys for switching between views
- **Global search cross-view navigation**: Search results navigate to the correct view and focus on the selected item
- **Partition filter syntax**: Use `p:partition_name` to filter by partition
- **Users view admin filter**: Press `a` to toggle showing only administrators
- **Status bar feedback**: Users view shows status messages when toggling filters
- **Username resolution**: Jobs display actual usernames instead of numeric UIDs
- Centralized version management and automated release processes
- Comprehensive release notes following Keep a Changelog format

Notable fixes:
- Fixed drain operation to actually drain nodes
- Fixed resume operation modal issues
- Fixed node state truncation (shows full state like `IDLE+DRAIN`)
- Fixed filter input hijacking by global shortcuts
- Fixed search deadlocks in event handlers
- Environment variables now properly override config file values
- Case-insensitive admin level comparison in Users view

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

- [v0.3.0](../CHANGELOG.md#030---2026-01-30) - Latest release
- [v0.1.0](../CHANGELOG.md#010---2026-01-21) - Initial release
- [Unreleased](../CHANGELOG.md#unreleased) - Upcoming changes

## Semantic Versioning

This project follows [Semantic Versioning](https://semver.org/) (MAJOR.MINOR.PATCH):
- **MAJOR**: Incompatible API changes
- **MINOR**: Added functionality in a backwards-compatible manner
- **PATCH**: Backwards-compatible bug fixes

## Contributing

If you've found a bug or have a feature request, please check the changelog to see if it's already been addressed or planned. For more information about contributing, refer to the project's contributing guidelines.
