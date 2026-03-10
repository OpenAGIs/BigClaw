from pathlib import Path

from bigclaw.reports import (
    CrossTeamFlowOverview,
    CrossTeamFlowSnapshot,
    PilotMetric,
    PilotPortfolio,
    PilotScorecard,
    build_auto_triage_center,
    build_cross_team_flow_overview,
    evaluate_issue_closure,
    render_auto_triage_center_report,
    render_cross_team_flow_overview_page,
    render_cross_team_flow_overview_report,
    render_issue_validation_report,
    render_pilot_portfolio_report,
    render_pilot_scorecard,
    validation_report_exists,
    write_report,
)
from bigclaw.observability import TaskRun
from bigclaw.models import RiskLevel, Task
from bigclaw.orchestration import CrossDepartmentOrchestrator, PremiumOrchestrationPolicy


def test_render_and_write_report(tmp_path: Path):
    content = render_issue_validation_report("BIG-101", "v0.1", "sandbox", "pass")
    out = tmp_path / "report.md"
    write_report(str(out), content)
    assert out.exists()
    text = out.read_text()
    assert "BIG-101" in text
    assert "pass" in text


def test_render_pilot_scorecard_includes_roi_and_recommendation():
    scorecard = PilotScorecard(
        issue_id="OPE-60",
        customer="Design Partner A",
        period="2026-Q2",
        metrics=[
            PilotMetric(
                name="Automation coverage",
                baseline=35,
                current=82,
                target=80,
                unit="%",
            ),
            PilotMetric(
                name="Manual review time",
                baseline=12,
                current=4,
                target=5,
                unit="h",
                higher_is_better=False,
            ),
        ],
        monthly_benefit=12000,
        monthly_cost=2500,
        implementation_cost=18000,
        benchmark_score=96,
        benchmark_passed=True,
    )

    content = render_pilot_scorecard(scorecard)

    assert scorecard.metrics_met == 2
    assert scorecard.recommendation == "go"
    assert scorecard.payback_months == 1.9
    assert "Annualized ROI: 200.0%" in content
    assert "Recommendation: go" in content
    assert "Benchmark Score: 96" in content
    assert "Automation coverage" in content


def test_pilot_scorecard_returns_hold_when_value_is_negative():
    scorecard = PilotScorecard(
        issue_id="OPE-60",
        customer="Design Partner B",
        period="2026-Q2",
        metrics=[PilotMetric(name="Backlog aging", baseline=5, current=7, target=4, unit="d", higher_is_better=False)],
        monthly_benefit=1000,
        monthly_cost=3000,
        implementation_cost=12000,
        benchmark_passed=False,
    )

    assert scorecard.monthly_net_value == -2000
    assert scorecard.payback_months is None
    assert scorecard.recommendation == "hold"


def test_issue_closure_requires_non_empty_validation_report(tmp_path: Path):
    report_path = tmp_path / "validation.md"

    decision = evaluate_issue_closure("BIG-602", str(report_path))

    assert decision.allowed is False
    assert decision.reason == "validation report required before closing issue"
    assert validation_report_exists(str(report_path)) is False


def test_issue_closure_blocks_failed_validation_report(tmp_path: Path):
    report_path = tmp_path / "validation.md"
    write_report(str(report_path), "# Validation\n\nfailed")

    decision = evaluate_issue_closure("BIG-602", str(report_path), validation_passed=False)

    assert decision.allowed is False
    assert decision.reason == "validation failed; issue must remain open"
    assert validation_report_exists(str(report_path)) is True


def test_issue_closure_allows_completed_validation_report(tmp_path: Path):
    report_path = tmp_path / "validation.md"
    content = render_issue_validation_report("BIG-602", "v0.1", "sandbox", "pass")
    write_report(str(report_path), content)

    decision = evaluate_issue_closure("BIG-602", str(report_path), validation_passed=True)

    assert decision.allowed is True
    assert decision.reason == "validation report present; issue can be closed"
    assert decision.report_path == str(report_path)


def test_render_pilot_portfolio_report_summarizes_commercial_readiness():
    portfolio = PilotPortfolio(
        name="Design Partners",
        period="2026-H1",
        scorecards=[
            PilotScorecard(
                issue_id="OPE-60",
                customer="Partner A",
                period="2026-Q2",
                metrics=[PilotMetric(name="Coverage", baseline=40, current=85, target=80, unit="%")],
                monthly_benefit=15000,
                monthly_cost=3000,
                implementation_cost=18000,
                benchmark_score=97,
                benchmark_passed=True,
            ),
            PilotScorecard(
                issue_id="OPE-61",
                customer="Partner B",
                period="2026-Q2",
                metrics=[PilotMetric(name="Cycle time", baseline=12, current=7, target=5, unit="h", higher_is_better=False)],
                monthly_benefit=9000,
                monthly_cost=2500,
                implementation_cost=12000,
                benchmark_score=88,
                benchmark_passed=True,
            ),
        ],
    )

    content = render_pilot_portfolio_report(portfolio)

    assert portfolio.total_monthly_net_value == 18500
    assert portfolio.average_roi == 195.2
    assert portfolio.recommendation_counts == {"go": 1, "iterate": 1, "hold": 0}
    assert portfolio.recommendation == "continue"
    assert "Recommendation Mix: go=1 iterate=1 hold=0" in content
    assert "Partner A: recommendation=go" in content
    assert "Partner B: recommendation=iterate" in content


