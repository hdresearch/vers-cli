#!/bin/sh
# vers-cli installation script
# Usage: curl -fsSL https://raw.githubusercontent.com/hdresearch/vers-cli/main/install.sh | sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO="hdresearch/vers-cli"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
GITHUB_API="https://api.github.com/repos/${REPO}/releases/latest"

# Helper functions
info() {
    printf "${BLUE}info${NC}: %s\n" "$1"
}

success() {
    printf "${GREEN}success${NC}: %s\n" "$1"
}

warn() {
    printf "${YELLOW}warning${NC}: %s\n" "$1"
}

error() {
    printf "${RED}error${NC}: %s\n" "$1" >&2
    exit 1
}

# Detect OS
detect_os() {
    OS="$(uname -s)"
    case "$OS" in
        Linux*)     OS='linux';;
        Darwin*)    OS='darwin';;
        MINGW*|MSYS*|CYGWIN*)    OS='windows';;
        *)          error "Unsupported operating system: $OS";;
    esac
    echo "$OS"
}

# Detect architecture
detect_arch() {
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64|amd64)   ARCH='amd64';;
        aarch64|arm64)  ARCH='arm64';;
        *)              error "Unsupported architecture: $ARCH";;
    esac
    echo "$ARCH"
}

# Get the latest release version
get_latest_version() {
    if command -v curl >/dev/null 2>&1; then
        VERSION=$(curl -fsSL "$GITHUB_API" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command -v wget >/dev/null 2>&1; then
        VERSION=$(wget -qO- "$GITHUB_API" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        error "Neither curl nor wget is available. Please install one of them."
    fi

    if [ -z "$VERSION" ]; then
        error "Failed to get the latest version"
    fi

    echo "$VERSION"
}

# Download file
download() {
    URL="$1"
    OUTPUT="$2"

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL -o "$OUTPUT" "$URL" || error "Failed to download $URL"
    elif command -v wget >/dev/null 2>&1; then
        wget -q -O "$OUTPUT" "$URL" || error "Failed to download $URL"
    fi
}

# Verify checksum
verify_checksum() {
    FILE="$1"
    CHECKSUM_FILE="$2"

    EXPECTED_CHECKSUM=$(cat "$CHECKSUM_FILE" | awk '{print $1}')

    if command -v shasum >/dev/null 2>&1; then
        ACTUAL_CHECKSUM=$(shasum -a 256 "$FILE" | awk '{print $1}')
    elif command -v sha256sum >/dev/null 2>&1; then
        ACTUAL_CHECKSUM=$(sha256sum "$FILE" | awk '{print $1}')
    else
        warn "Neither shasum nor sha256sum found, skipping checksum verification"
        return 0
    fi

    if [ "$EXPECTED_CHECKSUM" != "$ACTUAL_CHECKSUM" ]; then
        error "Checksum verification failed!\nExpected: $EXPECTED_CHECKSUM\nActual: $ACTUAL_CHECKSUM"
    fi

    success "Checksum verified"
}

# Main installation logic
main() {
    info "Installing vers-cli..."

    # Detect system
    OS=$(detect_os)
    ARCH=$(detect_arch)
    info "Detected OS: $OS, Architecture: $ARCH"

    # Get version
    VERSION="${VERS_VERSION:-$(get_latest_version)}"
    info "Installing version: $VERSION"

    # Construct binary name
    if [ "$OS" = "windows" ]; then
        BINARY_NAME="vers-${OS}-${ARCH}.exe"
        INSTALL_NAME="vers.exe"
    else
        BINARY_NAME="vers-${OS}-${ARCH}"
        INSTALL_NAME="vers"
    fi

    # Create temporary directory
    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT

    # Construct download URLs
    BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
    BINARY_URL="${BASE_URL}/${BINARY_NAME}"
    CHECKSUM_URL="${BASE_URL}/${BINARY_NAME}.sha256"

    info "Downloading ${BINARY_NAME}..."
    download "$BINARY_URL" "$TMP_DIR/$BINARY_NAME"

    info "Downloading checksum..."
    download "$CHECKSUM_URL" "$TMP_DIR/${BINARY_NAME}.sha256"

    # Verify checksum
    info "Verifying checksum..."
    verify_checksum "$TMP_DIR/$BINARY_NAME" "$TMP_DIR/${BINARY_NAME}.sha256"

    # Create install directory if it doesn't exist
    if [ ! -d "$INSTALL_DIR" ]; then
        info "Creating installation directory: $INSTALL_DIR"
        mkdir -p "$INSTALL_DIR" || error "Failed to create $INSTALL_DIR"
    fi

    # Install binary
    info "Installing to $INSTALL_DIR/$INSTALL_NAME..."

    # Try to install without sudo first
    if cp "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$INSTALL_NAME" 2>/dev/null; then
        chmod +x "$INSTALL_DIR/$INSTALL_NAME" 2>/dev/null || true
    else
        # If that fails, try with sudo
        warn "Permission denied. Attempting to install with sudo..."
        sudo cp "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/$INSTALL_NAME" || error "Failed to install binary"
        sudo chmod +x "$INSTALL_DIR/$INSTALL_NAME" || error "Failed to make binary executable"
    fi

    success "vers-cli installed successfully!"

    # Check if install directory is in PATH
    case ":$PATH:" in
        *":$INSTALL_DIR:"*)
            ;;
        *)
            echo ""
            warn "Installation directory is not in your PATH!"
            info "Add the following to your shell configuration file (.bashrc, .zshrc, etc.):"
            printf "    ${GREEN}export PATH=\"\$PATH:$INSTALL_DIR\"${NC}\n"
            echo ""
            info "Or run the command above in your current shell to use vers immediately."
            ;;
    esac

    echo ""
    info "Run 'vers --version' to verify the installation"
    info "Run 'vers --help' to get started"
    echo ""
    info "Documentation: https://github.com/${REPO}"
}

# Run main function
main "$@"
