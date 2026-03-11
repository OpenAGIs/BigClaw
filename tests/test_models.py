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
