package cmd

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kalverra/agents/internal/eval"
)

var evalCompareCmd = &cobra.Command{
	Use:   "eval-compare <main.json> <pr.json> <output.md>",
	Short: "Compare eval results between main and PR branches for CI",
	Args:  cobra.ExactArgs(3),
	RunE: func(_ *cobra.Command, args []string) error {
		mainPath, prPath, outPath := args[0], args[1], args[2]

		mainHistory, err := eval.LoadHistory(mainPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Main json not found at %s, proceeding blindly.\n", mainPath)
			mainHistory = make(eval.History)
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

		var mainScores []float64
		for _, v := range mainHistory {
			if v.AvgScore != nil {
				mainScores = append(mainScores, *v.AvgScore)
			}
		}
		mainTotal := 0.0
		if len(mainScores) > 0 {
			for _, s := range mainScores {
				mainTotal += s
			}
			mainTotal /= float64(len(mainScores))
		}

		totalDiff := compareDiff(prTotal, mainTotal)
		scoreEmoji := evalScoreEmoji(prTotal)

		var b strings.Builder
		fmt.Fprintf(&b, "## Agent Prompt Evaluation\n")
		fmt.Fprintf(&b, "**Overall Score:** %s %.2f/5 %s\n\n", scoreEmoji, prTotal, totalDiff)

		if prTotal < 4.0 {
			b.WriteString("> FAILED: Overall score is below the 4.0 threshold.\n\n")
		} else if prTotal < mainTotal {
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
			mainData := mainHistory[caseID]

			scoreStr := "ERR"
			if prData.AvgScore != nil {
				diff := compareDiffPtr(prData.AvgScore, mainData.AvgScore)
				scoreStr = fmt.Sprintf("%s %.2f %s", evalScoreEmoji(*prData.AvgScore), *prData.AvgScore, diff)
			}

			tsStr := "?"
			if prData.TokenScore != nil {
				diff := compareDiffIntPtr(prData.TokenScore, mainData.TokenScore)
				tsStr = strings.TrimSpace(fmt.Sprintf("%d %s", *prData.TokenScore, diff))
			}

			fmt.Fprintf(&b, "| %s | %s | %s |\n", caseID, strings.TrimSpace(scoreStr), tsStr)
		}

		if err := os.WriteFile(outPath, []byte(b.String()), 0o600); err != nil { //nolint:gosec
			return err
		}

		if prTotal < 4.0 {
			fmt.Println("Score below 4.0 threshold.")
			os.Exit(1)
		}

		fmt.Println("Evaluation checks passed.")
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

func compareDiffPtr(current, previous *float64) string {
	if current == nil || previous == nil {
		return ""
	}
	return compareDiff(*current, *previous)
}

func compareDiffIntPtr(current, previous *int) string {
	if current == nil || previous == nil {
		return ""
	}
	diff := *current - *previous
	if diff == 0 {
		return ""
	}
	sign := "+"
	if diff < 0 {
		sign = ""
	}
	return fmt.Sprintf("(%s%d)", sign, diff)
}

func init() {
	rootCmd.AddCommand(evalCompareCmd)
}
