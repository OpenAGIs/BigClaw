package collaboration

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
	Surface   string     `json:"surface"`
	TargetID  string     `json:"target_id"`
	Comments  []Comment  `json:"comments,omitempty"`
	Decisions []Decision `json:"decisions,omitempty"`
}

func BuildThread(surface string, targetID string, comments []Comment, decisions []Decision) *Thread {
	return &Thread{
		Surface:   surface,
		TargetID:  targetID,
		Comments:  append([]Comment(nil), comments...),
		Decisions: append([]Decision(nil), decisions...),
	}
}

func MergeThreads(targetID string, nativeThread *Thread, repoThread *Thread) *Thread {
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
		merged.Decisions = append(merged.Decisions, repoThread.Decisions...)
	}
	return merged
}
