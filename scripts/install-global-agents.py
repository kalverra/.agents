#!/usr/bin/env python3
"""Install GLOBAL_AGENTS.md (and optional hooks) for Claude Code, Gemini CLI,
Antigravity, and Cursor.

Usage:
  ./scripts/install-global-agents.py discover [--verbose]
  ./scripts/install-global-agents.py install [--copy] [--dry-run] [--verbose] [--targets a,b,c] [--with-hooks]
"""

from __future__ import annotations

import argparse
import json
import os
import re
import shutil
import sys
from pathlib import Path


# ---------------------------------------------------------------------------
# Paths
# ---------------------------------------------------------------------------

def repo_root() -> Path:
    return Path(__file__).resolve().parent.parent


def script_path() -> Path:
    return Path(__file__).resolve()


def global_src() -> Path:
    return repo_root() / "GLOBAL_AGENTS.md"


def hooks_src_dir() -> Path:
    return repo_root() / "hooks"


HOOKS_DEPLOY_DIR = Path.home() / ".agents-hooks"


def gemini_config_dir() -> Path:
    env = os.environ.get("GEMINI_CONFIG_DIR", "").strip()
    return Path(env).expanduser() if env else Path.home() / ".gemini"


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def vlog(verbose: bool, msg: str) -> None:
    if verbose:
        print(f"[verbose] {msg}", file=sys.stderr)


def which(name: str) -> str | None:
    return shutil.which(name)


def target_wanted(name: str, targets: list[str] | None) -> bool:
    if not targets:
        return True
    return name in targets


def forcing_targets(targets: list[str] | None) -> bool:
    return bool(targets)


# ---------------------------------------------------------------------------
# Detection
# ---------------------------------------------------------------------------

def detect_claude(verbose: bool) -> bool:
    p = which("claude")
    if p:
        vlog(verbose, f"claude: found in PATH ({p})")
        return True
    d = Path.home() / ".claude"
    if d.is_dir():
        vlog(verbose, "claude: ~/.claude exists")
        return True
    return False


def detect_gemini(verbose: bool) -> bool:
    for name in ("gemini", "gemini-cli"):
        p = which(name)
        if p:
            vlog(verbose, f"{name}: found in PATH ({p})")
            return True
    gd = gemini_config_dir()
    if gd.is_dir():
        vlog(verbose, f"gemini: config dir exists ({gd})")
        return True
    return False


def detect_antigravity(verbose: bool) -> bool:
    """Google Antigravity shares ~/.gemini/ with Gemini CLI; detection is for discover/install UX."""
    p = which("antigravity")
    if p:
        vlog(verbose, f"antigravity: found in PATH ({p})")
        return True
    if (Path.home() / ".antigravity-server").is_dir():
        vlog(verbose, "antigravity: ~/.antigravity-server exists")
        return True
    sysname = os.uname().sysname
    if sysname == "Darwin":
        ag = Path.home() / "Library/Application Support/Antigravity"
        if ag.is_dir():
            vlog(verbose, f"antigravity: {ag} exists")
            return True
    if sysname == "Linux":
        ag = Path.home() / ".config/Antigravity"
        if ag.is_dir():
            vlog(verbose, f"antigravity: {ag} exists")
            return True
    return False


def detect_cursor(verbose: bool) -> bool:
    if (Path.home() / ".cursor").is_dir():
        vlog(verbose, "cursor: ~/.cursor exists")
        return True
    sysname = os.uname().sysname
    if sysname == "Darwin":
        p = Path.home() / "Library/Application Support/Cursor"
        if p.is_dir():
            vlog(verbose, "cursor: ~/Library/Application Support/Cursor exists")
            return True
    if sysname == "Linux":
        p = Path.home() / ".config/Cursor"
        if p.is_dir():
            vlog(verbose, "cursor: ~/.config/Cursor exists")
            return True
    return False


# ---------------------------------------------------------------------------
# Hookable-section stripping
# ---------------------------------------------------------------------------

_HOOKABLE_RE = re.compile(
    r"^<!-- hookable: \w+ -->\s*\n"
    r"(.*?\n)*?"
    r"<!-- /hookable: \w+ -->\s*\n",
    re.MULTILINE,
)


def strip_hookable_sections(text: str) -> str:
    return _HOOKABLE_RE.sub("", text)


# ---------------------------------------------------------------------------
# Context deployment
# ---------------------------------------------------------------------------

