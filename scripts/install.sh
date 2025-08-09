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
INSTALL_DIR="/usr/local/bin"
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
    
    # Detect OS
    case "$(uname -s)" in
        Linux*)
            os="linux"
            ;;
        Darwin*)
            os="darwin"
            ;;
        CYGWIN*|MINGW32*|MSYS*|MINGW*)
            os="windows"
            ;;
        *)
            error "Unsupported operating system: $(uname -s)"
            ;;
    esac
    
    # Detect architecture
    case "$(uname -m)" in
        x86_64|amd64)
            arch="amd64"
            ;;
        arm64|aarch64)
            arch="arm64"
            ;;
        i386|i686)
            arch="386"
            ;;
        armv6*|armv7*)
            arch="arm"
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
    
    # Check if we need sudo
    if [ ! -w "$INSTALL_DIR" ]; then
        if command_exists sudo; then
            SUDO="sudo"
            warn "Administrator privileges required for installation"
        else
            error "Cannot write to $INSTALL_DIR and sudo is not available"
        fi
    fi
    
    # Copy binary
    $SUDO cp "$BINARY_PATH" "$INSTALL_DIR/$BINARY_NAME" || error "Failed to install binary"
    $SUDO chmod +x "$INSTALL_DIR/$BINARY_NAME" || error "Failed to make binary executable"
    
    log "s9s installed successfully!"
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

# Default cluster (optional - you can specify multiple clusters)
# clusters:
#   production:
#     url: https://slurm.example.com
#     auth:
#       method: token
#       token: ${SLURM_TOKEN}

# User preferences
preferences:
  theme: dark
  refresh_interval: 30s
  default_view: jobs

# Logging
logging:
  level: info
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
        installed_version=$("$BINARY_NAME" --version 2>/dev/null | head -1 | awk '{print $3}' || echo "unknown")
        log "s9s $installed_version installed successfully!"
        
        echo
        echo "🚀 Quick Start:"
        echo "  # Run with mock data (no SLURM cluster required)"
        echo "  s9s --mock"
        echo
        echo "  # Connect to your SLURM cluster"
        echo "  export SLURM_URL=https://your-slurm-cluster.com"
        echo "  export SLURM_TOKEN=your-token"
        echo "  s9s"
        echo
        echo "📖 Documentation:"
        echo "  https://s9s.dev/docs"
        echo
        echo "🔧 Configuration:"
        echo "  $CONFIG_DIR/config.yaml"
        echo
        echo "❓ Get help:"
        echo "  s9s --help"
        
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
    --install-dir DIR     Installation directory (default: /usr/local/bin)
    --debug              Enable debug output
    --help               Show this help message

EXAMPLES:
    # Install latest version
    curl -sSL https://get.s9s.dev | bash
    
    # Install specific version
    curl -sSL https://get.s9s.dev | bash -s -- --version v1.0.0
    
    # Install to custom directory
    curl -sSL https://get.s9s.dev | bash -s -- --install-dir ~/.local/bin

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