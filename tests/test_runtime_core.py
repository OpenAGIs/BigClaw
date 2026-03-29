import json
from pathlib import Path

from bigclaw.models import Priority, RiskLevel, Task
from bigclaw.observability import GitSyncTelemetry, ObservabilityLedger, PullRequestFreshness, RepoSyncAudit, TaskRun
from bigclaw.orchestration import (
    CrossDepartmentOrchestrator,
    PremiumOrchestrationPolicy,
    render_orchestration_plan,
)
from bigclaw.queue import PersistentTaskQueue
from bigclaw.reports import PilotMetric, PilotScorecard
from bigclaw.runtime import ClawWorkerRuntime, SandboxRouter, ToolPolicy, ToolRuntime
from bigclaw.scheduler import Scheduler, SchedulerDecision
from bigclaw.workflow import AcceptanceGate, WorkflowEngine, WorkpadJournal


def test_sandbox_router_maps_execution_media():
    router = SandboxRouter()

    assert router.profile_for("docker").isolation == "container"
    assert router.profile_for("browser").network_access == "enabled"
    assert router.profile_for("vm").filesystem_access == "workspace-write"
    assert router.profile_for("unknown").medium == "none"


def test_tool_runtime_blocks_disallowed_tool_and_audits(tmp_path: Path):
    task = Task(task_id="BIG-303", source="linear", title="tool policy", description="")
    run = TaskRun.from_task(task, run_id="run-tool-1", medium="docker")
    runtime = ToolRuntime(policy=ToolPolicy(allowed_tools=["github"], blocked_tools=["browser"]))

    result = runtime.invoke("browser", action="execute", payload={"url": "https://example.com"}, run=run)

    assert result.success is False
    assert result.error == "tool blocked by policy"
    assert run.audits[-1].action == "tool.invoke"
    assert run.audits[-1].outcome == "blocked"


def test_worker_runtime_returns_tool_results_for_approved_task():
    task = Task(
        task_id="BIG-301",
        source="linear",
        title="worker runtime",
        description="return tool results",
        required_tools=["github"],
    )
    run = TaskRun.from_task(task, run_id="run-worker-1", medium="docker")
    tool_runtime = ToolRuntime(handlers={"github": lambda action, payload: f"{action}:{payload['repo']}"})
    worker = ClawWorkerRuntime(tool_runtime=tool_runtime)

    result = worker.execute(
        task,
        SchedulerDecision(medium="docker", approved=True, reason="default low risk path"),
        run,
        tool_payloads={"github": {"repo": "OpenAGIs/BigClaw"}},
    )

    assert result.sandbox_profile.medium == "docker"
    assert result.tool_results[0].success is True
    assert result.tool_results[0].output == "execute:OpenAGIs/BigClaw"
    assert run.audits[-1].action == "worker.lifecycle"
    assert run.audits[-1].outcome == "completed"


def test_scheduler_records_worker_runtime_results_and_waits_on_high_risk(tmp_path: Path):
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="BIG-302",
        source="jira",
        title="prod sandbox",
        description="needs approval",
        risk_level=RiskLevel.HIGH,
        required_tools=["browser"],
    )

    record = Scheduler().execute(task, run_id="run-worker-2", ledger=ledger)
    entry = ledger.load()[0]

    assert record.decision.medium == "vm"
    assert record.run.status == "needs-approval"
    assert record.tool_results == []
    assert entry["audits"][-1]["action"] == "worker.lifecycle"
    assert entry["audits"][-1]["outcome"] == "waiting-approval"


def test_scheduler_pauses_execution_when_budget_cannot_cover_docker(tmp_path: Path):
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="BIG-202-budget",
        source="linear",
        title="budget pause",
        description="budget should stop execution",
        budget=5.0,
        required_tools=["github"],
    )

    record = Scheduler().execute(task, run_id="run-budget-pause", ledger=ledger)
    entry = ledger.load()[0]

    assert record.decision.medium == "none"
    assert record.decision.approved is False
    assert record.run.status == "paused"
    assert record.tool_results == []
    assert entry["audits"][-1]["action"] == "worker.lifecycle"
    assert entry["audits"][-1]["outcome"] == "paused"


