# `~/.agents`

A global home for all of your AI agents! Define global rules once, and sync them to Claude Code, Gemini CLI, Antigravity, Cursor, and any other agents you use.

The skills and rules here are highly optimized for minimal token usage, taking up as little context space as possible while delivering high performance.

## Quick Start

Clone this repo into your home directory, then run the install script to detect what agents you use and install the global rules, hooks, and skills to them. **WARNING**: This will overwrite any existing `CLAUDE.md`, `GEMINI.md`, `.cursor/global-rules.mdc`, etc. files.

```sh
# Use [`just`](https://github.com/casey/just) (a `make` alternative) to install.
just install

# Run in dry-run mode to see what would be installed
just install --dry-run

# Or use the raw python script
python scripts/install-global-agents.py install
```

### Install Dependencies

You'll need to setup these tools for all the skills and instructions to work properly, mostly to save on token costs.

- [rtk](https://github.com/rtk-ai/rtk): Don't pay for extraneous shell command tokens.
- [ctx7](https://context7.com/): Efficient and intelligent docs lookup for ai agents.
- [scrapling](https://github.com/scrapling/scrapling): Efficient web scraping for ai agents, don't waste tokens on <html> tags.

## Global Rules and Skills

### [GLOBAL_AGENTS.md](GLOBAL_AGENTS.md)

Your global context that is loaded into every agent. It replaces `CLAUDE.md`, `GEMINI.md`, `.cursor/global-rules.mdc`, etc.

### USER_AGENTS.md

An optional, hidden, user-specific context that is loaded into `GLOBAL_AGENTS.md` at install time. Example:
```md
Name: Adam
Username: kalverra
<stack>
Go
Python
</stack>
<goals>
Write highly testable code.
Master AI engineering.
</goals>
```

### [`find-docs`](skills/find-docs/SKILL.md)

Lookup current package/library docs using `ctx7`.

### [`pr-review`](skills/pr-review/SKILL.md)

Pull PR review comments from GitHub for your current repo and respond to them.

### [`rewrite-agent-instructions`](skills/rewrite-agent-instructions/SKILL.md)

Rewrite your `AGENTS.md` and other LLM focused files for token efficiency. Uses `xml` tags [for better LLM understanding](https://limitededitionjonathan.substack.com/p/the-definitive-guide-to-prompt-structure).

## Contributing

I've only thoroughly tested things on a few tools I personally use. If you notice issues with any, please make an issue or PR!

### Evaluation

To ensure models actually follow the rules defined in `GLOBAL_AGENTS.md` and follow skill definitions, this repo includes an automated evaluation harness leveraging a subject model (e.g., `llama3.1:8b`) and a judge model (Prometheus). I run these on my local machine using Ollama for the sake of cost, but you can route them to frontier models if you wish.

```bash
# Run tests 1 time and output to terminal
just eval

# Run tests 3 times and write a markdown report (scripts/eval/eval_results.md)
just eval-multi

# Run specific tests (e.g., tests matching the 'tools' tag)
just eval-multi 3 --filter tools
```
