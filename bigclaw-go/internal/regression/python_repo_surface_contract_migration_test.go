package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPythonRepoSurfaceContractMigration(t *testing.T) {
	repoRoot := repoRoot(t)
	payload := runPythonRepoSurfaceContracts(t, repoRoot)

	dashboard := payload.DashboardRunContract
	if !dashboard.DefaultReleaseReady || !dashboard.ReportHasDashboardID || !dashboard.ReportHasRunID || !dashboard.ReportHasReleaseReady {
		t.Fatalf("unexpected dashboard default contract: %+v", dashboard)
	}
	if len(dashboard.MissingDashboardFields) != 1 || dashboard.MissingDashboardFields[0] != "summary.success_rate" {
		t.Fatalf("unexpected dashboard missing fields: %+v", dashboard.MissingDashboardFields)
	}
	if len(dashboard.MissingDashboardSampleGaps) != 1 || dashboard.MissingDashboardSampleGaps[0] != "activity" {
		t.Fatalf("unexpected dashboard sample gaps: %+v", dashboard.MissingDashboardSampleGaps)
	}
	if len(dashboard.MissingRunDetailFields) != 1 || dashboard.MissingRunDetailFields[0] != "closeout.git_log_stat_output" {
		t.Fatalf("unexpected run detail missing fields: %+v", dashboard.MissingRunDetailFields)
	}
	if len(dashboard.MissingRunDetailSampleGaps) != 1 || dashboard.MissingRunDetailSampleGaps[0] != "closeout.git_log_stat_output" {
		t.Fatalf("unexpected run detail sample gaps: %+v", dashboard.MissingRunDetailSampleGaps)
	}
	if dashboard.BrokenReleaseReady || !dashboard.RoundTripEqual || !dashboard.RestoredAuditReleaseReady || !dashboard.HasDashboardIDSchemaField {
		t.Fatalf("unexpected dashboard round-trip contract: %+v", dashboard)
	}

	gateway := payload.RepoGateway
	if gateway.CommitHash != "abc123" || gateway.LineageLeavesCount != 1 || gateway.LineageLeaf != "def456" || gateway.DiffFilesChanged != 3 || gateway.PayloadActor != "native cloud" || gateway.PayloadCommitHash != "def456" {
		t.Fatalf("unexpected repo gateway normalization: %+v", gateway)
	}
	if gateway.TimeoutCode != "timeout" || !gateway.TimeoutRetryable || gateway.NotFoundCode != "not_found" || gateway.NotFoundRetryable {
		t.Fatalf("unexpected repo gateway error normalization: %+v", gateway)
	}

	risk := payload.Risk
	if risk.LowTotal != 0 || risk.LowLevel != "low" || risk.LowRequiresApproval {
		t.Fatalf("unexpected low risk score: %+v", risk)
	}
	if risk.MediumTotal != 40 || risk.MediumLevel != "medium" || risk.MediumRequiresApproval {
		t.Fatalf("unexpected medium risk score: %+v", risk)
	}
	if risk.ExecuteRiskTotal != 70 || risk.ExecuteRiskLevel != "high" || risk.ExecuteMedium != "vm" || risk.ExecuteApproved || !risk.HasRiskTrace || !risk.HasRiskAudit {
		t.Fatalf("unexpected scheduler risk execution contract: %+v", risk)
	}

	runtime := payload.RuntimeMatrix
	if runtime.ToolResultCount != 2 || !runtime.AllToolResultsSuccessful || runtime.LastWorkerLifecycleAction != "worker.lifecycle" || runtime.LastWorkerLifecycleOutcome != "completed" {
		t.Fatalf("unexpected runtime worker lifecycle contract: %+v", runtime)
	}
	if runtime.LowMedium != "docker" || runtime.HighMedium != "vm" || (runtime.BrowserMedium != "browser" && runtime.BrowserMedium != "docker") {
		t.Fatalf("unexpected runtime routing contract: %+v", runtime)
	}
	if !runtime.AllowSuccess || runtime.BlockSuccess || !runtime.HasSuccessOutcome || !runtime.HasBlockedOutcome {
		t.Fatalf("unexpected runtime tool policy contract: %+v", runtime)
	}
}

