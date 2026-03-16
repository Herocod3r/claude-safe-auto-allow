# Architecture

**Analysis Date:** 2026-03-15

## Pattern Overview

**Overall:** Multi-stage filtering pipeline with pluggable rule sets

**Key Characteristics:**
- Command evaluation flows through sequential decision gates (dangerous → allowlist → safe patterns → fallthrough)
- Pre-tool-use and post-tool-use hook stages enable learning feedback loop
- Pattern matching via compiled regular expressions for performance
- Graceful degradation (corrupt configs default to conservative fallthrough)

## Layers

**Entry Layer (CLI):**
- Purpose: CLI command routing and hook input/output marshaling
- Location: `cmd/claude-safety/main.go`
- Contains: Main entry point, hook communication protocol (JSON I/O)
- Depends on: guard, learn packages
- Used by: Claude Code via pre/post-tool-use hooks

**Guard Layer (Pre-Tool-Use Decision):**
- Purpose: Evaluate bash commands and render approve/deny/fallthrough decisions
- Location: `internal/guard/`
- Contains: Evaluation logic, safe pattern matching, dangerous pattern matching
- Depends on: patterns, allowlist packages
- Used by: main.go for PreToolUse hook

**Learn Layer (Post-Tool-Use Feedback):**
- Purpose: Capture user approval of previously-rejected commands and suggest allowlist entries
- Location: `internal/learn/`
- Contains: Pending command tracking, learning suggestion generation
- Depends on: None (communicates via temp file coordination)
- Used by: main.go for PostToolUse hook

**Pattern Matching Layer:**
- Purpose: Centralized dangerous and safe command pattern definitions
- Location: `internal/patterns/` and `internal/guard/patterns.go`
- Contains: Regex patterns for dangerous operations, safe commands, rm-specific rules
- Depends on: None
- Used by: guard package

**Data Persistence Layer:**
- Purpose: Load and match against user allowlist rules
- Location: `internal/allowlist/`
- Contains: JSON allowlist parsing, pattern matching logic (prefix/exact/regex)
- Depends on: None
- Used by: guard package

**Utility Packages:**
- Purpose: Version comparison for update detection
- Location: `internal/version/`
- Contains: Semantic version parsing and comparison
- Depends on: None
- Used by: main.go for version checks

## Data Flow

**PreToolUse (Guard) Flow:**

1. User enters bash command in Claude Code
2. Claude Code invokes `claude-safety guard` with JSON input (tool_name, command, session_id)
3. main.go reads stdin, unmarshals hookInput
4. binaryDir() locates safety-allowlist.json and version.txt in executable directory
5. Version check: if buildVersion > installed version, emit update notice and exit
6. Extract command from ToolInput
7. guard.Evaluate() executes decision pipeline:
   - Non-bash tools → immediate approve
   - Empty command → fallthrough (no decision)
   - Check all dangerous patterns (SQL, filesystem, system, etc.)
   - If dangerous → return with Dangerous flag set
   - Load and check allowlist (prefix/exact/regex matching)
   - If allowlisted → approve
   - Check for command substitution ($(...) or backticks) → fallthrough if present
   - Check if all subcommands (split on &&, ||, |, ;) match safe patterns
   - If all safe → approve with reason "Safe command pattern"
   - Otherwise → fallthrough
8. Return JSON with Decision (empty for fallthrough), Reason, or hook-specific output
9. If command fell through and not dangerous, write PendingCommand to temp file for learn stage

**PostToolUse (Learn) Flow:**

1. User approves a previously-rejected command via Claude's permission prompt
2. Claude Code invokes `claude-safety learn` with JSON input
3. main.go reads stdin, unmarshals hookInput
4. learn.Evaluate() executes:
   - Check if tool is Bash (ignore others)
   - Read PendingCommand from temp file (keyed by session_id)
   - Verify timestamp is within 60 seconds
   - Reject if command was marked dangerous (safety model: don't learn dangerous)
   - Suggest allowlist entry (prefix or exact match)
   - Return formatted message to stdout
5. Claude Code displays message to user with options to add to allowlist

**State Management:**

- **Allowlist:** Persistent JSON file at `~/.claude-safe-auto-allow/safety-allowlist.json`
- **Pending Commands:** Ephemeral temp files at `/tmp/claude-safety-pending-{sessionID}.json` (cleaned up after 60 seconds)
- **Version:** Text file at same directory as binary for update detection

## Key Abstractions

**Pattern:**
- Purpose: Represent a matching rule in allowlist
- Examples: `internal/allowlist/allowlist.go` Pattern struct
- Pattern: Type (prefix|exact|regex) with Value field; compiled regex cached during match

**Result:**
- Purpose: Decision result from guard evaluation
- Examples: `internal/guard/guard.go` Result struct
- Pattern: Contains Decision (string), Reason (string), Dangerous (bool) flag

**Allowlist:**
- Purpose: Collection of rules to match commands against
- Examples: `internal/allowlist/allowlist.go` Allowlist struct
- Pattern: Loads from JSON gracefully (empty allowlist on error), implements Match() method supporting three pattern types

**PendingCommand:**
- Purpose: Capture state of command awaiting user approval
- Examples: `internal/learn/learn.go` PendingCommand struct
- Pattern: JSON serialized to temp file with command, timestamp, and dangerous flag

## Entry Points

**PreToolUse Hook:**
- Location: `cmd/claude-safety/main.go::runGuard()`
- Triggers: When user types bash command in Claude Code
- Responsibilities: Read hook input, evaluate command safety, write JSON decision to stdout, optionally write pending command to temp file

**PostToolUse Hook:**
- Location: `cmd/claude-safety/main.go::runLearn()`
- Triggers: After user approves a command via Claude's permission prompt
- Responsibilities: Read hook input, check for pending command, generate learning suggestion, write JSON message to stdout

**CLI Command Dispatch:**
- Location: `cmd/claude-safety/main.go::main()`
- Triggers: Binary invocation with subcommand argument (guard|learn|version)
- Responsibilities: Route to appropriate handler, print version on request

## Error Handling

**Strategy:** Fail-safe (errors result in fallthrough, not approval)

**Patterns:**

- **Stdin timeout:** 3-second timeout on reading stdin; if exceeded, exit 0 (no decision)
- **JSON unmarshal error:** Invalid JSON exits 0 (no decision)
- **File I/O errors:** ReadFile on allowlist returns empty allowlist (conservative fallthrough)
- **Corrupt JSON in files:** json.Unmarshal failure returns empty struct gracefully
- **Invalid regex in allowlist:** Regex compilation error skipped in pattern matching loop
- **Missing temp file:** Pending command not found returns empty string (no suggestion)
- **Expired pending command:** Timestamp > 60 seconds returns empty string (no suggestion)

All errors are silent (no stderr output) to avoid disrupting Claude Code workflow.

## Cross-Cutting Concerns

**Logging:**
- None in production code (silent failure model)
- Test-driven verification via guard_test.go examples

**Validation:**
- Command parsing: strings.TrimSpace, regex-based parsing
- Dangerous pattern matching: Regex compiled at init, cached in var
- Allowlist loading: JSON schema validated via Go struct unmarshaling

**Authentication:**
- None (hooks run in user's process context with access to their allowlist)
- Security model: Dangerous patterns bypass allowlist entirely
- Version-based freshness check at guard time

---

*Architecture analysis: 2026-03-15*
