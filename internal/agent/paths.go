package agent

import (
	"os"
	"path/filepath"
)

// GeminiConfigDir returns the Gemini CLI config directory,
// respecting the GEMINI_CONFIG_DIR env var.
func GeminiConfigDir() string {
	if env := os.Getenv("GEMINI_CONFIG_DIR"); env != "" {
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
	case Gemini, Antigravity:
		return filepath.Join(GeminiConfigDir(), "GEMINI.md")
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
	case Gemini, Antigravity:
		return filepath.Join(GeminiConfigDir(), "settings.json")
	case Cursor:
		return filepath.Join(homeDir(), ".cursor", "hooks.json")
	default:
		return ""
	}
}

// SkillsDest returns the directory where skills are copied for the agent.
// Gemini CLI resolves universal skills from the repo's skills/ directory (typically ~/.agents/skills);
// install does not mirror into GeminiConfigDir()/skills (see installSkills). Antigravity uses
// a separate directory under GeminiConfigDir().
// Returns empty string for unknown agents.
func SkillsDest(a Agent) string {
	switch a {
	case Claude:
		return filepath.Join(homeDir(), ".claude", "skills")
	case Cursor:
		return filepath.Join(homeDir(), ".cursor", "skills")
	case Gemini:
		return filepath.Join(GeminiConfigDir(), "skills")
	case Antigravity:
		return filepath.Join(GeminiConfigDir(), "antigravity", "skills")
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
	case Gemini, Antigravity:
		return "gemini-settings-snippet.json"
	case Cursor:
		return "cursor-hooks-snippet.json"
	default:
		return ""
	}
}