type pythonRepoSurfaceContractsPayload struct {
	DashboardRunContract struct {
		DefaultReleaseReady        bool     `json:"default_release_ready"`
		ReportHasDashboardID       bool     `json:"report_has_dashboard_id"`
		ReportHasRunID             bool     `json:"report_has_run_id"`
		ReportHasReleaseReady      bool     `json:"report_has_release_ready"`
		MissingDashboardFields     []string `json:"missing_dashboard_fields"`
		MissingDashboardSampleGaps []string `json:"missing_dashboard_sample_gaps"`
		MissingRunDetailFields     []string `json:"missing_run_detail_fields"`
		MissingRunDetailSampleGaps []string `json:"missing_run_detail_sample_gaps"`
		BrokenReleaseReady         bool     `json:"broken_release_ready"`
		RoundTripEqual             bool     `json:"round_trip_equal"`
		RestoredAuditReleaseReady  bool     `json:"restored_audit_release_ready"`
		HasDashboardIDSchemaField  bool     `json:"has_dashboard_id_schema_field"`
	} `json:"dashboard_run_contract"`
	RepoGateway struct {
		CommitHash         string `json:"commit_hash"`
		LineageLeavesCount int    `json:"lineage_leaves_count"`
		LineageLeaf        string `json:"lineage_leaf"`
		DiffFilesChanged   int    `json:"diff_files_changed"`
		PayloadActor       string `json:"payload_actor"`
		PayloadCommitHash  string `json:"payload_commit_hash"`
		TimeoutCode        string `json:"timeout_code"`
		TimeoutRetryable   bool   `json:"timeout_retryable"`
		NotFoundCode       string `json:"not_found_code"`
		NotFoundRetryable  bool   `json:"not_found_retryable"`
	} `json:"repo_gateway"`
	Risk struct {
		LowTotal               int    `json:"low_total"`
		LowLevel               string `json:"low_level"`
		LowRequiresApproval    bool   `json:"low_requires_approval"`
		MediumTotal            int    `json:"medium_total"`
		MediumLevel            string `json:"medium_level"`
		MediumRequiresApproval bool   `json:"medium_requires_approval"`
		ExecuteRiskTotal       int    `json:"execute_risk_total"`
		ExecuteRiskLevel       string `json:"execute_risk_level"`
		ExecuteMedium          string `json:"execute_medium"`
		ExecuteApproved        bool   `json:"execute_approved"`
		HasRiskTrace           bool   `json:"has_risk_trace"`
		HasRiskAudit           bool   `json:"has_risk_audit"`
	} `json:"risk"`
	RuntimeMatrix struct {
		ToolResultCount            int    `json:"tool_result_count"`
		AllToolResultsSuccessful   bool   `json:"all_tool_results_successful"`
		LastWorkerLifecycleAction  string `json:"last_worker_lifecycle_action"`
		LastWorkerLifecycleOutcome string `json:"last_worker_lifecycle_outcome"`
		LowMedium                  string `json:"low_medium"`
		HighMedium                 string `json:"high_medium"`
		BrowserMedium              string `json:"browser_medium"`
		AllowSuccess               bool   `json:"allow_success"`
		BlockSuccess               bool   `json:"block_success"`
		HasSuccessOutcome          bool   `json:"has_success_outcome"`
		HasBlockedOutcome          bool   `json:"has_blocked_outcome"`
	} `json:"runtime_matrix"`
}

