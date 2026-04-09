package cmd

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func formatDiff(current, previous *float64, invertColor bool) string {
	if current == nil || previous == nil {
		return ""
	}
	diff := *current - *previous
	if math.Abs(diff) < 0.01 {
		return ""
	}

	sign := "+"
	if diff < 0 {
		sign = ""
	}
	valStr := fmt.Sprintf("%s%.2f", sign, diff)
	// if it is an integer, drop .00
	if diff == math.Trunc(diff) {
		valStr = fmt.Sprintf("%s%d", sign, int(diff))
	}

	return fmt.Sprintf("(%s)", valStr)
}

func scoreEmoji(score *float64) string {
	if score == nil {
		return "❓"
	}
	emojis := []string{"", "🔴", "🟠", "🟡", "🟢", "✅"}
	idx := int(math.Round(*score))
	if idx < 0 {
		idx = 0
	}
	if idx > 5 {
		idx = 5
	}
	return emojis[idx]
}

type evalData struct {
	AvgScore   *float64 `json:"avg_score"`
	TokenScore *float64 `json:"token_score"`
}

var ciCompareCmd = &cobra.Command{
	Use:   "ci-compare [MAIN_JSON] [PR_JSON] [OUTPUT_MD]",
	Short: "Compare evaluation results for CI",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		mainPath := args[0]
		prPath := args[1]
		outPath := args[2]

		mainHistory := make(map[string]evalData)
		if b, err := os.ReadFile(mainPath); err != nil {
			fmt.Printf("Main json not found at %s, proceeding blindly.\n", mainPath)
		} else {
			json.Unmarshal(b, &mainHistory)
		}

		prHistory := make(map[string]evalData)
		if b, err := os.ReadFile(prPath); err != nil {
			fmt.Printf("PR json not found at %s. Fail.\n", prPath)
			os.Exit(1)
		} else {
			json.Unmarshal(b, &prHistory)
		}

		var prScores []float64
		for _, v := range prHistory {
			if v.AvgScore != nil {
				prScores = append(prScores, *v.AvgScore)
			}
		}
		var prTotal float64 = 0
		if len(prScores) > 0 {
			var sum float64 = 0
			for _, s := range prScores {
				sum += s
			}
			prTotal = sum / float64(len(prScores))
		}

		var mainScores []float64
		for _, v := range mainHistory {
			if v.AvgScore != nil {
				mainScores = append(mainScores, *v.AvgScore)
			}
		}
		var mainTotal float64 = 0
		if len(mainScores) > 0 {
			var sum float64 = 0
			for _, s := range mainScores {
				sum += s
			}
			mainTotal = sum / float64(len(mainScores))
		}

		totalDiff := formatDiff(&prTotal, &mainTotal, false)

		lines := []string{
			"## Agent Prompt Evaluation",
			fmt.Sprintf("**Overall Score:** %s %.2f/5 %s", scoreEmoji(&prTotal), prTotal, totalDiff),
			"",
		}

		if prTotal < 4.0 {
			lines = append(lines, "> 🔴 **FAILED:** Overall score is below the 4.0 threshold.", "")
		} else if prTotal < mainTotal {
			lines = append(lines, fmt.Sprintf("> 🟠 **WARNING:** Quality regression detected! Score dropped from %.2f to %.2f.", mainTotal, prTotal), "")
		}

		lines = append(lines, "| Case | Score | Token Score |")
		lines = append(lines, "|------|-------|-------------|")

		for caseID, prData := range prHistory {
			mainData := mainHistory[caseID]

			scoreDf := formatDiff(prData.AvgScore, mainData.AvgScore, false)
			tokenScoreDf := formatDiff(prData.TokenScore, mainData.TokenScore, true)

			scStr := "ERR"
			if prData.AvgScore != nil {
				scStr = strings.TrimSpace(fmt.Sprintf("%s %.2f %s", scoreEmoji(prData.AvgScore), *prData.AvgScore, scoreDf))
			}

			tsVal := "?"
			if prData.TokenScore != nil {
				tsVal = fmt.Sprintf("%v", *prData.TokenScore)
			}
			tsStr := strings.TrimSpace(fmt.Sprintf("%s %s", tsVal, tokenScoreDf))

			lines = append(lines, fmt.Sprintf("| %s | %s | %s |", caseID, scStr, tsStr))
		}

		os.WriteFile(outPath, []byte(strings.Join(lines, "\n")), 0644)

		if prTotal < 4.0 {
			fmt.Println("Score below 4.0 threshold.")
			os.Exit(1)
		}

		fmt.Println("Evaluation checks passed.")
	},
}

func init() {
	rootCmd.AddCommand(ciCompareCmd)
}
