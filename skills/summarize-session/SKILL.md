---
name: summarize-session
description: Post session summary as Todoist task comment. Triggers: SessionGoal done, /summarize-session.
---

<intent>
Close loop: human-readable recap on same ticket as SessionGoal.
</intent>

<prereq>
If SessionGoal or Todoist task ref unknown, stop; ask for task id or app task URL before any CLI post.
</prereq>

<workflow>
1. Draft ticket summary: decisions, files touched, blockers, open questions. Cap five bullets.
2. Run `~/.agents/agents skills ticket comment <task_id_or_url> --body "<summary>" --ai-output` (escape body for shell).
3. Confirm post succeeded; else surface CLI error.
</workflow>

<errors>
- Missing token: User must set `TODOIST_API_TOKEN`.
- Empty body: Do not invoke CLI until summary text non-empty.
</errors>
