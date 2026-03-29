package planningparity

import (
	"reflect"
	"strings"
	"testing"

	"bigclaw-go/internal/governance"
)

func TestCandidateBacklogRoundTripPreservesManifestShape(t *testing.T) {
	t.Parallel()

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
			ValidationCommand: "python3 -m pytest tests/test_design_system.py -q",
			Capabilities:      []string{"release-gate", "reporting"},
			Evidence:          []string{"acceptance-suite", "validation-report"},
			EvidenceLinks: []EvidenceLink{{
				Label:      "ui-acceptance",
				Target:     "tests/test_design_system.py",
				Capability: "release-gate",
				Note:       "role-permission and audit readiness coverage",
			}},
		}},
	}

	data, err := backlog.ToMap()
	if err != nil {
		t.Fatalf("to map: %v", err)
	}
	restored, err := CandidateBacklogFromMap(data)
	if err != nil {
		t.Fatalf("from map: %v", err)
	}

	if !reflect.DeepEqual(restored, backlog) {
		t.Fatalf("restored backlog mismatch:\nwant: %#v\ngot:  %#v", backlog, restored)
	}
}

func TestCandidateBacklogRanksReadyItemsAheadOfBlockedWork(t *testing.T) {
	t.Parallel()

	backlog := CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3候选与进入条件",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{
				CandidateID:       "candidate-risky",
				Title:             "Risky migration",
				Theme:             "runtime",
				Priority:          "P0",
				Owner:             "runtime",
				Outcome:           "Move execution runtime to the next rollout ring.",
				ValidationCommand: "python3 -m pytest tests/test_runtime.py -q",
				Capabilities:      []string{"runtime-hardening"},
				Evidence:          []string{"benchmark"},
				Blockers:          []string{"missing rollback plan"},
			},
			{
				CandidateID:       "candidate-ready",
				Title:             "Release control center",
				Theme:             "console-governance",
				Priority:          "P1",
				Owner:             "platform-ui",
				Outcome:           "Unify console release gates and promotion evidence.",
				ValidationCommand: "python3 -m pytest tests/test_design_system.py -q",
				Capabilities:      []string{"release-gate", "reporting"},
				Evidence:          []string{"acceptance-suite", "validation-report"},
			},
		},
	}

	ranked := backlog.RankedCandidates()
	got := []string{ranked[0].CandidateID, ranked[1].CandidateID}
	want := []string{"candidate-ready", "candidate-risky"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ranked ids mismatch: want %v got %v", want, got)
	}
}

func TestEntryGateEvaluationRequiresReadyCandidatesCapabilitiesAndEvidence(t *testing.T) {
	t.Parallel()

	backlog := CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3候选与进入条件",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{CandidateID: "candidate-release-control", Title: "Release control center", Theme: "console-governance", Priority: "P0", Owner: "platform-ui", Outcome: "Unify console release gates and promotion evidence.", ValidationCommand: "python3 -m pytest tests/test_design_system.py -q", Capabilities: []string{"release-gate", "reporting"}, Evidence: []string{"acceptance-suite", "validation-report"}},
			{CandidateID: "candidate-ops-hardening", Title: "Ops hardening", Theme: "ops-command-center", Priority: "P0", Owner: "ops-platform", Outcome: "Package the command-center rollout with weekly review evidence.", ValidationCommand: "cd bigclaw-go && go test ./internal/reporting", Capabilities: []string{"ops-control"}, Evidence: []string{"weekly-review"}},
			{CandidateID: "candidate-orchestration", Title: "Orchestration rollout", Theme: "agent-orchestration", Priority: "P1", Owner: "orchestration", Outcome: "Promote cross-team orchestration with commercialization visibility.", ValidationCommand: "python3 -m pytest tests/test_orchestration.py -q", Capabilities: []string{"commercialization", "handoff"}, Evidence: []string{"pilot-evidence"}},
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
	decision := (CandidatePlanner{}).EvaluateGate(backlog, gate, &governance.ScopeFreezeAudit{
		BoardName:  "BigClaw v2.0 Freeze",
		Version:    "v2.0",
		TotalItems: 5,
	})

	if !decision.Passed {
		t.Fatalf("expected gate to pass: %+v", decision)
	}
	if got, want := set(decision.ReadyCandidateIDs), set([]string{"candidate-release-control", "candidate-ops-hardening", "candidate-orchestration"}); !reflect.DeepEqual(got, want) {
		t.Fatalf("ready candidates mismatch: want %v got %v", want, got)
	}
	if len(decision.MissingCapabilities) != 0 || len(decision.MissingEvidence) != 0 || !decision.BaselineReady || len(decision.BaselineFindings) != 0 {
		t.Fatalf("unexpected decision: %+v", decision)
	}
}

