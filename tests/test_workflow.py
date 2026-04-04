import json
import warnings
from pathlib import Path

from bigclaw import deprecation, orchestration, queue, runtime, scheduler, workflow
from bigclaw.legacy_shim import (
    LEGACY_PYTHON_WRAPPER_NOTICE,
    append_missing_flag,
    build_github_sync_args,
    build_refill_args,
    build_workspace_bootstrap_args,
    build_workspace_validate_args,
    translate_workspace_validate_args,
)
from bigclaw.models import (
    BillingInterval,
    BillingRate,
    BillingSummary,
    FlowRun,
    FlowRunStatus,
    FlowStepRun,
    FlowStepStatus,
    FlowTemplate,
    FlowTemplateStep,
    FlowTrigger,
    Priority,
    RiskAssessment,
    RiskLevel,
    RiskSignal,
    Task,
    TriageLabel,
    TriageRecord,
    TriageStatus,
    UsageRecord,
)
from bigclaw.observability import GitSyncTelemetry, ObservabilityLedger, PullRequestFreshness, RepoSyncAudit, TaskRun
from bigclaw.orchestration import (
    CrossDepartmentOrchestrator,
    PremiumOrchestrationPolicy,
    render_orchestration_plan,
)
from bigclaw.reports import PilotMetric, PilotScorecard
from bigclaw.runtime import ClawWorkerRuntime, SandboxRouter, ToolPolicy, ToolRuntime
from bigclaw.scheduler import RiskScorer, Scheduler, SchedulerDecision
from bigclaw.workflow import AcceptanceGate, WorkflowDefinition, WorkflowEngine, WorkpadJournal


REPO_ROOT = Path("/repo")


def test_workpad_journal_can_replay_and_reload(tmp_path: Path):
    journal = WorkpadJournal(task_id="BIG-402-replay", run_id="run-journal-1")
    journal.record("intake", "recorded", source="local")
    journal.record("execution", "approved", medium="docker")
    output = journal.write(str(tmp_path / "journals" / "run-journal-1.json"))

    loaded = WorkpadJournal.read(output)

    assert loaded.task_id == "BIG-402-replay"
    assert loaded.run_id == "run-journal-1"
    assert loaded.replay() == ["intake:recorded", "execution:approved"]


def test_workflow_definition_parses_and_renders_templates():
    definition = WorkflowDefinition.from_json(
        '{'
        '"name": "release-closeout", '
        '"steps": [{"name": "execute", "kind": "scheduler"}], '
        '"report_path_template": "reports/{task_id}/{run_id}.md", '
        '"journal_path_template": "journals/{workflow}/{run_id}.json", '
        '"validation_evidence": ["pytest"], '
        '"approvals": ["ops-review"]'
        '}'
    )
    task = Task(task_id="BIG-401", source="linear", title="DSL", description="")

    assert definition.steps[0].name == "execute"
    assert definition.render_report_path(task, "run-1") == "reports/BIG-401/run-1.md"
    assert definition.render_journal_path(task, "run-1") == "journals/release-closeout/run-1.json"


def test_workflow_engine_runs_definition_end_to_end(tmp_path: Path):
    definition = WorkflowDefinition.from_dict(
        {
            "name": "acceptance-closeout",
            "steps": [{"name": "execute", "kind": "scheduler"}],
            "report_path_template": str(tmp_path / "reports" / "{task_id}" / "{run_id}.md"),
            "journal_path_template": str(tmp_path / "journals" / "{workflow}" / "{run_id}.json"),
            "validation_evidence": ["pytest", "report-shared"],
        }
    )
    task = Task(
        task_id="BIG-401-flow",
        source="linear",
        title="Run workflow definition",
        description="dsl execution",
        acceptance_criteria=["report-shared"],
        validation_plan=["pytest"],
    )

    result = WorkflowEngine().run_definition(
        task,
        definition=definition,
        run_id="run-dsl-1",
        ledger=ObservabilityLedger(str(tmp_path / "ledger.json")),
    )

    assert result.acceptance.status == "accepted"
    assert Path(definition.render_report_path(task, "run-dsl-1")).exists()
    assert Path(definition.render_journal_path(task, "run-dsl-1")).exists()


