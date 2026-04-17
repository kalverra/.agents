package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kalverra/agents/internal/agent"
	"github.com/kalverra/agents/internal/output"
)

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "List detected agents and their install paths",
	RunE: func(cmd *cobra.Command, _ []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")
		home := homeDir()

		skillCount := countRepoSkills()
		skillsNote := fmt.Sprintf("%d in repo skills/", skillCount)
		if skillCount == 0 {
			skillsNote = "none in repo skills/"
		}

		type info struct {
			Name     string `json:"name"`
			Detected bool   `json:"detected"`
			Context  string `json:"context,omitempty"`
			Hooks    string `json:"hooks,omitempty"`
			Skills   string `json:"skills,omitempty"`
			Note     string `json:"note,omitempty"`
		}

		var results []info

		// Claude
		c := info{Name: "claude-code", Detected: agent.Detect(agent.Claude, verbose)}
		if c.Detected {
			c.Context = filepath.Join(home, ".claude", "CLAUDE.md")
			c.Hooks = filepath.Join(home, ".claude", "settings.json")
			c.Skills = filepath.Join(home, ".claude", "skills")
		}
		results = append(results, c)

		// Gemini
		geminiOK := agent.Detect(agent.Gemini, verbose)
		geminiDir := agent.GeminiConfigDir()
		g := info{Name: "gemini-cli", Detected: geminiOK}
		if g.Detected {
			g.Context = filepath.Join(geminiDir, "GEMINI.md")
			g.Hooks = filepath.Join(geminiDir, "settings.json")
			g.Skills = fmt.Sprintf("%s; %s", agent.SkillsDest(agent.Gemini), agent.SkillsDest(agent.Antigravity))
		}
		results = append(results, g)

		// Antigravity
		antigravityOK := agent.Detect(agent.Antigravity, verbose)
		a := info{Name: "antigravity", Detected: antigravityOK}
		if a.Detected {
			a.Context = filepath.Join(geminiDir, "GEMINI.md")
			a.Hooks = filepath.Join(geminiDir, "settings.json")
			a.Skills = fmt.Sprintf("%s; %s", agent.SkillsDest(agent.Gemini), agent.SkillsDest(agent.Antigravity))
		}
		results = append(results, a)

		// Cursor
		cur := info{Name: "cursor", Detected: agent.Detect(agent.Cursor, verbose)}
		if cur.Detected {
			cur.Context = filepath.Join(home, ".cursor", "rules", "global-agents.mdc")
			cur.Hooks = filepath.Join(home, ".cursor", "hooks.json")
			cur.Skills = filepath.Join(home, ".cursor", "skills")
		}
		results = append(results, cur)

		output.Write("discover", results, func() {
			output.Println("Discovery (signals only — tools need not be running):")
			output.Println()

			for _, r := range results {
				if r.Detected {
					output.Printf("  %-13s yes   context -> %s\n", r.Name, r.Context)
					if r.Hooks != "" {
						output.Printf("                      hooks   -> %s\n", r.Hooks)
					}
					if r.Skills != "" {
						output.Printf("                      skills  -> %s (%s)\n", r.Skills, skillsNote)
					}
				} else {
					output.Printf("  %-13s no\n", r.Name)
				}
			}

			output.Println()
			anyFound := false
			for _, r := range results {
				if r.Detected {
					anyFound = true
					break
				}
			}

			if anyFound {
				output.Println("Run: agents install [--copy] [--dry-run] [--no-hooks] [--no-skills]")
			} else {
				output.Println(
					"No known agent paths detected. Install tools first, or use install --targets to force paths.",
				)
			}
		})

		return nil
	},
}

func init() {
	discoverCmd.Flags().BoolP("verbose", "v", false, "Show detection details")
	rootCmd.AddCommand(discoverCmd)
}
