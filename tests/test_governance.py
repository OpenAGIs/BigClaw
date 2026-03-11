from bigclaw.governance import (
    FreezeException,
    GovernanceBacklogItem,
    ScopeFreezeAudit,
    ScopeFreezeBoard,
    ScopeFreezeGovernance,
    render_scope_freeze_report,
)


def test_scope_freeze_board_round_trip_preserves_manifest_shape():
    board = ScopeFreezeBoard(
        name="BigClaw v4.0 Freeze",
        version="v4.0",
        freeze_date="2026-03-11",
        freeze_owner="pm-director",
        backlog_items=[
            GovernanceBacklogItem(
                issue_id="OPE-116",
                title="Scope freeze and task governance",
                phase="step-1",
                owner="pm-director",
                status="ready",
                scope_status="frozen",
                acceptance_criteria=["Epic scope frozen", "Backlog sequencing documented"],
                validation_plan=["governance-audit", "report-shared"],
            )
        ],
        exceptions=[
            FreezeException(
                issue_id="OPE-121",
                reason="Compliance requirement discovered after freeze",
                approved_by="cto",
                decision_note="Allow as approved exception.",
            )
        ],
    )

    restored = ScopeFreezeBoard.from_dict(board.to_dict())

    assert restored == board


def test_scope_freeze_audit_flags_backlog_governance_and_closeout_gaps():
    board = ScopeFreezeBoard(
        name="BigClaw v4.0 Freeze",
        version="v4.0",
        freeze_date="2026-03-11",
        freeze_owner="pm-director",
        backlog_items=[
            GovernanceBacklogItem(
                issue_id="OPE-116",
                title="Scope freeze and task governance",
                phase="step-1",
                owner="",
                status="planned",
                scope_status="proposed",
                acceptance_criteria=[],
                validation_plan=[],
                required_closeout=["validation-evidence"],
            ),
            GovernanceBacklogItem(
                issue_id="OPE-118",
                title="Duplicate issue record",
                phase="step-1",
                owner="pm-director",
                status="planned",
                scope_status="unknown",
                acceptance_criteria=["Keep backlog aligned"],
                validation_plan=["audit"],
                required_closeout=["validation-evidence", "git-push"],
            ),
            GovernanceBacklogItem(
                issue_id="OPE-116",
                title="Mirrored tracking entry",
                phase="step-1",
                owner="pm-director",
                status="planned",
                scope_status="frozen",
                acceptance_criteria=["Track freeze decisions"],
                validation_plan=["audit"],
            ),
        ],
        exceptions=[
            FreezeException(
                issue_id="OPE-130",
                reason="Pending steering review",
            )
        ],
    )

    audit = ScopeFreezeGovernance().audit(board)

    assert audit.duplicate_issue_ids == ["OPE-116"]
    assert audit.missing_owners == ["OPE-116"]
    assert audit.missing_acceptance == ["OPE-116"]
    assert audit.missing_validation == ["OPE-116"]
    assert audit.missing_closeout_requirements == {
        "OPE-116": ["git-push", "git-log-stat"],
        "OPE-118": ["git-log-stat"],
    }
    assert audit.unauthorized_scope_changes == ["OPE-116"]
    assert audit.invalid_scope_statuses == ["OPE-118"]
    assert audit.unapproved_exceptions == ["OPE-130"]
    assert audit.release_ready is False
    assert audit.readiness_score == 0.0


def test_scope_freeze_audit_round_trip_and_ready_state():
    audit = ScopeFreezeAudit(
        board_name="BigClaw v4.0 Freeze",
        version="v4.0",
        total_items=1,
    )

    restored = ScopeFreezeAudit.from_dict(audit.to_dict())

    assert restored == audit
    assert restored.release_ready is True
    assert restored.readiness_score == 100.0


def test_render_scope_freeze_report_summarizes_board_and_run_closeout_requirements():
    board = ScopeFreezeBoard(
        name="BigClaw v4.0 Freeze",
        version="v4.0",
        freeze_date="2026-03-11",
        freeze_owner="pm-director",
        backlog_items=[
            GovernanceBacklogItem(
                issue_id="OPE-116",
                title="Scope freeze and task governance",
                phase="step-1",
                owner="pm-director",
                status="ready",
                scope_status="frozen",
                acceptance_criteria=["Epic scope frozen"],
                validation_plan=["governance-audit"],
                required_closeout=["validation-evidence", "git-push", "git-log-stat"],
            )
        ],
        exceptions=[
            FreezeException(
                issue_id="OPE-121",
                reason="Approved scope exception",
                approved_by="cto",
            )
        ],
    )

    audit = ScopeFreezeGovernance().audit(board)
    report = render_scope_freeze_report(board, audit)

    assert "# Scope Freeze Governance Report" in report
    assert "- Readiness Score: 100.0" in report
    assert "- Release Ready: True" in report
    assert "- OPE-116: phase=step-1 owner=pm-director status=ready scope=frozen closeout=validation-evidence, git-push, git-log-stat" in report
    assert "- OPE-121: approved_by=cto reason=Approved scope exception" in report
    assert "- Missing closeout requirements: none" in report