def do_link_or_copy(
    src: Path,
    dest: Path,
    *,
    use_copy: bool,
    dry_run: bool,
) -> None:
    if dry_run:
        verb = "cp" if use_copy else "symlink"
        print(f"[dry-run] {verb} {src} -> {dest}")
        return
    dest.parent.mkdir(parents=True, exist_ok=True)
    dest.unlink(missing_ok=True)
    if use_copy:
        shutil.copy2(src, dest)
        print(f"Copied: {dest}")
    else:
        dest.symlink_to(src.resolve())
        print(f"Symlinked: {dest} -> {src.resolve()}")


def write_filtered_copy(
    src: Path,
    dest: Path,
    *,
    strip_hooks: bool,
    dry_run: bool,
) -> None:
    """Write a filtered copy of the source file (always a copy, never symlink)."""
    if dry_run:
        print(f"[dry-run] write {dest} (filtered copy, strip_hooks={strip_hooks})")
        return
    body = src.read_text(encoding="utf-8")
    if strip_hooks:
        body = strip_hookable_sections(body)
    dest.parent.mkdir(parents=True, exist_ok=True)
    dest.write_text(body, encoding="utf-8")
    print(f"Wrote: {dest}")


def write_cursor_mdc(
    global_path: Path,
    *,
    strip_hooks: bool,
    dry_run: bool,
) -> None:
    dest = Path.home() / ".cursor/rules/global-agents.mdc"
    if dry_run:
        print(f"[dry-run] write {dest} (frontmatter + GLOBAL_AGENTS.md, strip_hooks={strip_hooks})")
        return
    dest.parent.mkdir(parents=True, exist_ok=True)
    body = global_path.read_text(encoding="utf-8")
    if strip_hooks:
        body = strip_hookable_sections(body)
    root = repo_root()
    header = f"""---
description: Machine-wide context from {root}/GLOBAL_AGENTS.md
alwaysApply: true
---

"""
    dest.write_text(header + body, encoding="utf-8")
    print(f"Wrote: {dest}")


# ---------------------------------------------------------------------------
# Hook deployment
# ---------------------------------------------------------------------------

def deploy_hook_scripts(*, dry_run: bool) -> Path:
    """Copy hook scripts to ~/.agents-hooks/ and return the deploy dir."""
    src_dir = hooks_src_dir()
    dest_dir = HOOKS_DEPLOY_DIR
    scripts = [
        "rtk-prepend.sh",
        "claude-rtk.sh",
        "gemini-rtk.sh",
        "cursor-rtk.sh",
    ]
    if dry_run:
        for s in scripts:
            print(f"[dry-run] cp {src_dir / s} -> {dest_dir / s}")
        return dest_dir

    dest_dir.mkdir(parents=True, exist_ok=True)
    for s in scripts:
        src = src_dir / s
        dst = dest_dir / s
        shutil.copy2(src, dst)
        dst.chmod(0o755)
    print(f"Deployed hook scripts to {dest_dir}")
    return dest_dir


def load_snippet(name: str, hooks_dir: Path) -> dict:
    """Load a JSON snippet from hooks/ and replace <HOOKS_DIR> placeholder."""
    raw = (hooks_src_dir() / name).read_text(encoding="utf-8")
    raw = raw.replace("<HOOKS_DIR>", str(hooks_dir))
    return json.loads(raw)


def merge_hooks_into_settings(
    settings_path: Path,
    snippet: dict,
    hook_key: str,
    *,
    match_field: str,
    dry_run: bool,
) -> None:
    """Merge a hooks snippet into an existing settings.json, idempotently.

    match_field is the field used to identify our entry for dedup
    (e.g. "command" for Claude/Cursor, "name" for Gemini).
    """
    if dry_run:
        print(f"[dry-run] merge hooks into {settings_path} (key: {hook_key})")
        return

    settings_path.parent.mkdir(parents=True, exist_ok=True)
    if settings_path.is_file():
        existing = json.loads(settings_path.read_text(encoding="utf-8"))
    else:
        existing = {}

    hooks_section = existing.setdefault("hooks", {})
    existing_entries: list = hooks_section.setdefault(hook_key, [])

    new_entries = snippet["hooks"][hook_key]

    for new_entry in new_entries:
        if "hooks" in new_entry:
            new_id = new_entry["hooks"][0].get(match_field, "")
        else:
            new_id = new_entry.get(match_field, "")

        replaced = False
        for i, old in enumerate(existing_entries):
            if "hooks" in old:
                old_id = old["hooks"][0].get(match_field, "")
            else:
                old_id = old.get(match_field, "")
            if old_id and old_id == new_id:
                existing_entries[i] = new_entry
                replaced = True
                break
        if not replaced:
            existing_entries.append(new_entry)

    settings_path.write_text(
        json.dumps(existing, indent=2, ensure_ascii=False) + "\n",
        encoding="utf-8",
    )
    print(f"Merged hooks into {settings_path}")


