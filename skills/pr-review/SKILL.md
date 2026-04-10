---
name: pr-review
description: Fetch open GitHub PR review comments and propose fixes.
---

Fetch and analyze open PR comments.

<setup>
- CLI: agents fetch-pr (or `go run ~/.agents fetch-pr`)
- Auth: GITHUB_TOKEN or gh auth login.
</setup>

<workflow>
1. Run: agents fetch-pr --dir <repo_path>
2. Analyze unresolved threads. Classify as fix or misunderstanding.
3. Plan fixes with file, function, and change.
4. Ask before implementation. Do not reply on GitHub or auto-commit.
</workflow>

<errors>
- No PR: Inform and stop.
- No threads: State PR is clean; summarize.
- Auth error: Report and suggest fix.
</errors>
