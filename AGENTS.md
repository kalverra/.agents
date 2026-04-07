Portable skills and rules. GLOBAL_AGENTS.md is machine-wide context.

<setup>
- `just install` to install
- `just eval` to test changes
</setup>

<skills>
Located in `skills/`
- Define general, helpful skills
- Helped by Go programs in `cmd/`. Push as much work as possible to Go scripts, only involve AI when absolutely necessary.
</skills>

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
