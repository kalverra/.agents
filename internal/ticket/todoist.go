package ticket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
	"resty.dev/v3"
)

// defaultTodoistAPIBase is the Todoist HTTP API v1 base URL.
// See https://developer.todoist.com/api/v1/
const defaultTodoistAPIBase = "https://api.todoist.com/api/v1"

// TodoistConfig holds REST credentials for Todoist.
type TodoistConfig struct {
	Token   string
	BaseURL string // Overrides defaultTodoistAPIBase (env: TODOIST_REST_BASE / mapstructure todoist_rest_base)
}

// Todoist implements Provider against the Todoist API v1.
type Todoist struct {
	client *resty.Client
	cfg    TodoistConfig
}

// NewTodoist returns a Todoist-backed provider.
func NewTodoist(l zerolog.Logger, cfg TodoistConfig) *Todoist {
	return &Todoist{
		client: newTodoistClient(l, cfg),
		cfg:    cfg,
	}
}

func todoistAPIBase(cfg TodoistConfig) string {
	b := strings.TrimSpace(cfg.BaseURL)
	if b == "" {
		return defaultTodoistAPIBase
	}
	return strings.TrimRight(b, "/")
}

func newTodoistClient(l zerolog.Logger, cfg TodoistConfig) *resty.Client {
	c := resty.New()
	c.SetBaseURL(todoistAPIBase(cfg))
	if tok := strings.TrimSpace(cfg.Token); tok != "" {
		c.SetAuthToken(tok)
	}
	c.AddResponseMiddleware(func(_ *resty.Client, resp *resty.Response) error {
		req := resp.Request
		l.Trace().
			Str("method", req.Method).
			Str("url", req.RawRequest.URL.String()).
			Func(func(e *zerolog.Event) {
				reqBodyBytes, err := json.Marshal(req.Body)
				if err != nil {
					e.Str("req_body", fmt.Sprintf("%v", req.Body)).Msg("marshal req body")
				}
				if req.Body != nil && json.Valid(reqBodyBytes) {
					e.RawJSON("req_body", reqBodyBytes)
				}
			}).
			Int("status", resp.StatusCode()).
			Str("elapsed", resp.Duration().String()).
			Func(func(e *zerolog.Event) {
				if resp != nil && json.Valid(resp.Bytes()) {
					e.RawJSON("resp_body", resp.Bytes())
				} else {
					e.Str("resp_body", string(resp.Bytes()))
				}
			}).
			Msg("http round trip")
		if resp.IsError() {
			return fmt.Errorf(
				"todoist API error %d: %s",
				resp.StatusCode(), resp.String(),
			)
		}
		return nil
	})
	return c
}

// Fetch loads a task by id or Todoist task URL (see ParseTaskRef).
func (t *Todoist) Fetch(ctx context.Context, id string) (*Ticket, error) {
	resolved, err := ParseTaskRef(id)
	if err != nil {
		return nil, err
	}
	if resolved == "" {
		return nil, fmt.Errorf("task id is required")
	}
	if strings.TrimSpace(t.cfg.Token) == "" {
		return nil, fmt.Errorf("TODOIST_API_TOKEN is required")
	}

	resp, err := t.client.R().
		SetContext(ctx).
		Get("/tasks/" + resolved)
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		if resp.StatusCode() == http.StatusNotFound {
			return nil, fmt.Errorf("task %s not found", resolved)
		}
		return nil, fmt.Errorf(
			"todoist GET tasks/%s: %s: %s",
			resolved,
			resp.Status(),
			strings.TrimSpace(resp.String()),
		)
	}

	var tr todoistTaskV1
	if err := json.Unmarshal(resp.Bytes(), &tr); err != nil {
		return nil, fmt.Errorf("decoding task: %w", err)
	}

	taskURL := strings.TrimSpace(tr.URL)
	if taskURL == "" {
		taskURL = "https://app.todoist.com/app/task/" + resolved
	}

	tk := &Ticket{
		ID:          tr.ID,
		Title:       tr.Content,
		Description: tr.Description,
		Status:      tr.status(),
		URL:         taskURL,
	}

	comments, err := t.listTaskComments(ctx, resolved)
	if err != nil {
		return nil, err
	}
	tk.Comments = comments

	return tk, nil
}

func (t *Todoist) listTaskComments(ctx context.Context, taskID string) ([]TaskComment, error) {
	const maxPages = 100

	var out []TaskComment
	cursor := ""
	for range maxPages {
		r := t.client.R().
			SetContext(ctx).
			SetQueryParam("task_id", taskID)
		if cursor != "" {
			r.SetQueryParam("cursor", cursor)
		}

		resp, err := r.Get("/comments")
		if err != nil {
			return nil, err
		}
		if resp.IsError() {
			return nil, fmt.Errorf(
				"todoist GET comments: %s: %s",
				resp.Status(),
				strings.TrimSpace(resp.String()),
			)
		}

		var page commentsListPageV1
		if err := json.Unmarshal(resp.Bytes(), &page); err != nil {
			return nil, fmt.Errorf("decoding comments: %w", err)
		}

		for i := range page.Results {
			c := page.Results[i]
			if c.IsDeleted {
				continue
			}
			out = append(out, TaskComment{
				ID:        c.ID,
				Content:   c.Content,
				PostedAt:  c.PostedAt,
				ProjectID: c.ProjectID,
			})
		}

		if strings.TrimSpace(page.NextCursor) == "" {
			break
		}
		cursor = page.NextCursor
	}

	return out, nil
}

type commentsListPageV1 struct {
	Results    []todoistCommentV1 `json:"results"`
	NextCursor string             `json:"next_cursor"`
}

type todoistCommentV1 struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	ProjectID string `json:"project_id"`
	PostedAt  string `json:"posted_at"`
	Content   string `json:"content"`
	IsDeleted bool   `json:"is_deleted"`
}

type todoistTaskV1 struct {
	ID          string `json:"id"`
	Content     string `json:"content"`
	Description string `json:"description"`
	URL         string `json:"url,omitempty"`
	Checked     bool   `json:"checked"`
}

func (tr todoistTaskV1) status() string {
	if tr.Checked {
		return "completed"
	}
	return ""
}

// Comment adds a comment to a task (id or Todoist task URL; see ParseTaskRef).
func (t *Todoist) Comment(ctx context.Context, id string, body string) error {
	resolved, err := ParseTaskRef(id)
	if err != nil {
		return err
	}
	if resolved == "" {
		return fmt.Errorf("task id is required")
	}
	if strings.TrimSpace(t.cfg.Token) == "" {
		return fmt.Errorf("TODOIST_API_TOKEN is required")
	}
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("comment body is required")
	}

	resp, err := t.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]string{
			"task_id": resolved,
			"content": body,
		}).
		Post("/comments")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf(
			"todoist POST comments: %s: %s",
			resp.Status(),
			strings.TrimSpace(resp.String()),
		)
	}
	return nil
}