def test_scheduler_high_risk_requires_approval():
    s = Scheduler()
    t = Task(task_id="x", source="jira", title="prod op", description="", risk_level=RiskLevel.HIGH)
    d = s.decide(t)
    assert d.medium == "vm"
    assert d.approved is False


def test_scheduler_browser_task_routes_browser():
    s = Scheduler()
    t = Task(task_id="y", source="github", title="ui test", description="", required_tools=["browser"])
    d = s.decide(t)
    assert d.medium == "browser"
    assert d.approved is True


def test_scheduler_over_budget_degrades_browser_task_to_docker():
    s = Scheduler()
    t = Task(
        task_id="z",
        source="github",
        title="budgeted ui test",
        description="",
        required_tools=["browser"],
        budget=15.0,
    )

    d = s.decide(t)

    assert d.medium == "docker"
    assert d.approved is True
    assert "budget degraded browser route to docker" in d.reason


def test_scheduler_over_budget_pauses_task():
    s = Scheduler()
    t = Task(task_id="b", source="linear", title="tiny budget", description="", budget=5.0)

    d = s.decide(t)

    assert d.medium == "none"
    assert d.approved is False
    assert d.reason == "paused: budget 5.0 below required docker budget 10.0"


def test_workpad_journal_can_replay_and_reload(tmp_path: Path):
    journal = WorkpadJournal(task_id="BIG-402-replay", run_id="run-journal-1")
    journal.record("intake", "recorded", source="local")
    journal.record("execution", "approved", medium="docker")
    output = journal.write(str(tmp_path / "journals" / "run-journal-1.json"))

    loaded = WorkpadJournal.read(output)

    assert loaded.task_id == "BIG-402-replay"
    assert loaded.run_id == "run-journal-1"
    assert loaded.replay() == ["intake:recorded", "execution:approved"]


def test_acceptance_gate_rejects_missing_evidence(tmp_path: Path):
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
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

    assert decision.passed is False
    assert decision.status == "rejected"
    assert decision.missing_acceptance_criteria == ["report-shared"]
    assert decision.missing_validation_steps == []


def test_workflow_engine_records_journal_and_accepts_complete_evidence(tmp_path: Path):
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
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
        report_path=str(tmp_path / "reports" / "run-wf-1.md"),
        journal_path=str(tmp_path / "journals" / "run-wf-1.json"),
        validation_evidence=["pytest", "report-shared"],
        orchestration_report_path=str(tmp_path / "reports" / "run-wf-1-orchestration.md"),
        git_push_succeeded=True,
        git_push_output="main -> origin/main",
        git_log_stat_output="commit 123abc\n 3 files changed, 12 insertions(+)",
    )

    assert result.execution.decision.medium == "browser"
    assert result.acceptance.passed is True
    assert result.acceptance.status == "accepted"
    assert result.journal_path is not None
    assert result.orchestration_report_path is not None

    journal = json.loads(Path(result.journal_path).read_text())
    assert [entry["step"] for entry in journal["entries"]] == ["intake", "execution", "orchestration", "acceptance", "closeout"]
    assert journal["entries"][-2]["status"] == "accepted"
    assert journal["entries"][-1]["status"] == "complete"
    assert ledger.load()[0]["closeout"]["git_push_succeeded"] is True


def test_workflow_engine_keeps_high_risk_task_pending_manual_approval(tmp_path: Path):
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
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

    assert result.execution.run.status == "needs-approval"
    assert result.acceptance.passed is False
    assert result.acceptance.status == "needs-approval"


def test_workflow_engine_writes_pilot_scorecard_and_accepts_positive_roi(tmp_path: Path):
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
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
        journal_path=str(tmp_path / "journals" / "run-wf-pilot-1.json"),
        validation_evidence=["pytest", "report-shared", "pilot-scorecard"],
        pilot_scorecard=scorecard,
        pilot_report_path=str(tmp_path / "reports" / "pilot-scorecard.md"),
        git_push_succeeded=True,
        git_push_output="main -> origin/main",
        git_log_stat_output="commit 456def\n 2 files changed, 9 insertions(+)",
    )

    assert result.acceptance.passed is True
    assert result.acceptance.status == "accepted"
    assert result.pilot_report_path is not None
    assert Path(result.pilot_report_path).exists()

    journal = json.loads(Path(result.journal_path).read_text())
    assert [entry["step"] for entry in journal["entries"]] == ["intake", "execution", "pilot-scorecard", "acceptance", "closeout"]
    assert journal["entries"][2]["status"] == "go"
    assert "Annualized ROI" in Path(result.pilot_report_path).read_text()


