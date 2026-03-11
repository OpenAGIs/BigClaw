from bigclaw.planning import (
    CandidateBacklog,
    CandidateEntry,
    CandidatePlanner,
    EvidenceLink,
    EntryGate,
    EntryGateDecision,
    ResourceIsolationDecision,
    ResourceIsolationPolicy,
    render_candidate_backlog_report,
)
from bigclaw.governance import ScopeFreezeAudit


def test_candidate_backlog_round_trip_preserves_manifest_shape() -> None:
    backlog = CandidateBacklog(
        epic_id="BIG-EPIC-20",
        title="v4.0 v3候选与进入条件",
        version="v4.0-v3",
        candidates=[
            CandidateEntry(
                candidate_id="candidate-release-control",
                title="Release control center",
                theme="console-governance",
                priority="P0",
                owner="platform-ui",
                outcome="Unify console release gates and promotion evidence.",
                validation_command="python3 -m pytest tests/test_design_system.py -q",
                capabilities=["release-gate", "reporting"],
                evidence=["acceptance-suite", "validation-report"],
                evidence_links=[
                    EvidenceLink(
                        label="ui-acceptance",
                        target="tests/test_design_system.py",
                        capability="release-gate",
                        note="role-permission and audit readiness coverage",
                    )
                ],
                classification="release-core",
                resource_lanes=["v3-console-release"],
            )
        ],
    )

    restored = CandidateBacklog.from_dict(backlog.to_dict())

    assert restored == backlog


def test_candidate_backlog_ranks_ready_items_ahead_of_blocked_work() -> None:
    backlog = CandidateBacklog(
        epic_id="BIG-EPIC-20",
        title="v4.0 v3候选与进入条件",
        version="v4.0-v3",
        candidates=[
            CandidateEntry(
                candidate_id="candidate-risky",
                title="Risky migration",
                theme="runtime",
                priority="P0",
                owner="runtime",
                outcome="Move execution runtime to the next rollout ring.",
                validation_command="python3 -m pytest tests/test_runtime.py -q",
                capabilities=["runtime-hardening"],
                evidence=["benchmark"],
                blockers=["missing rollback plan"],
                classification="platform-risk",
                resource_lanes=["v2-runtime-core"],
            ),
            CandidateEntry(
                candidate_id="candidate-ready",
                title="Release control center",
                theme="console-governance",
                priority="P1",
                owner="platform-ui",
                outcome="Unify console release gates and promotion evidence.",
                validation_command="python3 -m pytest tests/test_design_system.py -q",
                capabilities=["release-gate", "reporting"],
                evidence=["acceptance-suite", "validation-report"],
                evidence_links=[
                    EvidenceLink(
                        label="ui-acceptance",
                        target="tests/test_design_system.py",
                        capability="release-gate",
                    )
                ],
                classification="release-core",
                resource_lanes=["v3-console-release"],
            ),
        ],
    )

    ranked_ids = [candidate.candidate_id for candidate in backlog.ranked_candidates]

    assert ranked_ids == ["candidate-ready", "candidate-risky"]


