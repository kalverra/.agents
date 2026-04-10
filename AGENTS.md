Portable skills and rules. GLOBAL_AGENTS.md is machine-wide context.

<commands>
go build -o agents . # build
go test ./... # test
golangci-lint run ./... --fix # lint
go run . -h # help menu

DO NOT run `go fmt`, `goimports`, or any other base go commands outside of the above
</setup>

<skills>
Located in `skills/`
- Define general, helpful skills
- The `agents` CLI (`go run .`) provides subcommands: `install`, `discover`, `fetch-pr`, `eval`, `eval-compare`, `count-tokens`
- Push as much work as possible to Go, only involve AI when absolutely necessary.
</skills>

<docs>
Use ctx7 docs <path> <question>.
- Claude: /websites/platform_claude_en
- Gemini API: /websites/ai_google_dev_gemini-api
- Gemini Go Package: /googleapis/go-genai
- Cursor: /websites/cursor
- Antigravity: /websites/antigravity
</docs>
