Portable skills and rules. GLOBAL_AGENTS.md is machine-wide context.

<commands>
go mod tidy # tidy dependencies
go build -o agents . # build
go test ./... # test
golangci-lint run ./... --fix # lint
go run . -h # help menu
go run . [cmd] --ai-output # run commands with output specifically for LLM consumption

DO NOT run `go fmt`, `goimports`, or any other base go commands outside of the above
</commands>

<skills>
Located in `skills/`
- Define general, helpful skills for all agents to run
- Push as much work as possible to Go, only involve AI when absolutely necessary.
</skills>

<style>
- Use zerolog for all logging. Logging is not user output, it is only for debugging.
- When an official Go package doesn't exist to write a client to an API, use resty.
- Each Go command should utilize the `--ai-output` flag to format output for consumption by LLMs.
- AI output uses a consistent JSON envelope: `{"status":"ok","command":"<name>","data":<payload>}` (except `skills ticket fetch --ai-output`, which prints XML for the ticket payload).
- Use `output.Write(command, data, func() {...})` to cleanly handle both JSON and human output paths in one call without branching.
</style>

<docs>
Use ctx7 docs <path> <question>.
<agents>
- Claude: /websites/platform_claude_en
- Gemini API: /websites/ai_google_dev_gemini-api
- Gemini Go Package: /googleapis/go-genai
- Cursor: /websites/cursor
- Antigravity: /websites/antigravity
</agents>
- zerolog: /rs/zerolog
- todoist API: /websites/developer_todoist_api_v1
- resty: /go-resty/docs
- go-github: /google/go-github
</docs>
