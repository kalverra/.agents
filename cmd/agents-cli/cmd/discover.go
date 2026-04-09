package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	infoStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	boldStyle    = lipgloss.NewStyle().Bold(true)
)

func getGeminiConfigDir() string {
	if env := strings.TrimSpace(os.Getenv("GEMINI_CONFIG_DIR")); env != "" {
		if strings.HasPrefix(env, "~") {
			env = filepath.Join(getHomeDir(), env[1:])
		}
		return env
	}
	return filepath.Join(getHomeDir(), ".gemini")
}

func which(name string) string {
	p, err := exec.LookPath(name)
	if err == nil {
		return p
	}
	return ""
}

func detectClaude(verbose bool) bool {
	if p := which("claude"); p != "" {
		if verbose {
			fmt.Fprintf(os.Stderr, "[verbose] claude: found in PATH (%s)\n", p)
		}
		return true
	}
	d := filepath.Join(getHomeDir(), ".claude")
	if info, err := os.Stat(d); err == nil && info.IsDir() {
		if verbose {
			fmt.Fprintln(os.Stderr, "[verbose] claude: ~/.claude exists")
		}
		return true
	}
	return false
}

func detectGemini(verbose bool) bool {
	for _, name := range []string{"gemini", "gemini-cli"} {
		if p := which(name); p != "" {
			if verbose {
				fmt.Fprintf(os.Stderr, "[verbose] %s: found in PATH (%s)\n", name, p)
			}
			return true
		}
	}
	gd := getGeminiConfigDir()
	if info, err := os.Stat(gd); err == nil && info.IsDir() {
		if verbose {
			fmt.Fprintf(os.Stderr, "[verbose] gemini: config dir exists (%s)\n", gd)
		}
		return true
	}
	return false
}

func detectAntigravity(verbose bool) bool {
	if p := which("antigravity"); p != "" {
		if verbose {
			fmt.Fprintf(os.Stderr, "[verbose] antigravity: found in PATH (%s)\n", p)
		}
		return true
	}
	if info, err := os.Stat(filepath.Join(getHomeDir(), ".antigravity-server")); err == nil && info.IsDir() {
		if verbose {
			fmt.Fprintln(os.Stderr, "[verbose] antigravity: ~/.antigravity-server exists")
		}
		return true
	}
	sysname := runtime.GOOS
	if sysname == "darwin" {
		ag := filepath.Join(getHomeDir(), "Library/Application Support/Antigravity")
		if info, err := os.Stat(ag); err == nil && info.IsDir() {
			if verbose {
				fmt.Fprintf(os.Stderr, "[verbose] antigravity: %s exists\n", ag)
			}
			return true
		}
	}
	if sysname == "linux" {
		ag := filepath.Join(getHomeDir(), ".config/Antigravity")
		if info, err := os.Stat(ag); err == nil && info.IsDir() {
			if verbose {
				fmt.Fprintf(os.Stderr, "[verbose] antigravity: %s exists\n", ag)
			}
			return true
		}
	}
	return false
}

func detectCursor(verbose bool) bool {
	if info, err := os.Stat(filepath.Join(getHomeDir(), ".cursor")); err == nil && info.IsDir() {
		if verbose {
			fmt.Fprintln(os.Stderr, "[verbose] cursor: ~/.cursor exists")
		}
		return true
	}
	sysname := runtime.GOOS
	if sysname == "darwin" {
		p := filepath.Join(getHomeDir(), "Library/Application Support/Cursor")
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			if verbose {
				fmt.Fprintln(os.Stderr, "[verbose] cursor: ~/Library/Application Support/Cursor exists")
			}
			return true
		}
	}
	if sysname == "linux" {
		p := filepath.Join(getHomeDir(), ".config/Cursor")
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			if verbose {
				fmt.Fprintln(os.Stderr, "[verbose] cursor: ~/.config/Cursor exists")
			}
			return true
		}
	}
	return false
}

