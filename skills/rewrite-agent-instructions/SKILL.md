---
name: rewrite-agent-instructions
description: >-
  Audit and rewrite LLM specific files like AGENTS.md, SKILL.md, etc. for brevity, clarity, and efficiency.
---

# Rewrite Agent Instructions

## Priorities

1. **Clarity**: Rewrite things in a way that is best understood and formatted for understanding by LLMs.
2. **Concise Output**: Output tokens cost 3x-5x more than input tokens. Include instructions to truncate all unnecessary or elaborate outputs unless otherwise specified.
3. **Concise Input**: Input tokens bloat context and add costs. Make sure files are as minimal as possible without sacrificing priorities 1 or 2.
