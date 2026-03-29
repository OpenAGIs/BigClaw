from bigclaw.issue_archive import (
    ArchivedIssue,
    IssuePriorityArchive,
    IssuePriorityArchiveAudit,
    IssuePriorityArchivist,
    render_issue_priority_archive_report,
)


def test_issue_priority_archive_round_trip_preserves_manifest_shape() -> None:
    archive = IssuePriorityArchive(
        issue_id="BIG-4307",
        title="Issue archive readiness",
        version="v2",
        findings=[
            ArchivedIssue(
                finding_id="finding-1",
                summary="Missing owner",
                category="ui",
                priority="P1",
                owner="design",
                surface="overview",
                impact="triage delay",
                evidence=["board-snapshot"],
            )
        ],
    )

    restored = IssuePriorityArchive.from_dict(archive.to_dict())

    assert restored == archive


def test_issue_priority_archivist_flags_gaps_and_normalizes_findings() -> None:
    archive = IssuePriorityArchive(
        issue_id="BIG-4307",
        title="Issue archive readiness",
        version="v2",
        findings=[
            ArchivedIssue(
                finding_id="finding-1",
                summary="Missing owner",
                category=" UI ",
                priority=" p0 ",
                owner="",
                status="open",
            ),
            ArchivedIssue(
                finding_id="finding-2",
                summary="Unexpected priority",
                category="ops",
                priority="P9",
                owner="ops",
                status="resolved",
            ),
        ],
    )

    audit = IssuePriorityArchivist().audit(archive)

    assert audit.ready is False
    assert audit.priority_counts == {"P0": 1, "P1": 0, "P2": 0}
    assert audit.category_counts == {"ia": 0, "metric": 0, "permission": 0, "ui": 1}
    assert audit.missing_owners == ["finding-1"]
    assert audit.invalid_priorities == ["finding-2"]
    assert audit.invalid_categories == ["finding-2"]
    assert audit.unresolved_p0_findings == ["finding-1"]


def test_issue_priority_archive_audit_round_trip_preserves_findings() -> None:
    audit = IssuePriorityArchiveAudit(
        ready=False,
        finding_count=2,
        priority_counts={"P0": 1, "P1": 0, "P2": 1},
        category_counts={"ui": 1, "ia": 1, "permission": 0, "metric": 0},
        missing_owners=["finding-1"],
        invalid_priorities=["finding-3"],
        invalid_categories=["finding-4"],
        unresolved_p0_findings=["finding-1"],
    )

    restored = IssuePriorityArchiveAudit.from_dict(audit.to_dict())

    assert restored == audit
    assert restored.summary == "HOLD: findings=2 missing_owners=1 invalid_priorities=1 invalid_categories=1 unresolved_p0=1"


def test_render_issue_priority_archive_report_summarizes_findings_and_audit() -> None:
    archive = IssuePriorityArchive(
        issue_id="BIG-4307",
        title="Issue archive readiness",
        version="v2",
        findings=[
            ArchivedIssue(
                finding_id="finding-1",
                summary="Missing owner",
                category="ui",
                priority="P1",
                owner="design",
                surface="overview",
                impact="triage delay",
                evidence=["board-snapshot"],
            )
        ],
    )

    audit = IssuePriorityArchivist().audit(archive)
    report = render_issue_priority_archive_report(archive, audit)

    assert "# Issue Priority Archive" in report
    assert "- Audit: READY: findings=1 missing_owners=0 invalid_priorities=0 invalid_categories=0 unresolved_p0=0" in report
    assert "- finding-1: Missing owner category=ui priority=P1 owner=design status=open" in report
    assert "surface=overview impact=triage delay evidence=board-snapshot" in report
    assert "- Missing owners: none" in report
