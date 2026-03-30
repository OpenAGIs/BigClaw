package planningsurface

import (
	"reflect"
	"strings"
	"testing"

	"bigclaw-go/internal/governance"
)

func TestCandidateBacklogRoundTripPreservesManifestShape(t *testing.T) {
	backlog := CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3候选与进入条件",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{{
			CandidateID:       "candidate-release-control",
			Title:             "Release control center",
			Theme:             "console-governance",
			Priority:          "P0",
			Owner:             "platform-ui",
			Outcome:           "Unify console release gates and promotion evidence.",
			ValidationCommand: "cd bigclaw-go && go test ./internal/product",
			Capabilities:      []string{"release-gate", "reporting"},
			Evidence:          []string{"acceptance-suite", "validation-report"},
			EvidenceLinks: []EvidenceLink{{
				Label: "ui-acceptance", Target: "bigclaw-go/internal/product/console_test.go", Capability: "release-gate", Note: "repo-native design-system and audit readiness coverage",
			}},
		}},
	}
	var restored CandidateBacklog
	if err := MarshalRoundTrip(backlog, &restored); err != nil {
		t.Fatalf("round trip backlog: %v", err)
	}
	if !reflect.DeepEqual(restored, backlog) {
		t.Fatalf("unexpected restored backlog: %+v", restored)
	}
}

func TestCandidateBacklogRanksReadyItemsAheadOfBlockedWork(t *testing.T) {
	backlog := CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3候选与进入条件",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{CandidateID: "candidate-risky", Title: "Risky migration", Theme: "runtime", Priority: "P0", Owner: "runtime", Outcome: "Move execution runtime to the next rollout ring.", ValidationCommand: "cd bigclaw-go && go test ./internal/worker ./internal/scheduler", Capabilities: []string{"runtime-hardening"}, Evidence: []string{"benchmark"}, Blockers: []string{"missing rollback plan"}},
			{CandidateID: "candidate-ready", Title: "Release control center", Theme: "console-governance", Priority: "P1", Owner: "platform-ui", Outcome: "Unify console release gates and promotion evidence.", ValidationCommand: "cd bigclaw-go && go test ./internal/product", Capabilities: []string{"release-gate", "reporting"}, Evidence: []string{"acceptance-suite", "validation-report"}},
		},
	}
	ranked := backlog.RankedCandidates()
	if ranked[0].CandidateID != "candidate-ready" || ranked[1].CandidateID != "candidate-risky" {
		t.Fatalf("unexpected ranking: %+v", ranked)
	}
}

