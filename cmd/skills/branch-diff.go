package skills

import (
	"github.com/spf13/cobra"
)

var branchDiffCmd = &cobra.Command{
	Use:   "branch-diff",
	Short: "Compare the current branch to the main branch",
}

func init() {
	branchDiffCmd.Flags().String("base", "main", "The base branch to compare against")
}
