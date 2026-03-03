// Package safety provides a 4-level safety classification system for tool operations.
// It classifies operations as Green (safe), Yellow (caution), Red (dangerous), or Black (forbidden).
package safety

import (
	"fmt"
	"strings"

	"github.com/awch-D/ForgeX/forgex-core/logger"
)

// Level represents the safety classification of an operation.
type Level int

const (
	Green  Level = 1 // Read-only operations — auto-approve
	Yellow Level = 2 // Write operations — approve by default, configurable
	Red    Level = 3 // System commands — require confirmation
	Black  Level = 4 // Dangerous operations — always blocked
)

func (l Level) String() string {
	switch l {
	case Green:
		return "🟢 Green"
	case Yellow:
		return "🟡 Yellow"
	case Red:
		return "🔴 Red"
	case Black:
		return "⚫ Black"
	default:
		return "Unknown"
	}
}

func (l Level) Emoji() string {
	switch l {
	case Green:
		return "🟢"
	case Yellow:
		return "🟡"
	case Red:
		return "🔴"
	case Black:
		return "⚫"
	default:
		return "❓"
	}
}

// NeedsApproval returns true if this level requires user confirmation
// given the configured auto-approve threshold.
func (l Level) NeedsApproval(autoApproveLevel Level) bool {
	return l > autoApproveLevel
}

// IsBlocked returns true if this operation should be unconditionally blocked.
func (l Level) IsBlocked() bool {
	return l >= Black
}

// blacklistPatterns are shell commands/patterns that are always blocked.
var blacklistPatterns = []string{
	"rm -rf /",
	"rm -rf ~",
	"rm -rf .",
	"mkfs",
	"dd if=",
	":(){:|:&};:", // Fork bomb
	"chmod -R 777 /",
	"> /dev/sda",
	"shutdown",
	"reboot",
	"init 0",
	"init 6",
}

// dangerousPatterns are commands that require confirmation but aren't blocked.
var dangerousPatterns = []string{
	"rm -rf",
	"rm -r",
	"sudo",
	"chmod",
	"chown",
	"kill",
	"pkill",
	"curl",
	"wget",
	"pip install",
	"npm install -g",
	"brew install",
	"apt install",
	"apt-get install",
}

// Classify determines the safety level of a tool invocation.
func Classify(toolName string, args map[string]string) Level {
	switch toolName {
	case "read_file", "list_dir":
		return Green

	case "write_file":
		path := args["path"]
		// Block writes to sensitive paths
		if isSensitivePath(path) {
			return Red
		}
		return Yellow

	case "run_command":
		cmd := strings.ToLower(args["command"])
		// Check blacklist first
		for _, pattern := range blacklistPatterns {
			if strings.Contains(cmd, pattern) {
				logger.L().Warnw("⚫ BLOCKED: dangerous command detected",
					"command", args["command"], "pattern", pattern)
				return Black
			}
		}
		// Check piped execution patterns (curl/wget piped to shell)
		if (strings.Contains(cmd, "curl") || strings.Contains(cmd, "wget")) &&
			(strings.Contains(cmd, "| sh") || strings.Contains(cmd, "| bash") ||
				strings.Contains(cmd, "|sh") || strings.Contains(cmd, "|bash")) {
			logger.L().Warnw("⚫ BLOCKED: piped remote execution detected",
				"command", args["command"])
			return Black
		}
		// Check dangerous patterns
		for _, pattern := range dangerousPatterns {
			if strings.Contains(cmd, pattern) {
				return Red
			}
		}
		return Red // All commands default to Red

	default:
		return Yellow
	}
}

// isSensitivePath checks if a file path targets sensitive locations.
func isSensitivePath(path string) bool {
	sensitive := []string{
		"/etc/", "/usr/", "/bin/", "/sbin/",
		"~/.ssh", "~/.aws", "~/.config",
		".env", ".git/config", "id_rsa",
	}
	lower := strings.ToLower(path)
	for _, s := range sensitive {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}

// ParseLevel converts a config string to a safety level.
func ParseLevel(s string) Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "green":
		return Green
	case "yellow":
		return Yellow
	case "red":
		return Red
	case "black":
		return Black
	default:
		return Yellow // Default: auto-approve up to yellow
	}
}

// FormatDecision returns a human-readable string describing the safety decision.
func FormatDecision(toolName string, level Level, approved bool) string {
	action := "✅ APPROVED"
	if !approved {
		action = "❌ BLOCKED"
	}
	return fmt.Sprintf("%s %s [%s] tool=%s", level.Emoji(), action, level.String(), toolName)
}