func TestEntryGateEvaluationRequiresReadyCandidatesCapabilitiesAndEvidence(t *testing.T) {
	backlog := CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3候选与进入条件",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{CandidateID: "candidate-release-control", Title: "Release control center", Theme: "console-governance", Priority: "P0", Owner: "platform-ui", Outcome: "Unify console release gates and promotion evidence.", ValidationCommand: "cd bigclaw-go && go test ./internal/product", Capabilities: []string{"release-gate", "reporting"}, Evidence: []string{"acceptance-suite", "validation-report"}},
			{CandidateID: "candidate-ops-hardening", Title: "Ops hardening", Theme: "ops-command-center", Priority: "P0", Owner: "ops-platform", Outcome: "Package the command-center rollout with weekly review evidence.", ValidationCommand: "cd bigclaw-go && go test ./internal/reporting", Capabilities: []string{"ops-control"}, Evidence: []string{"weekly-review"}},
			{CandidateID: "candidate-orchestration", Title: "Orchestration rollout", Theme: "agent-orchestration", Priority: "P1", Owner: "orchestration", Outcome: "Promote cross-team orchestration with commercialization visibility.", ValidationCommand: "python3 -m pytest tests/test_orchestration.py -q", Capabilities: []string{"commercialization", "handoff"}, Evidence: []string{"pilot-evidence"}},
		},
	}
	gate := EntryGate{GateID: "gate-v3-entry", Name: "V3 Entry Gate", MinReadyCandidates: 3, RequiredCapabilities: []string{"release-gate", "ops-control", "commercialization"}, RequiredEvidence: []string{"acceptance-suite", "pilot-evidence", "validation-report"}, RequiredBaselineVersion: "v2.0"}
	baseline := &governance.ScopeFreezeAudit{BoardName: "BigClaw v2.0 Freeze", Version: "v2.0", TotalItems: 5}
	decision := (CandidatePlanner{}).EvaluateGate(backlog, gate, baseline)
	if !decision.Passed || !reflect.DeepEqual(decision.MissingCapabilities, []string{}) || !reflect.DeepEqual(decision.MissingEvidence, []string{}) || !decision.BaselineReady || !reflect.DeepEqual(decision.ReadyCandidateIDs, []string{"candidate-ops-hardening", "candidate-release-control", "candidate-orchestration"}) {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestEntryGateHoldsWhenV2BaselineIsMissingOrNotReady(t *testing.T) {
	backlog := CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3候选与进入条件",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{CandidateID: "candidate-release-control", Title: "Release control center", Theme: "console-governance", Priority: "P0", Owner: "platform-ui", Outcome: "Unify console release gates and promotion evidence.", ValidationCommand: "cd bigclaw-go && go test ./internal/product", Capabilities: []string{"release-gate"}, Evidence: []string{"acceptance-suite", "validation-report"}},
			{CandidateID: "candidate-ops-hardening", Title: "Ops hardening", Theme: "ops-command-center", Priority: "P0", Owner: "ops-platform", Outcome: "Package the command-center rollout with weekly review evidence.", ValidationCommand: "cd bigclaw-go && go test ./internal/reporting", Capabilities: []string{"ops-control"}, Evidence: []string{"weekly-review"}},
			{CandidateID: "candidate-orchestration", Title: "Orchestration rollout", Theme: "agent-orchestration", Priority: "P1", Owner: "orchestration", Outcome: "Promote cross-team orchestration with commercialization visibility.", ValidationCommand: "python3 -m pytest tests/test_orchestration.py -q", Capabilities: []string{"commercialization"}, Evidence: []string{"pilot-evidence"}},
		},
	}
	gate := EntryGate{GateID: "gate-v3-entry", Name: "V3 Entry Gate", MinReadyCandidates: 3, RequiredCapabilities: []string{"release-gate", "ops-control", "commercialization"}, RequiredEvidence: []string{"acceptance-suite", "pilot-evidence", "validation-report"}, RequiredBaselineVersion: "v2.0"}
	missingBaseline := (CandidatePlanner{}).EvaluateGate(backlog, gate, nil)
	failedBaseline := (CandidatePlanner{}).EvaluateGate(backlog, gate, &governance.ScopeFreezeAudit{BoardName: "BigClaw v2.0 Freeze", Version: "v2.0", TotalItems: 5, MissingValidation: []string{"OPE-116"}})
	if missingBaseline.Passed || missingBaseline.BaselineReady || !reflect.DeepEqual(missingBaseline.BaselineFindings, []string{"missing baseline audit for v2.0"}) {
		t.Fatalf("unexpected missing-baseline decision: %+v", missingBaseline)
	}
	if failedBaseline.Passed || failedBaseline.BaselineReady || !reflect.DeepEqual(failedBaseline.BaselineFindings, []string{"baseline v2.0 is not release ready (87.5)"}) {
		t.Fatalf("unexpected failed-baseline decision: %+v", failedBaseline)
	}
}

func TestEntryGateDecisionRoundTripPreservesFindings(t *testing.T) {
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
	var restored EntryGateDecision
	if err := MarshalRoundTrip(decision, &restored); err != nil {
		t.Fatalf("round trip decision: %v", err)
	}
	if !reflect.DeepEqual(restored, decision) {
		t.Fatalf("unexpected restored decision: %+v", restored)
	}
}

