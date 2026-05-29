package agent

import (
	"os"
	"path/filepath"
)

// AntigravityConfigDir returns the Antigravity config directory,
// respecting the ANTIGRAVITY_CONFIG_DIR env var.
func AntigravityConfigDir() string {
	if env := os.Getenv("ANTIGRAVITY_CONFIG_DIR"); env != "" {
		return env
	}
	return filepath.Join(homeDir(), ".gemini")
}

// CodexConfigDir returns the Codex config directory,
// respecting the CODEX_HOME env var.
func CodexConfigDir() string {
	if env := os.Getenv("CODEX_HOME"); env != "" {
		return env
	}
	return filepath.Join(homeDir(), ".codex")
}

// HooksDeployDir is where hook scripts are copied at install time.
func HooksDeployDir() string {
	return filepath.Join(homeDir(), ".agents-hooks")
}

// MarkdownDest returns the destination path for the global agents markdown file.
func MarkdownDest(a Agent) string {
	switch a {
	case Claude:
		return filepath.Join(homeDir(), ".claude", "CLAUDE.md")
	case Antigravity:
		return filepath.Join(AntigravityConfigDir(), "GEMINI.md")
	case Cursor:
		return filepath.Join(homeDir(), ".cursor", "rules", "global-agents.mdc")
	case Codex:
		return filepath.Join(CodexConfigDir(), "AGENTS.md")
	default:
		return ""
	}
}

// HookSettingsPath returns the settings file that hooks are merged into.
func HookSettingsPath(a Agent) string {
	switch a {
	case Claude:
		return filepath.Join(homeDir(), ".claude", "settings.json")
	case Cursor:
		return filepath.Join(homeDir(), ".cursor", "hooks.json")
	default:
		return ""
	}
}

// SkillsDest returns the directory where skills are copied for the agent.
// Returns empty string for unknown agents.
func SkillsDest(a Agent) string {
	switch a {
	case Claude:
		return filepath.Join(homeDir(), ".claude", "skills")
	case Cursor:
		return filepath.Join(homeDir(), ".cursor", "skills")
	case Antigravity:
		return filepath.Join(AntigravityConfigDir(), "skills")
	case Codex:
		return filepath.Join(CodexConfigDir(), "skills")
	default:
		return ""
	}
}

// HookSnippetFile returns the JSON snippet filename in the hooks/ directory for the agent.
func HookSnippetFile(a Agent) string {
	switch a {
	case Claude:
		return "claude-settings-snippet.json"
	case Cursor:
		return "cursor-hooks-snippet.json"
	default:
		return ""
	}
}
