#!/bin/bash
set -e

# s9s Installation Script
# This script installs the latest version of s9s for your platform

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO="jontk/s9s"
INSTALL_DIR="$HOME/.local/bin"
BINARY_NAME="s9s"
CONFIG_DIR="$HOME/.s9s"

# Helper functions
log() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
    exit 1
}

debug() {
    if [ "${DEBUG:-}" = "1" ]; then
        echo -e "${BLUE}[DEBUG]${NC} $1" >&2
    fi
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Detect OS and architecture
detect_platform() {
    local os
    local arch

    # Detect OS (match GitHub release naming: Linux, Darwin, Windows)
    case "$(uname -s)" in
        Linux*)
            os="Linux"
            ;;
        Darwin*)
            os="Darwin"
            ;;
        CYGWIN*|MINGW32*|MSYS*|MINGW*)
            os="Windows"
            ;;
        *)
            error "Unsupported operating system: $(uname -s)"
            ;;
    esac

    # Detect architecture (match GitHub release naming: x86_64, arm64, armv7)
    case "$(uname -m)" in
        x86_64|amd64)
            arch="x86_64"
            ;;
        arm64|aarch64)
            arch="arm64"
            ;;
        i386|i686)
            arch="i386"
            ;;
        armv6*|armv7*)
            arch="armv7"
            ;;
        *)
            error "Unsupported architecture: $(uname -m)"
            ;;
    esac

    PLATFORM="${os}_${arch}"
    debug "Detected platform: $PLATFORM"
}

