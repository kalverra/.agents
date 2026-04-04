"""Shared logic for <hookable name="..."> regions in GLOBAL_AGENTS.md."""

from __future__ import annotations

import re

_SECTION_RE = re.compile(
    r"^<hookable name=\"\w+\">\s*\n"
    r"(.*?\n)*?"
    r"</hookable>\s*\n",
    re.MULTILINE,
)


def strip_hookable_sections(text: str) -> str:
    """Remove entire hookable regions (used when hooks are installed)."""
    return _SECTION_RE.sub("", text)


def strip_hookable_delimiter_lines(text: str) -> str:
    """Remove only <hookable name=\"…\"> and </hookable> lines; keep inner content."""
    out: list[str] = []
    for line in text.splitlines(keepends=True):
        s = line.strip()
        if re.fullmatch(r"<hookable name=\"\w+\">", s):
            continue
        if re.fullmatch(r"</hookable>", s):
            continue
        out.append(line)
    return "".join(out)
