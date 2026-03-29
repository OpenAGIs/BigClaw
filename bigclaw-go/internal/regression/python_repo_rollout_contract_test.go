package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLane8PythonRepoRolloutContractStaysAligned(t *testing.T) {
	goRepoRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))
	scriptPath := filepath.Join(t.TempDir(), "repo_rollout_contract.py")
	script := `import json
import sys
from pathlib import Path

repo_root = Path(sys.argv[1])
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.planning import (
    EntryGateDecision,
    build_pilot_rollout_scorecard,
    evaluate_candidate_gate,
    render_pilot_rollout_gate_report,
)
from bigclaw.reports import render_repo_narrative_exports, render_weekly_repo_evidence_section

scorecard = build_pilot_rollout_scorecard(
    adoption=84,
    convergence_improvement=78,
    review_efficiency=82,
    governance_incidents=1,
    evidence_completeness=88,
)
gate_decision = EntryGateDecision(gate_id="gate-v3", passed=True)
result = evaluate_candidate_gate(gate_decision=gate_decision, rollout_scorecard=scorecard)
report = render_pilot_rollout_gate_report(result)

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

print(json.dumps({
    "scorecard": scorecard,
    "candidate_gate": result["candidate_gate"],
    "report_has_candidate_gate": "Candidate gate" in report,
    "section_has_commits": "Accepted Commits: 7" in section,
    "markdown_has_summary": "Repo Evidence Summary" in exports["markdown"],
    "text_has_commits": "Accepted Commits: 7" in exports["text"],
    "html_has_section": "<section><h2>Repo Evidence Summary</h2>" in exports["html"],
}))
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write repo rollout contract script: %v", err)
	}

	cmd := exec.Command("python3", scriptPath, repoRoot)
	cmd.Dir = goRepoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run repo rollout contract script: %v\n%s", err, string(output))
	}

	var decoded struct {
		Scorecard struct {
			Recommendation string `json:"recommendation"`
		} `json:"scorecard"`
		CandidateGate          string `json:"candidate_gate"`
		ReportHasCandidateGate bool   `json:"report_has_candidate_gate"`
		SectionHasCommits      bool   `json:"section_has_commits"`
		MarkdownHasSummary     bool   `json:"markdown_has_summary"`
		TextHasCommits         bool   `json:"text_has_commits"`
		HTMLHasSection         bool   `json:"html_has_section"`
	}
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("decode repo rollout contract output: %v\n%s", err, string(output))
	}

	if decoded.Scorecard.Recommendation != "go" ||
		decoded.CandidateGate != "enable-by-default" ||
		!decoded.ReportHasCandidateGate ||
		!decoded.SectionHasCommits ||
		!decoded.MarkdownHasSummary ||
		!decoded.TextHasCommits ||
		!decoded.HTMLHasSection {
		t.Fatalf("unexpected repo rollout payload: %+v", decoded)
	}
}
