// Package ticket abstracts issue trackers for session workflows.
package ticket

import "context"

// Ticket is a normalized task/issue for display and SessionGoal text.
type Ticket struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	Status      string        `json:"status,omitempty"`
	URL         string        `json:"url,omitempty"`
	Comments    []TaskComment `json:"comments,omitempty"`
}

// Provider fetches tasks and posts follow-up comments.
type Provider interface {
	Fetch(ctx context.Context, id string) (*Ticket, error)
	Comment(ctx context.Context, id string, body string) error
}
