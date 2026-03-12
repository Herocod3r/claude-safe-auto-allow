#!/usr/bin/env bash
# claude-safe-auto-allow installer
# One-liner: curl -fsSL https://raw.githubusercontent.com/Herocod3r/claude-safe-auto-allow/main/install.sh | bash
set -euo pipefail

REPO="Herocod3r/claude-safe-auto-allow"
INSTALL_DIR="${HOME}/.claude-safe-auto-allow"

echo "claude-safe-auto-allow installer"
echo "================================"
echo ""

# Check for node
if ! command -v node &> /dev/null; then
  echo "Error: Node.js is required but not installed."
  echo "Install it from https://nodejs.org or via your package manager."
  exit 1
fi

NODE_VERSION=$(node -v | sed 's/v//' | cut -d. -f1)
if [ "$NODE_VERSION" -lt 16 ]; then
  echo "Error: Node.js >= 16 required. Found: $(node -v)"
  exit 1
fi

# Check for git
if ! command -v git &> /dev/null; then
  echo "Error: git is required but not installed."
  exit 1
fi

# Clone or update
if [ -d "$INSTALL_DIR" ]; then
  echo "Updating existing installation at $INSTALL_DIR..."
  cd "$INSTALL_DIR"
  git pull --ff-only origin main
else
  echo "Cloning repository to $INSTALL_DIR..."
  git clone "https://github.com/${REPO}.git" "$INSTALL_DIR"
  cd "$INSTALL_DIR"
fi

echo ""

# Run the Node.js installer which patches settings.json
node scripts/install.js

echo ""
echo "Installation directory: $INSTALL_DIR"
echo ""
echo "To uninstall:"
echo "  node ${INSTALL_DIR}/scripts/uninstall.js"
echo "  rm -rf ${INSTALL_DIR}"
