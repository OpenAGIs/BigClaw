package collaboration

import (
	"fmt"
	"strings"
)

type Comment struct {
	CommentID string   `json:"comment_id"`
	Author    string   `json:"author"`
	Body      string   `json:"body"`
	CreatedAt string   `json:"created_at,omitempty"`
	Mentions  []string `json:"mentions,omitempty"`
	Anchor    string   `json:"anchor,omitempty"`
	Status    string   `json:"status,omitempty"`
}

type Decision struct {
	DecisionID        string   `json:"decision_id"`
	Author            string   `json:"author"`
	Outcome           string   `json:"outcome"`
	Summary           string   `json:"summary"`
	RecordedAt        string   `json:"recorded_at,omitempty"`
	Mentions          []string `json:"mentions,omitempty"`
	RelatedCommentIDs []string `json:"related_comment_ids,omitempty"`
	FollowUp          string   `json:"follow_up,omitempty"`
}

type Thread struct {
	Surface      string     `json:"surface"`
	TargetID     string     `json:"target_id"`
	MentionCount int        `json:"mention_count"`
	Comments     []Comment  `json:"comments,omitempty"`
	Decisions    []Decision `json:"decisions,omitempty"`
}

type RepoDiscussionBoard struct {
	posts []RepoPost
}

type RepoPost struct {
	Channel       string `json:"channel"`
	Author        string `json:"author"`
	Body          string `json:"body"`
	TargetSurface string `json:"target_surface"`
	TargetID      string `json:"target_id"`
}

func BuildCollaborationThread(surface string, targetID string, comments []Comment, decisions []Decision) Thread {
	return Thread{
		Surface:      surface,
		TargetID:     targetID,
		MentionCount: mentionCount(comments, decisions),
		Comments:     append([]Comment(nil), comments...),
		Decisions:    append([]Decision(nil), decisions...),
	}
}

func (b *RepoDiscussionBoard) CreatePost(channel, author, body, targetSurface, targetID string) RepoPost {
	post := RepoPost{
		Channel:       channel,
		Author:        author,
		Body:          body,
		TargetSurface: targetSurface,
		TargetID:      targetID,
	}
	b.posts = append(b.posts, post)
	return post
}

func (p RepoPost) ToComment() Comment {
	return Comment{
		Author: p.Author,
		Body:   p.Body,
		Status: "open",
	}
}

func MergeCollaborationThreads(targetID string, nativeThread *Thread, repoThread *Thread) *Thread {
	if nativeThread == nil && repoThread == nil {
		return nil
	}
	merged := &Thread{
		Surface:  "merged",
		TargetID: targetID,
	}
	if nativeThread != nil {
		merged.Comments = append(merged.Comments, nativeThread.Comments...)
		merged.Decisions = append(merged.Decisions, nativeThread.Decisions...)
	}
	if repoThread != nil {
		merged.Comments = append(merged.Comments, repoThread.Comments...)
	}
	return merged
}

func BuildCollaborationThreadFromAudits(audits []map[string]any, surface string, targetID string) *Thread {
	comments := make([]Comment, 0)
	decisions := make([]Decision, 0)
	for _, audit := range audits {
		action, _ := audit["action"].(string)
		details, _ := audit["details"].(map[string]any)
		switch action {
		case "collaboration.comment":
			if details == nil {
				continue
			}
			comments = append(comments, Comment{
				CommentID: stringValue(details["comment_id"]),
				Author:    firstNonEmpty(stringValue(details["author"]), stringValue(audit["actor"])),
				Body:      stringValue(details["body"]),
				CreatedAt: firstNonEmpty(stringValue(audit["timestamp"]), stringValue(details["created_at"]), stringValue(details["anchor"])),
				Mentions:  stringSlice(details["mentions"]),
				Anchor:    stringValue(details["anchor"]),
				Status:    firstNonEmpty(stringValue(details["status"]), "open"),
			})
		case "collaboration.decision":
			if details == nil {
				continue
			}
			decisions = append(decisions, Decision{
				DecisionID:        stringValue(details["decision_id"]),
				Author:            firstNonEmpty(stringValue(details["author"]), stringValue(audit["actor"])),
				Outcome:           firstNonEmpty(stringValue(details["outcome"]), stringValue(audit["outcome"])),
				Summary:           stringValue(details["summary"]),
				RecordedAt:        firstNonEmpty(stringValue(audit["timestamp"]), stringValue(details["recorded_at"])),
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

func mentionCount(comments []Comment, decisions []Decision) int {
	total := 0
	for _, comment := range comments {
		if len(comment.Mentions) > 0 {
			total += len(comment.Mentions)
		} else {
			total += strings.Count(comment.Body, "@")
		}
	}
	for _, decision := range decisions {
		if len(decision.Mentions) > 0 {
			total += len(decision.Mentions)
		} else {
			total += strings.Count(decision.Summary, "@")
		}
		total += strings.Count(decision.FollowUp, "@")
	}
	return total
}

func (t Thread) ParticipantCount() int {
	participants := make(map[string]struct{})
	for _, comment := range t.Comments {
		if author := strings.TrimSpace(comment.Author); author != "" {
			participants[author] = struct{}{}
		}
	}
	for _, decision := range t.Decisions {
		if author := strings.TrimSpace(decision.Author); author != "" {
			participants[author] = struct{}{}
		}
	}
	return len(participants)
}

func (t Thread) OpenCommentCount() int {
	count := 0
	for _, comment := range t.Comments {
		if !strings.EqualFold(strings.TrimSpace(comment.Status), "resolved") {
			count++
		}
	}
	return count
}

func (t Thread) Recommendation() string {
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

func RenderCollaborationLines(thread *Thread) []string {
	if thread == nil {
		return nil
	}
	lines := []string{
		"## Collaboration",
		"",
		"- Surface: " + thread.Surface,
		"- Target: " + thread.TargetID,
		fmt.Sprintf("- Participants: %d", thread.ParticipantCount()),
		fmt.Sprintf("- Comments: %d", len(thread.Comments)),
		fmt.Sprintf("- Open Comments: %d", thread.OpenCommentCount()),
		fmt.Sprintf("- Mentions: %d", thread.MentionCount),
		fmt.Sprintf("- Decision Notes: %d", len(thread.Decisions)),
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
			lines = append(lines, fmt.Sprintf("- %s: author=%s status=%s anchor=%s mentions=%s body=%s", firstNonEmpty(comment.CommentID, "comment"), comment.Author, firstNonEmpty(comment.Status, "open"), firstNonEmpty(comment.Anchor, "none"), mentions, comment.Body))
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
		lines = append(lines, fmt.Sprintf("- %s: author=%s outcome=%s mentions=%s related=%s summary=%s follow_up=%s", firstNonEmpty(decision.DecisionID, "decision"), decision.Author, decision.Outcome, mentions, related, decision.Summary, firstNonEmpty(decision.FollowUp, "none")))
	}
	return lines
}

func stringValue(value any) string {
	switch current := value.(type) {
	case string:
		return current
	case fmt.Stringer:
		return current.String()
	default:
		return ""
	}
}

func stringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		if direct, ok := value.([]string); ok {
			return append([]string(nil), direct...)
		}
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if str := stringValue(item); str != "" {
			out = append(out, str)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
