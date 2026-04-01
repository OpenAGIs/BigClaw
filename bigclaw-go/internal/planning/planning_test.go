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
		Title:   "v4.0 v3 candidates and entry gate",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{
				CandidateID:       "candidate-release-control",
				Title:             "Release control center",
				Theme:             "console-governance",
				Priority:          "P0",
				Owner:             "platform-ui",
				Outcome:           "Unify console release gates and promotion evidence.",
				ValidationCommand: "cd bigclaw-go && go test ./internal/reporting",
				Capabilities:      []string{"release-gate", "reporting"},
				Evidence:          []string{"acceptance-suite", "validation-report"},
				EvidenceLinks: []EvidenceLink{
					{Label: "ui-acceptance", Target: "bigclaw-go/internal/reporting/reporting_test.go", Capability: "release-gate", Note: "role-permission and audit readiness coverage"},
				},
			},
		},
	}

	restored, err := roundTrip(backlog)
	if err != nil {
		t.Fatalf("round trip backlog: %v", err)
	}
	if !reflect.DeepEqual(restored, backlog) {
		t.Fatalf("restored backlog mismatch: %+v", restored)
	}
}

func TestCandidateBacklogRanksReadyItemsAheadOfBlockedWork(t *testing.T) {
	backlog := CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3 candidates and entry gate",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{CandidateID: "candidate-risky", Title: "Risky migration", Theme: "runtime", Priority: "P0", Owner: "runtime", Outcome: "Move execution runtime to the next rollout ring.", ValidationCommand: "cd bigclaw-go && go test ./internal/worker ./internal/scheduler", Capabilities: []string{"runtime-hardening"}, Evidence: []string{"benchmark"}, Blockers: []string{"missing rollback plan"}},
			{CandidateID: "candidate-ready", Title: "Release control center", Theme: "console-governance", Priority: "P1", Owner: "platform-ui", Outcome: "Unify console release gates and promotion evidence.", ValidationCommand: "cd bigclaw-go && go test ./internal/reporting", Capabilities: []string{"release-gate", "reporting"}, Evidence: []string{"acceptance-suite", "validation-report"}},
		},
	}

	ranked := backlog.RankedCandidates()
	if got := []string{ranked[0].CandidateID, ranked[1].CandidateID}; !reflect.DeepEqual(got, []string{"candidate-ready", "candidate-risky"}) {
		t.Fatalf("unexpected ranked ids: %v", got)
	}
}

func TestEntryGateEvaluationRequiresReadyCandidatesCapabilitiesAndEvidence(t *testing.T) {
	backlog := CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3 candidates and entry gate",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{CandidateID: "candidate-release-control", Title: "Release control center", Priority: "P0", Owner: "platform-ui", ValidationCommand: "cd bigclaw-go && go test ./internal/reporting", Capabilities: []string{"release-gate", "reporting"}, Evidence: []string{"acceptance-suite", "validation-report"}},
			{CandidateID: "candidate-ops-hardening", Title: "Ops hardening", Priority: "P0", Owner: "ops-platform", ValidationCommand: "cd bigclaw-go && go test ./internal/queue ./internal/reporting", Capabilities: []string{"ops-control"}, Evidence: []string{"weekly-review"}},
			{CandidateID: "candidate-orchestration", Title: "Orchestration rollout", Priority: "P1", Owner: "orchestration", ValidationCommand: "cd bigclaw-go && go test ./internal/workflow", Capabilities: []string{"commercialization", "handoff"}, Evidence: []string{"pilot-evidence"}},
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
	baseline := &governance.ScopeFreezeAudit{BoardName: "BigClaw v2.0 Freeze", Version: "v2.0", TotalItems: 5}

	decision := CandidatePlanner{}.EvaluateGate(backlog, gate, baseline)

	if !decision.Passed {
		t.Fatalf("expected gate decision to pass, got %+v", decision)
	}
	if !reflect.DeepEqual(asSet(decision.ReadyCandidateIDs), asSet([]string{"candidate-release-control", "candidate-ops-hardening", "candidate-orchestration"})) {
		t.Fatalf("unexpected ready ids: %v", decision.ReadyCandidateIDs)
	}
	if len(decision.MissingCapabilities) != 0 || len(decision.MissingEvidence) != 0 || !decision.BaselineReady || len(decision.BaselineFindings) != 0 {
		t.Fatalf("unexpected gate findings: %+v", decision)
	}
}

