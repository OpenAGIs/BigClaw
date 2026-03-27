from pathlib import Path


def test_harness_migration_plan_captures_required_sections() -> None:
    plan = Path("bigclaw-go/docs/reports/test-harness-migration-plan.md").read_text()

    required_phrases = [
        "three Go-native harness lanes",
        "Current Harness Split",
        "First-Batch Implementation and Retrofit List",
        "Validation and Regression Surface",
        "Branch and PR Recommendation",
        "Branch naming: `BIG-GO-903-test-harness-migration`",
        "python3 -m pytest tests/test_harness_migration_plan.py",
        "cd bigclaw-go && go test ./internal/regression",
        "go test ./internal/queue ./internal/scheduler ./internal/worker",
        "scripts/e2e/run_all.sh",
        "scripts/migration/live_shadow_scorecard.py",
    ]

    for phrase in required_phrases:
        assert phrase in plan


def test_harness_migration_plan_is_linked_from_migration_entrypoints() -> None:
    linked_docs = [
        Path("bigclaw-go/docs/migration.md"),
        Path("bigclaw-go/docs/reports/migration-readiness-report.md"),
        Path("bigclaw-go/docs/reports/migration-plan-review-notes.md"),
        Path("bigclaw-go/docs/reports/review-readiness.md"),
        Path("bigclaw-go/docs/reports/issue-coverage.md"),
        Path("bigclaw-go/docs/reports/parallel-validation-matrix.md"),
        Path("bigclaw-go/docs/e2e-validation.md"),
        Path("docs/openclaw-parallel-gap-analysis.md"),
    ]

    for path in linked_docs:
        assert "test-harness-migration-plan.md" in path.read_text(), path
