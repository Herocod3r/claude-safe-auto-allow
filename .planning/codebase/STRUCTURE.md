# Codebase Structure

**Analysis Date:** 2026-03-15

## Directory Layout

```
claude-safe-auto-allow/
├── cmd/
│   └── claude-safety/         # CLI entry point and hook handlers
│       └── main.go            # Command routing, hook I/O, version check
├── internal/
│   ├── guard/                 # PreToolUse hook: decision logic
│   │   ├── guard.go           # Evaluate() function, Result type
│   │   ├── patterns.go        # Safe command patterns (30+ regexes)
│   │   └── guard_test.go      # 14 test cases covering decision paths
│   ├── learn/                 # PostToolUse hook: learning suggestion
│   │   ├── learn.go           # Evaluate() for pending commands
│   │   └── learn_test.go      # Pending command suggestion tests
│   ├── patterns/              # Dangerous pattern definitions
│   │   ├── patterns.go        # Dangerous patterns (15+ regexes), SplitSubCommands()
│   │   ├── rm.go              # Destructive rm detection logic
│   │   └── patterns_test.go   # Pattern matching tests
│   ├── allowlist/             # Allowlist file parsing and matching
│   │   ├── allowlist.go       # Pattern type, Allowlist struct, Load(), Match()
│   │   └── allowlist_test.go  # Prefix/exact/regex matching tests
│   └── version/               # Version comparison utilities
│       ├── version.go         # Semantic version parsing and Gt() comparison
│       └── version_test.go    # Version comparison tests
├── tests/
│   └── test-install-e2e.sh    # End-to-end installation verification script
├── go.mod                      # Module declaration (github.com/Herocod3r/claude-safe-auto-allow)
├── go.sum                      # Dependency checksums (none - no external deps)
├── README.md                   # User documentation
├── install.sh                  # Installation script for end users
└── uninstall.sh                # Uninstallation script
```

## Directory Purposes

**cmd/claude-safety/:**
- Purpose: Binary entry point; handles CLI routing and hook protocol
- Contains: Single main.go file with command dispatch
- Key files: `main.go`

**internal/guard/:**
- Purpose: Decision engine for PreToolUse hook
- Contains: Command evaluation, safe pattern definitions, test cases
- Key files: `guard.go` (16 lines of logic), `patterns.go` (30+ safe patterns), `guard_test.go` (14 comprehensive tests)

**internal/learn/:**
- Purpose: Learning engine for PostToolUse hook
- Contains: Pending command tracking, allowlist suggestion generation
- Key files: `learn.go` (generates user-facing suggestion messages)

**internal/patterns/:**
- Purpose: Dangerous command pattern library
- Contains: 15+ dangerous regex patterns, rm-specific validation
- Key files: `patterns.go` (dangerous checks), `rm.go` (recursive+force+critical-dir logic)

**internal/allowlist/:**
- Purpose: Allowlist persistence and matching
- Contains: JSON unmarshaling, pattern matching (prefix/exact/regex)
- Key files: `allowlist.go` (Load, Match implementations)

**internal/version/:**
- Purpose: Version comparison for update detection
- Contains: Semantic version parsing (major.minor.patch)
- Key files: `version.go` (Gt comparison function)

**tests/:**
- Purpose: End-to-end testing
- Contains: Installation and hook integration verification
- Key files: `test-install-e2e.sh`

## Key File Locations

**Entry Points:**
- `cmd/claude-safety/main.go`: CLI entry point; routes subcommands to guard/learn/version handlers

**Configuration:**
- `~/.claude-safe-auto-allow/safety-allowlist.json`: User allowlist (loaded by guard at runtime)
- `~/.claude-safe-auto-allow/version.txt`: Installed version for update detection

**Core Logic:**
- `internal/guard/guard.go`: Evaluate() function implements decision pipeline
- `internal/guard/patterns.go`: 30+ safe command patterns
- `internal/patterns/patterns.go`: 15+ dangerous command patterns
- `internal/patterns/rm.go`: Destructive rm detection (recursive + force + critical dir)
- `internal/allowlist/allowlist.go`: Pattern matching against allowlist

**Testing:**
- `internal/guard/guard_test.go`: 14 test cases for decision paths
- `internal/patterns/patterns_test.go`: Dangerous pattern matching tests
- `internal/allowlist/allowlist_test.go`: Pattern matching tests
- `internal/version/version_test.go`: Version comparison tests
- `internal/learn/learn_test.go`: Pending command and suggestion tests
- `tests/test-install-e2e.sh`: Installation and hook integration E2E tests

## Naming Conventions

**Files:**
- Go files follow standard Go convention: lowercase, underscores for multi-word (e.g., `guard_test.go`)
- Package-scoped files: lowercase (e.g., `allowlist.go`, `patterns.go`)
- Main entry: `main.go` in cmd/ subdirectory

**Directories:**
- Internal packages: lowercase, single word or snake_case (e.g., `guard`, `allowlist`, `patterns`)
- Command packages: cmd/binary-name/ (e.g., `cmd/claude-safety/`)

**Functions:**
- Public API: PascalCase (e.g., Evaluate, Load, Match, Gt, IsDangerous)
- Private helpers: camelCase (e.g., isSafe, isSafeCurl, readStdin, binaryDir, runGuard)

**Types:**
- PascalCase (e.g., Result, Allowlist, Pattern, PendingCommand, hookInput, hookOutput)

**Variables:**
- camelCase for local/package vars (e.g., cmd, safePatterns, dangerousPatterns)
- UPPERCASE for constants (e.g., none currently, but pattern would apply)

## Where to Add New Code

**New Guard Pattern (Safe or Dangerous Command):**
- Safe patterns: Add to `safePatterns` slice in `internal/guard/patterns.go`
- Dangerous patterns: Add to `dangerousPatterns` slice in `internal/patterns/patterns.go`
- Tests: Add test case to `TestEvaluateSafeCommands` or `TestEvaluateDangerousCommandsFallthrough` in `internal/guard/guard_test.go`

**New Allowlist Feature:**
- Pattern type implementation: Add case to Pattern.Type switch in `internal/allowlist/allowlist.go::Match()`
- Tests: Add test to `internal/allowlist/allowlist_test.go`

**New Hook or CLI Command:**
- Handler function: Add runX() function in `cmd/claude-safety/main.go`
- Main dispatch: Add case to switch in main()

**New Utility Package:**
- Location: `internal/{package-name}/`
- Structure: package.go (implementation) + package_test.go (tests)

## Special Directories

**internal/:**
- Purpose: Go internal package boundary (enforced by Go compiler)
- Generated: No
- Committed: Yes
- Notes: All non-CLI code goes here; prevents external imports of internal packages

**.planning/codebase/:**
- Purpose: Codebase analysis documents (this directory)
- Generated: Yes (by GSD mapper)
- Committed: Yes
- Notes: Reference documents for code generation and architecture decisions

**cmd/:**
- Purpose: Compilable entry points
- Generated: No
- Committed: Yes
- Notes: One subdirectory per binary (here: claude-safety)

**tests/:**
- Purpose: E2E and integration tests separate from unit tests
- Generated: No
- Committed: Yes
- Notes: Shell scripts for testing installation and hook integration

---

*Structure analysis: 2026-03-15*
