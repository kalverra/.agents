---
name: find-docs
description: >-
  Retrieves up-to-date docs, API details, config, and examples via Context7 CLI.
  Use for libraries, frameworks, languages, SDKs, APIs, CLIs, and platforms
  whenever accuracy matters or training data may be stale.
---

# Documentation lookup (Context7)

## Prerequisite

Assume `ctx7` is on `PATH`. If missing, prompt the user to install it and stop—do not guess documentation.

## Workflow

Resolve a library ID, then query. **Run `ctx7 library` first** unless the user already gave a Context7 ID (`/org/project`).

```
ctx7 library "<name>" "<intent query>"
ctx7 docs <libraryId> "<question>"
```

**Cap:** at most **3** Context7 invocations per question; then use the best result and say if incomplete.

**Never** put secrets in queries (keys, passwords, credentials, PII, proprietary code).

## Resolve (`ctx7 library`)

Pick best match by name fit, snippet count, reputation (prefer High/Medium). Versioned: `ctx7 docs /vercel/next.js/v14.3.0-canary.87 "…"`. If ambiguous, ask.

## Query (`ctx7 docs`)

Be specific (e.g. `"JWT auth in Express"`, not `"auth"`); vague one-word queries yield weak results.

## Quota errors

Quota exceeded? Suggest `ctx7 login` or `CONTEXT7_API_KEY`; if the user skips auth, warn answers may be outdated.
