---
name: pr-review
description: Load open GitHub PR review threads; propose fixes. No GitHub replies; no auto-commit.
---

<intent>
Turn review noise into actionable local patch plan.
</intent>

<workflow>
1. Run: ~/.agents/agents skills fetch-pr <repo_path>
2. Scan unresolved threads. Label each fix vs misunderstanding.
3. Plan fixes: file, function, delta.
4. Pause before edits. Ask user. Never post to GitHub or commit without explicit OK.
</workflow>

<errors>
- No PR: Say so. Stop.
- No threads: Say PR clean; one-line recap.
- Auth error: Report. Point to credential fix.
</errors>