def test_acceptance_gate_rejects_hold_pilot_scorecard(tmp_path: Path):
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
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
    decision = AcceptanceGate().evaluate(
        task,
        execution,
        validation_evidence=["pytest", "pilot-scorecard"],
        pilot_scorecard=scorecard,
    )

    assert decision.passed is False
    assert decision.status == "rejected"
    assert decision.summary == "pilot scorecard indicates insufficient ROI or KPI progress"


def test_workflow_engine_writes_orchestration_report_without_duplicating_ledger_entries(tmp_path: Path):
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
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
        journal_path=str(tmp_path / "journals" / "run-wf-ope-66.json"),
        orchestration_report_path=str(tmp_path / "reports" / "run-wf-ope-66-orchestration.md"),
        orchestration_canvas_path=str(tmp_path / "reports" / "run-wf-ope-66-canvas.md"),
        validation_evidence=["pytest", "report-shared"],
        git_push_succeeded=True,
        git_push_output="main -> origin/main",
        git_log_stat_output="commit 789fed\n 4 files changed, 16 insertions(+)",
    )

    assert result.orchestration_report_path is not None
    assert Path(result.orchestration_report_path).exists()
    assert result.orchestration_canvas_path is not None
    assert Path(result.orchestration_canvas_path).exists()
    report = Path(result.orchestration_report_path).read_text()
    assert "- customer-success:" not in report
    assert "Upgrade Required: True" in report
    assert "Human Handoff Team: operations" in report
    canvas = Path(result.orchestration_canvas_path).read_text()
    assert "# Orchestration Canvas" in canvas
    assert "Recommendation: resolve-entitlement-gap" in canvas

    entries = ledger.load()
    assert len(entries) == 1
    assert entries[0]["artifacts"][0]["name"] == "cross-department-orchestration"
    assert entries[0]["artifacts"][1]["name"] == "orchestration-canvas"

    journal = json.loads(Path(result.journal_path).read_text())
    assert journal["entries"][2]["step"] == "orchestration"
    assert journal["entries"][-1]["step"] == "closeout"


def test_workflow_engine_writes_repo_sync_audit_report_and_records_failure_categories(tmp_path: Path):
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
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
        journal_path=str(tmp_path / "journals" / "run-wf-ope-219.json"),
        validation_evidence=["pytest", "report-shared", "repo-sync-audit"],
        repo_sync_audit=repo_sync_audit,
        repo_sync_report_path=str(tmp_path / "reports" / "run-wf-ope-219-repo-sync.md"),
        git_push_succeeded=True,
        git_push_output="feature/OPE-219 -> origin/feature/OPE-219",
        git_log_stat_output="commit abc123\n 3 files changed, 18 insertions(+)",
    )

    assert result.acceptance.passed is True
    assert result.repo_sync_report_path is not None
    assert Path(result.repo_sync_report_path).exists()
    report = Path(result.repo_sync_report_path).read_text()
    assert "Failure Category: divergence" in report
    assert "Body State: drifted" in report

    journal = json.loads(Path(result.journal_path).read_text())
    assert [entry["step"] for entry in journal["entries"]] == [
        "intake",
        "execution",
        "repo-sync",
        "acceptance",
        "closeout",
    ]
    assert journal["entries"][2]["details"]["failure_category"] == "divergence"
    entries = ledger.load()
    audit_actions = [entry["action"] for entry in entries[0]["audits"]]
    assert "repo.sync" in audit_actions
    assert "repo.pr-freshness" in audit_actions
    assert entries[0]["artifacts"][0]["name"] == "repo-sync-audit"
    assert entries[0]["closeout"]["repo_sync_audit"]["sync"]["failure_category"] == "divergence"


