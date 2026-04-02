package planning

import (
	"encoding/json"
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
		Candidates: []CandidateEntry{
			{
				CandidateID:       "candidate-release-control",
				Title:             "Release control center",
				Theme:             "console-governance",
				Priority:          "P0",
				Owner:             "platform-ui",
				Outcome:           "Unify console release gates and promotion evidence.",
				ValidationCommand: "cd bigclaw-go && go test ./internal/designsystem ./internal/uireview ./internal/planning",
				Capabilities:      []string{"release-gate", "reporting"},
				Evidence:          []string{"acceptance-suite", "validation-report"},
				EvidenceLinks: []EvidenceLink{
					{
						Label:      "ui-acceptance",
						Target:     "bigclaw-go/internal/designsystem/designsystem_test.go",
						Capability: "release-gate",
						Note:       "role-permission and audit readiness coverage",
					},
				},
			},
		},
	}

	payload, err := json.Marshal(backlog)
	if err != nil {
		t.Fatalf("marshal backlog: %v", err)
	}
	var restored CandidateBacklog
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal backlog: %v", err)
	}
	if !reflect.DeepEqual(restored, backlog) {
		t.Fatalf("restored backlog mismatch: got %+v want %+v", restored, backlog)
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
				ValidationCommand: "cd bigclaw-go && go test ./internal/worker ./internal/scheduler",
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
				ValidationCommand: "cd bigclaw-go && go test ./internal/designsystem ./internal/uireview ./internal/planning",
				Capabilities:      []string{"release-gate", "reporting"},
				Evidence:          []string{"acceptance-suite", "validation-report"},
			},
		},
	}

	ranked := backlog.RankedCandidates()
	got := []string{ranked[0].CandidateID, ranked[1].CandidateID}
	want := []string{"candidate-ready", "candidate-risky"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ranked candidates mismatch: got %+v want %+v", got, want)
	}
}

func TestEntryGateEvaluationRequiresReadyCandidatesCapabilitiesAndEvidence(t *testing.T) {
	backlog := CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3候选与进入条件",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{
				CandidateID:       "candidate-release-control",
				Title:             "Release control center",
				Theme:             "console-governance",
				Priority:          "P0",
				Owner:             "platform-ui",
				Outcome:           "Unify console release gates and promotion evidence.",
				ValidationCommand: "cd bigclaw-go && go test ./internal/designsystem ./internal/uireview ./internal/planning",
				Capabilities:      []string{"release-gate", "reporting"},
				Evidence:          []string{"acceptance-suite", "validation-report"},
			},
			{
				CandidateID:       "candidate-ops-hardening",
				Title:             "Ops hardening",
				Theme:             "ops-command-center",
				Priority:          "P0",
				Owner:             "ops-platform",
				Outcome:           "Package the command-center rollout with weekly review evidence.",
				ValidationCommand: "python3 -m pytest tests/test_operations.py -q",
				Capabilities:      []string{"ops-control"},
				Evidence:          []string{"weekly-review"},
			},
			{
				CandidateID:       "candidate-orchestration",
				Title:             "Orchestration rollout",
				Theme:             "agent-orchestration",
				Priority:          "P1",
				Owner:             "orchestration",
				Outcome:           "Promote cross-team orchestration with commercialization visibility.",
				ValidationCommand: "cd bigclaw-go && go test ./internal/collaboration ./internal/pilot",
				Capabilities:      []string{"commercialization", "handoff"},
				Evidence:          []string{"pilot-evidence"},
			},
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
	baselineAudit := governance.ScopeFreezeAudit{
		BoardName:  "BigClaw v2.0 Freeze",
		Version:    "v2.0",
		TotalItems: 5,
	}

	decision := CandidatePlanner{}.EvaluateGate(backlog, gate, &baselineAudit)

	if !decision.Passed {
		t.Fatalf("expected gate to pass, got %+v", decision)
	}
	gotReady := append([]string(nil), decision.ReadyCandidateIDs...)
	for i := 0; i < len(gotReady); i++ {
		for j := i + 1; j < len(gotReady); j++ {
			if gotReady[j] < gotReady[i] {
				gotReady[i], gotReady[j] = gotReady[j], gotReady[i]
			}
		}
	}
	if want := []string{"candidate-ops-hardening", "candidate-orchestration", "candidate-release-control"}; !reflect.DeepEqual(gotReady, want) {
		t.Fatalf("ready candidate ids mismatch: got %+v want %+v", gotReady, want)
	}
	if len(decision.MissingCapabilities) != 0 || len(decision.MissingEvidence) != 0 {
		t.Fatalf("expected no missing gate inputs, got %+v", decision)
	}
	if !decision.BaselineReady || len(decision.BaselineFindings) != 0 {
		t.Fatalf("expected baseline ready, got %+v", decision)
	}
}