func runPythonRepoSurfaceContracts(t *testing.T, repoRoot string) pythonRepoSurfaceContractsPayload {
	t.Helper()

	code := `
import json
import tempfile
from pathlib import Path

from bigclaw.dashboard_run_contract import DashboardRunContract, DashboardRunContractAudit, DashboardRunContractLibrary, SchemaField, render_dashboard_run_contract_report
from bigclaw.models import Priority, RiskLevel, Task
from bigclaw.observability import ObservabilityLedger, TaskRun
from bigclaw.repo_gateway import normalize_commit, normalize_diff, normalize_gateway_error, normalize_lineage, repo_audit_payload
from bigclaw.risk import RiskScorer
from bigclaw.runtime import ClawWorkerRuntime, ToolPolicy, ToolRuntime
from bigclaw.scheduler import Scheduler

library = DashboardRunContractLibrary()
contract = library.build_default_contract()
audit = library.audit(contract)
report = render_dashboard_run_contract_report(contract, audit)

broken = library.build_default_contract()
broken.dashboard_schema.fields = [field for field in broken.dashboard_schema.fields if field.name != "summary.success_rate"]
broken.dashboard_schema.sample.pop("activity")
broken.run_detail_schema.fields = [field for field in broken.run_detail_schema.fields if field.name != "closeout.git_log_stat_output"]
broken.run_detail_schema.sample["closeout"].pop("git_log_stat_output")
broken_audit = library.audit(broken)

restored = DashboardRunContract.from_dict(contract.to_dict())
restored_audit = DashboardRunContractAudit.from_dict(library.audit(contract).to_dict())

commit = normalize_commit({"commit_hash": "abc123", "title": "feat: add repo plane", "author": "bot"})
lineage = normalize_lineage({
    "root_hash": "abc123",
    "lineage": [commit.to_dict()],
    "children": {"abc123": ["def456"]},
    "leaves": ["def456"],
})
diff = normalize_diff({
    "left_hash": "abc123",
    "right_hash": "def456",
    "files_changed": 3,
    "insertions": 20,
    "deletions": 4,
    "summary": "3 files changed",
})
payload = repo_audit_payload(
    actor="native cloud",
    action="repo.diff",
    outcome="success",
    commit_hash="def456",
    repo_space_id="space-1",
)
timeout = normalize_gateway_error(RuntimeError("gateway timeout while fetching lineage"))
missing = normalize_gateway_error(RuntimeError("commit not found"))

low = RiskScorer().score_task(Task(task_id="BIG-902-low", source="linear", title="doc cleanup", description=""))
medium = RiskScorer().score_task(Task(
    task_id="BIG-902-mid",
    source="linear",
    title="release verification",
    description="prod browser change",
    labels=["prod"],
    priority=Priority.P0,
    required_tools=["browser"],
))

with tempfile.TemporaryDirectory() as tmpdir:
    ledger = ObservabilityLedger(str(Path(tmpdir) / "ledger.json"))
    task = Task(
        task_id="BIG-902-high",
        source="linear",
        title="security deploy",
        description="prod deploy",
        labels=["security", "prod"],
        priority=Priority.P0,
        required_tools=["deploy"],
    )
    record = Scheduler().execute(task, run_id="run-risk", ledger=ledger)
    entry = ledger.load()[0]

task = Task(
    task_id="BIG-301-matrix",
    source="github",
    title="worker lifecycle matrix",
    description="validate stable lifecycle",
    required_tools=["github", "browser"],
)
run = TaskRun.from_task(task, run_id="run-big301-matrix", medium="docker")
runtime = ToolRuntime(handlers={
    "github": lambda action, payload: f"{action}:{payload.get('repo', 'none')}",
    "browser": lambda action, payload: f"{action}:{payload.get('url', 'none')}",
})
worker = ClawWorkerRuntime(tool_runtime=runtime)
result = worker.execute(
    task,
    decision=type("Decision", (), {"medium": "docker", "approved": True, "reason": "ok"})(),
    run=run,
    tool_payloads={"github": {"repo": "OpenAGIs/BigClaw"}, "browser": {"url": "https://example.com"}},
)

scheduler = Scheduler()
low_task = Task(task_id="low", source="local", title="low", description="", risk_level=RiskLevel.LOW)
high_task = Task(task_id="high", source="local", title="high", description="", risk_level=RiskLevel.HIGH)
browser_task = Task(task_id="browser", source="local", title="browser", description="", required_tools=["browser"], risk_level=RiskLevel.MEDIUM)

policy_task = Task(task_id="BIG-303-matrix", source="local", title="tool policy", description="", required_tools=["github", "browser"])
policy_run = TaskRun.from_task(policy_task, run_id="run-big303-matrix", medium="docker")
policy_runtime = ToolRuntime(
    policy=ToolPolicy(allowed_tools=["github"], blocked_tools=["browser"]),
    handlers={"github": lambda action, payload: "ok"},
)
allow = policy_runtime.invoke("github", action="execute", payload={"repo": "OpenAGIs/BigClaw"}, run=policy_run)
block = policy_runtime.invoke("browser", action="execute", payload={"url": "https://example.com"}, run=policy_run)
outcomes = [audit.outcome for audit in policy_run.audits if audit.action == "tool.invoke"]

print(json.dumps({
    "dashboard_run_contract": {
        "default_release_ready": audit.release_ready,
        "report_has_dashboard_id": "eng-overview-core-product" in report,
        "report_has_run_id": '"run_id": "run-204"' in report,
        "report_has_release_ready": "- Release Ready: True" in report,
        "missing_dashboard_fields": broken_audit.dashboard_missing_fields,
        "missing_dashboard_sample_gaps": broken_audit.dashboard_sample_gaps,
        "missing_run_detail_fields": broken_audit.run_detail_missing_fields,
        "missing_run_detail_sample_gaps": broken_audit.run_detail_sample_gaps,
        "broken_release_ready": broken_audit.release_ready,
        "round_trip_equal": restored == contract,
        "restored_audit_release_ready": restored_audit.release_ready,
        "has_dashboard_id_schema_field": any(field == SchemaField("dashboard_id", "string", description="Stable dashboard identifier.") for field in restored.dashboard_schema.fields),
    },
    "repo_gateway": {
        "commit_hash": commit.commit_hash,
        "lineage_leaves_count": len(lineage.leaves),
        "lineage_leaf": lineage.leaves[0],
        "diff_files_changed": diff.files_changed,
        "payload_actor": payload["actor"],
        "payload_commit_hash": payload["commit_hash"],
        "timeout_code": timeout.code,
        "timeout_retryable": timeout.retryable,
        "not_found_code": missing.code,
        "not_found_retryable": missing.retryable,
    },
    "risk": {
        "low_total": low.total,
        "low_level": low.level.value,
        "low_requires_approval": low.requires_approval,
        "medium_total": medium.total,
        "medium_level": medium.level.value,
        "medium_requires_approval": medium.requires_approval,
        "execute_risk_total": record.risk_score.total if record.risk_score is not None else -1,
        "execute_risk_level": record.risk_score.level.value if record.risk_score is not None else "",
        "execute_medium": record.decision.medium,
        "execute_approved": record.decision.approved,
        "has_risk_trace": any(trace["span"] == "risk.score" for trace in entry["traces"]),
        "has_risk_audit": any(audit["action"] == "risk.score" for audit in entry["audits"]),
    },
    "runtime_matrix": {
        "tool_result_count": len(result.tool_results),
        "all_tool_results_successful": all(item.success for item in result.tool_results),
        "last_worker_lifecycle_action": run.audits[-1].action,
        "last_worker_lifecycle_outcome": run.audits[-1].outcome,
        "low_medium": scheduler.decide(low_task).medium,
        "high_medium": scheduler.decide(high_task).medium,
        "browser_medium": scheduler.decide(browser_task).medium,
        "allow_success": allow.success,
        "block_success": block.success,
        "has_success_outcome": "success" in outcomes,
        "has_blocked_outcome": "blocked" in outcomes,
    },
}))
`

	cmd := exec.Command("python3", "-c", code)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "PYTHONPATH="+filepath.Join(repoRoot, "src"))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run python repo surface contracts: %v\n%s", err, output)
	}

	var payload pythonRepoSurfaceContractsPayload
	if err := json.Unmarshal(output, &payload); err != nil {
		t.Fatalf("decode python repo surface contracts payload: %v\n%s", err, output)
	}
	return payload
}
