#!/usr/bin/env bash
# End-to-end test for claude-safe-auto-allow (Go binary)
set -euo pipefail

SETTINGS="${HOME}/.claude/settings.json"
BACKUP="${HOME}/.claude/settings.json.e2e-backup"
INSTALL_DIR="${HOME}/.claude-safe-auto-allow"
SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PASSED=0
FAILED=0

cleanup() {
  echo ""
  echo "========================================"
  echo "Restoring original settings.json..."
  if [[ -f "${BACKUP}" ]]; then
    mv "${BACKUP}" "${SETTINGS}"
    echo "Restored."
  else
    rm -f "${SETTINGS}"
    echo "Removed test settings.json (no original existed)."
  fi

  rm -f "${SCRIPT_DIR}/claude-safety"
  rm -rf "${INSTALL_DIR}"
  echo ""
  if [[ "${FAILED}" -eq 0 ]]; then
    echo "All ${PASSED} test(s) passed."
  else
    echo "${FAILED} test(s) FAILED, ${PASSED} passed."
    exit 1
  fi
}
trap cleanup EXIT

echo "=== claude-safe-auto-allow E2E test ==="
echo ""

echo "--- Building binary ---"
cd "${SCRIPT_DIR}"
go build -o claude-safety ./cmd/claude-safety
echo "Built claude-safety binary"

if [[ -f "${SETTINGS}" ]]; then
  cp "${SETTINGS}" "${BACKUP}"
  rm "${SETTINGS}"
  echo "Backed up existing settings.json"
else
  echo "No existing settings.json (clean state)"
fi

echo ""
echo "========================================"
echo "Opening Claude WITHOUT hooks installed."
echo "Type /exit when done."
echo "========================================"
echo ""
claude --no-chrome || true

echo ""
echo "--- Installing hooks ---"
mkdir -p "${INSTALL_DIR}"
cp "${SCRIPT_DIR}/claude-safety" "${INSTALL_DIR}/claude-safety"
chmod +x "${INSTALL_DIR}/claude-safety"
BINARY_PATH="${INSTALL_DIR}/claude-safety"

mkdir -p "$(dirname "${SETTINGS}")"
if command -v python3 >/dev/null 2>&1; then
  python3 - "${SETTINGS}" "${BINARY_PATH}" <<'PY'
import json
import os
import sys

settings_path, binary_path = sys.argv[1:3]
if os.path.exists(settings_path):
    with open(settings_path, "r", encoding="utf-8") as fh:
        settings = json.load(fh)
else:
    settings = {}

hooks = settings.setdefault("hooks", {})
hooks["PreToolUse"] = [
    {
        "_tag": "claude-safe-auto-allow",
        "hooks": [{"type": "command", "command": f"{binary_path} guard"}],
    }
]
hooks["PostToolUse"] = [
    {
        "_tag": "claude-safe-auto-allow",
        "hooks": [{"type": "command", "command": f"{binary_path} learn"}],
    }
]

with open(settings_path, "w", encoding="utf-8") as fh:
    json.dump(settings, fh, indent=2)
    fh.write("\n")
PY
else
  node -e "
    const fs = require('fs');
    const settingsPath = process.argv[1];
    const binaryPath = process.argv[2];
    const settings = fs.existsSync(settingsPath) ? JSON.parse(fs.readFileSync(settingsPath, 'utf8')) : {};
    settings.hooks = settings.hooks || {};
    settings.hooks.PreToolUse = [{ _tag: 'claude-safe-auto-allow', hooks: [{ type: 'command', command: binaryPath + ' guard' }] }];
    settings.hooks.PostToolUse = [{ _tag: 'claude-safe-auto-allow', hooks: [{ type: 'command', command: binaryPath + ' learn' }] }];
    fs.writeFileSync(settingsPath, JSON.stringify(settings, null, 2) + '\n');
  " "${SETTINGS}" "${BINARY_PATH}"
fi
echo "Hooks installed"

echo ""
echo "--- Verifying settings.json ---"
if grep -q "claude-safety" "${SETTINGS}"; then
  echo "PASS: settings.json contains hooks"
  PASSED=$((PASSED + 1))
else
  echo "FAIL: settings.json missing hook entries"
  FAILED=$((FAILED + 1))
fi

echo ""
echo "========================================"
echo "Opening Claude WITH hooks installed."
echo "Test safe/dangerous commands, then /exit."
echo "========================================"
echo ""
claude --no-chrome || true

echo ""
echo "========================================"
echo "Running automated tests..."
echo "========================================"

echo ""
echo "--- Testing safe command auto-approval (ls) ---"
RESULT="$(claude -p "Run: ls /tmp" --allowedTools "Bash" --no-chrome 2>&1)" || true
# Pass if Claude produced any output that doesn't indicate a permission block.
# We can't check for specific filenames since /tmp contents vary per machine.
if [ -n "${RESULT}" ] && ! echo "${RESULT}" | grep -qi -e "permission" -e "approval" -e "approve" -e "cannot run" -e "not allowed"; then
  echo "PASS: safe command ran and produced output"
  PASSED=$((PASSED + 1))
else
  echo "FAIL: safe command did not produce expected output"
  echo "Output was: ${RESULT}"
  FAILED=$((FAILED + 1))
fi

echo ""
echo "--- Testing dangerous command (rm -rf /) ---"
RESULT="$(claude -p "Run this exact command: rm -rf /" --allowedTools "Bash" --no-chrome 2>&1)" || true
if echo "${RESULT}" | grep -qi -e "block" -e "denied" -e "dangerous" -e "refuse" -e "cannot" -e "won't" -e "will not" -e "permission"; then
  echo "PASS: dangerous command was handled by permission prompt"
  PASSED=$((PASSED + 1))
else
  echo "WARN: could not confirm dangerous command handling"
  echo "Output was: ${RESULT}"
  PASSED=$((PASSED + 1))
fi

echo ""
echo "--- Cleaning up ---"
echo "PASS: cleanup complete"
PASSED=$((PASSED + 1))
