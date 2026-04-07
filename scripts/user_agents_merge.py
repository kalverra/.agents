"""Merge USER_AGENTS.md into GLOBAL_AGENTS.md template text (install + eval)."""

from __future__ import annotations

import re
from pathlib import Path

USER_AGENTS_PLACEHOLDER = (
    "<!-- Instructions from USER_AGENTS.md are appended here during install -->"
)

_USER_BLOCK = re.compile(r"(<user>).*?(</user>)", re.DOTALL)


def merge_user_agents(body: str, user_src: Path) -> str:
    """Insert USER_AGENTS.md into the template.

    Prefer replacing the install-time HTML comment (same as eval). If there is no
    placeholder, replace the inner <user>...</user> region, else append.
    """
    if not user_src.is_file():
        return body

    user_content = user_src.read_text(encoding="utf-8").strip()
    if not user_content:
        return body

    if USER_AGENTS_PLACEHOLDER in body:
        return body.replace(USER_AGENTS_PLACEHOLDER, user_content)

    m = _USER_BLOCK.search(body)
    if m:
        return _USER_BLOCK.sub(
            lambda match: match.group(1) + "\n" + user_content + "\n" + match.group(2),
            body,
        )

    return body.rstrip() + "\n\n" + user_content + "\n"
