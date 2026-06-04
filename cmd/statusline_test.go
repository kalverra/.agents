package cmd

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetContextColor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		usedPct  float64
		expected string
	}{
		{"0%", 0.0, "\033[38;2;0;255;0m"},
		{"50%", 50.0, "\033[38;2;255;60;0m"},
		{"90%", 90.0, "\033[38;2;255;0;0m"},
		{"95%", 95.0, "\033[38;2;255;0;0m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := getContextColor(tt.usedPct)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestFormatTokenCount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		n        *int
		expected string
	}{
		{nil, ""},
		{new(500), "500"},
		{new(88244), "88.2k"},
		{new(61074), "61.1k"},
		{new(1250000), "1.2M"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.n), func(t *testing.T) {
			t.Parallel()
			got := formatTokenCount(tt.n)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestGetContextDisplay(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		ctxData  statuslineContextWindow
		expected string
	}{
		{
			name: "Percentage Only",
			ctxData: statuslineContextWindow{
				UsedPercentage: 50.0,
			},
			expected: "\033[38;2;255;60;0m(50.0%)",
		},
		{
			name: "With Tokens",
			ctxData: statuslineContextWindow{
				UsedPercentage:    50.0,
				TotalInputTokens:  new(88244),
				TotalOutputTokens: new(61074),
			},
			expected: "\033[38;2;255;60;0m▲ 88.2k ▼ 61.1k (50.0%)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := getContextDisplay(tt.ctxData)
			assert.Equal(t, stripAnsi(tt.expected), stripAnsi(got))
		})
	}
}

func stripAnsi(str string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(str, "")
}

//nolint:paralleltest // cannot run in parallel as it modifies package-level getGitInfo
func TestParseAndBuildStatusline(t *testing.T) {
	oldGetGitInfo := getGitInfo
	defer func() { getGitInfo = oldGetGitInfo }()
	getGitInfo = func(_ string) (string, bool) {
		return "mock-git-branch", true
	}

	tests := []struct {
		name      string
		jsonInput string
		expected  string
	}{
		{
			name: "Antigravity Payload",
			jsonInput: `{
				"model": {
					"id": "gemini-3.5-flash",
					"display_name": "Gemini 3.5 Flash"
				},
				"cwd": "/Users/adamhamrick/Projects/testrig",
				"vcs": {
					"type": "git",
					"branch": "main",
					"dirty": true
				},
				"context_window": {
					"used_percentage": 15.5,
					"total_input_tokens": 15000,
					"total_output_tokens": 500,
					"thinking_tokens": 200
				},
				"agent_state": "working"
			}`,
			expected: "Gemini 3.5 Flash | ▲ 15.0k ▼ 500 🧠 200 (15.5%) | ~/Projects/testrig (main*) | working",
		},
		{
			name: "Claude Payload",
			jsonInput: `{
				"model": {
					"id": "claude-3-5-sonnet",
					"display_name": "Sonnet 3.5"
				},
				"cwd": "/Users/adamhamrick/Projects/testrig",
				"context_window": {
					"used_percentage": 42.0,
					"total_input_tokens": 42000,
					"total_output_tokens": 1200
				},
				"cost": {
					"total_cost_usd": 0.0543,
					"total_duration_ms": 125000
				}
			}`,
			expected: "Sonnet 3.5 | ▲ 42.0k ▼ 1.2k (42.0%) | ~/Projects/testrig (mock-git-branch*) | 💰 $0.05",
		},
		{
			name: "Claude Payload with total_thinking_tokens",
			jsonInput: `{
				"model": {
					"id": "claude-3-5-sonnet",
					"display_name": "Sonnet 3.5"
				},
				"cwd": "/Users/adamhamrick/Projects/testrig",
				"context_window": {
					"used_percentage": 42.0,
					"total_input_tokens": 42000,
					"total_output_tokens": 1200,
					"total_thinking_tokens": 800
				},
				"cost": {
					"total_cost_usd": 0.0543,
					"total_duration_ms": 125000
				}
			}`,
			expected: "Sonnet 3.5 | ▲ 42.0k ▼ 1.2k 🧠 800 (42.0%) | ~/Projects/testrig (mock-git-branch*) | 💰 $0.05",
		},
		{
			name: "Idle State Payload",
			jsonInput: `{
				"model": {
					"id": "gemini-3.5-flash",
					"display_name": "Gemini 3.5 Flash"
				},
				"cwd": "/Users/adamhamrick/Projects/testrig",
				"context_window": {
					"used_percentage": 10.0
				},
				"agent_state": "idle"
			}`,
			expected: "Gemini 3.5 Flash | (10.0%) | ~/Projects/testrig (mock-git-branch*)",
		},
		{
			name: "Antigravity thoughts_token_count",
			jsonInput: `{
				"model": {
					"id": "gemini-3.5-flash",
					"display_name": "Gemini 3.5 Flash"
				},
				"cwd": "/Users/adamhamrick/Projects/testrig",
				"context_window": {
					"used_percentage": 15.5,
					"total_input_tokens": 15000,
					"total_output_tokens": 500,
					"thoughts_token_count": 300
				}
			}`,
			expected: "Gemini 3.5 Flash | ▲ 15.0k ▼ 500 🧠 300 (15.5%) | ~/Projects/testrig (mock-git-branch*)",
		},
		{
			name: "Antigravity nested thoughts_token_count",
			jsonInput: `{
				"model": {
					"id": "gemini-3.5-flash",
					"display_name": "Gemini 3.5 Flash"
				},
				"cwd": "/Users/adamhamrick/Projects/testrig",
				"context_window": {
					"used_percentage": 15.5,
					"total_input_tokens": 15000,
					"total_output_tokens": 500,
					"current_usage": {
						"thoughts_token_count": 400
					}
				}
			}`,
			expected: "Gemini 3.5 Flash | ▲ 15.0k ▼ 500 🧠 400 (15.5%) | ~/Projects/testrig (mock-git-branch*)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAndBuildStatusline([]byte(tt.jsonInput))
			require.NoError(t, err)
			assert.Equal(t, tt.expected, stripAnsi(got))
		})
	}
}
