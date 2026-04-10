package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kalverra/agents/internal/agent"
)

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "List detected agents and their install paths",
	RunE: func(cmd *cobra.Command, _ []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")
		home := homeDir()

		fmt.Println("Discovery (signals only — tools need not be running):")
		fmt.Println()

		skillCount := countRepoSkills()
		skillsNote := fmt.Sprintf("%d in repo skills/", skillCount)
		if skillCount == 0 {
			skillsNote = "none in repo skills/"
		}

		anyFound := false

		if agent.Detect(agent.Claude, verbose) {
			fmt.Printf("  claude-code   yes   context -> %s\n", filepath.Join(home, ".claude", "CLAUDE.md"))
			fmt.Printf(
				"                      hooks   -> %s (PreToolUse)\n",
				filepath.Join(home, ".claude", "settings.json"),
			)
			fmt.Printf("                      skills  -> %s/ (%s; copy on install)\n",
				filepath.Join(home, ".claude", "skills"), skillsNote)
			anyFound = true
		} else {
			fmt.Println("  claude-code   no    (no claude in PATH, no ~/.claude)")
		}

		geminiOK := agent.Detect(agent.Gemini, verbose)
		antigravityOK := agent.Detect(agent.Antigravity, verbose)
		geminiDir := agent.GeminiConfigDir()

		if geminiOK {
			fmt.Printf("  gemini-cli    yes   context -> %s\n", filepath.Join(geminiDir, "GEMINI.md"))
			fmt.Printf("                      hooks   -> %s (BeforeTool)\n", filepath.Join(geminiDir, "settings.json"))
			fmt.Printf("                      skills  -> ~/.agents/skills/ (%s; Gemini discovers here)\n", skillsNote)
			anyFound = true
		} else {
			fmt.Println("  gemini-cli    no    (no gemini/gemini-cli in PATH, no config dir)")
		}

		if antigravityOK {
			fmt.Printf("  antigravity   yes   context -> %s (shared with gemini-cli)\n",
				filepath.Join(geminiDir, "GEMINI.md"))
			fmt.Printf("                      hooks   -> %s (BeforeTool, shared)\n",
				filepath.Join(geminiDir, "settings.json"))
			fmt.Printf("                      skills  -> %s/ (%s; copy on install)\n",
				filepath.Join(home, ".gemini", "antigravity", "skills"), skillsNote)
			anyFound = true
		} else {
			fmt.Println("  antigravity   no    (no antigravity in PATH, no Antigravity app dirs)")
		}

		if geminiOK && antigravityOK {
			fmt.Println()
			fmt.Println(
				"  Note: gemini-cli and antigravity share ~/.gemini/ (GEMINI.md, settings.json). One install updates both.",
			)
		}

		if agent.Detect(agent.Cursor, verbose) {
			fmt.Printf("  cursor        yes   context -> %s (best-effort)\n",
				filepath.Join(home, ".cursor", "rules", "global-agents.mdc"))
			fmt.Printf("                      hooks   -> %s (preToolUse)\n",
				filepath.Join(home, ".cursor", "hooks.json"))
			fmt.Printf("                      skills  -> %s/ (%s; copy on install)\n",
				filepath.Join(home, ".cursor", "skills"), skillsNote)
			anyFound = true
		} else {
			fmt.Println("  cursor        no    (no ~/.cursor or OS-specific Cursor app support dir)")
		}

		fmt.Println()
		if anyFound {
			fmt.Println("Run: agents install [--copy] [--dry-run] [--no-hooks] [--no-skills]")
		} else {
			fmt.Println("No known agent paths detected. Install tools first, or use install --targets to force paths.")
		}

		return nil
	},
}

func init() {
	discoverCmd.Flags().BoolP("verbose", "v", false, "Show detection details")
	rootCmd.AddCommand(discoverCmd)
}