func TestRenderCandidateBacklogReportSummarizesBacklogAndGateFindings(t *testing.T) {
	backlog := CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3候选与进入条件",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{{
			CandidateID:       "candidate-release-control",
			Title:             "Release control center",
			Theme:             "console-governance",
			Priority:          "P0",
			Owner:             "platform-ui",
			Outcome:           "Unify console release gates and promotion evidence.",
			ValidationCommand: "cd bigclaw-go && go test ./internal/product",
			Capabilities:      []string{"release-gate", "reporting"},
			Evidence:          []string{"acceptance-suite", "validation-report"},
			EvidenceLinks:     []EvidenceLink{{Label: "ui-acceptance", Target: "bigclaw-go/internal/product/console_test.go", Capability: "release-gate"}},
		}},
	}
	gate := EntryGate{GateID: "gate-v3-entry", Name: "V3 Entry Gate", MinReadyCandidates: 1, RequiredCapabilities: []string{"release-gate"}, RequiredEvidence: []string{"validation-report"}, RequiredBaselineVersion: "v2.0"}
	decision := (CandidatePlanner{}).EvaluateGate(backlog, gate, &governance.ScopeFreezeAudit{BoardName: "BigClaw v2.0 Freeze", Version: "v2.0", TotalItems: 5})
	report := RenderCandidateBacklogReport(backlog, gate, decision)
	for _, fragment := range []string{"# V3 Candidate Backlog Report", "- Epic: BIG-EPIC-20 v4.0 v3候选与进入条件", "- Decision: PASS: ready=1 blocked=0 missing_capabilities=0 missing_evidence=0 baseline_findings=0", "- candidate-release-control: Release control center priority=P0 owner=platform-ui score=100 ready=true", "validation=cd bigclaw-go && go test ./internal/product", "- ui-acceptance -> bigclaw-go/internal/product/console_test.go capability=release-gate", "- Missing evidence: none", "- Baseline ready: true", "- Baseline findings: none"} {
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
		ValidationCommand: "cd bigclaw-go && go test ./internal/reporting",
		Capabilities:      []string{"ops-control", "saved-views"},
		Evidence:          []string{"weekly-review", "validation-report"},
		EvidenceLinks: []EvidenceLink{
			{Label: "queue-control-center", Target: "src/bigclaw/operations.py", Capability: "ops-control", Note: "queue and approval command center"},
			{Label: "saved-view-report", Target: "src/bigclaw/saved_views.py", Capability: "saved-views", Note: "team saved views and digest evidence"},
		},
	}
	var restored CandidateEntry
	if err := MarshalRoundTrip(candidate, &restored); err != nil {
		t.Fatalf("round trip candidate: %v", err)
	}
	if !reflect.DeepEqual(restored, candidate) {
		t.Fatalf("unexpected restored candidate: %+v", restored)
	}
}

func TestFourWeekExecutionPlanRoundTripPreservesWeeksAndGoals(t *testing.T) {
	plan := BuildBig4701ExecutionPlan()
	var restored FourWeekExecutionPlan
	if err := MarshalRoundTrip(plan, &restored); err != nil {
		t.Fatalf("round trip plan: %v", err)
	}
	if !reflect.DeepEqual(restored, plan) {
		t.Fatalf("unexpected restored plan: %+v", restored)
	}
}

func TestFourWeekExecutionPlanRollsUpProgressAndAtRiskWeeks(t *testing.T) {
	plan := BuildBig4701ExecutionPlan()
	if plan.TotalGoals() != 8 || plan.CompletedGoals() != 2 || plan.OverallProgressPercent() != 25 || !reflect.DeepEqual(plan.AtRiskWeeks(), []int{2}) {
		t.Fatalf("unexpected plan rollup: %+v", plan)
	}
	if !reflect.DeepEqual(plan.GoalStatusCounts(), map[string]int{"done": 2, "on-track": 1, "at-risk": 1, "not-started": 4}) {
		t.Fatalf("unexpected status counts: %+v", plan.GoalStatusCounts())
	}
}

