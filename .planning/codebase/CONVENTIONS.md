# Coding Conventions

**Analysis Date:** 2026-03-15

## Naming Patterns

**Files:**
- Lowercase with underscores for compound names: `allowlist.go`, `patterns.go`
- Test files suffix with `_test.go`: `allowlist_test.go`, `guard_test.go`
- Main entry: `main.go` in `cmd/` directory subdirectories

**Functions:**
- PascalCase for exported functions (public API): `Evaluate()`, `Match()`, `Load()`, `IsDangerous()`
- camelCase for unexported functions (package-private): `isSafe()`, `binaryDir()`, `readStdin()`, `parse()`
- Function names are descriptive verbs: `CheckDestructiveRm()`, `SplitSubCommands()`, `HasCommandSubstitution()`

**Variables:**
- camelCase for local and package variables: `dataCh`, `errCh`, `subs`, `sessionID`
- Descriptive names: `command`, `allowlistPath`, `dangerous`, `result`
- Interface{} maps use descriptive keys: `ToolName`, `ToolInput`, `SessionID` (PascalCase for JSON/struct fields)

**Types:**
- PascalCase for struct names: `Result`, `Pattern`, `Allowlist`, `PendingCommand`, `hookInput`, `hookOutput`
- PascalCase for struct fields exported in JSON: `Type`, `Value`, `Decision`, `Reason`, `Command`, `Timestamp`, `Dangerous`
- Constants in camelCase: `buildVersion`

**Constants:**
- Compiled regex patterns stored in package-level `var` blocks: `dangerousPatterns`, `rmCommandRe`, `hasRecursiveRe`
- Magic values inline with comments or in constants: `3 * time.Second`, `60_000` (milliseconds)

## Code Style

**Formatting:**
- Standard Go fmt tool (implicit - no .editorconfig or prettier config found)
- Line length: No strict limit observed, pragmatic wrapping around 80-100 characters
- Indentation: Tabs (Go standard)

**Linting:**
- No explicit linting configuration found (no .eslintrc, golangci.yml)
- Code follows Go idioms: minimal error handling for non-critical operations (silent failures where appropriate)

## Import Organization

**Order:**
1. Standard library imports (e.g., `"fmt"`, `"os"`, `"encoding/json"`, `"regexp"`)
2. Blank line
3. Internal package imports (e.g., `"github.com/Herocod3r/claude-safe-auto-allow/internal/guard"`)

**Example from `cmd/claude-safety/main.go`:**
```go
import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Herocod3r/claude-safe-auto-allow/internal/guard"
	"github.com/Herocod3r/claude-safe-auto-allow/internal/learn"
	versionpkg "github.com/Herocod3r/claude-safe-auto-allow/internal/version"
)
```

**Path Aliases:**
- Aliased imports to avoid conflicts: `versionpkg "github.com/Herocod3r/claude-safe-auto-allow/internal/version"` (to avoid conflict with `version.go`)

## Error Handling

**Patterns:**
- Silent failures for non-critical operations: `_ = json.NewEncoder(os.Stdout).Encode(...)` (discarding error intentionally)
- Graceful degradation on missing/corrupt data: `allowlist.Load()` returns empty allowlist on file not found or JSON parse error
- Early return on error without logging: Common for guard/learn pattern where failed validations don't error-out
- Named return values not used; explicit returns preferred: `return Result{Decision: "approve", Reason: "..."}`

**Error Handling Strategy:**
```go
// Silent failure when file missing or corrupt
data, err := os.ReadFile(path)
if err != nil {
	return &Allowlist{}, nil  // Degrade gracefully
}

// Blunt check: if anything fails, return empty/default
var al Allowlist
if err := json.Unmarshal(data, &al); err != nil {
	return &Allowlist{}, nil
}
```

## Logging

**Framework:** None (no logger dependency found)

**Patterns:**
- No application logging
- Uses JSON encoding to stdout for structured output: `json.NewEncoder(os.Stdout).Encode()`
- Only for critical flow: decision output in `main.go`
- Silent failures everywhere else (no stderr, no logging)

## Comments

**When to Comment:**
- Function-level doc comments for exported functions: `// Load reads a JSON allowlist file and degrades to an empty allowlist on missing or corrupt input.`
- Inline comments for non-obvious logic: regex pattern explanations, dangerous command checks
- No comments for straightforward code

**Doc Comments:**
- Go doc comment style: Function name as first word for exported functions
- Example from `internal/allowlist/allowlist.go`:
  ```go
  // Load reads a JSON allowlist file and degrades to an empty allowlist on
  // missing or corrupt input.
  func Load(path string) (*Allowlist, error) { ... }
  ```

## Function Design

**Size:**
- Small focused functions: 10-50 lines typical (e.g., `Evaluate()` in guard is 40 lines, `Load()` is 13 lines)
- Functions with clear single responsibility

**Parameters:**
- Positional parameters, no options objects
- Mix of required and optional (used in guard: required toolName/command, optional allowlistPath)
- Return `Result` struct instead of multiple return values for complex results

**Return Values:**
- Single return value preferred: `Match(cmd string) bool`
- Multiple values for error: `Load(path string) (*Allowlist, error)`
- Result structs for complex returns: `type Result struct { Decision, Reason string; Dangerous bool }`

## Module Design

**Exports:**
- Clear public API per package via PascalCase exports
- `internal/` directory enforces Go's internal package convention
- Main entry point: `cmd/claude-safety/main.go`

**Barrel Files:**
- Not used - each package is self-contained
- `cmd/main.go` imports directly from internal packages

**Package Organization:**
- `internal/guard/` - Command evaluation
- `internal/allowlist/` - Pattern matching and loading
- `internal/patterns/` - Dangerous pattern detection, rm checks
- `internal/learn/` - Learning from denied commands
- `internal/version/` - Version comparison
- `cmd/claude-safety/` - CLI entry point and I/O handling

---

*Convention analysis: 2026-03-15*
