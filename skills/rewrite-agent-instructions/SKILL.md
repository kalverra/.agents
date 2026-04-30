---
name: rewrite-agent-instructions
description: Audit LLM-facing files; rewrite for brevity and machine-parseable structure.
---

<intent>
Rewrite natural language in agent-facing docs. Preserve meaning. Do not alter code fences or CLI verbatim.
</intent>

<workflow>
1. Read target file.
2. List violations (verbosity, hedging, ambiguous steps).
3. Rewrite file. Keep intent.
4. Emit short before/after summary for user.
</workflow>

<rules>
<rule name="ignore-code">
Do not edit contents inside code fences or copy-paste CLI lines. Rewrite prose outside those only.
</rule>
<rule name="caveman">
User-facing summary after rewrite: terse caveman style. Drop articles and filler. No pleasantries. No hedging. Fragments OK. Keep technical terms exact. Leave code blocks literal.
Pattern: thing, action, reason. Next step.
</rule>
<rule name="machine_clarity">
Structure rewrites with XML section tags where helpful. Imperative steps. Short sentences. Strip bold, italics, decorative markdown from instruction text.
</rule>
<rule name="output_brevity">
In summary: cap lists, summarize deltas, skip boilerplate praise or disclaimers.
</rule>
<rule name="input_brevity">
In rewritten files: cut token count. Drop redundant examples and repeated rationale. Avoid wrapping entire file in one outer tag unless it aids parsing.
</rule>
</rules>
