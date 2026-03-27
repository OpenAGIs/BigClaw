package repo

import (
	"fmt"
	"strings"
	"time"
)

type RepoPost struct {
	PostID        string         `json:"post_id"`
	Channel       string         `json:"channel"`
	Author        string         `json:"author"`
	Body          string         `json:"body"`
	TargetSurface string         `json:"target_surface,omitempty"`
	TargetID      string         `json:"target_id,omitempty"`
	ParentPostID  string         `json:"parent_post_id,omitempty"`
	CreatedAt     string         `json:"created_at"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type CollaborationComment struct {
	CommentID string   `json:"comment_id"`
	Author    string   `json:"author"`
	Body      string   `json:"body"`
	CreatedAt string   `json:"created_at"`
	Mentions  []string `json:"mentions,omitempty"`
	Anchor    string   `json:"anchor,omitempty"`
	Status    string   `json:"status"`
}

type RepoDiscussionBoard struct {
	Posts []RepoPost `json:"posts,omitempty"`
	Now   func() time.Time
}

func RepoPostFromMap(payload map[string]any) RepoPost {
	post := RepoPost{
		PostID:        stringValue(payload["post_id"]),
		Channel:       stringValue(payload["channel"]),
		Author:        stringValue(payload["author"]),
		Body:          stringValue(payload["body"]),
		TargetSurface: firstNonEmptyString(stringValue(payload["target_surface"]), "task"),
		TargetID:      stringValue(payload["target_id"]),
		ParentPostID:  stringValue(payload["parent_post_id"]),
		CreatedAt:     stringValue(payload["created_at"]),
		Metadata:      copyMap(mapValue(payload["metadata"])),
	}
	if post.CreatedAt == "" {
		post.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return post
}

func (p RepoPost) ToMap() map[string]any {
	return map[string]any{
		"post_id":        p.PostID,
		"channel":        p.Channel,
		"author":         p.Author,
		"body":           p.Body,
		"target_surface": firstNonEmptyString(p.TargetSurface, "task"),
		"target_id":      p.TargetID,
		"parent_post_id": p.ParentPostID,
		"created_at":     p.CreatedAt,
		"metadata":       copyMap(p.Metadata),
	}
}

func (p RepoPost) ToCollaborationComment() CollaborationComment {
	status := "open"
	if resolved, ok := p.Metadata["resolved"].(bool); ok && resolved {
		status = "resolved"
	}
	targetSurface := firstNonEmptyString(p.TargetSurface, "task")
	return CollaborationComment{
		CommentID: "repo-" + strings.TrimSpace(p.PostID),
		Author:    p.Author,
		Body:      p.Body,
		CreatedAt: p.CreatedAt,
		Anchor:    targetSurface + ":" + p.TargetID,
		Status:    status,
	}
}

func (b *RepoDiscussionBoard) CreatePost(channel, author, body, targetSurface, targetID string, metadata map[string]any) RepoPost {
	post := RepoPost{
		PostID:        fmt.Sprintf("post-%d", len(b.Posts)+1),
		Channel:       channel,
		Author:        author,
		Body:          body,
		TargetSurface: firstNonEmptyString(targetSurface, "task"),
		TargetID:      targetID,
		CreatedAt:     b.now().UTC().Format(time.RFC3339),
		Metadata:      copyMap(metadata),
	}
	b.Posts = append(b.Posts, post)
	return post
}

func (b *RepoDiscussionBoard) Reply(parentPostID, author, body string) (RepoPost, error) {
	for _, post := range b.Posts {
		if post.PostID != parentPostID {
			continue
		}
		reply := RepoPost{
			PostID:        fmt.Sprintf("post-%d", len(b.Posts)+1),
			Channel:       post.Channel,
			Author:        author,
			Body:          body,
			TargetSurface: post.TargetSurface,
			TargetID:      post.TargetID,
			ParentPostID:  parentPostID,
			CreatedAt:     b.now().UTC().Format(time.RFC3339),
		}
		b.Posts = append(b.Posts, reply)
		return reply, nil
	}
	return RepoPost{}, fmt.Errorf("unknown parent post: %s", parentPostID)
}

func (b RepoDiscussionBoard) ListPosts(channel, targetSurface, targetID string) []RepoPost {
	result := make([]RepoPost, 0, len(b.Posts))
	for _, post := range b.Posts {
		if channel != "" && post.Channel != channel {
			continue
		}
		if targetSurface != "" && post.TargetSurface != targetSurface {
			continue
		}
		if targetID != "" && post.TargetID != targetID {
			continue
		}
		result = append(result, post)
	}
	return result
}

func (b RepoDiscussionBoard) now() time.Time {
	if b.Now != nil {
		return b.Now()
	}
	return time.Now()
}

func copyMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func mapValue(value any) map[string]any {
	if value == nil {
		return nil
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return nil
}

func firstNonEmptyString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
