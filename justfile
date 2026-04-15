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

eval *args:
    go run . eval {{args}}
