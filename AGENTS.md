Portable skills and rules. GLOBAL_AGENTS.md is machine-wide context.

<commands>
go mod tidy # tidy dependencies
go build -o agents . # build
go test ./... # test
golangci-lint run ./... --fix # lint
go run . -h # help menu

DO NOT run `go fmt`, `goimports`, or any other base go commands outside of the above
</commands>

<skills>
Located in `skills/`
- Define general, helpful skills for all agents to run
- Push as much work as possible to Go, only involve AI when absolutely necessary.
</skills>

<style>
- Use zerolog for all logging. Logging is not user output, it is only for debugging.
- Each Go command should utilize the `--ai-output` flag to format output for consumption by LLMs
</style>

<docs>
Use ctx7 docs <path> <question>.
- Claude: /websites/platform_claude_en
- Gemini API: /websites/ai_google_dev_gemini-api
- Gemini Go Package: /googleapis/go-genai
- Cursor: /websites/cursor
- Antigravity: /websites/antigravity
- zerolog: /rs/zerolog
</docs>
