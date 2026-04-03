---
name: find-docs
description: >-
  Retrieve up-to-date docs via Context7 CLI (`ctx7`). Use when accuracy matters
  or training data may be stale.
---

# Documentation lookup (Context7)

Assume `ctx7` is on `PATH`. If missing, tell the user to install it and stop — never guess docs.

## Workflow

1. **Resolve** a library ID (skip if the user gave a Context7 ID like `/org/project`):
   `ctx7 library "<name>" "<intent>"`
   Pick the best match by name fit, snippet count, reputation (prefer High/Medium). Pin a version when needed: `ctx7 docs /vercel/next.js/v14.3.0-canary.87 "…"`. If ambiguous, ask.
2. **Query**: `ctx7 docs <libraryId> "<specific question>"`
   Be specific (e.g. `"JWT auth in Express"`, not `"auth"`).
3. **Cap:** ≤ **3** Context7 calls per question; use the best result and note if incomplete.

**Never** include secrets, credentials, or PII in queries.

## Errors

- **Quota exceeded** — suggest `ctx7 login` or setting `CONTEXT7_API_KEY`. If the user skips auth, warn answers may be stale.

## Output

Do **not** narrate lookup steps. Report findings directly as they relate to the user's query.
