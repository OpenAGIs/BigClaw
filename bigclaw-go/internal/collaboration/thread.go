package collaboration

import (
	"fmt"
	"strings"
)

type Comment struct {
	CommentID string `json:"comment_id"`
	Author    string `json:"author"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at,omitempty"`
}

type Decision struct {
	DecisionID string `json:"decision_id"`
	Author     string `json:"author"`
	Outcome    string `json:"outcome"`
	Summary    string `json:"summary"`
	RecordedAt string `json:"recorded_at,omitempty"`
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
				Author:    stringValue(details["author"]),
				Body:      stringValue(details["body"]),
				CreatedAt: stringValue(details["anchor"]),
			})
		case "collaboration.decision":
			if details == nil {
				continue
			}
			decisions = append(decisions, Decision{
				DecisionID: stringValue(details["decision_id"]),
				Author:     stringValue(details["author"]),
				Outcome:    stringValue(details["outcome"]),
				Summary:    stringValue(details["summary"]),
				RecordedAt: stringValue(details["follow_up"]),
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
		total += strings.Count(comment.Body, "@")
	}
	for _, decision := range decisions {
		total += strings.Count(decision.Summary, "@")
		total += strings.Count(decision.RecordedAt, "@")
	}
	return total
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
