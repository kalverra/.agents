---
name: start-session
description: Fetch Todoist task by id or app URL; set SessionGoal. Triggers: task link, bare id, /start-session.
---

<intent>
Bind one Todoist ticket as SessionGoal for session scope.
</intent>

<workflow>
1. Run `~/.agents/agents skills ticket fetch <task_id_or_url> --ai-output` (or `agents` from PATH if installed). Uses [Todoist API v1](https://developer.todoist.com/api/v1/). The CLI accepts a bare id or a full `https://app.todoist.com/app/task/...` link. With `--ai-output`, stdout is indented JSON: top-level envelope `status`, `command`, `data`; `data.task` has id/title/description/status/url; `data.comments` is an array of `{id, content, posted_at, project_id}` (all pages merged, API order).
2. Build SessionGoal from task title, description, due date when present.
3. State goal once. Continue work against that goal.
</workflow>

<errors>
- Missing token: User must set `TODOIST_API_TOKEN`; rerun install if token never configured.
- Task not found: Verify id; verify token can read that task.
</errors>
