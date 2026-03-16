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
    review = Path("bigclaw-go/docs/reports/review-readiness.md").read_text()
    coverage = Path("bigclaw-go/docs/reports/issue-coverage.md").read_text()
    observability = Path("bigclaw-go/docs/reports/go-control-plane-observability-report.md").read_text()
    gap_analysis = Path("docs/openclaw-parallel-gap-analysis.md").read_text()

    for digest in [
        "docs/reports/tracing-backend-follow-up-digest.md",
        "docs/reports/telemetry-pipeline-controls-follow-up-digest.md",
    ]:
        assert f"`{digest}`" in review or digest in review
        assert f"`{digest}`" in coverage or digest in coverage
        assert f"`{digest}`" in observability or digest in observability

    assert "OPE-264" in gap_analysis
    assert "OPE-265" in gap_analysis
