package collaboration

import "sort"

type Comment struct {
	CommentID string
	Author    string
	Body      string
	CreatedAt string
	Anchor    string
}

type Decision struct {
	DecisionID string
	Author     string
	Outcome    string
	Summary    string
	RecordedAt string
}

type Thread struct {
	Surface   string
	TargetID  string
	Comments  []Comment
	Decisions []Decision
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

	merged := &Thread{
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
	sort.SliceStable(merged.Comments, func(i, j int) bool {
		return merged.Comments[i].CreatedAt < merged.Comments[j].CreatedAt
	})
	sort.SliceStable(merged.Decisions, func(i, j int) bool {
		return merged.Decisions[i].RecordedAt < merged.Decisions[j].RecordedAt
	})
	return merged
}
