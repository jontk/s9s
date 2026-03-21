# Installation Guide

s9s can be installed using several methods. Choose the one that best fits your environment.

## System Requirements

- **Operating System**: Linux, macOS, or Windows
- **Go Version**: 1.24 or higher (for building from source)
- **Terminal**: 256 color support recommended
- **SLURM REST API**: A running `slurmrestd` instance — see [Troubleshooting](../guides/troubleshooting.md#cannot-connect-to-slurm-cluster) if connection fails ([mock mode](../guides/mock-mode.md) available for testing without SLURM)

## Installation Methods

### 1. Quick Install (Recommended)

The easiest way to install s9s is using our installation script:

```bash
curl -sSL https://get.s9s.dev | bash
```

This script will:
- Detect your operating system
- Download the appropriate binary
- Install s9s to `~/.local/bin`
- Set up initial configuration

### 2. Binary Download

Download pre-built binaries from our [releases page](https://github.com/jontk/s9s/releases):

```bash
# Linux (x86_64)
curl -LO https://github.com/jontk/s9s/releases/latest/download/s9s_0.7.1_Linux_x86_64.tar.gz
tar -xzf s9s_0.7.1_Linux_x86_64.tar.gz
mkdir -p ~/.local/bin
mv s9s ~/.local/bin/

# Linux (ARM64)
curl -LO https://github.com/jontk/s9s/releases/latest/download/s9s_0.7.1_Linux_arm64.tar.gz
tar -xzf s9s_0.7.1_Linux_arm64.tar.gz
mkdir -p ~/.local/bin
mv s9s ~/.local/bin/

# macOS (Apple Silicon)
curl -LO https://github.com/jontk/s9s/releases/latest/download/s9s_0.7.1_Darwin_arm64.tar.gz
tar -xzf s9s_0.7.1_Darwin_arm64.tar.gz
mkdir -p ~/.local/bin
mv s9s ~/.local/bin/

# macOS (Intel)
curl -LO https://github.com/jontk/s9s/releases/latest/download/s9s_0.7.1_Darwin_x86_64.tar.gz
tar -xzf s9s_0.7.1_Darwin_x86_64.tar.gz
mkdir -p ~/.local/bin
mv s9s ~/.local/bin/
```

> **Note:** Replace `0.7.1` with the latest version from the [releases page](https://github.com/jontk/s9s/releases).

### 3. Using Go Install

If you have Go installed:

```bash
go install github.com/jontk/s9s/cmd/s9s@latest
```

Make sure `$GOPATH/bin` is in your PATH:

```bash
export PATH=$PATH:$GOPATH/bin
```

### 4. Building from Source

For the latest development version:

```bash
# Clone the repository
git clone https://github.com/jontk/s9s.git
cd s9s

# Build the binary
make build

# Install to GOPATH/bin
make install

# Or install to user directory
mkdir -p ~/.local/bin
cp ./build/s9s ~/.local/bin/
export PATH=$PATH:~/.local/bin
```

## Post-Installation Setup

### 1. Verify Installation

```bash
s9s --version
```

You should see output like:
```
S9S - SLURM Terminal UI version 0.7.1
```

### 2. Initial Configuration

On any SLURM node, **no configuration is needed** — s9s auto-discovers the slurmrestd endpoint using DNS SRV records and `scontrol ping` (which finds controllers even on remote hosts), and authenticates via `scontrol token` or the `SLURM_JWT` environment variable.

If auto-discovery doesn't find your cluster, use the setup wizard:

```bash
s9s setup
```

Or create a config file manually:

```bash
mkdir -p ~/.s9s
cat > ~/.s9s/config.yaml << EOF
defaultCluster: production

clusters:
  - name: production
    cluster:
      endpoint: https://your-slurm-api.example.com:6820
      token: "your-jwt-token"
      timeout: 30s
EOF
```

See [Configuration Guide](configuration.md) for complete configuration options.

### 3. Test Connection

```bash
# On a SLURM node (auto-discovers slurmrestd)
s9s

# Or connect manually
export SLURM_REST_URL=https://your-slurm-api.example.com:6820
export SLURM_JWT=your-token
s9s
```

If connection succeeds, you should see the s9s dashboard. See [Mock Mode](../guides/mock-mode.md) for testing without a SLURM cluster.

## Platform-Specific Notes

### Linux

s9s works on most Linux distributions. Ensure you have:
- glibc 2.31 or newer (for pre-built binaries)
- Terminal with 256 color support

### macOS

For macOS users:
- macOS 11 (Big Sur) or newer recommended
- Terminal.app, iTerm2, or similar terminal emulator
- Rosetta 2 (for Intel binaries on Apple Silicon)

### Windows

Windows users can run s9s via:
- **WSL2** (recommended) - Install in Linux subsystem
- **Git Bash** - May have limited functionality
- **Windows Terminal** - Best terminal experience

## Uninstallation

To remove s9s:

```bash
# Remove binary
rm ~/.local/bin/s9s

# Remove configuration (optional)
rm -rf ~/.s9s

# Remove cache (optional)
rm -rf ~/.s9s/cache
```

## Troubleshooting Installation

### Permission Denied

If you get permission errors, install to your user directory:

```bash
# Install to user directory (default)
mkdir -p ~/.local/bin
mv s9s ~/.local/bin/
echo 'export PATH=$PATH:~/.local/bin' >> ~/.bashrc
source ~/.bashrc
```

### Command Not Found

If `s9s` is not found after installation:

1. Check if the binary exists:
   ```bash
   ls -la ~/.local/bin/s9s
   ```

2. Ensure the directory is in your PATH:
   ```bash
   echo $PATH | grep -q "$HOME/.local/bin" && echo "In PATH" || echo "Not in PATH"
   ```

3. Add to PATH if needed:
   ```bash
   export PATH=$PATH:~/.local/bin
   # Add to ~/.bashrc or ~/.zshrc for persistence
   echo 'export PATH=$PATH:~/.local/bin' >> ~/.bashrc
   ```

### SSL/TLS Issues

If you encounter SSL certificate errors:

Set `insecure: true` in the cluster config for self-signed certificates:

```yaml
clusters:
  - name: default
    cluster:
      endpoint: https://slurm.example.com:6820
      insecure: true
```

### Connection Refused

If you cannot connect to SLURM:

1. **Check SLURM API access**:
   ```bash
   curl https://your-slurm-api.example.com/slurm/v0.0.43/jobs
   ```

2. **Verify authentication**:
   - Ensure SLURM_JWT is set
   - Check token is valid
   - Verify API URL is correct

3. **Try mock mode** to rule out s9s issues (see [Mock Mode Guide](../guides/mock-mode.md)):
   ```bash
   export S9S_ENABLE_MOCK=1
   s9s --mock
   ```

### Binary Won't Execute

On Linux, if you get "cannot execute binary file":

```bash
# Check if you downloaded the correct architecture
uname -m

# For x86_64/AMD64
curl -LO https://github.com/jontk/s9s/releases/latest/download/s9s_0.7.1_Linux_x86_64.tar.gz

# For ARM64/aarch64
curl -LO https://github.com/jontk/s9s/releases/latest/download/s9s_0.7.1_Linux_arm64.tar.gz
```

## Upgrading

To upgrade to the latest version:

```bash
# If installed via script
curl -sSL https://get.s9s.dev | bash

# If installed via binary download (replace version and arch as needed)
curl -LO https://github.com/jontk/s9s/releases/latest/download/s9s_0.7.1_Linux_x86_64.tar.gz
tar -xzf s9s_0.7.1_Linux_x86_64.tar.gz
mkdir -p ~/.local/bin
mv s9s ~/.local/bin/

# If installed via Go
go install github.com/jontk/s9s/cmd/s9s@latest
```

## Next Steps

- Read the [Quick Start Guide](quickstart.md) to learn basic usage
- Configure s9s for your environment with our [Configuration Guide](configuration.md)
- Learn keyboard shortcuts in the [Keyboard Shortcuts Guide](../user-guide/keyboard-shortcuts.md)
- Try [Mock Mode](../guides/mock-mode.md) to explore without SLURM
