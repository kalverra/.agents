#!/usr/bin/env bash
# Shared hook: prepend `rtk` to shell commands before execution.
# Called by per-agent wrappers that set AGENT_TYPE={claude,gemini,cursor}.
# Requires: rtk on PATH, jq on PATH (for fallback).
set -euo pipefail

export RTK_SUPPRESS_HOOK_WARNING=1

# Delegate to native 'rtk hook' for supported agents
if command -v rtk >/dev/null 2>&1; then
  case "${AGENT_TYPE:-}" in
    gemini|copilot)
      export RTK_HOOK_AUDIT=1
      exec rtk hook "$AGENT_TYPE"
      ;;
  esac
fi

# Fallback: Manual prepend using jq for unsupported or unknown agents
if ! command -v jq >/dev/null 2>&1 || ! command -v rtk >/dev/null 2>&1; then
  exit 0
fi

INPUT="$(cat)"
# Handle both .tool_input.command (Gemini/Claude) and .arguments.command (Cursor/Generic)
COMMAND="$(jq -r '.tool_input.command // .arguments.command // empty' <<< "$INPUT")"

if [ -z "$COMMAND" ] || [ "$COMMAND" = "null" ]; then
  exit 0
fi

# Skip if already prefixed, or if the command is a shell builtin / env setup
case "$COMMAND" in
  rtk\ *|cd\ *|export\ *|source\ *|.\ *|alias\ *|unset\ *|eval\ *)
    exit 0
    ;;
esac

# https://github.com/rtk-ai/rtk/issues/682
REWRITTEN="rtk $COMMAND"

case "${AGENT_TYPE:-}" in
  claude)
    jq -n --arg cmd "$REWRITTEN" '{
      hookSpecificOutput: {
        hookEventName: "PreToolUse",
        permissionDecision: "allow",
        updatedInput: { command: $cmd }
      }
    }'
    ;;
  cursor)
    jq -n --arg cmd "$REWRITTEN" '{
      permission: "allow",
      updated_input: { command: $cmd }
    }'
    ;;
  *)
    # Generic backup: Try both common update keys
    jq -n --arg cmd "$REWRITTEN" '{
      updatedInput: { command: $cmd },
      updated_input: { command: $cmd }
    }'
    ;;
esac