def test_cross_department_orchestrator_routes_security_data_and_customer_work() -> None:
    task = Task(
        task_id="OPE-66",
        source="linear",
        title="Coordinate customer analytics rollout approval",
        description="Need stakeholder sign-off for warehouse-backed browser workflow",
        labels=["data", "customer", "premium"],
        priority=Priority.P0,
        risk_level=RiskLevel.HIGH,
        required_tools=["browser", "sql"],
        acceptance_criteria=["approval recorded"],
        validation_plan=["customer signoff"],
    )

    plan = CrossDepartmentOrchestrator().plan(task)

    assert plan.collaboration_mode == "cross-functional"
    assert plan.departments == ["operations", "engineering", "security", "data", "customer-success"]
    assert plan.required_approvals == ["security-review"]


def test_standard_policy_limits_advanced_cross_department_routing() -> None:
    task = Task(
        task_id="OPE-66-standard",
        source="linear",
        title="Coordinate customer analytics rollout approval",
        description="Need stakeholder sign-off for warehouse-backed browser workflow",
        labels=["data", "customer"],
        required_tools=["browser", "sql"],
        risk_level=RiskLevel.HIGH,
    )

    raw_plan = CrossDepartmentOrchestrator().plan(task)
    plan, policy = PremiumOrchestrationPolicy().apply(task, raw_plan)

    assert plan.collaboration_mode == "tier-limited"
    assert plan.departments == ["operations", "engineering"]
    assert policy.upgrade_required is True
    assert policy.entitlement_status == "upgrade-required"
    assert policy.billing_model == "standard-blocked"
    assert policy.included_usage_units == 2
    assert policy.overage_usage_units == 3
    assert policy.overage_cost_usd == 12.0
    assert policy.estimated_cost_usd == 15.0
    assert policy.blocked_departments == ["security", "data", "customer-success"]


def test_render_orchestration_plan_lists_handoffs_and_policy() -> None:
    task = Task(
        task_id="OPE-66-render",
        source="jira",
        title="Warehouse rollout",
        description="Customer-ready release",
        labels=["data", "customer"],
        required_tools=["sql"],
    )

    raw_plan = CrossDepartmentOrchestrator().plan(task)
    plan, policy = PremiumOrchestrationPolicy().apply(task, raw_plan)
    content = render_orchestration_plan(plan, policy)

    assert "# Cross-Department Orchestration Plan" in content
    assert "- Departments: operations, engineering" in content
    assert "- Tier: standard" in content
    assert "- Entitlement Status: upgrade-required" in content
    assert "- Billing Model: standard-blocked" in content
    assert "- Estimated Cost (USD): 11.00" in content
    assert "- Blocked Departments: data, customer-success" in content
    assert "- Human Handoff Team:" not in content


def test_scheduler_execution_records_orchestration_plan_and_policy(tmp_path: Path) -> None:
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="OPE-66-exec",
        source="linear",
        title="Cross-team browser change",
        description="Program-managed rollout",
        labels=["ops"],
        priority=Priority.P0,
        risk_level=RiskLevel.MEDIUM,
        required_tools=["browser"],
    )

    record = Scheduler().execute(task, run_id="run-ope-66", ledger=ledger)
    entry = ledger.load()[0]

    assert record.orchestration_plan is not None
    assert record.orchestration_policy is not None
    assert record.orchestration_plan.departments == ["operations", "engineering"]
    assert record.orchestration_policy.upgrade_required is False
    assert record.orchestration_policy.entitlement_status == "included"
    assert record.orchestration_policy.billing_model == "standard-included"
    assert record.orchestration_policy.estimated_cost_usd == 3.0
    assert any(trace["span"] == "orchestration.plan" for trace in entry["traces"])
    assert any(trace["span"] == "orchestration.policy" for trace in entry["traces"])
    assert any(audit["action"] == "orchestration.plan" for audit in entry["audits"])
    assert any(audit["action"] == "orchestration.policy" for audit in entry["audits"])
    policy_audit = next(audit for audit in entry["audits"] if audit["action"] == "orchestration.policy")
    assert policy_audit["details"]["entitlement_status"] == "included"
    assert policy_audit["details"]["billing_model"] == "standard-included"


