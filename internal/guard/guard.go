package guard

import (
	"strings"

	"github.com/Herocod3r/claude-safe-auto-allow/internal/allowlist"
	"github.com/Herocod3r/claude-safe-auto-allow/internal/patterns"
	"github.com/Herocod3r/claude-safe-auto-allow/internal/shell"
)

type Result struct {
	Decision  string
	Reason    string
	Dangerous bool
}

// Evaluate decides whether a tool invocation should be approved, flagged as
// dangerous, or fall through to Claude's normal permission prompt.
//
// cwd is the working directory used to resolve git-repo scoped allowlist
// entries. Pass "" to skip git-scope matching.
func Evaluate(toolName, command, allowlistPath, cwd string) Result {
	if toolName != "Bash" {
		return Result{Decision: "approve", Reason: "Non-bash tool, safe by default"}
	}

	command = strings.TrimSpace(command)
	if command == "" {
		return Result{}
	}

	parsed := shell.Parse(command)

	// Unparseable shell: can't verify safety, require manual approval.
	if parsed.ParseError {
		return Result{}
	}

	// Fork bomb: :(){ :|:& };: defines a function named ":" that recursively forks.
	if parsed.HasForkBomb {
		return Result{Dangerous: true}
	}

	// Check every command node in the AST (including those inside pipelines,
	// conditionals, and substitutions) for known dangerous patterns.
	for _, cmd := range parsed.Commands {
		if patterns.IsDangerous(cmd) {
			return Result{Dangerous: true}
		}
	}

	// Check for redirects that write to block devices (e.g. > /dev/sda).
	for _, redir := range parsed.Redirects {
		if patterns.IsDangerousRedirect(redir) {
			return Result{Dangerous: true}
		}
	}

	// Command substitutions produce values only known at runtime; we cannot
	// statically verify their safety so require manual approval.
	// This check must precede the allowlist so that a prefix-matched entry
	// like "docker build" cannot be used to auto-approve "docker build $(evil)".
	if parsed.HasSubstitution {
		return Result{}
	}

	// Load the allowlist (seeding from embedded defaults on first run if path
	// is provided but the file does not exist). When path is empty the embedded
	// defaults are used in-memory so safe commands still get auto-approved.
	// cwd is forwarded so git-repo scoped entries are evaluated correctly.
	al, _ := allowlist.LoadOrSeed(allowlistPath)
	if al.Match(command, cwd) {
		return Result{Decision: "approve", Reason: "Allowlisted"}
	}

	return Result{}
}