def install_hooks_claude(hooks_dir: Path, *, dry_run: bool) -> None:
    snippet = load_snippet("claude-settings-snippet.json", hooks_dir)
    merge_hooks_into_settings(
        Path.home() / ".claude/settings.json",
        snippet,
        "PreToolUse",
        match_field="command",
        dry_run=dry_run,
    )


def install_hooks_gemini(hooks_dir: Path, *, dry_run: bool) -> None:
    snippet = load_snippet("gemini-settings-snippet.json", hooks_dir)
    merge_hooks_into_settings(
        gemini_config_dir() / "settings.json",
        snippet,
        "BeforeTool",
        match_field="command",
        dry_run=dry_run,
    )


def install_hooks_cursor(hooks_dir: Path, *, dry_run: bool) -> None:
    snippet = load_snippet("cursor-hooks-snippet.json", hooks_dir)
    hooks_json = Path.home() / ".cursor/hooks.json"
    merge_hooks_into_settings(
        hooks_json,
        snippet,
        "preToolUse",
        match_field="command",
        dry_run=dry_run,
    )


# ---------------------------------------------------------------------------
# Commands
# ---------------------------------------------------------------------------

def cmd_discover(verbose: bool) -> int:
    print("Discovery (signals only — tools need not be running):\n")
    any_yes = False
    if detect_claude(verbose):
        print(f"  claude-code   yes   context -> {Path.home() / '.claude/CLAUDE.md'}")
        print(f"                      hooks   -> {Path.home() / '.claude/settings.json'} (PreToolUse)")
        any_yes = True
    else:
        print("  claude-code   no    (no claude in PATH, no ~/.claude)")

    gemini_ok = detect_gemini(verbose)
    antigravity_ok = detect_antigravity(verbose)

    if gemini_ok:
        print(f"  gemini-cli    yes   context -> {gemini_config_dir() / 'GEMINI.md'}")
        print(f"                      hooks   -> {gemini_config_dir() / 'settings.json'} (BeforeTool)")
        any_yes = True
    else:
        print("  gemini-cli    no    (no gemini/gemini-cli in PATH, no config dir)")

    if antigravity_ok:
        print(f"  antigravity   yes   context -> {gemini_config_dir() / 'GEMINI.md'} (shared with gemini-cli)")
        print(f"                      hooks   -> {gemini_config_dir() / 'settings.json'} (BeforeTool, shared)")
        any_yes = True
    else:
        print(
            "  antigravity   no    (no antigravity in PATH, no Antigravity app dirs, no ~/.antigravity-server)"
        )

    if gemini_ok and antigravity_ok:
        print()
        print(
            "  Note: gemini-cli and antigravity both use ~/.gemini/ (GEMINI.md, settings.json). "
            "One install updates both."
        )

    if detect_cursor(verbose):
        print(
            f"  cursor        yes   context -> {Path.home() / '.cursor/rules/global-agents.mdc'} "
            "(best-effort; see note)"
        )
        print(f"                      hooks   -> {Path.home() / '.cursor/hooks.json'} (preToolUse)")
        any_yes = True
    else:
        print(
            "  cursor        no    (no ~/.cursor or OS-specific Cursor app support dir)"
        )

    print()
    if any_yes:
        print(f"Run: {script_path()} install [--copy] [--dry-run] [--with-hooks]")
    else:
        print(
            "No known agent paths detected. Install tools first, or use "
            "install --targets to force paths."
        )
    print()
    print(
        "Cursor note: Global User Rules may be cloud/UI-only. If .mdc is not picked up globally,"
    )
    print(
        "  paste GLOBAL_AGENTS.md into Cursor Settings → Rules → User Rules, or sync from this file."
    )
    return 0


