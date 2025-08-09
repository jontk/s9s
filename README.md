# s9s - Terminal UI for SLURM

<p align="center">
  <img src="https://s9s.dev/logo.png" alt="s9s logo" width="200">
</p>

<p align="center">
  <a href="https://github.com/jontk/s9s/releases"><img src="https://img.shields.io/github/release/jontk/s9s.svg" alt="Latest Release"></a>
  <a href="https://github.com/jontk/s9s/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License"></a>
  <a href="https://goreportcard.com/report/github.com/jontk/s9s"><img src="https://goreportcard.com/badge/github.com/jontk/s9s" alt="Go Report Card"></a>
  <a href="https://s9s.dev/docs"><img src="https://img.shields.io/badge/docs-s9s.dev-brightgreen.svg" alt="Documentation"></a>
</p>

s9s provides a terminal interface for managing SLURM clusters, inspired by the popular [k9s](https://k9scli.io/) Kubernetes CLI. It allows HPC administrators and users to monitor and manage jobs, nodes, and cluster resources efficiently from the terminal.

## 📚 Documentation

- **User Documentation**: [https://s9s.dev/docs](https://s9s.dev/docs)
- **Plugin Development**: [docs/PLUGIN_DEVELOPMENT.md](docs/PLUGIN_DEVELOPMENT.md)
- **Performance Analysis**: [docs/PERFORMANCE_ANALYSIS.md](docs/PERFORMANCE_ANALYSIS.md)  
- **Streaming Guide**: [docs/STREAMING_GUIDE.md](docs/STREAMING_GUIDE.md)
- **API Reference**: [docs/API.md](docs/API.md)
- **Architecture**: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- **Development Guide**: [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)

## ✨ Features

- **Real-time Monitoring**: Live updates of jobs, nodes, and cluster status
- **Multi-View Interface**: Dedicated views for jobs, nodes, partitions, users, QoS, and more
- **Job Management**: Submit, cancel, hold, release, and monitor jobs
- **Batch Operations**: Perform actions on multiple jobs simultaneously
- **Advanced Filtering**: Powerful search and filter capabilities
- **SSH Integration**: Connect directly to compute nodes
- **Export Capabilities**: Export data in CSV, JSON, Markdown, HTML formats
- **Real-time Job Log Streaming**: Live job output monitoring with filtering
- **Plugin System**: Extensible architecture for custom functionality
- **Performance Analysis**: Built-in profiling and optimization tools
- **Vim-like Navigation**: Familiar keyboard shortcuts for power users
- **Mock Mode**: Built-in SLURM simulator for development and testing

## 🚀 Quick Start

### Prerequisites

- Go 1.19 or higher
- Access to a SLURM cluster (or use mock mode)
- Terminal with 256 color support

### Installation

#### Quick Install (Recommended)

```bash
curl -sSL https://get.s9s.dev | bash
```

#### Using Go Install

```bash
go install github.com/jontk/s9s/cmd/s9s@latest
```

#### From Source

```bash
git clone https://github.com/jontk/s9s.git
cd s9s
go build -o s9s cmd/s9s/main.go
sudo mv s9s /usr/local/bin/
```

### Basic Usage

```bash
# Connect to your SLURM cluster
s9s

# Use mock mode for testing (no SLURM required)
s9s --mock

# Connect to a specific cluster
s9s --cluster production

# Enable debug logging
s9s --debug
```

### Configuration

#### Quick Setup (Recommended)

Run the interactive setup wizard for first-time configuration:

```bash
s9s setup
```

The wizard will guide you through:
- 🏢 Cluster connection settings
- 🔐 Authentication configuration  
- 🔒 Secure credential storage
- ⚡ Performance optimization

#### Manual Configuration

s9s looks for configuration in the following order:
1. `~/.s9s/config.yaml`
2. `~/.config/s9s/config.yaml`
3. Environment variables
4. Command-line flags

Example configuration:

```yaml
# ~/.s9s/config.yaml
clusters:
  production:
    url: https://slurm-api.example.com
    token: ${SLURM_TOKEN}
    default: true
  
  development:
    url: https://slurm-dev.example.com
    auth:
      username: ${SLURM_USER}
      password: ${SLURM_PASS}

preferences:
  theme: dark
  refreshInterval: 5s
  defaultView: jobs
```

## 🎮 Key Bindings

### Global

| Key | Action |
|-----|--------|
| `?` | Show help |
| `q` | Quit |
| `:` | Command mode |
| `/` | Search |
| `Tab` | Switch view |
| `Ctrl+r` | Force refresh |

### Jobs View

| Key | Action |
|-----|--------|
| `c` | Cancel job |
| `h` | Hold job |
| `r` | Release job |
| `d` | Show job details |
| `o` | View job output |
| `s` | Submit new job |
| `b` | Batch operations |
| `m` | Toggle auto-refresh |

### Nodes View

| Key | Action |
|-----|--------|
| `d` | Drain node |
| `r` | Resume node |
| `s` | SSH to node |
| `i` | Node info |

## 🏗️ Architecture

s9s follows a modular architecture with clear separation of concerns:

```
cmd/s9s/          # Main application entry point
internal/
  ├── app/        # Application lifecycle management
  ├── dao/        # Data Access Objects (SLURM client abstraction)
  ├── views/      # Terminal UI views
  ├── config/     # Configuration management
  ├── ui/         # UI components and utilities
  └── slurm/      # SLURM client implementation
pkg/
  └── slurm/      # Mock SLURM implementation
```

For detailed architecture information, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## 🔧 Development

### Setup Development Environment

```bash
# Clone the repository
git clone https://github.com/jontk/s9s.git
cd s9s

# Install dependencies
go mod download

# Run tests
go test ./...

# Run with mock data
go run cmd/s9s/main.go --mock

# Build binary
go build -o s9s cmd/s9s/main.go
```

### Running Tests

```bash
# Unit tests
go test ./...

# Integration tests
go test -tags=integration ./test/integration

# Benchmarks
go test -bench=. ./test/performance

# Coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Debug Mode

Enable debug logging to troubleshoot issues:

```bash
s9s --debug

# Or set environment variable
export S9S_DEBUG=true
s9s
```

Debug logs are written to `~/.s9s/debug.log`.

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Quick Contribution Guide

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`go test ./...`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by [k9s](https://k9scli.io/) - Kubernetes CLI
- Built with [tview](https://github.com/rivo/tview) - Terminal UI framework
- [SLURM](https://slurm.schedmd.com/) - HPC workload manager

## 🔗 Links

- [Website](https://s9s.dev)
- [Documentation](https://s9s.dev/docs)
- [Issue Tracker](https://github.com/jontk/s9s/issues)
- [Discord Community](https://discord.gg/s9s)

---

<p align="center">Made with ❤️ for the HPC community</p>