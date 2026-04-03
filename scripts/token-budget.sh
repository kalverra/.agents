#!/usr/bin/env bash
# Token budget for LLM-facing context: GLOBAL_AGENTS.md + skills/**/*.md (all Markdown under skills/).
# Writes report to repo root .token-budget (cl100k_base via count-tokens.py).
# Requires: scripts/.venv with tiktoken (see AGENTS.md Venv).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"
PYTHON="${SCRIPT_DIR}/.venv/bin/python"
COUNT="${SCRIPT_DIR}/count-tokens.py"
OUT="${REPO_ROOT}/.token-budget"

if [[ ! -x "$PYTHON" ]]; then
  echo "token-budget: missing $PYTHON — run:" >&2
  echo "  python3 -m venv scripts/.venv && ./scripts/.venv/bin/pip install -r scripts/requirements.txt" >&2
  exit 1
fi

files=("${REPO_ROOT}/GLOBAL_AGENTS.md")
while IFS= read -r line; do
  [[ -n "$line" ]] && files+=("$line")
done < <(find "${REPO_ROOT}/skills" -type f -name '*.md' 2>/dev/null | sort)

{
  printf '# Token budget (cl100k_base; scripts/count-tokens.py)\n'
  printf '# Regenerate: ./scripts/token-budget.sh\n'
  printf '\n'
  printf '%-55s %s\n' "File" "Tokens"
  printf '%-55s %s\n' "----" "------"
  total=0
  for f in "${files[@]}"; do
    [[ -f "$f" ]] || continue
    rel="${f#"${REPO_ROOT}/"}"
    n=$("$PYTHON" "$COUNT" "$f")
    total=$((total + n))
    printf '%-55s %d\n' "$rel" "$n"
  done
  printf '%-55s %s\n' "----" "------"
  printf '%-55s %d\n' "TOTAL" "$total"
} | tee "$OUT"
