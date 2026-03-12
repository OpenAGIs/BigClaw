from bigclaw.planning import (
    EntryGateDecision,
    build_pilot_rollout_scorecard,
    evaluate_candidate_gate,
    render_pilot_rollout_gate_report,
)
from bigclaw.reports import render_repo_narrative_exports, render_weekly_repo_evidence_section


def test_pilot_rollout_scorecard_and_candidate_gate():
    scorecard = build_pilot_rollout_scorecard(
        adoption=84,
        convergence_improvement=78,
        review_efficiency=82,
        governance_incidents=1,
        evidence_completeness=88,
    )
    assert scorecard["recommendation"] == "go"

    gate_decision = EntryGateDecision(gate_id="gate-v3", passed=True)
    result = evaluate_candidate_gate(gate_decision=gate_decision, rollout_scorecard=scorecard)

    assert result["candidate_gate"] == "enable-by-default"
    report = render_pilot_rollout_gate_report(result)
    assert "Candidate gate" in report


def test_repo_weekly_narrative_exports_remain_consistent():
    section = render_weekly_repo_evidence_section(
        experiment_volume=14,
        converged_tasks=9,
        accepted_commits=7,
        hottest_threads=["repo/ope-168", "repo/ope-170"],
    )
    exports = render_repo_narrative_exports(
        experiment_volume=14,
        converged_tasks=9,
        accepted_commits=7,
        hottest_threads=["repo/ope-168", "repo/ope-170"],
    )

    assert "Accepted Commits: 7" in section
    assert "Repo Evidence Summary" in exports["markdown"]
    assert "Accepted Commits: 7" in exports["text"]
    assert "<section><h2>Repo Evidence Summary</h2>" in exports["html"]
