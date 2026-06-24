#!/bin/sh
set -e

# Detect OS and Architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

# Normalize ARCH
case "$ARCH" in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

# Normalize OS
case "$OS" in
    darwin)
        OS="darwin"
        ;;
    linux)
        OS="linux"
        ;;
    *)
        echo "Unsupported operating system: $OS"
        exit 1
        ;;
esac

BINARY_NAME="mean-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/mini-page/mean-cli/releases/latest/download/${BINARY_NAME}"

echo "Installing mean-cli..."
echo "Downloading from: ${DOWNLOAD_URL}"

# Create a temporary directory
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

# Download binary
curl -sSfL "$DOWNLOAD_URL" -o "$TMP_DIR/mean"

# Determine install directory (fallback to ~/.local/bin if /usr/local/bin is read-only)
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
fi

# Move and make executable
mv "$TMP_DIR/mean" "$INSTALL_DIR/mean"
chmod +x "$INSTALL_DIR/mean"

echo ""
echo "✓ mean-cli successfully installed at: $INSTALL_DIR/mean"
echo "  Run 'mean' to launch the TUI or check the CLI help with 'mean --help'."
echo ""
