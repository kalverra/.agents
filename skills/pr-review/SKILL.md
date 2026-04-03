---
name: pr-review
description: >-
  Fetches open PR comments for review and fixing.
---

# PR review analysis

## Prerequisite

- **Binary:** `~/.agents/cmd/pr-review/pr-review` (adjust prefix if `.agents` is cloned elsewhere).
- **Build** (one-time or after updates): `cd ~/.agents/cmd/pr-review && go build -o pr-review .`
- **Auth:** `GITHUB_TOKEN` or `gh auth login`.

## Workflow

1. **Run** by full path. Pass **`--dir`** with the git working tree (usually the user’s project, not `.agents`):

   ```bash
   ~/.agents/cmd/pr-review/pr-review --dir /path/to/the/project/repo
   ```

   Optional: `--include-resolved` to expand resolved review threads.

2. Read the structured markdown output.

3. **Each unresolved thread:** read the conversation, check the referenced code, decide if the comment is a valid fix/improvement or a misunderstanding.

4. **Plan:**
   - Valid comments: concrete change (file, function, what/why).
   - Disagreements: explain so the user can decide.
   - Number items. **Ask before implementing.** **Do not** reply on GitHub, commit, or submit reviews.

## Edge cases

- "No open PR found" — inform user and stop.
- Zero unresolved threads — say PR looks clean; summarize review state.
- Auth errors — relay the message (includes fix hints).
