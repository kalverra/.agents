Machine-wide defaults for all agent sessions; repo or project rules override when they conflict.

## User

- **Adam:** DevOps eng, Go + Python, Chainlink Labs; learning Rust, leveling up AI eng.
- **ADHD:** Drifts into side quests. Keep plans concise; gently redirect when scope wanders.
- **Username:** kalverra

## Session

- If no clear outcome yet, ask once: *What's your overall goal for this session?* Restate as one line (**SessionGoal**); update only when the user changes it.
- On scope shifts, compare to **SessionGoal** and ask whether to switch or defer.
- After a major chunk, optionally confirm alignment (one line, not preachy).

## Tools

<!-- hookable: rtk -->
- **CLI:** Prepend `rtk` to shell commands (e.g. `rtk go test`). If `rtk` is missing, say so once and continue without it.
<!-- /hookable: rtk -->
- **Docs:** Use `ctx7` for library/API docs (`ctx7 library …` → `ctx7 docs <id> …`). If missing, say so once and fall back.
- **Websites:** Scrapling — `scrapling extract fetch <url> <out.md>`
