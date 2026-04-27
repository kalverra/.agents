package cmd

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kalverra/agents/internal/agent"
	"github.com/kalverra/agents/internal/output"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Deploy agent instructions, hooks, and skills to detected tools",
	RunE: func(cmd *cobra.Command, _ []string) error {
		useCopy, _ := cmd.Flags().GetBool("copy")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		verbose, _ := cmd.Flags().GetBool("verbose")
		targetsStr, _ := cmd.Flags().GetString("targets")
		noHooks, _ := cmd.Flags().GetBool("no-hooks")
		noSkills, _ := cmd.Flags().GetBool("no-skills")
		yes, _ := cmd.Flags().GetBool("yes")

		inst := &agent.Installer{
			RepoRoot:   repoRoot(),
			DryRun:     dryRun,
			Yes:        yes,
			UseCopy:    useCopy,
			Verbose:    verbose,
			WithHooks:  !noHooks,
			WithSkills: !noSkills,
			Targets:    agent.ParseTargets(targetsStr),
		}

		report, err := inst.Install()
		if err != nil {
			return err
		}

		output.Write("install", report, nil)
		return nil
	},
}

func init() {
	installCmd.Flags().Bool("copy", false, "Copy instead of symlink")
	installCmd.Flags().Bool("dry-run", false, "Print actions only")
	installCmd.Flags().BoolP("verbose", "v", false, "Show detection details")
	installCmd.Flags().
		String("targets", "", "Comma-separated: claude,gemini,antigravity,cursor,codex (default: detected only)")
	installCmd.Flags().Bool("no-hooks", false, "Skip hook deploy and settings merge")
	installCmd.Flags().Bool("no-skills", false, "Skip skill directory copies")
	installCmd.Flags().BoolP("yes", "y", false, "Skip install confirmation prompt")
	rootCmd.AddCommand(installCmd)
}

// repoRoot finds the repo root by looking for GLOBAL_AGENTS.md.
// Falls back to the executable's directory, then cwd.
func repoRoot() string {
	// Check if cwd contains GLOBAL_AGENTS.md
	if cwd, err := os.Getwd(); err == nil {
		if _, err := os.Stat(filepath.Join(cwd, "GLOBAL_AGENTS.md")); err == nil {
			return cwd
		}
	}

	// Check executable's directory
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if _, err := os.Stat(filepath.Join(dir, "GLOBAL_AGENTS.md")); err == nil {
			return dir
		}
	}

	// Default to cwd
	cwd, _ := os.Getwd()
	return cwd
}

// homeDir returns the user's home directory.
func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return h
}

// countRepoSkills counts skill directories with SKILL.md.
func countRepoSkills() int {
	skillsRoot := filepath.Join(repoRoot(), "skills")
	entries, err := os.ReadDir(skillsRoot)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() && e.Name()[0] != '.' {
			if _, err := os.Stat(filepath.Join(skillsRoot, e.Name(), "SKILL.md")); err == nil {
				n++
			}
		}
	}
	return n
}
