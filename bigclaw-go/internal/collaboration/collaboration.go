package collaboration

import "sort"

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

func BuildThread(surface, targetID string, comments []CollaborationComment, decisions []DecisionNote) CollaborationThread {
	return CollaborationThread{
		Surface:   surface,
		TargetID:  targetID,
		Comments:  append([]CollaborationComment(nil), comments...),
		Decisions: append([]DecisionNote(nil), decisions...),
	}
}

func MergeThreads(targetID string, nativeThread, repoThread *CollaborationThread) *CollaborationThread {
	if nativeThread == nil && repoThread == nil {
		return nil
	}

	mergedComments := []CollaborationComment{}
	mergedDecisions := []DecisionNote{}
	for _, thread := range []*CollaborationThread{nativeThread, repoThread} {
		if thread == nil {
			continue
		}
		mergedComments = append(mergedComments, thread.Comments...)
		mergedDecisions = append(mergedDecisions, thread.Decisions...)
	}

	sort.SliceStable(mergedComments, func(i, j int) bool { return mergedComments[i].CreatedAt < mergedComments[j].CreatedAt })
	sort.SliceStable(mergedDecisions, func(i, j int) bool { return mergedDecisions[i].RecordedAt < mergedDecisions[j].RecordedAt })

	return &CollaborationThread{
		Surface:   "merged",
		TargetID:  targetID,
		Comments:  mergedComments,
		Decisions: mergedDecisions,
	}
}
