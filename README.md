# claude-safe-auto-allow

Smart safety hooks for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) that auto-approve safe commands, let dangerous commands fall through to Claude's normal permission prompt, and learn from commands you choose to approve.

Zero runtime dependencies. Single binary. Works on macOS and Linux.

## How it works

```
Command entered
    |
    v
PreToolUse (claude-safety guard)
    |
    |-- Matches allowlist?   --> AUTO-APPROVED
    |-- Matches safe list?   --> AUTO-APPROVED
    |-- Matches dangerous?   --> Claude permission prompt
    |
    '-- No match -----------> Claude permission prompt
                                  |
                                  v
                              User approves
                                  |
                                  v
                      PostToolUse (claude-safety learn)
                          "Want to always allow this?"
```

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/Herocod3r/claude-safe-auto-allow/main/install.sh | bash
```

The installer downloads the latest release binary for your OS and architecture, writes it to `~/.claude-safe-auto-allow/claude-safety`, migrates any old allowlist, and patches `~/.claude/settings.json`.

Restart Claude Code after installation.

## Uninstall

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/Herocod3r/claude-safe-auto-allow/main/uninstall.sh)
```

The uninstaller removes the hook entries from `~/.claude/settings.json` and optionally deletes `~/.claude-safe-auto-allow/`.

## Safety model

Dangerous commands are not auto-approved and are not auto-learned. They fall through to Claude's built-in permission prompt so the user still has final control.

Examples of dangerous commands:

| Category | Examples |
|----------|----------|
| Destructive filesystem | `rm -rf /`, `rm -rf ~`, `rm -rf .`, `rm -rf ..`, `rm -rf /*` |
| Critical system dirs | `rm -rf /etc`, `/usr`, `/bin`, `/System`, `/Library` |
| Database destruction | `DROP DATABASE`, `DROP TABLE`, `TRUNCATE`, `DELETE FROM` without `WHERE` |
| Disk operations | `mkfs`, `dd of=/dev/...`, redirect to raw device |
| Fork bombs | `:(){ :\|:& };:` |
| Permission nuking | `chmod -R 777 /`, `chown -R ... /` |
| Force push protected | `git push --force origin main/master` |
| Kubernetes destruction | `kubectl delete namespace`, `kubectl delete --all` |
| System commands | `shutdown`, `reboot`, `halt`, `poweroff` |
| Critical services | `systemctl stop docker/sshd/network` |

Examples of safe commands that are auto-approved:

| Category | Examples |
|----------|----------|
| Read-only | `ls`, `cat`, `head`, `tail`, `find`, `stat`, `du`, `df` |
| Inspection | `ps`, `top`, `lsof`, `netstat`, `whoami`, `uname` |
| Network (read) | `curl` GETs, `wget` without output or POST flags, `dig`, `ping` |
| Git (safe) | `status`, `log`, `diff`, `add`, `commit`, `push` without `--force`, `fetch`, `pull` |
| Build/test | `npm test`, `go test`, `make`, `pytest`, `jest`, `cargo build` |
| Interpreters | `node script.js`, `python3 script.py`, `ruby script.rb` |
| K8s (read) | `kubectl get`, `describe`, `logs`, `explain` |
| AWS (read) | `aws sts`, `aws s3 ls`, `aws ec2 describe` |
| Search | `grep`, `rg`, `ag`, `sed -n` |
| Terraform (read) | `plan`, `show`, `validate`, `fmt`, `init` |
| Utilities | `mkdir`, `jq`, `yq`, `echo`, `date` |

## Learning allowlist

When a command falls through to the normal prompt and you approve it, Claude can offer to save it for next time as a prefix or exact match.

The allowlist lives at `~/.claude-safe-auto-allow/safety-allowlist.json`:

```json
{
  "patterns": [
    { "type": "prefix", "value": "docker build" },
    { "type": "exact", "value": "terraform apply -auto-approve" },
    { "type": "regex", "value": "^ansible-playbook\\s" }
  ]
}
```

Dangerous commands never bypass the dangerous check, even if the allowlist contains a matching entry.

## Building from source

```bash
go test ./...
go build -o claude-safety ./cmd/claude-safety
```

The binary looks for `safety-allowlist.json` and `version.txt` in the same directory as the executable.

## Tests

```bash
go test ./...
bash tests/test-install-e2e.sh
```

## Requirements

- Claude Code
- No runtime dependencies for end users
- Go 1.21+ only if you are building from source

## License

MIT
