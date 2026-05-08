package ticket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
	"resty.dev/v3"

	"github.com/kalverra/agents/internal/mactls"
)

const jiraRESTPath = "/rest/api/3"

// JiraConfig holds Atlassian Cloud Jira REST API credentials (Basic auth: email + API token).
type JiraConfig struct {
	Email    string
	APIToken string
	// Domain is the site hostname only, e.g. your-org.atlassian.net (no scheme, no path).
	Domain string
	// HTTPScheme is "https" by default; set to "http" for tests with httptest.
	HTTPScheme string
}

// Jira implements Provider against Jira Cloud REST API v3.
type Jira struct {
	client *resty.Client
	cfg    JiraConfig
}

// NewJira returns a Jira-backed provider.
func NewJira(l zerolog.Logger, cfg JiraConfig) *Jira {
	return &Jira{
		client: newJiraClient(l, cfg),
		cfg:    cfg,
	}
}

func jiraAPIBase(cfg JiraConfig) string {
	scheme := strings.TrimSpace(cfg.HTTPScheme)
	if scheme == "" {
		scheme = "https"
	}
	host := strings.Trim(strings.TrimSpace(cfg.Domain), "/")
	return fmt.Sprintf("%s://%s%s", scheme, host, jiraRESTPath)
}

func newJiraClient(l zerolog.Logger, cfg JiraConfig) *resty.Client {
	c := resty.New()
	if rt := mactls.RoundTripper(); rt != nil {
		c.SetTransport(rt)
	}
	c.SetBaseURL(jiraAPIBase(cfg))
	email := strings.TrimSpace(cfg.Email)
	tok := strings.TrimSpace(cfg.APIToken)
	if email != "" && tok != "" {
		c.SetBasicAuth(email, tok)
	}
	c.AddResponseMiddleware(func(_ *resty.Client, resp *resty.Response) error {
		req := resp.Request
		l.Trace().
			Str("method", req.Method).
			Str("url", req.RawRequest.URL.String()).
			Int("status", resp.StatusCode()).
			Str("elapsed", resp.Duration().String()).
			Func(func(e *zerolog.Event) {
				if resp != nil && json.Valid(resp.Bytes()) {
					e.RawJSON("resp_body", resp.Bytes())
				} else {
					e.Str("resp_body", string(resp.Bytes()))
				}
			}).
			Msg("jira http round trip")
		if resp.IsError() {
			return fmt.Errorf(
				"jira API error %d: %s",
				resp.StatusCode(), strings.TrimSpace(resp.String()),
			)
		}
		return nil
	})
	return c
}

func (j *Jira) validate() error {
	if strings.TrimSpace(j.cfg.Email) == "" ||
		strings.TrimSpace(j.cfg.APIToken) == "" ||
		strings.TrimSpace(j.cfg.Domain) == "" {
		return fmt.Errorf("JIRA_EMAIL, JIRA_API_TOKEN, and JIRA_DOMAIN are required")
	}
	return nil
}

func (j *Jira) browseURL(issueKey string) string {
	scheme := strings.TrimSpace(j.cfg.HTTPScheme)
	if scheme == "" {
		scheme = "https"
	}
	host := strings.Trim(strings.TrimSpace(j.cfg.Domain), "/")
	return fmt.Sprintf("%s://%s/browse/%s", scheme, host, issueKey)
}