func TestEntryGateHoldsWhenV2BaselineIsMissingOrNotReady(t *testing.T) {
	backlog := CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3候选与进入条件",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{CandidateID: "candidate-release-control", Title: "Release control center", Theme: "console-governance", Priority: "P0", Owner: "platform-ui", Outcome: "Unify console release gates and promotion evidence.", ValidationCommand: "cd bigclaw-go && go test ./internal/designsystem ./internal/uireview ./internal/planning", Capabilities: []string{"release-gate"}, Evidence: []string{"acceptance-suite", "validation-report"}},
			{CandidateID: "candidate-ops-hardening", Title: "Ops hardening", Theme: "ops-command-center", Priority: "P0", Owner: "ops-platform", Outcome: "Package the command-center rollout with weekly review evidence.", ValidationCommand: "python3 -m pytest tests/test_operations.py -q", Capabilities: []string{"ops-control"}, Evidence: []string{"weekly-review"}},
			{CandidateID: "candidate-orchestration", Title: "Orchestration rollout", Theme: "agent-orchestration", Priority: "P1", Owner: "orchestration", Outcome: "Promote cross-team orchestration with commercialization visibility.", ValidationCommand: "cd bigclaw-go && go test ./internal/collaboration ./internal/pilot", Capabilities: []string{"commercialization"}, Evidence: []string{"pilot-evidence"}},
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

	if missingBaseline.Passed || missingBaseline.BaselineReady {
		t.Fatalf("expected missing baseline to hold, got %+v", missingBaseline)
	}
	if got, want := missingBaseline.BaselineFindings, []string{"missing baseline audit for v2.0"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("missing baseline findings mismatch: got %+v want %+v", got, want)
	}
	if failedBaseline.Passed || failedBaseline.BaselineReady {
		t.Fatalf("expected failed baseline to hold, got %+v", failedBaseline)
	}
	if got, want := failedBaseline.BaselineFindings, []string{"baseline v2.0 is not release ready (87.5)"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("failed baseline findings mismatch: got %+v want %+v", got, want)
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

	payload, err := json.Marshal(decision)
	if err != nil {
		t.Fatalf("marshal decision: %v", err)
	}
	var restored EntryGateDecision
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal decision: %v", err)
	}
	if !reflect.DeepEqual(restored, decision) {
		t.Fatalf("restored decision mismatch: got %+v want %+v", restored, decision)
	}
}

