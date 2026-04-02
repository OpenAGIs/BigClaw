package planning

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"bigclaw-go/internal/governance"
)

func TestCandidateBacklogRoundTripAndRanking(t *testing.T) {
	backlog := CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3候选与进入条件",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{
				CandidateID:       "candidate-risky",
				Title:             "Risky migration",
				Priority:          "P0",
				Owner:             "runtime",
				ValidationCommand: "cd bigclaw-go && go test ./internal/worker ./internal/scheduler",
				Capabilities:      []string{"runtime-hardening"},
				Evidence:          []string{"benchmark"},
				Blockers:          []string{"missing rollback plan"},
			},
			{
				CandidateID:       "candidate-ready",
				Title:             "Release control center",
				Priority:          "P1",
				Owner:             "platform-ui",
				ValidationCommand: "python3 -m pytest tests/test_design_system.py -q",
				Capabilities:      []string{"release-gate", "reporting"},
				Evidence:          []string{"acceptance-suite", "validation-report"},
			},
		},
	}

	body, err := json.Marshal(backlog)
	if err != nil {
		t.Fatalf("marshal backlog: %v", err)
	}
	var restored CandidateBacklog
	if err := json.Unmarshal(body, &restored); err != nil {
		t.Fatalf("unmarshal backlog: %v", err)
	}
	if !reflect.DeepEqual(restored, backlog) {
		t.Fatalf("unexpected roundtrip backlog: %+v", restored)
	}
	ranked := restored.RankedCandidates()
	if got := []string{ranked[0].CandidateID, ranked[1].CandidateID}; !reflect.DeepEqual(got, []string{"candidate-ready", "candidate-risky"}) {
		t.Fatalf("unexpected ranked ids: %+v", got)
	}
}

func TestEntryGateEvaluationHandlesCapabilitiesEvidenceAndBaseline(t *testing.T) {
	backlog := CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3候选与进入条件",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{CandidateID: "candidate-release-control", Capabilities: []string{"release-gate", "reporting"}, Evidence: []string{"acceptance-suite", "validation-report"}},
			{CandidateID: "candidate-ops-hardening", Capabilities: []string{"ops-control"}, Evidence: []string{"weekly-review"}},
			{CandidateID: "candidate-orchestration", Capabilities: []string{"commercialization", "handoff"}, Evidence: []string{"pilot-evidence"}},
		},
	}
	gate := EntryGate{
		GateID:                  "gate-v3-entry",
		Name:                    "V3 Entry Gate",
		MinReadyCandidates:      3,
		RequiredCapabilities:    []string{"release-gate", "ops-control", "commercialization"},
		RequiredEvidence:        []string{"acceptance-suite", "pilot-evidence", "validation-report"},
		RequiredBaselineVersion: "v2.0",
	}

	decision := CandidatePlanner{}.EvaluateGate(backlog, gate, &governance.ScopeFreezeAudit{
		BoardName:  "BigClaw v2.0 Freeze",
		Version:    "v2.0",
		TotalItems: 5,
	})
	if !decision.Passed || !decision.BaselineReady || len(decision.MissingCapabilities) != 0 || len(decision.MissingEvidence) != 0 {
		t.Fatalf("unexpected passing gate decision: %+v", decision)
	}

	missingBaseline := CandidatePlanner{}.EvaluateGate(backlog, gate, nil)
	if missingBaseline.Passed || missingBaseline.BaselineReady || !reflect.DeepEqual(missingBaseline.BaselineFindings, []string{"missing baseline audit for v2.0"}) {
		t.Fatalf("unexpected missing baseline decision: %+v", missingBaseline)
	}

	failedBaseline := CandidatePlanner{}.EvaluateGate(backlog, gate, &governance.ScopeFreezeAudit{
		BoardName:         "BigClaw v2.0 Freeze",
		Version:           "v2.0",
		TotalItems:        5,
		MissingValidation: []string{"OPE-116"},
	})
	if failedBaseline.Passed || failedBaseline.BaselineReady || !reflect.DeepEqual(failedBaseline.BaselineFindings, []string{"baseline v2.0 is not release ready (87.5)"}) {
		t.Fatalf("unexpected failed baseline decision: %+v", failedBaseline)
	}
}

