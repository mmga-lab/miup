#!/bin/sh
# MiUp Installation Script
# Usage: curl -sSL https://raw.githubusercontent.com/mmga-lab/miup/main/install.sh | sh
#
# Environment variables:
#   MIUP_HOME    - Installation directory (default: ~/.miup)
#   MIUP_VERSION - Specific version to install (default: latest)

set -e

REPO="mmga-lab/miup"
BINARY_NAME="miup"
DEFAULT_INSTALL_DIR="${MIUP_HOME:-$HOME/.miup}"
BIN_DIR="$DEFAULT_INSTALL_DIR/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info() {
    printf "${BLUE}info:${NC} %s\n" "$1"
}

success() {
    printf "${GREEN}success:${NC} %s\n" "$1"
}

warn() {
    printf "${YELLOW}warning:${NC} %s\n" "$1"
}

error() {
    printf "${RED}error:${NC} %s\n" "$1" >&2
    exit 1
}

# Detect OS
detect_os() {
    OS="$(uname -s)"
    case "$OS" in
        Linux*)  OS="linux" ;;
        Darwin*) OS="darwin" ;;
        *)       error "Unsupported operating system: $OS" ;;
    esac
    echo "$OS"
}

# Detect architecture
detect_arch() {
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64|amd64)  ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *)             error "Unsupported architecture: $ARCH" ;;
    esac
    echo "$ARCH"
}

# Get latest version from GitHub
get_latest_version() {
    if command -v curl >/dev/null 2>&1; then
        curl -sL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/'
    else
        error "curl or wget is required"
    fi
}

# Download file
download() {
    URL="$1"
    DEST="$2"

    info "Downloading from $URL"

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$URL" -o "$DEST"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$URL" -O "$DEST"
    else
        error "curl or wget is required"
    fi
}

# Main installation
main() {
    info "Installing MiUp..."

    OS=$(detect_os)
    ARCH=$(detect_arch)

    info "Detected OS: $OS, Arch: $ARCH"

    # Get version
    VERSION="${MIUP_VERSION:-}"
    if [ -z "$VERSION" ]; then
        info "Fetching latest version..."
        VERSION=$(get_latest_version)
        if [ -z "$VERSION" ]; then
            error "Failed to get latest version. Please specify MIUP_VERSION or check your network."
        fi
    fi

    info "Version: $VERSION"

    # Construct download URL
    BINARY_FILENAME="${BINARY_NAME}-${OS}-${ARCH}"
    if [ "$OS" = "windows" ]; then
        BINARY_FILENAME="${BINARY_FILENAME}.exe"
    fi

    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$BINARY_FILENAME"

    # Create installation directory
    mkdir -p "$BIN_DIR"

    # Download binary
    TMP_FILE=$(mktemp)
    trap "rm -f $TMP_FILE" EXIT

    download "$DOWNLOAD_URL" "$TMP_FILE"

    # Install binary
    INSTALL_PATH="$BIN_DIR/$BINARY_NAME"
    mv "$TMP_FILE" "$INSTALL_PATH"
    chmod +x "$INSTALL_PATH"

    success "MiUp $VERSION installed to $INSTALL_PATH"

    # Check if bin directory is in PATH
    case ":$PATH:" in
        *":$BIN_DIR:"*) ;;
        *)
            echo ""
            warn "Add the following to your shell profile (.bashrc, .zshrc, etc.):"
            echo ""
            echo "  export PATH=\"\$PATH:$BIN_DIR\""
            echo ""
            ;;
    esac

    # Verify installation
    if [ -x "$INSTALL_PATH" ]; then
        echo ""
        success "Installation complete!"
        echo ""
        info "Run 'miup --help' to get started"
        info "Quick start: 'miup playground start' to start a local Milvus instance"
    else
        error "Installation failed"
    fi
}

main "$@"