func TestRenderCandidateBacklogReportSummarizesBacklogAndGateFindings(t *testing.T) {
	backlog := CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3候选与进入条件",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{
				CandidateID:       "candidate-release-control",
				Title:             "Release control center",
				Theme:             "console-governance",
				Priority:          "P0",
				Owner:             "platform-ui",
				Outcome:           "Unify console release gates and promotion evidence.",
				ValidationCommand: "cd bigclaw-go && go test ./internal/designsystem ./internal/uireview ./internal/planning",
				Capabilities:      []string{"release-gate", "reporting"},
				Evidence:          []string{"acceptance-suite", "validation-report"},
				EvidenceLinks:     []EvidenceLink{{Label: "ui-acceptance", Target: "bigclaw-go/internal/designsystem/designsystem_test.go", Capability: "release-gate"}},
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
		"- candidate-release-control: Release control center priority=P0 owner=platform-ui score=100 ready=True",
		"validation=cd bigclaw-go && go test ./internal/designsystem ./internal/uireview ./internal/planning",
		"- ui-acceptance -> bigclaw-go/internal/designsystem/designsystem_test.go capability=release-gate",
		"- Missing evidence: none",
		"- Baseline ready: True",
		"- Baseline findings: none",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected report to contain %q, got %s", want, report)
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
		ValidationCommand: "python3 -m pytest tests/test_operations.py -q && (cd bigclaw-go && go test ./internal/product)",
		Capabilities:      []string{"ops-control", "saved-views"},
		Evidence:          []string{"weekly-review", "validation-report"},
		EvidenceLinks: []EvidenceLink{
			{Label: "queue-control-center", Target: "src/bigclaw/__init__.py", Capability: "ops-control", Note: "folded package queue and approval command center"},
			{Label: "saved-view-report", Target: "src/bigclaw/saved_views.py", Capability: "saved-views", Note: "team saved views and digest evidence"},
		},
	}

	payload, err := json.Marshal(candidate)
	if err != nil {
		t.Fatalf("marshal candidate: %v", err)
	}
	var restored CandidateEntry
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal candidate: %v", err)
	}
	if !reflect.DeepEqual(restored, candidate) {
		t.Fatalf("restored candidate mismatch: got %+v want %+v", restored, candidate)
	}
}

func TestFourWeekExecutionPlanRoundTripPreservesWeeksAndGoals(t *testing.T) {
	plan := BuildBig4701ExecutionPlan()

	payload, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("marshal plan: %v", err)
	}
	var restored FourWeekExecutionPlan
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal plan: %v", err)
	}
	if !reflect.DeepEqual(restored, plan) {
		t.Fatalf("restored plan mismatch: got %+v want %+v", restored, plan)
	}
}

