# .agents/

Portable skills and rules for AI agents.

[GLOBAL_AGENTS.md](GLOBAL_AGENTS.md) is the **source** for machine-wide context — deployed to `~/.claude/CLAUDE.md`, `~/.gemini/GEMINI.md` (Gemini CLI **and** Google Antigravity share this path), `~/.cursor/rules/global-agents.mdc`, etc.

**Editing:** Keep artifacts minimal and reusable; follow existing `SKILL.md` patterns.

**Split:** Identity, habits, machine-wide tools → [GLOBAL_AGENTS.md](GLOBAL_AGENTS.md); re-deploy after edits. Project-only content stays in that project's `AGENTS.md` or `.cursor/rules`.

## Deploy

[`scripts/install-global-agents.py`](scripts/install-global-agents.py) — Python 3.9+, stdlib-only.

```
./scripts/install-global-agents.py discover          # detect installed agents
./scripts/install-global-agents.py install            # symlink context to each agent
  --copy              # copy instead of symlink (Claude/Gemini)
  --targets claude,gemini,antigravity,cursor   # force targets if detection misses one (antigravity uses ~/.gemini/ like gemini)
  --with-hooks        # deploy rtk hooks + strip rtk instruction from context
  --with-skills       # copy skills/ to each agent's user skills dir (see below)
  --dry-run           # preview without writing
```

**Verify:** After deploy, run the prompts in [`docs/global-agents-smoke.md`](docs/global-agents-smoke.md) in a fresh chat per agent.

## Skills

[`skills/`](skills/) — one directory per skill with required [`SKILL.md`](skills/find-docs/SKILL.md) (YAML `name`, `description`, body). Optional: `scripts/`, `references/`, `assets/`.

With `--with-skills`, the installer **copies** into each installed agent’s user skills dir (same `--targets` as context): Claude `~/.claude/skills/<name>/`; Gemini / Antigravity `~/.gemini/skills/<name>/`; Cursor `~/.cursor/skills/<name>/`.

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
- `*-settings-snippet.json` — config fragments merged into each agent's settings by `install --with-hooks`.

When `--with-hooks` is used, the installer:
1. Copies hook scripts to `~/.agents-hooks/`.
2. Merges hook config into each agent's settings JSON (idempotent).
3. Strips `<!-- hookable: … -->` sections from deployed context so the model never sees the instruction.

Without `--with-hooks`, the full `GLOBAL_AGENTS.md` (including the `rtk` bullet) is deployed as-is.

## Venv

```
python3 -m venv scripts/.venv
./scripts/.venv/bin/pip install -r scripts/requirements.txt
```

`count-tokens.py` needs deps; `install-global-agents.py` is stdlib-only. **`./scripts/token-budget.sh`** writes [`.token-budget`](.token-budget) with token counts for `GLOBAL_AGENTS.md` and every `skills/**/*.md` (`cl100k_base`, same as `count-tokens.py` default). Refresh and stage `.token-budget` when counts change; **pre-commit** regenerates it and fails if the working tree differs from the index (run the script and `git add .token-budget`).

## Docs

```
ctx7 docs "/websites/platform_claude_en" "<question>"     # Claude Code
ctx7 docs "/google-gemini/gemini-cli" "<question>"         # Gemini CLI
ctx7 docs "/websites/cursor" "<question>"                  # Cursor
```
