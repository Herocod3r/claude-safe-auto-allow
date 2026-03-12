#!/usr/bin/env bash
# claude-safe-auto-allow uninstaller
set -euo pipefail

INSTALL_DIR="${HOME}/.claude-safe-auto-allow"

if [ ! -d "$INSTALL_DIR" ]; then
  echo "Installation not found at $INSTALL_DIR"
  echo "If you installed from a different location, run:"
  echo "  node /path/to/claude-safe-auto-allow/scripts/uninstall.js"
  exit 1
fi

node "${INSTALL_DIR}/scripts/uninstall.js"

read -p "Remove installation directory ${INSTALL_DIR}? [y/N] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
  rm -rf "$INSTALL_DIR"
  echo "Removed $INSTALL_DIR"
fi

echo "Done."
