package regression

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type ambiguousPublishProofSummary struct {
	Ticket           string `json:"ticket"`
	Track            string `json:"track"`
	Status           string `json:"status"`
	SourceValidation struct {
		ScenarioID      string   `json:"scenario_id"`
		StubReport      string   `json:"stub_report"`
		ValidationPack  string   `json:"validation_pack"`
		RolloutContract string   `json:"rollout_contract"`
		ReviewReadiness string   `json:"review_readiness"`
		CodePaths       []string `json:"code_paths"`
		ValidationTests []string `json:"validation_tests"`
	} `json:"source_validation"`
	ScenarioSnapshot struct {
		PublishOutcomes struct {
			Committed     int `json:"committed"`
			Rejected      int `json:"rejected"`
			UnknownCommit int `json:"unknown_commit"`
		} `json:"publish_outcomes"`
	} `json:"scenario_snapshot"`
	ClassificationSummary []struct {
		Outcome          string   `json:"outcome"`
		ProofRule        string   `json:"proof_rule"`
		RequiredEvidence []string `json:"required_evidence"`
		OperatorAction   string   `json:"operator_action"`
	} `json:"classification_summary"`
}

func TestAmbiguousPublishOutcomeProofSummary(t *testing.T) {
	repoRoot := repoRoot(t)
	reportPath := filepath.Join(repoRoot, "docs", "reports", "ambiguous-publish-outcome-proof-summary.json")
	contents, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read ambiguous publish proof summary: %v", err)
	}

	var report ambiguousPublishProofSummary
	if err := json.Unmarshal(contents, &report); err != nil {
		t.Fatalf("decode ambiguous publish proof summary: %v", err)
	}

	if report.Ticket != "OPE-260" || report.Track != "BIG-PAR-104" || report.Status != "repo-proof-summary" {
		t.Fatalf("unexpected report identity: %+v", report)
	}
	if report.SourceValidation.ScenarioID != "BF-05" {
		t.Fatalf("expected BF-05 source scenario, got %+v", report.SourceValidation)
	}
	if report.ScenarioSnapshot.PublishOutcomes.Committed != 1 || report.ScenarioSnapshot.PublishOutcomes.Rejected != 1 || report.ScenarioSnapshot.PublishOutcomes.UnknownCommit != 1 {
		t.Fatalf("expected one committed, rejected, and unknown_commit outcome, got %+v", report.ScenarioSnapshot.PublishOutcomes)
	}
	if len(report.ClassificationSummary) != 3 {
		t.Fatalf("expected committed/rejected/unknown_commit classifications, got %+v", report.ClassificationSummary)
	}
	for _, outcome := range []string{"committed", "rejected", "unknown_commit"} {
		item := findClassification(report.ClassificationSummary, outcome)
		if item == nil {
			t.Fatalf("missing classification outcome %q", outcome)
		}
		if strings.TrimSpace(item.ProofRule) == "" || strings.TrimSpace(item.OperatorAction) == "" || len(item.RequiredEvidence) == 0 {
			t.Fatalf("classification %q is missing proof detail: %+v", outcome, item)
		}
	}

	requiredPaths := []string{
		report.SourceValidation.StubReport,
		report.SourceValidation.ValidationPack,
		report.SourceValidation.RolloutContract,
		report.SourceValidation.ReviewReadiness,
	}
	requiredPaths = append(requiredPaths, report.SourceValidation.CodePaths...)
	requiredPaths = append(requiredPaths, report.SourceValidation.ValidationTests...)
	for _, candidate := range requiredPaths {
		if candidate == "" {
			t.Fatal("proof summary contains an empty path")
		}
		if _, err := os.Stat(resolveRepoPath(repoRoot, candidate)); err != nil {
			t.Fatalf("expected referenced path %q to exist: %v", candidate, err)
		}
	}

	for _, candidate := range []string{
		report.SourceValidation.ValidationPack,
		report.SourceValidation.RolloutContract,
		report.SourceValidation.ReviewReadiness,
	} {
		doc, err := os.ReadFile(resolveRepoPath(repoRoot, candidate))
		if err != nil {
			t.Fatalf("read linked doc %q: %v", candidate, err)
		}
		if !strings.Contains(string(doc), "ambiguous-publish-outcome-proof-summary.json") {
			t.Fatalf("expected %q to reference the ambiguous publish proof summary", candidate)
		}
	}
}

func findClassification(items []struct {
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
