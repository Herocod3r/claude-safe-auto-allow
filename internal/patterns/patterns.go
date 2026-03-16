package patterns

import (
	"regexp"
	"strings"

	"github.com/Herocod3r/claude-safe-auto-allow/internal/shell"
)

// IsDangerous reports whether cmd matches known dangerous command patterns.
// sudo/doas are automatically unwrapped so the real subcommand is checked.
func IsDangerous(cmd shell.Command) bool {
	if cmd.Name == "" {
		return false
	}

	if isSudoLike(cmd.Name) {
		return IsDangerous(unwrapSudo(cmd))
	}

	if CheckDestructiveRm(cmd) != "" {
		return true
	}

	switch baseName(cmd.Name) {
	case "mkfs", "mkfs.ext4", "mkfs.vfat", "mkfs.xfs", "mkfs.btrfs", "mkfs.ntfs":
		return true
	case "dd":
		return isDangerousDD(cmd)
	case "chmod":
		return isDangerousChmod(cmd)
	case "chown":
		return isDangerousChown(cmd)
	case "git":
		return isDangerousGit(cmd)
	case "kubectl":
		return isDangerousKubectl(cmd)
	case "shutdown", "reboot", "halt", "poweroff":
		return true
	case "systemctl":
		return isDangerousSystemctl(cmd)
	case "mysql", "psql", "sqlite3":
		return isSQLDangerous(cmd)
	}

	return false
}

// IsDangerousRedirect reports whether a redirect targets a block device or
// other path that should never be written to by an automated tool.
func IsDangerousRedirect(r shell.Redirect) bool {
	t := r.Target
	if t == "" {
		return false
	}
	return strings.HasPrefix(t, "/dev/sd") ||
		strings.HasPrefix(t, "/dev/nvme") ||
		strings.HasPrefix(t, "/dev/vd") ||
		strings.HasPrefix(t, "/dev/hd")
}

func isSudoLike(name string) bool {
	return name == "sudo" || name == "doas"
}

// unwrapSudo returns a new Command for the real subcommand inside sudo/doas,
// skipping any leading flags. HasNonLiteralArg is propagated from the outer
// command so that e.g. "sudo rm -rf $HOME" is still caught.
func unwrapSudo(cmd shell.Command) shell.Command {
	for i, arg := range cmd.Args {
		if !strings.HasPrefix(arg, "-") {
			return shell.Command{Name: arg, Args: cmd.Args[i+1:], HasNonLiteralArg: cmd.HasNonLiteralArg}
		}
	}
	return shell.Command{}
}

// baseName strips any directory prefix so "/bin/rm" → "rm".
func baseName(name string) string {
	if i := strings.LastIndex(name, "/"); i >= 0 {
		return name[i+1:]
	}
	return name
}

func isDangerousDD(cmd shell.Command) bool {
	for _, arg := range cmd.Args {
		if strings.HasPrefix(strings.ToLower(arg), "of=/dev/") {
			return true
		}
	}
	return false
}

// dangerousChmodModes contains mode strings that grant excessive permissions
// or remove all permissions from a path when applied recursively.
var dangerousChmodModes = map[string]bool{
	"777": true, "0777": true, "a+rwx": true,
	"o+w": true, "a+w": true, "ugo+w": true,
	"000": true, "0000": true,
}

func isDangerousChmod(cmd shell.Command) bool {
	hasRecursive := false
	var modeArg string
	var targets []string

	for _, arg := range cmd.Args {
		switch {
		case arg == "-R" || arg == "--recursive" || (isShortFlags(arg) && strings.ContainsRune(arg, 'R')):
			hasRecursive = true
		case !strings.HasPrefix(arg, "-") && modeArg == "":
			modeArg = arg // first non-flag positional is the mode
		case !strings.HasPrefix(arg, "-"):
			targets = append(targets, arg)
		}
	}

	if !hasRecursive || !dangerousChmodModes[modeArg] {
		return false
	}

	for _, target := range targets {
		if target == "/" || target == "/." || matchesCriticalDir(target) {
			return true
		}
	}
	return false
}

func isDangerousChown(cmd shell.Command) bool {
	hasRecursive, targetsRoot := false, false
	for _, arg := range cmd.Args {
		switch {
		case arg == "-R" || arg == "--recursive" || (isShortFlags(arg) && strings.ContainsRune(arg, 'R')):
			hasRecursive = true
		case arg == "/" || arg == "/.":
			targetsRoot = true
		}
	}
	return hasRecursive && targetsRoot
}

func isDangerousGit(cmd shell.Command) bool {
	if len(cmd.Args) == 0 || cmd.Args[0] != "push" {
		return false
	}
	hasForce, hasProtected := false, false
	for _, arg := range cmd.Args[1:] {
		switch arg {
		case "--force", "-f", "--force-with-lease":
			hasForce = true
		case "main", "master":
			hasProtected = true
		default:
			// Detect refspec force-push syntax: +<src>:<dst> or +<branch>
			// e.g. "git push origin +main" or "git push origin +HEAD:main"
			if strings.HasPrefix(arg, "+") && arg != "+" {
				hasForce = true
				// Check if the refspec targets a protected branch
				ref := arg[1:]
				if i := strings.LastIndex(ref, ":"); i >= 0 {
					ref = ref[i+1:]
				}
				if ref == "main" || ref == "master" {
					hasProtected = true
				}
			}
		}
	}
	return hasForce && hasProtected
}

// dangerousKubectlResources are resource types whose deletion has a large
// blast radius (entire namespace, cluster-wide RBAC, persistent storage).
var dangerousKubectlResources = map[string]bool{
	// Namespace-scoped: wipes everything in a namespace
	"namespace": true, "ns": true,
	// Cluster-wide storage
	"persistentvolume": true, "pv": true,
	"persistentvolumeclaim": true, "pvc": true,
	"storageclass": true,
	// Cluster-wide RBAC
	"clusterrole": true, "clusterrolebinding": true,
}

func isDangerousKubectl(cmd shell.Command) bool {
	if len(cmd.Args) == 0 || cmd.Args[0] != "delete" {
		return false
	}
	for _, arg := range cmd.Args[1:] {
		if arg == "--all" || arg == "-A" {
			return true
		}
		if dangerousKubectlResources[strings.ToLower(arg)] {
			return true
		}
	}
	return false
}

func isDangerousSystemctl(cmd shell.Command) bool {
	if len(cmd.Args) == 0 {
		return false
	}
	switch cmd.Args[0] {
	case "stop", "disable":
	default:
		return false
	}
	critical := map[string]bool{
		"docker": true, "sshd": true, "ssh": true,
		"network": true, "networking": true,
		"firewall": true, "firewalld": true, "ufw": true,
	}
	for _, arg := range cmd.Args[1:] {
		if critical[arg] {
			return true
		}
	}
	return false
}

// sqlDangerousRe matches destructive SQL statements that may appear as literal
// arguments to database CLI tools (mysql -e "...", psql -c "...").
var sqlDangerousRe = regexp.MustCompile(
	`(?i)\b(DROP\s+(DATABASE|SCHEMA|TABLE)|TRUNCATE(\s+TABLE)?\s+\w|DELETE\s+FROM\s+\S+\s*(;|\s*$))`,
)

func isSQLDangerous(cmd shell.Command) bool {
	for _, arg := range cmd.Args {
		if sqlDangerousRe.MatchString(arg) {
			return true
		}
	}
	return false
}

// isShortFlags reports whether s is a combined short-flag token like "-rf".
func isShortFlags(s string) bool {
	return len(s) >= 2 && s[0] == '-' && s[1] != '-'
}
