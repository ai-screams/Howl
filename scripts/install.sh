#!/bin/bash
set -euo pipefail

REPO="ai-screams/Howl"
INSTALL_DIR="$HOME/.claude/hud"
BINARY="howl"
SETTINGS="$HOME/.claude/settings.json"

echo "Installing Howl..."

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  darwin|linux) ;;
  *) echo "Error: Unsupported OS: $OS (supported: darwin, linux)"; exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)        ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Error: Unsupported architecture: $ARCH (supported: amd64, arm64)"; exit 1 ;;
esac

echo "  Platform: ${OS}/${ARCH}"

# Get latest release version
echo "  Fetching latest release..."
RELEASE_JSON=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null) || {
  echo "Error: Failed to fetch release info from GitHub API"
  echo "  This may be caused by rate limiting (60 req/hour for unauthenticated requests)"
  echo "  Try: https://github.com/$REPO/releases/latest"
  exit 1
}

if command -v jq &>/dev/null; then
  VERSION=$(echo "$RELEASE_JSON" | jq -r '.tag_name' | sed 's/^v//')
else
  VERSION=$(echo "$RELEASE_JSON" | grep '"tag_name"' | sed 's/.*"tag_name": *"v\{0,1\}//' | sed 's/".*//')
fi

if [ -z "$VERSION" ]; then
  echo "Error: Could not determine latest version"
  exit 1
fi

echo "  Version: v${VERSION}"

# Download binary and checksum
ASSET_NAME="howl_${OS}_${ARCH}"
BASE_URL="https://github.com/$REPO/releases/download/v${VERSION}"
echo "  Downloading: $ASSET_NAME"

mkdir -p "$INSTALL_DIR"
TMPDIR_DL=$(mktemp -d)
trap 'rm -rf "$TMPDIR_DL"' EXIT

if ! curl -fsSL -o "$TMPDIR_DL/$ASSET_NAME" "$BASE_URL/${ASSET_NAME}"; then
  echo "Error: Download failed from $BASE_URL/${ASSET_NAME}"
  echo "  Try manual download: https://github.com/$REPO/releases/latest"
  exit 1
fi

# Verify SHA256 checksum
if curl -fsSL -o "$TMPDIR_DL/checksums.txt" "$BASE_URL/checksums.txt" 2>/dev/null; then
  EXPECTED=$(grep "$ASSET_NAME" "$TMPDIR_DL/checksums.txt" | awk '{print $1}')
  if [ -n "$EXPECTED" ]; then
    if command -v sha256sum &>/dev/null; then
      ACTUAL=$(sha256sum "$TMPDIR_DL/$ASSET_NAME" | awk '{print $1}')
    elif command -v shasum &>/dev/null; then
      ACTUAL=$(shasum -a 256 "$TMPDIR_DL/$ASSET_NAME" | awk '{print $1}')
    else
      echo "Error: No sha256sum or shasum available — cannot verify binary integrity"
      exit 1
    fi
    if [ "$EXPECTED" != "$ACTUAL" ]; then
      echo "Error: Checksum verification failed"
      echo "  Expected: $EXPECTED"
      echo "  Actual:   $ACTUAL"
      exit 1
    fi
    echo "  Checksum verified (SHA256)"
  else
    echo "Error: Asset '$ASSET_NAME' not found in checksums.txt"
    exit 1
  fi
else
  echo "Error: Could not download checksums.txt for verification"
  exit 1
fi

# Install binary
mv "$TMPDIR_DL/$ASSET_NAME" "$INSTALL_DIR/$BINARY"
chmod +x "$INSTALL_DIR/$BINARY"
echo "  Binary installed: $INSTALL_DIR/$BINARY"

# Configure statusline in settings.json
configure_settings() {
  local cmd="$INSTALL_DIR/$BINARY"

  if [ ! -f "$SETTINGS" ]; then
    mkdir -p "$(dirname "$SETTINGS")"
    printf '{\n  "statusLine": {\n    "type": "command",\n    "command": "%s"\n  }\n}\n' "$cmd" > "$SETTINGS"
    echo "  Created $SETTINGS"
    return
  fi

  # Check if statusLine already configured
  if grep -q '"statusLine"' "$SETTINGS" 2>/dev/null; then
    echo "  statusLine already configured in $SETTINGS"
    echo "  Verify it points to: $cmd"
    return
  fi

  # Backup before modifying
  cp "$SETTINGS" "$SETTINGS.bak"

  # Append statusLine to existing settings using available JSON tool
  if command -v jq &>/dev/null; then
    jq --arg cmd "$cmd" '.statusLine = {"type": "command", "command": $cmd}' \
      "$SETTINGS" > "$SETTINGS.tmp" && mv "$SETTINGS.tmp" "$SETTINGS"
  elif command -v python3 &>/dev/null; then
    HOWL_SETTINGS_PATH="$SETTINGS" HOWL_BINARY_PATH="$cmd" python3 -c "
import json, os
settings_path = os.environ['HOWL_SETTINGS_PATH']
binary_path = os.environ['HOWL_BINARY_PATH']
with open(settings_path) as f:
    d = json.load(f)
d['statusLine'] = {'type': 'command', 'command': binary_path}
with open(settings_path, 'w') as f:
    json.dump(d, f, indent=2)
    f.write('\n')
"
  else
    echo "  Warning: Neither jq nor python3 found."
    echo "  Add manually to $SETTINGS:"
    echo "    \"statusLine\": {\"type\": \"command\", \"command\": \"$cmd\"}"
    return
  fi
  echo "  Statusline configured in $SETTINGS"
}

configure_settings

echo ""
echo "Howl v${VERSION} installed successfully!"
echo "Restart Claude Code to activate the statusline."
