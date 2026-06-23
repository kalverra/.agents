package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
)

const (
	// SuggestionSourceBody marks suggestions parsed from ```suggestion markdown in comment bodies.
	SuggestionSourceBody = "body"
	// SuggestionSourceAutomated marks Copilot "Suggested changeset" diffs scraped from thread HTML.
	SuggestionSourceAutomated = "automated"

	embeddedDataMarker = `<script type="application/json" data-target="react-partial.embeddedData">`
)

// Suggestion is replacement code proposed in a review comment.
type Suggestion struct {
	Code      string
	Source    string
	Path      string
	StartLine int
	EndLine   int
}

type automatedDiffLine struct {
	Text  string `json:"text"`
	Type  string `json:"type"`
	Left  *int   `json:"left"`
	Right *int   `json:"right"`
}

type automatedDiffEntry struct {
	Path      string              `json:"path"`
	DiffLines []automatedDiffLine `json:"diffLines"`
}

type automatedSuggestionPayload struct {
	Props struct {
		Comment struct {
			AutomatedComment struct {
				Suggestion struct {
					DiffEntries []automatedDiffEntry `json:"diffEntries"`
				} `json:"suggestion"`
			} `json:"automatedComment"`
		} `json:"comment"`
	} `json:"props"`
}

// EnrichAutomatedSuggestions fetches Copilot "Suggested changeset" diffs from GitHub
// thread partial HTML. GitHub REST/GraphQL APIs omit these; they only exist in the UI payload.
func EnrichAutomatedSuggestions(ctx context.Context, token, owner, repo string, pr *PR) error {
	if pr == nil {
		return nil
	}

	client := newHTTPClient(token)
	for i := range pr.Threads {
		thread := &pr.Threads[i]
		if thread.ID == "" || !threadHasCopilotComment(thread) {
			continue
		}
		if threadHasBodySuggestions(thread) {
			continue
		}

		suggestions, err := fetchAutomatedSuggestionsForThread(ctx, client, owner, repo, pr.Number, thread.ID)
		if err != nil || len(suggestions) == 0 {
			continue
		}

		target := copilotCommentIndex(thread)
		thread.Comments[target].Suggestions = append(thread.Comments[target].Suggestions, suggestions...)
	}
	return nil
}

func threadHasCopilotComment(thread *ReviewThread) bool {
	for _, c := range thread.Comments {
		if isCopilotAuthor(c.Author) {
			return true
		}
	}
	return false
}

func threadHasBodySuggestions(thread *ReviewThread) bool {
	for _, c := range thread.Comments {
		for _, s := range c.Suggestions {
			if s.Source == SuggestionSourceBody {
				return true
			}
		}
	}
	return false
}

func copilotCommentIndex(thread *ReviewThread) int {
	for i, c := range thread.Comments {
		if isCopilotAuthor(c.Author) {
			return i
		}
	}
	return 0
}

func isCopilotAuthor(author string) bool {
	lower := strings.ToLower(author)
	return strings.Contains(lower, "copilot")
}

func decodeNodeDatabaseID(nodeID string) (int64, error) {
	_, payload, ok := strings.Cut(nodeID, "_")
	if !ok || payload == "" {
		return 0, fmt.Errorf("invalid node ID: %q", nodeID)
	}

	padded := payload + strings.Repeat("=", (4-len(payload)%4)%4)
	raw, err := base64.RawURLEncoding.DecodeString(padded)
	if err != nil {
		raw, err = base64.URLEncoding.DecodeString(padded)
		if err != nil {
			return 0, fmt.Errorf("decode node ID %q: %w", nodeID, err)
		}
	}

	if len(raw) < 4 {
		return 0, fmt.Errorf("node ID payload too short: %q", nodeID)
	}

	dbID := int64(raw[len(raw)-4])<<24 | int64(raw[len(raw)-3])<<16 | int64(raw[len(raw)-2])<<8 | int64(raw[len(raw)-1])
	return dbID, nil
}

func fetchAutomatedSuggestionsForThread(
	ctx context.Context,
	client *http.Client,
	owner, repo string,
	pullNumber int,
	threadNodeID string,
) ([]Suggestion, error) {
	threadDBID, err := decodeNodeDatabaseID(threadNodeID)
	if err != nil {
		return nil, err
	}

	threadURL := fmt.Sprintf(
		"https://github.com/%s/%s/pull/%d/threads/%d?rendering_on_files_tab=true",
		owner, repo, pullNumber, threadDBID,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, threadURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "text/html")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf(
			"thread partial request failed with status %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(body)),
		)
	}

	html, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, err
	}

	return parseAutomatedSuggestionsFromHTML(string(html))
}

func parseAutomatedSuggestionsFromHTML(html string) ([]Suggestion, error) {
	start := 0
	for {
		idx := strings.Index(html[start:], embeddedDataMarker)
		if idx == -1 {
			break
		}
		idx += start
		contentStart := idx + len(embeddedDataMarker)
		contentEnd := strings.Index(html[contentStart:], "</script>")
		if contentEnd == -1 {
			break
		}

		var payload automatedSuggestionPayload
		if err := json.Unmarshal([]byte(html[contentStart:contentStart+contentEnd]), &payload); err == nil {
			if suggestions := suggestionsFromAutomatedPayload(payload); len(suggestions) > 0 {
				return suggestions, nil
			}
		}

		start = contentStart + contentEnd
	}

	return nil, nil
}

func suggestionsFromAutomatedPayload(payload automatedSuggestionPayload) []Suggestion {
	diffEntries := payload.Props.Comment.AutomatedComment.Suggestion.DiffEntries
	if len(diffEntries) == 0 {
		return nil
	}

	suggestions := make([]Suggestion, 0, len(diffEntries))
	for _, entry := range diffEntries {
		code, startLine, endLine := buildSuggestionFromDiffLines(entry.DiffLines)
		if code == "" {
			continue
		}
		s := Suggestion{
			Path:   entry.Path,
			Code:   code,
			Source: SuggestionSourceAutomated,
		}
		if startLine != nil {
			s.StartLine = *startLine
		}
		if endLine != nil {
			s.EndLine = *endLine
		}
		suggestions = append(suggestions, s)
	}
	return suggestions
}

func buildSuggestionFromDiffLines(lines []automatedDiffLine) (string, *int, *int) {
	var builder strings.Builder
	var startLine, endLine *int

	for _, line := range lines {
		switch line.Type {
		case "HUNK":
			continue
		case "ADDITION", "CONTEXT":
			if builder.Len() > 0 {
				builder.WriteByte('\n')
			}
			builder.WriteString(line.Text)
			if line.Right != nil {
				if startLine == nil {
					startLine = line.Right
				}
				endLine = line.Right
			}
		}
	}

	if builder.Len() == 0 {
		return "", nil, nil
	}
	return builder.String(), startLine, endLine
}

func newHTTPClient(token string) *http.Client {
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	return &http.Client{
		Transport: &oauth2.Transport{Source: src},
	}
}