def test_entry_gate_evaluation_requires_ready_candidates_capabilities_and_evidence() -> None:
    backlog = CandidateBacklog(
        epic_id="BIG-EPIC-20",
        title="v4.0 v3候选与进入条件",
        version="v4.0-v3",
        candidates=[
            CandidateEntry(
                candidate_id="candidate-release-control",
                title="Release control center",
                theme="console-governance",
                priority="P0",
                owner="platform-ui",
                outcome="Unify console release gates and promotion evidence.",
                validation_command="python3 -m pytest tests/test_design_system.py -q",
                capabilities=["release-gate", "reporting"],
                evidence=["acceptance-suite", "validation-report"],
                classification="release-core",
                resource_lanes=["v3-console-release"],
            ),
            CandidateEntry(
                candidate_id="candidate-ops-hardening",
                title="Ops hardening",
                theme="ops-command-center",
                priority="P0",
                owner="ops-platform",
                outcome="Package the command-center rollout with weekly review evidence.",
                validation_command="python3 -m pytest tests/test_operations.py -q",
                capabilities=["ops-control"],
                evidence=["weekly-review"],
                classification="operations-core",
                resource_lanes=["v3-ops-control"],
            ),
            CandidateEntry(
                candidate_id="candidate-orchestration",
                title="Orchestration rollout",
                theme="agent-orchestration",
                priority="P1",
                owner="orchestration",
                outcome="Promote cross-team orchestration with commercialization visibility.",
                validation_command="python3 -m pytest tests/test_orchestration.py -q",
                capabilities=["commercialization", "handoff"],
                evidence=["pilot-evidence"],
                classification="commercialization-core",
                resource_lanes=["v3-orchestration-rollout"],
            ),
        ],
    )
    gate = EntryGate(
        gate_id="gate-v3-entry",
        name="V3 Entry Gate",
        min_ready_candidates=3,
        required_capabilities=["release-gate", "ops-control", "commercialization"],
        required_evidence=["acceptance-suite", "pilot-evidence", "validation-report"],
        required_baseline_version="v2.0",
    )
    baseline_audit = ScopeFreezeAudit(
        board_name="BigClaw v2.0 Freeze",
        version="v2.0",
        total_items=5,
    )

    decision = CandidatePlanner().evaluate_gate(backlog, gate, baseline_audit=baseline_audit)

    assert decision.passed is True
    assert set(decision.ready_candidate_ids) == {
        "candidate-release-control",
        "candidate-ops-hardening",
        "candidate-orchestration",
    }
    assert decision.missing_capabilities == []
    assert decision.missing_evidence == []
    assert decision.baseline_ready is True
    assert decision.baseline_findings == []


def test_entry_gate_holds_when_v2_baseline_is_missing_or_not_ready() -> None:
    backlog = CandidateBacklog(
        epic_id="BIG-EPIC-20",
        title="v4.0 v3候选与进入条件",
        version="v4.0-v3",
        candidates=[
            CandidateEntry(
                candidate_id="candidate-release-control",
                title="Release control center",
                theme="console-governance",
                priority="P0",
                owner="platform-ui",
                outcome="Unify console release gates and promotion evidence.",
                validation_command="python3 -m pytest tests/test_design_system.py -q",
                capabilities=["release-gate"],
                evidence=["acceptance-suite", "validation-report"],
            ),
            CandidateEntry(
                candidate_id="candidate-ops-hardening",
                title="Ops hardening",
                theme="ops-command-center",
                priority="P0",
                owner="ops-platform",
                outcome="Package the command-center rollout with weekly review evidence.",
                validation_command="python3 -m pytest tests/test_operations.py -q",
                capabilities=["ops-control"],
                evidence=["weekly-review"],
            ),
            CandidateEntry(
                candidate_id="candidate-orchestration",
                title="Orchestration rollout",
                theme="agent-orchestration",
                priority="P1",
                owner="orchestration",
                outcome="Promote cross-team orchestration with commercialization visibility.",
                validation_command="python3 -m pytest tests/test_orchestration.py -q",
                capabilities=["commercialization"],
                evidence=["pilot-evidence"],
            ),
        ],
    )
    gate = EntryGate(
        gate_id="gate-v3-entry",
        name="V3 Entry Gate",
        min_ready_candidates=3,
        required_capabilities=["release-gate", "ops-control", "commercialization"],
        required_evidence=["acceptance-suite", "pilot-evidence", "validation-report"],
        required_baseline_version="v2.0",
    )

    missing_baseline = CandidatePlanner().evaluate_gate(backlog, gate)
    failed_baseline = CandidatePlanner().evaluate_gate(
        backlog,
        gate,
        baseline_audit=ScopeFreezeAudit(
            board_name="BigClaw v2.0 Freeze",
            version="v2.0",
            total_items=5,
            missing_validation=["OPE-116"],
        ),
    )

    assert missing_baseline.passed is False
    assert missing_baseline.baseline_ready is False
    assert missing_baseline.baseline_findings == ["missing baseline audit for v2.0"]
    assert failed_baseline.passed is False
    assert failed_baseline.baseline_ready is False
    assert failed_baseline.baseline_findings == ["baseline v2.0 is not release ready (87.5)"]


