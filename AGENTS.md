# .agents/

Portable skills and rules for AI agents.

[GLOBAL_AGENTS.md](GLOBAL_AGENTS.md) is the **source** for machine-wide context — deployed to `~/.claude/CLAUDE.md`, `~/.gemini/GEMINI.md` (Gemini CLI **and** Google Antigravity share this path), `~/.cursor/rules/global-agents.mdc`, etc.

**Editing:** Keep artifacts minimal and reusable; follow existing `SKILL.md` patterns.

**Split:** Identity, habits, machine-wide tools → [GLOBAL_AGENTS.md](GLOBAL_AGENTS.md); re-deploy after edits. Project-only content stays in that project's `AGENTS.md` or `.cursor/rules`.

## Deploy

[`scripts/install-global-agents.py`](scripts/install-global-agents.py) — Python 3.9+; stdlib + repo-local [`scripts/hookable_markdown.py`](scripts/hookable_markdown.py) only (no PyPI deps).

```
./scripts/install-global-agents.py discover         # detect installed agents
./scripts/install-global-agents.py install         # context + hooks + skills (defaults)
  --copy              # copy instead of symlink (Claude/Gemini)
  --targets claude,gemini,antigravity,cursor   # force targets if detection misses one (antigravity uses ~/.gemini/ like gemini)
  --no-hooks          # skip hook deploy; full GLOBAL_AGENTS.md (keep hookable sections)
  --no-skills         # do not copy skills/ to user skills dirs
  --dry-run           # preview without writing
```

**Verify:** After deploy, run the prompts in [`docs/global-agents-smoke.md`](docs/global-agents-smoke.md) in a fresh chat per agent.

## Skills

[`skills/`](skills/) — one directory per skill with required [`SKILL.md`](skills/find-docs/SKILL.md) (YAML `name`, `description`, body). Optional: `scripts/`, `references/`, `assets/`.

By default (unless `install --no-skills`), the installer **copies** into each installed agent’s user skills dir (same `--targets` as context): Claude `~/.claude/skills/<name>/`; Gemini / Antigravity `~/.gemini/skills/<name>/`; Cursor `~/.cursor/skills/<name>/`.

Stick to portable frontmatter (`name`, `description`) for cross-tool behavior; optional agent-specific keys (e.g. Cursor `disable-model-invocation`, Claude `allowed-tools`) are ignored elsewhere.

This script does **not** touch project-level skills in a repo’s `.claude/skills/`, `.gemini/skills/`, or `.cursor/skills/`.

## Go Binaries

[`cmd/`](cmd/) contains Go CLI tools that back specific skills. Each subdirectory is a standalone Go module.

| Binary | Skill | Purpose |
|--------|-------|---------|
| `cmd/pr-review/` | `skills/pr-review/` | Fetch open PR context (reviews, threads, comments) via GitHub GraphQL and output LLM-friendly markdown. |

Build once in that module, then run by full path (not `PATH`):

```
cd cmd/pr-review && go build -o pr-review .
~/.agents/cmd/pr-review/pr-review --dir /path/to/repo   # adjust ~/.agents if needed
```

For development only you can use `go run .` in `cmd/pr-review`; the skill expects the built binary to avoid compile latency.

## Hooks

[`hooks/`](hooks/) contains execution-time hooks that enforce behaviors **without model instructions**. Currently:

- **`rtk` prepend** — transparently rewrites shell commands to include `rtk` before they run.

Architecture:
- `rtk-prepend.sh` — shared logic: reads JSON stdin, prepends `rtk` to the command, outputs agent-specific JSON. Requires `jq`; silently passes through if `rtk` or `jq` is missing.
- `claude-rtk.sh` / `gemini-rtk.sh` / `cursor-rtk.sh` — thin wrappers that set `AGENT_TYPE` and exec the shared script.
- `*-settings-snippet.json` — config fragments merged into each agent's settings on `install` (unless `--no-hooks`).

On a normal `install` (hooks on by default), the installer:
1. Copies hook scripts to `~/.agents-hooks/`.
2. Merges hook config into each agent's settings JSON (idempotent).
3. Strips `<!-- hookable: … -->` sections from deployed context so the model never sees the instruction.

With `install --no-hooks`, hook **regions stay** (e.g. the `rtk` bullet) but the `<!-- hookable … -->` / `<!-- /hookable … -->` lines are removed from the written file so the model does not see installer metadata. Symlink/copy is used only when the source has no such lines. Settings are not merged for hooks.

## Venv

```
python3 -m venv scripts/.venv
./scripts/.venv/bin/pip install -r scripts/requirements.txt
```

`count-tokens.py` needs deps; the installer has no PyPI deps (`hookable_markdown` ships in `scripts/`). **`./scripts/token-budget.sh`** writes [`.token-budget`](.token-budget) using `count-tokens.py --strip-hookable-markers` so `<!-- hookable … -->` lines are excluded (same as deployed text). Refresh and stage `.token-budget` when counts change; **pre-commit** enforces it.

## Docs

```
ctx7 docs "/websites/platform_claude_en" "<question>"     # Claude Code
ctx7 docs "/google-gemini/gemini-cli" "<question>"         # Gemini CLI
ctx7 docs "/websites/cursor" "<question>"                  # Cursor
```