def test_scheduler_creates_handoff_for_policy_or_approval_blockers(tmp_path: Path) -> None:
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
    task = Task(
        task_id="OPE-66-handoff",
        source="linear",
        title="Customer analytics rollout",
        description="Need cross-team coordination",
        labels=["customer", "data"],
        required_tools=["browser", "sql"],
    )

    record = Scheduler().execute(task, run_id="run-ope-66-handoff", ledger=ledger)
    entry = ledger.load()[0]

    assert record.handoff_request is not None
    assert record.handoff_request.target_team == "operations"
    assert any(trace["span"] == "orchestration.handoff" for trace in entry["traces"])
    assert any(audit["action"] == "orchestration.handoff" for audit in entry["audits"])


def test_queue_persistence_and_priority(tmp_path: Path):
    qfile = tmp_path / "queue.json"
    q = PersistentTaskQueue(str(qfile))

    q.enqueue(Task(task_id="t2", source="linear", title="P1", description="", priority=Priority.P1))
    q.enqueue(Task(task_id="t1", source="linear", title="P0", description="", priority=Priority.P0))

    assert q.size() == 2
    first = q.dequeue()
    assert first["task_id"] == "t1"

    q2 = PersistentTaskQueue(str(qfile))
    assert q2.size() == 1


def test_queue_creates_parent_directory_and_preserves_task_payload(tmp_path: Path):
    qfile = tmp_path / "state" / "queue.json"
    q = PersistentTaskQueue(str(qfile))

    q.enqueue(
        Task(
            task_id="t-meta",
            source="linear",
            title="Persist metadata",
            description="keep fields",
            labels=["platform"],
            required_tools=["browser"],
            acceptance_criteria=["queue survives restart"],
            validation_plan=["pytest tests/test_runtime_core.py"],
            priority=Priority.P1,
        )
    )

    reloaded = PersistentTaskQueue(str(qfile))
    task = reloaded.dequeue_task()

    assert qfile.exists()
    assert task is not None
    assert task.labels == ["platform"]
    assert task.required_tools == ["browser"]
    assert task.acceptance_criteria == ["queue survives restart"]
    assert task.validation_plan == ["pytest tests/test_runtime_core.py"]


def test_queue_dead_letter_and_retry_persist_across_reload(tmp_path: Path):
    qfile = tmp_path / "queue.json"
    q = PersistentTaskQueue(str(qfile))
    q.enqueue(Task(task_id="t-dead", source="linear", title="dead", description="", priority=Priority.P0))

    task = q.dequeue_task()
    assert task is not None

    q.dead_letter(task, reason="executor crashed")

    dead_letters = q.list_dead_letters()
    assert q.size() == 0
    assert len(dead_letters) == 1
    assert dead_letters[0].task.task_id == "t-dead"
    assert dead_letters[0].reason == "executor crashed"

    reloaded = PersistentTaskQueue(str(qfile))
    persisted_dead_letters = reloaded.list_dead_letters()
    assert len(persisted_dead_letters) == 1
    assert persisted_dead_letters[0].task.task_id == "t-dead"

    assert reloaded.retry_dead_letter("t-dead") is True
    assert reloaded.retry_dead_letter("missing") is False
    assert reloaded.list_dead_letters() == []
    assert reloaded.size() == 1

    replayed = PersistentTaskQueue(str(qfile)).dequeue_task()
    assert replayed is not None
    assert replayed.task_id == "t-dead"


def test_queue_loads_legacy_list_storage(tmp_path: Path):
    qfile = tmp_path / "queue.json"
    qfile.write_text(
        json.dumps(
            [
                {
                    "priority": 0,
                    "task_id": "legacy",
                    "task": Task(
                        task_id="legacy",
                        source="linear",
                        title="legacy",
                        description="legacy payload",
                        priority=Priority.P0,
                    ).to_dict(),
                }
            ]
        ),
        encoding="utf-8",
    )

    queue = PersistentTaskQueue(str(qfile))

    assert queue.size() == 1
    task = queue.dequeue_task()
    assert task is not None
    assert task.task_id == "legacy"