def test_auto_triage_center_prioritizes_failed_and_pending_runs():
    approval_task = Task(task_id="OPE-76-risk", source="linear", title="Prod approval", description="")
    approval_run = TaskRun.from_task(approval_task, run_id="run-risk", medium="vm")
    approval_run.trace("scheduler.decide", "pending")
    approval_run.audit(
        "scheduler.decision",
        "scheduler",
        "pending",
        reason="requires approval for high-risk task",
    )
    approval_run.finalize("needs-approval", "requires approval for high-risk task")

    failed_task = Task(task_id="OPE-76-browser", source="linear", title="Replay browser task", description="")
    failed_run = TaskRun.from_task(failed_task, run_id="run-browser", medium="browser")
    failed_run.trace("runtime.execute", "failed")
    failed_run.audit("runtime.execute", "worker", "failed", reason="browser session crashed")
    failed_run.finalize("failed", "browser session crashed")

    healthy_task = Task(task_id="OPE-76-ok", source="linear", title="Healthy run", description="")
    healthy_run = TaskRun.from_task(healthy_task, run_id="run-ok", medium="docker")
    healthy_run.trace("scheduler.decide", "ok")
    healthy_run.audit("scheduler.decision", "scheduler", "approved", reason="default low risk path")
    healthy_run.finalize("approved", "default low risk path")

    center = build_auto_triage_center(
        [healthy_run, approval_run, failed_run],
        name="Engineering Ops",
        period="2026-03-10",
    )
    report = render_auto_triage_center_report(center, total_runs=3)

    assert center.flagged_runs == 2
    assert center.severity_counts == {"critical": 1, "high": 1, "medium": 0}
    assert center.owner_counts == {"security": 1, "engineering": 1, "operations": 0}
    assert center.recommendation == "immediate-attention"
    assert [finding.run_id for finding in center.findings] == ["run-browser", "run-risk"]
    assert center.findings[0].next_action == "replay run and inspect tool failures"
    assert center.findings[1].next_action == "request approval and queue security review"
    assert "Flagged Runs: 2" in report
    assert "Severity Mix: critical=1 high=1 medium=0" in report
    assert "run-browser: severity=critical owner=engineering status=failed" in report
    assert "run-risk: severity=high owner=security status=needs-approval" in report


def test_cross_team_flow_overview_summarizes_handoffs_and_blockers() -> None:
    orchestrator = CrossDepartmentOrchestrator()

    approval_task = Task(
        task_id="OPE-83-approval",
        source="linear",
        title="Customer analytics rollout approval",
        description="Stakeholder rollout needs security approval",
        labels=["data", "customer", "premium"],
        risk_level=RiskLevel.HIGH,
        required_tools=["browser", "sql"],
        acceptance_criteria=["approval recorded"],
    )
    approval_plan = orchestrator.plan(approval_task)
    approval_run = TaskRun.from_task(approval_task, run_id="run-approval", medium="browser")
    approval_run.finalize("needs-approval", "waiting on security review before customer rollout")

    blocked_task = Task(
        task_id="OPE-83-blocked",
        source="jira",
        title="Warehouse launch coordination",
        description="Customer-ready release with data validation",
        labels=["data", "customer"],
        required_tools=["sql"],
        risk_level=RiskLevel.HIGH,
    )
    blocked_plan = orchestrator.plan(blocked_task)
    constrained_plan, blocked_policy = PremiumOrchestrationPolicy().apply(blocked_task, blocked_plan)

    healthy_task = Task(
        task_id="OPE-83-healthy",
        source="linear",
        title="Cross-team browser rollout",
        description="Program managed release",
        labels=["ops"],
        required_tools=["browser"],
    )
    healthy_plan = orchestrator.plan(healthy_task)
    healthy_run = TaskRun.from_task(healthy_task, run_id="run-healthy", medium="browser")
    healthy_run.finalize("approved", "handoff completed and rollout is on track")

    overview = build_cross_team_flow_overview(
        [
            CrossTeamFlowSnapshot(plan=approval_plan, run=approval_run),
            CrossTeamFlowSnapshot(plan=constrained_plan, policy=blocked_policy, source="jira"),
            CrossTeamFlowSnapshot(plan=healthy_plan, run=healthy_run),
        ],
        name="Program Rollouts",
        period="2026-W11",
    )

    report = render_cross_team_flow_overview_report(overview)
    page = render_cross_team_flow_overview_page(overview)

    assert isinstance(overview, CrossTeamFlowOverview)
    assert overview.total_flows == 3
    assert overview.cross_team_flows == 3
    assert overview.approval_queue_depth == 1
    assert overview.blocked_flows == 1
    assert overview.department_counts == {
        "customer-success": 1,
        "data": 1,
        "engineering": 3,
        "operations": 3,
        "security": 1,
    }
    assert overview.source_counts == {"jira": 1, "linear": 2}
    assert overview.status_counts == {"approved": 1, "needs-approval": 1, "planned": 1}
    assert [flow.task_id for flow in overview.at_risk_flows] == ["OPE-83-approval", "OPE-83-blocked"]
    assert "# Cross-Team Flow Overview" in report
    assert "- Approval Queue Depth: 1" in report
    assert "OPE-83-blocked: source=jira" in report
    assert "premium tier required to unblock security, data, customer-success" in report
    assert "Cross-Team Flow Overview" in page
    assert "upgrade tier to unlock security, data, customer-success" in page
    assert "waiting on security review before customer rollout" in page
