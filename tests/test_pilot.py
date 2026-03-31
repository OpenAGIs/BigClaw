from bigclaw.pilot import PilotImplementationResult, PilotKPI, render_pilot_implementation_report


def test_pilot_ready_requires_runs_no_incidents_and_kpi_threshold() -> None:
    result = PilotImplementationResult(
        customer="Acme",
        environment="prod",
        production_runs=3,
        incidents=0,
        kpis=[
            PilotKPI(name="accuracy", target=95.0, actual=96.0),
            PilotKPI(name="latency", target=250.0, actual=220.0, higher_is_better=False),
            PilotKPI(name="coverage", target=90.0, actual=80.0),
            PilotKPI(name="satisfaction", target=4.0, actual=4.5),
            PilotKPI(name="adoption", target=70.0, actual=72.0),
        ],
    )

    assert result.kpi_pass_rate == 80.0
    assert result.ready is True


def test_render_pilot_implementation_report_handles_empty_kpis() -> None:
    result = PilotImplementationResult(customer="Acme", environment="staging")

    report = render_pilot_implementation_report(result)

    assert "# Pilot Implementation Report" in report
    assert "- KPI Pass Rate: 0.0%" in report
    assert "- none" in report
