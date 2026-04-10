// Package github provides GitHub GraphQL API types and fetch helpers.
package github

import (
	"context"
	"fmt"
	"time"

	"github.com/shurcooL/githubv4"
)

// PR holds the fetched data for a GitHub pull request.
type PR struct {
	Number         int
	Title          string
	Body           string
	URL            string
	State          string
	CreatedAt      time.Time
	Author         string
	BaseRef        string
	HeadRef        string
	ReviewDecision string
	Reviews        []Review
	Threads        []ReviewThread
	Comments       []Comment
}

// Review represents a reviewer's overall assessment of a pull request.
type Review struct {
	Author    string
	State     string
	Body      string
	CreatedAt time.Time
}

// ReviewThread groups inline review comments on a specific file location.
type ReviewThread struct {
	IsResolved bool
	IsOutdated bool
	Path       string
	Line       int
	Comments   []Comment
}

// Comment is a single comment left on a PR or review thread.
type Comment struct {
	Author    string
	Body      string
	CreatedAt time.Time
}

// FetchPR queries GitHub for the most recent open PR on the given branch.
func FetchPR(ctx context.Context, client *githubv4.Client, owner, repo, branch string) (*PR, error) {
	var query struct {
		Repository struct {
			PullRequests struct {
				Nodes []struct {
					Number    int
					Title     string
					Body      string
					URL       string `graphql:"url"`
					State     string
					CreatedAt time.Time
					Author    struct {
						Login string
					}
					BaseRefName    string
					HeadRefName    string
					ReviewDecision string

					Reviews struct {
						Nodes []struct {
							Author struct {
								Login string
							}
							State     string
							Body      string
							CreatedAt time.Time
						}
					} `graphql:"reviews(first: 50)"`

					ReviewThreads struct {
						Nodes []struct {
							IsResolved bool
							IsOutdated bool
							Path       string
							Line       int
							Comments   struct {
								Nodes []struct {
									Author struct {
										Login string
									}
									Body      string
									CreatedAt time.Time
								}
							} `graphql:"comments(first: 50)"`
						}
					} `graphql:"reviewThreads(first: 100)"`

					Comments struct {
						Nodes []struct {
							Author struct {
								Login string
							}
							Body      string
							CreatedAt time.Time
						}
					} `graphql:"comments(first: 100)"`
				}
			} `graphql:"pullRequests(headRefName: $branch, states: OPEN, first: 5, orderBy: {field: CREATED_AT, direction: DESC})"`
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	variables := map[string]any{
		"owner":  githubv4.String(owner),
		"name":   githubv4.String(repo),
		"branch": githubv4.String(branch),
	}

	if err := client.Query(ctx, &query, variables); err != nil {
		return nil, fmt.Errorf("GitHub GraphQL query failed: %w", err)
	}

	nodes := query.Repository.PullRequests.Nodes
	if len(nodes) == 0 {
		return nil, nil
	}

	n := nodes[0]
	pr := &PR{
		Number:         n.Number,
		Title:          n.Title,
		Body:           n.Body,
		URL:            n.URL,
		State:          n.State,
		CreatedAt:      n.CreatedAt,
		Author:         n.Author.Login,
		BaseRef:        n.BaseRefName,
		HeadRef:        n.HeadRefName,
		ReviewDecision: n.ReviewDecision,
	}

	for _, r := range n.Reviews.Nodes {
		pr.Reviews = append(pr.Reviews, Review{
			Author:    r.Author.Login,
			State:     r.State,
			Body:      r.Body,
			CreatedAt: r.CreatedAt,
		})
	}

	for _, t := range n.ReviewThreads.Nodes {
		thread := ReviewThread{
			IsResolved: t.IsResolved,
			IsOutdated: t.IsOutdated,
			Path:       t.Path,
			Line:       t.Line,
		}
		for _, c := range t.Comments.Nodes {
			thread.Comments = append(thread.Comments, Comment{
				Author:    c.Author.Login,
				Body:      c.Body,
				CreatedAt: c.CreatedAt,
			})
		}
		pr.Threads = append(pr.Threads, thread)
	}

	for _, c := range n.Comments.Nodes {
		pr.Comments = append(pr.Comments, Comment{
			Author:    c.Author.Login,
			Body:      c.Body,
			CreatedAt: c.CreatedAt,
		})
	}

	return pr, nil
}
