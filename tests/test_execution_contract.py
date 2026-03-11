from bigclaw.execution_contract import (
    AuditPolicy,
    ExecutionApiSpec,
    ExecutionContract,
    ExecutionContractAudit,
    ExecutionContractLibrary,
    ExecutionField,
    ExecutionModel,
    ExecutionPermission,
    ExecutionPermissionMatrix,
    MetricDefinition,
    render_execution_contract_report,
)


def build_contract() -> ExecutionContract:
    return ExecutionContract(
        contract_id="BIG-EPIC-18",
        version="v4.0",
        models=[
            ExecutionModel(
                name="ExecutionRequest",
                owner="runtime",
                fields=[
                    ExecutionField("task_id", "string"),
                    ExecutionField("actor", "string"),
                    ExecutionField("requested_tools", "string[]"),
                    ExecutionField("approval_token", "string", required=False),
                ],
            ),
            ExecutionModel(
                name="ExecutionResponse",
                owner="runtime",
                fields=[
                    ExecutionField("run_id", "string"),
                    ExecutionField("status", "string"),
                    ExecutionField("sandbox_profile", "string"),
                ],
            ),
        ],
        apis=[
            ExecutionApiSpec(
                name="start_execution",
                method="POST",
                path="/execution/runs",
                request_model="ExecutionRequest",
                response_model="ExecutionResponse",
                required_permission="execution.run.write",
                emitted_audits=["execution.run.started", "execution.permission.checked"],
                emitted_metrics=["execution.request.count", "execution.duration.ms"],
            )
        ],
        permissions=[
            ExecutionPermission(
                name="execution.run.write",
                resource="execution-run",
                actions=["create"],
                scopes=["project", "workspace"],
            )
        ],
        metrics=[
            MetricDefinition("execution.request.count", "count", owner="runtime"),
            MetricDefinition("execution.duration.ms", "ms", owner="runtime"),
        ],
        audit_policies=[
            AuditPolicy(
                event_type="execution.run.started",
                required_fields=["task_id", "run_id", "actor"],
                retention_days=180,
                severity="info",
            ),
            AuditPolicy(
                event_type="execution.permission.checked",
                required_fields=["task_id", "actor", "permission", "allowed"],
                retention_days=180,
                severity="info",
            ),
        ],
    )


def test_execution_contract_audit_accepts_well_formed_contract() -> None:
    contract = build_contract()

    audit = ExecutionContractLibrary().audit(contract)
    report = render_execution_contract_report(contract, audit)

    assert audit.release_ready is True
    assert audit.readiness_score == 100.0
    assert "- Release Ready: True" in report
    assert "POST /execution/runs" in report


def test_execution_contract_audit_surfaces_contract_gaps() -> None:
    contract = build_contract()
    contract.models[0] = ExecutionModel(
        name="ExecutionRequest",
        owner="runtime",
        fields=[ExecutionField("task_id", "string")],
    )
    contract.apis[0] = ExecutionApiSpec(
        name="start_execution",
        method="POST",
        path="/execution/runs",
        request_model="ExecutionRequest",
        response_model="MissingResponse",
        required_permission="execution.run.approve",
        emitted_audits=["execution.run.finished"],
        emitted_metrics=["execution.queue.depth"],
    )
    contract.audit_policies[0] = AuditPolicy(
        event_type="execution.run.started",
        required_fields=["task_id"],
        retention_days=7,
        severity="info",
    )

    audit = ExecutionContractLibrary().audit(contract)

    assert audit.models_missing_required_fields == {
        "ExecutionRequest": ["actor", "requested_tools"]
    }
    assert audit.undefined_model_refs == {"start_execution": ["MissingResponse"]}
    assert audit.undefined_permissions == {"start_execution": "execution.run.approve"}
    assert audit.undefined_metrics == {"start_execution": ["execution.queue.depth"]}
    assert audit.undefined_audit_events == {"start_execution": ["execution.run.finished"]}
    assert audit.audit_policies_below_retention == ["execution.run.started"]
    assert audit.release_ready is False


def test_execution_contract_round_trip_and_permission_matrix() -> None:
    contract = build_contract()
    audit = ExecutionContractAudit.from_dict(ExecutionContractLibrary().audit(contract).to_dict())
    restored = ExecutionContract.from_dict(contract.to_dict())
    decision = ExecutionPermissionMatrix(restored.permissions).evaluate(
        ["execution.run.write", "missing.permission"],
        ["execution.run.write", "unknown.permission"],
    )

    assert restored == contract
    assert audit.release_ready is True
    assert decision.allowed is False
    assert decision.granted_permissions == ["execution.run.write"]
    assert decision.missing_permissions == ["missing.permission"]
