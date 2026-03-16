#!/usr/bin/env bash
# claude-safe-auto-allow installer
# Downloads prebuilt binary and patches ~/.claude/settings.json
# One-liner: curl -fsSL https://raw.githubusercontent.com/Herocod3r/claude-safe-auto-allow/main/install.sh | bash
set -euo pipefail

REPO="Herocod3r/claude-safe-auto-allow"
INSTALL_DIR="${HOME}/.claude-safe-auto-allow"
SETTINGS="${HOME}/.claude/settings.json"
TAG="claude-safe-auto-allow"

echo "claude-safe-auto-allow installer"
echo "================================"
echo ""

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "${ARCH}" in
  x86_64) ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
esac

if [[ "${OS}" != "darwin" && "${OS}" != "linux" ]]; then
  echo "Error: Unsupported OS: ${OS} (need darwin or linux)"
  exit 1
fi

if [[ "${ARCH}" != "amd64" && "${ARCH}" != "arm64" ]]; then
  echo "Error: Unsupported architecture: ${ARCH} (need amd64 or arm64)"
  exit 1
fi

BINARY_NAME="claude-safety-${OS}-${ARCH}"
echo "Detected: ${OS}/${ARCH}"

echo "Fetching latest release..."
LATEST="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')"
if [[ -z "${LATEST}" ]]; then
  echo "Error: Could not determine latest release version"
  exit 1
fi
echo "Latest version: ${LATEST}"

mkdir -p "${INSTALL_DIR}"

DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST}/${BINARY_NAME}"
echo "Downloading ${DOWNLOAD_URL}..."
curl -fsSL -o "${INSTALL_DIR}/claude-safety" "${DOWNLOAD_URL}"
chmod +x "${INSTALL_DIR}/claude-safety"
printf '%s\n' "${LATEST}" > "${INSTALL_DIR}/version.txt"
echo "Installed binary to ${INSTALL_DIR}/claude-safety"

OLD_ALLOWLIST_PATHS=(
  "${INSTALL_DIR}/hooks/safety-allowlist.json"
  "${HOME}/.claude-safe-auto-allow/hooks/safety-allowlist.json"
)
for old_path in "${OLD_ALLOWLIST_PATHS[@]}"; do
  if [[ -f "${old_path}" && "${old_path}" != "${INSTALL_DIR}/safety-allowlist.json" ]] && grep -q '"patterns"' "${old_path}" 2>/dev/null; then
    cp "${old_path}" "${INSTALL_DIR}/safety-allowlist.json"
    echo "Migrated allowlist from ${old_path}"
    break
  fi
done

mkdir -p "${HOME}/.claude"
if [[ -f "${SETTINGS}" ]]; then
  backup="${SETTINGS}.bak.$(date +%s)"
  cp "${SETTINGS}" "${backup}"
  echo "Backed up settings to ${backup}"
fi

BINARY_PATH="${INSTALL_DIR}/claude-safety"

if command -v python3 >/dev/null 2>&1; then
  python3 - "${SETTINGS}" "${TAG}" "${BINARY_PATH}" <<'PY'
import json
import os
import sys

settings_path, tag, binary_path = sys.argv[1:4]

if os.path.exists(settings_path):
    with open(settings_path, "r", encoding="utf-8") as fh:
        settings = json.load(fh)
else:
    settings = {}

hooks = settings.setdefault("hooks", {})
for hook_type in ("PreToolUse", "PostToolUse"):
    entries = hooks.get(hook_type, [])
    cleaned = []
    for entry in entries:
        if entry.get("_tag") == tag:
            continue
        commands = [hook.get("command", "") for hook in entry.get("hooks", [])]
        if any(
            "safety-guard.js" in command
            or "safety-learn.js" in command
            or "claude-safety" in command
            for command in commands
        ):
            continue
        cleaned.append(entry)
    hooks[hook_type] = cleaned

hooks.setdefault("PreToolUse", []).insert(
    0,
    {
        "_tag": tag,
        "hooks": [{"type": "command", "command": f"{binary_path} guard"}],
    },
)
hooks.setdefault("PostToolUse", []).append(
    {
        "_tag": tag,
        "hooks": [{"type": "command", "command": f"{binary_path} learn"}],
    }
)

with open(settings_path, "w", encoding="utf-8") as fh:
    json.dump(settings, fh, indent=2)
    fh.write("\n")
PY
elif command -v node >/dev/null 2>&1; then
  node -e "
    const fs = require('fs');
    const settingsPath = process.argv[1];
    const tag = process.argv[2];
    const binaryPath = process.argv[3];
    const settings = fs.existsSync(settingsPath) ? JSON.parse(fs.readFileSync(settingsPath, 'utf8')) : {};
    const hooks = settings.hooks || (settings.hooks = {});
    for (const hookType of ['PreToolUse', 'PostToolUse']) {
      const entries = hooks[hookType] || [];
      hooks[hookType] = entries.filter(entry => {
        if (entry._tag === tag) return false;
        return !(entry.hooks || []).some(hook => {
          const command = hook.command || '';
          return command.includes('safety-guard.js') || command.includes('safety-learn.js') || command.includes('claude-safety');
        });
      });
    }
    hooks.PreToolUse = hooks.PreToolUse || [];
    hooks.PostToolUse = hooks.PostToolUse || [];
    hooks.PreToolUse.unshift({ _tag: tag, hooks: [{ type: 'command', command: binaryPath + ' guard' }] });
    hooks.PostToolUse.push({ _tag: tag, hooks: [{ type: 'command', command: binaryPath + ' learn' }] });
    fs.writeFileSync(settingsPath, JSON.stringify(settings, null, 2) + '\n');
  " "${SETTINGS}" "${TAG}" "${BINARY_PATH}"
else
  echo "Error: Need python3 or node to patch settings.json"
  exit 1
fi

echo ""
echo "Installation complete! Restart Claude Code to activate."
echo ""
echo "What happens now:"
echo "  - Dangerous commands go through Claude's normal permission prompt"
echo "  - Safe commands (ls, git status, npm test, etc.) are AUTO-APPROVED"
echo "  - Unknown commands prompt you, then offer to remember your choice"
echo ""
echo "Your learned allowlist: ${INSTALL_DIR}/safety-allowlist.json"
echo ""
echo "To uninstall:"
echo "  bash <(curl -fsSL https://raw.githubusercontent.com/${REPO}/main/uninstall.sh)"
