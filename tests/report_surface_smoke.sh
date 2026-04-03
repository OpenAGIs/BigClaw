#!/bin/sh
set -eu

cd "$(dirname "$0")/.."
PYTHONPATH=src python3 - <<'PY'
from pathlib import Path
from tempfile import TemporaryDirectory

from bigclaw.models import Priority, RiskLevel, Task, WorkflowDefinition
from bigclaw.observability import ObservabilityLedger, TaskRun
from bigclaw.operations import (
    OperationsSnapshot,
    build_operations_api_contract,
    render_operations_dashboard,
)
from bigclaw.planning import CandidateBacklog, ScopeFreezeBoard
from bigclaw.reports import (
    NarrativeSection,
    PilotMetric,
    PilotScorecard,
    ReportStudio,
    SharedViewContext,
    SharedViewFilter,
    render_issue_validation_report,
    render_pilot_scorecard,
    render_report_studio_report,
    render_shared_view_context,
    render_task_run_report,
    write_report,
)
from bigclaw.runtime import ClawWorkerRuntime, ToolRuntime
from bigclaw.scheduler import Scheduler
from bigclaw.workflow import WorkflowEngine


task = Task(
    task_id="BIG-GO-1030-smoke",
    source="linear",
    title="repo surface smoke",
    description="validate remaining consolidated python surfaces",
    priority=Priority.P0,
    risk_level=RiskLevel.MEDIUM,
    required_tools=["github"],
)
run = TaskRun.from_task(task, run_id="run-smoke", medium="docker")
run.log("info", "smoke started")
run.trace("scheduler.decide", "ok", approved=True)
run.audit("scheduler.approved", "system", "success")
run.finalize("succeeded", "ok")

with TemporaryDirectory() as tmpdir:
    ledger = ObservabilityLedger(str(Path(tmpdir) / "ledger.json"))
    ledger.append(run)
    loaded = ledger.load_runs()
    assert len(loaded) == 1
    assert loaded[0].status == "succeeded"

    report_path = Path(tmpdir) / "report.md"
    write_report(str(report_path), render_issue_validation_report("BIG-GO-1030", "v1", "sandbox", "pass"))
    assert report_path.exists()

task_report = render_task_run_report(run)
assert "Run ID: run-smoke" in task_report

studio = ReportStudio(
    name="Smoke Narrative",
    issue_id="BIG-GO-1030",
    audience="ops",
    period="2026-W14",
    summary="Python asset count reduced while compatibility surfaces remain available.",
    sections=[
        NarrativeSection(
            heading="Surface",
            body="Legacy package imports resolve through the consolidated report surface.",
            evidence=["smoke"],
        )
    ],
)
assert "Smoke Narrative" in render_report_studio_report(studio)

scorecard = PilotScorecard(
    issue_id="BIG-GO-1030",
    customer="internal",
    period="2026-Q2",
    metrics=[PilotMetric(name="py files", baseline=15, current=2, target=2, unit="count", higher_is_better=False)],
    monthly_benefit=1000,
    monthly_cost=100,
    implementation_cost=500,
    benchmark_score=95,
    benchmark_passed=True,
)
assert "Recommendation:" in render_pilot_scorecard(scorecard)

shared = SharedViewContext(
    filters=[SharedViewFilter(label="Scope", value="repo")],
    result_count=1,
    last_updated="2026-04-04T00:00:00Z",
)
assert "- Scope: repo" in "\n".join(render_shared_view_context(shared))

definition = WorkflowDefinition.from_dict(
    {
        "name": "closeout",
        "steps": [{"name": "execute", "kind": "scheduler"}],
        "validation_evidence": ["smoke"],
    }
)
with TemporaryDirectory() as tmpdir:
    result = WorkflowEngine().run_definition(
        task,
        definition=definition,
        run_id="workflow-smoke",
        ledger=ObservabilityLedger(str(Path(tmpdir) / "workflow-ledger.json")),
    )
    assert result.acceptance.status == "accepted"

runtime = ClawWorkerRuntime(tool_runtime=ToolRuntime(handlers={"github": lambda action, payload: "ok"}))
decision = Scheduler().decide(task)
worker_result = runtime.execute(task, decision=decision, run=TaskRun.from_task(task, run_id="worker-smoke", medium=decision.medium))
assert worker_result.tool_results[0].success is True

assert CandidateBacklog.from_dict(
    {
        "epic_id": "BIG-EPIC-20",
        "title": "smoke",
        "version": "v1",
        "candidates": [],
    }
).epic_id == "BIG-EPIC-20"
assert ScopeFreezeBoard.from_dict(
    {
        "name": "board",
        "version": "v1",
        "freeze_date": "2026-04-04",
        "freeze_owner": "ops",
        "backlog_items": [],
        "exceptions": [],
    }
).name == "board"

assert build_operations_api_contract().contract_id
dashboard = render_operations_dashboard(
    OperationsSnapshot(
        total_runs=1,
        status_counts={"succeeded": 1},
        success_rate=100.0,
        approval_queue_depth=0,
        sla_target_minutes=30,
        sla_breach_count=0,
        average_cycle_minutes=5.0,
    )
)
assert "Operations Dashboard" in dashboard

print("smoke-ok")
PY
