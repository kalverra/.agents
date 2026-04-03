# `.agents`

A small home for **portable AI-agent configuration**: machine-wide rules you can version in git, sync to Claude Code, Gemini CLI, Google Antigravity, Cursor, and anything else you wire up later.

If you use more than one coding agent, this repo is the single place to edit “how they should behave by default,” then push those defaults onto disk with one command.

## What lives here

| Piece | Role |
|--------|------|
| **[`GLOBAL_AGENTS.md`](GLOBAL_AGENTS.md)** | Your **global** context: who you are, session habits, doc and web tooling. This is the file you edit most often. |
| **[`AGENTS.md`](AGENTS.md)** | **Repo meta**: how to deploy, how hooks work, venv for scripts. Orient here when changing the machinery, not the global prompt text. |
| **[`hooks/`](hooks/)** | **Runtime hooks** (optional): behaviors enforced when a tool runs, so the model does not have to “remember” them. |
| **[`scripts/install-global-agents.py`](scripts/install-global-agents.py)** | Installs `GLOBAL_AGENTS.md` to each tool’s expected path and (optionally) merges hook settings. |
| **[`docs/global-agents-smoke.md`](docs/global-agents-smoke.md)** | Manual smoke prompts to check that deployed context behaves the way you expect. |

**Global vs project:** Everything in `GLOBAL_AGENTS.md` is meant to apply everywhere. Repo- or project-specific rules stay in that project’s `AGENTS.md` or `.cursor/rules` so you do not bloat global context.

## Quick start

You need **Python 3.9+** for the installer (`install-global-agents.py` uses only the standard library).

From this directory:

```bash
./scripts/install-global-agents.py discover
./scripts/install-global-agents.py install
```

That detects which agents are present and **symlinks** `GLOBAL_AGENTS.md` into the usual locations (for example `~/.claude/CLAUDE.md`, `~/.gemini/GEMINI.md`, and a Cursor rule file under `~/.cursor/rules/`). **Antigravity** uses the same `~/.gemini/` files as Gemini CLI—one write updates both. Use `./scripts/install-global-agents.py install --copy` if you prefer copies instead of symlinks for Claude and Gemini.

**First time?** After install, open a **new** chat in each product and skim [the smoke doc](docs/global-agents-smoke.md) if you want a quick behavioral check.

## Hooks (optional)

Some instructions are better enforced **at execution time** than repeated in the model prompt. This repo supports that pattern for shell command rewriting—for example prepending your **`rtk`** CLI to compress command output.

```bash
./scripts/install-global-agents.py install --with-hooks
```

What that does, in plain terms:

1. Copies hook scripts to **`~/.agents-hooks/`**.
2. Merges small JSON fragments into each agent’s hook configuration (without wiping your existing hooks—see [`AGENTS.md`](AGENTS.md) for details).
3. **Strips** marked sections from the deployed copy of `GLOBAL_AGENTS.md` so the model is not told to do something the hook already does.

If you skip `--with-hooks`, the full `GLOBAL_AGENTS.md` is deployed, including any text between `<!-- hookable: … -->` comments in the source.

Details and file layout: **[`AGENTS.md` → Hooks](AGENTS.md#hooks)**.

## Optional: token counting

[`scripts/count-tokens.py`](scripts/count-tokens.py) estimates token counts for files or strings (uses `tiktoken`—not required for the installer).

```bash
python3 -m venv scripts/.venv
./scripts/.venv/bin/pip install -r scripts/requirements.txt
./scripts/.venv/bin/python scripts/count-tokens.py --help
```

## Tips

- Run **`discover`** whenever you install a new agent or change paths; it only prints where things would go.
- Use **`--dry-run`** on **`install`** to preview merges and writes without changing files.
- Use **`--targets claude,gemini,antigravity,cursor`** if detection misses a tool but you still want files written (`antigravity` deploys to the same paths as `gemini`).

For exact flags and contributor notes, keep **[`AGENTS.md`](AGENTS.md)** as the reference.
