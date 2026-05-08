package ticket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJira_Fetch_rendersDescriptionAndCommentsAsMarkdown(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/rest/api/3/issue/PROJ-2":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":  "200",
				"key": "PROJ-2",
				"fields": map[string]any{
					"summary": "Spec",
					"description": map[string]any{
						"type": "doc", "version": 1,
						"content": []any{
							map[string]any{
								"type": "heading", "attrs": map[string]any{"level": float64(2)},
								"content": []any{map[string]any{"type": "text", "text": "Goals"}},
							},
							map[string]any{
								"type": "bulletList",
								"content": []any{
									map[string]any{
										"type": "listItem",
										"content": []any{
											map[string]any{
												"type": "paragraph",
												"content": []any{
													map[string]any{"type": "text", "text": "Ship "},
													map[string]any{
														"type":  "text",
														"text":  "v1",
														"marks": []any{map[string]any{"type": "strong"}},
													},
												},
											},
										},
									},
								},
							},
						},
					},
					"status": map[string]any{"name": "Open"},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/rest/api/3/issue/PROJ-2/comment":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"comments": []map[string]any{
					{
						"id": "c1", "created": "2026-05-01T12:00:00.000+0000",
						"body": map[string]any{
							"type": "doc", "version": 1,
							"content": []any{
								map[string]any{
									"type": "paragraph",
									"content": []any{
										map[string]any{"type": "text", "text": "See "},
										map[string]any{
											"type": "text",
											"text": "docs",
											"marks": []any{
												map[string]any{
													"type":  "link",
													"attrs": map[string]any{"href": "https://docs.example/doc"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
				"startAt": 0, "maxResults": 50, "total": 1,
			})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.String())
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	u := srv.URL
	host := u[len("http://"):]
	p := NewJira(zerolog.Nop(), JiraConfig{
		Email: "a@b.c", APIToken: "tok", Domain: host, HTTPScheme: "http",
	})
	got, err := p.Fetch(context.Background(), "PROJ-2")
	require.NoError(t, err)
	assert.Equal(t, "## Goals\n\n- Ship **v1**", got.Description)
	require.Len(t, got.Comments, 1)
	assert.Equal(t, "See [docs](https://docs.example/doc)", got.Comments[0].Content)
}

func TestJira_Fetch_success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/rest/api/3/issue/TASK-1":
			assert.NotEmpty(t, r.Header.Get("Authorization"))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":  "10000",
				"key": "TASK-1",
				"fields": map[string]any{
					"summary": "Fix thing",
					"description": map[string]any{
						"type": "doc", "version": 1,
						"content": []any{
							map[string]any{
								"type": "paragraph",
								"content": []any{
									map[string]any{"type": "text", "text": "Do it"},
								},
							},
						},
					},
					"status": map[string]any{"name": "In Progress"},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/rest/api/3/issue/TASK-1/comment":
			assert.Equal(t, "0", r.URL.Query().Get("startAt"))
			_ = json.NewEncoder(w).Encode(map[string]any{
				"comments": []map[string]any{
					{
						"id":      "101",
						"created": "2026-01-02T10:00:00.000+0000",
						"body": map[string]any{
							"type": "doc", "version": 1,
							"content": []any{
								map[string]any{
									"type": "paragraph",
									"content": []any{
										map[string]any{"type": "text", "text": "Note one"},
									},
								},
							},
						},
					},
				},
				"startAt":    0,
				"maxResults": 50,
				"total":      1,
			})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.String())
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	u := srv.URL
	host := u[len("http://"):]
	p := NewJira(zerolog.Nop(), JiraConfig{
		Email:      "a@b.c",
		APIToken:   "tok",
		Domain:     host,
		HTTPScheme: "http",
	})
	got, err := p.Fetch(context.Background(), "TASK-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "TASK-1", got.ID)
	assert.Equal(t, "Fix thing", got.Title)
	assert.Equal(t, "Do it", got.Description)
	assert.Equal(t, "In Progress", got.Status)
	assert.Equal(t, "http://"+host+"/browse/TASK-1", got.URL)
	require.Len(t, got.Comments, 1)
	assert.Equal(t, "101", got.Comments[0].ID)
	assert.Equal(t, "Note one", got.Comments[0].Content)
	assert.Equal(t, "2026-01-02T10:00:00.000+0000", got.Comments[0].PostedAt)
}

func TestJira_Comment_success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/rest/api/3/issue/TASK-9/comment" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.String())
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		doc, ok := body["body"].(map[string]any)
		if !ok {
			t.Errorf("body is not a map[string]any")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		assert.Equal(t, "doc", doc["type"])
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"999"}`))
	}))
	t.Cleanup(srv.Close)

	u := srv.URL
	host := u[len("http://"):]
	p := NewJira(zerolog.Nop(), JiraConfig{
		Email:      "a@b.c",
		APIToken:   "tok",
		Domain:     host,
		HTTPScheme: "http",
	})
	err := p.Comment(context.Background(), "TASK-9", "hello there")
	require.NoError(t, err)
}

func TestJira_Fetch_missingToken(t *testing.T) {
	t.Parallel()
	p := NewJira(zerolog.Nop(), JiraConfig{
		Email:    "a@b.c",
		Domain:   "x.atlassian.net",
		APIToken: "",
	})
	_, err := p.Fetch(context.Background(), "X-1")
	require.Error(t, err)
}