func repoSkillCount() int {
	root := filepath.Join(repoRoot(), "skills")
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			if _, err := os.Stat(filepath.Join(root, entry.Name(), "SKILL.md")); err == nil {
				count++
			}
		}
	}
	return count
}

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "List detected agents, install paths, and hook targets",
	Run: func(cmd *cobra.Command, args []string) {
		verbose, _ := cmd.Flags().GetBool("verbose")

		fmt.Println(headerStyle.Render("Discovery (signals only — tools need not be running):\n"))
		nSkills := repoSkillCount()
		var skillsRepo string
		if nSkills > 0 {
			skillsRepo = fmt.Sprintf("%d in repo skills/", nSkills)
		} else {
			skillsRepo = "none in repo/skills/"
		}
		geminiSkillsNote := fmt.Sprintf("%s; Gemini CLI discovers ~/.agents/skills (and ~/.gemini/skills); install does not copy there", skillsRepo)

		anyYes := false

		if detectClaude(verbose) {
			fmt.Printf("  %s   %s   context -> %s\n", boldStyle.Render("claude-code"), successStyle.Render("yes"), filepath.Join(getHomeDir(), ".claude/CLAUDE.md"))
			fmt.Printf("                      hooks   -> %s (PreToolUse)\n", filepath.Join(getHomeDir(), ".claude/settings.json"))
			fmt.Printf("                      skills  -> %s/ (%s; copy on install)\n", filepath.Join(getHomeDir(), ".claude/skills"), skillsRepo)
			anyYes = true
		} else {
			fmt.Printf("  %s   %s    (no claude in PATH, no ~/.claude)\n", boldStyle.Render("claude-code"), errorStyle.Render("no"))
		}

		geminiOk := detectGemini(verbose)
		antigravityOk := detectAntigravity(verbose)

		if geminiOk {
			fmt.Printf("  %s    %s   context -> %s\n", boldStyle.Render("gemini-cli"), successStyle.Render("yes"), filepath.Join(getGeminiConfigDir(), "GEMINI.md"))
			fmt.Printf("                      hooks   -> %s (BeforeTool)\n", filepath.Join(getGeminiConfigDir(), "settings.json"))
			fmt.Printf("                      skills  -> %s/ (%s)\n", filepath.Join(getHomeDir(), ".agents/skills"), geminiSkillsNote)
			anyYes = true
		} else {
			fmt.Printf("  %s    %s    (no gemini/gemini-cli in PATH, no config dir)\n", boldStyle.Render("gemini-cli"), errorStyle.Render("no"))
		}

		if antigravityOk {
			fmt.Printf("  %s   %s   context -> %s (shared with gemini-cli)\n", boldStyle.Render("antigravity"), successStyle.Render("yes"), filepath.Join(getGeminiConfigDir(), "GEMINI.md"))
			fmt.Printf("                      hooks   -> %s (BeforeTool, shared)\n", filepath.Join(getGeminiConfigDir(), "settings.json"))
			fmt.Printf("                      skills  -> %s/ (%s; copy on install)\n", filepath.Join(getHomeDir(), ".gemini/antigravity/skills"), skillsRepo)
			anyYes = true
		} else {
			fmt.Printf("  %s   %s    (no antigravity in PATH, no Antigravity app dirs, no ~/.antigravity-server)\n", boldStyle.Render("antigravity"), errorStyle.Render("no"))
		}

		if geminiOk && antigravityOk {
			fmt.Println(infoStyle.Render("\n  Note: gemini-cli and antigravity both use ~/.gemini/ (GEMINI.md, settings.json). One install updates both."))
		}

		if detectCursor(verbose) {
			fmt.Printf("  %s        %s   context -> %s (best-effort; see note)\n", boldStyle.Render("cursor"), successStyle.Render("yes"), filepath.Join(getHomeDir(), ".cursor/rules/global-agents.mdc"))
			fmt.Printf("                      hooks   -> %s (preToolUse)\n", filepath.Join(getHomeDir(), ".cursor/hooks.json"))
			fmt.Printf("                      skills  -> %s/ (%s; copy on install)\n", filepath.Join(getHomeDir(), ".cursor/skills"), skillsRepo)
			anyYes = true
		} else {
			fmt.Printf("  %s        %s    (no ~/.cursor or OS-specific Cursor app support dir)\n", boldStyle.Render("cursor"), errorStyle.Render("no"))
		}

		fmt.Println()
		if anyYes {
			fmt.Println(infoStyle.Render("Run: agents-cli install [--copy] [--dry-run] [--no-hooks] [--no-skills]"))
		} else {
			fmt.Println(errorStyle.Render("No known agent paths detected. Install tools first, or use install --targets to force paths."))
		}
		fmt.Println(infoStyle.Render("\nCursor note: Global User Rules may be cloud/UI-only. If .mdc is not picked up globally,"))
		fmt.Println(infoStyle.Render("  paste GLOBAL_AGENTS.md into Cursor Settings → Rules → User Rules, or sync from this file."))
	},
}

func init() {
	rootCmd.AddCommand(discoverCmd)
	discoverCmd.Flags().BoolP("verbose", "v", false, "Verbose output")
}
