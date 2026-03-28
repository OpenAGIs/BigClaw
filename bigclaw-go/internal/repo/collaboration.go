package repo

import "sort"

type DecisionNote struct {
	DecisionID string   `json:"decision_id"`
	Author     string   `json:"author"`
	Outcome    string   `json:"outcome"`
	Summary    string   `json:"summary"`
	RecordedAt string   `json:"recorded_at"`
	Mentions   []string `json:"mentions,omitempty"`
	FollowUp   string   `json:"follow_up,omitempty"`
}

type CollaborationThread struct {
	Surface   string                 `json:"surface"`
	TargetID  string                 `json:"target_id"`
	Comments  []CollaborationComment `json:"comments,omitempty"`
	Decisions []DecisionNote         `json:"decisions,omitempty"`
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

	merged := &CollaborationThread{
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
	sort.Slice(merged.Comments, func(i, j int) bool {
		return merged.Comments[i].CreatedAt < merged.Comments[j].CreatedAt
	})
	sort.Slice(merged.Decisions, func(i, j int) bool {
		return merged.Decisions[i].RecordedAt < merged.Decisions[j].RecordedAt
	})
	return merged
}