func TestEntryGateHoldsWhenV2BaselineIsMissingOrNotReady(t *testing.T) {
	t.Parallel()

	backlog := CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3候选与进入条件",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{CandidateID: "candidate-release-control", Title: "Release control center", Theme: "console-governance", Priority: "P0", Owner: "platform-ui", Outcome: "Unify console release gates and promotion evidence.", ValidationCommand: "python3 -m pytest tests/test_design_system.py -q", Capabilities: []string{"release-gate"}, Evidence: []string{"acceptance-suite", "validation-report"}},
			{CandidateID: "candidate-ops-hardening", Title: "Ops hardening", Theme: "ops-command-center", Priority: "P0", Owner: "ops-platform", Outcome: "Package the command-center rollout with weekly review evidence.", ValidationCommand: "cd bigclaw-go && go test ./internal/reporting", Capabilities: []string{"ops-control"}, Evidence: []string{"weekly-review"}},
			{CandidateID: "candidate-orchestration", Title: "Orchestration rollout", Theme: "agent-orchestration", Priority: "P1", Owner: "orchestration", Outcome: "Promote cross-team orchestration with commercialization visibility.", ValidationCommand: "python3 -m pytest tests/test_orchestration.py -q", Capabilities: []string{"commercialization"}, Evidence: []string{"pilot-evidence"}},
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

	missingBaseline := (CandidatePlanner{}).EvaluateGate(backlog, gate, nil)
	failedBaseline := (CandidatePlanner{}).EvaluateGate(backlog, gate, &governance.ScopeFreezeAudit{
		BoardName:         "BigClaw v2.0 Freeze",
		Version:           "v2.0",
		TotalItems:        5,
		MissingValidation: []string{"OPE-116"},
	})

	if missingBaseline.Passed || missingBaseline.BaselineReady || !reflect.DeepEqual(missingBaseline.BaselineFindings, []string{"missing baseline audit for v2.0"}) {
		t.Fatalf("unexpected missing baseline decision: %+v", missingBaseline)
	}
	if failedBaseline.Passed || failedBaseline.BaselineReady || !reflect.DeepEqual(failedBaseline.BaselineFindings, []string{"baseline v2.0 is not release ready (87.5)"}) {
		t.Fatalf("unexpected failed baseline decision: %+v", failedBaseline)
	}
}

func TestEntryGateDecisionRoundTripPreservesFindings(t *testing.T) {
	t.Parallel()

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

	data, err := decision.ToMap()
	if err != nil {
		t.Fatalf("to map: %v", err)
	}
	restored, err := EntryGateDecisionFromMap(data)
	if err != nil {
		t.Fatalf("from map: %v", err)
	}

	if !reflect.DeepEqual(restored, decision) {
		t.Fatalf("restored decision mismatch:\nwant: %#v\ngot:  %#v", decision, restored)
	}
}

func TestRenderCandidateBacklogReportSummarizesBacklogAndGateFindings(t *testing.T) {
	t.Parallel()

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
			ValidationCommand: "python3 -m pytest tests/test_design_system.py -q",
			Capabilities:      []string{"release-gate", "reporting"},
			Evidence:          []string{"acceptance-suite", "validation-report"},
			EvidenceLinks: []EvidenceLink{{
				Label:      "ui-acceptance",
				Target:     "tests/test_design_system.py",
				Capability: "release-gate",
			}},
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
	decision := (CandidatePlanner{}).EvaluateGate(backlog, gate, &governance.ScopeFreezeAudit{
		BoardName:  "BigClaw v2.0 Freeze",
		Version:    "v2.0",
		TotalItems: 5,
	})

	report := RenderCandidateBacklogReport(backlog, gate, decision)
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
			t.Fatalf("expected %q in report, got:\n%s", fragment, report)
		}
	}
}

