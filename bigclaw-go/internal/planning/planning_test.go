package planning

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
			ValidationCommand: "cd bigclaw-go && go test ./internal/api ./internal/reporting",
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

	restored := MustJSONRoundTrip(backlog)
	if !reflect.DeepEqual(restored, backlog) {
		t.Fatalf("round trip mismatch: got=%+v want=%+v", restored, backlog)
	}
}

func TestCandidateBacklogRanksReadyItemsAheadOfBlockedWork(t *testing.T) {
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
				ValidationCommand: "cd bigclaw-go && go test ./internal/scheduler ./internal/worker ./internal/workflow",
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
				ValidationCommand: "cd bigclaw-go && go test ./internal/api ./internal/reporting",
				Capabilities:      []string{"release-gate", "reporting"},
				Evidence:          []string{"acceptance-suite", "validation-report"},
			},
		},
	}

	ranked := backlog.RankedCandidates()
	if got := []string{ranked[0].CandidateID, ranked[1].CandidateID}; !reflect.DeepEqual(got, []string{"candidate-ready", "candidate-risky"}) {
		t.Fatalf("unexpected ranked ids: %+v", got)
	}
}

func TestEntryGateEvaluationRequiresReadyCandidatesCapabilitiesAndEvidence(t *testing.T) {
	backlog := CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3候选与进入条件",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{CandidateID: "candidate-release-control", Title: "Release control center", Theme: "console-governance", Priority: "P0", Owner: "platform-ui", Outcome: "Unify console release gates and promotion evidence.", ValidationCommand: "cd bigclaw-go && go test ./internal/api ./internal/reporting", Capabilities: []string{"release-gate", "reporting"}, Evidence: []string{"acceptance-suite", "validation-report"}},
			{CandidateID: "candidate-ops-hardening", Title: "Ops hardening", Theme: "ops-command-center", Priority: "P0", Owner: "ops-platform", Outcome: "Package the command-center rollout with weekly review evidence.", ValidationCommand: "cd bigclaw-go && go test ./internal/product ./internal/reporting ./internal/workflow", Capabilities: []string{"ops-control"}, Evidence: []string{"weekly-review"}},
			{CandidateID: "candidate-orchestration", Title: "Orchestration rollout", Theme: "agent-orchestration", Priority: "P1", Owner: "orchestration", Outcome: "Promote cross-team orchestration with commercialization visibility.", ValidationCommand: "cd bigclaw-go && go test ./internal/reporting ./internal/scheduler ./internal/workflow", Capabilities: []string{"commercialization", "handoff"}, Evidence: []string{"pilot-evidence"}},
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
	baselineAudit := governance.ScopeFreezeAudit{BoardName: "BigClaw v2.0 Freeze", Version: "v2.0", TotalItems: 5}

	decision := CandidatePlanner{}.EvaluateGate(backlog, gate, &baselineAudit)
	if !decision.Passed {
		t.Fatalf("expected pass, got %+v", decision)
	}
	if got := map[string]bool{
		"candidate-release-control": contains(decision.ReadyCandidateIDs, "candidate-release-control"),
		"candidate-ops-hardening":   contains(decision.ReadyCandidateIDs, "candidate-ops-hardening"),
		"candidate-orchestration":   contains(decision.ReadyCandidateIDs, "candidate-orchestration"),
	}; !reflect.DeepEqual(got, map[string]bool{
		"candidate-release-control": true,
		"candidate-ops-hardening":   true,
		"candidate-orchestration":   true,
	}) {
		t.Fatalf("unexpected ready ids: %+v", decision.ReadyCandidateIDs)
	}
	if len(decision.MissingCapabilities) != 0 || len(decision.MissingEvidence) != 0 || !decision.BaselineReady || len(decision.BaselineFindings) != 0 {
		t.Fatalf("unexpected decision findings: %+v", decision)
	}
}

