package main

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

// buildVersion is set via ldflags at build time.
var buildVersion = "dev"

type hookInput struct {
	ToolName  string                 `json:"tool_name"`
	ToolInput map[string]interface{} `json:"tool_input"`
	SessionID string                 `json:"session_id"`
}

type hookOutput struct {
	Decision           string        `json:"decision,omitempty"`
	Reason             string        `json:"reason,omitempty"`
	HookSpecificOutput *hookSpecific `json:"hookSpecificOutput,omitempty"`
}

type hookSpecific struct {
	HookEventName     string `json:"hookEventName"`
	AdditionalContext string `json:"additionalContext"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: claude-safety <guard|learn|version>\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "guard":
		runGuard()
	case "learn":
		runLearn()
	case "version":
		fmt.Println(buildVersion)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nUsage: claude-safety <guard|learn|version>\n", os.Args[1])
		os.Exit(1)
	}
}

func readStdin() ([]byte, error) {
	dataCh := make(chan []byte, 1)
	errCh := make(chan error, 1)

	go func() {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			errCh <- err
			return
		}
		dataCh <- data
	}()

	select {
	case data := <-dataCh:
		return data, nil
	case err := <-errCh:
		return nil, err
	case <-time.After(3 * time.Second):
		return nil, fmt.Errorf("stdin timeout")
	}
}

func binaryDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}

	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return filepath.Dir(exe)
	}

	return filepath.Dir(resolved)
}

func runGuard() {
	data, err := readStdin()
	if err != nil {
		os.Exit(0)
	}

	var input hookInput
	if err := json.Unmarshal(data, &input); err != nil {
		os.Exit(0)
	}

	dir := binaryDir()
	if installedBytes, err := os.ReadFile(filepath.Join(dir, "version.txt")); err == nil {
		installed := strings.TrimSpace(string(installedBytes))
		if versionpkg.Gt(buildVersion, installed) {
			_ = json.NewEncoder(os.Stdout).Encode(hookOutput{
				HookSpecificOutput: &hookSpecific{
					HookEventName:     "PreToolUse",
					AdditionalContext: fmt.Sprintf("Your claude-safe-auto-allow hooks are outdated (installed: %s, latest: %s). Please run the install script again to update.", installed, buildVersion),
				},
			})
			os.Exit(0)
		}
	}

	command := ""
	if raw, ok := input.ToolInput["command"]; ok {
		if cmd, ok := raw.(string); ok {
			command = strings.TrimSpace(cmd)
		}
	}

	cwd, _ := os.Getwd()
	result := guard.Evaluate(input.ToolName, command, filepath.Join(dir, "safety-allowlist.json"), cwd)
	if result.Decision == "approve" {
		_ = json.NewEncoder(os.Stdout).Encode(hookOutput{
			Decision: result.Decision,
			Reason:   result.Reason,
		})
		return
	}

	if command != "" {
		sessionID := input.SessionID
		if sessionID == "" {
			sessionID = "unknown"
		}
		pending := learn.PendingCommand{
			Command:   command,
			Timestamp: time.Now().UnixMilli(),
			Dangerous: result.Dangerous,
		}
		if body, err := json.Marshal(pending); err == nil {
			_ = os.WriteFile(filepath.Join(os.TempDir(), "claude-safety-pending-"+sessionID+".json"), body, 0o644)
		}
	}

	os.Exit(0)
}

func runLearn() {
	data, err := readStdin()
	if err != nil {
		os.Exit(0)
	}

	var input hookInput
	if err := json.Unmarshal(data, &input); err != nil {
		os.Exit(0)
	}

	sessionID := input.SessionID
	if sessionID == "" {
		sessionID = "unknown"
	}

	dir := binaryDir()
	msg := learn.Evaluate(input.ToolName, sessionID, os.TempDir(), filepath.Join(dir, "safety-allowlist.json"))
	if msg == "" {
		os.Exit(0)
	}

	_ = json.NewEncoder(os.Stdout).Encode(hookOutput{
		HookSpecificOutput: &hookSpecific{
			HookEventName:     "PostToolUse",
			AdditionalContext: msg,
		},
	})
}
