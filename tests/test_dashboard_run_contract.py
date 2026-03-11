from bigclaw.dashboard_run_contract import (
    DashboardRunContract,
    DashboardRunContractAudit,
    DashboardRunContractLibrary,
    SchemaField,
    render_dashboard_run_contract_report,
)


def test_dashboard_run_contract_default_bundle_is_release_ready() -> None:
    library = DashboardRunContractLibrary()
    contract = library.build_default_contract()

    audit = library.audit(contract)
    report = render_dashboard_run_contract_report(contract, audit)

    assert audit.release_ready is True
    assert "eng-overview-core-product" in report
    assert '"run_id": "run-204"' in report
    assert "- Release Ready: True" in report


def test_dashboard_run_contract_audit_detects_missing_field_definitions_and_samples() -> None:
    library = DashboardRunContractLibrary()
    contract = library.build_default_contract()
    contract.dashboard_schema.fields = [
        field for field in contract.dashboard_schema.fields if field.name != "summary.success_rate"
    ]
    contract.dashboard_schema.sample.pop("activity")
    contract.run_detail_schema.fields = [
        field for field in contract.run_detail_schema.fields if field.name != "closeout.git_log_stat_output"
    ]
    contract.run_detail_schema.sample["closeout"].pop("git_log_stat_output")

    audit = library.audit(contract)

    assert audit.dashboard_missing_fields == ["summary.success_rate"]
    assert audit.dashboard_sample_gaps == ["activity"]
    assert audit.run_detail_missing_fields == ["closeout.git_log_stat_output"]
    assert audit.run_detail_sample_gaps == ["closeout.git_log_stat_output"]
    assert audit.release_ready is False


def test_dashboard_run_contract_round_trip_preserves_samples_and_audit() -> None:
    library = DashboardRunContractLibrary()
    contract = library.build_default_contract()

    restored = DashboardRunContract.from_dict(contract.to_dict())
    audit = DashboardRunContractAudit.from_dict(library.audit(contract).to_dict())

    assert restored == contract
    assert audit.release_ready is True
    assert any(
        field == SchemaField("dashboard_id", "string", description="Stable dashboard identifier.")
        for field in restored.dashboard_schema.fields
    )