def test_entry_gate_decision_round_trip_preserves_findings() -> None:
    decision = EntryGateDecision(
        gate_id="gate-v3-entry",
        passed=False,
        ready_candidate_ids=["candidate-release-control"],
        blocked_candidate_ids=["candidate-runtime"],
        missing_capabilities=["commercialization"],
        missing_evidence=["pilot-evidence"],
        baseline_ready=False,
        baseline_findings=["baseline v2.0 is not release ready (87.5)"],
        blocker_count=1,
    )

    restored = EntryGateDecision.from_dict(decision.to_dict())

    assert restored == decision


def test_resource_isolation_policy_and_decision_round_trip_preserve_findings() -> None:
    policy = ResourceIsolationPolicy(
        policy_id="policy-v3-isolation",
        name="V3 Backlog Isolation",
        protected_v2_lanes=["v2-runtime-core", "v2-console-ops"],
        allowed_classifications=["release-core", "operations-core", "commercialization-core"],
        exception_candidate_ids=["candidate-exception"],
    )
    decision = ResourceIsolationDecision(
        policy_id="policy-v3-isolation",
        passed=False,
        classified_candidate_ids=["candidate-release-control"],
        unclassified_candidate_ids=["candidate-unassigned"],
        disallowed_classifications={"candidate-risk": "legacy"},
        conflicting_lanes={"candidate-runtime": ["v2-runtime-core"]},
    )

    assert ResourceIsolationPolicy.from_dict(policy.to_dict()) == policy
    assert ResourceIsolationDecision.from_dict(decision.to_dict()) == decision


def test_resource_isolation_audit_flags_unclassified_and_v2_lane_conflicts() -> None:
    backlog = CandidateBacklog(
        epic_id="BIG-EPIC-20",
        title="v4.0 v3候选与进入条件",
        version="v4.0-v3",
        candidates=[
            CandidateEntry(
                candidate_id="candidate-release-control",
                title="Release control center",
                theme="console-governance",
                priority="P0",
                owner="platform-ui",
                outcome="Unify console release gates and promotion evidence.",
                validation_command="python3 -m pytest tests/test_design_system.py -q",
                capabilities=["release-gate"],
                evidence=["validation-report"],
                classification="release-core",
                resource_lanes=["v3-console-release"],
            ),
            CandidateEntry(
                candidate_id="candidate-runtime-borrow",
                title="Borrow v2 runtime lane",
                theme="runtime",
                priority="P1",
                owner="runtime",
                outcome="Attempt shared rollout using protected runtime staff.",
                validation_command="python3 -m pytest tests/test_runtime.py -q",
                capabilities=["runtime-hardening"],
                evidence=["benchmark"],
                classification="operations-core",
                resource_lanes=["v2-runtime-core"],
            ),
            CandidateEntry(
                candidate_id="candidate-unassigned",
                title="Unassigned backlog draft",
                theme="planning",
                priority="P2",
                owner="pm",
                outcome="Draft backlog slice without a final class.",
                validation_command="python3 -m pytest tests/test_planning.py -q",
                capabilities=["planning"],
                evidence=["validation-report"],
                resource_lanes=["v3-planning"],
            ),
        ],
    )
    policy = ResourceIsolationPolicy(
        policy_id="policy-v3-isolation",
        name="V3 Backlog Isolation",
        protected_v2_lanes=["v2-runtime-core", "v2-console-ops"],
        allowed_classifications=["release-core", "operations-core", "commercialization-core"],
    )

    decision = CandidatePlanner().audit_resource_isolation(backlog, policy)

    assert decision.passed is False
    assert decision.classified_candidate_ids == ["candidate-release-control", "candidate-runtime-borrow"]
    assert decision.unclassified_candidate_ids == ["candidate-unassigned"]
    assert decision.conflicting_candidate_ids == ["candidate-runtime-borrow"]
    assert decision.conflicting_lanes == {"candidate-runtime-borrow": ["v2-runtime-core"]}


