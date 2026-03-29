package reporting

import "strings"

type CollaborationComment struct {
	CommentID string   `json:"comment_id"`
	Author    string   `json:"author"`
	Body      string   `json:"body"`
	CreatedAt string   `json:"created_at,omitempty"`
	Mentions  []string `json:"mentions,omitempty"`
	Anchor    string   `json:"anchor,omitempty"`
	Status    string   `json:"status,omitempty"`
}

type DecisionNote struct {
	DecisionID        string   `json:"decision_id"`
	Author            string   `json:"author"`
	Outcome           string   `json:"outcome"`
	Summary           string   `json:"summary"`
	RecordedAt        string   `json:"recorded_at,omitempty"`
	Mentions          []string `json:"mentions,omitempty"`
	RelatedCommentIDs []string `json:"related_comment_ids,omitempty"`
	FollowUp          string   `json:"follow_up,omitempty"`
}

type CollaborationThread struct {
	Surface   string                 `json:"surface"`
	TargetID  string                 `json:"target_id"`
	Comments  []CollaborationComment `json:"comments,omitempty"`
	Decisions []DecisionNote         `json:"decisions,omitempty"`
}

func (t CollaborationThread) ParticipantCount() int {
	participants := make(map[string]struct{})
	for _, comment := range t.Comments {
		if strings.TrimSpace(comment.Author) != "" {
			participants[comment.Author] = struct{}{}
		}
	}
	for _, decision := range t.Decisions {
		if strings.TrimSpace(decision.Author) != "" {
			participants[decision.Author] = struct{}{}
		}
	}
	return len(participants)
}

func (t CollaborationThread) MentionCount() int {
	total := 0
	for _, comment := range t.Comments {
		total += len(comment.Mentions)
	}
	for _, decision := range t.Decisions {
		total += len(decision.Mentions)
	}
	return total
}

func (t CollaborationThread) OpenCommentCount() int {
	total := 0
	for _, comment := range t.Comments {
		if comment.Status != "resolved" {
			total++
		}
	}
	return total
}

func (t CollaborationThread) Recommendation() string {
	if len(t.Decisions) > 0 {
		return "share-latest-decision"
	}
	if t.OpenCommentCount() > 0 {
		return "resolve-open-comments"
	}
	if len(t.Comments) > 0 {
		return "monitor-collaboration"
	}
	return "no-collaboration-recorded"
}

func BuildCollaborationThread(surface string, targetID string, comments []CollaborationComment, decisions []DecisionNote) CollaborationThread {
	return CollaborationThread{
		Surface:   surface,
		TargetID:  targetID,
		Comments:  append([]CollaborationComment(nil), comments...),
		Decisions: append([]DecisionNote(nil), decisions...),
	}
}

func RenderCollaborationLines(thread *CollaborationThread) []string {
	if thread == nil {
		return nil
	}
	lines := []string{
		"## Collaboration",
		"",
		"- Surface: " + thread.Surface,
		"- Target: " + thread.TargetID,
		"- Participants: " + itoaReporting(thread.ParticipantCount()),
		"- Comments: " + itoaReporting(len(thread.Comments)),
		"- Open Comments: " + itoaReporting(thread.OpenCommentCount()),
		"- Mentions: " + itoaReporting(thread.MentionCount()),
		"- Decision Notes: " + itoaReporting(len(thread.Decisions)),
		"- Recommendation: " + thread.Recommendation(),
		"",
		"## Comments",
		"",
	}
	if len(thread.Comments) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, comment := range thread.Comments {
			mentions := "none"
			if len(comment.Mentions) > 0 {
				mentions = strings.Join(comment.Mentions, ", ")
			}
			anchor := comment.Anchor
			if strings.TrimSpace(anchor) == "" {
				anchor = "none"
			}
			status := comment.Status
			if strings.TrimSpace(status) == "" {
				status = "open"
			}
			lines = append(lines, "- "+comment.CommentID+": author="+comment.Author+" status="+status+" anchor="+anchor+" mentions="+mentions+" body="+comment.Body)
		}
	}
	lines = append(lines, "", "## Decision Notes", "")
	if len(thread.Decisions) == 0 {
		lines = append(lines, "- None")
		return lines
	}
	for _, decision := range thread.Decisions {
		mentions := "none"
		if len(decision.Mentions) > 0 {
			mentions = strings.Join(decision.Mentions, ", ")
		}
		related := "none"
		if len(decision.RelatedCommentIDs) > 0 {
			related = strings.Join(decision.RelatedCommentIDs, ", ")
		}
		followUp := decision.FollowUp
		if strings.TrimSpace(followUp) == "" {
			followUp = "none"
		}
		lines = append(lines, "- "+decision.DecisionID+": author="+decision.Author+" outcome="+decision.Outcome+" mentions="+mentions+" related="+related+" follow_up="+followUp+" summary="+decision.Summary)
	}
	return lines
}

type SharedViewFilter struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type SharedViewContext struct {
	Filters       []SharedViewFilter   `json:"filters,omitempty"`
	ResultCount   *int                 `json:"result_count,omitempty"`
	Loading       bool                 `json:"loading,omitempty"`
	Errors        []string             `json:"errors,omitempty"`
	PartialData   []string             `json:"partial_data,omitempty"`
	EmptyMessage  string               `json:"empty_message,omitempty"`
	LastUpdated   string               `json:"last_updated,omitempty"`
	Collaboration *CollaborationThread `json:"collaboration,omitempty"`
}

func (v SharedViewContext) State() string {
	if v.Loading {
		return "loading"
	}
	if len(v.Errors) > 0 && (v.ResultCount == nil || *v.ResultCount == 0) {
		return "error"
	}
	if v.ResultCount != nil && *v.ResultCount == 0 && len(v.PartialData) == 0 {
		return "empty"
	}
	if len(v.Errors) > 0 || len(v.PartialData) > 0 {
		return "partial-data"
	}
	return "ready"
}

func (v SharedViewContext) Summary() string {
	switch v.State() {
	case "loading":
		return "Loading data for the current filters."
	case "error":
		return "Unable to load data for the current filters."
	case "empty":
		if strings.TrimSpace(v.EmptyMessage) != "" {
			return v.EmptyMessage
		}
		return "No records match the current filters."
	case "partial-data":
		return "Showing partial data while one or more sources are unavailable."
	default:
		return "Data is current for the selected filters."
	}
}

func RenderSharedViewContext(view *SharedViewContext) []string {
	if view == nil {
		return nil
	}
	lines := []string{
		"## View State",
		"",
		"- State: " + view.State(),
		"- Summary: " + view.Summary(),
	}
	if view.ResultCount != nil {
		lines = append(lines, "- Result Count: "+itoaReporting(*view.ResultCount))
	}
	if strings.TrimSpace(view.LastUpdated) != "" {
		lines = append(lines, "- Last Updated: "+view.LastUpdated)
	}
	lines = append(lines, "", "## Filters", "")
	if len(view.Filters) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, item := range view.Filters {
			lines = append(lines, "- "+item.Label+": "+item.Value)
		}
	}
	if len(view.Errors) > 0 {
		lines = append(lines, "", "## Errors", "")
		for _, message := range view.Errors {
			lines = append(lines, "- "+message)
		}
	}
	if len(view.PartialData) > 0 {
		lines = append(lines, "", "## Partial Data", "")
		for _, message := range view.PartialData {
			lines = append(lines, "- "+message)
		}
	}
	lines = append(lines, RenderCollaborationLines(view.Collaboration)...)
	lines = append(lines, "")
	return lines
}
