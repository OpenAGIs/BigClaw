package repo

import (
	"fmt"
	"sort"
	"time"
)

type CollaborationComment struct {
	CommentID string   `json:"comment_id"`
	Author    string   `json:"author"`
	Body      string   `json:"body"`
	CreatedAt string   `json:"created_at"`
	Mentions  []string `json:"mentions,omitempty"`
	Anchor    string   `json:"anchor,omitempty"`
	Status    string   `json:"status,omitempty"`
}

type DecisionNote struct {
	DecisionID        string   `json:"decision_id"`
	Author            string   `json:"author"`
	Outcome           string   `json:"outcome"`
	Summary           string   `json:"summary"`
	RecordedAt        string   `json:"recorded_at"`
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

func collaborationNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func BuildCollaborationThread(surface, targetID string, comments []CollaborationComment, decisions []DecisionNote) CollaborationThread {
	return CollaborationThread{
		Surface:   surface,
		TargetID:  targetID,
		Comments:  append([]CollaborationComment(nil), comments...),
		Decisions: append([]DecisionNote(nil), decisions...),
	}
}

func MergeCollaborationThreads(targetID string, nativeThread, repoThread *CollaborationThread) *CollaborationThread {
	if nativeThread == nil && repoThread == nil {
		return nil
	}

	merged := CollaborationThread{
		Surface:  "merged",
		TargetID: targetID,
	}
	for _, thread := range []*CollaborationThread{nativeThread, repoThread} {
		if thread == nil {
			continue
		}
		merged.Comments = append(merged.Comments, thread.Comments...)
		merged.Decisions = append(merged.Decisions, thread.Decisions...)
	}
	sort.SliceStable(merged.Comments, func(i, j int) bool {
		return merged.Comments[i].CreatedAt < merged.Comments[j].CreatedAt
	})
	sort.SliceStable(merged.Decisions, func(i, j int) bool {
		return merged.Decisions[i].RecordedAt < merged.Decisions[j].RecordedAt
	})
	return &merged
}

func BuildCollaborationThreadFromAudits(audits []map[string]any, surface, targetID string) *CollaborationThread {
	comments := make([]CollaborationComment, 0)
	decisions := make([]DecisionNote, 0)
	for _, audit := range audits {
		details, _ := audit["details"].(map[string]any)
		if details == nil {
			details = map[string]any{}
		}
		if stringValue(details["surface"]) != surface {
			continue
		}
		switch stringValue(audit["action"]) {
		case "collaboration.comment":
			comments = append(comments, CollaborationComment{
				CommentID: stringValue(details["comment_id"]),
				Author:    stringValue(audit["actor"]),
				Body:      stringValue(details["body"]),
				CreatedAt: valueOrString(audit["timestamp"], collaborationNow()),
				Mentions:  stringSlice(details["mentions"]),
				Anchor:    stringValue(details["anchor"]),
				Status:    valueOrString(details["status"], "open"),
			})
		case "collaboration.decision":
			decisions = append(decisions, DecisionNote{
				DecisionID:        stringValue(details["decision_id"]),
				Author:            stringValue(audit["actor"]),
				Outcome:           stringValue(audit["outcome"]),
				Summary:           stringValue(details["summary"]),
				RecordedAt:        valueOrString(audit["timestamp"], collaborationNow()),
				Mentions:          stringSlice(details["mentions"]),
				RelatedCommentIDs: stringSlice(details["related_comment_ids"]),
				FollowUp:          stringValue(details["follow_up"]),
			})
		}
	}
	if len(comments) == 0 && len(decisions) == 0 {
		return nil
	}
	thread := BuildCollaborationThread(surface, targetID, comments, decisions)
	return &thread
}

func (t CollaborationThread) ParticipantCount() int {
	participants := map[string]struct{}{}
	for _, comment := range t.Comments {
		if comment.Author != "" {
			participants[comment.Author] = struct{}{}
		}
	}
	for _, decision := range t.Decisions {
		if decision.Author != "" {
			participants[decision.Author] = struct{}{}
		}
	}
	return len(participants)
}

func (t CollaborationThread) MentionCount() int {
	count := 0
	for _, comment := range t.Comments {
		count += len(comment.Mentions)
	}
	for _, decision := range t.Decisions {
		count += len(decision.Mentions)
	}
	return count
}

func (t CollaborationThread) OpenCommentCount() int {
	count := 0
	for _, comment := range t.Comments {
		if comment.Status != "resolved" {
			count++
		}
	}
	return count
}

func (t CollaborationThread) Recommendation() string {
	switch {
	case len(t.Decisions) > 0:
		return "share-latest-decision"
	case t.OpenCommentCount() > 0:
		return "resolve-open-comments"
	case len(t.Comments) > 0:
		return "monitor-collaboration"
	default:
		return "no-collaboration-recorded"
	}
}

func RenderCollaborationLines(thread *CollaborationThread) []string {
	if thread == nil {
		return nil
	}
	lines := []string{
		"## Collaboration",
		"",
		fmt.Sprintf("- Surface: %s", thread.Surface),
		fmt.Sprintf("- Target: %s", thread.TargetID),
		fmt.Sprintf("- Participants: %d", thread.ParticipantCount()),
		fmt.Sprintf("- Comments: %d", len(thread.Comments)),
		fmt.Sprintf("- Open Comments: %d", thread.OpenCommentCount()),
		fmt.Sprintf("- Mentions: %d", thread.MentionCount()),
		fmt.Sprintf("- Decision Notes: %d", len(thread.Decisions)),
		fmt.Sprintf("- Recommendation: %s", thread.Recommendation()),
		"",
		"## Comments",
		"",
	}
	if len(thread.Comments) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, comment := range thread.Comments {
			lines = append(lines, fmt.Sprintf(
				"- %s: author=%s status=%s anchor=%s mentions=%s body=%s",
				comment.CommentID,
				comment.Author,
				valueOrString(comment.Status, "open"),
				valueOrString(comment.Anchor, "none"),
				csvOrNone(comment.Mentions),
				comment.Body,
			))
		}
	}
	lines = append(lines, "", "## Decision Notes", "")
	if len(thread.Decisions) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, decision := range thread.Decisions {
			lines = append(lines, fmt.Sprintf(
				"- %s: outcome=%s author=%s mentions=%s related=%s summary=%s follow_up=%s",
				decision.DecisionID,
				decision.Outcome,
				decision.Author,
				csvOrNone(decision.Mentions),
				csvOrNone(decision.RelatedCommentIDs),
				decision.Summary,
				valueOrString(decision.FollowUp, "none"),
			))
		}
	}
	return append(lines, "")
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}

func valueOrString(value any, fallback string) string {
	text := stringValue(value)
	if text == "" {
		return fallback
	}
	return text
}

func stringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, stringValue(item))
		}
		return out
	default:
		return nil
	}
}

func csvOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return joinCSV(values)
}
