from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[1]
DIGEST = REPO_ROOT / "bigclaw-go/docs/reports/event-delivery-semantics-follow-up-digest.md"
LINKED_DOCS = [
    REPO_ROOT / "bigclaw-go/docs/reports/event-bus-reliability-report.md",
    REPO_ROOT / "bigclaw-go/docs/reports/issue-coverage.md",
    REPO_ROOT / "bigclaw-go/docs/reports/review-readiness.md",
    REPO_ROOT / "bigclaw-go/docs/reports/multi-subscriber-takeover-validation-report.md",
    REPO_ROOT / "docs/openclaw-parallel-gap-analysis.md",
]
CANONICAL_SNIPPETS = [
    "durable dedupe store keyed by `delivery.idempotency_key`",
    "No end-to-end delivery acknowledgement protocol exists beyond sink-level best-effort delivery.",
    "BigClaw remains replay-safe, not globally exactly-once.",
]


def test_event_delivery_digest_exists_and_links_point_to_it() -> None:
    assert DIGEST.exists()
    digest_rel = "docs/reports/event-delivery-semantics-follow-up-digest.md"
    root_rel = "bigclaw-go/docs/reports/event-delivery-semantics-follow-up-digest.md"

    for path in LINKED_DOCS:
        content = path.read_text()
        assert digest_rel in content or root_rel in content


def test_event_delivery_digest_captures_canonical_semantics() -> None:
    content = DIGEST.read_text()

    for snippet in CANONICAL_SNIPPETS:
        assert snippet in content


def test_event_bus_and_gap_analysis_use_canonical_exactly_once_wording() -> None:
    event_bus = (REPO_ROOT / "bigclaw-go/docs/reports/event-bus-reliability-report.md").read_text()
    gap_analysis = (REPO_ROOT / "docs/openclaw-parallel-gap-analysis.md").read_text()

    assert "BigClaw remains replay-safe, not globally exactly-once." in event_bus
    assert "the system remains replay-safe, not globally exactly-once." in gap_analysis
