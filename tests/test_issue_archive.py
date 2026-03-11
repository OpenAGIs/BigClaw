from bigclaw.issue_archive import (
    ArchivedIssue,
    IssuePriorityArchive,
    IssuePriorityArchiveAudit,
    IssuePriorityArchivist,
    render_issue_priority_archive_report,
)


def test_issue_priority_archive_round_trip_preserves_manifest_shape() -> None:
    archive = IssuePriorityArchive(
        issue_id="OPE-137",
        title="BIG-4403 问题清单与优先级归档",
        version="v4.0-review-1",
        findings=[
            ArchivedIssue(
                finding_id="BIG-4403-1",
                summary="Global nav alert badge lacks escalation color contrast.",
                category="ui",
                priority="P1",
                owner="product-experience",
                surface="console-header",
                impact="Operators can miss pending incident alerts.",
                evidence=["wireframe-review", "contrast-audit"],
            )
        ],
    )

    restored = IssuePriorityArchive.from_dict(archive.to_dict())

    assert restored == archive


def test_issue_priority_archive_audit_flags_owner_priority_category_and_open_p0_gaps() -> None:
    archive = IssuePriorityArchive(
        issue_id="OPE-137",
        title="BIG-4403 问题清单与优先级归档",
        version="v4.0-review-1",
        findings=[
            ArchivedIssue(
                finding_id="BIG-4403-1",
                summary="Primary queue screen omits inline permission guidance.",
                category="permission",
                priority="P0",
                owner="",
                status="open",
            ),
            ArchivedIssue(
                finding_id="BIG-4403-2",
                summary="Information architecture duplicates run history entry points.",
                category="ia",
                priority="P1",
                owner="product-experience",
                status="open",
            ),
            ArchivedIssue(
                finding_id="BIG-4403-3",
                summary="Latency metric label mismatches backend contract.",
                category="metrics",
                priority="P3",
                owner="engineering-operations",
                status="resolved",
            ),
        ],
    )

    audit = IssuePriorityArchivist().audit(archive)

    assert audit.ready is False
    assert audit.finding_count == 3
    assert audit.priority_counts == {"P0": 1, "P1": 1, "P2": 0}
    assert audit.category_counts == {"ia": 1, "metric": 0, "permission": 1, "ui": 0}
    assert audit.missing_owners == ["BIG-4403-1"]
    assert audit.invalid_priorities == ["BIG-4403-3"]
    assert audit.invalid_categories == ["BIG-4403-3"]
    assert audit.unresolved_p0_findings == ["BIG-4403-1"]


def test_issue_priority_archive_audit_round_trip_and_ready_state() -> None:
    audit = IssuePriorityArchiveAudit(
        ready=True,
        finding_count=2,
        priority_counts={"P0": 0, "P1": 1, "P2": 1},
        category_counts={"ui": 1, "ia": 1, "permission": 0, "metric": 0},
    )

    restored = IssuePriorityArchiveAudit.from_dict(audit.to_dict())

    assert restored == audit
    assert restored.summary == "READY: findings=2 missing_owners=0 invalid_priorities=0 invalid_categories=0 unresolved_p0=0"


def test_render_issue_priority_archive_report_summarizes_findings_and_rollups() -> None:
    archive = IssuePriorityArchive(
        issue_id="OPE-137",
        title="BIG-4403 问题清单与优先级归档",
        version="v4.0-review-1",
        findings=[
            ArchivedIssue(
                finding_id="BIG-4403-1",
                summary="Queue dashboard card spacing breaks scannability under dense data.",
                category="ui",
                priority="P1",
                owner="product-experience",
                surface="operations-dashboard",
                impact="Dense incident queues become harder to triage.",
                status="resolved",
                evidence=["wireframe-pack"],
            ),
            ArchivedIssue(
                finding_id="BIG-4403-2",
                summary="Run detail permission matrix is not exposed in empty states.",
                category="permission",
                priority="P2",
                owner="security-platform",
                surface="run-detail",
                impact="Reviewers cannot confirm role gating behavior.",
                status="resolved",
                evidence=["review-notes", "permission-checklist"],
            ),
        ],
    )

    audit = IssuePriorityArchivist().audit(archive)
    report = render_issue_priority_archive_report(archive, audit)

    assert "# Issue Priority Archive" in report
    assert "- Audit: READY: findings=2 missing_owners=0 invalid_priorities=0 invalid_categories=0 unresolved_p0=0" in report
    assert "- Priority Counts: P0=0 P1=1 P2=1" in report
    assert "- Category Counts: ui=1 ia=0 permission=1 metric=0" in report
    assert (
        "- BIG-4403-1: Queue dashboard card spacing breaks scannability under dense data. "
        "category=ui priority=P1 owner=product-experience status=resolved"
    ) in report
    assert "- Unresolved P0 findings: none" in report
