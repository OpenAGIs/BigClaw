from pathlib import Path


DIGESTS = {
    "OPE-264": {
        "path": Path("bigclaw-go/docs/reports/tracing-backend-follow-up-digest.md"),
        "title": "BIG-PAR-075",
        "links": [
            "docs/reports/go-control-plane-observability-report.md",
            "docs/reports/review-readiness.md",
            "docs/reports/issue-coverage.md",
            "internal/observability/recorder.go",
            "internal/api/server.go",
        ],
        "phrases": [
            "no external tracing backend",
            "no cross-process span propagation beyond in-memory trace grouping",
        ],
        "indexes": [
            Path("bigclaw-go/docs/reports/go-control-plane-observability-report.md"),
            Path("bigclaw-go/docs/reports/review-readiness.md"),
            Path("bigclaw-go/docs/reports/issue-coverage.md"),
            Path("docs/openclaw-parallel-gap-analysis.md"),
        ],
    },
    "OPE-265": {
        "path": Path("bigclaw-go/docs/reports/telemetry-pipeline-controls-follow-up-digest.md"),
        "title": "BIG-PAR-076",
        "links": [
            "docs/reports/go-control-plane-observability-report.md",
            "docs/reports/review-readiness.md",
            "internal/api/server.go",
            "internal/observability/recorder.go",
            "internal/worker/runtime.go",
        ],
        "phrases": [
            "no full OpenTelemetry-native metrics / tracing pipeline",
            "no configurable sampling or high-cardinality controls",
        ],
        "indexes": [
            Path("bigclaw-go/docs/reports/go-control-plane-observability-report.md"),
            Path("bigclaw-go/docs/reports/review-readiness.md"),
            Path("bigclaw-go/docs/reports/issue-coverage.md"),
            Path("docs/openclaw-parallel-gap-analysis.md"),
        ],
    },
    "OPE-266": {
        "path": Path("bigclaw-go/docs/reports/live-shadow-comparison-follow-up-digest.md"),
        "title": "BIG-PAR-077",
        "links": [
            "docs/reports/migration-readiness-report.md",
            "docs/migration-shadow.md",
            "docs/reports/shadow-compare-report.json",
            "docs/reports/shadow-matrix-report.json",
            "docs/reports/migration-plan-review-notes.md",
        ],
        "phrases": [
            "no live legacy-vs-Go production traffic comparison",
            "fixture-backed",
        ],
        "indexes": [
            Path("bigclaw-go/docs/reports/migration-readiness-report.md"),
            Path("bigclaw-go/docs/migration-shadow.md"),
            Path("bigclaw-go/docs/reports/migration-plan-review-notes.md"),
            Path("bigclaw-go/docs/reports/review-readiness.md"),
            Path("bigclaw-go/docs/reports/issue-coverage.md"),
            Path("docs/openclaw-parallel-gap-analysis.md"),
        ],
    },
    "OPE-267": {
        "path": Path("bigclaw-go/docs/reports/rollback-safeguard-follow-up-digest.md"),
        "title": "BIG-PAR-078",
        "links": [
            "docs/reports/migration-readiness-report.md",
            "docs/migration.md",
            "docs/reports/migration-plan-review-notes.md",
            "docs/reports/review-readiness.md",
            "docs/reports/issue-coverage.md",
        ],
        "phrases": [
            "rollback remains operator-driven",
            "no tenant-scoped automated rollback trigger",
        ],
        "indexes": [
            Path("bigclaw-go/docs/reports/migration-readiness-report.md"),
            Path("bigclaw-go/docs/migration.md"),
            Path("bigclaw-go/docs/reports/migration-plan-review-notes.md"),
            Path("bigclaw-go/docs/reports/review-readiness.md"),
            Path("bigclaw-go/docs/reports/issue-coverage.md"),
            Path("docs/openclaw-parallel-gap-analysis.md"),
        ],
    },
    "OPE-268": {
        "path": Path("bigclaw-go/docs/reports/production-corpus-migration-coverage-digest.md"),
        "title": "BIG-PAR-079",
        "links": [
            "docs/reports/migration-readiness-report.md",
            "docs/reports/shadow-matrix-report.json",
            "docs/reports/shadow-compare-report.json",
            "docs/migration-shadow.md",
            "docs/reports/issue-coverage.md",
            "examples/shadow-corpus-manifest.json",
        ],
        "phrases": [
            "fixture-backed evidence only",
            "no real production issue/task corpus coverage",
        ],
        "indexes": [
            Path("bigclaw-go/docs/reports/migration-readiness-report.md"),
            Path("bigclaw-go/docs/migration-shadow.md"),
            Path("bigclaw-go/docs/reports/review-readiness.md"),
            Path("bigclaw-go/docs/reports/issue-coverage.md"),
            Path("docs/openclaw-parallel-gap-analysis.md"),
        ],
    },
    "OPE-269": {
        "path": Path("bigclaw-go/docs/reports/subscriber-takeover-executability-follow-up-digest.md"),
        "title": "BIG-PAR-080",
        "links": [
            "docs/reports/multi-subscriber-takeover-validation-report.md",
            "docs/reports/multi-subscriber-takeover-validation-report.json",
            "scripts/e2e/subscriber_takeover_fault_matrix.py",
            "docs/reports/event-bus-reliability-report.md",
            "docs/reports/issue-coverage.md",
            "docs/reports/review-readiness.md",
            "docs/openclaw-parallel-gap-analysis.md",
        ],
        "phrases": [
            "deterministic local harness only",
            "not yet a live multi-node subscriber takeover proof",
        ],
        "indexes": [
            Path("bigclaw-go/docs/reports/multi-subscriber-takeover-validation-report.md"),
            Path("bigclaw-go/docs/reports/event-bus-reliability-report.md"),
            Path("bigclaw-go/docs/reports/review-readiness.md"),
            Path("bigclaw-go/docs/reports/issue-coverage.md"),
            Path("docs/openclaw-parallel-gap-analysis.md"),
            Path("bigclaw-go/docs/e2e-validation.md"),
        ],
    },
    "OPE-270": {
        "path": Path("bigclaw-go/docs/reports/cross-process-coordination-boundary-digest.md"),
        "title": "BIG-PAR-081",
        "links": [
            "docs/reports/event-bus-reliability-report.md",
            "docs/reports/multi-node-coordination-report.md",
            "docs/reports/review-readiness.md",
            "docs/reports/issue-coverage.md",
            "docs/openclaw-parallel-gap-analysis.md",
        ],
        "phrases": [
            "no partitioned topic model",
            "no broker-backed cross-process subscriber coordination",
        ],
        "indexes": [
            Path("bigclaw-go/docs/reports/event-bus-reliability-report.md"),
            Path("bigclaw-go/docs/reports/multi-node-coordination-report.md"),
            Path("bigclaw-go/docs/reports/review-readiness.md"),
            Path("bigclaw-go/docs/reports/issue-coverage.md"),
            Path("docs/openclaw-parallel-gap-analysis.md"),
        ],
    },
    "OPE-271": {
        "path": Path("bigclaw-go/docs/reports/validation-bundle-continuation-digest.md"),
        "title": "BIG-PAR-082",
        "links": [
            "docs/reports/live-validation-index.md",
            "docs/reports/live-validation-summary.json",
            "docs/reports/multi-node-coordination-report.md",
            "docs/reports/review-readiness.md",
            "docs/openclaw-parallel-gap-analysis.md",
        ],
        "phrases": [
            "point-in-time validation bundle only",
            "continuation across future validation bundles remains manual",
        ],
        "indexes": [
            Path("bigclaw-go/docs/reports/live-validation-index.md"),
            Path("bigclaw-go/docs/reports/multi-node-coordination-report.md"),
            Path("bigclaw-go/docs/reports/review-readiness.md"),
            Path("docs/openclaw-parallel-gap-analysis.md"),
        ],
    },
}


def test_followup_digests_capture_links_and_constraints() -> None:
    for issue_id, expectation in DIGESTS.items():
        text = expectation["path"].read_text()

        assert issue_id in text
        assert expectation["title"] in text
        for link in expectation["links"]:
            assert f"`{link}`" in text
        for phrase in expectation["phrases"]:
            assert phrase in text


def test_followup_indexes_reference_new_digests() -> None:
    for expectation in DIGESTS.values():
        digest = expectation["path"].as_posix().replace("bigclaw-go/", "")
        full_digest = expectation["path"].as_posix()
        for index in expectation["indexes"]:
            text = index.read_text()
            assert digest in text or full_digest in text
