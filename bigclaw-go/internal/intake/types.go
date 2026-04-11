package intake

type SourceIssue struct {
	Source      string            `json:"source"`
	SourceID    string            `json:"source_id"`
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	Labels      []string          `json:"labels,omitempty"`
	Priority    string            `json:"priority,omitempty"`
	State       string            `json:"state,omitempty"`
	Links       map[string]string `json:"links,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}
