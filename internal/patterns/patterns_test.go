package patterns

import (
	"testing"

	"github.com/Herocod3r/claude-safe-auto-allow/internal/shell"
)

// parseFirst is a test helper that parses a shell string and returns the first
// Command. For sudo-wrapped commands it returns the sudo Command so callers
// can test IsDangerous (which unwraps sudo) rather than CheckDestructiveRm.
func parseFirst(t *testing.T, s string) shell.Command {
	t.Helper()
	result := shell.Parse(s)
	if result.ParseError {
		t.Fatalf("parseFirst(%q): parse error", s)
	}
	if len(result.Commands) == 0 {
		t.Fatalf("parseFirst(%q): no commands found", s)
	}
	return result.Commands[0]
}

func TestCheckDestructiveRm(t *testing.T) {
	// These are all rm-level commands (no sudo wrapper).
	// sudo rm cases are tested via TestIsDangerous.
	dangerous := []string{
		"rm -rf /",
		"rm -rf ~",
		"rm -rf $HOME",
		"rm -rf .",
		"rm -rf ..",
		"rm -rf /*",
		"rm -r -f /",
		"rm --recursive --force /",
		"/bin/rm -rf /",
		"/usr/bin/rm -rf /",
		"rm -rf /etc",
		"rm -rf /System",
		"rm -rf /Library",
	}
	for _, s := range dangerous {
		cmd := parseFirst(t, s)
		// For path-prefixed rm (/bin/rm) the parsed name includes the path.
		if result := CheckDestructiveRm(cmd); result == "" {
			t.Errorf("CheckDestructiveRm(%q) should be dangerous, got empty", s)
		}
	}

	safe := []string{
		"rm file.txt",
		"rm -f file.txt",
		"rm -r /tmp/mydir",
	}
	for _, s := range safe {
		cmd := parseFirst(t, s)
		if result := CheckDestructiveRm(cmd); result != "" {
			t.Errorf("CheckDestructiveRm(%q) = %q, want empty", s, result)
		}
	}
}

func TestIsDangerous(t *testing.T) {
	dangerous := []string{
		`mysql -e "DROP DATABASE production"`,
		`psql -c "DROP TABLE users"`,
		`mysql -e "TRUNCATE TABLE logs"`,
		`psql -c "DELETE FROM users;"`,
		"mkfs.ext4 /dev/sda1",
		"dd if=/dev/zero of=/dev/sda bs=1M",
		"chmod -R 777 /",
		"git push --force origin main",
		"kubectl delete namespace production",
		"shutdown -h now",
		"reboot",
		// sudo unwrapping
		"sudo rm -rf /",
		"sudo shutdown -h now",
	}
	for _, s := range dangerous {
		cmd := parseFirst(t, s)
		if !IsDangerous(cmd) {
			t.Errorf("IsDangerous(%q) = false, want true", s)
		}
	}

	safe := []string{
		"ls -la",
		"git status",
		"npm test",
		"grep reboot /var/log/syslog",
		"cat shutdown.log",
		"docker run ubuntu",
	}
	for _, s := range safe {
		cmd := parseFirst(t, s)
		if IsDangerous(cmd) {
			t.Errorf("IsDangerous(%q) = true, want false", s)
		}
	}
}

func TestIsDangerousKubectlResources(t *testing.T) {
	dangerous := []string{
		"kubectl delete namespace production",
		"kubectl delete ns staging",
		"kubectl delete pv my-volume",
		"kubectl delete pvc my-claim",
		"kubectl delete storageclass fast",
		"kubectl delete clusterrole admin",
		"kubectl delete clusterrolebinding admin-binding",
		"kubectl delete pods --all",
		"kubectl delete deployments -A",
	}
	for _, s := range dangerous {
		cmd := parseFirst(t, s)
		if !IsDangerous(cmd) {
			t.Errorf("IsDangerous(%q) = false, want true", s)
		}
	}

	// Deleting a single pod or deployment by name is not considered dangerous.
	safe := []string{
		"kubectl delete pod my-pod",
		"kubectl delete deployment old-app",
	}
	for _, s := range safe {
		cmd := parseFirst(t, s)
		if IsDangerous(cmd) {
			t.Errorf("IsDangerous(%q) = true, want false", s)
		}
	}
}

func TestIsDangerousChmod(t *testing.T) {
	dangerous := []string{
		"chmod -R 777 /",
		"chmod -R 777 /etc",
		"chmod -R 000 /usr",
		"chmod -R o+w /",
		"chmod -R a+w /etc",
		"chmod -R 0777 /bin",
	}
	for _, s := range dangerous {
		cmd := parseFirst(t, s)
		if !IsDangerous(cmd) {
			t.Errorf("IsDangerous(%q) = false, want true", s)
		}
	}

	safe := []string{
		"chmod 755 script.sh",
		"chmod -R 755 ./dist",
		"chmod -R 777 /tmp/mydir", // /tmp is not a critical dir
	}
	for _, s := range safe {
		cmd := parseFirst(t, s)
		if IsDangerous(cmd) {
			t.Errorf("IsDangerous(%q) = true, want false", s)
		}
	}
}

func TestIsDangerousRedirect(t *testing.T) {
	dangerous := []shell.Redirect{
		{Target: "/dev/sda"},
		{Target: "/dev/sdb1"},
		{Target: "/dev/nvme0n1"},
		{Target: "/dev/vda"},
	}
	for _, r := range dangerous {
		if !IsDangerousRedirect(r) {
			t.Errorf("IsDangerousRedirect(%q) = false, want true", r.Target)
		}
	}

	safe := []shell.Redirect{
		{Target: "/dev/null"},
		{Target: "/tmp/out.txt"},
		{Target: ""},
	}
	for _, r := range safe {
		if IsDangerousRedirect(r) {
			t.Errorf("IsDangerousRedirect(%q) = true, want false", r.Target)
		}
	}
}
