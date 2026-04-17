---
name: rewrite-agent-instructions
description: Audit and rewrite LLM-facing files for brevity and machine clarity.
---

Audit and rewrite files for machine clarity.

<workflow>
1. Read target file.
2. Identify violations.
3. Rewrite; preserve intent.
4. Provide before/after summary.
</workflow>

<rules>
<rule name="ignore-code">
Leave code examples and CLI commands exactly as they are. Only rewrite natural language.
</rule>
<rule name="caveman">
Terse like caveman. Technical substance exact. Only fluff die.
Drop: articles, filler (just/really/basically), pleasantries, hedging.
Fragments OK. Short synonyms. Code unchanged.
Pattern: [thing] [action] [reason]. [next step].
/caveman skill
</rule>
<rule name="machine_clarity">
Use XML tags for sections. Use imperative voice and short sentences. Remove hedging, filler, and formatting (bold, italics).
</rule>
<rule name="output_brevity">
Specify minimal output: truncate lists, summarize, omit boilerplate.
</rule>
<rule name="input_brevity">
Minimize token count. Remove redundant examples, rationale, and global wrapping tags.
</rule>
</rules>
