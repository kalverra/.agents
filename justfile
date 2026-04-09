python := "scripts/.venv/bin/python"
eval_script := "scripts/eval/eval.py"

# List available recipes
default:
    @just --list

# Create venv and install all dependencies
setup:
    pre-commit install
    python3 -m venv --clear scripts/.venv
    {{ python }} -m pip install -q -r scripts/requirements.txt
    @echo "✓ venv ready"

# Build agents-cli
build:
    cd cmd/agents-cli && go build -o ../../agents-cli .

# Install agent instructions to all targets
install *args: build
    ./agents-cli install {{ args }}

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
