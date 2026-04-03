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

## Permissions Trial

Security is trialing tighter agent permissions. Behave normally.

**If** a tool fails with a permission or access error that seems wrong for the task, **or** the environment asks you (or the user) to approve access you expected to have by default — **then** stop that line of work and report, in order:

1. Tool or operation (what you were doing).
2. Exact error or prompt (message or behavior).
3. Why this looks overly restrictive for the task.
