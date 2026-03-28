package collaboration

import "sort"

type CollaborationComment struct {
	CommentID string   `json:"comment_id"`
	Author    string   `json:"author"`
	Body      string   `json:"body"`
	CreatedAt string   `json:"created_at"`
	Mentions  []string `json:"mentions,omitempty"`
	Anchor    string   `json:"anchor,omitempty"`
	Status    string   `json:"status"`
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

func BuildCollaborationThread(surface string, targetID string, comments []CollaborationComment, decisions []DecisionNote) CollaborationThread {
	return CollaborationThread{
		Surface:   surface,
		TargetID:  targetID,
		Comments:  append([]CollaborationComment(nil), comments...),
		Decisions: append([]DecisionNote(nil), decisions...),
	}
}

func MergeCollaborationThreads(targetID string, nativeThread *CollaborationThread, repoThread *CollaborationThread) *CollaborationThread {
	if nativeThread == nil && repoThread == nil {
		return nil
	}

	mergedComments := make([]CollaborationComment, 0)
	mergedDecisions := make([]DecisionNote, 0)
	for _, thread := range []*CollaborationThread{nativeThread, repoThread} {
		if thread == nil {
			continue
		}
		mergedComments = append(mergedComments, thread.Comments...)
		mergedDecisions = append(mergedDecisions, thread.Decisions...)
	}

	sort.Slice(mergedComments, func(i int, j int) bool {
		return mergedComments[i].CreatedAt < mergedComments[j].CreatedAt
	})
	sort.Slice(mergedDecisions, func(i int, j int) bool {
		return mergedDecisions[i].RecordedAt < mergedDecisions[j].RecordedAt
	})

	return &CollaborationThread{
		Surface:   "merged",
		TargetID:  targetID,
		Comments:  mergedComments,
		Decisions: mergedDecisions,
	}
}
