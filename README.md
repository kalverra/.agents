# `~/.agents`

A global home for all of your AI agents! Define global rules once, and sync them to Claude Code, Gemini CLI, Antigravity, Cursor, and any other agents you use.

## Quick Start

Clone this repo into your home directory, then run the install script to detect what agents you use and install the global rules, hooks, and skills to them. **WARNING**: This will overwrite any existing `CLAUDE.md`, `GEMINI.md`, `.cursor/global-rules.mdc`, etc. files.

### Install Dependencies

You'll need to setup these tools for all the skills and instructions to work properly, mostly to save on token costs.

- [Go](https://go.dev/): The CLI is written in Go.
- [rtk](https://github.com/rtk-ai/rtk): Don't pay for extraneous shell command tokens.
- [ctx7](https://context7.com/): Efficient and intelligent docs lookup for ai agents.
- [scrapling](https://github.com/scrapling/scrapling): Efficient web scraping for ai agents, don't waste tokens on <html> tags.

### Edit `USER_AGENTS.md`

An optional, gitignored file with your personal context. At `just install`, it is merged into the files deployed to each tool (and into `GLOBAL_AGENTS.local.md` in this repo for `@`-mentions). The tracked `GLOBAL_AGENTS.md` keeps a placeholder so the template stays shareable. Example:

```md
Name: <your-name>
Username: <your-username>
<stack>
Go
Python
</stack>
<goals>
Write highly testable code.
Master AI engineering.
</goals>
```

### Install Skills and Hooks

```sh
# Clone repo to your `~/.agents` folder
cd ~/ && git clone git@github.com:kalverra/.agents.git

# Make backups of your files
mv ~/.claude/CLAUDE.md ~/.claude/CLAUDE.backup.md
mv ~/.gemini/GEMINI.md ~/.gemini/GEMINI.backup.md
mv ~/.cursor/global-rules.mdc ~/.cursor/global-rules.backup.mdc # If you have one

# Use just (https://github.com/casey/just), a `make` alternative.
brew install just

# Run in dry-run mode to see what would be installed
just install --dry-run

# Install the real thing
just install
```

## Global Rules and Skills

### Session Prompt

Your agents should now start most sessions asking you for a SessionGoal. You can pre-provide it by starting your first prompt with `SessionGoal: Figure out an auth bug`.

This serves to keep you focused, and helps stop you from bloating AI context. When the goal is complete, you can move to a new session and get your context space back.

### [GLOBAL_AGENTS.md](GLOBAL_AGENTS.md)

Your global context that is loaded into every agent. It replaces `CLAUDE.md`, `GEMINI.md`, `.cursor/global-rules.mdc`, etc.

### [`find-docs`](skills/find-docs/SKILL.md)

Lookup current package/library docs using `ctx7`.

### [`pr-review`](skills/pr-review/SKILL.md)

Pull PR review comments from GitHub for your current repo and respond to them.

### [`rewrite-agent-instructions`](skills/rewrite-agent-instructions/SKILL.md)

Rewrite your `AGENTS.md` and other LLM focused files for token efficiency. Uses `xml` tags [for better LLM understanding](https://limitededitionjonathan.substack.com/p/the-definitive-guide-to-prompt-structure).

## Philosophy

**Use the agent for reasoning. Use code for everything else.**

Skills and rules are kept as compact as possible — minimal tokens, maximum signal. Rather than relying on MCP servers (which bloat context and hallucinate), we use deterministic Go scripts to fetch exactly what's needed and hand off only the relevant data to the agent.

## Contributing

I've only thoroughly tested things on a few tools I personally use. If you notice issues with any, please make an issue or PR!

### Evaluation

The repo includes an evaluation harness to test whether agents accurately follow the instructions and skills defined here, while keeping token usage low.

1. Feeds [test cases](./scripts/eval/cases/) to a subject LLM and captures output
2. Feeds the input and output to a judge LLM, which scores each response 1-5 against a rubric
3. Reports scores and token counts

Eval uses the Gemini API (requires `GEMINI_API_KEY`).

```bash
# Defaults: gemini-2.5-flash as subject, gemini-2.5-pro as judge
GEMINI_API_KEY="your-key" just eval

# Run 3 evaluations and collect averages
GEMINI_API_KEY="your-key" just eval 3
```

Full results are written to `scripts/eval/eval_results.md`.
