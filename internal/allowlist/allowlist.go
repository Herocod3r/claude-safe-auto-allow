package allowlist

import (
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed default-allowlist.json
var defaultAllowlistJSON []byte

// Pattern is a single allowlist rule. Type controls how Value is matched
// against the command string. Scope restricts when the rule is active:
//
//   - "" (empty)   — always active
//   - "git-repo"   — active only when the working directory is inside a git
//     repository (checked via .git walk-up from cwd)
type Pattern struct {
	Type  string `json:"type"`  // "prefix" | "exact" | "regex"
	Scope string `json:"scope"` // "" | "git-repo"
	Value string `json:"value"`
}

type Allowlist struct {
	Patterns []Pattern `json:"patterns"`
	compiled []*regexp.Regexp // pre-compiled regex patterns; index matches Patterns
}

// compile pre-compiles all regex patterns in the allowlist and caches them.
// Invalid patterns are stored as nil and silently skipped during Match.
func (al *Allowlist) compile() {
	al.compiled = make([]*regexp.Regexp, len(al.Patterns))
	for i, p := range al.Patterns {
		if p.Type == "regex" {
			al.compiled[i], _ = regexp.Compile(p.Value)
		}
	}
}

// Load reads a JSON allowlist file and degrades to an empty allowlist on
// missing or corrupt input.
func Load(path string) (*Allowlist, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return &Allowlist{}, nil
	}

	var al Allowlist
	if err := json.Unmarshal(data, &al); err != nil {
		return &Allowlist{}, nil
	}

	al.compile()
	return &al, nil
}

// LoadOrSeed loads the allowlist at path, seeding it from the embedded
// defaults if the file does not yet exist. When path is empty the embedded
// defaults are decoded in memory with no file I/O.
func LoadOrSeed(path string) (*Allowlist, error) {
	if path == "" {
		var al Allowlist
		if err := json.Unmarshal(defaultAllowlistJSON, &al); err != nil {
			return &Allowlist{}, nil
		}
		al.compile()
		return &al, nil
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if mkErr := os.MkdirAll(filepath.Dir(path), 0o755); mkErr == nil {
			_ = os.WriteFile(path, defaultAllowlistJSON, 0o644)
		}
	}

	return Load(path)
}

// Match reports whether cmd is permitted by the allowlist given the working
// directory cwd. Patterns with scope "git-repo" only match when cwd (or one
// of its parents) contains a .git entry.
func (al *Allowlist) Match(cmd, cwd string) bool {
	if al.compiled == nil {
		al.compile()
	}
	for i, pattern := range al.Patterns {
		if pattern.Scope == "git-repo" && !IsInsideGitRepo(cwd) {
			continue
		}
		switch pattern.Type {
		case "prefix":
			if strings.HasPrefix(cmd, pattern.Value) {
				return true
			}
		case "exact":
			if cmd == pattern.Value {
				return true
			}
		case "regex":
			if i < len(al.compiled) && al.compiled[i] != nil && al.compiled[i].MatchString(cmd) {
				return true
			}
		}
	}

	return false
}

// IsInsideGitRepo reports whether dir (or any of its parents) contains a .git
// entry. It recognises both standard repos (.git directory) and worktrees
// (.git file).
func IsInsideGitRepo(dir string) bool {
	if dir == "" {
		return false
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}
