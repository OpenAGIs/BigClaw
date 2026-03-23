package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type publishAckOutcomeSurface struct {
	Ticket  string `json:"ticket"`
	Track   string `json:"track"`
	Status  string `json:"status"`
	Summary struct {
		ScenarioID         string   `json:"scenario_id"`
		ProofStatus        string   `json:"proof_status"`
		RequiredOutcomes   []string `json:"required_outcomes"`
		CommittedCount     int      `json:"committed_count"`
		RejectedCount      int      `json:"rejected_count"`
		UnknownCommitCount int      `json:"unknown_commit_count"`
	} `json:"summary"`
	SourceReports []string `json:"source_reports"`
	ReviewerLinks []string `json:"reviewer_links"`
	Outcomes      []struct {
		Outcome          string   `json:"outcome"`
		ProofRule        string   `json:"proof_rule"`
		RequiredEvidence []string `json:"required_evidence"`
		OperatorAction   string   `json:"operator_action"`
	} `json:"outcomes"`
	Limitations []string `json:"limitations"`
}

func TestPublishAckOutcomeSurfaceStaysAligned(t *testing.T) {
	root := repoRoot(t)

	var surface publishAckOutcomeSurface
	readJSONFile(t, filepath.Join(root, "docs", "reports", "publish-ack-outcome-surface.json"), &surface)

	var proof ambiguousPublishProofSummary
	readJSONFile(t, filepath.Join(root, "docs", "reports", "ambiguous-publish-outcome-proof-summary.json"), &proof)

	if surface.Ticket != "OPE-5" || surface.Track != "BIG-DUR-101" || surface.Status != "checked_in_surface" {
		t.Fatalf("unexpected publish-ack surface identity: %+v", surface)
	}
	if surface.Summary.ScenarioID != "BF-05" ||
		surface.Summary.ProofStatus != proof.Status ||
		surface.Summary.CommittedCount != proof.ScenarioSnapshot.PublishOutcomes.Committed ||
		surface.Summary.RejectedCount != proof.ScenarioSnapshot.PublishOutcomes.Rejected ||
		surface.Summary.UnknownCommitCount != proof.ScenarioSnapshot.PublishOutcomes.UnknownCommit {
		t.Fatalf("unexpected publish-ack summary: %+v", surface.Summary)
	}
	if len(surface.Summary.RequiredOutcomes) != 3 ||
		surface.Summary.RequiredOutcomes[0] != "committed" ||
		surface.Summary.RequiredOutcomes[1] != "rejected" ||
		surface.Summary.RequiredOutcomes[2] != "unknown_commit" {
		t.Fatalf("unexpected required outcomes ordering: %+v", surface.Summary.RequiredOutcomes)
	}
	if len(surface.SourceReports) != 4 || len(surface.ReviewerLinks) != 5 || len(surface.Outcomes) != 3 || len(surface.Limitations) != 3 {
		t.Fatalf("unexpected publish-ack surface sizes: sources=%d links=%d outcomes=%d limitations=%d", len(surface.SourceReports), len(surface.ReviewerLinks), len(surface.Outcomes), len(surface.Limitations))
	}

	for _, candidate := range surface.SourceReports {
		if _, err := os.Stat(resolveRepoPath(root, candidate)); err != nil {
			t.Fatalf("expected source report %q to exist: %v", candidate, err)
		}
	}
	if !containsPublishAckValue(surface.SourceReports, "docs/reports/ambiguous-publish-outcome-proof-summary.json") ||
		!containsPublishAckValue(surface.SourceReports, "docs/reports/replicated-event-log-durability-rollout-contract.md") {
		t.Fatalf("unexpected publish-ack source reports: %+v", surface.SourceReports)
	}
	for _, link := range []string{"/debug/status", "/v2/control-center", "/v2/reports/distributed", "/v2/reports/distributed/export", "docs/reports/ambiguous-publish-outcome-proof-summary.json"} {
		if !containsPublishAckValue(surface.ReviewerLinks, link) {
			t.Fatalf("expected reviewer links to include %q, got %+v", link, surface.ReviewerLinks)
		}
	}

	for _, outcome := range []string{"committed", "rejected", "unknown_commit"} {
		item := findPublishAckOutcome(surface.Outcomes, outcome)
		if item == nil {
			t.Fatalf("missing publish-ack outcome %q", outcome)
		}
		proofItem := findClassification(proof.ClassificationSummary, outcome)
		if proofItem == nil {
			t.Fatalf("missing ambiguous proof outcome %q", outcome)
		}
		if item.ProofRule != proofItem.ProofRule ||
			item.OperatorAction != proofItem.OperatorAction ||
			len(item.RequiredEvidence) != len(proofItem.RequiredEvidence) {
			t.Fatalf("publish-ack outcome %q drifted from ambiguous proof summary: surface=%+v proof=%+v", outcome, item, proofItem)
		}
		for _, evidence := range item.RequiredEvidence {
			if _, err := os.Stat(resolveRepoPath(root, evidence)); err != nil {
				t.Fatalf("expected publish-ack evidence %q to exist: %v", evidence, err)
			}
		}
	}

	for _, relative := range []string{
		"docs/reports/replicated-event-log-durability-rollout-contract.md",
		"docs/reports/review-readiness.md",
	} {
		body := readRepoFile(t, root, relative)
		if !strings.Contains(body, "publish-ack-outcome-surface.json") {
			t.Fatalf("expected %s to reference publish-ack outcome surface", relative)
		}
	}
}

func findPublishAckOutcome(items []struct {
	Outcome          string   `json:"outcome"`
	ProofRule        string   `json:"proof_rule"`
	RequiredEvidence []string `json:"required_evidence"`
	OperatorAction   string   `json:"operator_action"`
}, outcome string) *struct {
	Outcome          string   `json:"outcome"`
	ProofRule        string   `json:"proof_rule"`
	RequiredEvidence []string `json:"required_evidence"`
	OperatorAction   string   `json:"operator_action"`
} {
	for index := range items {
		if items[index].Outcome == outcome {
			return &items[index]
		}
	}
	return nil
}

func containsPublishAckValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
