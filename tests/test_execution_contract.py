from bigclaw.execution_contract import (
    AuditPolicy,
    build_operations_api_contract,
    ExecutionApiSpec,
    ExecutionContract,
    ExecutionContractAudit,
    ExecutionContractLibrary,
    ExecutionField,
    ExecutionModel,
    ExecutionPermission,
    ExecutionPermissionMatrix,
    ExecutionRole,
    MetricDefinition,
    render_execution_contract_report,
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
    RiskAssessment,
    RiskLevel,
    RiskSignal,
    TriageLabel,
    TriageRecord,
    TriageStatus,
    UsageRecord,
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
            ),
            ExecutionPermission(
                name="execution.run.approve",
                resource="execution-run",
                actions=["approve"],
                scopes=["workspace"],
            ),
            ExecutionPermission(
                name="execution.audit.read",
                resource="execution-audit",
                actions=["read"],
                scopes=["workspace", "portfolio"],
            ),
            ExecutionPermission(
                name="execution.orchestration.manage",
                resource="orchestration-plan",
                actions=["read", "update"],
                scopes=["cross-team"],
            ),
        ],
        roles=[
            ExecutionRole(
                name="eng-lead",
                personas=["Eng Lead"],
                granted_permissions=["execution.run.write", "execution.run.approve"],
                scope_bindings=["project"],
                escalation_target="vp-eng",
            ),
            ExecutionRole(
                name="platform-admin",
                personas=["Platform Admin"],
                granted_permissions=["execution.run.write", "execution.audit.read"],
                scope_bindings=["workspace"],
                escalation_target="vp-eng",
            ),
            ExecutionRole(
                name="vp-eng",
                personas=["VP Eng"],
                granted_permissions=["execution.run.approve", "execution.audit.read"],
                scope_bindings=["portfolio", "workspace"],
                escalation_target="none",
            ),
            ExecutionRole(
                name="cross-team-operator",
                personas=["Cross-Team Operator"],
                granted_permissions=["execution.run.write", "execution.orchestration.manage"],
                scope_bindings=["cross-team", "project"],
                escalation_target="eng-lead",
            ),
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
    contract.roles = [
        ExecutionRole(
            name="eng-lead",
            personas=[],
            granted_permissions=[],
            scope_bindings=[],
            escalation_target="",
        ),
        ExecutionRole(
            name="platform-admin",
            personas=["Platform Admin"],
            granted_permissions=["execution.audit.override"],
            scope_bindings=["workspace"],
            escalation_target="vp-eng",
        ),
    ]

    audit = ExecutionContractLibrary().audit(contract)

    assert audit.models_missing_required_fields == {
        "ExecutionRequest": ["actor", "requested_tools"]
    }
    assert audit.undefined_model_refs == {"start_execution": ["MissingResponse"]}
    assert audit.undefined_permissions == {}
    assert audit.missing_roles == ["cross-team-operator", "vp-eng"]
    assert audit.roles_missing_personas == ["eng-lead"]
    assert audit.roles_missing_scope_bindings == ["eng-lead"]
    assert audit.roles_missing_escalation_targets == ["eng-lead"]
    assert audit.roles_missing_permissions == ["eng-lead"]
    assert audit.undefined_role_permissions == {"platform-admin": ["execution.audit.override"]}
    assert audit.apis_without_role_coverage == ["start_execution"]
    assert audit.permissions_without_roles == [
        "execution.audit.read",
        "execution.orchestration.manage",
        "execution.run.approve",
        "execution.run.write",
    ]
    assert audit.undefined_metrics == {"start_execution": ["execution.queue.depth"]}
    assert audit.undefined_audit_events == {"start_execution": ["execution.run.finished"]}
    assert audit.audit_policies_below_retention == ["execution.run.started"]
    assert audit.release_ready is False


def test_execution_contract_round_trip_and_permission_matrix() -> None:
    contract = build_contract()
    audit = ExecutionContractAudit.from_dict(ExecutionContractLibrary().audit(contract).to_dict())
    restored = ExecutionContract.from_dict(contract.to_dict())
    matrix = ExecutionPermissionMatrix(restored.permissions, restored.roles)
    decision = matrix.evaluate(
        ["execution.run.write", "missing.permission"],
        ["execution.run.write", "unknown.permission"],
    )
    role_decision = matrix.evaluate_roles(
        ["execution.run.write", "execution.orchestration.manage"],
        ["cross-team-operator", "unknown-role"],
    )

    assert restored == contract
    assert audit.release_ready is True
    assert decision.allowed is False
    assert decision.granted_permissions == ["execution.run.write"]
    assert decision.missing_permissions == ["missing.permission"]
    assert role_decision.allowed is True
    assert role_decision.granted_permissions == ["execution.orchestration.manage", "execution.run.write"]
    assert role_decision.missing_permissions == []


def test_render_execution_contract_report_includes_role_matrix() -> None:
    contract = build_contract()

    report = render_execution_contract_report(contract, ExecutionContractLibrary().audit(contract))

    assert "- Roles: 4" in report
    assert "## Roles" in report
    assert "- eng-lead: personas=Eng Lead permissions=execution.run.write, execution.run.approve" in report
    assert "- Missing roles: none" in report
    assert "- Roles missing personas: none" in report
    assert "- Roles missing scope bindings: none" in report
    assert "- Roles missing escalation targets: none" in report


def test_operations_api_contract_draft_is_release_ready() -> None:
    contract = build_operations_api_contract()

    audit = ExecutionContractLibrary().audit(contract)
    report = render_execution_contract_report(contract, audit)

    assert contract.contract_id == "OPE-131"
    assert audit.release_ready is True
    assert len(contract.apis) == 12
    assert "GET /operations/dashboard" in report
    assert "GET /operations/runs/{run_id}" in report
    assert "GET /operations/queue/control-center" in report
    assert "GET /operations/risk/overview" in report
    assert "GET /operations/sla/overview" in report
    assert "GET /operations/regressions" in report
    assert "GET /operations/flows/{run_id}" in report
    assert "GET /operations/billing/entitlements" in report


def test_operations_api_contract_permissions_cover_read_and_action_paths() -> None:
    contract = build_operations_api_contract()
    matrix = ExecutionPermissionMatrix(contract.permissions)

    viewer = matrix.evaluate(
        ["operations.dashboard.read", "operations.queue.read", "operations.run.read"],
        ["operations.dashboard.read", "operations.queue.read", "operations.run.read"],
    )
    operator = matrix.evaluate(
        ["operations.queue.act", "operations.run.approve", "operations.billing.read"],
        ["operations.queue.act", "operations.billing.read"],
    )

    assert viewer.allowed is True
    assert operator.allowed is False
    assert operator.missing_permissions == ["operations.run.approve"]


def test_execution_contract_audit_requires_persona_scope_and_escalation_metadata() -> None:
    contract = build_contract()
    contract.roles[0] = ExecutionRole(
        name="eng-lead",
        personas=[],
        granted_permissions=["execution.run.write"],
        scope_bindings=[],
        escalation_target="",
    )

    audit = ExecutionContractLibrary().audit(contract)

    assert audit.roles_missing_personas == ["eng-lead"]
    assert audit.roles_missing_scope_bindings == ["eng-lead"]
    assert audit.roles_missing_escalation_targets == ["eng-lead"]
    assert audit.release_ready is False
