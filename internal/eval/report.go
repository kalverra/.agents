package eval

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/kalverra/agents/internal/output"
)

var scoreBadge = map[int]string{
	1: "1/5",
	2: "2/5",
	3: "3/5",
	4: "4/5",
	5: "5/5",
}

// PrintSummary prints a text summary table of results.
func PrintSummary(results []Result, iterations int) {
	var avgScores []float64
	var totalTokensIn, totalTokensOut int
	var totalCost float64

	for _, r := range results {
		if r.AvgScore != nil {
			avgScores = append(avgScores, *r.AvgScore)
		}
		if r.InputTokens != nil {
			totalTokensIn += *r.InputTokens
		}
		totalTokensOut += r.AvgOutputTokens
		totalCost += r.Cost
	}

	totalAvg := 0.0
	if len(avgScores) > 0 {
		for _, s := range avgScores {
			totalAvg += s
		}
		totalAvg /= float64(len(avgScores))
	}
	passed := 0
	for _, s := range avgScores {
		if s >= 4.0 {
			passed++
		}
	}

	output.Println("\n--- Results ---")
	for _, r := range results {
		if r.Error != "" {
			output.Printf("  %s: ERR (%s)\n", r.Case, r.Error)
			continue
		}
		if iterations > 1 {
			avg := "ERR"
			if r.AvgScore != nil {
				avg = fmt.Sprintf("%.2f", *r.AvgScore)
			}
			output.Printf("  %s: Avg=%s Min=%v Max=%v In=%v Out=%d Cost=$%.6f\n",
				r.Case, avg, ptrStr(r.MinScore), ptrStr(r.MaxScore),
				ptrStr(r.InputTokens), r.AvgOutputTokens, r.Cost)
		} else {
			avg := "ERR"
			if r.AvgScore != nil {
				avg = fmt.Sprintf("%.0f/5", *r.AvgScore)
			}
			output.Printf("  %s: %s In=%v Out=%d Cost=$%.6f\n",
				r.Case, avg, ptrStr(r.InputTokens), r.AvgOutputTokens, r.Cost)
		}
	}
	output.Printf("\nOverall Average: %.2f/5  |  Passed (Avg >= 4.0): %d/%d\n", totalAvg, passed, len(avgScores))
	output.Printf("Total Cost: $%.6f (in=%d out=%d)\n", totalCost, totalTokensIn, totalTokensOut)
}

