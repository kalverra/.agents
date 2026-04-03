---
name: pr-review
description: >-
  Fetch open PR review comments and propose fixes.
---

# PR review analysis

## Prerequisites

- **Binary:** `~/.agents/cmd/pr-review/pr-review` (adjust if `.agents` is elsewhere).
- **Build** (once): `cd ~/.agents/cmd/pr-review && go build -o pr-review .`
- **Auth:** `GITHUB_TOKEN` or `gh auth login`.

## Workflow

1. **Run** against the user's repo (not `.agents`):
   `~/.agents/cmd/pr-review/pr-review --dir /path/to/repo`
   Optional: `--include-resolved` to show resolved threads.
2. **For each unresolved thread:** read the conversation, check referenced code, classify as valid fix or misunderstanding.
3. **Present a numbered plan:**
   - Valid → concrete change (file, function, what/why).
   - Disagreement → explain so the user can decide.
   - **Ask before implementing.** Do **not** reply on GitHub, commit, or submit reviews.

## Edge cases

- No open PR → inform user and stop.
- Zero unresolved threads → say PR looks clean; summarize review state.
- Auth errors → relay the message (includes fix hints).