func TestCandidateEntryRoundTripPreservesEvidenceLinks(t *testing.T) {
	t.Parallel()

	candidate := CandidateEntry{
		CandidateID:       "candidate-ops-hardening",
		Title:             "Ops hardening",
		Theme:             "ops-command-center",
		Priority:          "P0",
		Owner:             "ops-platform",
		Outcome:           "Package command-center and approval surfaces with linked evidence.",
		ValidationCommand: "cd bigclaw-go && go test ./internal/reporting ./internal/product",
		Capabilities:      []string{"ops-control", "saved-views"},
		Evidence:          []string{"weekly-review", "validation-report"},
		EvidenceLinks: []EvidenceLink{
			{Label: "queue-control-center", Target: "src/bigclaw/operations.py", Capability: "ops-control", Note: "queue and approval command center"},
			{Label: "saved-view-report", Target: "src/bigclaw/saved_views.py", Capability: "saved-views", Note: "team saved views and digest evidence"},
		},
	}

	data, err := candidate.ToMap()
	if err != nil {
		t.Fatalf("to map: %v", err)
	}
	restored, err := CandidateEntryFromMap(data)
	if err != nil {
		t.Fatalf("from map: %v", err)
	}

	if !reflect.DeepEqual(restored, candidate) {
		t.Fatalf("restored candidate mismatch:\nwant: %#v\ngot:  %#v", candidate, restored)
	}
}

func TestFourWeekExecutionPlanRoundTripPreservesWeeksAndGoals(t *testing.T) {
	t.Parallel()

	plan := BuildBig4701ExecutionPlan()

	data, err := plan.ToMap()
	if err != nil {
		t.Fatalf("to map: %v", err)
	}
	restored, err := FourWeekExecutionPlanFromMap(data)
	if err != nil {
		t.Fatalf("from map: %v", err)
	}

	if !reflect.DeepEqual(restored, plan) {
		t.Fatalf("restored plan mismatch:\nwant: %#v\ngot:  %#v", plan, restored)
	}
}

func TestFourWeekExecutionPlanRollsUpProgressAndAtRiskWeeks(t *testing.T) {
	t.Parallel()

	plan := BuildBig4701ExecutionPlan()

	if plan.TotalGoals() != 8 || plan.CompletedGoals() != 2 || plan.OverallProgressPercent() != 25 {
		t.Fatalf("unexpected progress rollup: total=%d completed=%d percent=%d", plan.TotalGoals(), plan.CompletedGoals(), plan.OverallProgressPercent())
	}
	if !reflect.DeepEqual(plan.AtRiskWeeks(), []int{2}) {
		t.Fatalf("unexpected at-risk weeks: %v", plan.AtRiskWeeks())
	}
	want := map[string]int{"done": 2, "on-track": 1, "at-risk": 1, "not-started": 4}
	if !reflect.DeepEqual(plan.GoalStatusCounts(), want) {
		t.Fatalf("status counts mismatch: want %v got %v", want, plan.GoalStatusCounts())
	}
}

func TestFourWeekExecutionPlanValidateRejectsMissingOrUnorderedWeeks(t *testing.T) {
	t.Parallel()

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

	err := plan.Validate()
	if err == nil || err.Error() != "Four-week execution plans must include weeks 1 through 4 in order" {
		t.Fatalf("unexpected validate error: %v", err)
	}
}

func TestRenderFourWeekExecutionReportSummarizesPlanStatus(t *testing.T) {
	t.Parallel()

	report, err := RenderFourWeekExecutionReport(BuildBig4701ExecutionPlan())
	if err != nil {
		t.Fatalf("render report: %v", err)
	}

	for _, fragment := range []string{
		"# Four-Week Execution Plan",
		"- Plan: BIG-4701 4周执行计划与周目标",
		"- Overall progress: 2/8 goals complete (25%)",
		"- At-risk weeks: 2",
		"- Week 2: Build and integration progress=0/2 (0%)",
		"- w2-handoff-sync: Resolve orchestration and console handoff dependencies owner=orchestration-office status=at-risk",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got:\n%s", fragment, report)
		}
	}
}

