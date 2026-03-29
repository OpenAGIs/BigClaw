package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLane8PythonPlanningContractStaysAligned(t *testing.T) {
	goRepoRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))
	scriptPath := filepath.Join(t.TempDir(), "planning_contract.py")
	script := `import json
import sys
from pathlib import Path

repo_root = Path(sys.argv[1])
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.planning import (
    FourWeekExecutionPlan,
    CandidateBacklog,
    CandidateEntry,
    CandidatePlanner,
    EvidenceLink,
    EntryGate,
    EntryGateDecision,
    WeeklyExecutionPlan,
    WeeklyGoal,
    build_big_4701_execution_plan,
    build_v3_candidate_backlog,
    build_v3_entry_gate,
    render_candidate_backlog_report,
    render_four_week_execution_report,
)
from bigclaw.governance import ScopeFreezeAudit

backlog = CandidateBacklog(
    epic_id="BIG-EPIC-20",
    title="v4.0 v3候选与进入条件",
    version="v4.0-v3",
    candidates=[
        CandidateEntry(
            candidate_id="candidate-release-control",
            title="Release control center",
            theme="console-governance",
            priority="P0",
            owner="platform-ui",
            outcome="Unify console release gates and promotion evidence.",
            validation_command="python3 -m pytest tests/test_design_system.py -q",
            capabilities=["release-gate", "reporting"],
            evidence=["acceptance-suite", "validation-report"],
            evidence_links=[
                EvidenceLink(
                    label="ui-acceptance",
                    target="tests/test_design_system.py",
                    capability="release-gate",
                    note="role-permission and audit readiness coverage",
                )
            ],
        )
    ],
)
round_trip_backlog = {"equal": CandidateBacklog.from_dict(backlog.to_dict()) == backlog}

backlog = CandidateBacklog(
    epic_id="BIG-EPIC-20",
    title="v4.0 v3候选与进入条件",
    version="v4.0-v3",
    candidates=[
        CandidateEntry(
            candidate_id="candidate-risky",
            title="Risky migration",
            theme="runtime",
            priority="P0",
            owner="runtime",
            outcome="Move execution runtime to the next rollout ring.",
            validation_command="python3 -m pytest tests/test_runtime.py -q",
            capabilities=["runtime-hardening"],
            evidence=["benchmark"],
            blockers=["missing rollback plan"],
        ),
        CandidateEntry(
            candidate_id="candidate-ready",
            title="Release control center",
            theme="console-governance",
            priority="P1",
            owner="platform-ui",
            outcome="Unify console release gates and promotion evidence.",
            validation_command="python3 -m pytest tests/test_design_system.py -q",
            capabilities=["release-gate", "reporting"],
            evidence=["acceptance-suite", "validation-report"],
        ),
    ],
)
ranking = {"ranked_ids": [candidate.candidate_id for candidate in backlog.ranked_candidates]}

backlog = CandidateBacklog(
    epic_id="BIG-EPIC-20",
    title="v4.0 v3候选与进入条件",
    version="v4.0-v3",
    candidates=[
        CandidateEntry(
            candidate_id="candidate-release-control",
            title="Release control center",
            theme="console-governance",
            priority="P0",
            owner="platform-ui",
            outcome="Unify console release gates and promotion evidence.",
            validation_command="python3 -m pytest tests/test_design_system.py -q",
            capabilities=["release-gate", "reporting"],
            evidence=["acceptance-suite", "validation-report"],
        ),
        CandidateEntry(
            candidate_id="candidate-ops-hardening",
            title="Ops hardening",
            theme="ops-command-center",
            priority="P0",
            owner="ops-platform",
            outcome="Package the command-center rollout with weekly review evidence.",
            validation_command="python3 -m pytest tests/test_operations.py -q",
            capabilities=["ops-control"],
            evidence=["weekly-review"],
        ),
        CandidateEntry(
            candidate_id="candidate-orchestration",
            title="Orchestration rollout",
            theme="agent-orchestration",
            priority="P1",
            owner="orchestration",
            outcome="Promote cross-team orchestration with commercialization visibility.",
            validation_command="python3 -m pytest tests/test_orchestration.py -q",
            capabilities=["commercialization", "handoff"],
            evidence=["pilot-evidence"],
        ),
    ],
)
gate = EntryGate(
    gate_id="gate-v3-entry",
    name="V3 Entry Gate",
    min_ready_candidates=3,
    required_capabilities=["release-gate", "ops-control", "commercialization"],
    required_evidence=["acceptance-suite", "pilot-evidence", "validation-report"],
    required_baseline_version="v2.0",
)
baseline_audit = ScopeFreezeAudit(
    board_name="BigClaw v2.0 Freeze",
    version="v2.0",
    total_items=5,
)
decision = CandidatePlanner().evaluate_gate(backlog, gate, baseline_audit=baseline_audit)
entry_gate = {
    "passed": decision.passed,
    "ready_candidate_ids": decision.ready_candidate_ids,
    "missing_capabilities": decision.missing_capabilities,
    "missing_evidence": decision.missing_evidence,
    "baseline_ready": decision.baseline_ready,
    "baseline_findings": decision.baseline_findings,
}

missing_baseline = CandidatePlanner().evaluate_gate(backlog, gate)
failed_baseline = CandidatePlanner().evaluate_gate(
    backlog,
    gate,
    baseline_audit=ScopeFreezeAudit(
        board_name="BigClaw v2.0 Freeze",
        version="v2.0",
        total_items=5,
        missing_validation=["OPE-116"],
    ),
)
baseline_hold = {
    "missing_passed": missing_baseline.passed,
    "missing_ready": missing_baseline.baseline_ready,
    "missing_findings": missing_baseline.baseline_findings,
    "failed_passed": failed_baseline.passed,
    "failed_ready": failed_baseline.baseline_ready,
    "failed_findings": failed_baseline.baseline_findings,
}

decision = EntryGateDecision(
    gate_id="gate-v3-entry",
    passed=False,
    ready_candidate_ids=["candidate-release-control"],
    blocked_candidate_ids=["candidate-runtime"],
    missing_capabilities=["commercialization"],
    missing_evidence=["pilot-evidence"],
    baseline_ready=False,
    baseline_findings=["baseline v2.0 is not release ready (87.5)"],
    blocker_count=1,
)
decision_round_trip = {"equal": EntryGateDecision.from_dict(decision.to_dict()) == decision}

backlog = CandidateBacklog(
    epic_id="BIG-EPIC-20",
    title="v4.0 v3候选与进入条件",
    version="v4.0-v3",
    candidates=[
        CandidateEntry(
            candidate_id="candidate-release-control",
            title="Release control center",
            theme="console-governance",
            priority="P0",
            owner="platform-ui",
            outcome="Unify console release gates and promotion evidence.",
            validation_command="python3 -m pytest tests/test_design_system.py -q",
            capabilities=["release-gate", "reporting"],
            evidence=["acceptance-suite", "validation-report"],
            evidence_links=[
                EvidenceLink(
                    label="ui-acceptance",
                    target="tests/test_design_system.py",
                    capability="release-gate",
                )
            ],
        )
    ],
)
gate = EntryGate(
    gate_id="gate-v3-entry",
    name="V3 Entry Gate",
    min_ready_candidates=1,
    required_capabilities=["release-gate"],
    required_evidence=["validation-report"],
    required_baseline_version="v2.0",
)
decision = CandidatePlanner().evaluate_gate(
    backlog,
    gate,
    baseline_audit=ScopeFreezeAudit(
        board_name="BigClaw v2.0 Freeze",
        version="v2.0",
        total_items=5,
    ),
)
report = render_candidate_backlog_report(backlog, gate, decision)
backlog_report = {
    "has_title": "# V3 Candidate Backlog Report" in report,
    "has_epic": "- Epic: BIG-EPIC-20 v4.0 v3候选与进入条件" in report,
    "has_decision": "- Decision: PASS: ready=1 blocked=0 missing_capabilities=0 missing_evidence=0 baseline_findings=0" in report,
    "has_candidate": "- candidate-release-control: Release control center priority=P0 owner=platform-ui score=100 ready=True" in report,
    "has_validation": "validation=python3 -m pytest tests/test_design_system.py -q" in report,
    "has_link": "- ui-acceptance -> tests/test_design_system.py capability=release-gate" in report,
    "has_missing_evidence": "- Missing evidence: none" in report,
    "has_baseline_ready": "- Baseline ready: True" in report,
    "has_baseline_findings": "- Baseline findings: none" in report,
}

candidate = CandidateEntry(
    candidate_id="candidate-ops-hardening",
    title="Ops hardening",
    theme="ops-command-center",
    priority="P0",
    owner="ops-platform",
    outcome="Package command-center and approval surfaces with linked evidence.",
    validation_command="python3 -m pytest tests/test_operations.py tests/test_saved_views.py -q",
    capabilities=["ops-control", "saved-views"],
    evidence=["weekly-review", "validation-report"],
    evidence_links=[
        EvidenceLink(
            label="queue-control-center",
            target="src/bigclaw/operations.py",
            capability="ops-control",
            note="queue and approval command center",
        ),
        EvidenceLink(
            label="saved-view-report",
            target="src/bigclaw/saved_views.py",
            capability="saved-views",
            note="team saved views and digest evidence",
        ),
    ],
)
candidate_round_trip = {"equal": CandidateEntry.from_dict(candidate.to_dict()) == candidate}

plan = build_big_4701_execution_plan()
four_week_round_trip = {"equal": FourWeekExecutionPlan.from_dict(plan.to_dict()) == plan}
four_week_rollup = {
    "total_goals": plan.total_goals,
    "completed_goals": plan.completed_goals,
    "overall_progress_percent": plan.overall_progress_percent,
    "at_risk_weeks": plan.at_risk_weeks,
    "goal_status_counts": plan.goal_status_counts(),
}

plan = FourWeekExecutionPlan(
    plan_id="BIG-4701",
    title="4周执行计划与周目标",
    owner="execution-office",
    start_date="2026-03-11",
    weeks=[
        WeeklyExecutionPlan(week_number=1, theme="One", objective="One"),
        WeeklyExecutionPlan(week_number=3, theme="Three", objective="Three"),
        WeeklyExecutionPlan(week_number=2, theme="Two", objective="Two"),
        WeeklyExecutionPlan(week_number=4, theme="Four", objective="Four"),
    ],
)
validation_error = ""
try:
    plan.validate()
except ValueError as exc:
    validation_error = str(exc)

report = render_four_week_execution_report(build_big_4701_execution_plan())
four_week_report = {
    "has_title": "# Four-Week Execution Plan" in report,
    "has_plan": "- Plan: BIG-4701 4周执行计划与周目标" in report,
    "has_progress": "- Overall progress: 2/8 goals complete (25%)" in report,
    "has_at_risk_weeks": "- At-risk weeks: 2" in report,
    "has_week_two": "- Week 2: Build and integration progress=0/2 (0%)" in report,
    "has_goal": "- w2-handoff-sync: Resolve orchestration and console handoff dependencies owner=orchestration-office status=at-risk" in report,
}

week = WeeklyExecutionPlan(
    week_number=2,
    theme="Build and integration",
    objective="Land high-risk integration work.",
    goals=[
        WeeklyGoal(
            goal_id="w2-green",
            title="Green goal",
            owner="eng",
            status="on-track",
            success_metric="merged PRs",
            target_value="2",
        ),
        WeeklyGoal(
            goal_id="w2-blocked",
            title="Blocked goal",
            owner="eng",
            status="blocked",
            success_metric="open blockers",
            target_value="0",
        ),
    ],
)
weekly = {"at_risk_goal_ids": week.at_risk_goal_ids}

backlog = build_v3_candidate_backlog()
ops_candidate = next(candidate for candidate in backlog.candidates if candidate.candidate_id == "candidate-ops-hardening")
traceability = {
    "epic_id": backlog.epic_id,
    "title": backlog.title,
    "ranked_ids": [candidate.candidate_id for candidate in backlog.ranked_candidates],
    "all_ready": all(candidate.ready for candidate in backlog.candidates),
    "ops_targets": sorted(link.target for link in ops_candidate.evidence_links),
}

gate = build_v3_entry_gate()
decision = CandidatePlanner().evaluate_gate(
    backlog,
    gate,
    baseline_audit=ScopeFreezeAudit(
        board_name="BigClaw v2.0 Freeze",
        version="v2.0",
        total_items=25,
    ),
)
report = render_candidate_backlog_report(backlog, gate, decision)
built = {
    "passed": decision.passed,
    "ready_candidate_ids": decision.ready_candidate_ids,
    "missing_capabilities": decision.missing_capabilities,
    "missing_evidence": decision.missing_evidence,
    "has_ops_candidate": "candidate-ops-hardening: Operations command-center hardening" in report,
    "has_command_center_link": "- command-center-src -> src/bigclaw/operations.py capability=ops-control" in report,
    "has_report_studio_link": "- report-studio-tests -> tests/test_reports.py capability=commercialization" in report,
}

print(json.dumps({
    "round_trip_backlog": round_trip_backlog,
    "ranking": ranking,
    "entry_gate": entry_gate,
    "baseline_hold": baseline_hold,
    "decision_round_trip": decision_round_trip,
    "backlog_report": backlog_report,
    "candidate_round_trip": candidate_round_trip,
    "four_week_round_trip": four_week_round_trip,
    "four_week_rollup": four_week_rollup,
    "validation_error": validation_error,
    "four_week_report": four_week_report,
    "weekly": weekly,
    "traceability": traceability,
    "built": built,
}))
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write planning contract script: %v", err)
	}

	cmd := exec.Command("python3", scriptPath, repoRoot)
	cmd.Dir = goRepoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run planning contract script: %v\n%s", err, string(output))
	}

	var decoded struct {
		RoundTripBacklog struct{ Equal bool `json:"equal"` } `json:"round_trip_backlog"`
		Ranking          struct {
			RankedIDs []string `json:"ranked_ids"`
		} `json:"ranking"`
		EntryGate struct {
			Passed            bool     `json:"passed"`
			ReadyCandidateIDs []string `json:"ready_candidate_ids"`
			MissingCapabilities []string `json:"missing_capabilities"`
			MissingEvidence   []string `json:"missing_evidence"`
			BaselineReady     bool     `json:"baseline_ready"`
			BaselineFindings  []string `json:"baseline_findings"`
		} `json:"entry_gate"`
		BaselineHold struct {
			MissingPassed  bool     `json:"missing_passed"`
			MissingReady   bool     `json:"missing_ready"`
			MissingFindings []string `json:"missing_findings"`
			FailedPassed   bool     `json:"failed_passed"`
			FailedReady    bool     `json:"failed_ready"`
			FailedFindings []string `json:"failed_findings"`
		} `json:"baseline_hold"`
		DecisionRoundTrip struct{ Equal bool `json:"equal"` } `json:"decision_round_trip"`
		BacklogReport struct {
			HasTitle            bool `json:"has_title"`
			HasEpic             bool `json:"has_epic"`
			HasDecision         bool `json:"has_decision"`
			HasCandidate        bool `json:"has_candidate"`
			HasValidation       bool `json:"has_validation"`
			HasLink             bool `json:"has_link"`
			HasMissingEvidence  bool `json:"has_missing_evidence"`
			HasBaselineReady    bool `json:"has_baseline_ready"`
			HasBaselineFindings bool `json:"has_baseline_findings"`
		} `json:"backlog_report"`
		CandidateRoundTrip struct{ Equal bool `json:"equal"` } `json:"candidate_round_trip"`
		FourWeekRoundTrip  struct{ Equal bool `json:"equal"` } `json:"four_week_round_trip"`
		FourWeekRollup     struct {
			TotalGoals             int            `json:"total_goals"`
			CompletedGoals         int            `json:"completed_goals"`
			OverallProgressPercent int            `json:"overall_progress_percent"`
			AtRiskWeeks            []int          `json:"at_risk_weeks"`
			GoalStatusCounts       map[string]int `json:"goal_status_counts"`
		} `json:"four_week_rollup"`
		ValidationError string `json:"validation_error"`
		FourWeekReport  struct {
			HasTitle       bool `json:"has_title"`
			HasPlan        bool `json:"has_plan"`
			HasProgress    bool `json:"has_progress"`
			HasAtRiskWeeks bool `json:"has_at_risk_weeks"`
			HasWeekTwo     bool `json:"has_week_two"`
			HasGoal        bool `json:"has_goal"`
		} `json:"four_week_report"`
		Weekly struct {
			AtRiskGoalIDs []string `json:"at_risk_goal_ids"`
		} `json:"weekly"`
		Traceability struct {
			EpicID    string   `json:"epic_id"`
			Title     string   `json:"title"`
			RankedIDs []string `json:"ranked_ids"`
			AllReady  bool     `json:"all_ready"`
			OpsTargets []string `json:"ops_targets"`
		} `json:"traceability"`
		Built struct {
			Passed              bool     `json:"passed"`
			ReadyCandidateIDs   []string `json:"ready_candidate_ids"`
			MissingCapabilities []string `json:"missing_capabilities"`
			MissingEvidence     []string `json:"missing_evidence"`
			HasOpsCandidate     bool     `json:"has_ops_candidate"`
			HasCommandCenterLink bool    `json:"has_command_center_link"`
			HasReportStudioLink bool     `json:"has_report_studio_link"`
		} `json:"built"`
	}
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("decode planning contract output: %v\n%s", err, string(output))
	}

	if !decoded.RoundTripBacklog.Equal {
		t.Fatalf("unexpected backlog round-trip payload: %+v", decoded.RoundTripBacklog)
	}
	if len(decoded.Ranking.RankedIDs) != 2 || decoded.Ranking.RankedIDs[0] != "candidate-ready" || decoded.Ranking.RankedIDs[1] != "candidate-risky" {
		t.Fatalf("unexpected ranking payload: %+v", decoded.Ranking)
	}
	if !decoded.EntryGate.Passed || len(decoded.EntryGate.ReadyCandidateIDs) != 3 || len(decoded.EntryGate.MissingCapabilities) != 0 || len(decoded.EntryGate.MissingEvidence) != 0 || !decoded.EntryGate.BaselineReady || len(decoded.EntryGate.BaselineFindings) != 0 {
		t.Fatalf("unexpected entry gate payload: %+v", decoded.EntryGate)
	}
	if decoded.BaselineHold.MissingPassed || decoded.BaselineHold.MissingReady || len(decoded.BaselineHold.MissingFindings) != 1 || decoded.BaselineHold.MissingFindings[0] != "missing baseline audit for v2.0" || decoded.BaselineHold.FailedPassed || decoded.BaselineHold.FailedReady || len(decoded.BaselineHold.FailedFindings) != 1 || decoded.BaselineHold.FailedFindings[0] != "baseline v2.0 is not release ready (87.5)" {
		t.Fatalf("unexpected baseline-hold payload: %+v", decoded.BaselineHold)
	}
	if !decoded.DecisionRoundTrip.Equal || !decoded.CandidateRoundTrip.Equal || !decoded.FourWeekRoundTrip.Equal {
		t.Fatalf("unexpected round-trip payloads: decision=%+v candidate=%+v fourWeek=%+v", decoded.DecisionRoundTrip, decoded.CandidateRoundTrip, decoded.FourWeekRoundTrip)
	}
	if !decoded.BacklogReport.HasTitle || !decoded.BacklogReport.HasEpic || !decoded.BacklogReport.HasDecision || !decoded.BacklogReport.HasCandidate || !decoded.BacklogReport.HasValidation || !decoded.BacklogReport.HasLink || !decoded.BacklogReport.HasMissingEvidence || !decoded.BacklogReport.HasBaselineReady || !decoded.BacklogReport.HasBaselineFindings {
		t.Fatalf("unexpected backlog report payload: %+v", decoded.BacklogReport)
	}
	if decoded.FourWeekRollup.TotalGoals != 8 || decoded.FourWeekRollup.CompletedGoals != 2 || decoded.FourWeekRollup.OverallProgressPercent != 25 || len(decoded.FourWeekRollup.AtRiskWeeks) != 1 || decoded.FourWeekRollup.AtRiskWeeks[0] != 2 || decoded.FourWeekRollup.GoalStatusCounts["done"] != 2 || decoded.FourWeekRollup.GoalStatusCounts["on-track"] != 1 || decoded.FourWeekRollup.GoalStatusCounts["at-risk"] != 1 || decoded.FourWeekRollup.GoalStatusCounts["not-started"] != 4 {
		t.Fatalf("unexpected four-week rollup payload: %+v", decoded.FourWeekRollup)
	}
	if decoded.ValidationError != "Four-week execution plans must include weeks 1 through 4 in order" {
		t.Fatalf("unexpected validation error payload: %q", decoded.ValidationError)
	}
	if !decoded.FourWeekReport.HasTitle || !decoded.FourWeekReport.HasPlan || !decoded.FourWeekReport.HasProgress || !decoded.FourWeekReport.HasAtRiskWeeks || !decoded.FourWeekReport.HasWeekTwo || !decoded.FourWeekReport.HasGoal {
		t.Fatalf("unexpected four-week report payload: %+v", decoded.FourWeekReport)
	}
	if len(decoded.Weekly.AtRiskGoalIDs) != 1 || decoded.Weekly.AtRiskGoalIDs[0] != "w2-blocked" {
		t.Fatalf("unexpected weekly at-risk payload: %+v", decoded.Weekly)
	}
	if decoded.Traceability.EpicID != "BIG-EPIC-20" || decoded.Traceability.Title != "v4.0 v3候选与进入条件" || len(decoded.Traceability.RankedIDs) != 3 || decoded.Traceability.RankedIDs[0] != "candidate-ops-hardening" || decoded.Traceability.RankedIDs[1] != "candidate-orchestration-rollout" || decoded.Traceability.RankedIDs[2] != "candidate-release-control" || !decoded.Traceability.AllReady {
		t.Fatalf("unexpected traceability payload: %+v", decoded.Traceability)
	}
	requiredTargets := map[string]bool{
		"src/bigclaw/operations.py":    false,
		"tests/test_control_center.py": false,
		"tests/test_operations.py":     false,
		"src/bigclaw/execution_contract.py": false,
		"src/bigclaw/workflow.py":      false,
		"tests/test_workflow.py":       false,
		"tests/test_execution_flow.py": false,
		"src/bigclaw/saved_views.py":   false,
		"tests/test_saved_views.py":    false,
		"src/bigclaw/evaluation.py":    false,
		"tests/test_evaluation.py":     false,
	}
	for _, target := range decoded.Traceability.OpsTargets {
		if _, ok := requiredTargets[target]; ok {
			requiredTargets[target] = true
		}
	}
	for target, present := range requiredTargets {
		if !present {
			t.Fatalf("missing traceability target %s in %+v", target, decoded.Traceability.OpsTargets)
		}
	}
	if !decoded.Built.Passed || len(decoded.Built.ReadyCandidateIDs) != 3 || len(decoded.Built.MissingCapabilities) != 0 || len(decoded.Built.MissingEvidence) != 0 || !decoded.Built.HasOpsCandidate || !decoded.Built.HasCommandCenterLink || !decoded.Built.HasReportStudioLink {
		t.Fatalf("unexpected built backlog payload: %+v", decoded.Built)
	}
}
