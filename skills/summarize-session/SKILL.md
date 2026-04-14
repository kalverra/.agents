---
name: summarize-session
description: Summarizes completed session work and posts it as a comment on the linked Todoist task. Use when the SessionGoal is done or the user invokes /summarize-session.
---

If no `SessionGoal` or Todoist task reference is known, ask for the task id or paste the app task URL before posting.

<workflow>
1. Produce a short summary for the ticket: key decisions, files touched, blockers, open questions. Maximum five bullet points.
2. Run `~/.agents/agents skills ticket comment <task_id_or_url> --body "<summary>" --ai-output` (escape or pass body safely for the shell).
3. Confirm the comment was posted or report the error.
</workflow>

<errors>
- Missing token: Tell the user to set `TODOIST_API_TOKEN`.
- Empty body: Do not call the CLI until the summary text is non-empty.
</errors>
