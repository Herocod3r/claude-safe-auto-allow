#!/usr/bin/env node
// Install script for claude-safe-auto-allow
// Patches ~/.claude/settings.json to add the PreToolUse and PostToolUse hooks
// pointing at the hook scripts in this package.
//
// Usage:
//   node scripts/install.js          # install from repo clone
//   npx claude-safe-auto-allow       # install via npx (runs postinstall)

const fs = require('fs');
const path = require('path');
const os = require('os');

const SETTINGS_PATH = path.join(os.homedir(), '.claude', 'settings.json');
const HOOKS_DIR = path.resolve(__dirname, '..', 'hooks');
const GUARD_CMD = `node "${path.join(HOOKS_DIR, 'safety-guard.js')}"`;
const LEARN_CMD = `node "${path.join(HOOKS_DIR, 'safety-learn.js')}"`;

const TAG = 'claude-safe-auto-allow'; // used to identify our hooks for uninstall

function main() {
  console.log('claude-safe-auto-allow installer');
  console.log('================================\n');

  // Ensure ~/.claude directory exists
  const claudeDir = path.join(os.homedir(), '.claude');
  if (!fs.existsSync(claudeDir)) {
    fs.mkdirSync(claudeDir, { recursive: true });
    console.log(`Created ${claudeDir}`);
  }

  // Load or create settings
  let settings = {};
  if (fs.existsSync(SETTINGS_PATH)) {
    const backupPath = `${SETTINGS_PATH}.bak.${Date.now()}`;
    fs.copyFileSync(SETTINGS_PATH, backupPath);
    console.log(`Backed up settings to ${backupPath}`);
    settings = JSON.parse(fs.readFileSync(SETTINGS_PATH, 'utf8'));
  } else {
    console.log('No existing settings.json found, creating new one.');
  }

  if (!settings.hooks) {
    settings.hooks = {};
  }

  // --- Add PreToolUse hook ---
  if (!settings.hooks.PreToolUse) {
    settings.hooks.PreToolUse = [];
  }

  // Remove any existing claude-safe-auto-allow entries
  settings.hooks.PreToolUse = settings.hooks.PreToolUse.filter(
    entry => !isOurHook(entry)
  );

  settings.hooks.PreToolUse.unshift({
    _tag: TAG,
    hooks: [
      {
        type: 'command',
        command: GUARD_CMD
      }
    ]
  });

  console.log('Added PreToolUse hook (safety-guard.js)');

  // --- Add PostToolUse hook ---
  if (!settings.hooks.PostToolUse) {
    settings.hooks.PostToolUse = [];
  }

  settings.hooks.PostToolUse = settings.hooks.PostToolUse.filter(
    entry => !isOurHook(entry)
  );

  settings.hooks.PostToolUse.push({
    _tag: TAG,
    hooks: [
      {
        type: 'command',
        command: LEARN_CMD
      }
    ]
  });

  console.log('Added PostToolUse hook (safety-learn.js)');

  // Write settings
  fs.writeFileSync(SETTINGS_PATH, JSON.stringify(settings, null, 2) + '\n');

  console.log(`\nSettings written to ${SETTINGS_PATH}`);
  console.log('\nInstallation complete! Restart Claude Code to activate.\n');
  console.log('What happens now:');
  console.log('  - Dangerous commands (rm -rf /, DROP DATABASE, etc.) are BLOCKED');
  console.log('  - Safe commands (ls, git status, npm test, etc.) are AUTO-APPROVED');
  console.log('  - Unknown commands prompt you, then offer to remember your choice');
  console.log(`\nYour learned allowlist: ${path.join(HOOKS_DIR, 'safety-allowlist.json')}`);
}

function isOurHook(entry) {
  if (entry._tag === TAG) return true;
  // Also match by command path in case tag was stripped
  if (entry.hooks) {
    return entry.hooks.some(h =>
      h.command && h.command.includes('safety-guard.js') || h.command.includes('safety-learn.js')
    );
  }
  return false;
}

main();
