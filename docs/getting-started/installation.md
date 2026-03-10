# Installation Guide

s9s can be installed using several methods. Choose the one that best fits your environment.

## System Requirements

- **Operating System**: Linux, macOS, or Windows
- **Go Version**: 1.19 or higher (for building from source)
- **Terminal**: 256 color support recommended
- **SLURM REST API**: A running `slurmrestd` instance (optional - mock mode available)

### slurmrestd (SLURM REST API)

s9s connects to SLURM through **slurmrestd**, the SLURM REST API daemon. This is a separate service from `slurmctld` (the controller) and `slurmdbd` (the accounting database). It typically runs on port **6820**.

If slurmrestd is not already running on your cluster, start it:

```bash
# Start slurmrestd (as root or SlurmUser)
slurmrestd 0.0.0.0:6820

# Or with systemd (if configured)
sudo systemctl start slurmrestd
```

Verify it's running:

```bash
# Check the port is listening
ss -tlnp | grep 6820

# Test the API
curl http://localhost:6820/slurm/v0.0.43/ping
```

> **Note**: Having `slurmctld` and `slurmdbd` running is **not sufficient** - s9s specifically requires `slurmrestd` for its REST API.

## Installation Methods

### 1. Quick Install (Recommended)

The easiest way to install s9s is using our installation script:

```bash
curl -sSL https://s9s.dev/install.sh | bash
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
curl -LO https://github.com/jontk/s9s/releases/latest/download/s9s_0.6.2_Linux_x86_64.tar.gz
tar -xzf s9s_0.6.2_Linux_x86_64.tar.gz
mkdir -p ~/.local/bin
mv s9s ~/.local/bin/

# Linux (ARM64)
curl -LO https://github.com/jontk/s9s/releases/latest/download/s9s_0.6.2_Linux_arm64.tar.gz
tar -xzf s9s_0.6.2_Linux_arm64.tar.gz
mkdir -p ~/.local/bin
mv s9s ~/.local/bin/

# macOS (Apple Silicon)
curl -LO https://github.com/jontk/s9s/releases/latest/download/s9s_0.6.2_Darwin_arm64.tar.gz
tar -xzf s9s_0.6.2_Darwin_arm64.tar.gz
mkdir -p ~/.local/bin
mv s9s ~/.local/bin/

# macOS (Intel)
curl -LO https://github.com/jontk/s9s/releases/latest/download/s9s_0.6.2_Darwin_x86_64.tar.gz
tar -xzf s9s_0.6.2_Darwin_x86_64.tar.gz
mkdir -p ~/.local/bin
mv s9s ~/.local/bin/
```

> **Note:** Replace `0.6.2` with the latest version from the [releases page](https://github.com/jontk/s9s/releases).

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

# Install to system path
sudo make install

# Or install to user directory
mkdir -p ~/.local/bin
cp ./bin/s9s ~/.local/bin/
export PATH=$PATH:~/.local/bin
```

## Post-Installation Setup

### 1. Verify Installation

```bash
s9s --version
```

You should see output like:
```
s9s version 0.3.0
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
# Test with your SLURM cluster
s9s

# Or test with mock mode (no SLURM required)
s9s --mock
```

If connection succeeds, you should see the s9s dashboard.

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
rm -rf ~/.config/s9s

# Remove cache (optional)
rm -rf ~/.cache/s9s
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

```bash
# For self-signed certificates
export S9S_INSECURE_TLS=true

# Or add to config
echo "insecureTLS: true" >> ~/.s9s/config.yaml
```

### Connection Refused

If you cannot connect to SLURM:

1. **Check SLURM API access**:
   ```bash
   curl https://your-slurm-api.example.com/slurm/v0.0.40/jobs
   ```

2. **Verify authentication**:
   - Ensure SLURM_TOKEN is set
   - Check token is valid
   - Verify API URL is correct

3. **Try mock mode** to rule out s9s issues:
   ```bash
   s9s --mock
   ```

### Binary Won't Execute

On Linux, if you get "cannot execute binary file":

```bash
# Check if you downloaded the correct architecture
uname -m

# For x86_64/AMD64
wget https://github.com/jontk/s9s/releases/latest/download/s9s-linux-amd64

# For ARM64/aarch64
wget https://github.com/jontk/s9s/releases/latest/download/s9s-linux-arm64
```

## Upgrading

To upgrade to the latest version:

```bash
# If installed via script
curl -sSL https://s9s.dev/install.sh | bash

# If installed via binary download
wget https://github.com/jontk/s9s/releases/latest/download/s9s-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m)
chmod +x s9s-*
mkdir -p ~/.local/bin
mv s9s-* ~/.local/bin/s9s

# If installed via Go
go install github.com/jontk/s9s/cmd/s9s@latest
```

## Next Steps

- Read the [Quick Start Guide](quickstart.md) to learn basic usage
- Configure s9s for your environment with our [Configuration Guide](configuration.md)
- Learn keyboard shortcuts in the [Keyboard Shortcuts Guide](../user-guide/keyboard-shortcuts.md)
- Try [Mock Mode](../guides/mock-mode.md) to explore without SLURM
