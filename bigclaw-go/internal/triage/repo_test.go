package triage

import (
	"reflect"
	"testing"
)

func TestRecommendRepoActionFollowsLineageAndDiscussionEvidence(t *testing.T) {
	replay := RecommendRepoAction("failed", LineageEvidence{SimilarFailureCount: 2})
	if replay.Action != "replay" || replay.Reason != "similar lineage failures detected" {
		t.Fatalf("unexpected replay recommendation: %+v", replay)
	}

	approve := RecommendRepoAction("needs-approval", LineageEvidence{AcceptedAncestor: "abc123"})
	if approve.Action != "approve" || approve.Reason != "accepted ancestor exists" {
		t.Fatalf("unexpected approve recommendation: %+v", approve)
	}

	handoff := RecommendRepoAction("running", LineageEvidence{DiscussionOpen: 1})
	if handoff.Action != "handoff" || handoff.Reason != "open repo discussion requires reviewer" {
		t.Fatalf("unexpected handoff recommendation: %+v", handoff)
	}

	retry := RecommendRepoAction("running", LineageEvidence{})
	if retry.Action != "retry" || retry.Reason != "default retry path" {
		t.Fatalf("unexpected retry recommendation: %+v", retry)
	}
}

func TestApprovalEvidencePacketCapturesAcceptedAndCandidateLinks(t *testing.T) {
	links := []map[string]string{
		{"role": "source", "commit_hash": "aaa111"},
		{"role": "candidate", "commit_hash": "bbb222"},
		{"role": "accepted", "commit_hash": "ccc333"},
	}
	packet := ApprovalEvidencePacket("run-143", links, "accepted ancestor found")
	if packet["run_id"] != "run-143" || packet["accepted_commit_hash"] != "ccc333" || packet["candidate_commit_hash"] != "bbb222" || packet["lineage_summary"] != "accepted ancestor found" {
		t.Fatalf("unexpected approval evidence packet: %+v", packet)
	}
	if got, ok := packet["links"].([]map[string]string); !ok || !reflect.DeepEqual(got, links) {
		t.Fatalf("expected links to survive packet build, got %+v", packet["links"])
	}
}
