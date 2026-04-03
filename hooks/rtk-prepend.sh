#!/usr/bin/env bash
# Shared hook: prepend `rtk` to shell commands before execution.
# Called by per-agent wrappers that set AGENT_TYPE={claude,gemini,cursor}.
# Requires: jq on PATH, rtk on PATH (silently passes through if either is missing).
set -euo pipefail

INPUT="$(cat)"
COMMAND="$(echo "$INPUT" | jq -r '.tool_input.command // empty')"

if [ -z "$COMMAND" ]; then
  exit 0
fi

if ! command -v rtk >/dev/null 2>&1; then
  exit 0
fi

# Skip if already prefixed, or if the command is a shell builtin / env setup
case "$COMMAND" in
  rtk\ *|cd\ *|export\ *|source\ *|.\ *|alias\ *|unset\ *|eval\ *)
    exit 0
    ;;
esac

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
  gemini)
    jq -n --arg cmd "$REWRITTEN" '{
      hookSpecificOutput: {
        hookEventName: "BeforeTool",
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
    echo "rtk-prepend.sh: unknown AGENT_TYPE='${AGENT_TYPE:-}'" >&2
    exit 0
    ;;
esac
