package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLane8PythonWorkflowContractStaysAligned(t *testing.T) {
	goRepoRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))
	scriptPath := filepath.Join(t.TempDir(), "workflow_contract.py")
	script := `import json
import tempfile
import sys
from pathlib import Path

repo_root = Path(sys.argv[1])
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.models import Priority, RiskLevel, Task
from bigclaw.observability import GitSyncTelemetry, ObservabilityLedger, PullRequestFreshness, RepoSyncAudit
from bigclaw.reports import PilotMetric, PilotScorecard
from bigclaw.workflow import AcceptanceGate, WorkflowEngine, WorkpadJournal

journal = WorkpadJournal(task_id="BIG-402-replay", run_id="run-journal-1")
journal.record("intake", "recorded", source="local")
journal.record("execution", "approved", medium="docker")

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    journal_path = journal.write(str(td / "journals" / "run-journal-1.json"))
    loaded = WorkpadJournal.read(journal_path)
    replay = {
        "task_id": loaded.task_id,
        "run_id": loaded.run_id,
        "replay": loaded.replay(),
    }

with tempfile.TemporaryDirectory() as td:
    ledger = ObservabilityLedger(str(Path(td) / "ledger.json"))
    task = Task(
        task_id="BIG-403",
        source="linear",
        title="Close acceptance gate",
        description="need validation evidence",
        priority=Priority.P0,
        acceptance_criteria=["report-shared"],
        validation_plan=["pytest"],
    )
    execution = WorkflowEngine().scheduler.execute(task, run_id="run-gate-1", ledger=ledger)
    decision = AcceptanceGate().evaluate(task, execution, validation_evidence=["pytest"])
    missing = {
        "passed": decision.passed,
        "status": decision.status,
        "missing_acceptance": decision.missing_acceptance_criteria,
        "missing_validation": decision.missing_validation_steps,
    }

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    ledger = ObservabilityLedger(str(td / "ledger.json"))
    task = Task(
        task_id="BIG-402",
        source="linear",
        title="Record workflow journal",
        description="capture execution closure",
        priority=Priority.P0,
        acceptance_criteria=["report-shared"],
        validation_plan=["pytest"],
        required_tools=["browser"],
    )
    result = WorkflowEngine().run(
        task,
        run_id="run-wf-1",
        ledger=ledger,
        report_path=str(td / "reports" / "run-wf-1.md"),
        journal_path=str(td / "journals" / "run-wf-1.json"),
        validation_evidence=["pytest", "report-shared"],
        orchestration_report_path=str(td / "reports" / "run-wf-1-orchestration.md"),
        git_push_succeeded=True,
        git_push_output="main -> origin/main",
        git_log_stat_output="commit 123abc\n 3 files changed, 12 insertions(+)",
    )
    loaded = json.loads(Path(result.journal_path).read_text())
    accepted = {
        "medium": result.execution.decision.medium,
        "passed": result.acceptance.passed,
        "status": result.acceptance.status,
        "has_journal_path": result.journal_path is not None,
        "has_orchestration_report_path": result.orchestration_report_path is not None,
        "steps": [entry["step"] for entry in loaded["entries"]],
        "acceptance_step_status": loaded["entries"][-2]["status"],
        "closeout_step_status": loaded["entries"][-1]["status"],
        "git_push_succeeded": ledger.load()[0]["closeout"]["git_push_succeeded"],
    }

with tempfile.TemporaryDirectory() as td:
    ledger = ObservabilityLedger(str(Path(td) / "ledger.json"))
    task = Task(
        task_id="BIG-403-risk",
        source="linear",
        title="Approve prod change",
        description="manual gate",
        risk_level=RiskLevel.HIGH,
        acceptance_criteria=["rollback-plan"],
        validation_plan=["integration-test"],
    )
    result = WorkflowEngine().run(
        task,
        run_id="run-wf-2",
        ledger=ledger,
        validation_evidence=["rollback-plan", "integration-test"],
    )
    high_risk = {
        "run_status": result.execution.run.status,
        "acceptance_passed": result.acceptance.passed,
        "acceptance_status": result.acceptance.status,
    }

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    ledger = ObservabilityLedger(str(td / "ledger.json"))
    task = Task(
        task_id="OPE-60",
        source="linear",
        title="Pilot closeout",
        description="capture KPI and ROI evidence",
        priority=Priority.P0,
        acceptance_criteria=["pilot-scorecard", "report-shared"],
        validation_plan=["pytest"],
    )
    scorecard = PilotScorecard(
        issue_id="OPE-60",
        customer="Design Partner A",
        period="2026-Q2",
        metrics=[
            PilotMetric(name="Automation coverage", baseline=30, current=81, target=80, unit="%"),
            PilotMetric(name="Review cycle time", baseline=10, current=4, target=5, unit="h", higher_is_better=False),
        ],
        monthly_benefit=15000,
        monthly_cost=3000,
        implementation_cost=18000,
        benchmark_score=98,
        benchmark_passed=True,
    )
    result = WorkflowEngine().run(
        task,
        run_id="run-wf-pilot-1",
        ledger=ledger,
        journal_path=str(td / "journals" / "run-wf-pilot-1.json"),
        validation_evidence=["pytest", "report-shared", "pilot-scorecard"],
        pilot_scorecard=scorecard,
        pilot_report_path=str(td / "reports" / "pilot-scorecard.md"),
        git_push_succeeded=True,
        git_push_output="main -> origin/main",
        git_log_stat_output="commit 456def\n 2 files changed, 9 insertions(+)",
    )
    loaded = json.loads(Path(result.journal_path).read_text())
    pilot = {
        "passed": result.acceptance.passed,
        "status": result.acceptance.status,
        "pilot_report_exists": result.pilot_report_path is not None and Path(result.pilot_report_path).exists(),
        "steps": [entry["step"] for entry in loaded["entries"]],
        "pilot_step_status": loaded["entries"][2]["status"],
        "has_roi_text": "Annualized ROI" in Path(result.pilot_report_path).read_text(),
    }

with tempfile.TemporaryDirectory() as td:
    ledger = ObservabilityLedger(str(Path(td) / "ledger.json"))
    task = Task(
        task_id="OPE-60-hold",
        source="linear",
        title="Pilot hold decision",
        description="scorecard blocks closure",
        acceptance_criteria=["pilot-scorecard"],
        validation_plan=["pytest"],
    )
    scorecard = PilotScorecard(
        issue_id="OPE-60",
        customer="Design Partner B",
        period="2026-Q2",
        metrics=[PilotMetric(name="Backlog aging", baseline=4, current=6, target=4, unit="d", higher_is_better=False)],
        monthly_benefit=1000,
        monthly_cost=2500,
        implementation_cost=8000,
        benchmark_passed=False,
    )
    execution = WorkflowEngine().scheduler.execute(task, run_id="run-gate-pilot-1", ledger=ledger)
    decision = AcceptanceGate().evaluate(task, execution, validation_evidence=["pytest", "pilot-scorecard"], pilot_scorecard=scorecard)
    pilot_hold = {
        "passed": decision.passed,
        "status": decision.status,
        "summary": decision.summary,
    }

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    ledger = ObservabilityLedger(str(td / "ledger.json"))
    task = Task(
        task_id="OPE-66-workflow",
        source="linear",
        title="Coordinate customer rollout",
        description="Need browser and analytics support",
        labels=["customer", "data"],
        priority=Priority.P0,
        required_tools=["browser", "sql"],
        acceptance_criteria=["report-shared"],
        validation_plan=["pytest"],
    )
    result = WorkflowEngine().run(
        task,
        run_id="run-wf-ope-66",
        ledger=ledger,
        journal_path=str(td / "journals" / "run-wf-ope-66.json"),
        orchestration_report_path=str(td / "reports" / "run-wf-ope-66-orchestration.md"),
        orchestration_canvas_path=str(td / "reports" / "run-wf-ope-66-canvas.md"),
        validation_evidence=["pytest", "report-shared"],
        git_push_succeeded=True,
        git_push_output="main -> origin/main",
        git_log_stat_output="commit 789fed\n 4 files changed, 16 insertions(+)",
    )
    report = Path(result.orchestration_report_path).read_text()
    canvas = Path(result.orchestration_canvas_path).read_text()
    journal_loaded = json.loads(Path(result.journal_path).read_text())
    entries = ledger.load()
    orchestration = {
        "report_exists": result.orchestration_report_path is not None and Path(result.orchestration_report_path).exists(),
        "canvas_exists": result.orchestration_canvas_path is not None and Path(result.orchestration_canvas_path).exists(),
        "report_excludes_customer_success": "- customer-success:" not in report,
        "report_has_upgrade": "Upgrade Required: True" in report,
        "report_has_handoff": "Human Handoff Team: operations" in report,
        "canvas_has_title": "# Orchestration Canvas" in canvas,
        "canvas_has_recommendation": "Recommendation: resolve-entitlement-gap" in canvas,
        "entry_count": len(entries),
        "artifact_names": [item["name"] for item in entries[0]["artifacts"]],
        "journal_step": journal_loaded["entries"][2]["step"],
        "journal_last_step": journal_loaded["entries"][-1]["step"],
    }

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    ledger = ObservabilityLedger(str(td / "ledger.json"))
    task = Task(
        task_id="OPE-219",
        source="linear",
        title="Audit repo sync",
        description="capture sync failures and pr freshness",
        priority=Priority.P1,
        acceptance_criteria=["repo-sync-audit", "report-shared"],
        validation_plan=["pytest"],
    )
    repo_sync_audit = RepoSyncAudit(
        sync=GitSyncTelemetry(
            status="failed",
            failure_category="divergence",
            summary="branch diverged from remote",
            branch="dcjcloud/ope-219",
            remote_ref="origin/dcjcloud/ope-219",
            ahead_by=2,
            behind_by=1,
        ),
        pull_request=PullRequestFreshness(
            pr_number=219,
            pr_url="https://github.com/OpenAGIs/BigClaw/pull/219",
            branch_state="out-of-sync",
            body_state="drifted",
            branch_head_sha="abc123",
            pr_head_sha="def456",
            expected_body_digest="expected",
            actual_body_digest="actual",
        ),
    )
    result = WorkflowEngine().run(
        task,
        run_id="run-wf-ope-219",
        ledger=ledger,
        journal_path=str(td / "journals" / "run-wf-ope-219.json"),
        validation_evidence=["pytest", "report-shared", "repo-sync-audit"],
        repo_sync_audit=repo_sync_audit,
        repo_sync_report_path=str(td / "reports" / "run-wf-ope-219-repo-sync.md"),
        git_push_succeeded=True,
        git_push_output="feature/OPE-219 -> origin/feature/OPE-219",
        git_log_stat_output="commit abc123\n 3 files changed, 18 insertions(+)",
    )
    report = Path(result.repo_sync_report_path).read_text()
    journal_loaded = json.loads(Path(result.journal_path).read_text())
    entries = ledger.load()
    repo_sync = {
        "passed": result.acceptance.passed,
        "report_exists": result.repo_sync_report_path is not None and Path(result.repo_sync_report_path).exists(),
        "report_has_failure_category": "Failure Category: divergence" in report,
        "report_has_body_state": "Body State: drifted" in report,
        "steps": [entry["step"] for entry in journal_loaded["entries"]],
        "failure_category": journal_loaded["entries"][2]["details"]["failure_category"],
        "audit_actions": [entry["action"] for entry in entries[0]["audits"]],
        "artifact_name": entries[0]["artifacts"][0]["name"],
        "closeout_failure_category": entries[0]["closeout"]["repo_sync_audit"]["sync"]["failure_category"],
    }

print(json.dumps({
    "replay": replay,
    "missing": missing,
    "accepted": accepted,
    "high_risk": high_risk,
    "pilot": pilot,
    "pilot_hold": pilot_hold,
    "orchestration": orchestration,
    "repo_sync": repo_sync,
}))
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write workflow contract script: %v", err)
	}

	cmd := exec.Command("python3", scriptPath, repoRoot)
	cmd.Dir = goRepoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run workflow contract script: %v\n%s", err, string(output))
	}

	var decoded struct {
		Replay struct {
			TaskID string   `json:"task_id"`
			RunID  string   `json:"run_id"`
			Replay []string `json:"replay"`
		} `json:"replay"`
		Missing struct {
			Passed            bool     `json:"passed"`
			Status            string   `json:"status"`
			MissingAcceptance []string `json:"missing_acceptance"`
			MissingValidation []string `json:"missing_validation"`
		} `json:"missing"`
		Accepted struct {
			Medium                    string   `json:"medium"`
			Passed                    bool     `json:"passed"`
			Status                    string   `json:"status"`
			HasJournalPath            bool     `json:"has_journal_path"`
			HasOrchestrationReportPath bool    `json:"has_orchestration_report_path"`
			Steps                     []string `json:"steps"`
			AcceptanceStepStatus      string   `json:"acceptance_step_status"`
			CloseoutStepStatus        string   `json:"closeout_step_status"`
			GitPushSucceeded          bool     `json:"git_push_succeeded"`
		} `json:"accepted"`
		HighRisk struct {
			RunStatus        string `json:"run_status"`
			AcceptancePassed bool   `json:"acceptance_passed"`
			AcceptanceStatus string `json:"acceptance_status"`
		} `json:"high_risk"`
		Pilot struct {
			Passed            bool     `json:"passed"`
			Status            string   `json:"status"`
			PilotReportExists bool     `json:"pilot_report_exists"`
			Steps             []string `json:"steps"`
			PilotStepStatus   string   `json:"pilot_step_status"`
			HasROIText        bool     `json:"has_roi_text"`
		} `json:"pilot"`
		PilotHold struct {
			Passed  bool   `json:"passed"`
			Status  string `json:"status"`
			Summary string `json:"summary"`
		} `json:"pilot_hold"`
		Orchestration struct {
			ReportExists                  bool     `json:"report_exists"`
			CanvasExists                  bool     `json:"canvas_exists"`
			ReportExcludesCustomerSuccess bool     `json:"report_excludes_customer_success"`
			ReportHasUpgrade              bool     `json:"report_has_upgrade"`
			ReportHasHandoff              bool     `json:"report_has_handoff"`
			CanvasHasTitle                bool     `json:"canvas_has_title"`
			CanvasHasRecommendation       bool     `json:"canvas_has_recommendation"`
			EntryCount                    int      `json:"entry_count"`
			ArtifactNames                 []string `json:"artifact_names"`
			JournalStep                   string   `json:"journal_step"`
			JournalLastStep               string   `json:"journal_last_step"`
		} `json:"orchestration"`
		RepoSync struct {
			Passed                 bool     `json:"passed"`
			ReportExists           bool     `json:"report_exists"`
			ReportHasFailureCategory bool   `json:"report_has_failure_category"`
			ReportHasBodyState     bool     `json:"report_has_body_state"`
			Steps                  []string `json:"steps"`
			FailureCategory        string   `json:"failure_category"`
			AuditActions           []string `json:"audit_actions"`
			ArtifactName           string   `json:"artifact_name"`
			CloseoutFailureCategory string  `json:"closeout_failure_category"`
		} `json:"repo_sync"`
	}
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("decode workflow contract output: %v\n%s", err, string(output))
	}

	if decoded.Replay.TaskID != "BIG-402-replay" || decoded.Replay.RunID != "run-journal-1" || len(decoded.Replay.Replay) != 2 || decoded.Replay.Replay[0] != "intake:recorded" || decoded.Replay.Replay[1] != "execution:approved" {
		t.Fatalf("unexpected workpad journal payload: %+v", decoded.Replay)
	}
	if decoded.Missing.Passed || decoded.Missing.Status != "rejected" || len(decoded.Missing.MissingAcceptance) != 1 || decoded.Missing.MissingAcceptance[0] != "report-shared" || len(decoded.Missing.MissingValidation) != 0 {
		t.Fatalf("unexpected acceptance-gate missing-evidence payload: %+v", decoded.Missing)
	}
	if decoded.Accepted.Medium != "browser" || !decoded.Accepted.Passed || decoded.Accepted.Status != "accepted" || !decoded.Accepted.HasJournalPath || !decoded.Accepted.HasOrchestrationReportPath || len(decoded.Accepted.Steps) != 5 || decoded.Accepted.Steps[0] != "intake" || decoded.Accepted.Steps[1] != "execution" || decoded.Accepted.Steps[2] != "orchestration" || decoded.Accepted.Steps[3] != "acceptance" || decoded.Accepted.Steps[4] != "closeout" || decoded.Accepted.AcceptanceStepStatus != "accepted" || decoded.Accepted.CloseoutStepStatus != "complete" || !decoded.Accepted.GitPushSucceeded {
		t.Fatalf("unexpected accepted workflow payload: %+v", decoded.Accepted)
	}
	if decoded.HighRisk.RunStatus != "needs-approval" || decoded.HighRisk.AcceptancePassed || decoded.HighRisk.AcceptanceStatus != "needs-approval" {
		t.Fatalf("unexpected high-risk workflow payload: %+v", decoded.HighRisk)
	}
	if !decoded.Pilot.Passed || decoded.Pilot.Status != "accepted" || !decoded.Pilot.PilotReportExists || len(decoded.Pilot.Steps) != 5 || decoded.Pilot.Steps[2] != "pilot-scorecard" || decoded.Pilot.PilotStepStatus != "go" || !decoded.Pilot.HasROIText {
		t.Fatalf("unexpected pilot workflow payload: %+v", decoded.Pilot)
	}
	if decoded.PilotHold.Passed || decoded.PilotHold.Status != "rejected" || decoded.PilotHold.Summary != "pilot scorecard indicates insufficient ROI or KPI progress" {
		t.Fatalf("unexpected pilot-hold payload: %+v", decoded.PilotHold)
	}
	if !decoded.Orchestration.ReportExists || !decoded.Orchestration.CanvasExists || !decoded.Orchestration.ReportExcludesCustomerSuccess || !decoded.Orchestration.ReportHasUpgrade || !decoded.Orchestration.ReportHasHandoff || !decoded.Orchestration.CanvasHasTitle || !decoded.Orchestration.CanvasHasRecommendation || decoded.Orchestration.EntryCount != 1 || len(decoded.Orchestration.ArtifactNames) != 2 || decoded.Orchestration.ArtifactNames[0] != "cross-department-orchestration" || decoded.Orchestration.ArtifactNames[1] != "orchestration-canvas" || decoded.Orchestration.JournalStep != "orchestration" || decoded.Orchestration.JournalLastStep != "closeout" {
		t.Fatalf("unexpected orchestration workflow payload: %+v", decoded.Orchestration)
	}
	if !decoded.RepoSync.Passed || !decoded.RepoSync.ReportExists || !decoded.RepoSync.ReportHasFailureCategory || !decoded.RepoSync.ReportHasBodyState || len(decoded.RepoSync.Steps) != 5 || decoded.RepoSync.Steps[2] != "repo-sync" || decoded.RepoSync.FailureCategory != "divergence" || decoded.RepoSync.ArtifactName != "repo-sync-audit" || decoded.RepoSync.CloseoutFailureCategory != "divergence" {
		t.Fatalf("unexpected repo-sync workflow payload: %+v", decoded.RepoSync)
	}
	if !containsAuditAction(decoded.RepoSync.AuditActions, "repo.sync") || !containsAuditAction(decoded.RepoSync.AuditActions, "repo.pr-freshness") {
		t.Fatalf("unexpected repo-sync audit actions: %+v", decoded.RepoSync.AuditActions)
	}
}

func containsAuditAction(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