func TestFourWeekExecutionPlanValidateRejectsMissingOrUnorderedWeeks(t *testing.T) {
	plan := FourWeekExecutionPlan{
		PlanID:    "BIG-4701",
		Title:     "4周执行计划与周目标",
		Owner:     "execution-office",
		StartDate: "2026-03-11",
		Weeks: []WeeklyExecutionPlan{
			{WeekNumber: 1, Theme: "One", Objective: "One"},
			{WeekNumber: 3, Theme: "Three", Objective: "Three"},
			{WeekNumber: 2, Theme: "Two", Objective: "Two"},
			{WeekNumber: 4, Theme: "Four", Objective: "Four"},
		},
	}
	if err := plan.Validate(); err == nil || err.Error() != "Four-week execution plans must include weeks 1 through 4 in order" {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestRenderFourWeekExecutionReportSummarizesPlanStatus(t *testing.T) {
	report, err := RenderFourWeekExecutionReport(BuildBig4701ExecutionPlan())
	if err != nil {
		t.Fatalf("render report: %v", err)
	}
	for _, fragment := range []string{"# Four-Week Execution Plan", "- Plan: BIG-4701 4周执行计划与周目标", "- Overall progress: 2/8 goals complete (25%)", "- At-risk weeks: 2", "- Week 2: Build and integration progress=0/2 (0%)", "- w2-handoff-sync: Resolve orchestration and console handoff dependencies owner=orchestration-office status=at-risk"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
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
		t.Fatalf("unexpected at-risk goals: %+v", week.AtRiskGoalIDs())
	}
}

func TestBuildV3CandidateBacklogMatchesIssuePlanTraceability(t *testing.T) {
	backlog := BuildV3CandidateBacklog()
	ranked := backlog.RankedCandidates()
	gotIDs := []string{ranked[0].CandidateID, ranked[1].CandidateID, ranked[2].CandidateID}
	if backlog.EpicID != "BIG-EPIC-20" || backlog.Title != "v4.0 v3候选与进入条件" || !reflect.DeepEqual(gotIDs, []string{"candidate-ops-hardening", "candidate-orchestration-rollout", "candidate-release-control"}) {
		t.Fatalf("unexpected backlog: %+v", backlog)
	}
	for _, candidate := range backlog.Candidates {
		if !candidate.Ready() {
			t.Fatalf("expected ready candidate, got %+v", candidate)
		}
	}
	var opsCandidate CandidateEntry
	for _, candidate := range backlog.Candidates {
		if candidate.CandidateID == "candidate-ops-hardening" {
			opsCandidate = candidate
		}
	}
	targets := map[string]struct{}{}
	for _, link := range opsCandidate.EvidenceLinks {
		targets[link.Target] = struct{}{}
	}
	for _, target := range []string{
		"src/bigclaw/operations.py",
		"tests/test_control_center.py",
		"bigclaw-go/internal/reporting/reporting_test.go",
		"src/bigclaw/execution_contract.py",
		"src/bigclaw/workflow.py",
		"bigclaw-go/internal/workflow/engine_test.go",
		"bigclaw-go/internal/worker/runtime_test.go",
		"src/bigclaw/saved_views.py",
		"tests/test_saved_views.py",
		"src/bigclaw/evaluation.py",
		"tests/test_evaluation.py",
	} {
		if _, ok := targets[target]; !ok {
			t.Fatalf("expected target %q in %+v", target, targets)
		}
	}
}

func TestBuildV3EntryGatePassesBuiltCandidateBacklogAgainstV2Baseline(t *testing.T) {
	backlog := BuildV3CandidateBacklog()
	gate := BuildV3EntryGate()
	decision := (CandidatePlanner{}).EvaluateGate(backlog, gate, &governance.ScopeFreezeAudit{BoardName: "BigClaw v2.0 Freeze", Version: "v2.0", TotalItems: 25})
	report := RenderCandidateBacklogReport(backlog, gate, decision)
	if !decision.Passed || !reflect.DeepEqual(decision.ReadyCandidateIDs, []string{"candidate-ops-hardening", "candidate-orchestration-rollout", "candidate-release-control"}) || len(decision.MissingCapabilities) != 0 || len(decision.MissingEvidence) != 0 {
		t.Fatalf("unexpected decision: %+v", decision)
	}
	for _, fragment := range []string{"candidate-ops-hardening: Operations command-center hardening", "- command-center-src -> src/bigclaw/operations.py capability=ops-control", "- report-studio-tests -> bigclaw-go/internal/reporting/reporting_test.go capability=commercialization"} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}
