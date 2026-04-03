---
name: rewrite-agent-instructions
description: >-
  Audit and rewrite LLM-facing files (AGENTS.md, SKILL.md, rules, prompts) for brevity and machine clarity.
---

# Rewrite Agent Instructions

Audit and rewrite LLM-facing files (AGENTS.md, SKILL.md, rules, prompts) for brevity and machine clarity.

## Workflow

1. Read the target file fully.
2. Identify violations of the rules below.
3. Rewrite; preserve all semantic intent.
4. Show a before/after summary (not full diffs) so the user can verify nothing was lost.

## Rules (in priority order)

1. **Machine clarity** — Use imperative voice, short sentences, bold key terms, and lists over prose. Remove hedging ("you might want to…"), filler, and redundant headers. If a section heading just restates the first sentence, cut the heading.
2. **Output brevity** — When the file instructs an agent to produce output, ensure it says to keep output minimal: truncate long listings, summarize instead of echoing, omit boilerplate unless asked.
3. **Input brevity** — Minimize token count of the file itself. Cut examples that don't add new information; collapse multi-sentence explanations into one; remove rationale the model doesn't need to follow the instruction.
