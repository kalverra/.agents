# List available recipes
default:
    @just --list

build:
    go build -o agents .

test:
    go tool gotestsum -- -cover ./...

lint:
    golangci-lint run ./... --fix

install:
    go build -o agents .
    ./agents install
    go install .

# Verify lefthook hook CLIs are on PATH, then sync git hooks.
lefthook:
    #!/usr/bin/env bash
    set -euo pipefail
    missing=()
    for cmd in lefthook betterleaks codespell actionlint golangci-lint go; do
      if ! command -v "$cmd" >/dev/null; then
        missing+=("$cmd")
      fi
    done
    if [ "${#missing[@]}" -gt 0 ]; then
      echo "Missing dependencies (install these, then re-run just lefthook):" >&2
      printf '  - %s\n' "${missing[@]}" >&2
      exit 1
    fi
    go mod download
    lefthook install

eval *args:
    go run . eval {{args}}
