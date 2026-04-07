python := "scripts/.venv/bin/python"
eval_script := "scripts/eval/eval.py"

# List available recipes
default:
    @just --list

# Create venv and install all dependencies
setup:
    python3 -m venv --clear scripts/.venv
    {{ python }} -m pip install -q -r scripts/requirements.txt
    @echo "✓ venv ready"

# Install agent instructions to all targets
install *args:
    {{ python }} scripts/install-global-agents.py install {{ args }}

# Run all eval cases with multiple iterations (default: 1) and write report
eval iter="1" *args:
    {{ python }} {{ eval_script }} --iterations {{ iter }} --report scripts/eval/eval_results.md {{ args }}
    @echo "→ scripts/eval/eval_results.md"

# Run eval cases for a specific tag (e.g. just eval-tag tools)
eval-tag tag *args:
    {{ python }} {{ eval_script }} --filter {{ tag }} --report scripts/eval/eval_results.md {{ args }}
    @echo "→ scripts/eval/eval_results.md"

# Run eval with verbose output (shows full responses)
eval-verbose *args:
    {{ python }} {{ eval_script }} --verbose --report scripts/eval/eval_results.md {{ args }}
    @echo "→ scripts/eval/eval_results.md"

# Run eval using Gemini API (requires GEMINI_API_KEY)
eval-gemini iter="1" *args:
    {{ python }} {{ eval_script }} --subject-type gemini --judge-type gemini --iterations {{ iter }} --report scripts/eval/eval_results.md {{ args }}
    @echo "→ scripts/eval/eval_results.md"
