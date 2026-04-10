// Package skills contains skill-focused commands and flows
package skills

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kalverra/agents/internal/git"
	gh "github.com/kalverra/agents/internal/github"
)

var fetchPRCmd = &cobra.Command{
	Use:   "fetch-pr",
	Short: "Fetch and format the open PR for the current branch",
	RunE: func(cmd *cobra.Command, _ []string) error {
		owner, _ := cmd.Flags().GetString("owner")
		repo, _ := cmd.Flags().GetString("repo")
		branch, _ := cmd.Flags().GetString("branch")
		includeResolved, _ := cmd.Flags().GetBool("include-resolved")
		dir, _ := cmd.Flags().GetString("dir")

		if dir == "" {
			dir = "."
		}

		if owner == "" || repo == "" || branch == "" {
			info, err := git.DetectRepo(dir)
			if err != nil {
				return fmt.Errorf("detecting repo: %w", err)
			}
			if owner == "" {
				owner = info.Owner
			}
			if repo == "" {
				repo = info.Name
			}
			if branch == "" {
				branch = info.Branch
			}
		}

		token, err := gh.ResolveToken()
		if err != nil {
			return err
		}

		client := gh.NewClient(token)
		ctx := context.Background()

		pr, err := gh.FetchPR(ctx, client, owner, repo, branch)
		if err != nil {
			return fmt.Errorf("fetching PR: %w", err)
		}
		if pr == nil {
			fmt.Fprintf(os.Stderr, "No open PR found for %s/%s branch %s\n", owner, repo, branch)
			return nil
		}

		fmt.Print(gh.FormatPR(pr, includeResolved))
		return nil
	},
}

func init() {
	fetchPRCmd.Flags().String("owner", "", "GitHub repo owner (default: detected from git remote)")
	fetchPRCmd.Flags().String("repo", "", "GitHub repo name (default: detected from git remote)")
	fetchPRCmd.Flags().String("branch", "", "Branch name (default: current branch)")
	fetchPRCmd.Flags().String("dir", "", "Working directory for git detection (default: .)")
	fetchPRCmd.Flags().Bool("include-resolved", false, "Include resolved review threads")
}