// Fetch loads an issue by key or Jira issue URL (see ParseJiraRef).
func (j *Jira) Fetch(ctx context.Context, ref string) (*Ticket, error) {
	if err := j.validate(); err != nil {
		return nil, err
	}
	key, err := ParseJiraRef(ref)
	if err != nil {
		return nil, err
	}

	resp, err := j.client.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"fields": "summary,description,status",
		}).
		Get("/issue/" + key)
	if err != nil {
		if resp != nil && resp.StatusCode() == http.StatusNotFound {
			return nil, fmt.Errorf("issue %s not found", key)
		}
		return nil, err
	}

	var issue jiraIssueEnvelope
	if err := json.Unmarshal(resp.Bytes(), &issue); err != nil {
		return nil, fmt.Errorf("decoding issue: %w", err)
	}

	desc := ""
	if len(issue.Fields.Description) > 0 {
		var adf any
		if err := json.Unmarshal(issue.Fields.Description, &adf); err == nil {
			desc = adfToMarkdown(adf)
		}
	}

	status := ""
	if issue.Fields.Status.Name != "" {
		status = issue.Fields.Status.Name
	}

	tk := &Ticket{
		ID:          issue.Key,
		Title:       issue.Fields.Summary,
		Description: desc,
		Status:      status,
		URL:         j.browseURL(issue.Key),
	}

	comments, err := j.listIssueComments(ctx, key)
	if err != nil {
		return nil, err
	}
	tk.Comments = comments

	return tk, nil
}

func (j *Jira) listIssueComments(ctx context.Context, issueKey string) ([]TaskComment, error) {
	const pageSize = 50
	var out []TaskComment
	startAt := 0

	for {
		resp, err := j.client.R().
			SetContext(ctx).
			SetQueryParams(map[string]string{
				"startAt":    fmt.Sprintf("%d", startAt),
				"maxResults": fmt.Sprintf("%d", pageSize),
			}).
			Get("/issue/" + issueKey + "/comment")
		if err != nil {
			return nil, err
		}

		var page jiraCommentsPage
		if err := json.Unmarshal(resp.Bytes(), &page); err != nil {
			return nil, fmt.Errorf("decoding comments: %w", err)
		}

		for i := range page.Comments {
			c := page.Comments[i]
			var adf any
			content := ""
			if len(c.Body) > 0 {
				if err := json.Unmarshal(c.Body, &adf); err == nil {
					content = adfToMarkdown(adf)
				}
			}
			out = append(out, TaskComment{
				ID:       c.ID,
				Content:  content,
				PostedAt: c.Created,
			})
		}

		startAt += len(page.Comments)
		if startAt >= page.Total || len(page.Comments) == 0 {
			break
		}
	}

	return out, nil
}

// Comment adds a comment to an issue (key or Jira issue URL; see ParseJiraRef).
func (j *Jira) Comment(ctx context.Context, ref string, body string) error {
	if err := j.validate(); err != nil {
		return err
	}
	key, err := ParseJiraRef(ref)
	if err != nil {
		return err
	}
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("comment body is required")
	}

	payload := map[string]any{
		"body": plainBodyToADF(body),
	}

	_, err = j.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(payload).
		Post("/issue/" + key + "/comment")
	if err != nil {
		return err
	}
	return nil
}

type jiraIssueEnvelope struct {
	Key    string `json:"key"`
	Fields struct {
		Summary     string          `json:"summary"`
		Description json.RawMessage `json:"description"`
		Status      struct {
			Name string `json:"name"`
		} `json:"status"`
	} `json:"fields"`
}

type jiraCommentsPage struct {
	Comments   []jiraComment `json:"comments"`
	StartAt    int           `json:"startAt"`
	MaxResults int           `json:"maxResults"`
	Total      int           `json:"total"`
}

type jiraComment struct {
	ID      string          `json:"id"`
	Created string          `json:"created"`
	Body    json.RawMessage `json:"body"`
}

func plainBodyToADF(text string) map[string]any {
	lines := strings.Split(text, "\n")
	paras := make([]any, 0, len(lines))
	for _, line := range lines {
		paras = append(paras, map[string]any{
			"type": "paragraph",
			"content": []any{
				map[string]any{"type": "text", "text": line},
			},
		})
	}
	if len(paras) == 0 {
		paras = append(paras, map[string]any{
			"type":    "paragraph",
			"content": []any{map[string]any{"type": "text", "text": ""}},
		})
	}
	return map[string]any{
		"type":    "doc",
		"version": 1,
		"content": paras,
	}
}
