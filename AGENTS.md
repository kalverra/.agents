Portable skills and rules. GLOBAL_AGENTS.md is machine-wide context.

<setup>
Deploy with scripts/install-global-agents.py.
- python3 scripts/install-global-agents.py install
- --copy for Claude/Gemini.
- --targets for specific agents (claude, gemini, antigravity, cursor).
- --no-hooks or --no-skills to skip.
- run `just eval` to test changes.
</setup>

<skills>
Located in skills/. Requires SKILL.md with name and description YAML.
- Gemini/Antigravity: ~/.agents/skills/.
- Claude/Cursor: ~/.claude/skills/ or ~/.cursor/skills/.
- Use portable frontmatter.
</skills>

<binaries>
Go modules in cmd/ back skills.
- Build: cd cmd/<name> && go build -o <name> .
- Path: ~/.agents/cmd/<name>/<name>.
- pr-review: GitHub GraphQL PR fetcher.
</binaries>

<venv>
- python3 -m venv scripts/.venv && ./scripts/.venv/bin/pip install -r scripts/requirements.txt
</venv>

<docs>
Use ctx7 docs <path> <question>.
- Claude: /websites/platform_claude_en
- Gemini CLI: /google-gemini/gemini-cli
- Cursor: /websites/cursor
- Antigravity: /websites/antigravity
</docs>
