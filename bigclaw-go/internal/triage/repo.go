package triage

type LineageEvidence struct {
	CandidateCommit     string `json:"candidate_commit"`
	AcceptedAncestor    string `json:"accepted_ancestor,omitempty"`
	SimilarFailureCount int    `json:"similar_failure_count,omitempty"`
	DiscussionOpen      int    `json:"discussion_open,omitempty"`
}

type Recommendation struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

func RecommendRepoAction(status string, evidence LineageEvidence) Recommendation {
	switch {
	case (status == "failed" || status == "rejected") && evidence.SimilarFailureCount >= 2:
		return Recommendation{Action: "replay", Reason: "similar lineage failures detected"}
	case status == "needs-approval" && evidence.AcceptedAncestor != "":
		return Recommendation{Action: "approve", Reason: "accepted ancestor exists"}
	case evidence.DiscussionOpen > 0:
		return Recommendation{Action: "handoff", Reason: "open repo discussion requires reviewer"}
	default:
		return Recommendation{Action: "retry", Reason: "default retry path"}
	}
}

func ApprovalEvidencePacket(runID string, links []map[string]string, lineageSummary string) map[string]any {
	accepted := ""
	candidate := ""
	for _, link := range links {
		if accepted == "" && link["role"] == "accepted" {
			accepted = link["commit_hash"]
		}
		if candidate == "" && link["role"] == "candidate" {
			candidate = link["commit_hash"]
		}
	}
	return map[string]any{
		"run_id":                runID,
		"accepted_commit_hash":  accepted,
		"candidate_commit_hash": candidate,
		"lineage_summary":       lineageSummary,
		"links":                 links,
	}
}
