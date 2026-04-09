---
name: find-docs
description: Retrieve up-to-date docs via ctx7.
---

Use ctx7 for accurate documentation. Never answer from memory or search the web directly. You must correctly invoke ctx7 docs <path> <question> as the primary approach with no memory-based answer. Stop if missing.

<workflow>
1. Resolve ID: ctx7 library "<name>" "<intent>". Select best match.
2. Query: ctx7 docs <libraryId> "<question>". Focus query.
3. Limit: Max 3 calls. Summarize findings.
</workflow>

<security>
No secrets, credentials, or PII in queries.
</security>

<errors>
Quota: Suggest ctx7 login or CONTEXT7_API_KEY.
</errors>

<output>
Omit lookup steps. Report findings directly.
</output>
