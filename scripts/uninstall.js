#!/usr/bin/env node
// Uninstall script for claude-safe-auto-allow
// Removes the PreToolUse and PostToolUse hooks from ~/.claude/settings.json

const fs = require('fs');
const path = require('path');
const os = require('os');

const SETTINGS_PATH = path.join(os.homedir(), '.claude', 'settings.json');
const TAG = 'claude-safe-auto-allow';

function main() {
  console.log('claude-safe-auto-allow uninstaller');
  console.log('==================================\n');

  if (!fs.existsSync(SETTINGS_PATH)) {
    console.log('No settings.json found. Nothing to uninstall.');
    return;
  }

  const backupPath = `${SETTINGS_PATH}.bak.${Date.now()}`;
  fs.copyFileSync(SETTINGS_PATH, backupPath);
  console.log(`Backed up settings to ${backupPath}`);

  const settings = JSON.parse(fs.readFileSync(SETTINGS_PATH, 'utf8'));

  if (!settings.hooks) {
    console.log('No hooks found in settings. Nothing to uninstall.');
    return;
  }

  let removed = 0;

  for (const hookType of ['PreToolUse', 'PostToolUse']) {
    if (!settings.hooks[hookType]) continue;

    const before = settings.hooks[hookType].length;
    settings.hooks[hookType] = settings.hooks[hookType].filter(entry => {
      if (entry._tag === TAG) return false;
      if (entry.hooks) {
        const ours = entry.hooks.some(h =>
          h.command && (h.command.includes('safety-guard.js') || h.command.includes('safety-learn.js'))
        );
        if (ours) return false;
      }
      return true;
    });
    removed += before - settings.hooks[hookType].length;

    // Clean up empty arrays
    if (settings.hooks[hookType].length === 0) {
      delete settings.hooks[hookType];
    }
  }

  // Clean up empty hooks object
  if (Object.keys(settings.hooks).length === 0) {
    delete settings.hooks;
  }

  fs.writeFileSync(SETTINGS_PATH, JSON.stringify(settings, null, 2) + '\n');

  console.log(`Removed ${removed} hook(s) from settings.json`);
  console.log('\nUninstall complete. Restart Claude Code to apply changes.');
  console.log('Note: Your allowlist was NOT deleted. Remove it manually if desired:');
  console.log(`  ${path.resolve(__dirname, '..', 'hooks', 'safety-allowlist.json')}`);
}

main();
