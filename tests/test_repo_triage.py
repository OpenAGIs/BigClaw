from bigclaw.repo_triage import (
    LineageEvidence,
    approval_evidence_packet,
    recommend_triage_action,
)


def test_lineage_aware_recommendations():
    rec = recommend_triage_action(
        status="needs-approval",
        evidence=LineageEvidence(candidate_commit="abc", accepted_ancestor="0001", similar_failure_count=0),
    )
    assert rec.action == "approve"

    rec = recommend_triage_action(
        status="failed",
        evidence=LineageEvidence(candidate_commit="abc", similar_failure_count=3),
    )
    assert rec.action == "replay"


def test_approval_evidence_packet_includes_candidate_and_accepted_hash():
    packet = approval_evidence_packet(
        run_id="run-170",
        links=[
            {"role": "candidate", "commit_hash": "abc111"},
            {"role": "accepted", "commit_hash": "def222"},
        ],
        lineage_summary="candidate descends from accepted baseline",
    )

    assert packet["accepted_commit_hash"] == "def222"
    assert packet["candidate_commit_hash"] == "abc111"
    assert "accepted baseline" in packet["lineage_summary"]
