#!/usr/bin/env node
// Safety Guard - PreToolUse hook
// Auto-approves safe commands, blocks dangerous ones, and lets ambiguous ones
// fall through to the normal permission prompt.
//
// Decision logic:
//   "block"   -> dangerous command, prevent execution
//   "approve" -> safe command, skip permission prompt
//   (exit 0 with no output) -> let normal permission flow handle it
//
// Also checks <install_dir>/safety-allowlist.json for user-learned patterns.
// When falling through, writes the command to a temp file so the PostToolUse
// learning hook can offer to add it to the allowlist.

const fs = require('fs');
const os = require('os');
const path = require('path');

// Allowlist lives next to this script
const ALLOWLIST_PATH = path.join(__dirname, 'safety-allowlist.json');

let input = '';
const stdinTimeout = setTimeout(() => process.exit(0), 3000);
process.stdin.setEncoding('utf8');
process.stdin.on('data', chunk => input += chunk);
process.stdin.on('end', () => {
  clearTimeout(stdinTimeout);
  try {
    const data = JSON.parse(input);
    const toolName = data.tool_name;
    const toolInput = data.tool_input || {};

    // Only inspect Bash commands -- other tools are inherently safe or handled
    // by Claude Code's own permission system
    if (toolName !== 'Bash') {
      approve('Non-bash tool, safe by default');
      return;
    }

    const cmd = (toolInput.command || '').trim();
    if (!cmd) {
      process.exit(0);
      return;
    }

    // --- BLOCK: clearly dangerous commands ---
    const blockPatterns = [
      // Destructive filesystem operations
      { pattern: /\brm\s+(-[a-zA-Z]*f[a-zA-Z]*\s+)?(-[a-zA-Z]*r[a-zA-Z]*\s+)?\s*\/\s*$/i, reason: 'rm targeting root filesystem' },
      { pattern: /\brm\s+.*-[a-zA-Z]*r[a-zA-Z]*f[a-zA-Z]*\s+\/(bin|boot|dev|etc|lib|proc|root|sbin|sys|usr|var|opt|snap|System|Applications|Library)\b/i, reason: 'rm targeting critical system directory' },
      { pattern: /\brm\s+.*-[a-zA-Z]*r[a-zA-Z]*f[a-zA-Z]*\s+~\s*$/i, reason: 'rm -rf targeting home directory' },
      { pattern: /\brm\s+.*-[a-zA-Z]*r[a-zA-Z]*f[a-zA-Z]*\s+\$HOME\b/i, reason: 'rm -rf targeting $HOME' },
      { pattern: /\brm\s+.*-[a-zA-Z]*r[a-zA-Z]*f[a-zA-Z]*\s+\.\s*$/i, reason: 'rm -rf targeting current directory (.)' },
      { pattern: /\brm\s+.*-[a-zA-Z]*r[a-zA-Z]*f[a-zA-Z]*\s+\.\.\s*$/i, reason: 'rm -rf targeting parent directory (..)' },

      // Destructive database commands
      { pattern: /\bDROP\s+(DATABASE|SCHEMA)\b/i, reason: 'DROP DATABASE command' },
      { pattern: /\bDROP\s+TABLE\b/i, reason: 'DROP TABLE command' },
      { pattern: /\bTRUNCATE\s+(TABLE\s+)?\w/i, reason: 'TRUNCATE TABLE command' },
      { pattern: /\bDELETE\s+FROM\s+\w+\s*;/i, reason: 'DELETE FROM without WHERE clause' },

      // Destructive disk operations
      { pattern: /\bmkfs\b/i, reason: 'Filesystem format command' },
      { pattern: /\bdd\s+.*\bof=\/dev\//i, reason: 'dd writing to device' },
      { pattern: />\s*\/dev\/sd[a-z]/i, reason: 'Redirect to raw disk device' },

      // Fork bomb
      { pattern: /:\(\)\s*\{\s*:\|:&?\s*\}\s*;?\s*:/i, reason: 'Fork bomb detected' },

      // Dangerous permission changes
      { pattern: /\bchmod\s+.*-[a-zA-Z]*R[a-zA-Z]*\s+777\s+\//i, reason: 'Recursive chmod 777 on root' },
      { pattern: /\bchown\s+.*-[a-zA-Z]*R[a-zA-Z]*\s+.*\s+\/\s*$/i, reason: 'Recursive chown on root' },

      // Dangerous git operations on main/master
      { pattern: /\bgit\s+push\s+.*--force.*\b(main|master)\b/i, reason: 'Force push to main/master' },
      { pattern: /\bgit\s+push\s+.*\b(main|master)\b.*--force/i, reason: 'Force push to main/master' },

      // Kubernetes destructive commands
      { pattern: /\bkubectl\s+delete\s+(namespace|ns)\b/i, reason: 'kubectl delete namespace' },
      { pattern: /\bkubectl\s+delete\s+.*--all\s*$/i, reason: 'kubectl delete --all' },

      // Dangerous system commands
      { pattern: /\bsudo\s+rm\s+.*-[a-zA-Z]*r[a-zA-Z]*f/i, reason: 'sudo rm -rf' },
      { pattern: /\b(shutdown|reboot|halt|poweroff)\b/i, reason: 'System shutdown/reboot command' },
      { pattern: /\bsystemctl\s+(stop|disable)\s+(docker|sshd|network|firewall)/i, reason: 'Stopping critical system service' },
    ];

    for (const { pattern, reason } of blockPatterns) {
      if (pattern.test(cmd)) {
        block(`BLOCKED: ${reason}. Command: ${cmd}`);
        return;
      }
    }

    // --- ALLOWLIST: user-approved patterns from learning hook ---
    try {
      if (fs.existsSync(ALLOWLIST_PATH)) {
        const allowlist = JSON.parse(fs.readFileSync(ALLOWLIST_PATH, 'utf8'));
        for (const entry of (allowlist.patterns || [])) {
          if (entry.type === 'prefix' && cmd.startsWith(entry.value)) {
            approve(`Allowlisted prefix: "${entry.value}"`);
            return;
          }
          if (entry.type === 'exact' && cmd === entry.value) {
            approve(`Allowlisted exact: "${entry.value}"`);
            return;
          }
          if (entry.type === 'regex' && new RegExp(entry.value).test(cmd)) {
            approve(`Allowlisted regex: "${entry.value}"`);
            return;
          }
        }
      }
    } catch (e) {
      // Ignore corrupt allowlist -- fall through to normal checks
    }

    // --- APPROVE: clearly safe commands ---
    const safePatterns = [
      // Read-only / inspection commands
      /^\s*(cat|head|tail|less|more|wc|file|stat|du|df)\s/,
      /^\s*(ls|ll|la|tree|find|locate|which|whereis|type|command\s+-v)\b/,
      /^\s*(echo|printf|date|whoami|hostname|uname|id|env|printenv)\b/,
      /^\s*(pwd|cd)\b/,
      /^\s*(ps|top|htop|pgrep|lsof|netstat|ss)\b/,
      /^\s*(curl|wget|http)\s/,
      /^\s*(dig|nslookup|host|ping|traceroute)\b/,

      // Version / help
      /^\s*\S+\s+(--version|-v|--help|-h)\s*$/,

      // Build / dev tools (read-only or local)
      /^\s*(go|node|python|python3|ruby|cargo|rustc|javac|java)\s/,
      /^\s*(npm|yarn|pnpm|bun|pip|pip3|gem|cargo)\s+(run|start|test|build|lint|check|list|info|show|outdated|audit|ci|install|add)\b/,
      /^\s*(make|cmake|bazel|gradle|mvn)\s/,
      /^\s*(tsc|eslint|prettier|black|flake8|pylint|mypy|rubocop|golangci-lint|staticcheck)\b/,
      /^\s*(jest|pytest|rspec|mocha|vitest|go\s+test)\b/,
      /^\s*(docker|podman)\s+(ps|images|logs|inspect|stats|top|port|diff|history)\b/,

      // Git read operations
      /^\s*git\s+(status|log|diff|show|branch|tag|remote|stash\s+list|describe|rev-parse|ls-files|shortlog|blame|reflog)\b/,
      /^\s*git\s+(fetch|pull|clone|checkout|switch)\b/,
      /^\s*git\s+(add|commit|stash\s+(push|save|pop|apply|drop))\b/,
      /^\s*git\s+push\b(?!.*--force)/,

      // kubectl read operations
      /^\s*kubectl\s+(get|describe|logs|explain|top|api-resources|api-versions|cluster-info|config)\b/,

      // AWS read operations
      /^\s*aws\s+(sts|s3\s+ls|ec2\s+describe|iam\s+list|iam\s+get|sso\s+login)\b/,

      // Grep / search
      /^\s*(grep|rg|ag|ack|sed\s+-n|awk)\s/,

      // Directory creation (non-destructive)
      /^\s*mkdir\b/,

      // Graphite CLI
      /^\s*gt\s/,

      // Terraform plan (read-only)
      /^\s*terraform\s+(plan|show|state\s+list|validate|fmt|init)\b/,

      // jq / yq processing
      /^\s*(jq|yq)\s/,
    ];

    for (const pattern of safePatterns) {
      if (pattern.test(cmd)) {
        approve('Safe command pattern');
        return;
      }
    }

    // --- FALLTHROUGH: let Claude Code's normal permission prompt handle it ---
    // Write pending command so the PostToolUse learning hook can offer to allowlist it
    try {
      const sessionId = data.session_id || 'unknown';
      const pendingPath = path.join(os.tmpdir(), `claude-safety-pending-${sessionId}.json`);
      fs.writeFileSync(pendingPath, JSON.stringify({
        command: cmd,
        timestamp: Date.now(),
        allowlistPath: ALLOWLIST_PATH
      }));
    } catch (e) {
      // Ignore write failures
    }
    process.exit(0);

  } catch (e) {
    // Silent fail -- never block tool execution on hook errors
    process.exit(0);
  }
});

function block(reason) {
  process.stdout.write(JSON.stringify({ decision: 'block', reason }));
}

function approve(reason) {
  process.stdout.write(JSON.stringify({ decision: 'approve', reason }));
}
