#!/usr/bin/env node
// Safety Learn - PostToolUse hook
// After a Bash command executes that wasn't auto-approved or blocked (i.e., the
// user had to manually approve it), this hook injects a message asking if they
// want to always allow that command pattern in the future.
//
// Works in tandem with safety-guard.js (PreToolUse), which writes a "pending"
// file when a command falls through to manual approval.
//
// The allowlist is stored alongside safety-guard.js as safety-allowlist.json.

const fs = require('fs');
const os = require('os');
const path = require('path');

let input = '';
const stdinTimeout = setTimeout(() => process.exit(0), 3000);
process.stdin.setEncoding('utf8');
process.stdin.on('data', chunk => input += chunk);
process.stdin.on('end', () => {
  clearTimeout(stdinTimeout);
  try {
    const data = JSON.parse(input);
    const toolName = data.tool_name;

    if (toolName !== 'Bash') {
      process.exit(0);
      return;
    }

    const sessionId = data.session_id || 'unknown';
    const pendingPath = path.join(os.tmpdir(), `claude-safety-pending-${sessionId}.json`);

    if (!fs.existsSync(pendingPath)) {
      process.exit(0);
      return;
    }

    const pending = JSON.parse(fs.readFileSync(pendingPath, 'utf8'));

    // Clean up the pending file
    try { fs.unlinkSync(pendingPath); } catch (e) {}

    // Only act on recent pending commands (within 60s)
    if (Date.now() - pending.timestamp > 60000) {
      process.exit(0);
      return;
    }

    const cmd = pending.command;
    const allowlistPath = pending.allowlistPath || path.join(__dirname, 'safety-allowlist.json');

    // Extract a sensible prefix suggestion (first word or first two words)
    const parts = cmd.trim().split(/\s+/);
    let suggestedPrefix = parts[0];
    if (parts.length > 1) {
      suggestedPrefix = parts.slice(0, 2).join(' ');
    }

    const message =
      `The command \`${cmd}\` required manual approval because it wasn't in the safe list or your allowlist. ` +
      `Ask the user: "Would you like me to add this to your safety allowlist so it's auto-approved next time? ` +
      `Options:\\n` +
      `1. **Prefix match** -- always allow commands starting with \`${suggestedPrefix}\`\\n` +
      `2. **Exact match** -- only allow this exact command: \`${cmd}\`\\n` +
      `3. **No** -- don't add it"\\n\\n` +
      `If they choose an option, update the allowlist at ${allowlistPath} using the Write tool. ` +
      `The format is: { "patterns": [{ "type": "prefix"|"exact"|"regex", "value": "..." }] }. ` +
      `Read the file first if it exists, then append the new entry.`;

    const output = {
      hookSpecificOutput: {
        hookEventName: 'PostToolUse',
        additionalContext: message
      }
    };

    process.stdout.write(JSON.stringify(output));
  } catch (e) {
    process.exit(0);
  }
});
