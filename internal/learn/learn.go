package learn

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type PendingCommand struct {
	Command   string `json:"command"`
	Timestamp int64  `json:"timestamp"`
	Dangerous bool   `json:"dangerous"`
}

func Evaluate(toolName, sessionID, tmpDir, allowlistPath string) string {
	if toolName != "Bash" {
		return ""
	}

	path := filepath.Join(tmpDir, "claude-safety-pending-"+sessionID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	_ = os.Remove(path)

	var pending PendingCommand
	if err := json.Unmarshal(data, &pending); err != nil {
		return ""
	}
	if time.Now().UnixMilli()-pending.Timestamp > 60_000 {
		return ""
	}
	if pending.Dangerous {
		return ""
	}

	parts := strings.Fields(strings.TrimSpace(pending.Command))
	if len(parts) == 0 {
		return ""
	}

	suggestedPrefix := parts[0]
	if len(parts) > 1 {
		suggestedPrefix = parts[0] + " " + parts[1]
	}

	return fmt.Sprintf(
		"The command `%s` required manual approval because it wasn't in the safe list or your allowlist. "+
			"Ask the user: \"Would you like me to add this to your safety allowlist so it's auto-approved next time? "+
			"Options:\\n"+
			"1. **Prefix match** -- always allow commands starting with `%s`\\n"+
			"2. **Exact match** -- only allow this exact command: `%s`\\n"+
			"3. **No** -- don't add it\"\\n\\n"+
			"If they choose an option, update the allowlist at %s using the Write tool. "+
			"The format is: { \"patterns\": [{ \"type\": \"prefix\"|\"exact\"|\"regex\", \"value\": \"...\" }] }. "+
			"Read the file first if it exists, then append the new entry.",
		pending.Command, suggestedPrefix, pending.Command, allowlistPath,
	)
}
