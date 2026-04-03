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
  --dry-run           # preview without writing
```

**Verify:** After deploy, run the prompts in [`docs/global-agents-smoke.md`](docs/global-agents-smoke.md) in a fresh chat per agent.

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

`count-tokens.py` needs deps; `install-global-agents.py` is stdlib-only.

## Docs

```
ctx7 docs "/websites/platform_claude_en" "<question>"     # Claude Code
ctx7 docs "/google-gemini/gemini-cli" "<question>"         # Gemini CLI
ctx7 docs "/websites/cursor" "<question>"                  # Cursor
```