# Get latest release version
get_latest_version() {
    log "Fetching latest release information..."
    
    if command_exists curl; then
        VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command_exists wget; then
        VERSION=$(wget -qO- "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        error "curl or wget is required to download s9s"
    fi
    
    if [ -z "$VERSION" ]; then
        error "Failed to get latest release version"
    fi
    
    debug "Latest version: $VERSION"
}

# Download binary
download_binary() {
    local download_url="https://github.com/$REPO/releases/download/$VERSION/s9s_${VERSION#v}_${PLATFORM}.tar.gz"
    local temp_dir=$(mktemp -d)
    local archive_file="$temp_dir/s9s.tar.gz"
    
    log "Downloading s9s $VERSION for $PLATFORM..."
    debug "Download URL: $download_url"
    
    if command_exists curl; then
        curl -L -o "$archive_file" "$download_url" || error "Failed to download s9s"
    elif command_exists wget; then
        wget -O "$archive_file" "$download_url" || error "Failed to download s9s"
    else
        error "curl or wget is required to download s9s"
    fi
    
    # Extract archive
    log "Extracting archive..."
    cd "$temp_dir"
    tar -xzf "$archive_file" || error "Failed to extract archive"
    
    # Find the binary
    if [ -f "s9s" ]; then
        BINARY_PATH="$temp_dir/s9s"
    elif [ -f "s9s.exe" ]; then
        BINARY_PATH="$temp_dir/s9s.exe"
        BINARY_NAME="s9s.exe"
    else
        error "Could not find s9s binary in archive"
    fi
    
    debug "Binary path: $BINARY_PATH"
}

# Install binary
install_binary() {
    log "Installing s9s to $INSTALL_DIR..."

    # Create install directory if it doesn't exist
    if [ ! -d "$INSTALL_DIR" ]; then
        mkdir -p "$INSTALL_DIR" || error "Failed to create $INSTALL_DIR"
        log "Created directory: $INSTALL_DIR"
    fi

    # Copy binary
    cp "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME" || error "Failed to install binary"
    chmod +x "$INSTALL_DIR/$BINARY_NAME" || error "Failed to make binary executable"

    log "s9s installed successfully!"

    # Check if install directory is in PATH
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        warn "$INSTALL_DIR is not in your PATH"
        echo
        echo "Add the following to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
        echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
        echo
        echo "Then reload your shell:"
        echo "  source ~/.bashrc  # or ~/.zshrc"
        echo
    fi
}

# Setup configuration
setup_config() {
    log "Setting up configuration directory..."
    
    # Create config directory
    mkdir -p "$CONFIG_DIR"
    
    # Create basic config if it doesn't exist
    if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
        log "Creating default configuration..."
        cat > "$CONFIG_DIR/config.yaml" << 'EOF'
# s9s Configuration File
# See https://s9s.dev/docs/configuration for full documentation

# Cluster connections (optional - s9s auto-discovers on SLURM nodes)
# clusters:
#   - name: default
#     cluster:
#       endpoint: https://slurm.example.com:6820
#       token: ${SLURM_JWT}
#       timeout: 30s
EOF
        log "Created default configuration at $CONFIG_DIR/config.yaml"
    else
        debug "Configuration file already exists"
    fi
    
    # Create plugins directory
    mkdir -p "$CONFIG_DIR/plugins"
    debug "Created plugins directory at $CONFIG_DIR/plugins"
}

# Verify installation
verify_installation() {
    log "Verifying installation..."
    
    if command_exists "$BINARY_NAME"; then
        local installed_version
        installed_version=$("$BINARY_NAME" --version 2>/dev/null | head -1 | awk '{print $NF}' || echo "unknown")
        log "s9s $installed_version installed successfully!"
        
        echo
        echo "🚀 Quick Start:"
        echo
        echo "  # On a SLURM node (auto-discovers slurmrestd):"
        echo "  s9s"
        echo
        echo "  # Or connect manually:"
        echo "  export SLURM_REST_URL=https://your-slurm-cluster.com:6820"
        echo "  export SLURM_JWT=your-token"
        echo "  s9s"
        echo
        echo "  # If auto-discovery doesn't work, run the setup wizard:"
        echo "  s9s setup"
        echo
        echo "📖 Documentation: https://s9s.dev/docs"
        echo "🔧 Configuration: $CONFIG_DIR/config.yaml"
        echo "❓ Help:          s9s --help"
        
        return 0
    else
        error "Installation verification failed - s9s command not found"
    fi
}

# Cleanup temporary files
cleanup() {
    if [ -n "${temp_dir:-}" ] && [ -d "$temp_dir" ]; then
        rm -rf "$temp_dir"
        debug "Cleaned up temporary directory: $temp_dir"
    fi
}

# Main installation function
main() {
    log "Starting s9s installation..."
    
    # Set up cleanup trap
    trap cleanup EXIT
    
    # Check prerequisites
    if ! command_exists tar; then
        error "tar is required for installation"
    fi
    
    # Detect platform
    detect_platform
    
    # Get version to install
    if [ -n "${VERSION:-}" ]; then
        log "Installing specified version: $VERSION"
    else
        get_latest_version
    fi
    
    # Download and install
    download_binary
    install_binary
    setup_config
    verify_installation
    
    log "Installation completed successfully! 🎉"
}

# Command line argument parsing
while [[ $# -gt 0 ]]; do
    case $1 in
        --version)
            VERSION="$2"
            shift 2
            ;;
        --install-dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        --debug)
            DEBUG=1
            shift
            ;;
        --help|-h)
            cat << 'EOF'
s9s Installation Script

USAGE:
    curl -sSL https://get.s9s.dev | bash

OPTIONS:
    --version VERSION     Install specific version (default: latest)
    --install-dir DIR     Installation directory (default: ~/.local/bin)
    --debug              Enable debug output
    --help               Show this help message

EXAMPLES:
    # Install latest version
    curl -sSL https://get.s9s.dev | bash

    # Install specific version
    curl -sSL https://get.s9s.dev | bash -s -- --version v0.6.1

    # Install to custom directory (e.g., system-wide)
    curl -sSL https://get.s9s.dev | bash -s -- --install-dir /usr/local/bin

For more information, visit: https://s9s.dev/docs/installation
EOF
            exit 0
            ;;
        *)
            error "Unknown option: $1"
            ;;
    esac
done

# Run main installation
main