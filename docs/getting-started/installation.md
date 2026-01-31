# Installation Guide

s9s can be installed using several methods. Choose the one that best fits your environment.

## System Requirements

- **Operating System**: Linux, macOS, or Windows
- **Go Version**: 1.19 or higher (for building from source)
- **Terminal**: 256 color support recommended
- **SLURM Access**: Connection to a SLURM cluster (optional - mock mode available)

## Installation Methods

### 1. Quick Install (Recommended)

The easiest way to install s9s is using our installation script:

```bash
curl -sSL https://get.s9s.dev | bash
```

This script will:
- Detect your operating system
- Download the appropriate binary
- Install s9s to `/usr/local/bin`
- Set up initial configuration

### 2. Binary Download

Download pre-built binaries from our [releases page](https://github.com/jontk/s9s/releases):

```bash
# Linux (AMD64)
wget https://github.com/jontk/s9s/releases/latest/download/s9s-linux-amd64
chmod +x s9s-linux-amd64
sudo mv s9s-linux-amd64 /usr/local/bin/s9s

# Linux (ARM64)
wget https://github.com/jontk/s9s/releases/latest/download/s9s-linux-arm64
chmod +x s9s-linux-arm64
sudo mv s9s-linux-arm64 /usr/local/bin/s9s

# macOS (Apple Silicon)
wget https://github.com/jontk/s9s/releases/latest/download/s9s-darwin-arm64
chmod +x s9s-darwin-arm64
sudo mv s9s-darwin-arm64 /usr/local/bin/s9s

# macOS (Intel)
wget https://github.com/jontk/s9s/releases/latest/download/s9s-darwin-amd64
chmod +x s9s-darwin-amd64
sudo mv s9s-darwin-amd64 /usr/local/bin/s9s
```

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

### 5. Package Managers

#### Homebrew (macOS/Linux)

```bash
brew tap jontk/s9s
brew install s9s
```

#### Snap (Linux)

```bash
sudo snap install s9s
```

#### AUR (Arch Linux)

```bash
yay -S s9s
# or
paru -S s9s
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

Run the setup wizard:

```bash
s9s setup
```

Or create a config file manually:

```bash
mkdir -p ~/.s9s
cat > ~/.s9s/config.yaml << EOF
clusters:
  default:
    url: https://your-slurm-api.example.com
    auth:
      method: token
      token: \${SLURM_TOKEN}
preferences:
  theme: dark
  defaultView: dashboard
  refreshInterval: 30s
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

## Docker Installation

For containerized environments:

```bash
# Pull the image
docker pull ghcr.io/jontk/s9s:latest

# Run interactively
docker run -it --rm \
  -v ~/.s9s:/root/.s9s \
  -e SLURM_TOKEN=$SLURM_TOKEN \
  ghcr.io/jontk/s9s:latest

# Create an alias for convenience
alias s9s='docker run -it --rm -v ~/.s9s:/root/.s9s -e SLURM_TOKEN=$SLURM_TOKEN ghcr.io/jontk/s9s:latest'
```

### Docker Compose

```yaml
version: '3.8'
services:
  s9s:
    image: ghcr.io/jontk/s9s:latest
    environment:
      - SLURM_TOKEN=${SLURM_TOKEN}
    volumes:
      - ~/.s9s:/root/.s9s
    stdin_open: true
    tty: true
```

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
sudo rm /usr/local/bin/s9s

# Remove configuration (optional)
rm -rf ~/.s9s
rm -rf ~/.config/s9s

# Remove cache (optional)
rm -rf ~/.cache/s9s
```

## Troubleshooting Installation

### Permission Denied

If you get permission errors:

```bash
# Use sudo for system-wide installation
sudo mv s9s /usr/local/bin/

# Or install to user directory
mkdir -p ~/.local/bin
mv s9s ~/.local/bin/
echo 'export PATH=$PATH:~/.local/bin' >> ~/.bashrc
source ~/.bashrc
```

### Command Not Found

If `s9s` is not found after installation:

1. Check if the binary exists:
   ```bash
   ls -la /usr/local/bin/s9s
   ```

2. Ensure the directory is in your PATH:
   ```bash
   echo $PATH
   ```

3. Add to PATH if needed:
   ```bash
   export PATH=$PATH:/usr/local/bin
   # Add to ~/.bashrc or ~/.zshrc for persistence
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
curl -sSL https://get.s9s.dev | bash

# If installed via binary download
wget https://github.com/jontk/s9s/releases/latest/download/s9s-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m)
chmod +x s9s-*
sudo mv s9s-* /usr/local/bin/s9s

# If installed via Homebrew
brew upgrade s9s

# If installed via Go
go install github.com/jontk/s9s/cmd/s9s@latest
```

## Next Steps

- Read the [Quick Start Guide](quickstart.md) to learn basic usage
- Configure s9s for your environment with our [Configuration Guide](configuration.md)
- Learn keyboard shortcuts in the [Keyboard Shortcuts Guide](../user-guide/keyboard-shortcuts.md)
- Try [Mock Mode](../guides/mock-mode.md) to explore without SLURM
