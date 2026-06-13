package skills

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kalverra/agents/internal/git"
	gh "github.com/kalverra/agents/internal/github"
	"github.com/kalverra/agents/internal/output"
	"github.com/kalverra/agents/internal/review"
)

type prReviewResult struct {
	ReviewPath      string         `json:"review_path"`
	PRNumber        int            `json:"pr_number,omitempty"`
	PRURL           string         `json:"pr_url,omitempty"`
	CurrentBranch   string         `json:"current_branch"`
	BaseBranch      string         `json:"base_branch"`
	MergeBase       string         `json:"merge_base"`
	Head            string         `json:"head"`
	HasLocalChanges bool           `json:"has_local_changes"`
	UnresolvedCount int            `json:"unresolved_thread_count"`
	Stats           git.DiffStats  `json:"stats"`
	Files           []git.FileDiff `json:"files"`
}

var prReviewCmd = &cobra.Command{
	Use:   "pr-review",
	Short: "Bundle open PR review threads and branch diff into one review file",
	RunE: func(cmd *cobra.Command, _ []string) error {
		dir, err := cmd.Flags().GetString("dir")
		if err != nil {
			return err
		}
		if dir == "" {
			dir = "."
		}

		owner, err := cmd.Flags().GetString("owner")
		if err != nil {
			return err
		}
		repo, err := cmd.Flags().GetString("repo")
		if err != nil {
			return err
		}
		branch, err := cmd.Flags().GetString("branch")
		if err != nil {
			return err
		}
		base, err := cmd.Flags().GetString("base")
		if err != nil {
			return err
		}
		contextLines, err := cmd.Flags().GetInt("context")
		if err != nil {
			return err
		}
		includeResolved, err := cmd.Flags().GetBool("include-resolved")
		if err != nil {
			return err
		}

		if owner == "" || repo == "" || branch == "" {
			info, detectErr := git.DetectRepo(dir)
			if detectErr != nil {
				return fmt.Errorf("detecting repo: %w", detectErr)
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

		diffBase := base
		if diffBase == "" && pr != nil {
			diffBase = pr.BaseRef
		}

		diff, err := git.BranchDiff(dir, git.BranchDiffOptions{
			Base:         diffBase,
			ContextLines: contextLines,
		})
		if err != nil {
			return fmt.Errorf("reading branch diff: %w", err)
		}

		content := review.FormatBundle(pr, diff, includeResolved)
		filename := review.Filename(pr, diff)
		reviewPath := filepath.Join(dir, filename)

		if err := os.WriteFile(reviewPath, []byte(content), 0o600); err != nil {
			return fmt.Errorf("writing review file: %w", err)
		}

		unresolved := 0
		if pr != nil {
			for _, thread := range pr.Threads {
				if !thread.IsResolved {
					unresolved++
				}
			}
		}

		for i := range diff.Files {
			diff.Files[i].Patch = ""
		}

		result := prReviewResult{
			ReviewPath:      filename,
			CurrentBranch:   diff.CurrentBranch,
			BaseBranch:      diff.BaseBranch,
			MergeBase:       diff.MergeBase,
			Head:            diff.Head,
			HasLocalChanges: diff.HasLocalChanges,
			UnresolvedCount: unresolved,
			Stats:           diff.Stats,
			Files:           diff.Files,
		}
		if pr != nil {
			result.PRNumber = pr.Number
			result.PRURL = pr.URL
		}

		output.Write("pr-review", result, func() {
			output.Printf(
				"Saved review bundle to %s\nRead this file to review the PR and code changes.\nDelete it when you are done.\n",
				filename,
			)
		})
		return nil
	},
}

func init() {
	prReviewCmd.Flags().String("dir", "", "Working directory for git detection (default: .)")
	prReviewCmd.Flags().String("owner", "", "GitHub repo owner (default: detected from git remote)")
	prReviewCmd.Flags().String("repo", "", "GitHub repo name (default: detected from git remote)")
	prReviewCmd.Flags().String("branch", "", "Branch name (default: current branch)")
	prReviewCmd.Flags().String("base", "", "Base branch override for diff (default: PR base or auto-detect)")
	prReviewCmd.Flags().Int("context", 3, "Number of context lines in diffs")
	prReviewCmd.Flags().Bool("include-resolved", false, "Include resolved review threads")
}