func TestEntryGateHoldsWhenBaselineIsMissingOrNotReady(t *testing.T) {
	backlog := CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3 candidates and entry gate",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{CandidateID: "candidate-release-control", Title: "Release control center", Priority: "P0", Owner: "platform-ui", Capabilities: []string{"release-gate"}, Evidence: []string{"acceptance-suite", "validation-report"}},
			{CandidateID: "candidate-ops-hardening", Title: "Ops hardening", Priority: "P0", Owner: "ops-platform", Capabilities: []string{"ops-control"}, Evidence: []string{"weekly-review"}},
			{CandidateID: "candidate-orchestration", Title: "Orchestration rollout", Priority: "P1", Owner: "orchestration", Capabilities: []string{"commercialization"}, Evidence: []string{"pilot-evidence"}},
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

	missing := CandidatePlanner{}.EvaluateGate(backlog, gate, nil)
	failed := CandidatePlanner{}.EvaluateGate(backlog, gate, &governance.ScopeFreezeAudit{
		BoardName:         "BigClaw v2.0 Freeze",
		Version:           "v2.0",
		TotalItems:        5,
		MissingValidation: []string{"OPE-116"},
	})

	if missing.Passed || missing.BaselineReady || !reflect.DeepEqual(missing.BaselineFindings, []string{"missing baseline audit for v2.0"}) {
		t.Fatalf("unexpected missing baseline decision: %+v", missing)
	}
	if failed.Passed || failed.BaselineReady || !reflect.DeepEqual(failed.BaselineFindings, []string{"baseline v2.0 is not release ready (87.5)"}) {
		t.Fatalf("unexpected failed baseline decision: %+v", failed)
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

	restored, err := roundTrip(decision)
	if err != nil {
		t.Fatalf("round trip decision: %v", err)
	}
	if !reflect.DeepEqual(restored, decision) {
		t.Fatalf("restored decision mismatch: %+v", restored)
	}
}

func TestRenderCandidateBacklogReportSummarizesBacklogAndGateFindings(t *testing.T) {
	backlog := CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3 candidates and entry gate",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{
				CandidateID:       "candidate-release-control",
				Title:             "Release control center",
				Priority:          "P0",
				Owner:             "platform-ui",
				ValidationCommand: "cd bigclaw-go && go test ./internal/reporting",
				Capabilities:      []string{"release-gate", "reporting"},
				Evidence:          []string{"acceptance-suite", "validation-report"},
				EvidenceLinks:     []EvidenceLink{{Label: "ui-acceptance", Target: "bigclaw-go/internal/reporting/reporting_test.go", Capability: "release-gate"}},
			},
		},
	}
	gate := EntryGate{
		GateID:                  "gate-v3-entry",
		Name:                    "V3 Entry Gate",
		MinReadyCandidates:      1,
		RequiredCapabilities:    []string{"release-gate"},
		RequiredEvidence:        []string{"validation-report"},
		RequiredBaselineVersion: "v2.0",
	}
	decision := CandidatePlanner{}.EvaluateGate(backlog, gate, &governance.ScopeFreezeAudit{BoardName: "BigClaw v2.0 Freeze", Version: "v2.0", TotalItems: 5})

	report := RenderCandidateBacklogReport(backlog, gate, decision)

	for _, fragment := range []string{
		"# V3 Candidate Backlog Report",
		"- Epic: BIG-EPIC-20 v4.0 v3 candidates and entry gate",
		"- Decision: PASS: ready=1 blocked=0 missing_capabilities=0 missing_evidence=0 baseline_findings=0",
		"- candidate-release-control: Release control center priority=P0 owner=platform-ui score=100 ready=true",
		"validation=cd bigclaw-go && go test ./internal/reporting",
		"- ui-acceptance -> bigclaw-go/internal/reporting/reporting_test.go capability=release-gate",
		"- Missing evidence: none",
		"- Baseline ready: true",
		"- Baseline findings: none",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func TestFourWeekExecutionPlanRoundTripPreservesWeeksAndGoals(t *testing.T) {
	plan := BuildBig4701ExecutionPlan()
	restored, err := roundTrip(plan)
	if err != nil {
		t.Fatalf("round trip plan: %v", err)
	}
	if !reflect.DeepEqual(restored, plan) {
		t.Fatalf("restored plan mismatch: %+v", restored)
	}
}

func TestFourWeekExecutionPlanRollsUpProgressAndAtRiskWeeks(t *testing.T) {
	plan := BuildBig4701ExecutionPlan()

	if plan.TotalGoals() != 8 || plan.CompletedGoals() != 2 || plan.OverallProgressPercent() != 25 {
		t.Fatalf("unexpected plan progress: %+v", plan)
	}
	if !reflect.DeepEqual(plan.AtRiskWeeks(), []int{2}) {
		t.Fatalf("unexpected at-risk weeks: %v", plan.AtRiskWeeks())
	}
	if !reflect.DeepEqual(plan.GoalStatusCounts(), map[string]int{"done": 2, "on-track": 1, "at-risk": 1, "not-started": 4}) {
		t.Fatalf("unexpected goal status counts: %+v", plan.GoalStatusCounts())
	}
}

func TestFourWeekExecutionPlanValidateRejectsMissingOrUnorderedWeeks(t *testing.T) {
	plan := FourWeekExecutionPlan{
		PlanID:    "BIG-4701",
		Title:     "Four-week execution plan and weekly goals",
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
		t.Fatalf("expected ordered-weeks validation error, got %v", err)
	}
}

func TestRenderFourWeekExecutionReportSummarizesPlanStatus(t *testing.T) {
	report := RenderFourWeekExecutionReport(BuildBig4701ExecutionPlan())

	for _, fragment := range []string{
		"# Four-Week Execution Plan",
		"- Plan: BIG-4701 Four-week execution plan and weekly goals",
		"- Overall progress: 2/8 goals complete (25%)",
		"- At-risk weeks: 2",
		"- Week 2: Build and integration progress=0/2 (0%)",
		"- w2-handoff-sync: Resolve orchestration and console handoff dependencies owner=orchestration-office status=at-risk",
	} {
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
		t.Fatalf("unexpected at-risk goal ids: %v", week.AtRiskGoalIDs())
	}
}

func TestBuildV3CandidateBacklogMatchesIssuePlanTraceability(t *testing.T) {
	backlog := BuildV3CandidateBacklog()

	if backlog.EpicID != "BIG-EPIC-20" || backlog.Title != "v4.0 v3 candidates and entry gate" {
		t.Fatalf("unexpected backlog metadata: %+v", backlog)
	}
	ranked := backlog.RankedCandidates()
	if got := []string{ranked[0].CandidateID, ranked[1].CandidateID, ranked[2].CandidateID}; !reflect.DeepEqual(got, []string{"candidate-ops-hardening", "candidate-orchestration-rollout", "candidate-release-control"}) {
		t.Fatalf("unexpected ranked ids: %v", got)
	}
	for _, candidate := range backlog.Candidates {
		if !candidate.Ready() {
			t.Fatalf("expected all built candidates to be ready, got %+v", candidate)
		}
	}
}

func TestBuildV3EntryGatePassesBuiltCandidateBacklogAgainstBaseline(t *testing.T) {
	backlog := BuildV3CandidateBacklog()
	gate := BuildV3EntryGate()

	decision := CandidatePlanner{}.EvaluateGate(backlog, gate, &governance.ScopeFreezeAudit{BoardName: "BigClaw v2.0 Freeze", Version: "v2.0", TotalItems: 25})
	report := RenderCandidateBacklogReport(backlog, gate, decision)

	if !decision.Passed {
		t.Fatalf("expected built backlog to pass entry gate, got %+v", decision)
	}
	if !reflect.DeepEqual(decision.ReadyCandidateIDs, []string{"candidate-ops-hardening", "candidate-orchestration-rollout", "candidate-release-control"}) {
		t.Fatalf("unexpected ready ids: %v", decision.ReadyCandidateIDs)
	}
	if len(decision.MissingCapabilities) != 0 || len(decision.MissingEvidence) != 0 {
		t.Fatalf("unexpected missing requirements: %+v", decision)
	}
	for _, fragment := range []string{
		"candidate-ops-hardening: Operations command-center hardening",
		"- command-center-src -> bigclaw-go/internal/reporting/reporting.go capability=ops-control",
		"- report-studio-tests -> bigclaw-go/internal/reporollout/rollout_test.go capability=commercialization",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("expected %q in report, got %s", fragment, report)
		}
	}
}

func asSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}