def test_workflow_definition_rejects_unknown_step_kind(tmp_path: Path):
    definition = WorkflowDefinition.from_dict(
        {
            "name": "broken-flow",
            "steps": [{"name": "hack", "kind": "unknown-kind"}],
        }
    )
    task = Task(task_id="BIG-401-invalid", source="local", title="invalid", description="")

    try:
        WorkflowEngine().run_definition(
            task,
            definition=definition,
            run_id="run-dsl-invalid",
            ledger=ObservabilityLedger(str(tmp_path / "ledger.json")),
        )
        assert False, "expected ValueError for invalid step kind"
    except ValueError as exc:
        assert "invalid workflow step kind" in str(exc)


def test_workflow_definition_manual_approval_closes_high_risk_task(tmp_path: Path):
    definition = WorkflowDefinition.from_dict(
        {
            "name": "prod-approval",
            "steps": [{"name": "review", "kind": "approval"}],
            "validation_evidence": ["rollback-plan", "integration-test"],
            "approvals": ["release-manager"],
        }
    )
    task = Task(
        task_id="BIG-403-dsl",
        source="linear",
        title="Prod rollout",
        description="needs manual closure",
        risk_level=RiskLevel.HIGH,
        acceptance_criteria=["rollback-plan"],
        validation_plan=["integration-test"],
    )

    result = WorkflowEngine().run_definition(
        task,
        definition=definition,
        run_id="run-dsl-2",
        ledger=ObservabilityLedger(str(tmp_path / "ledger.json")),
    )

    assert result.execution.run.status == "needs-approval"
    assert result.acceptance.status == "accepted"
    assert result.acceptance.approvals == ["release-manager"]


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

    record = WorkflowEngine().scheduler.execute(task, run_id="run-ope-66", ledger=ledger)
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

    record = WorkflowEngine().scheduler.execute(task, run_id="run-ope-66-handoff", ledger=ledger)
    entry = ledger.load()[0]

    assert record.handoff_request is not None
    assert record.handoff_request.target_team == "operations"
    assert any(trace["span"] == "orchestration.handoff" for trace in entry["traces"])
    assert any(audit["action"] == "orchestration.handoff" for audit in entry["audits"])


def test_risk_assessment_round_trip_preserves_signals_and_mitigations() -> None:
    assessment = RiskAssessment(
        assessment_id="risk-1",
        task_id="OPE-130",
        level=RiskLevel.HIGH,
        total_score=75,
        requires_approval=True,
        signals=[
            RiskSignal(
                name="prod-deploy",
                score=20,
                reason="production deployment surface",
                source="scheduler",
                metadata={"tool": "deploy"},
            )
        ],
        mitigations=["security review", "rollback plan"],
        reviewer="ops-review",
        notes="Requires explicit sign-off.",
    )

    payload = assessment.to_dict()
    restored = RiskAssessment.from_dict(payload)

    assert payload["level"] == "high"
    assert restored == assessment
    assert restored.signals[0].metadata == {"tool": "deploy"}


def test_triage_record_round_trip_preserves_queue_labels_and_actions() -> None:
    record = TriageRecord(
        triage_id="triage-1",
        task_id="OPE-130",
        status=TriageStatus.ESCALATED,
        queue="risk-review",
        owner="ops",
        summary="High-risk flow needs billing review",
        labels=[
            TriageLabel(name="risk", confidence=0.9, source="heuristic"),
            TriageLabel(name="billing", confidence=0.8, source="classifier"),
        ],
        related_run_id="run-1",
        escalation_target="finance",
        actions=["route-to-finance", "request-approval"],
    )

    payload = record.to_dict()
    restored = TriageRecord.from_dict(payload)

    assert payload["status"] == "escalated"
    assert restored == record
    assert [label.name for label in restored.labels] == ["risk", "billing"]


