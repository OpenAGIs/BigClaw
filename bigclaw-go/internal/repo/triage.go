package repo

import "strings"

type LineageEvidence struct {
	CandidateCommit     string `json:"candidate_commit"`
	AcceptedAncestor    string `json:"accepted_ancestor,omitempty"`
	SimilarFailureCount int    `json:"similar_failure_count,omitempty"`
	DiscussionOpen      int    `json:"discussion_open,omitempty"`
}

type TriageRecommendation struct {
	Action string `json:"action"`
	Reason string `json:"reason"`
}

type ApprovalEvidencePacket struct {
	RunID               string          `json:"run_id"`
	AcceptedCommitHash  string          `json:"accepted_commit_hash,omitempty"`
	CandidateCommitHash string          `json:"candidate_commit_hash,omitempty"`
	LineageSummary      string          `json:"lineage_summary,omitempty"`
	Links               []RunCommitLink `json:"links,omitempty"`
}

func RecommendTriageAction(status string, evidence LineageEvidence) TriageRecommendation {
	switch {
	case isFailureStatus(status) && evidence.SimilarFailureCount >= 2:
		return TriageRecommendation{Action: "replay", Reason: "similar lineage failures detected"}
	case strings.EqualFold(strings.TrimSpace(status), "needs-approval") && strings.TrimSpace(evidence.AcceptedAncestor) != "":
		return TriageRecommendation{Action: "approve", Reason: "accepted ancestor exists"}
	case evidence.DiscussionOpen > 0:
		return TriageRecommendation{Action: "handoff", Reason: "open repo discussion requires reviewer"}
	default:
		return TriageRecommendation{Action: "retry", Reason: "default retry path"}
	}
}

func BuildApprovalEvidencePacket(runID string, links []RunCommitLink, lineageSummary string) ApprovalEvidencePacket {
	packet := ApprovalEvidencePacket{
		RunID:          strings.TrimSpace(runID),
		LineageSummary: strings.TrimSpace(lineageSummary),
		Links:          append([]RunCommitLink(nil), links...),
	}
	for _, link := range links {
		switch strings.ToLower(strings.TrimSpace(link.Role)) {
		case "accepted":
			if packet.AcceptedCommitHash == "" {
				packet.AcceptedCommitHash = strings.TrimSpace(link.CommitHash)
			}
		case "candidate":
			if packet.CandidateCommitHash == "" {
				packet.CandidateCommitHash = strings.TrimSpace(link.CommitHash)
			}
		}
	}
	return packet
}

func isFailureStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "rejected":
		return true
	default:
		return false
	}
}