def cmd_install(
    *,
    use_copy: bool,
    dry_run: bool,
    verbose: bool,
    targets: list[str] | None,
    with_hooks: bool,
) -> int:
    src = global_src()
    if not src.is_file():
        print(f"Missing source file: {src}", file=sys.stderr)
        return 1

    hooks_dir = HOOKS_DEPLOY_DIR
    if with_hooks:
        hooks_dir = deploy_hook_scripts(dry_run=dry_run)

    installed = False

    if target_wanted("claude", targets):
        if detect_claude(verbose) or forcing_targets(targets):
            if not detect_claude(verbose) and forcing_targets(targets):
                warn = Path.home() / ".claude/CLAUDE.md"
                print(f"Warning: claude-code not detected; writing {warn} anyway (--targets).")
            if with_hooks:
                write_filtered_copy(src, Path.home() / ".claude/CLAUDE.md", strip_hooks=True, dry_run=dry_run)
                install_hooks_claude(hooks_dir, dry_run=dry_run)
            else:
                do_link_or_copy(src, Path.home() / ".claude/CLAUDE.md", use_copy=use_copy, dry_run=dry_run)
            installed = True

    gemini_paths_requested = target_wanted("gemini", targets) or target_wanted(
        "antigravity", targets
    )
    gemini_stack_detected = detect_gemini(verbose) or detect_antigravity(verbose)
    if gemini_paths_requested:
        if gemini_stack_detected or forcing_targets(targets):
            if not gemini_stack_detected and forcing_targets(targets):
                dest = gemini_config_dir() / "GEMINI.md"
                print(
                    f"Warning: gemini-cli / antigravity not detected; writing {dest} anyway (--targets)."
                )
            if with_hooks:
                write_filtered_copy(src, gemini_config_dir() / "GEMINI.md", strip_hooks=True, dry_run=dry_run)
                install_hooks_gemini(hooks_dir, dry_run=dry_run)
            else:
                do_link_or_copy(src, gemini_config_dir() / "GEMINI.md", use_copy=use_copy, dry_run=dry_run)
            installed = True

    if target_wanted("cursor", targets):
        if detect_cursor(verbose) or forcing_targets(targets):
            if not detect_cursor(verbose) and forcing_targets(targets):
                print("Warning: cursor dirs not detected; writing ~/.cursor/rules/global-agents.mdc anyway (--targets).")
            write_cursor_mdc(src, strip_hooks=with_hooks, dry_run=dry_run)
            if with_hooks:
                install_hooks_cursor(hooks_dir, dry_run=dry_run)
            installed = True

    if not installed:
        sp = script_path()
        print(f"Nothing installed. Try: {sp} discover")
        print(f"Force paths with: {sp} install --targets claude,gemini,antigravity,cursor")
        return 1
    return 0


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------

def parse_targets(s: str | None) -> list[str] | None:
    if not s or not s.strip():
        return None
    return [t.strip().lower() for t in s.split(",") if t.strip()]


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Install GLOBAL_AGENTS.md (and optional hooks) for Claude Code, Gemini CLI, Antigravity, and Cursor."
    )
    sub = parser.add_subparsers(dest="command", required=True)

    p_disc = sub.add_parser("discover", help="List detected agents, install paths, and hook targets")
    p_disc.add_argument("--verbose", "-v", action="store_true")

    p_ins = sub.add_parser("install", help="Deploy GLOBAL_AGENTS.md and optional hooks to tool paths")
    p_ins.add_argument(
        "--copy",
        action="store_true",
        help="Copy instead of symlink (Claude/Gemini only; Cursor .mdc is always generated)",
    )
    p_ins.add_argument("--dry-run", action="store_true", help="Print actions only")
    p_ins.add_argument("--verbose", "-v", action="store_true")
    p_ins.add_argument(
        "--targets",
        metavar="LIST",
        help="Comma-separated: claude, gemini, antigravity, cursor (default: only detected). "
        "antigravity uses the same ~/.gemini/ paths as gemini. Forces writes even if not detected.",
    )
    p_ins.add_argument(
        "--with-hooks",
        action="store_true",
        help="Deploy rtk-prepend hooks and merge into agent settings. "
        "Strips hookable sections from deployed markdown.",
    )

    args = parser.parse_args()
    if args.command == "discover":
        return cmd_discover(args.verbose)
    if args.command == "install":
        return cmd_install(
            use_copy=args.copy,
            dry_run=args.dry_run,
            verbose=args.verbose,
            targets=parse_targets(args.targets),
            with_hooks=args.with_hooks,
        )
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
