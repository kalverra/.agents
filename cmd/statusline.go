package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// Customize the statusline for various CLI agents.
// antigravity-cli: https://antigravity.google/docs/cli-statusline
// claude-code: https://code.claude.com/docs/en/statusline

type statuslineModel struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type statuslineVCS struct {
	Type   string `json:"type"`
	Branch string `json:"branch"`
	Dirty  bool   `json:"dirty"`
}
type statuslineCurrentUsage struct {
	InputTokens              *int `json:"input_tokens"`
	OutputTokens             *int `json:"output_tokens"`
	ThinkingTokens           *int `json:"thinking_tokens"`
	ReasoningTokens          *int `json:"reasoning_tokens"`
	ThoughtsTokens           *int `json:"thoughts_tokens"`
	ThoughtsTokenCount       *int `json:"thoughts_token_count"`
	ReasoningTokenCount      *int `json:"reasoning_token_count"`
	ThinkingTokenCount       *int `json:"thinking_token_count"`
	TotalThinkingTokens      *int `json:"total_thinking_tokens"`
	CacheCreationInputTokens *int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     *int `json:"cache_read_input_tokens"`
}

type statuslineContextWindow struct {
	UsedPercentage       float64                 `json:"used_percentage"`
	TotalInputTokens     *int                    `json:"total_input_tokens"`
	TotalOutputTokens    *int                    `json:"total_output_tokens"`
	ThinkingTokens       *int                    `json:"thinking_tokens"`
	ReasoningTokens      *int                    `json:"reasoning_tokens"`
	ThoughtsTokens       *int                    `json:"thoughts_tokens"`
	ThoughtsTokenCount   *int                    `json:"thoughts_token_count"`
	ReasoningTokenCount  *int                    `json:"reasoning_token_count"`
	ThinkingTokenCount   *int                    `json:"thinking_token_count"`
	TotalThinkingTokens  *int                    `json:"total_thinking_tokens"`
	CacheReadInputTokens *int                    `json:"cache_read_input_tokens"`
	CurrentUsage         *statuslineCurrentUsage `json:"current_usage"`
}

type statuslineTask struct {
	Status string `json:"status"`
}

type statuslineSubagent struct {
	Status string `json:"status"`
}

type statuslineCost struct {
	TotalCostUSD    float64 `json:"total_cost_usd"`
	TotalDurationMS int64   `json:"total_duration_ms"`
}

type statuslinePR struct {
	Number      int    `json:"number"`
	URL         string `json:"url"`
	ReviewState string `json:"review_state"`
}

type statuslineInputPayload struct {
	Model           statuslineModel         `json:"model"`
	CWD             string                  `json:"cwd"`
	VCS             statuslineVCS           `json:"vcs"`
	ContextWindow   statuslineContextWindow `json:"context_window"`
	BackgroundTasks []statuslineTask        `json:"background_tasks"`
	Subagents       []statuslineSubagent    `json:"subagents"`
	AgentState      string                  `json:"agent_state"`
	Cost            statuslineCost          `json:"cost"`
	PR              *statuslinePR           `json:"pr"`
}

