# `.agents`

A small home for **portable AI-agent configuration**: machine-wide rules you can version in git, sync to Claude Code, Gemini CLI, Google Antigravity, Cursor, and anything else you wire up later.

If you use more than one coding agent, this repo is the single place to edit “how they should behave by default,” then push those defaults onto disk with one command.

## What lives here

| Piece                                                                      | Role                                                                                                                                 |
| -------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| **[`GLOBAL_AGENTS.md`](GLOBAL_AGENTS.md)**                                 | Your **global** context: who you are, session habits, doc and web tooling. This is the file you edit most often.                     |
| **[`AGENTS.md`](AGENTS.md)**                                               | **Repo meta**: how to deploy, how hooks work, venv for scripts. Orient here when changing the machinery, not the global prompt text. |
| **[`hooks/`](hooks/)**                                                     | **Runtime hooks**: installed with `install` by default (`--no-hooks` to skip).                                                       |
| **[`skills/`](skills/)**                                                   | **Agent skills**: reusable workflows in `SKILL.md` form; copied on `install` by default (`--no-skills` to skip).                     |
| **[`cmd/`](cmd/)**                                                         | **Go binaries** that back specific skills (e.g. `pr-review`). Each subdirectory is a standalone Go module.                           |
| **[`scripts/install-global-agents.py`](scripts/install-global-agents.py)** | Installs context, hooks, and skills by default; `--no-hooks` / `--no-skills` to skip.                                                |
| **[`docs/global-agents-smoke.md`](docs/global-agents-smoke.md)**           | Manual smoke prompts to check that deployed context behaves the way you expect.                                                      |
| **[`.token-budget`](.token-budget)**                                       | A simple count of how many tokens each md file adds to your agents' global contexts                                                  |

**Global vs project:** Everything in `GLOBAL_AGENTS.md` is meant to apply everywhere. Repo- or project-specific rules stay in that project’s `AGENTS.md` or `.cursor/rules` so you do not bloat global context.

## Quick start

You need **Python 3.9+** for the installer (stdlib + a small repo-local helper in `scripts/`; no extra `pip` packages).

From this directory:

```bash
./scripts/install-global-agents.py discover
./scripts/install-global-agents.py install
```

That detects which agents are present, **symlinks** (or copies with `--copy`) `GLOBAL_AGENTS.md`, **deploys hooks** (unless `--no-hooks`), and **copies skills** (unless `--no-skills`). **Antigravity** uses the same `~/.gemini/` files as Gemini CLI—one write updates both.

**First time?** After install, open a **new** chat in each product and skim [the smoke doc](docs/global-agents-smoke.md) if you want a quick behavioral check.

## Hooks

Some instructions are better enforced **at execution time** than repeated in the model prompt—for example prepending your **`rtk`** CLI to compress command output. A normal **`install`** does this by default.

1. Copies hook scripts to **`~/.agents-hooks/`**.
2. Merges small JSON fragments into each agent’s hook configuration (without wiping your existing hooks—see [`AGENTS.md`](AGENTS.md) for details).
3. **Strips** marked sections from the deployed copy of `GLOBAL_AGENTS.md` so the model is not told to do something the hook already does.

Use **`install --no-hooks`** to deploy the full `GLOBAL_AGENTS.md` (including `<!-- hookable: … -->` sections) and skip hook merges.

Details and file layout: **[`AGENTS.md` → Hooks](AGENTS.md#hooks)**.

## Agent skills

Skills are directories under [`skills/`](skills/) with a **`SKILL.md`** file (frontmatter + instructions). **`install`** copies them into each product’s user skills folder by default. Skip with **`install --no-skills`**. Combine with **`--targets`** as needed. Authoring details: **[`AGENTS.md` → Skills](AGENTS.md#skills)**.

Some skills rely on a Go program under [`cmd/`](cmd/). For example, `pr-review` fetches GitHub PR context via GraphQL—build `cmd/pr-review/pr-review` once, then run it by full path (see [`skills/pr-review/SKILL.md`](skills/pr-review/SKILL.md)); no need to put it on `PATH`.

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
- Use **`--no-skills`** if you only want global context + hooks without copying [`skills/`](skills/).

For exact flags and contributor notes, keep **[`AGENTS.md`](AGENTS.md)** as the reference.

## Contributing

```sh
pre-commit install
```
