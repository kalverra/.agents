---
name: pr-review
description: Analyze diff + PR review comments to refine code changes.
---

<intent>
Analyze code diff and PR review comments. Draft refinement plan.
</intent>

<workflow>
1. Run `~/.agents/agents skills pr-review`.
2. Review code. Priorities: correctness, testability + quality of tests, simplicity, performance.
3. Classify comments: question, misunderstanding, suggestion.
4. Output response using required format.
5. Stop. Wait for user approval.
</workflow>

<rules>
<rule name="no-unapproved-changes">
Do not make code changes until plan is approved.
</rule>
<rule name="no-direct-reply">
Do not post replies or comments directly to PR.
</rule>
</rules>

<format>

```md
## Intent

Summary of change intent and execution.

## Suggested Reviewers

_(Omit if `<suggested_reviewers>` is missing or empty)_

- Name 1
- Name 2

## Review

- [+] Good things
- [-] Needed improvements

## Extra Review

_(Omit if none)_

- Areas needing stringent human review

## Comments

_(Omit if none)_

> username: filename:line
> comment body

### Suggested change

`​`​`replacement code`​`​`

**Diff context**: code hunk targets.

[classification] Plan to fix/reply. Use explicit `<suggestion>` blocks if you agree. (STOP CAVEMAN for reply)
```

</format>
