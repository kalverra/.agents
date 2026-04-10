# Go CLI Refactor and Python-to-Go Conversion Plan

## Overview

Complete the partial Go CLI refactor (wire up fetch-pr subcommand) and convert five Python scripts into Go subcommands under the existing Cobra/Viper architecture. Work is ordered to keep the binary compiling at every step.

---

## Phase 1: Fix Existing Go Code and Wire Up fetch-pr

### 1.1 Fix cfg nil dereference in cmd/root.go

Problem: Line 35 does StringVarP with cfg.LogLevel but cfg is a nil *config.Config at init() time. This will panic on startup.

Fix: Initialize cfg as var cfg = config.Config{} at package level, providing a valid pointer for flag binding. PersistentPreRunE then overwrites it with the fully-loaded config.

File: /Users/adamhamrick/.agents/cmd/root.go

### 1.2 Move format.go from cmd/pr-review/ to internal/github/

format.go has package main and references types (PR, Review, ReviewThread, Comment) already defined in internal/github/github.go. Move it to internal/github/format.go, change to package github. Add tests at internal/github/format_test.go.

### 1.3 Create internal/github/auth.go

Port from deleted cmd/pr-review/main.go: ResolveToken() (string, error) checks GITHUB_TOKEN, GH_TOKEN, then gh auth token. NewClient(token string) creates the oauth2-backed GraphQL client. Promotes golang.org/x/oauth2 from indirect to direct.

### 1.4 Implement cmd/fetch_pr.go

Register fetchPRCmd with Use fetch-pr. Flags: --include-resolved (bool), --dir (string). RunE: ResolveToken, DetectRepo, NewClient, FetchPR, FormatPR, print stdout.

### 1.5 Verify compilation and tests

---

## Phase 2: Shared Internal Packages

### 2.1 internal/markdown/ package

hookable.go (port hookable_markdown.py): StripHookableSections, StripHookableDelimiterLines.
merge.go (port user_agents_merge.py): MergeUserAgents(body, userSrcPath) (string, error).
Both with _test.go files using table-driven tests.

### 2.2 internal/agent/ package for detection

detect.go: Agent type constants (Claude, Gemini, Antigravity, Cursor), DetectedAgent struct, DetectAll and per-agent functions.
paths.go: GeminiConfigDir, HooksDeployDir, per-agent destination paths.

### 2.3 internal/agent/install.go

Installer struct (RepoRoot, DryRun, UseCopy, Verbose, WithHooks, WithSkills, Targets). Methods: DeployGlobalAgentsMarkdown, WriteCursorMDC, DeployHookScripts, MergeHooksIntoSettings, CopySkillsToUserDir, WriteLocalMergedGlobal.

---

## Phase 3: CLI Subcommands

### 3.1 cmd/discover.go - agents discover [--verbose]
### 3.2 cmd/install.go - agents install [--copy] [--dry-run] [--verbose] [--targets LIST] [--no-hooks] [--no-skills]
### 3.3 Add RepoRoot and GithubToken to internal/config/config.go

---

## Phase 4: Eval Harness

### 4.1 internal/eval/ package (6 files + tests)

case.go: Case/Criteria structs, LoadCases, LoadSystemPrompt
gemini.go: Raw HTTP Gemini client (no SDK needed)
judge.go: PrometheusPrompt, FormatPrometheusPrompt, ParseScore
runner.go: RunConfig, Result, two-phase Run function
report.go: WriteMarkdownReport, PrintSummary
history.go: HistoryEntry, LoadHistory, SaveHistory, MergeResults

### 4.2 cmd/eval.go - agents eval [all Python flags] including --check-dirty
### 4.3 cmd/eval_compare.go - agents eval compare (3 positional args)

---

## Phase 5: Token Counting

Use github.com/pkoukk/tiktoken-go. Create cmd/count_tokens.go.

---

## Phase 6: Justfile and Cleanup

Update justfile, .gitignore, AGENTS.md. Remove cmd/pr-review/ and Python scripts.

---

## Implementation Order

1. Fix cmd/root.go nil cfg
2. Move format.go + create auth.go
3. Wire up fetch-pr subcommand
4. Create internal/markdown/
5. Create internal/agent/detect.go + paths.go
6. Create cmd/discover.go
7. Create internal/agent/install.go
8. Create cmd/install.go
9. Create cmd/count_tokens.go
10. Create internal/eval/
11. Create cmd/eval.go + cmd/eval_compare.go
12. Update justfile, .gitignore, AGENTS.md
13. Clean up old files

---

## Key Decisions

- hookable/merge -> internal/markdown/ (shared by install and eval)
- format.go -> internal/github/ (co-located with data model)
- Raw HTTP for Gemini (zero new deps, httptest testable)
- tiktoken-go for token counting
- New deps: promote oauth2 and yaml.v3, add tiktoken-go
