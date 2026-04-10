# Plan: Complete Go CLI Refactor & Convert Python Scripts

## Context

The project is mid-refactor from a Python + standalone Go binary setup into a unified Go CLI (`agents`). The Cobra/Viper skeleton exists with working internal packages (`config`, `git`, `github`) but no subcommands are wired up. Python scripts (`install-global-agents.py`, `eval.py`, `count-tokens.py`, etc.) need to be ported to Go subcommands. The old `cmd/pr-review/` standalone binary has been deleted from git but `format.go` remains orphaned there.

## Phase 1: Fix Existing Go Code & Wire Up `fetch-pr`

### 1.1 Fix nil `cfg` in `cmd/root.go`
- Line 16: `var cfg *config.Config` is nil, but line 35 dereferences it in `init()` via `StringVarP(&cfg.LogLevel, ...)`
- Fix: `var cfg = &config.Config{}`
- File: `cmd/root.go`

### 1.2 Move `cmd/pr-review/format.go` -> `internal/github/format.go`
- Change `package main` to `package github`
- Types (`PR`, `Review`, etc.) are already in `internal/github/github.go` — no import changes needed
- Delete `cmd/pr-review/` directory entirely

### 1.3 Create `internal/github/auth.go`
- Port from deleted `cmd/pr-review/main.go` (recoverable via `git show HEAD:cmd/pr-review/main.go`)
- `ResolveToken() (string, error)` — checks `GITHUB_TOKEN`, `GH_TOKEN`, then `gh auth token`
- `NewClient(token string) *githubv4.Client` — oauth2 HTTP client wrapper
- Promote `golang.org/x/oauth2` from indirect to direct

### 1.4 Implement `cmd/fetch_pr.go`
- Replace empty stub with Cobra subcommand
- Chain: `ResolveToken()` -> `DetectRepo(dir)` -> `NewClient()` -> `FetchPR()` -> `FormatPR()` -> stdout
- Flags: `--dir` (string, "."), `--include-resolved` (bool)

## Phase 2: Shared Internal Packages

### 2.1 Create `internal/markdown/` package

**`internal/markdown/hookable.go`** (port of `scripts/hookable_markdown.py`):
- `StripHookableSections(text string) string`
- `StripHookableDelimiterLines(text string) string`

**`internal/markdown/merge.go`** (port of `scripts/user_agents_merge.py`):
- `MergeUserAgents(body, userSrcPath string) (string, error)`

Tests: `hookable_test.go`, `merge_test.go` with table-driven tests

### 2.2 Create `internal/agent/` package

**`internal/agent/detect.go`**:
- `Agent` type with constants: `Claude`, `Gemini`, `Antigravity`, `Cursor`
- `Detect(agent Agent, verbose bool) bool` for each agent
- Uses `exec.LookPath` + `os.Stat` + `runtime.GOOS`

**`internal/agent/paths.go`**:
- `GeminiConfigDir() string` (respects `GEMINI_CONFIG_DIR` env)
- `HooksDeployDir() string` -> `~/.agents-hooks`
- Per-agent destination paths for markdown, hooks settings, skills

### 2.3 Create `internal/agent/install.go`
Port of `scripts/install-global-agents.py` deployment logic:
- `DeployMarkdown()` — merge user agents, process hookable sections, write/symlink
- `DeployHookScripts()` — copy shell scripts, chmod 0755
- `MergeHooksIntoSettings()` — load JSON snippet, replace `<HOOKS_DIR>`, idempotent merge
- `CopySkills()` — copy skill dirs containing SKILL.md
- `WriteCursorMDC()` — generate .mdc with YAML frontmatter
- `WriteLocalMergedGlobal()` — gitignored merged snapshot

## Phase 3: Install & Discover Commands

### 3.1 `cmd/discover.go`
- `agents discover [--verbose]`
- Calls detection functions, formats output matching current Python output

### 3.2 `cmd/install.go`
- `agents install [--copy] [--dry-run] [--verbose] [--targets LIST] [--no-hooks] [--no-skills]`
- Same conditional logic as Python `cmd_install()`

## Phase 4: Token Counting

### 4.1 `cmd/count_tokens.go`
- `agents count-tokens [FILE] [-s TEXT] [--encoding NAME] [--model NAME] [--strip-hookable-markers] [-v]`
- New dependency: `github.com/pkoukk/tiktoken-go`
- Reads file/stdin/string, optionally strips hookable markers, counts and prints

