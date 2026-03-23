#!/bin/sh
set -e

REPO="imclaw/wechat-claude-go"
BIN="weclaude"
INSTALL_DIR="/usr/local/bin"

# Detect OS and architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Darwin)
    case "$ARCH" in
      arm64)  FILE="weclaude-darwin-arm64" ;;
      x86_64) FILE="weclaude-darwin-amd64" ;;
      *)      echo "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    ;;
  Linux)
    case "$ARCH" in
      x86_64)  FILE="weclaude-linux-amd64" ;;
      aarch64) FILE="weclaude-linux-arm64" ;;
      *)       echo "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    ;;
  *)
    echo "Unsupported OS: $OS"
    echo "Windows users: download manually from https://github.com/$REPO/releases/latest"
    exit 1
    ;;
esac

# Fetch latest release version
echo "Fetching latest version..."
LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' \
  | sed 's/.*"tag_name": *"\(.*\)".*/\1/')

if [ -z "$LATEST" ]; then
  echo "Failed to fetch version. Check your network or download manually from https://github.com/$REPO/releases"
  exit 1
fi

echo "Latest version: $LATEST"

URL="https://github.com/$REPO/releases/download/$LATEST/$FILE"
TMP="$(mktemp)"

echo "Downloading $FILE ..."
curl -fsSL "$URL" -o "$TMP"
chmod +x "$TMP"

# Install to /usr/local/bin
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP" "$INSTALL_DIR/$BIN"
else
  echo "Requesting sudo to install to $INSTALL_DIR ..."
  sudo mv "$TMP" "$INSTALL_DIR/$BIN"
fi

echo ""
echo "Installation complete! Get started with:"
echo ""
echo "  weclaude login   # scan QR code to log in"
echo "  weclaude         # start the service"
echo ""
