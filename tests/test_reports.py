from pathlib import Path

from bigclaw.reports import (
    PilotMetric,
    PilotPortfolio,
    PilotScorecard,
    build_weekly_operations_report,
    evaluate_issue_closure,
    generate_weekly_operations_report,
    render_issue_validation_report,
    render_pilot_portfolio_report,
    render_pilot_scorecard,
    render_weekly_operations_report,
    validation_report_exists,
    write_report,
)


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


def test_build_weekly_operations_report_summarizes_runs_in_period():
    runs = [
        {
            "run_id": "run-1",
            "task_id": "OPE-74",
            "title": "SLA dashboard refresh",
            "source": "linear",
            "medium": "browser",
            "status": "approved",
            "started_at": "2026-03-03T09:00:00Z",
            "summary": "browser automation task",
            "artifacts": [{"name": "dashboard", "kind": "page", "path": "reports/run-1.html"}],
            "audits": [{"action": "scheduler.decision", "actor": "scheduler", "outcome": "approved", "details": {}}],
            "traces": [],
        },
        {
            "run_id": "run-2",
            "task_id": "OPE-75",
            "title": "Risk review",
            "source": "jira",
            "medium": "vm",
            "status": "needs-approval",
            "started_at": "2026-03-05T13:30:00Z",
            "summary": "requires approval for high-risk task",
            "artifacts": [],
            "audits": [
                {
                    "action": "scheduler.decision",
                    "actor": "scheduler",
                    "outcome": "pending",
                    "details": {"reason": "requires approval for high-risk task"},
                }
            ],
            "traces": [{"span": "scheduler.decide", "status": "pending"}],
        },
        {
            "run_id": "run-3",
            "task_id": "OPE-77",
            "title": "Regression analytics rerun",
            "source": "linear",
            "medium": "docker",
            "status": "succeeded",
            "started_at": "2026-03-08T18:45:00Z",
            "summary": "replay completed",
            "artifacts": [{"name": "replay", "kind": "report", "path": "reports/run-3.md"}],
            "audits": [{"action": "replay", "actor": "engine", "outcome": "success", "details": {}}],
            "traces": [{"span": "replay", "status": "ok"}],
        },
        {
            "run_id": "old-run",
            "task_id": "OPE-69",
            "title": "Out of range",
            "source": "linear",
            "medium": "docker",
            "status": "approved",
            "started_at": "2026-02-25T08:00:00Z",
            "summary": "ignored",
            "artifacts": [],
            "audits": [],
            "traces": [],
        },
    ]

    report = build_weekly_operations_report(
        runs,
        team="OpenAGI",
        period_start="2026-03-03T00:00:00Z",
        period_end="2026-03-09T23:59:59Z",
        generated_at="2026-03-10T00:00:00Z",
    )

    assert report.total_runs == 3
    assert report.successful_runs == 2
    assert report.success_rate == 66.7
    assert report.approvals_pending == 1
    assert report.status_counts == {"approved": 1, "needs-approval": 1, "succeeded": 1}
    assert report.source_counts == {"jira": 1, "linear": 2}
    assert report.medium_counts == {"browser": 1, "docker": 1, "vm": 1}
    assert report.daily_volume == {"2026-03-03": 1, "2026-03-05": 1, "2026-03-08": 1}
    assert report.focus_items[0].run_id == "run-2"
    assert report.focus_items[0].reason == "requires approval for high-risk task"

    content = render_weekly_operations_report(report)

    assert "# Weekly Operations Report" in content
    assert "Success Rate: 66.7%" in content
    assert "Approvals Pending: 1" in content
    assert "OPE-75/run-2: status=needs-approval" in content


def test_generate_weekly_operations_report_writes_markdown(tmp_path: Path):
    out = tmp_path / "reports" / "weekly-ops.md"

    report = generate_weekly_operations_report(
        runs=[
            {
                "run_id": "run-9",
                "task_id": "OPE-78",
                "title": "Generate weekly report",
                "source": "linear",
                "medium": "docker",
                "status": "completed",
                "started_at": "2026-03-09T09:00:00Z",
                "summary": "report generated",
                "artifacts": [],
                "audits": [],
                "traces": [],
            }
        ],
        report_path=str(out),
        team="OpenAGI",
        period_start="2026-03-09T00:00:00Z",
        period_end="2026-03-09T23:59:59Z",
        generated_at="2026-03-10T00:00:00Z",
    )

    assert report.total_runs == 1
    assert out.exists()
    assert "Generate weekly report" not in out.read_text()
    assert "Total Runs: 1" in out.read_text()
