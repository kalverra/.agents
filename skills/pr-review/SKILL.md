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

Summary of code change intent (why) and execution (how).

## Review

### Correctness

#### Good

- List
- Good
- Things

#### Improve

- list
- needed
- improvements

### Testability

#### Good

- List
- Good
- Things

#### Improve

- list
- needed
- improvements

### Simplicity

#### Good

- List
- Good
- Things

#### Improve

- list
- needed
- improvements

### Performance

#### Good

- List
- Good
- Things

#### Improve

- list
- needed
- improvements

## Extra Review

- List areas
- that might benefit
- from more stringent human review

## Comments (if any comment threads exist)

> username: filename:<lineNum>[-endLine]
> comment body

**Suggested change**:

```
suggested replacement code
```

**Diff context** (if present): code hunk the comment targets.

[classification] (STOP CAVEMAN for reply) Plan to fix or reply. Consider applying explicit `<suggestion>` blocks when present, assuming you agree.

```
</format>
```
