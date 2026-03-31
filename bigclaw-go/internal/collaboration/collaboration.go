package collaboration

import "sort"

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
	Surface   string     `json:"surface"`
	TargetID  string     `json:"target_id"`
	Comments  []Comment  `json:"comments,omitempty"`
	Decisions []Decision `json:"decisions,omitempty"`
}

func BuildThread(surface, targetID string, comments []Comment, decisions []Decision) Thread {
	return Thread{
		Surface:   surface,
		TargetID:  targetID,
		Comments:  append([]Comment(nil), comments...),
		Decisions: append([]Decision(nil), decisions...),
	}
}

func MergeThreads(targetID string, nativeThread, repoThread *Thread) *Thread {
	if nativeThread == nil && repoThread == nil {
		return nil
	}

	merged := Thread{
		Surface:  "merged",
		TargetID: targetID,
	}
	for _, thread := range []*Thread{nativeThread, repoThread} {
		if thread == nil {
			continue
		}
		merged.Comments = append(merged.Comments, thread.Comments...)
		merged.Decisions = append(merged.Decisions, thread.Decisions...)
	}

	sort.Slice(merged.Comments, func(i, j int) bool {
		return merged.Comments[i].CreatedAt < merged.Comments[j].CreatedAt
	})
	sort.Slice(merged.Decisions, func(i, j int) bool {
		return merged.Decisions[i].RecordedAt < merged.Decisions[j].RecordedAt
	})

	return &merged
}
