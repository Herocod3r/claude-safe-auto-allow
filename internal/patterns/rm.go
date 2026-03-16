package patterns

import (
	"strings"

	"github.com/Herocod3r/claude-safe-auto-allow/internal/shell"
)

var criticalDirs = []string{
	"/bin", "/boot", "/dev", "/etc", "/lib", "/proc",
	"/root", "/sbin", "/sys", "/usr", "/var", "/opt", "/snap",
	// macOS
	"/System", "/Applications", "/Library",
}

// CheckDestructiveRm returns a non-empty reason string if cmd is a dangerous
// recursive rm invocation, or empty string if it appears safe.
func CheckDestructiveRm(cmd shell.Command) string {
	if !isRmCommand(cmd.Name) {
		return ""
	}

	var flags, targets []string
	for _, arg := range cmd.Args {
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
		} else {
			targets = append(targets, arg)
		}
	}

	if !hasRecursiveFlag(flags) {
		return ""
	}

	hasForce := hasForceFlag(flags)

	// Conservative: rm -rf with any unresolvable argument (variable expansion,
	// arithmetic, etc.) cannot be statically verified as safe.
	if cmd.HasNonLiteralArg {
		if hasForce {
			return "recursive force rm with unresolved argument"
		}
		return "recursive rm with unresolved argument"
	}

	for _, target := range targets {
		if isDangerousTarget(target) {
			if hasForce {
				return "recursive force delete of critical path"
			}
			return "recursive delete of critical path"
		}
		if matchesCriticalDir(target) {
			if hasForce {
				return "recursive force delete of system directory"
			}
			return "recursive delete of system directory"
		}
	}

	return ""
}

func isRmCommand(name string) bool {
	return name == "rm" || strings.HasSuffix(name, "/rm")
}

func hasRecursiveFlag(flags []string) bool {
	for _, f := range flags {
		if f == "--recursive" {
			return true
		}
		if isShortFlags(f) && (strings.ContainsRune(f, 'r') || strings.ContainsRune(f, 'R')) {
			return true
		}
	}
	return false
}

func hasForceFlag(flags []string) bool {
	for _, f := range flags {
		if f == "--force" {
			return true
		}
		if isShortFlags(f) && strings.ContainsRune(f, 'f') {
			return true
		}
	}
	return false
}

var dangerousTargets = []string{"/", "~", "$HOME", ".", ".."}

func isDangerousTarget(t string) bool {
	for _, d := range dangerousTargets {
		if t == d || t == d+"/" {
			return true
		}
	}
	return t == "/*" || t == "~/*"
}

func matchesCriticalDir(t string) bool {
	// Normalize trailing slash, but preserve lone "/" as root.
	norm := strings.TrimRight(t, "/")
	if norm == "" {
		norm = "/"
	}
	for _, d := range criticalDirs {
		if norm == d || strings.HasPrefix(norm, d+"/") {
			return true
		}
	}
	return false
}