def test_flow_template_and_run_round_trip_preserve_steps_and_outputs() -> None:
    template = FlowTemplate(
        template_id="flow-template-1",
        name="Risk Triage Flow",
        version="v1",
        description="Routes risky work through triage and approval.",
        trigger=FlowTrigger.EVENT,
        default_risk=RiskLevel.MEDIUM,
        steps=[
            FlowTemplateStep(
                step_id="triage",
                name="Triage",
                kind="review",
                required_tools=["browser"],
                approvals=["ops"],
                metadata={"lane": "risk"},
            ),
            FlowTemplateStep(
                step_id="approve",
                name="Approval",
                kind="approval",
                approvals=["security"],
            ),
        ],
        tags=["risk", "triage"],
    )
    run = FlowRun(
        run_id="flow-run-1",
        template_id=template.template_id,
        task_id="OPE-130",
        status=FlowRunStatus.RUNNING,
        triggered_by="scheduler",
        started_at="2026-03-11T10:00:00Z",
        steps=[
            FlowStepRun(
                step_id="triage",
                status=FlowStepStatus.SUCCEEDED,
                actor="ops",
                started_at="2026-03-11T10:00:00Z",
                completed_at="2026-03-11T10:02:00Z",
                output={"decision": "escalate"},
            ),
            FlowStepRun(
                step_id="approve",
                status=FlowStepStatus.RUNNING,
                actor="security",
            ),
        ],
        outputs={"ticket": "SEC-42"},
        approval_refs=["security-review"],
    )

    template_payload = template.to_dict()
    run_payload = run.to_dict()

    assert FlowTemplate.from_dict(template_payload) == template
    assert FlowRun.from_dict(run_payload) == run
    assert run_payload["steps"][0]["status"] == "succeeded"
    assert template_payload["trigger"] == "event"


def test_billing_summary_round_trip_preserves_rates_and_usage() -> None:
    summary = BillingSummary(
        statement_id="bill-1",
        account_id="acct-1",
        billing_period="2026-03",
        rates=[
            BillingRate(
                metric="orchestration-run",
                interval=BillingInterval.MONTHLY,
                included_units=100,
                unit_price_usd=0.0,
                overage_price_usd=1.5,
            )
        ],
        usage=[
            UsageRecord(
                record_id="usage-1",
                account_id="acct-1",
                metric="orchestration-run",
                quantity=124,
                period="2026-03",
                run_id="flow-run-1",
                unit="run",
                metadata={"source": "workflow-engine"},
            )
        ],
        subtotal_usd=0.0,
        overage_usd=36.0,
        total_usd=36.0,
    )

    payload = summary.to_dict()
    restored = BillingSummary.from_dict(payload)

    assert payload["rates"][0]["interval"] == "monthly"
    assert restored == summary
    assert restored.usage[0].metadata["source"] == "workflow-engine"


def test_sandbox_router_maps_execution_media():
    router = SandboxRouter()

    assert router.profile_for("docker").isolation == "container"
    assert router.profile_for("browser").network_access == "enabled"
    assert router.profile_for("vm").filesystem_access == "workspace-write"
    assert router.profile_for("unknown").medium == "none"


def test_tool_runtime_blocks_disallowed_tool_and_audits(tmp_path: Path):
    task = Task(task_id="BIG-303", source="linear", title="tool policy", description="")
    run = TaskRun.from_task(task, run_id="run-tool-1", medium="docker")
    runtime_policy = ToolRuntime(policy=ToolPolicy(allowed_tools=["github"], blocked_tools=["browser"]))

    result = runtime_policy.invoke("browser", action="execute", payload={"url": "https://example.com"}, run=run)

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


def test_big301_worker_lifecycle_is_stable_with_multiple_tools() -> None:
    task = Task(
        task_id="BIG-301-matrix",
        source="github",
        title="worker lifecycle matrix",
        description="validate stable lifecycle",
        required_tools=["github", "browser"],
    )
    run = TaskRun.from_task(task, run_id="run-big301-matrix", medium="docker")
    runtime_matrix = ToolRuntime(
        handlers={
            "github": lambda action, payload: f"{action}:{payload.get('repo', 'none')}",
            "browser": lambda action, payload: f"{action}:{payload.get('url', 'none')}",
        }
    )
    worker = ClawWorkerRuntime(tool_runtime=runtime_matrix)

    result = worker.execute(
        task,
        decision=type("Decision", (), {"medium": "docker", "approved": True, "reason": "ok"})(),
        run=run,
        tool_payloads={"github": {"repo": "OpenAGIs/BigClaw"}, "browser": {"url": "https://example.com"}},
    )

    assert len(result.tool_results) == 2
    assert all(item.success for item in result.tool_results)
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


