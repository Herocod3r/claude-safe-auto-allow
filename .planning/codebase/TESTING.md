# Testing Patterns

**Analysis Date:** 2026-03-15

## Test Framework

**Runner:**
- Go standard library `testing` package (no external test frameworks like testify)
- Config: None - uses Go's built-in test runner

**Assertion Library:**
- No external assertion library (testify/assert not used)
- Manual assertions using `if/else` and `t.Errorf()`, `t.Fatal()`, `t.Fatalf()`

**Run Commands:**
```bash
go test ./...              # Run all tests
go test -v ./...           # Run with verbose output
go test -cover ./...       # Run with coverage
go test -run TestName ./...  # Run specific test
```

## Test File Organization

**Location:**
- Co-located with source: Test files in same directory as implementation
- Pattern: `<name>.go` implementation, `<name>_test.go` for tests

**Naming:**
- Test function format: `Test<FunctionName>` for exported functions
- Test variants: `Test<FunctionName><Scenario>` for multiple scenarios
- Examples:
  - `TestEvaluateNonBash()` - tests non-bash tool handling
  - `TestMatchPrefix()` - tests prefix pattern matching
  - `TestCheckDestructiveRm()` - tests rm pattern detection

**Structure:**
```
internal/guard/
├── guard.go
└── guard_test.go

internal/allowlist/
├── allowlist.go
└── allowlist_test.go

internal/patterns/
├── patterns.go
├── patterns_test.go
└── rm.go

internal/version/
├── version.go
└── version_test.go

internal/learn/
├── learn.go
└── learn_test.go
```

## Test Structure

**Suite Organization:**
```go
func TestEvaluateSafeCommands(t *testing.T) {
	safe := []string{
		"ls -la", "cat README.md", "echo hello",
	}

	for _, cmd := range safe {
		result := Evaluate("Bash", cmd, "")
		if result.Decision != "approve" {
			t.Errorf("Evaluate(Bash, %q) = %q, want approve", cmd, result.Decision)
		}
	}
}
```

**Patterns:**
- Arrange-Act-Assert: Setup input data, call function, check result
- Table-driven tests for multiple scenarios in one test
- No test fixtures or setup/teardown in init; use individual setup per test

## Mocking

**Framework:** No external mocking (no gomock, mockgen)

**Patterns:**
```go
// Temporary files for testing file I/O
func TestEvaluateAllowlist(t *testing.T) {
	dir := t.TempDir()  // Create temporary directory
	path := filepath.Join(dir, "allowlist.json")
	if err := os.WriteFile(path, []byte(`{"patterns":[...]}`), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	result := Evaluate("Bash", "docker build -t app .", path)
	if result.Decision != "approve" {
		t.Errorf("allowlisted command should approve, got %q", result.Decision)
	}
}
```

**Helper Function Pattern:**
```go
func writePending(t *testing.T, dir, sessionID, cmd string, dangerous bool, age time.Duration) string {
	t.Helper()  // Mark as helper to exclude from test output line numbers

	path := filepath.Join(dir, "claude-safety-pending-"+sessionID+".json")
	data := PendingCommand{
		Command:   cmd,
		Timestamp: time.Now().Add(-age).UnixMilli(),
		Dangerous: dangerous,
	}
	body, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	return path
}
```

**What to Mock:**
- File I/O: Use `t.TempDir()` for isolated test files
- Time: Use `time.Now()` with `Add()` to simulate different timestamps
- No mocking of other packages (minimal dependencies)

**What NOT to Mock:**
- Core business logic functions (test them directly)
- Pattern matching (test actual regex behavior)
- Data structures (test real behavior)

## Fixtures and Factories

**Test Data:**
- Inline in test functions as string slices:
```go
dangerous := []string{
	"rm -rf /", "rm -rf ~", "rm -rf $HOME",
	"rm -rf /*", "rm -r -f /", "rm --recursive --force /",
}
for _, cmd := range dangerous {
	if result := CheckDestructiveRm(cmd); result == "" {
		t.Errorf("CheckDestructiveRm(%q) should be dangerous", cmd)
	}
}
```

- Helper functions for complex setup (e.g., `writePending()` in learn_test.go)

**Location:**
- Inline in test files, no separate fixtures directory
- Test-scoped temporary files via `t.TempDir()`

## Coverage

**Requirements:** No enforced coverage minimum (no codecov config found)

**View Coverage:**
```bash
go test -cover ./...          # Show coverage percentage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out  # Open HTML coverage report
```

## Test Types

**Unit Tests:**
- Scope: Individual functions (Evaluate, Match, Load, IsDangerous)
- Approach: Test with various inputs, check return values and side effects
- Examples: `TestMatchPrefix()`, `TestEvaluateSafeCommands()`, `TestCheckDestructiveRm()`

**Integration Tests:**
- Scope: Cross-package flows (guard + allowlist, main + guard)
- Approach: File I/O integration (reading allowlist.json, writing pending files)
- Examples: `TestEvaluateAllowlist()`, `TestPendingFileDeletedAfterRead()`

**E2E Tests:**
- Not present - no shell integration tests found
- Separate shell script in `tests/test-install-e2e.sh` for script-level testing

## Common Patterns

**Table-Driven Tests:**
```go
func TestSplitSubCommands(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"ls", 1},
		{"ls && echo done", 2},
		{"ls; echo; pwd", 3},
		{"ls | grep foo", 2},
		{"ls || echo fail && pwd", 3},
	}

	for _, tt := range tests {
		got := SplitSubCommands(tt.input)
		if len(got) != tt.want {
			t.Errorf("SplitSubCommands(%q) = %d parts, want %d", tt.input, len(got), tt.want)
		}
	}
}
```

**Error Testing:**
```go
func TestLoadCorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "allowlist.json")
	if err := os.WriteFile(path, []byte(`not json {{{`), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	al, err := Load(path)
	if err != nil {  // Load() returns nil error on corrupt data (graceful degradation)
		t.Fatalf("Load should not error on corrupt file: %v", err)
	}
	if al.Match("anything") {
		t.Fatal("corrupt allowlist should not match")
	}
}
```

**Async/Time-Based Testing:**
```go
func TestEvaluateExpiredPending(t *testing.T) {
	dir := t.TempDir()
	writePending(t, dir, "expired", "docker run ubuntu", false, 2*time.Minute)

	msg := Evaluate("Bash", "expired", dir, "/tmp/allowlist.json")
	if msg != "" {
		t.Errorf("expired pending should return empty, got %q", msg)
	}
}
```

**File Cleanup Verification:**
```go
func TestPendingFileDeletedAfterRead(t *testing.T) {
	dir := t.TempDir()
	path := writePending(t, dir, "cleanup", "docker run ubuntu", false, 5*time.Second)

	Evaluate("Bash", "cleanup", dir, "/tmp/allowlist.json")
	if _, err := os.Stat(path); err == nil {
		t.Fatal("pending file should be deleted after read")
	}
}
```

## Coverage Summary

**Test Files and Line Coverage:**
- `internal/guard/guard_test.go` - 15 tests covering guard evaluation logic
- `internal/allowlist/allowlist_test.go` - 8 tests covering pattern matching
- `internal/patterns/patterns_test.go` - 6 tests covering pattern detection
- `internal/learn/learn_test.go` - 9 tests covering learning logic
- `internal/version/version_test.go` - 1 test covering version comparison

**Untested Areas:**
- `cmd/main.go` - Not directly tested (e2e via shell script)
- Error recovery paths (some silent failures)
- Concurrent execution scenarios

---

*Testing analysis: 2026-03-15*