func runCmd(dir string, name string, args ...string) (string, error) {
	//nolint:gosec // G204: subprocess command generated dynamically
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

var getGitInfo = func(cwd string) (string, bool) {
	if _, err := runCmd(cwd, "git", "rev-parse", "--is-inside-work-tree"); err != nil {
		return "", false
	}
	branch, err := runCmd(cwd, "git", "branch", "--show-current")
	if err != nil || branch == "" {
		branch, _ = runCmd(cwd, "git", "rev-parse", "--abbrev-ref", "HEAD")
	}
	status, err := runCmd(cwd, "git", "status", "--porcelain")
	dirty := err == nil && status != ""
	return branch, dirty
}

func getThinkingTokens(cw statuslineContextWindow) *int {
	if cw.ThinkingTokens != nil {
		return cw.ThinkingTokens
	}
	if cw.ReasoningTokens != nil {
		return cw.ReasoningTokens
	}
	if cw.ThoughtsTokens != nil {
		return cw.ThoughtsTokens
	}
	if cw.ThoughtsTokenCount != nil {
		return cw.ThoughtsTokenCount
	}
	if cw.ReasoningTokenCount != nil {
		return cw.ReasoningTokenCount
	}
	if cw.ThinkingTokenCount != nil {
		return cw.ThinkingTokenCount
	}
	if cw.TotalThinkingTokens != nil {
		return cw.TotalThinkingTokens
	}

	if cw.CurrentUsage != nil {
		cu := cw.CurrentUsage
		if cu.ThinkingTokens != nil {
			return cu.ThinkingTokens
		}
		if cu.ReasoningTokens != nil {
			return cu.ReasoningTokens
		}
		if cu.ThoughtsTokens != nil {
			return cu.ThoughtsTokens
		}
		if cu.ThoughtsTokenCount != nil {
			return cu.ThoughtsTokenCount
		}
		if cu.ReasoningTokenCount != nil {
			return cu.ReasoningTokenCount
		}
		if cu.ThinkingTokenCount != nil {
			return cu.ThinkingTokenCount
		}
		if cu.TotalThinkingTokens != nil {
			return cu.TotalThinkingTokens
		}
	}
	return nil
}

func getInputTokens(cw statuslineContextWindow) *int {
	if cw.TotalInputTokens != nil {
		return cw.TotalInputTokens
	}
	if cw.CurrentUsage != nil {
		return cw.CurrentUsage.InputTokens
	}
	return nil
}

func getOutputTokens(cw statuslineContextWindow) *int {
	if cw.TotalOutputTokens != nil {
		return cw.TotalOutputTokens
	}
	if cw.CurrentUsage != nil {
		return cw.CurrentUsage.OutputTokens
	}
	return nil
}

func getCacheReadTokens(cw statuslineContextWindow) *int {
	if cw.CacheReadInputTokens != nil {
		return cw.CacheReadInputTokens
	}
	if cw.CurrentUsage != nil {
		return cw.CurrentUsage.CacheReadInputTokens
	}
	return nil
}

func parseAndBuildStatusline(jsonData []byte) (string, error) {
	var payload statuslineInputPayload
	if err := json.Unmarshal(jsonData, &payload); err != nil {
		return "", err
	}

	modelName := payload.Model.DisplayName
	if modelName == "" {
		modelName = payload.Model.ID
	}
	if modelName == "" {
		modelName = "No Model"
	}

	cwd := payload.CWD
	if cwd == "" {
		if dir, err := os.Getwd(); err == nil {
			cwd = dir
		}
	}
	if strings.HasPrefix(cwd, "/Users/adamhamrick") {
		cwd = strings.Replace(cwd, "/Users/adamhamrick", "~", 1)
	}

	var vcsStr string
	if payload.VCS.Type == "git" && payload.VCS.Branch != "" {
		dirty := ""
		if payload.VCS.Dirty {
			dirty = "*"
		}
		vcsStr = fmt.Sprintf(" (%s%s)", payload.VCS.Branch, dirty)
	} else {
		fallbackDir := payload.CWD
		if fallbackDir == "" {
			fallbackDir = cwd
		}
		if branch, dirty := getGitInfo(fallbackDir); branch != "" {
			dirtyStr := ""
			if dirty {
				dirtyStr = "*"
			}
			vcsStr = fmt.Sprintf(" (%s%s)", branch, dirtyStr)
		}
	}

	ctxDisplay := getContextDisplay(payload.ContextWindow)

	ansiReset := "\033[0m"
	ansiModel := "\033[1;36m"
	ansiCWD := "\033[1;33m"

	var parts []string
	parts = append(parts, fmt.Sprintf("%s%s%s", ansiModel, modelName, ansiReset))
	parts = append(parts, fmt.Sprintf("%s%s", ctxDisplay, ansiReset))
	parts = append(parts, fmt.Sprintf("%s%s%s%s", ansiCWD, cwd, vcsStr, ansiReset))

	if payload.AgentState != "" && payload.AgentState != "idle" {
		stateColors := map[string]string{
			"idle":         "\033[2;37m",
			"thinking":     "\033[1;33m",
			"working":      "\033[1;32m",
			"tool_use":     "\033[1;36m",
			"initializing": "\033[1;35m",
		}
		ansiState, exists := stateColors[payload.AgentState]
		if !exists {
			ansiState = "\033[1;31m"
		}
		parts = append(parts, fmt.Sprintf("%s%s%s", ansiState, payload.AgentState, ansiReset))
	}

	if payload.Cost.TotalCostUSD > 0 {
		costStr := fmt.Sprintf("💰 $%.2f", payload.Cost.TotalCostUSD)
		parts = append(parts, costStr)
	}

	return strings.Join(parts, " | "), nil
}

func getContextColor(usedPct float64) string {
	if usedPct <= 0.0 {
		return "\033[38;2;0;255;0m"
	}
	if usedPct >= 90.0 {
		return "\033[38;2;255;0;0m"
	}
	if usedPct <= 50.0 {
		t := usedPct / 50.0
		r := int(0.0 + t*255.0)
		g := int(255.0 - t*(255.0-60.0))
		b := 0
		return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
	}
	t := (usedPct - 50.0) / 40.0
	r := 255
	g := int(60.0 - t*60.0)
	b := 0
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

func formatTokenCount(n *int) string {
	if n == nil {
		return ""
	}
	val := float64(*n)
	if val >= 1000000.0 {
		return fmt.Sprintf("%.1fM", val/1000000.0)
	}
	if val >= 1000.0 {
		return fmt.Sprintf("%.1fk", val/1000.0)
	}
	return fmt.Sprintf("%d", *n)
}

func getContextDisplay(ctxData statuslineContextWindow) string {
	color := getContextColor(ctxData.UsedPercentage)
	var tokensParts []string

	inPtr := getInputTokens(ctxData)
	if inPtr != nil {
		tokensParts = append(tokensParts, fmt.Sprintf("▲ %s", formatTokenCount(inPtr)))
	}

	outPtr := getOutputTokens(ctxData)
	if outPtr != nil {
		tokensParts = append(tokensParts, fmt.Sprintf("▼ %s", formatTokenCount(outPtr)))
	}

	thinkPtr := getThinkingTokens(ctxData)
	if thinkPtr != nil {
		tokensParts = append(tokensParts, fmt.Sprintf("🧠 %s", formatTokenCount(thinkPtr)))
	}

	tokensParts = append(tokensParts, fmt.Sprintf("(%.1f%%)", ctxData.UsedPercentage))

	cacheReadPtr := getCacheReadTokens(ctxData)
	if cacheReadPtr != nil {
		tokensParts = append(tokensParts, fmt.Sprintf("💾 %s", formatTokenCount(cacheReadPtr)))
	}

	return fmt.Sprintf("%s%s", color, strings.Join(tokensParts, " "))
}

var statuslineCmd = &cobra.Command{
	Use:   "statusline",
	Short: "Render statusline for agent",
	RunE: func(_ *cobra.Command, _ []string) error {
		inputBytes, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Println("agy | error")
			return nil
		}
		out, err := parseAndBuildStatusline(inputBytes)
		if err != nil {
			fmt.Println("agy | error")
			return nil
		}
		fmt.Println(out)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statuslineCmd)
}
