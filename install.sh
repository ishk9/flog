#!/bin/bash
# flog installer - https://github.com/ishk9/flog
# Usage: curl -sSL https://raw.githubusercontent.com/ishk9/flog/main/install.sh | bash

set -e

REPO="ishk9/flog"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="flog"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        linux*)  OS="linux" ;;
        darwin*) OS="darwin" ;;
        mingw*|msys*|cygwin*) OS="windows" ;;
        *) error "Unsupported OS: $OS" ;;
    esac

    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        arm64|aarch64) ARCH="arm64" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac

    echo "${OS}_${ARCH}"
}

# Get latest release version
get_latest_version() {
    curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

main() {
    echo ""
    echo "  ╭─────────────────────────────────────╮"
    echo "  │     flog - Fast Log Filter          │"
    echo "  │     https://github.com/${REPO}       │"
    echo "  ╰─────────────────────────────────────╯"
    echo ""

    # Check for required tools
    command -v curl >/dev/null 2>&1 || error "curl is required but not installed"

    # Detect platform
    PLATFORM=$(detect_platform)
    info "Detected platform: $PLATFORM"

    # Get latest version
    info "Fetching latest release..."
    VERSION=$(get_latest_version)
    
    if [ -z "$VERSION" ]; then
        error "Could not determine latest version. Check https://github.com/${REPO}/releases"
    fi
    
    info "Latest version: $VERSION"

    # Construct download URL
    FILENAME="${BINARY_NAME}_${VERSION#v}_${PLATFORM}"
    if [ "$OS" = "windows" ]; then
        FILENAME="${FILENAME}.zip"
    else
        FILENAME="${FILENAME}.tar.gz"
    fi
    
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"
    
    info "Downloading from: $DOWNLOAD_URL"

    # Create temp directory
    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT

    # Download
    if ! curl -sSL "$DOWNLOAD_URL" -o "$TMP_DIR/$FILENAME"; then
        error "Download failed. Please check if release exists at:\nhttps://github.com/${REPO}/releases"
    fi

    # Extract
    info "Extracting..."
    cd "$TMP_DIR"
    if [ "$OS" = "windows" ]; then
        unzip -q "$FILENAME"
    else
        tar -xzf "$FILENAME"
    fi

    # Install
    info "Installing to $INSTALL_DIR..."
    
    # Check if we need sudo
    if [ -w "$INSTALL_DIR" ]; then
        mv "$BINARY_NAME" "$INSTALL_DIR/"
        chmod +x "$INSTALL_DIR/$BINARY_NAME"
    else
        warn "Need sudo to install to $INSTALL_DIR"
        sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
        sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
    fi

    # Verify installation
    if command -v flog >/dev/null 2>&1; then
        echo ""
        info "✅ Successfully installed flog!"
        echo ""
        flog --version
        echo ""
        echo "  Run 'flog --help' to get started"
        echo ""
    else
        warn "Installed but flog not in PATH. Add $INSTALL_DIR to your PATH."
    fi
}

main "$@"

