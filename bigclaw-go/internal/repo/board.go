package repo

import (
	"fmt"
	"time"
)

type CollaborationComment struct {
	CommentID string   `json:"comment_id"`
	Author    string   `json:"author"`
	Body      string   `json:"body"`
	CreatedAt string   `json:"created_at,omitempty"`
	Mentions  []string `json:"mentions,omitempty"`
	Anchor    string   `json:"anchor,omitempty"`
	Status    string   `json:"status,omitempty"`
}

type RepoPost struct {
	PostID        string         `json:"post_id"`
	Channel       string         `json:"channel"`
	Author        string         `json:"author"`
	Body          string         `json:"body"`
	TargetSurface string         `json:"target_surface,omitempty"`
	TargetID      string         `json:"target_id,omitempty"`
	ParentPostID  string         `json:"parent_post_id,omitempty"`
	CreatedAt     string         `json:"created_at,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

func NormalizeRepoPost(payload map[string]any) RepoPost {
	return RepoPost{
		PostID:        stringValue(payload["post_id"]),
		Channel:       stringValue(payload["channel"]),
		Author:        stringValue(payload["author"]),
		Body:          stringValue(payload["body"]),
		TargetSurface: defaultString(payload["target_surface"], "task"),
		TargetID:      stringValue(payload["target_id"]),
		ParentPostID:  stringValue(payload["parent_post_id"]),
		CreatedAt:     defaultString(payload["created_at"], repoNow()),
		Metadata:      mapValue(payload["metadata"]),
	}
}

func (p RepoPost) ToCollaborationComment() CollaborationComment {
	status := "open"
	if resolved, ok := p.Metadata["resolved"].(bool); ok && resolved {
		status = "resolved"
	}
	return CollaborationComment{
		CommentID: "repo-" + p.PostID,
		Author:    p.Author,
		Body:      p.Body,
		CreatedAt: p.CreatedAt,
		Anchor:    fmt.Sprintf("%s:%s", p.TargetSurface, p.TargetID),
		Status:    status,
	}
}

type DiscussionBoard struct {
	Posts []RepoPost `json:"posts,omitempty"`
}

func (b *DiscussionBoard) CreatePost(channel string, author string, body string, targetSurface string, targetID string, metadata map[string]any) RepoPost {
	post := RepoPost{
		PostID:        fmt.Sprintf("post-%d", len(b.Posts)+1),
		Channel:       channel,
		Author:        author,
		Body:          body,
		TargetSurface: targetSurface,
		TargetID:      targetID,
		CreatedAt:     repoNow(),
		Metadata:      mapValue(metadata),
	}
	b.Posts = append(b.Posts, post)
	return post
}

func (b *DiscussionBoard) Reply(parentPostID string, author string, body string) (RepoPost, error) {
	for _, parent := range b.Posts {
		if parent.PostID != parentPostID {
			continue
		}
		post := RepoPost{
			PostID:        fmt.Sprintf("post-%d", len(b.Posts)+1),
			Channel:       parent.Channel,
			Author:        author,
			Body:          body,
			TargetSurface: parent.TargetSurface,
			TargetID:      parent.TargetID,
			ParentPostID:  parentPostID,
			CreatedAt:     repoNow(),
			Metadata:      map[string]any{},
		}
		b.Posts = append(b.Posts, post)
		return post, nil
	}
	return RepoPost{}, fmt.Errorf("unknown parent post: %s", parentPostID)
}

func (b DiscussionBoard) ListPosts(channel string, targetSurface string, targetID string) []RepoPost {
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

func repoNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}