func TestWeeklyExecutionPlanFlagsAtRiskGoalIDs(t *testing.T) {
	t.Parallel()

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
		t.Fatalf("unexpected at-risk goal ids: %v", week.AtRiskGoalIDs())
	}
}

func TestBuildV3CandidateBacklogMatchesIssuePlanTraceability(t *testing.T) {
	t.Parallel()

	backlog := BuildV3CandidateBacklog()

	if backlog.EpicID != "BIG-EPIC-20" || backlog.Title != "v4.0 v3候选与进入条件" {
		t.Fatalf("unexpected backlog metadata: %+v", backlog)
	}
	ranked := backlog.RankedCandidates()
	gotIDs := []string{ranked[0].CandidateID, ranked[1].CandidateID, ranked[2].CandidateID}
	wantIDs := []string{"candidate-ops-hardening", "candidate-orchestration-rollout", "candidate-release-control"}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("ranked candidate ids mismatch: want %v got %v", wantIDs, gotIDs)
	}
	for _, candidate := range backlog.Candidates {
		if !candidate.Ready() {
			t.Fatalf("expected candidate to be ready: %+v", candidate)
		}
	}

	var opsCandidate CandidateEntry
	for _, candidate := range backlog.Candidates {
		if candidate.CandidateID == "candidate-ops-hardening" {
			opsCandidate = candidate
			break
		}
	}
	targets := map[string]struct{}{}
	for _, link := range opsCandidate.EvidenceLinks {
		targets[link.Target] = struct{}{}
	}
	for _, want := range []string{
		"src/bigclaw/operations.py",
		"bigclaw-go/internal/controlcenterparity/controlcenterparity_test.go",
		"bigclaw-go/internal/reporting/reporting_test.go",
		"src/bigclaw/execution_contract.py",
		"src/bigclaw/workflow.py",
		"bigclaw-go/internal/workflow/engine_test.go",
		"bigclaw-go/internal/executionparity/executionparity_test.go",
		"src/bigclaw/saved_views.py",
		"bigclaw-go/internal/product/saved_views_test.go",
		"src/bigclaw/evaluation.py",
		"bigclaw-go/internal/evaluationparity/evaluationparity_test.go",
	} {
		if _, ok := targets[want]; !ok {
			t.Fatalf("missing evidence link target %q in ops candidate", want)
		}
	}
}

func TestBuildV3EntryGatePassesBuiltCandidateBacklogAgainstV2Baseline(t *testing.T) {
	t.Parallel()

	backlog := BuildV3CandidateBacklog()
	gate := BuildV3EntryGate()
	decision := (CandidatePlanner{}).EvaluateGate(backlog, gate, &governance.ScopeFreezeAudit{
		BoardName:  "BigClaw v2.0 Freeze",
		Version:    "v2.0",
		TotalItems: 25,
	})
	report := RenderCandidateBacklogReport(backlog, gate, decision)

	if !decision.Passed {
		t.Fatalf("expected decision to pass: %+v", decision)
	}
	wantIDs := []string{"candidate-ops-hardening", "candidate-orchestration-rollout", "candidate-release-control"}
	if !reflect.DeepEqual(decision.ReadyCandidateIDs, wantIDs) {
		t.Fatalf("ready candidate ids mismatch: want %v got %v", wantIDs, decision.ReadyCandidateIDs)
	}
	if len(decision.MissingCapabilities) != 0 || len(decision.MissingEvidence) != 0 {
		t.Fatalf("unexpected decision gaps: %+v", decision)
	}
	for _, fragment := range []string{
		"candidate-ops-hardening: Operations command-center hardening",
		"- command-center-src -> src/bigclaw/operations.py capability=ops-control",
		"- report-studio-tests -> tests/test_reports.py capability=commercialization",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got:\n%s", fragment, report)
		}
	}
}

func set(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}
