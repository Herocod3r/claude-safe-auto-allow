# claude-safe-auto-allow

Smart safety hooks for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) that **block dangerous commands**, **auto-approve safe ones**, and **learn from your approvals** over time.

Zero dependencies. Pure Node.js. Works on macOS and Linux.

## How it works

```
Command entered
    |
    v
PreToolUse (safety-guard.js)
    |
    |-- Matches block list?  --> BLOCKED (never runs)
    |-- Matches allowlist?   --> AUTO-APPROVED
    |-- Matches safe list?   --> AUTO-APPROVED
    |
    '-- No match --> Normal permission prompt
                         |
                         v
                     User approves
                         |
                         v
                 PostToolUse (safety-learn.js)
                     "Want to always allow this?"
                         |
                         |-- 1. Prefix match (e.g. "docker build")
                         |-- 2. Exact match  (full command)
                         '-- 3. No thanks
```

## Install

### One-liner (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/Herocod3r/claude-safe-auto-allow/main/install.sh | bash
```

### Manual (from clone)

```bash
git clone https://github.com/Herocod3r/claude-safe-auto-allow.git ~/.claude-safe-auto-allow
cd ~/.claude-safe-auto-allow
node scripts/install.js
```

Then **restart Claude Code**.

## Uninstall

```bash
node ~/.claude-safe-auto-allow/scripts/uninstall.js
rm -rf ~/.claude-safe-auto-allow
```

Or use the uninstall script:

```bash
bash ~/.claude-safe-auto-allow/uninstall.sh
```

## What gets blocked

Commands that are **always blocked** regardless of allowlist:

| Category | Examples |
|----------|----------|
| Destructive filesystem | `rm -rf /`, `rm -rf ~`, `rm -rf $HOME`, `sudo rm -rf` |
| Critical system dirs | `rm -rf /etc`, `/usr`, `/bin`, `/System`, `/Library` |
| Database destruction | `DROP DATABASE`, `DROP TABLE`, `TRUNCATE`, `DELETE FROM` without `WHERE` |
| Disk operations | `mkfs`, `dd of=/dev/...`, redirect to raw device |
| Fork bombs | `:(){ :\|:& };:` |
| Permission nuking | `chmod -R 777 /`, `chown -R ... /` |
| Force push protected | `git push --force origin main/master` |
| Kubernetes destruction | `kubectl delete namespace`, `kubectl delete --all` |
| System commands | `shutdown`, `reboot`, `halt`, `poweroff` |
| Critical services | `systemctl stop docker/sshd/network` |

## What gets auto-approved

Commands that are **always approved** (no prompt):

| Category | Examples |
|----------|----------|
| Read-only | `ls`, `cat`, `head`, `tail`, `find`, `stat`, `du`, `df` |
| Inspection | `ps`, `top`, `lsof`, `netstat`, `whoami`, `uname` |
| Network (read) | `curl`, `wget`, `dig`, `ping` |
| Git (safe) | `status`, `log`, `diff`, `add`, `commit`, `push` (non-force), `fetch`, `pull` |
| Build/test | `npm test`, `go test`, `make`, `pytest`, `jest`, `cargo build` |
| Linters | `eslint`, `prettier`, `mypy`, `golangci-lint` |
| K8s (read) | `kubectl get`, `describe`, `logs`, `explain` |
| AWS (read) | `aws sts`, `aws s3 ls`, `aws ec2 describe` |
| Search | `grep`, `rg`, `ag`, `awk` |
| Terraform (read) | `plan`, `show`, `validate`, `fmt`, `init` |
| Utilities | `mkdir`, `jq`, `yq`, `echo`, `date` |

## The learning allowlist

When a command falls through to the manual prompt and you approve it, Claude will ask:

> Would you like me to add this to your safety allowlist so it's auto-approved next time?
> 1. **Prefix match** -- always allow commands starting with `docker build`
> 2. **Exact match** -- only allow this exact command
> 3. **No** -- don't add it

Your choices are saved to `hooks/safety-allowlist.json`:

```json
{
  "patterns": [
    { "type": "prefix", "value": "docker build" },
    { "type": "exact", "value": "terraform apply -auto-approve" },
    { "type": "regex", "value": "^ansible-playbook\\s" }
  ]
}
```

The allowlist is checked **after** the block list, so you can never accidentally allowlist a dangerous command.

## Tests

```bash
npm test
# or
node tests/safety-guard.test.js
```

## Configuration

### Adding custom block patterns

Edit `hooks/safety-guard.js` and add entries to the `blockPatterns` array:

```js
{ pattern: /\bmy-dangerous-command\b/i, reason: 'Custom block reason' },
```

### Adding custom safe patterns

Add entries to the `safePatterns` array:

```js
/^\s*my-safe-tool\s/,
```

### Editing the allowlist directly

Edit `hooks/safety-allowlist.json`. Supported types:
- `prefix` -- matches if command starts with the value
- `exact` -- matches if command equals the value exactly
- `regex` -- matches if command matches the regex

## Requirements

- Node.js >= 16
- Claude Code

## License

MIT
