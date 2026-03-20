from bigclaw.pilot import PilotImplementationResult, PilotKPI, render_pilot_implementation_report


def test_big701_pilot_ready_when_kpis_pass_and_no_incidents():
    result = PilotImplementationResult(
        customer="Design Partner A",
        environment="production",
        production_runs=12,
        incidents=0,
        kpis=[
            PilotKPI(name="automation-coverage", target=80, actual=86),
            PilotKPI(name="lead-time-hours", target=6, actual=5, higher_is_better=False),
        ],
    )

    assert result.kpi_pass_rate == 100.0
    assert result.ready is True


def test_big701_render_pilot_report_contains_readiness_fields():
    result = PilotImplementationResult(
        customer="Design Partner B",
        environment="staging",
        production_runs=0,
        incidents=1,
        kpis=[PilotKPI(name="automation-coverage", target=80, actual=72)],
    )

    report = render_pilot_implementation_report(result)

    assert "Pilot Implementation Report" in report
    assert "Ready: False" in report
    assert "KPI Pass Rate: 0.0%" in report