## Phase 5: Eval Harness

### 5.1 `internal/eval/` package

**`case.go`**: `Case`, `Criteria` structs, `LoadCases(dir, tagFilter)`, `LoadSystemPrompt(case, repoRoot)`
- Promote `gopkg.in/yaml.v3` to direct dependency

**`gemini.go`**: Gemini API client using official Go SDK `google.golang.org/genai`
- `CallSubject(model, systemPrompt, userMessage) (response string, outputTokens int, err error)`
- `CallJudge(judgeModel, case, subjectResponse) (feedback string, score *int, err error)`
- Config: temperature 0.0, safety settings BLOCK_NONE
- Exponential backoff retry

**`judge.go`**: Prometheus prompt template, `FormatPrometheusPrompt()`, `ParseScore(text) *int`

**`runner.go`**: Two-phase orchestrator (generate responses, then evaluate). Returns `[]Result`

**`report.go`**: `WriteMarkdownReport()`, `PrintSummary()` with history diffs

**`history.go`**: `LoadHistory()`/`SaveHistory()` for `eval_history.json`, `MergeResults()`

### 5.2 `cmd/eval.go`
- `agents eval [flags]`
- Flags: `--subject`, `--judge`, `--cases`, `--filter`, `--iterations`, `--output`, `--report`, `--verbose`, `--check-dirty`, `--repo`

### 5.3 `cmd/eval_compare.go`
- `agents eval compare <main.json> <pr.json> <output.md>`
- Port of `scripts/eval/ci_compare.py`

## Phase 6: Justfile, Cleanup & Wiring

### 6.1 Update `justfile`
- Replace all `{{ python }}` invocations with `go run .` or built binary
- Remove `setup` recipe (no more venv needed)
- Keep eval case YAML files at `scripts/eval/cases/` (or move to `testdata/eval/cases/`)

### 6.2 Delete Python files
- Remove `scripts/*.py`, `scripts/requirements.txt`, `scripts/.venv/`, `scripts/__pycache__/`
- Keep `scripts/eval/cases/*.yaml` and `scripts/eval/eval_history.json`

### 6.3 Update config
- Update `.gitignore` (add `agents` binary, remove Python patterns)
- Update `AGENTS.md` with new commands
- Update `.github/workflows/eval-pr.yaml` to use Go binary
- Move eval cases to a better location if desired (e.g., `testdata/eval/cases/`)

## New Dependencies
- `golang.org/x/oauth2` — promote from indirect (GitHub auth)
- `gopkg.in/yaml.v3` — promote from indirect (eval YAML)
- `github.com/pkoukk/tiktoken-go` — new (token counting)
- `google.golang.org/genai` — new (Gemini API)

## Implementation Order
Build incrementally, keeping the binary compiling at each step:

1. Fix `cmd/root.go` nil cfg
2. Move format.go, create auth.go, delete cmd/pr-review/
3. Wire up fetch-pr subcommand -> **verify: `go build && ./agents fetch-pr`**
4. Create `internal/markdown/` with tests -> **verify: `go test ./internal/markdown/`**
5. Create `internal/agent/detect.go` + `paths.go`
6. Create `cmd/discover.go` -> **verify: `./agents discover`**
7. Create `internal/agent/install.go`
8. Create `cmd/install.go` -> **verify: `./agents install --dry-run`**
9. Create `cmd/count_tokens.go` -> **verify: `./agents count-tokens GLOBAL_AGENTS.md`**
10. Create `internal/eval/` (all files)
11. Create `cmd/eval.go` + `cmd/eval_compare.go` -> **verify: `./agents eval --cases scripts/eval/cases/`**
12. Update justfile, cleanup Python, update docs
13. Final `go test ./...` and `golangci-lint run`

## Verification
- `go build .` succeeds at each step
- `go test ./...` passes
- `golangci-lint run` clean
- `./agents fetch-pr` works in a repo with an open PR
- `./agents discover` detects installed agents
- `./agents install --dry-run` prints expected actions
- `./agents count-tokens GLOBAL_AGENTS.md` outputs token count
- `./agents eval --cases scripts/eval/cases/ --iterations 1` runs eval (needs GEMINI_API_KEY)
