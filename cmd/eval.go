package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kalverra/agents/internal/eval"
)

var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Run the prompt evaluation harness",
	RunE: func(cmd *cobra.Command, _ []string) error {
		subject, _ := cmd.Flags().GetString("subject")
		judge, _ := cmd.Flags().GetString("judge")
		casesDir, _ := cmd.Flags().GetString("cases")
		filter, _ := cmd.Flags().GetString("filter")
		iterations, _ := cmd.Flags().GetInt("iterations")
		output, _ := cmd.Flags().GetString("output")
		report, _ := cmd.Flags().GetString("report")
		verbose, _ := cmd.Flags().GetBool("verbose")
		checkDirty, _ := cmd.Flags().GetBool("check-dirty")
		repo, _ := cmd.Flags().GetString("repo")

		if repo == "" {
			repo = repoRoot()
		}

		historyPath := filepath.Join(filepath.Dir(casesDir), "eval_history.json")

		if checkDirty {
			return runCheckDirty(repo, historyPath)
		}

		cfg := eval.RunConfig{
			SubjectModel: subject,
			JudgeModel:   judge,
			CasesDir:     casesDir,
			TagFilter:    filter,
			Iterations:   iterations,
			RepoRoot:     repo,
			Verbose:      verbose,
		}

		history, err := eval.LoadHistory(historyPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not load history: %v\n", err)
			history = make(eval.History)
		}

		ctx := context.Background()
		results, err := eval.Run(ctx, cfg)
		if err != nil {
			return err
		}

		eval.PrintSummary(results, iterations)

		eval.MergeResults(history, results)
		if err := eval.SaveHistory(historyPath, history); err != nil {
			return fmt.Errorf("saving history: %w", err)
		}

		if output != "" {
			data, err := json.MarshalIndent(results, "", "  ")
			if err != nil {
				return err
			}
			if err := os.WriteFile(output, data, 0o600); err != nil { //nolint:gosec
				return err
			}
			fmt.Printf("JSON written to %s\n", output)
		}

		if report != "" {
			if err := eval.WriteMarkdownReport(report, results, history, cfg); err != nil {
				return fmt.Errorf("writing report: %w", err)
			}
			fmt.Printf("Report written to %s\n", report)
		}

		return nil
	},
}

func runCheckDirty(repo, historyPath string) error {
	if _, err := os.Stat(historyPath); err != nil {
		return fmt.Errorf("no eval_history.json found. Run `agents eval` before committing")
	}

	history, err := eval.LoadHistory(historyPath)
	if err != nil {
		return fmt.Errorf("corrupt eval_history.json: %w", err)
	}

	globalPath := filepath.Join(repo, "GLOBAL_AGENTS.md")
	if fi, err := os.Stat(globalPath); err == nil {
		histFi, err := os.Stat(historyPath)
		if err == nil && fi.ModTime().After(histFi.ModTime()) {
			return fmt.Errorf("GLOBAL_AGENTS.md modified since last eval. Run `agents eval` to update scores")
		}
	}

	var scores []float64
	for _, v := range history {
		if v.AvgScore != nil {
			scores = append(scores, *v.AvgScore)
		}
	}
	if len(scores) == 0 {
		return fmt.Errorf("no scores in eval_history.json")
	}
	total := 0.0
	for _, s := range scores {
		total += s
	}
	avg := total / float64(len(scores))
	if avg < 4.0 {
		return fmt.Errorf("overall average score is %.2f. Must be >= 4.0 to commit", avg)
	}

	return nil
}

func init() {
	evalCmd.Flags().String("subject", "gemini-2.5-flash", "Gemini subject model")
	evalCmd.Flags().String("judge", "gemini-2.5-pro", "Gemini judge model")
	evalCmd.Flags().String("cases", filepath.Join("scripts", "eval", "cases"), "Test cases directory")
	evalCmd.Flags().String("filter", "", "Only run cases with this tag")
	evalCmd.Flags().Int("iterations", 1, "Number of times to run each case")
	evalCmd.Flags().String("output", "", "Write JSON results to file")
	evalCmd.Flags().String("report", "", "Write markdown report to file")
	evalCmd.Flags().Bool("verbose", false, "Show full responses and judge feedback")
	evalCmd.Flags().Bool("check-dirty", false, "Validate history freshness (for pre-commit hooks)")
	evalCmd.Flags().String("repo", "", "Repo root for resolving system_prompt_file paths")
	rootCmd.AddCommand(evalCmd)
}
