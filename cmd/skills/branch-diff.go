package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kalverra/agents/internal/git"
	"github.com/kalverra/agents/internal/output"
)

var branchDiffCmd = &cobra.Command{
	Use:   "branch-diff",
	Short: "Diff current branch against the default base branch, including worktree changes",
	RunE: func(cmd *cobra.Command, _ []string) error {
		dir, err := cmd.Flags().GetString("dir")
		if err != nil {
			return err
		}
		if dir == "" {
			dir = "."
		}

		base, err := cmd.Flags().GetString("base")
		if err != nil {
			return err
		}

		contextLines, err := cmd.Flags().GetInt("context")
		if err != nil {
			return err
		}

		result, err := git.BranchDiff(dir, git.BranchDiffOptions{
			Base:         base,
			ContextLines: contextLines,
		})
		if err != nil {
			return err
		}

		safeCurrent := strings.ReplaceAll(result.CurrentBranch, "/", "-")
		safeBase := strings.ReplaceAll(result.BaseBranch, "/", "-")
		patchName := fmt.Sprintf("%s_%s.patch", safeCurrent, safeBase)
		patchPath := filepath.Join(dir, patchName)

		content := git.FormatHuman(result)
		if err := os.WriteFile(patchPath, []byte(content), 0o600); err != nil {
			return fmt.Errorf("writing patch file: %w", err)
		}

		// Clear the raw patch from the result struct so it doesn't bloat JSON output
		result.Patch = ""

		output.Write("branch-diff", result, func() {
			output.Printf(
				"Saved branch diff to %s\nRead this file to review the changes.\nDelete it when you are done.\n",
				patchName,
			)
		})
		return nil
	},
}

func init() {
	branchDiffCmd.Flags().String("dir", "", "Working directory for git detection (default: .)")
	branchDiffCmd.Flags().String("base", "", "Base branch to compare against (default: auto-detect)")
	branchDiffCmd.Flags().Int("context", 3, "Number of context lines in diffs")
}
