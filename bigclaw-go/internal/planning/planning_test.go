package planning

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"bigclaw-go/internal/governance"
)

func TestCandidateReadinessScoreAndGateEvaluation(t *testing.T) {
	backlog := CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3候选与进入条件",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{CandidateID: "candidate-a", Title: "A", Priority: "P0", Owner: "ops", Outcome: "ship", ValidationCommand: "go test ./...", Capabilities: []string{"release-gate"}, Evidence: []string{"validation-report"}},
			{CandidateID: "candidate-b", Title: "B", Priority: "P1", Owner: "ops", Outcome: "ship", ValidationCommand: "go test ./...", Capabilities: []string{"ops-control", "commercialization"}, Evidence: []string{"pilot-evidence", "validation-report"}},
			{CandidateID: "candidate-c", Title: "C", Priority: "P0", Owner: "ops", Outcome: "blocked", ValidationCommand: "go test ./...", Capabilities: []string{"reporting"}, Evidence: []string{"weekly-review"}, Blockers: []string{"waiting"}},
		},
	}
	gate := EntryGate{
		GateID:                  "gate-v3-entry",
		Name:                    "v3 entry gate",
		MinReadyCandidates:      2,
		RequiredCapabilities:    []string{"release-gate", "ops-control", "commercialization"},
		RequiredEvidence:        []string{"validation-report"},
		RequiredBaselineVersion: "v4.0",
		MaxBlockers:             1,
	}
	baseline := &governance.ScopeFreezeAudit{Version: "v4.0", TotalItems: 1}
	decision := CandidatePlanner{}.EvaluateGate(backlog, gate, baseline)
	if !decision.Passed {
		t.Fatalf("expected gate to pass, got %+v", decision)
	}
	if !reflect.DeepEqual(decision.ReadyCandidateIDs, []string{"candidate-a", "candidate-b"}) {
		t.Fatalf("unexpected ready candidates: %+v", decision.ReadyCandidateIDs)
	}
	if !reflect.DeepEqual(decision.BlockedCandidateIDs, []string{"candidate-c"}) {
		t.Fatalf("unexpected blocked candidates: %+v", decision.BlockedCandidateIDs)
	}
	if decision.Summary() != "PASS: ready=2 blocked=1 missing_capabilities=0 missing_evidence=0 baseline_findings=0" {
		t.Fatalf("unexpected decision summary: %s", decision.Summary())
	}
}

func TestCandidatePlannerFlagsBaselineFindings(t *testing.T) {
	backlog := CandidateBacklog{Candidates: []CandidateEntry{{CandidateID: "candidate-a", Title: "A", Priority: "P0", Owner: "ops", Outcome: "ship", ValidationCommand: "go test", Capabilities: []string{"release-gate"}, Evidence: []string{"validation-report"}}}}
	gate := EntryGate{GateID: "gate", Name: "entry", MinReadyCandidates: 1, RequiredBaselineVersion: "v4.0"}
	missing := CandidatePlanner{}.EvaluateGate(backlog, gate, nil)
	if missing.Passed || !reflect.DeepEqual(missing.BaselineFindings, []string{"missing baseline audit for v4.0"}) {
		t.Fatalf("expected missing baseline finding, got %+v", missing)
	}
	failed := CandidatePlanner{}.EvaluateGate(backlog, gate, &governance.ScopeFreezeAudit{
		Version:           "v3.9",
		MissingOwners:     []string{"BIG-1"},
		MissingValidation: []string{"BIG-1"},
	})
	want := []string{
		"baseline version mismatch: expected v4.0, got v3.9",
		"baseline v3.9 is not release ready (75.0)",
	}
	if !reflect.DeepEqual(failed.BaselineFindings, want) {
		t.Fatalf("unexpected baseline findings: %+v", failed.BaselineFindings)
	}
}

func TestRenderCandidateBacklogReportAndSeedBuilders(t *testing.T) {
	backlog := BuildV3CandidateBacklog()
	gate := BuildV3EntryGate()
	decision := CandidatePlanner{}.EvaluateGate(backlog, gate, &governance.ScopeFreezeAudit{Version: "v4.0", TotalItems: 1})
	report := RenderCandidateBacklogReport(backlog, gate, decision)
	for _, needle := range []string{
		"# V3 Candidate Backlog Report",
		"- Gate: v3 entry gate",
		"- candidate-ops-hardening: Operations command-center hardening priority=P0 owner=engineering-operations",
		"  - command-center-src -> src/bigclaw/operations.py capability=ops-control",
		"  - report-studio-tests -> tests/test_reports.py capability=commercialization",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("expected report to contain %q, got %s", needle, report)
		}
	}
	payload, err := json.Marshal(backlog)
	if err != nil {
		t.Fatalf("marshal backlog: %v", err)
	}
	var restored CandidateBacklog
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal backlog: %v", err)
	}
	if restored.EpicID != backlog.EpicID || len(restored.Candidates) != len(backlog.Candidates) {
		t.Fatalf("unexpected restored backlog: %+v", restored)
	}
}
