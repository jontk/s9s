# s9s Documentation

Welcome to s9s documentation! s9s is a powerful terminal UI for SLURM that brings the k9s experience to HPC cluster management.

![Overview Demo](/assets/demos/overview.gif)

*s9s in action - real-time cluster monitoring and management*

## What is s9s?

s9s is a modern, terminal-based interface for SLURM workload managers. It provides:

- **Real-time monitoring** - Live updates of jobs, nodes, and cluster status
- **Intuitive navigation** - Keyboard-driven interface inspired by k9s
- **Powerful filtering** - Advanced search and filter capabilities
- **Batch operations** - Manage multiple resources efficiently
- **SSH integration** - Direct node access from the UI
- **Plugin system** - Extend functionality with custom plugins

## Getting Started

### Quick Start

```bash
# Install s9s
curl -sSL https://get.s9s.dev | bash

# Launch s9s
s9s

# Or try mock mode (no SLURM required)
s9s --mock
```

### Documentation Sections

#### 📚 Getting Started
- [Installation](getting-started/installation.md) - Install s9s on your system
- [Quick Start](getting-started/quickstart.md) - Get running in minutes
- [Configuration](getting-started/configuration.md) - Configure for your environment

#### 👤 User Guide
- [Navigation](user-guide/navigation.md) - Master keyboard shortcuts
- [Views Overview](user-guide/views/index.md) - Understand all views
  - [Dashboard](user-guide/views/dashboard.md) - Cluster overview
  - [Jobs](user-guide/views/jobs.md) - Job management
  - [Nodes](user-guide/views/nodes.md) - Node operations
  - [Partitions](user-guide/views/partitions.md) - Partition monitoring
  - [Users](user-guide/views/users.md) - User accounts
  - [Accounts](user-guide/views/accounts.md) - Account hierarchy
  - [QoS](user-guide/views/qos.md) - Quality of Service policies
  - [Reservations](user-guide/views/reservations.md) - Resource reservations
  - [Health](user-guide/views/health.md) - Cluster health monitoring
- [Keyboard Shortcuts](user-guide/keyboard-shortcuts.md) - Complete shortcuts reference
- [Job Management](user-guide/job-management.md) - Submit and manage jobs
- [Node Operations](user-guide/node-operations.md) - Node administration
- [Batch Operations](user-guide/batch-operations.md) - Multi-resource operations
- [Filtering & Search](user-guide/filtering.md) - Advanced filtering
- [Export & Reporting](user-guide/export.md) - Export data

#### 📖 Guides
- [SSH Integration](guides/ssh-integration.md) - Direct node access
- [Job Streaming](guides/job-streaming.md) - Real-time log streaming
- [Mock Mode](guides/mock-mode.md) - Test without SLURM
- [Troubleshooting](guides/troubleshooting.md) - Common issues

#### 📘 Reference
- [Commands](reference/commands.md) - Command reference
- [Configuration](reference/configuration.md) - Config file reference
- [API](reference/api.md) - REST API documentation

#### 🔌 Plugins
- [Plugin Overview](plugins/overview.md) - Plugin system introduction
- [Plugin Development](plugins/development.md) - Create custom plugins
- [Observability Plugin](plugins/observability.md) - Monitoring and metrics

#### 🛠️ Development
- [Development Setup](development/setup.md) - Set up dev environment
- [Architecture](development/architecture.md) - System architecture
- [Contributing](development/contributing.md) - Contribution guidelines
- [Testing](development/testing.md) - Testing guide
- [Linting](development/linting.md) - Code quality
- [CI/CD](development/ci-cd.md) - Continuous integration

#### 🏢 Enterprise
- [Enterprise Features](enterprise/features.md) - Features for large deployments

#### ℹ️ About
- [Changelog](about/changelog.md) - Version history
- [License](about/license.md) - MIT License

## Quick Links

### Need Help Fast?
- [Quick Start Guide](getting-started/quickstart.md) - Get running quickly
- [Troubleshooting](guides/troubleshooting.md) - Common issues and solutions
- [Keyboard Shortcuts](user-guide/keyboard-shortcuts.md) - All shortcuts

### Popular Topics
- [Installation](getting-started/installation.md) - Installation methods
- [Configuration](getting-started/configuration.md) - Configure s9s
- [Job Management](user-guide/job-management.md) - Submit and monitor jobs
- [SSH Integration](guides/ssh-integration.md) - SSH to nodes
- [Mock Mode](guides/mock-mode.md) - Try without SLURM

### For Developers
- [Plugin Development](plugins/development.md) - Build plugins
- [Architecture](development/architecture.md) - Understanding s9s
- [Contributing](development/contributing.md) - Contribute to s9s
- [API Documentation](reference/api.md) - REST API

## Features Highlights

### Real-Time Monitoring
Monitor your SLURM cluster in real-time with auto-refreshing views and live metrics.

### Keyboard-Driven Interface
Navigate efficiently with vim-style keyboard shortcuts. No mouse required.

### Advanced Filtering
Powerful filtering system with expression-based search across all resource types.

### Batch Operations
Manage multiple jobs, nodes, or resources simultaneously with batch operations.

### SSH Integration
SSH directly to nodes from the UI with session management and multiplexing.

### Plugin System
Extend s9s with custom plugins for monitoring, automation, and integration.

## Support

- **GitHub Issues**: [Report bugs or request features](https://github.com/jontk/s9s/issues)
- **Discussions**: [Community discussions](https://github.com/jontk/s9s/discussions)
- **Contributing**: [Contribution guide](development/contributing.md)

## License

s9s is open source software licensed under the [MIT License](about/license.md).