func TestEntryGateHoldsWhenV2BaselineIsMissingOrNotReady(t *testing.T) {
	backlog := CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3候选与进入条件",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{CandidateID: "candidate-release-control", Title: "Release control center", Theme: "console-governance", Priority: "P0", Owner: "platform-ui", Outcome: "Unify console release gates and promotion evidence.", ValidationCommand: "cd bigclaw-go && go test ./internal/api ./internal/reporting", Capabilities: []string{"release-gate"}, Evidence: []string{"acceptance-suite", "validation-report"}},
			{CandidateID: "candidate-ops-hardening", Title: "Ops hardening", Theme: "ops-command-center", Priority: "P0", Owner: "ops-platform", Outcome: "Package the command-center rollout with weekly review evidence.", ValidationCommand: "cd bigclaw-go && go test ./internal/product ./internal/reporting ./internal/workflow", Capabilities: []string{"ops-control"}, Evidence: []string{"weekly-review"}},
			{CandidateID: "candidate-orchestration", Title: "Orchestration rollout", Theme: "agent-orchestration", Priority: "P1", Owner: "orchestration", Outcome: "Promote cross-team orchestration with commercialization visibility.", ValidationCommand: "cd bigclaw-go && go test ./internal/reporting ./internal/scheduler ./internal/workflow", Capabilities: []string{"commercialization"}, Evidence: []string{"pilot-evidence"}},
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

	missingBaseline := CandidatePlanner{}.EvaluateGate(backlog, gate, nil)
	failedBaseline := CandidatePlanner{}.EvaluateGate(backlog, gate, &governance.ScopeFreezeAudit{
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

	restored := MustJSONRoundTrip(decision)
	if !reflect.DeepEqual(restored, decision) {
		t.Fatalf("round trip mismatch: got=%+v want=%+v", restored, decision)
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
			ValidationCommand: "cd bigclaw-go && go test ./internal/api ./internal/reporting",
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
	decision := CandidatePlanner{}.EvaluateGate(backlog, gate, &governance.ScopeFreezeAudit{
		BoardName:  "BigClaw v2.0 Freeze",
		Version:    "v2.0",
		TotalItems: 5,
	})

	report := RenderCandidateBacklogReport(backlog, gate, decision)
	for _, want := range []string{
		"# V3 Candidate Backlog Report",
		"- Epic: BIG-EPIC-20 v4.0 v3候选与进入条件",
		"- Decision: PASS: ready=1 blocked=0 missing_capabilities=0 missing_evidence=0 baseline_findings=0",
		"- candidate-release-control: Release control center priority=P0 owner=platform-ui score=100 ready=true",
		"validation=cd bigclaw-go && go test ./internal/api ./internal/reporting",
		"- ui-acceptance -> tests/test_design_system.py capability=release-gate",
		"- Missing evidence: none",
		"- Baseline ready: true",
		"- Baseline findings: none",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report:\n%s", want, report)
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
		ValidationCommand: "cd bigclaw-go && go test ./internal/product ./internal/reporting ./internal/workflow",
		Capabilities:      []string{"ops-control", "saved-views"},
		Evidence:          []string{"weekly-review", "validation-report"},
		EvidenceLinks: []EvidenceLink{
			{Label: "queue-control-center", Target: "src/bigclaw/operations.py", Capability: "ops-control", Note: "queue and approval command center"},
			{Label: "saved-view-report", Target: "src/bigclaw/saved_views.py", Capability: "saved-views", Note: "team saved views and digest evidence"},
		},
	}

	restored := MustJSONRoundTrip(candidate)
	if !reflect.DeepEqual(restored, candidate) {
		t.Fatalf("round trip mismatch: got=%+v want=%+v", restored, candidate)
	}
}

func TestFourWeekExecutionPlanRoundTripPreservesWeeksAndGoals(t *testing.T) {
	plan := BuildBIG4701ExecutionPlan()
	restored := MustJSONRoundTrip(plan)
	if !reflect.DeepEqual(restored, plan) {
		t.Fatalf("round trip mismatch: got=%+v want=%+v", restored, plan)
	}
}

func TestFourWeekExecutionPlanRollsUpProgressAndAtRiskWeeks(t *testing.T) {
	plan := BuildBIG4701ExecutionPlan()
	if plan.TotalGoals() != 8 || plan.CompletedGoals() != 2 || plan.OverallProgressPercent() != 25 {
		t.Fatalf("unexpected rollup: %+v", plan)
	}
	if !reflect.DeepEqual(plan.AtRiskWeeks(), []int{2}) {
		t.Fatalf("unexpected at-risk weeks: %+v", plan.AtRiskWeeks())
	}
	if got := plan.GoalStatusCounts(); !reflect.DeepEqual(got, map[string]int{"done": 2, "on-track": 1, "at-risk": 1, "not-started": 4}) {
		t.Fatalf("unexpected status counts: %+v", got)
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

	if err := plan.Validate(); err == nil || !strings.Contains(err.Error(), "Four-week execution plans must include weeks 1 through 4 in order") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestRenderFourWeekExecutionReportSummarizesPlanStatus(t *testing.T) {
	report := RenderFourWeekExecutionReport(BuildBIG4701ExecutionPlan())
	for _, want := range []string{
		"# Four-Week Execution Plan",
		"- Plan: BIG-4701 4周执行计划与周目标",
		"- Overall progress: 2/8 goals complete (25%)",
		"- At-risk weeks: 2",
		"- Week 2: Build and integration progress=0/2 (0%)",
		"- w2-handoff-sync: Resolve orchestration and console handoff dependencies owner=orchestration-office status=at-risk",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report:\n%s", want, report)
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
		t.Fatalf("unexpected at-risk ids: %+v", week.AtRiskGoalIDs())
	}
}

func TestBuildV3CandidateBacklogMatchesIssuePlanTraceability(t *testing.T) {
	backlog := BuildV3CandidateBacklog()
	if backlog.EpicID != "BIG-EPIC-20" || backlog.Title != "v4.0 v3候选与进入条件" {
		t.Fatalf("unexpected backlog header: %+v", backlog)
	}
	ranked := backlog.RankedCandidates()
	if got := []string{ranked[0].CandidateID, ranked[1].CandidateID, ranked[2].CandidateID}; !reflect.DeepEqual(got, []string{"candidate-ops-hardening", "candidate-orchestration-rollout", "candidate-release-control"}) {
		t.Fatalf("unexpected ranked order: %+v", got)
	}
	for _, candidate := range backlog.Candidates {
		if !candidate.Ready() {
			t.Fatalf("expected all candidates ready: %+v", backlog.Candidates)
		}
	}
	var opsCandidate CandidateEntry
	for _, candidate := range backlog.Candidates {
		if candidate.CandidateID == "candidate-ops-hardening" {
			opsCandidate = candidate
			break
		}
	}
	targets := map[string]bool{}
	for _, link := range opsCandidate.EvidenceLinks {
		targets[link.Target] = true
	}
	for _, want := range []string{
		"src/bigclaw/operations.py",
		"tests/test_control_center.py",
		"src/bigclaw/execution_contract.py",
		"src/bigclaw/workflow.py",
		"tests/test_workflow.py",
		"tests/test_execution_flow.py",
		"src/bigclaw/saved_views.py",
		"tests/test_saved_views.py",
		"src/bigclaw/evaluation.py",
		"tests/test_evaluation.py",
	} {
		if !targets[want] {
			t.Fatalf("missing evidence link target %q in %+v", want, opsCandidate.EvidenceLinks)
		}
	}
}

func TestBuildV3EntryGatePassesBuiltCandidateBacklogAgainstV2Baseline(t *testing.T) {
	backlog := BuildV3CandidateBacklog()
	gate := BuildV3EntryGate()
	decision := CandidatePlanner{}.EvaluateGate(backlog, gate, &governance.ScopeFreezeAudit{
		BoardName:  "BigClaw v2.0 Freeze",
		Version:    "v2.0",
		TotalItems: 25,
	})
	report := RenderCandidateBacklogReport(backlog, gate, decision)

	if !decision.Passed {
		t.Fatalf("expected pass, got %+v", decision)
	}
	if !reflect.DeepEqual(decision.ReadyCandidateIDs, []string{"candidate-ops-hardening", "candidate-orchestration-rollout", "candidate-release-control"}) {
		t.Fatalf("unexpected ready ids: %+v", decision.ReadyCandidateIDs)
	}
	if len(decision.MissingCapabilities) != 0 || len(decision.MissingEvidence) != 0 {
		t.Fatalf("unexpected missing findings: %+v", decision)
	}
	for _, want := range []string{
		"candidate-ops-hardening: Operations command-center hardening",
		"- command-center-src -> src/bigclaw/operations.py capability=ops-control",
		"- report-studio-tests -> tests/test_reports.py capability=commercialization",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report:\n%s", want, report)
		}
	}
}
