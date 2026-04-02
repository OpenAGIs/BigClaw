package repo

import "bigclaw-go/internal/collaboration"

type DiscussionPost struct {
	Channel       string `json:"channel"`
	Author        string `json:"author"`
	Body          string `json:"body"`
	TargetSurface string `json:"target_surface"`
	TargetID      string `json:"target_id"`
}

func (p DiscussionPost) ToCollaborationComment() collaboration.Comment {
	return collaboration.Comment{
		CommentID: p.Channel + ":" + p.TargetID,
		Author:    p.Author,
		Body:      p.Body,
	}
}

type DiscussionBoard struct{}

func (DiscussionBoard) CreatePost(channel string, author string, body string, targetSurface string, targetID string) DiscussionPost {
	return DiscussionPost{
		Channel:       channel,
		Author:        author,
		Body:          body,
		TargetSurface: targetSurface,
		TargetID:      targetID,
	}
}
