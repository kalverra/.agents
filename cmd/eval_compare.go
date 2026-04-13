package cmd

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kalverra/agents/internal/eval"
	"github.com/kalverra/agents/internal/output"
)

var evalCompareCmd = &cobra.Command{
	Use:   "eval-compare <pr.json> <output.md>",
	Short: "Compare eval results against history.jsonl or main.json",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		prPath := args[0]
		outPath := args[1]

		var mainTotal float64
		var mainScores map[string]float64

		// Try to load baseline from history.jsonl first
		casesDir, _ := cmd.Flags().GetString("cases")
		historyLogPath := filepath.Join(filepath.Dir(casesDir), "history.jsonl")

		if last, err := eval.LoadLastSummary(historyLogPath); err == nil {
			commitStr := "unknown"
			if len(last.Commit) >= 7 {
				commitStr = last.Commit[:7]
			}
			output.Printf("Comparing against last record in history.jsonl (%s)\n", commitStr)
			mainTotal = last.AvgScore
			mainScores = last.Scores
		} else {
			// Fallback to explicit main.json if provided (legacy or CI)
			mainPath, _ := cmd.Flags().GetString("main")
			if mainPath != "" {
				mainHistory, err := eval.LoadHistory(mainPath)
				if err == nil {
					var scores []float64
					mainScores = make(map[string]float64)
					for k, v := range mainHistory {
						if v.AvgScore != nil {
							scores = append(scores, *v.AvgScore)
							mainScores[k] = *v.AvgScore
						}
					}
					if len(scores) > 0 {
						sum := 0.0
						for _, s := range scores {
							sum += s
						}
						mainTotal = sum / float64(len(scores))
					}
				}
			}
		}

		prHistory, err := eval.LoadHistory(prPath)
		if err != nil {
			return fmt.Errorf("PR json not found at %s", prPath)
		}

		var prScores []float64
		for _, v := range prHistory {
			if v.AvgScore != nil {
				prScores = append(prScores, *v.AvgScore)
			}
		}
		prTotal := 0.0
		if len(prScores) > 0 {
			for _, s := range prScores {
				prTotal += s
			}
			prTotal /= float64(len(prScores))
		}

		totalDiff := compareDiff(prTotal, mainTotal)
		scoreEmoji := evalScoreEmoji(prTotal)

		var b strings.Builder
		fmt.Fprintf(&b, "## Agent Prompt Evaluation\n")
		fmt.Fprintf(&b, "**Overall Score:** %s %.2f/5 %s\n\n", scoreEmoji, prTotal, totalDiff)

		if prTotal < 4.0 {
			b.WriteString("> FAILED: Overall score is below the 4.0 threshold.\n\n")
		} else if mainTotal > 0 && prTotal < mainTotal {
			fmt.Fprintf(
				&b,
				"> WARNING: Quality regression detected! Score dropped from %.2f to %.2f.\n\n",
				mainTotal,
				prTotal,
			)
		}

		b.WriteString("| Case | Score | Token Score |\n")
		b.WriteString("|------|-------|-------------|\n")

		for caseID, prData := range prHistory {
			scoreStr := "ERR"
			if prData.AvgScore != nil {
				prevScore := mainScores[caseID]
				diff := ""
				if prevScore > 0 {
					diff = compareDiff(*prData.AvgScore, prevScore)
				}
				scoreStr = fmt.Sprintf("%s %.2f %s", evalScoreEmoji(*prData.AvgScore), *prData.AvgScore, diff)
			}

			tsStr := "?"
			if prData.TokenScore != nil {
				tsStr = fmt.Sprintf("%d", *prData.TokenScore)
			}

			fmt.Fprintf(&b, "| %s | %s | %s |\n", caseID, strings.TrimSpace(scoreStr), tsStr)
		}

		if err := os.WriteFile(outPath, []byte(b.String()), 0o600); err != nil { //nolint:gosec
			return err
		}

		if prTotal < 4.0 {
			output.Println("Score below 4.0 threshold.")
			os.Exit(1)
		}

		output.Println("Evaluation checks passed.")
		return nil
	},
}

func evalScoreEmoji(score float64) string {
	rounded := int(math.Round(score))
	emojis := map[int]string{1: "1/5", 2: "2/5", 3: "3/5", 4: "4/5", 5: "5/5"}
	if e, ok := emojis[rounded]; ok {
		return e
	}
	return "?"
}

func compareDiff(current, previous float64) string {
	if previous == 0 {
		return ""
	}
	diff := current - previous
	if math.Abs(diff) < 0.01 {
		return ""
	}
	sign := "+"
	if diff < 0 {
		sign = ""
	}
	return fmt.Sprintf("(%s%.2f)", sign, diff)
}

func init() {
	evalCompareCmd.Flags().String("main", "", "Baseline results JSON (fallback)")
	evalCompareCmd.Flags().String("cases", filepath.Join("eval", "cases"), "Test cases directory for history.jsonl")
	rootCmd.AddCommand(evalCompareCmd)
}
