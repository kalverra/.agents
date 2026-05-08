---
name: find-docs
description: Retrieve up-to-date docs via ctx7.
---

<intent>
Ground answers in fetched docs; cap tool spend. If ctx7 CLI missing or broken, stop; tell user to install or configure. Do not invent docs.
</intent>

<restrictions>
* If provided a direct link, DO NOT use ctx7. Use `scrapling extract fetch --ai-targeted [URL] tmp.md && cat tmp.md && rm tmp.md` or native web fetch instead.
</restrictions>

<workflow>
1. Resolve ID: ctx7 library "<name>" "<intent>". Pick best match libraryId.
2. Query: ctx7 docs <libraryId> "<question>". Keep question narrow.
3. Max three ctx7 calls. Summarize findings for answer.
</workflow>

<security>
Never put secrets, credentials, or PII in ctx7 queries.
</security>

<errors>
- Quota or auth failure: Suggest ctx7 login or `CONTEXT7_API_KEY`.
</errors>

<output>
Skip narrating lookup steps in user reply; lead with findings.
</output>