def test_render_candidate_backlog_report_summarizes_backlog_and_gate_findings() -> None:
    backlog = CandidateBacklog(
        epic_id="BIG-EPIC-20",
        title="v4.0 v3候选与进入条件",
        version="v4.0-v3",
        candidates=[
            CandidateEntry(
                candidate_id="candidate-release-control",
                title="Release control center",
                theme="console-governance",
                priority="P0",
                owner="platform-ui",
                outcome="Unify console release gates and promotion evidence.",
                validation_command="python3 -m pytest tests/test_design_system.py -q",
                capabilities=["release-gate", "reporting"],
                evidence=["acceptance-suite", "validation-report"],
                evidence_links=[
                    EvidenceLink(
                        label="ui-acceptance",
                        target="tests/test_design_system.py",
                        capability="release-gate",
                    )
                ],
                classification="release-core",
                resource_lanes=["v3-console-release"],
            )
        ],
    )
    gate = EntryGate(
        gate_id="gate-v3-entry",
        name="V3 Entry Gate",
        min_ready_candidates=1,
        required_capabilities=["release-gate"],
        required_evidence=["validation-report"],
        required_baseline_version="v2.0",
    )
    decision = CandidatePlanner().evaluate_gate(
        backlog,
        gate,
        baseline_audit=ScopeFreezeAudit(
            board_name="BigClaw v2.0 Freeze",
            version="v2.0",
            total_items=5,
        ),
    )
    policy = ResourceIsolationPolicy(
        policy_id="policy-v3-isolation",
        name="V3 Backlog Isolation",
        protected_v2_lanes=["v2-runtime-core"],
        allowed_classifications=["release-core"],
    )
    isolation_decision = CandidatePlanner().audit_resource_isolation(backlog, policy)

    report = render_candidate_backlog_report(backlog, gate, decision, policy, isolation_decision)

    assert "# V3 Candidate Backlog Report" in report
    assert "- Epic: BIG-EPIC-20 v4.0 v3候选与进入条件" in report
    assert "- Decision: PASS: ready=1 blocked=0 missing_capabilities=0 missing_evidence=0 baseline_findings=0" in report
    assert (
        "- candidate-release-control: Release control center "
        "priority=P0 owner=platform-ui score=100 ready=True"
    ) in report
    assert "validation=python3 -m pytest tests/test_design_system.py -q" in report
    assert "- ui-acceptance -> tests/test_design_system.py capability=release-gate" in report
    assert "- Missing evidence: none" in report
    assert "- Baseline ready: True" in report
    assert "- Baseline findings: none" in report
    assert "classification=release-core resource_lanes=v3-console-release" in report
    assert "## Resource Isolation" in report
    assert "- Protected lane conflicts: none" in report


def test_candidate_entry_round_trip_preserves_evidence_links() -> None:
    candidate = CandidateEntry(
        candidate_id="candidate-ops-hardening",
        title="Ops hardening",
        theme="ops-command-center",
        priority="P0",
        owner="ops-platform",
        outcome="Package command-center and approval surfaces with linked evidence.",
        validation_command="python3 -m pytest tests/test_operations.py tests/test_saved_views.py -q",
        capabilities=["ops-control", "saved-views"],
        evidence=["weekly-review", "validation-report"],
        evidence_links=[
            EvidenceLink(
                label="queue-control-center",
                target="src/bigclaw/operations.py",
                capability="ops-control",
                note="queue and approval command center",
            ),
            EvidenceLink(
                label="saved-view-report",
                target="src/bigclaw/saved_views.py",
                capability="saved-views",
                note="team saved views and digest evidence",
            ),
        ],
        classification="operations-core",
        resource_lanes=["v3-ops-control"],
    )

    restored = CandidateEntry.from_dict(candidate.to_dict())

    assert restored == candidate
