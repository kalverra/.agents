---
name: start-session
description: >-
 Fetch Todoist task by id or app URL; set SessionGoal. Triggers: task link, bare id, /start-session.
---

<intent>
Bind one Todoist ticket as SessionGoal for session scope.
</intent>

<workflow>
1. Run `~/.agents/agents skills ticket fetch <task_id_or_url> --ai-output` (or `agents` from PATH if installed). The CLI accepts a Todoist bare id or `https://app.todoist.com/app/task/...` link, or a Jira key/URL when Jira env is set. With `--ai-output`, stdout is XML: root `<ticket_fetch status="ok">` with child elements `id`, `title`, optional `description`, `status`, `url`, and `<comments>` containing `<comment>` blocks (`id`, optional `posted_at`, `project_id`, `content`). Markdown in description/comments is plain text inside elements (XML-escaped).
2. Build SessionGoal from task title, description, due date when present.
3. State goal once. Continue work against that goal.
</workflow>

<errors>
- Missing token: User must set `TODOIST_API_TOKEN`; rerun install if token never configured.
- Task not found: Verify id; verify token can read that task.
</errors>