def test_warn_legacy_runtime_surface_emits_deprecation_warning():
    with warnings.catch_warnings(record=True) as caught:
        warnings.simplefilter("always")
        message = deprecation.warn_legacy_runtime_surface(
            "python -m bigclaw",
            "bash scripts/ops/bigclawctl",
        )

    assert "frozen for migration-only use" in message
    assert caught
    assert issubclass(caught[0].category, DeprecationWarning)
    assert "bash scripts/ops/bigclawctl" in str(caught[0].message)


def test_legacy_runtime_modules_expose_go_mainline_replacements():
    assert runtime.GO_MAINLINE_REPLACEMENT == "bigclaw-go/internal/worker/runtime.go"
    assert scheduler.GO_MAINLINE_REPLACEMENT == "bigclaw-go/internal/scheduler/scheduler.go"
    assert workflow.GO_MAINLINE_REPLACEMENT == "bigclaw-go/internal/workflow/engine.go"
    assert orchestration.GO_MAINLINE_REPLACEMENT == "bigclaw-go/internal/workflow/orchestration.go"
    assert queue.GO_MAINLINE_REPLACEMENT == "bigclaw-go/internal/queue/queue.go"
    assert "sole implementation mainline" in runtime.LEGACY_MAINLINE_STATUS


def test_big302_risk_routes_to_expected_sandbox_mediums() -> None:
    scheduler_runtime = Scheduler()

    low = Task(task_id="low", source="local", title="low", description="", risk_level=RiskLevel.LOW)
    high = Task(task_id="high", source="local", title="high", description="", risk_level=RiskLevel.HIGH)
    browser = Task(
        task_id="browser",
        source="local",
        title="browser",
        description="",
        required_tools=["browser"],
        risk_level=RiskLevel.MEDIUM,
    )

    assert scheduler_runtime.decide(low).medium == "docker"
    assert scheduler_runtime.decide(high).medium == "vm"
    assert scheduler_runtime.decide(browser).medium in {"browser", "docker"}


def test_big303_tool_runtime_policy_and_audit_chain() -> None:
    task = Task(task_id="BIG-303-matrix", source="local", title="tool policy", description="", required_tools=["github", "browser"])
    run = TaskRun.from_task(task, run_id="run-big303-matrix", medium="docker")

    runtime_matrix = ToolRuntime(
        policy=ToolPolicy(allowed_tools=["github"], blocked_tools=["browser"]),
        handlers={"github": lambda action, payload: "ok"},
    )

    allow = runtime_matrix.invoke("github", action="execute", payload={"repo": "OpenAGIs/BigClaw"}, run=run)
    block = runtime_matrix.invoke("browser", action="execute", payload={"url": "https://example.com"}, run=run)

    assert allow.success is True
    assert block.success is False
    outcomes = [audit.outcome for audit in run.audits if audit.action == "tool.invoke"]
    assert "success" in outcomes
    assert "blocked" in outcomes


def test_append_missing_flag_preserves_existing_values() -> None:
    assert append_missing_flag(["--repo-url", "ssh://example/repo.git"], "--repo-url", "git@github.com:OpenAGIs/BigClaw.git") == [
        "--repo-url",
        "ssh://example/repo.git",
    ]
    assert append_missing_flag(["--cache-key=openagis-bigclaw"], "--cache-key", "other") == ["--cache-key=openagis-bigclaw"]


def test_workspace_bootstrap_wrapper_injects_go_defaults() -> None:
    argv = build_workspace_bootstrap_args(REPO_ROOT, ["--workspace", "/tmp/ws"])
    assert argv[:4] == ["bash", "/repo/scripts/ops/bigclawctl", "workspace", "bootstrap"]
    assert "--repo-url" in argv
    assert "git@github.com:OpenAGIs/BigClaw.git" in argv
    assert "--cache-key" in argv
    assert "openagis-bigclaw" in argv


