package ticket

// FetchPayload is the normalized ticket-fetch result. With --ai-output, ticket fetch emits XML from this shape (see FetchPayloadToAIXML); human mode prints the same fields as plain text.
type FetchPayload struct {
	Task     TaskInfo      `json:"task"`
	Comments []TaskComment `json:"comments"`
}

// TaskInfo is the normalized task fields for JSON output.
type TaskInfo struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	URL         string `json:"url,omitempty"`
}

// TaskComment is a normalized Todoist comment.
type TaskComment struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	PostedAt  string `json:"posted_at,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
}

// ToFetchPayload builds the normalized payload used for AI XML output and internal testing.
func (tk *Ticket) ToFetchPayload() FetchPayload {
	comments := tk.Comments
	if comments == nil {
		comments = []TaskComment{}
	}
	return FetchPayload{
		Task: TaskInfo{
			ID:          tk.ID,
			Title:       tk.Title,
			Description: tk.Description,
			Status:      tk.Status,
			URL:         tk.URL,
		},
		Comments: comments,
	}
}
