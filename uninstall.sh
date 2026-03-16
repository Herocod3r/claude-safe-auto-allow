#!/usr/bin/env bash
# claude-safe-auto-allow uninstaller
set -euo pipefail

INSTALL_DIR="${HOME}/.claude-safe-auto-allow"
SETTINGS="${HOME}/.claude/settings.json"
TAG="claude-safe-auto-allow"

echo "claude-safe-auto-allow uninstaller"
echo "=================================="
echo ""

if [[ -f "${SETTINGS}" ]]; then
  backup="${SETTINGS}.bak.$(date +%s)"
  cp "${SETTINGS}" "${backup}"
  echo "Backed up settings to ${backup}"

  if command -v python3 >/dev/null 2>&1; then
    python3 - "${SETTINGS}" "${TAG}" <<'PY'
import json
import sys

settings_path, tag = sys.argv[1:3]

with open(settings_path, "r", encoding="utf-8") as fh:
    settings = json.load(fh)

hooks = settings.get("hooks", {})
for hook_type in ("PreToolUse", "PostToolUse"):
    if hook_type not in hooks:
        continue
    cleaned = []
    for entry in hooks[hook_type]:
        if entry.get("_tag") == tag:
            continue
        commands = [hook.get("command", "") for hook in entry.get("hooks", [])]
        if any("claude-safety" in command for command in commands):
            continue
        cleaned.append(entry)
    if cleaned:
        hooks[hook_type] = cleaned
    else:
        hooks.pop(hook_type, None)

if hooks:
    settings["hooks"] = hooks
else:
    settings.pop("hooks", None)

with open(settings_path, "w", encoding="utf-8") as fh:
    json.dump(settings, fh, indent=2)
    fh.write("\n")
PY
  elif command -v node >/dev/null 2>&1; then
    node -e "
      const fs = require('fs');
      const settingsPath = process.argv[1];
      const tag = process.argv[2];
      const settings = JSON.parse(fs.readFileSync(settingsPath, 'utf8'));
      const hooks = settings.hooks || {};
      for (const hookType of ['PreToolUse', 'PostToolUse']) {
        if (!hooks[hookType]) continue;
        hooks[hookType] = hooks[hookType].filter(entry => {
          if (entry._tag === tag) return false;
          return !(entry.hooks || []).some(hook => (hook.command || '').includes('claude-safety'));
        });
        if (hooks[hookType].length === 0) delete hooks[hookType];
      }
      if (Object.keys(hooks).length === 0) {
        delete settings.hooks;
      } else {
        settings.hooks = hooks;
      }
      fs.writeFileSync(settingsPath, JSON.stringify(settings, null, 2) + '\n');
    " "${SETTINGS}" "${TAG}"
  else
    echo "Warning: Need python3 or node to clean settings.json. Please remove claude-safety hooks manually."
  fi

  echo "Removed hooks from settings.json"
else
  echo "No settings.json found."
fi

echo ""
if [[ -d "${INSTALL_DIR}" ]]; then
  read -r -p "Remove installation directory ${INSTALL_DIR}? (Your allowlist will be deleted) [y/N] " reply
  if [[ "${reply}" =~ ^[Yy]$ ]]; then
    rm -rf "${INSTALL_DIR}"
    echo "Removed ${INSTALL_DIR}"
  else
    echo "Kept ${INSTALL_DIR} (allowlist preserved)"
  fi
else
  echo "No installation directory found."
fi

echo ""
echo "Uninstall complete. Restart Claude Code to apply changes."