func TestEntryGateDecisionRoundTripAndReport(t *testing.T) {
	decision := EntryGateDecision{
		GateID:              "gate-v3-entry",
		Passed:              false,
		ReadyCandidateIDs:   []string{"candidate-release-control"},
		BlockedCandidateIDs: []string{"candidate-runtime"},
		MissingCapabilities: []string{"commercialization"},
		MissingEvidence:     []string{"pilot-evidence"},
		BaselineReady:       false,
		BaselineFindings:    []string{"baseline v2.0 is not release ready (87.5)"},
		BlockerCount:        1,
	}
	body, err := json.Marshal(decision)
	if err != nil {
		t.Fatalf("marshal decision: %v", err)
	}
	var restored EntryGateDecision
	if err := json.Unmarshal(body, &restored); err != nil {
		t.Fatalf("unmarshal decision: %v", err)
	}
	if !reflect.DeepEqual(restored, decision) {
		t.Fatalf("unexpected roundtrip decision: %+v", restored)
	}

	backlog := CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3候选与进入条件",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{{
			CandidateID:       "candidate-release-control",
			Title:             "Release control center",
			Priority:          "P0",
			Owner:             "platform-ui",
			ValidationCommand: "python3 -m pytest tests/test_design_system.py -q",
			Capabilities:      []string{"release-gate", "reporting"},
			Evidence:          []string{"acceptance-suite", "validation-report"},
			EvidenceLinks:     []EvidenceLink{{Label: "ui-acceptance", Target: "tests/test_design_system.py", Capability: "release-gate"}},
		}},
	}
	gate := EntryGate{
		GateID:                  "gate-v3-entry",
		Name:                    "V3 Entry Gate",
		MinReadyCandidates:      1,
		RequiredCapabilities:    []string{"release-gate"},
		RequiredEvidence:        []string{"validation-report"},
		RequiredBaselineVersion: "v2.0",
	}
	passDecision := CandidatePlanner{}.EvaluateGate(backlog, gate, &governance.ScopeFreezeAudit{BoardName: "BigClaw v2.0 Freeze", Version: "v2.0", TotalItems: 5})
	report := RenderCandidateBacklogReport(backlog, gate, passDecision)
	for _, fragment := range []string{
		"# V3 Candidate Backlog Report",
		"- Epic: BIG-EPIC-20 v4.0 v3候选与进入条件",
		"- Decision: PASS: ready=1 blocked=0 missing_capabilities=0 missing_evidence=0 baseline_findings=0",
		"- candidate-release-control: Release control center priority=P0 owner=platform-ui score=100 ready=True",
		"validation=python3 -m pytest tests/test_design_system.py -q",
		"- ui-acceptance -> tests/test_design_system.py capability=release-gate",
		"- Missing evidence: none",
		"- Baseline ready: True",
		"- Baseline findings: none",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestCandidateEntryRoundTripPreservesEvidenceLinks(t *testing.T) {
	candidate := CandidateEntry{
		CandidateID:       "candidate-ops-hardening",
		Title:             "Ops hardening",
		Theme:             "ops-command-center",
		Priority:          "P0",
		Owner:             "ops-platform",
		Outcome:           "Package command-center and approval surfaces with linked evidence.",
		ValidationCommand: "python3 -m pytest tests/test_operations.py tests/test_saved_views.py -q",
		Capabilities:      []string{"ops-control", "saved-views"},
		Evidence:          []string{"weekly-review", "validation-report"},
		EvidenceLinks: []EvidenceLink{
			{Label: "queue-control-center", Target: "src/bigclaw/operations.py", Capability: "ops-control", Note: "queue and approval command center"},
			{Label: "saved-view-report", Target: "src/bigclaw/saved_views.py", Capability: "saved-views", Note: "team saved views and digest evidence"},
		},
	}
	body, err := json.Marshal(candidate)
	if err != nil {
		t.Fatalf("marshal candidate: %v", err)
	}
	var restored CandidateEntry
	if err := json.Unmarshal(body, &restored); err != nil {
		t.Fatalf("unmarshal candidate: %v", err)
	}
	if !reflect.DeepEqual(restored, candidate) {
		t.Fatalf("unexpected roundtrip candidate: %+v", restored)
	}
}

func TestFourWeekExecutionPlanRollupsValidationAndReport(t *testing.T) {
	plan := BuildBig4701ExecutionPlan()
	body, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal plan: %v", err)
	}
	var restored FourWeekExecutionPlan
	if err := json.Unmarshal(body, &restored); err != nil {
		t.Fatalf("unmarshal plan: %v", err)
	}
	if !reflect.DeepEqual(restored, plan) {
		t.Fatalf("unexpected roundtrip plan: %+v", restored)
	}
	if plan.TotalGoals() != 8 || plan.CompletedGoals() != 2 || plan.OverallProgressPercent() != 25 {
		t.Fatalf("unexpected plan rollups: %+v", plan)
	}
	if !reflect.DeepEqual(plan.AtRiskWeeks(), []int{2}) {
		t.Fatalf("unexpected at-risk weeks: %+v", plan.AtRiskWeeks())
	}
	if got := plan.GoalStatusCounts(); !reflect.DeepEqual(got, map[string]int{"done": 2, "on-track": 1, "at-risk": 1, "not-started": 4}) {
		t.Fatalf("unexpected status counts: %+v", got)
	}

	invalid := FourWeekExecutionPlan{
		PlanID: "BIG-4701",
		Title:  "4周执行计划与周目标",
		Owner:  "execution-office",
		Weeks: []WeeklyExecutionPlan{
			{WeekNumber: 1, Theme: "One", Objective: "One"},
			{WeekNumber: 3, Theme: "Three", Objective: "Three"},
			{WeekNumber: 2, Theme: "Two", Objective: "Two"},
			{WeekNumber: 4, Theme: "Four", Objective: "Four"},
		},
	}
	if err := invalid.Validate(); err == nil || err.Error() != "Four-week execution plans must include weeks 1 through 4 in order" {
		t.Fatalf("expected ordered week validation error, got %v", err)
	}

	report := RenderFourWeekExecutionReport(plan)
	for _, fragment := range []string{
		"# Four-Week Execution Plan",
		"- Plan: BIG-4701 4周执行计划与周目标",
		"- Overall progress: 2/8 goals complete (25%)",
		"- At-risk weeks: 2",
		"- Week 2: Build and integration progress=0/2 (0%)",
		"- w2-handoff-sync: Resolve orchestration and console handoff dependencies owner=orchestration-office status=at-risk",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in execution report, got %s", fragment, report)
		}
	}
}