// WriteMarkdownReport writes a full eval report as markdown.
func WriteMarkdownReport(path string, results []Result, history History, cfg RunConfig) error {
	var avgScores []float64
	var totalTokensIn, totalTokensOut int
	var totalCost float64

	for _, r := range results {
		if r.AvgScore != nil {
			avgScores = append(avgScores, *r.AvgScore)
		}
		if r.InputTokens != nil {
			totalTokensIn += *r.InputTokens
		}
		totalTokensOut += r.AvgOutputTokens
		totalCost += r.Cost
	}

	totalAvg := 0.0
	if len(avgScores) > 0 {
		for _, s := range avgScores {
			totalAvg += s
		}
		totalAvg /= float64(len(avgScores))
	}
	passed := 0
	for _, s := range avgScores {
		if s >= 4.0 {
			passed++
		}
	}

	now := time.Now().UTC().Format("2006-01-02 15:04 UTC")
	commit, dirty := gitInfo(cfg.RepoRoot)
	gitStatus := fmt.Sprintf("`%s`", commit)
	if dirty {
		gitStatus += " (with local changes)"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Eval Results\n\n")
	fmt.Fprintf(&b, "**Run:** %s  \n", now)
	fmt.Fprintf(&b, "**Commit:** %s  \n", gitStatus)
	fmt.Fprintf(&b, "**Subject:** `%s`  \n", cfg.SubjectModel)
	fmt.Fprintf(&b, "**Judge:** `%s`  \n", cfg.JudgeModel)
	fmt.Fprintf(&b, "**Cases:** %d  \n", len(results))
	fmt.Fprintf(&b, "**Iterations:** %d  \n", cfg.Iterations)
	fmt.Fprintf(
		&b,
		"**Average score:** %.2f/5  |  **Passed (Avg >= 4.0):** %d/%d  \n",
		totalAvg,
		passed,
		len(avgScores),
	)
	fmt.Fprintf(&b, "**Total Cost:** $%.6f (in=%d out=%d)\n\n", totalCost, totalTokensIn, totalTokensOut)
	b.WriteString("---\n\n## Summary\n\n")

	// Summary table
	if cfg.Iterations > 1 {
		b.WriteString("| # | Case | Tags | In | Out | Cost | Avg | Min | Max |\n")
		b.WriteString("|---|------|------|----|-----|------|-----|-----|-----|\n")
	} else {
		b.WriteString("| # | Case | Tags | In | Out | Cost | Score |\n")
		b.WriteString("|---|------|------|----|-----|------|-------|\n")
	}

	for i, r := range results {
		tags := formatTags(r.Tags)
		if r.Error != "" {
			if cfg.Iterations > 1 {
				fmt.Fprintf(&b, "| %d | %s | %s | ERR | ERR | ERR | ERR | ERR | ERR |\n", i+1, r.Case, tags)
			} else {
				fmt.Fprintf(&b, "| %d | %s | %s | ERR | ERR | ERR | error |\n", i+1, r.Case, tags)
			}
			continue
		}

		prev := history[r.Case]
		scoreDiff := formatDiff(r.AvgScore, prev.AvgScore)
		// tokenScoreDiff := formatDiffInt(r.TokenScore, prev.TokenScore)

		inTok := ptrStr(r.InputTokens)
		outTok := fmt.Sprintf("%d", r.AvgOutputTokens)
		costStr := fmt.Sprintf("$%.6f", r.Cost)

		if cfg.Iterations > 1 {
			scoreStr := "ERR"
			badge := "?"
			if r.AvgScore != nil {
				scoreStr = fmt.Sprintf("%.2f", *r.AvgScore)
				badge = scoreBadge[int(math.Round(*r.AvgScore))]
			}
			fmt.Fprintf(&b, "| %d | %s | %s | %s | %s | %s | %s %s %s | %s | %s |\n",
				i+1, r.Case, tags, inTok, outTok, costStr, badge, scoreStr, scoreDiff,
				ptrStr(r.MinScore), ptrStr(r.MaxScore))
		} else {
			badge := "?"
			if r.AvgScore != nil {
				badge = scoreBadge[int(*r.AvgScore)]
			}
			fmt.Fprintf(&b, "| %d | %s | %s | %s | %s | %s | %s %s |\n",
				i+1, r.Case, tags, inTok, outTok, costStr, badge, scoreDiff)
		}
	}

	// Case details
	b.WriteString("\n---\n\n## Case Details\n\n")
	for _, r := range results {
		tags := strings.Join(r.Tags, " ")
		fmt.Fprintf(&b, "### %s\n\n%s  \n", r.Case, tags)

		prev := history[r.Case]
		costDiff := formatDiff(new(r.Cost), new(prev.Cost))
		fmt.Fprintf(
			&b,
			"**Cost:** $%.6f %s (in=%s out=%d)  \n",
			r.Cost,
			costDiff,
			ptrStr(r.InputTokens),
			r.AvgOutputTokens,
		)
		fmt.Fprintf(&b, "**Description:** %s\n\n", r.Description)

		scoreDiff := formatDiff(r.AvgScore, prev.AvgScore)
		if cfg.Iterations > 1 && r.AvgScore != nil {
			badge := scoreBadge[int(math.Round(*r.AvgScore))]
			fmt.Fprintf(&b, "**Average Score:** %s %.2f %s (Min: %s, Max: %s)  \n",
				badge, *r.AvgScore, scoreDiff, ptrStr(r.MinScore), ptrStr(r.MaxScore))
		} else if r.AvgScore != nil {
			badge := scoreBadge[int(*r.AvgScore)]
			fmt.Fprintf(&b, "**Score:** %s %s  \n", badge, scoreDiff)
		}

		fmt.Fprintf(&b, "#### User Message\n\n```\n%s\n```\n\n", strings.TrimSpace(r.UserMessage))

		if r.Error != "" {
			fmt.Fprintf(&b, "> **Error:** %s\n\n---\n\n", r.Error)
			continue
		}

		for _, it := range r.Iterations {
			header := "Response"
			if cfg.Iterations > 1 {
				header = fmt.Sprintf("Iteration %d", it.Num)
			}
			scoreStr := "?"
			if it.Score != nil {
				scoreStr = scoreBadge[*it.Score]
			}
			fmt.Fprintf(&b, "#### %s (`%s`)\n**Score:** %s | **Cost:** $%.6f\n\n%s\n\n**Judge Feedback**\n\n%s\n\n",
				header, r.SubjectModel, scoreStr, it.Cost,
				strings.TrimSpace(it.SubjectResponse), strings.TrimSpace(it.JudgeRaw))
		}
		b.WriteString("---\n\n")
	}

	return os.WriteFile(path, []byte(b.String()), 0o600) //nolint:gosec
}

func formatTags(tags []string) string {
	var parts []string
	for _, t := range tags {
		parts = append(parts, fmt.Sprintf("`%s`", t))
	}
	return strings.Join(parts, ", ")
}

func formatDiff(current, previous *float64) string {
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
	return fmt.Sprintf("(%s%.2f)", sign, diff)
}

func ptrStr(p *int) string {
	if p == nil {
		return "?"
	}
	return fmt.Sprintf("%d", *p)
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
