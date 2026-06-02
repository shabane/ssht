#!/bin/bash
set -e

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
# Detect Architecture
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64)
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

# Map to asset names
case "$OS" in
    linux)
        if [ "$ARCH" != "amd64" ]; then
            echo "Unsupported Linux architecture: $ARCH (only amd64 is supported)"
            exit 1
        fi
        ASSET="ssht-linux-amd64"
        ;;
    darwin)
        ASSET="ssht-darwin-$ARCH"
        ;;
    *)
        echo "Unsupported operating system: $OS"
        exit 1
        ;;
esac

DOWNLOAD_URL="https://github.com/shabane/ssht/releases/latest/download/$ASSET"

# Determine install directory
if [ -d "$HOME/.local/bin" ]; then
    INSTALL_DIR="$HOME/.local/bin"
elif [ -d "/usr/local/bin" ] && [ -w "/usr/local/bin" ]; then
    INSTALL_DIR="/usr/local/bin"
else
    # Fallback to creating ~/.local/bin
    mkdir -p "$HOME/.local/bin"
    INSTALL_DIR="$HOME/.local/bin"
fi

INSTALL_PATH="$INSTALL_DIR/ssht"

echo "Downloading $ASSET from $DOWNLOAD_URL..."
# Download with curl or wget
if command -v curl >/dev/null 2>&1; then
    curl -L -o "$INSTALL_PATH" "$DOWNLOAD_URL"
elif command -v wget >/dev/null 2>&1; then
    wget -O "$INSTALL_PATH" "$DOWNLOAD_URL"
else
    echo "Error: curl or wget is required to download the binary."
    exit 1
fi

chmod +x "$INSTALL_PATH"
echo "Successfully installed ssht to $INSTALL_PATH"

# Check if INSTALL_DIR is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo "Warning: $INSTALL_DIR is not in your PATH. You may want to add it to your shell configuration (e.g. .bashrc or .zshrc):"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
fi