func TestFourWeekExecutionPlanRollsUpProgressAndAtRiskWeeks(t *testing.T) {
	plan := BuildBig4701ExecutionPlan()

	if plan.TotalGoals() != 8 || plan.CompletedGoals() != 2 || plan.OverallProgressPercent() != 25 {
		t.Fatalf("unexpected plan rollup: %+v", plan)
	}
	if got, want := plan.AtRiskWeeks(), []int{2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("at-risk weeks mismatch: got %+v want %+v", got, want)
	}
	if got, want := plan.GoalStatusCounts(), map[string]int{"done": 2, "on-track": 1, "at-risk": 1, "not-started": 4}; !reflect.DeepEqual(got, want) {
		t.Fatalf("goal status counts mismatch: got %+v want %+v", got, want)
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

	err := plan.Validate()
	if err == nil || err.Error() != "Four-week execution plans must include weeks 1 through 4 in order" {
		t.Fatalf("expected validate error, got %v", err)
	}
}

func TestRenderFourWeekExecutionReportSummarizesPlanStatus(t *testing.T) {
	report := RenderFourWeekExecutionReport(BuildBig4701ExecutionPlan())

	for _, want := range []string{
		"# Four-Week Execution Plan",
		"- Plan: BIG-4701 4周执行计划与周目标",
		"- Overall progress: 2/8 goals complete (25%)",
		"- At-risk weeks: 2",
		"- Week 2: Build and integration progress=0/2 (0%)",
		"- w2-handoff-sync: Resolve orchestration and console handoff dependencies owner=orchestration-office status=at-risk",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected report to contain %q, got %s", want, report)
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

	if got, want := week.AtRiskGoalIDs(), []string{"w2-blocked"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("at-risk goal ids mismatch: got %+v want %+v", got, want)
	}
}

func TestBuildV3CandidateBacklogMatchesIssuePlanTraceability(t *testing.T) {
	backlog := BuildV3CandidateBacklog()

	if backlog.EpicID != "BIG-EPIC-20" || backlog.Title != "v4.0 v3候选与进入条件" {
		t.Fatalf("unexpected backlog identity: %+v", backlog)
	}
	ranked := backlog.RankedCandidates()
	gotIDs := []string{ranked[0].CandidateID, ranked[1].CandidateID, ranked[2].CandidateID}
	wantIDs := []string{"candidate-ops-hardening", "candidate-orchestration-rollout", "candidate-release-control"}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("ranked candidate ids mismatch: got %+v want %+v", gotIDs, wantIDs)
	}
	for _, candidate := range backlog.Candidates {
		if !candidate.Ready() {
			t.Fatalf("expected candidate ready, got %+v", candidate)
		}
	}

	var opsCandidate CandidateEntry
	var releaseCandidate CandidateEntry
	for _, candidate := range backlog.Candidates {
		if candidate.CandidateID == "candidate-ops-hardening" {
			opsCandidate = candidate
		}
		if candidate.CandidateID == "candidate-release-control" {
			releaseCandidate = candidate
		}
	}
	targets := map[string]struct{}{}
	for _, link := range opsCandidate.EvidenceLinks {
		targets[link.Target] = struct{}{}
	}
	for _, want := range []string{
		"src/bigclaw/__init__.py",
		"tests/test_operations.py",
		"src/bigclaw/execution_contract.py",
		"src/bigclaw/workflow.py",
		"bigclaw-go/internal/product/saved_views_test.go",
		"bigclaw-go/internal/workflow/engine_test.go",
		"bigclaw-go/internal/worker/runtime_test.go",
		"src/bigclaw/saved_views.py",
		"bigclaw-go/internal/evaluation/evaluation.go",
		"bigclaw-go/internal/evaluation/evaluation_test.go",
	} {
		if _, ok := targets[want]; !ok {
			t.Fatalf("missing ops evidence target %q in %+v", want, targets)
		}
	}

	if got, want := releaseCandidate.ValidationCommand, "cd bigclaw-go && go test ./internal/designsystem ./internal/uireview ./internal/planning"; got != want {
		t.Fatalf("release-control validation command mismatch: got %q want %q", got, want)
	}
	releaseTargets := map[string]struct{}{}
	for _, link := range releaseCandidate.EvidenceLinks {
		releaseTargets[link.Target] = struct{}{}
	}
	for _, want := range []string{
		"bigclaw-go/internal/designsystem/designsystem.go",
		"bigclaw-go/internal/designsystem/designsystem_test.go",
		"bigclaw-go/internal/uireview/uireview.go",
		"bigclaw-go/internal/uireview/render.go",
		"bigclaw-go/internal/uireview/uireview_test.go",
		"bigclaw-go/internal/planning/planning_test.go",
	} {
		if _, ok := releaseTargets[want]; !ok {
			t.Fatalf("missing Go-native release-control evidence target %q in %+v", want, releaseTargets)
		}
	}
	if _, ok := releaseTargets["src/bigclaw/design_system.py"]; ok {
		t.Fatalf("deleted Python design-system target still present in %+v", releaseTargets)
	}
	if _, ok := releaseTargets["src/bigclaw/console_ia.py"]; ok {
		t.Fatalf("deleted Python console IA target still present in %+v", releaseTargets)
	}
	if _, ok := releaseTargets["tests/test_ui_review.py"]; ok {
		t.Fatalf("deleted Python review pack target still present in %+v", releaseTargets)
	}
	if _, ok := releaseTargets["src/bigclaw/ui_review.py"]; ok {
		t.Fatalf("deleted Python review pack source target still present in %+v", releaseTargets)
	}
	if _, ok := releaseTargets["tests/test_design_system.py"]; ok {
		t.Fatalf("deleted Python design-system test target still present in %+v", releaseTargets)
	}
	if _, ok := releaseTargets["tests/test_console_ia.py"]; ok {
		t.Fatalf("deleted Python console IA test target still present in %+v", releaseTargets)
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
		t.Fatalf("expected gate to pass, got %+v", decision)
	}
	if got, want := decision.ReadyCandidateIDs, []string{"candidate-ops-hardening", "candidate-orchestration-rollout", "candidate-release-control"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("ready candidate ids mismatch: got %+v want %+v", got, want)
	}
	if len(decision.MissingCapabilities) != 0 || len(decision.MissingEvidence) != 0 {
		t.Fatalf("expected no missing gate inputs, got %+v", decision)
	}
	for _, want := range []string{
		"candidate-ops-hardening: Operations command-center hardening",
		"- command-center-src -> src/bigclaw/__init__.py capability=ops-control",
		"- report-studio-tests -> tests/test_reports.py capability=commercialization",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected report to contain %q, got %s", want, report)
		}
	}
}
