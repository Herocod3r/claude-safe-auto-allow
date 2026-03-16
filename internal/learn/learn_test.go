package learn

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writePending(t *testing.T, dir, sessionID, cmd string, dangerous bool, age time.Duration) string {
	t.Helper()

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

func TestEvaluateNonBash(t *testing.T) {
	msg := Evaluate("Read", "test-session", os.TempDir(), "/tmp/allowlist.json")
	if msg != "" {
		t.Errorf("non-Bash should return empty, got %q", msg)
	}
}

func TestEvaluateNoPendingFile(t *testing.T) {
	msg := Evaluate("Bash", "nonexistent-session", os.TempDir(), "/tmp/allowlist.json")
	if msg != "" {
		t.Errorf("missing pending file should return empty, got %q", msg)
	}
}

func TestEvaluateExpiredPending(t *testing.T) {
	dir := t.TempDir()
	writePending(t, dir, "expired", "docker run ubuntu", false, 2*time.Minute)

	msg := Evaluate("Bash", "expired", dir, "/tmp/allowlist.json")
	if msg != "" {
		t.Errorf("expired pending should return empty, got %q", msg)
	}
}

func TestEvaluateDangerousPending(t *testing.T) {
	dir := t.TempDir()
	writePending(t, dir, "danger", "rm -rf /", true, 5*time.Second)

	msg := Evaluate("Bash", "danger", dir, "/tmp/allowlist.json")
	if msg != "" {
		t.Errorf("dangerous pending should return empty, got %q", msg)
	}
}

func TestEvaluateValidPending(t *testing.T) {
	dir := t.TempDir()
	writePending(t, dir, "valid", "docker build -t app .", false, 5*time.Second)

	msg := Evaluate("Bash", "valid", dir, "/tmp/allowlist.json")
	if msg == "" {
		t.Fatal("valid pending should return a message")
	}
	if !strings.Contains(msg, "docker build") {
		t.Errorf("message should contain command, got %q", msg)
	}
}

func TestEvaluateCorruptPending(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "claude-safety-pending-corrupt.json")
	if err := os.WriteFile(path, []byte("not json {{{"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	msg := Evaluate("Bash", "corrupt", dir, "/tmp/allowlist.json")
	if msg != "" {
		t.Errorf("corrupt pending should return empty, got %q", msg)
	}
}

func TestPendingFileDeletedAfterRead(t *testing.T) {
	dir := t.TempDir()
	path := writePending(t, dir, "cleanup", "docker run ubuntu", false, 5*time.Second)

	Evaluate("Bash", "cleanup", dir, "/tmp/allowlist.json")
	if _, err := os.Stat(path); err == nil {
		t.Fatal("pending file should be deleted after read")
	}
}