func TestWeeklyExecutionPlanFlagsAtRiskGoalIDs(t *testing.T) {
	week := WeeklyExecutionPlan{
		WeekNumber: 2,
		Theme:      "Build and integration",
		Objective:  "Land high-risk integration work.",
		Goals: []WeeklyGoal{
			{GoalID: "w2-green", Title: "Green goal", Owner: "eng", Status: "on-track", SuccessMetric: "merged PRs", TargetValue: "2"},
			{GoalID: "w2-blocked", Title: "Blocked goal", Owner: "eng", Status: "blocked", SuccessMetric: "open blockers", TargetValue: "0"},
		},
	}
	if !reflect.DeepEqual(week.AtRiskGoalIDs(), []string{"w2-blocked"}) {
		t.Fatalf("unexpected at-risk goal ids: %+v", week.AtRiskGoalIDs())
	}
}

func TestBuildV3CandidateBacklogAndEntryGate(t *testing.T) {
	backlog := BuildV3CandidateBacklog()
	if backlog.EpicID != "BIG-EPIC-20" || backlog.Title != "v4.0 v3候选与进入条件" {
		t.Fatalf("unexpected backlog header: %+v", backlog)
	}
	ranked := backlog.RankedCandidates()
	if got := []string{ranked[0].CandidateID, ranked[1].CandidateID, ranked[2].CandidateID}; !reflect.DeepEqual(got, []string{"candidate-ops-hardening", "candidate-orchestration-rollout", "candidate-release-control"}) {
		t.Fatalf("unexpected ranked candidates: %+v", got)
	}
	for _, candidate := range backlog.Candidates {
		if !candidate.Ready() {
			t.Fatalf("expected ready candidate, got %+v", candidate)
		}
	}
	ops := backlog.Candidates[0]
	targets := make([]string, 0, len(ops.EvidenceLinks))
	for _, link := range ops.EvidenceLinks {
		targets = append(targets, link.Target)
	}
	if !containsAll(targets, []string{
		"src/bigclaw/operations.py",
		"tests/test_control_center.py",
		"tests/test_operations.py",
		"src/bigclaw/execution_contract.py",
		"src/bigclaw/workflow.py",
		"bigclaw-go/internal/workflow/engine_test.go",
		"bigclaw-go/internal/worker/runtime_test.go",
		"src/bigclaw/saved_views.py",
		"tests/test_saved_views.py",
		"src/bigclaw/evaluation.py",
		"tests/test_evaluation.py",
	}) {
		t.Fatalf("unexpected ops evidence links: %+v", targets)
	}

	gate := BuildV3EntryGate()
	decision := CandidatePlanner{}.EvaluateGate(backlog, gate, &governance.ScopeFreezeAudit{
		BoardName:  "BigClaw v2.0 Freeze",
		Version:    "v2.0",
		TotalItems: 25,
	})
	report := RenderCandidateBacklogReport(backlog, gate, decision)

	if !decision.Passed || !reflect.DeepEqual(decision.ReadyCandidateIDs, []string{"candidate-ops-hardening", "candidate-orchestration-rollout", "candidate-release-control"}) {
		t.Fatalf("unexpected gate decision: %+v", decision)
	}
	if len(decision.MissingCapabilities) != 0 || len(decision.MissingEvidence) != 0 {
		t.Fatalf("expected no missing requirements, got %+v", decision)
	}
	if !strings.Contains(report, "candidate-ops-hardening: Operations command-center hardening") ||
		!strings.Contains(report, "- command-center-src -> src/bigclaw/operations.py capability=ops-control") ||
		!strings.Contains(report, "- report-studio-tests -> tests/test_reports.py capability=commercialization") {
		t.Fatalf("unexpected built backlog report: %s", report)
	}
}
