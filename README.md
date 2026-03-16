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
    |-- AST parse the command
    |
    |-- Dangerous pattern?   --> Claude permission prompt (never auto-approved)
    |-- Command substitution? -> Claude permission prompt (can't verify statically)
    |-- Matches allowlist?   --> AUTO-APPROVED
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

Commands are parsed using a real shell AST (not regex), so dangerous patterns inside pipelines, conditionals, and compound statements are caught correctly.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/Herocod3r/claude-safe-auto-allow/main/install.sh | bash
```

The installer downloads the latest release binary for your OS and architecture, writes it to `~/.claude-safe-auto-allow/claude-safety`, seeds the default allowlist, and patches `~/.claude/settings.json`.

Restart Claude Code after installation.

## Uninstall

```bash
bash <(curl -fsSL https://raw.githubusercontent.com/Herocod3r/claude-safe-auto-allow/main/uninstall.sh)
```

The uninstaller removes the hook entries from `~/.claude/settings.json` and optionally deletes `~/.claude-safe-auto-allow/`.

## Safety model

Dangerous commands are never auto-approved and never auto-learned. They fall through to Claude's built-in permission prompt so the user retains final control.

Commands containing `$()` or backtick substitutions always fall through — their arguments can't be verified statically.

Examples of dangerous commands:

| Category | Examples |
|----------|----------|
| Destructive filesystem | `rm -rf /`, `rm -rf ~`, `rm -rf .`, `rm -rf /*` |
| Critical system dirs | `rm -rf /etc`, `/usr`, `/bin`, `/System`, `/Library` |
| Database destruction | `DROP DATABASE`, `DROP TABLE`, `TRUNCATE`, `DELETE FROM` without `WHERE` |
| Disk operations | `mkfs`, `dd of=/dev/...`, redirect to raw block device |
| Fork bombs | `:(){ :\|:& };:` |
| Permission nuking | `chmod -R 777 /`, `chown -R ... /` |
| Force push protected | `git push --force origin main/master`, `git push origin +main` |
| Kubernetes destruction | `kubectl delete namespace`, `kubectl delete --all` |
| System commands | `shutdown`, `reboot`, `halt`, `poweroff` |
| Critical services | `systemctl stop docker/sshd/network` |

Examples of safe commands that are auto-approved:

| Category | Examples |
|----------|----------|
| Read-only filesystem | `ls`, `cat`, `head`, `tail`, `find`, `stat`, `du`, `df` |
| Inspection | `ps`, `lsof`, `netstat`, `whoami`, `uname` |
| Search | `grep`, `rg`, `ag`, `sed -n` |
| Git (safe) | `status`, `log`, `diff`, `add`, `commit`, `push`, `fetch`, `pull` |
| File mutation (git repos only) | `mv`, `cp`, `touch` — only inside a git-tracked directory |
| Build / test | `npm test`, `go test`, `make`, `pytest`, `jest`, `cargo build` |
| Runtimes | `node`, `python3`, `ruby`, `deno`, `bun`, `swift`, `dart` |
| Package managers | `npm`, `yarn`, `pnpm`, `pip`, `gem`, `cargo`, `composer` |
| K8s (read) | `kubectl get`, `describe`, `logs`, `explain`, `apply`, `port-forward` |
| Cloud CLIs | `aws sts/s3/ec2`, `gcloud`, `az`, `helm`, `doctl` |
| Terraform (safe) | `plan`, `show`, `validate`, `fmt`, `init` |
| Utilities | `mkdir`, `jq`, `yq`, `echo`, `date`, `open` |

### git-repo scoped entries

`mv`, `cp`, and `touch` are auto-approved only when Claude is operating inside a git repository (checked by walking up the directory tree for a `.git` entry). Outside a repo they fall through to the normal prompt.

## Default allowlist

On first run, `claude-safety guard` seeds `~/.claude-safe-auto-allow/safety-allowlist.json` with 240+ safe command patterns embedded in the binary. You can edit this file to add, remove, or tighten entries — changes take effect immediately without restarting.

## Learning allowlist

When a command falls through to the normal prompt and you approve it, Claude can offer to save it for next time as a prefix or exact match.

The allowlist at `~/.claude-safe-auto-allow/safety-allowlist.json` supports three pattern types:

```json
{
  "patterns": [
    { "type": "prefix", "value": "docker build" },
    { "type": "exact",  "value": "terraform apply -auto-approve" },
    { "type": "regex",  "value": "^ansible-playbook\\s" }
  ]
}
```

Dangerous commands never bypass the danger check, even if the allowlist contains a matching entry.

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
- Go 1.21+ only if building from source

## License

MIT
