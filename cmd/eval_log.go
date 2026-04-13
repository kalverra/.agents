package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kalverra/agents/internal/eval"
	"github.com/kalverra/agents/internal/output"
)

var evalLogCmd = &cobra.Command{
	Use:   "eval-log",
	Short: "Show recent evaluation runs from local.jsonl",
	RunE: func(cmd *cobra.Command, _ []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		casesDir, _ := cmd.Flags().GetString("cases")

		localLogPath := filepath.Join(filepath.Dir(casesDir), "local.jsonl")

		data, err := os.ReadFile(localLogPath) //nolint: gosec
		if err != nil {
			if os.IsNotExist(err) {
				output.Println("No local evaluation history found.")
				return nil
			}
			return err
		}

		rawLines := splitLines(string(data))
		if len(rawLines) == 0 {
			output.Println("No local evaluation history found.")
			return nil
		}

		var summaries []eval.RunSummary
		for _, line := range rawLines {
			var s eval.RunSummary
			if err := json.Unmarshal([]byte(line), &s); err == nil {
				summaries = append(summaries, s)
			}
		}

		// Sort by timestamp descending
		sort.Slice(summaries, func(i, j int) bool {
			return summaries[i].Timestamp > summaries[j].Timestamp
		})

		if len(summaries) > limit {
			summaries = summaries[:limit]
		}

		output.Write("eval-log", summaries, func() {
			output.Printf("%-20s | %-8s | %-5s | %-7s | %-15s\n", "Timestamp", "Commit", "Score", "Passed", "Subject")
			output.Println(strings.Repeat("-", 65))

			for _, s := range summaries {
				ts := s.Timestamp
				if len(ts) > 19 {
					ts = ts[:19] // YYYY-MM-DDTHH:MM:SS
				}
				commit := s.Commit
				if len(commit) > 7 {
					commit = commit[:7]
				}
				output.Printf("%-20s | %-8s | %-5.2f | %d/%-5d | %-15s\n",
					ts, commit, s.AvgScore, s.PassedCount, s.TotalCases, s.Subject)
			}
		})

		return nil
	},
}

func splitLines(s string) []string {
	var lines []string
	var start int
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if line != "" {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	return lines
}

func init() {
	evalLogCmd.Flags().Int("limit", 10, "Number of recent runs to show")
	evalLogCmd.Flags().String("cases", filepath.Join("eval", "cases"), "Test cases directory")
	rootCmd.AddCommand(evalLogCmd)
}
