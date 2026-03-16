package allowlist

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchPrefix(t *testing.T) {
	al := &Allowlist{Patterns: []Pattern{{Type: "prefix", Value: "docker build"}}}
	if !al.Match("docker build -t app .", "") {
		t.Fatal("expected prefix match")
	}
	if al.Match("docker run ubuntu", "") {
		t.Fatal("unexpected prefix match")
	}
}

func TestMatchExact(t *testing.T) {
	al := &Allowlist{Patterns: []Pattern{{Type: "exact", Value: "terraform apply -auto-approve"}}}
	if !al.Match("terraform apply -auto-approve", "") {
		t.Fatal("expected exact match")
	}
	if al.Match("terraform apply", "") {
		t.Fatal("unexpected exact match")
	}
}

func TestMatchRegex(t *testing.T) {
	al := &Allowlist{Patterns: []Pattern{{Type: "regex", Value: `^ansible-playbook\s`}}}
	if !al.Match("ansible-playbook deploy.yml", "") {
		t.Fatal("expected regex match")
	}
	if al.Match("ansible-galaxy install role", "") {
		t.Fatal("unexpected regex match")
	}
}

func TestInvalidRegexSkipped(t *testing.T) {
	al := &Allowlist{Patterns: []Pattern{{Type: "regex", Value: "(invalid["}}}
	if al.Match("anything", "") {
		t.Fatal("invalid regex should not match")
	}
}

func TestMatchGitRepoScopeApproves(t *testing.T) {
	repoDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	al := &Allowlist{Patterns: []Pattern{{Type: "prefix", Scope: "git-repo", Value: "mv "}}}
	if !al.Match("mv old.go new.go", repoDir) {
		t.Fatal("expected git-repo scoped match inside repo")
	}
}

func TestMatchGitRepoScopeFallthroughOutsideRepo(t *testing.T) {
	nonRepo := t.TempDir() // no .git

	al := &Allowlist{Patterns: []Pattern{{Type: "prefix", Scope: "git-repo", Value: "mv "}}}
	if al.Match("mv old.go new.go", nonRepo) {
		t.Fatal("git-repo scoped entry should not match outside a git repo")
	}
}

func TestMatchGitRepoScopeSubdirectory(t *testing.T) {
	// .git is at the root; cwd is a nested subdirectory.
	repoDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	subDir := filepath.Join(repoDir, "pkg", "foo")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("mkdirall: %v", err)
	}

	al := &Allowlist{Patterns: []Pattern{{Type: "prefix", Scope: "git-repo", Value: "cp "}}}
	if !al.Match("cp a.go b.go", subDir) {
		t.Fatal("expected match from subdirectory of git repo")
	}
}

func TestMatchGitRepoScopeEmptyCwd(t *testing.T) {
	al := &Allowlist{Patterns: []Pattern{{Type: "prefix", Scope: "git-repo", Value: "mv "}}}
	if al.Match("mv a b", "") {
		t.Fatal("git-repo scoped entry should not match when cwd is empty")
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "allowlist.json")
	if err := os.WriteFile(path, []byte(`{"patterns":[{"type":"prefix","value":"npm"}]}`), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	al, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if !al.Match("npm test", "") {
		t.Fatal("expected match after load")
	}
}

func TestLoadMissingFile(t *testing.T) {
	al, err := Load("/nonexistent/path.json")
	if err != nil {
		t.Fatalf("Load should not error on missing file: %v", err)
	}
	if al.Match("anything", "") {
		t.Fatal("empty allowlist should not match")
	}
}

func TestLoadCorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "allowlist.json")
	if err := os.WriteFile(path, []byte(`not json {{{`), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	al, err := Load(path)
	if err != nil {
		t.Fatalf("Load should not error on corrupt file: %v", err)
	}
	if al.Match("anything", "") {
		t.Fatal("corrupt allowlist should not match")
	}
}
