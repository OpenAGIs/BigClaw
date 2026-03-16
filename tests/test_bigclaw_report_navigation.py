from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
BIGCLAW_GO = REPO_ROOT / "bigclaw-go"


def test_review_navigation_reports_and_issue_ids_are_current() -> None:
    review = (BIGCLAW_GO / "docs/reports/review-readiness.md").read_text()
    closeout = (BIGCLAW_GO / "docs/reports/epic-closure-readiness-report.md").read_text()
    project_sync = (BIGCLAW_GO / "docs/reports/linear-project-sync-summary.md").read_text()

    for issue_id in ("OPE-254", "OPE-255", "OPE-256", "OPE-257"):
        assert issue_id in review
        assert issue_id in closeout
        assert issue_id in project_sync

    for report_path in (
        "docs/reports/review-readiness.md",
        "docs/reports/epic-closure-readiness-report.md",
        "docs/reports/benchmark-readiness-report.md",
        "docs/reports/long-duration-soak-report.md",
        "docs/reports/v2-phase1-operations-foundation-report.md",
        "docs/reports/scheduler-policy-report.md",
        "docs/reports/linear-project-sync-summary.md",
    ):
        assert report_path in review or report_path in closeout or report_path in project_sync
        assert (BIGCLAW_GO / report_path).exists(), report_path