def test_workspace_validate_wrapper_translates_legacy_flags() -> None:
    translated = translate_workspace_validate_args(
        [
            "--repo-url",
            "git@github.com:OpenAGIs/BigClaw.git",
            "--workspace-root",
            "/tmp/ws",
            "--issues",
            "BIG-1",
            "BIG-2",
            "--report-file",
            "/tmp/report.md",
            "--no-cleanup",
            "--json",
        ]
    )
    assert translated == [
        "--repo-url",
        "git@github.com:OpenAGIs/BigClaw.git",
        "--workspace-root",
        "/tmp/ws",
        "--issues",
        "BIG-1,BIG-2",
        "--report",
        "/tmp/report.md",
        "--cleanup=false",
        "--json",
    ]
    argv = build_workspace_validate_args(REPO_ROOT, ["--issues", "BIG-1", "BIG-2"])
    assert argv[:4] == ["bash", "/repo/scripts/ops/bigclawctl", "workspace", "validate"]
    assert argv[4:] == ["--issues", "BIG-1,BIG-2"]


def test_github_sync_and_refill_wrappers_target_go_shim() -> None:
    assert build_github_sync_args(REPO_ROOT, ["status", "--json"]) == [
        "bash",
        "/repo/scripts/ops/bigclawctl",
        "github-sync",
        "status",
        "--json",
    ]
    assert build_refill_args(REPO_ROOT, ["--apply"]) == ["bash", "/repo/scripts/ops/bigclawctl", "refill", "--apply"]
    assert "compatibility shim during migration" in LEGACY_PYTHON_WRAPPER_NOTICE


def test_scheduler_high_risk_requires_approval() -> None:
    scheduler_runtime = Scheduler()
    task = Task(task_id="x", source="jira", title="prod op", description="", risk_level=RiskLevel.HIGH)
    decision = scheduler_runtime.decide(task)
    assert decision.medium == "vm"
    assert decision.approved is False


def test_scheduler_browser_task_routes_browser() -> None:
    scheduler_runtime = Scheduler()
    task = Task(task_id="y", source="github", title="ui test", description="", required_tools=["browser"])
    decision = scheduler_runtime.decide(task)
    assert decision.medium == "browser"
    assert decision.approved is True


def test_scheduler_over_budget_degrades_browser_task_to_docker() -> None:
    scheduler_runtime = Scheduler()
    task = Task(
        task_id="z",
        source="github",
        title="budgeted ui test",
        description="",
        required_tools=["browser"],
        budget=15.0,
    )

    decision = scheduler_runtime.decide(task)

    assert decision.medium == "docker"
    assert decision.approved is True
    assert "budget degraded browser route to docker" in decision.reason


def test_scheduler_over_budget_pauses_task() -> None:
    scheduler_runtime = Scheduler()
    task = Task(task_id="b", source="linear", title="tiny budget", description="", budget=5.0)

    decision = scheduler_runtime.decide(task)

    assert decision.medium == "none"
    assert decision.approved is False
    assert decision.reason == "paused: budget 5.0 below required docker budget 10.0"


def test_risk_scorer_keeps_simple_low_risk_work_low() -> None:
    score = RiskScorer().score_task(Task(task_id="BIG-902-low", source="linear", title="doc cleanup", description=""))

    assert score.total == 0
    assert score.level == RiskLevel.LOW
    assert score.requires_approval is False


def test_risk_scorer_elevates_prod_browser_work() -> None:
    score = RiskScorer().score_task(
        Task(
            task_id="BIG-902-mid",
            source="linear",
            title="release verification",
            description="prod browser change",
            labels=["prod"],
            priority=Priority.P0,
            required_tools=["browser"],
        )
    )

    assert score.total == 40
    assert score.level == RiskLevel.MEDIUM
    assert score.requires_approval is False


def test_scheduler_uses_risk_score_to_require_approval(tmp_path: Path) -> None:
    ledger = ObservabilityLedger(str(tmp_path / "ledger.json"))
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

    assert record.risk_score is not None
    assert record.risk_score.total == 70
    assert record.risk_score.level == RiskLevel.HIGH
    assert record.decision.medium == "vm"
    assert record.decision.approved is False
    assert any(trace["span"] == "risk.score" for trace in entry["traces"])
    assert any(audit["action"] == "risk.score" for audit in entry["audits"])
