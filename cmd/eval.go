package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/kalverra/agents/internal/eval"
	"github.com/kalverra/agents/internal/ui"
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
		record, _ := cmd.Flags().GetBool("record")
		repo, _ := cmd.Flags().GetString("repo")

		if repo == "" {
			repo = repoRoot()
		}

		evalDir := filepath.Dir(casesDir)
		historyPath := filepath.Join(evalDir, "eval_history.json")
		historyLogPath := filepath.Join(evalDir, "history.jsonl")
		localLogPath := filepath.Join(evalDir, "local.jsonl")
		spendPath := filepath.Join(evalDir, "eval_spend.json")

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
			AIOutput:     cfg.AIOutput,
		}

		history, err := eval.LoadHistory(historyPath)
		if err != nil {
			ui.WarnPrintf("could not load history: %v\n", err)
			history = make(eval.History)
		}

		ctx := context.Background()
		results, err := eval.Run(ctx, cfg)
		if err != nil {
			return err
		}

		if ui.AIOutput {
			return ui.PrintJSON(results)
		}

		eval.PrintSummary(results, iterations)

		// Create Summary for logs
		commit, _ := gitInfo(repo)
		summary := eval.CreateSummary(results, commit, subject, judge)

		// 1. Update Spend
		if err := updateSpend(spendPath, results); err != nil {
			ui.WarnPrintf("could not update spend: %v\n", err)
		}

		// 2. Archive results
		if err := archiveResults(evalDir, results); err != nil {
			ui.WarnPrintf("could not archive results: %v\n", err)
		}

		// 3. Log to local.jsonl
		if err := eval.AppendToJSONL(localLogPath, summary); err != nil {
			ui.WarnPrintf("could not log to local.jsonl: %v\n", err)
		}

		// 4. Log to history.jsonl if requested
		if record {
			if err := eval.AppendToJSONL(historyLogPath, summary); err != nil {
				ui.WarnPrintf("could not log to history.jsonl: %v\n", err)
			} else {
				ui.SuccessPrintf("Run recorded to history.jsonl\n")
			}
		}

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
			ui.SuccessPrintf("JSON written to %s\n", output)
		}

		if report != "" {
			if err := eval.WriteMarkdownReport(report, results, history, cfg); err != nil {
				return fmt.Errorf("writing report: %w", err)
			}
			ui.SuccessPrintf("Report written to %s\n", report)
		}

		return nil
	},
}

func updateSpend(path string, results []eval.Result) error {
	spend, err := eval.LoadSpend(path)
	if err != nil {
		return err
	}
	for _, r := range results {
		spend.CumulativeCost += r.Cost
	}
	return eval.SaveSpend(path, spend)
}

func archiveResults(evalDir string, results []eval.Result) error {
	archiveDir := filepath.Join(evalDir, "archive")
	if err := os.MkdirAll(archiveDir, 0o750); err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(archiveDir, fmt.Sprintf("run_%s.json", timestamp))

	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(filename, data, 0o600); err != nil {
		return err
	}

	return pruneArchive(archiveDir, 10)
}

func pruneArchive(dir string, keep int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	var files []os.FileInfo
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			info, err := entry.Info()
			if err == nil {
				files = append(files, info)
			}
		}
	}

	if len(files) <= keep {
		return nil
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime().After(files[j].ModTime())
	})

	for i := keep; i < len(files); i++ {
		_ = os.Remove(filepath.Join(dir, files[i].Name()))
	}

	return nil
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

func gitInfo(repoRoot string) (commit string, dirty bool) {
	//nolint:gosec // git binary is hardcoded, not user-controlled
	out, err := exec.Command("git", "-C", repoRoot, "rev-parse", "--short", "HEAD").
		Output()
	if err != nil {
		return "unknown", false
	}
	commit = strings.TrimSpace(string(out))

	//nolint:gosec // git binary is hardcoded, not user-controlled
	err = exec.Command("git", "-C", repoRoot, "diff", "--quiet", "--", "GLOBAL_AGENTS.md", "USER_AGENTS.md").
		Run()
	dirty = err != nil

	return commit, dirty
}

func init() {
	evalCmd.Flags().String("subject", "gemini-2.5-flash", "Gemini subject model")
	evalCmd.Flags().String("judge", "gemini-2.5-pro", "Gemini judge model")
	evalCmd.Flags().String("cases", filepath.Join("eval", "cases"), "Test cases directory")
	evalCmd.Flags().String("filter", "", "Only run cases with this tag")
	evalCmd.Flags().Int("iterations", 1, "Number of times to run each case")
	evalCmd.Flags().String("output", "", "Write JSON results to file")
	evalCmd.Flags().String("report", "", "Write markdown report to file")
	evalCmd.Flags().Bool("verbose", false, "Show full responses and judge feedback")
	evalCmd.Flags().Bool("check-dirty", false, "Validate history freshness (for pre-commit hooks)")
	evalCmd.Flags().Bool("record", false, "Append summary to history.jsonl")
	evalCmd.Flags().String("repo", "", "Repo root for resolving system_prompt_file paths")
	rootCmd.AddCommand(evalCmd)
}
