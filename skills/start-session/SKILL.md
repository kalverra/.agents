---
name: start-session
description: Fetches a Todoist task by ID or app task URL and sets it as the SessionGoal. Use when the user provides a task link, id, or runs /start-session.
---

<workflow>
1. Run `~/.agents/agents skills ticket fetch <task_id_or_url> --ai-output` (or `agents` from PATH if installed). Uses [Todoist API v1](https://developer.todoist.com/api/v1/). The CLI accepts a bare id or a full `https://app.todoist.com/app/task/...` link. With `--ai-output`, stdout is indented JSON: top-level envelope `status`, `command`, `data`; `data.task` has id/title/description/status/url; `data.comments` is an array of `{id, content, posted_at, project_id}` (all pages merged, API order).
2. Set SessionGoal from task title plus description and due date when present.
3. State the goal clearly, then continue the session on that goal.
</workflow>

<errors>
- Missing token: Tell the user to set `TODOIST_API_TOKEN` and rerun install if needed.
- Task not found: Confirm the task id and that the token can access that task.
</errors>
