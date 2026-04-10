// Package agent provides detection, path resolution, and installation logic
// for deploying agent instructions to Claude Code, Gemini CLI, Antigravity, and Cursor.
package agent

import (
	"slices"
	"strings"
)

// Agent identifies a supported AI tool.
type Agent string

// Supported agent identifiers.
const (
	Claude      Agent = "claude"
	Gemini      Agent = "gemini"
	Antigravity Agent = "antigravity"
	Cursor      Agent = "cursor"
)

// All returns all supported agents.
func All() []Agent {
	return []Agent{Claude, Gemini, Antigravity, Cursor}
}

// ParseTargets splits a comma-separated string into a list of agents.
// Returns nil if the input is empty (meaning "all detected").
func ParseTargets(s string) []Agent {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []Agent
	for part := range strings.SplitSeq(s, ",") {
		t := strings.TrimSpace(strings.ToLower(part))
		if t != "" {
			out = append(out, Agent(t))
		}
	}
	return out
}

// Contains checks if the given agent is in the list.
func Contains(agents []Agent, a Agent) bool {
	return slices.Contains(agents, a)
}

// TargetWanted returns true if the agent should be acted on.
// If targets is nil (no filter), all agents are wanted.
func TargetWanted(a Agent, targets []Agent) bool {
	if targets == nil {
		return true
	}
	return Contains(targets, a)
}
