# API Documentation

**s9s is a terminal user interface (TUI) application and does not provide a REST API.**

## What is s9s?

s9s is a client-side TUI application that runs in your terminal to monitor and manage SLURM clusters. It connects directly to SLURM using SLURM's native command-line tools (squeue, scontrol, sacct, etc.) and the SLURM client library.

s9s does NOT:
- Run as a server or daemon
- Provide any REST API endpoints
- Accept network connections
- Offer programmatic API access

## Programmatic Access to SLURM

If you need programmatic or REST API access to SLURM clusters, use one of these official SLURM solutions:

### Slurmrestd (Recommended)
SLURM's official REST API daemon provides comprehensive HTTP API access to SLURM functionality.

- **Documentation**: [https://slurm.schedmd.com/rest_api.html](https://slurm.schedmd.com/rest_api.html)
- **Features**: Full REST API for jobs, nodes, partitions, reservations, and more
- **Authentication**: Support for JWT, user/token authentication
- **API Specification**: OpenAPI/Swagger documentation available

### SLURM Command-Line Tools
Use SLURM's CLI tools directly in scripts:
- `squeue` - View job queue
- `scontrol` - Administrative tool for viewing/modifying cluster state
- `sacct` - Job accounting data
- `sbatch` - Submit batch jobs
- `scancel` - Cancel jobs

### PySlurm
Python bindings for SLURM's C API:
- **GitHub**: [https://github.com/PySlurm/pyslurm](https://github.com/PySlurm/pyslurm)
- **Features**: Direct Python interface to SLURM functionality

### slurm-client (Go)
Go client library for SLURM REST API:
- **GitHub**: [https://github.com/jontk/slurm-client](https://github.com/jontk/slurm-client)
- **Features**: Native Go SDK for slurmrestd API with comprehensive type safety
- **Installation**: `go get github.com/jontk/slurm-client@latest`
- **Use Case**: Ideal for building Go applications that need to interact with SLURM clusters

## Using s9s

For information on using the s9s TUI application, see:
- [Views Guide](../guides/views.md) - Overview of all TUI views
- [Configuration](./configuration.md) - Configuration options
- [Keyboard Shortcuts](./keyboard-shortcuts.md) - Navigation and commands
