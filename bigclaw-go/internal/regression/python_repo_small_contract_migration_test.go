package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPythonRepoSmallContractMigration(t *testing.T) {
	repoRoot := repoRoot(t)
	payload := runPythonRepoSmallContracts(t, repoRoot)

	memory := payload.Memory
	if !memory.MatchedPreviousTask || !memory.InheritedAcceptanceContains || !memory.InheritedValidationContains || !memory.PreservedCurrentAcceptanceContains {
		t.Fatalf("unexpected memory contract: %+v", memory)
	}

	collab := payload.RepoCollaboration
	if !collab.MergedExists || collab.MergedSurface != "merged" || collab.CommentCount != 2 || collab.DecisionCount != 1 || collab.SecondCommentBody != "repo board context" {
		t.Fatalf("unexpected repo collaboration contract: %+v", collab)
	}

	registry := payload.RepoRegistry
	if !registry.ResolvedExists || registry.ResolvedRepo != "OpenAGIs/BigClaw" || registry.Channel != "bigclaw-ope-141" || registry.AgentID != "agent-native-cloud" || !registry.RestoredExists || registry.RestoredAgentID != "agent-native-cloud" {
		t.Fatalf("unexpected repo registry contract: %+v", registry)
	}

	rollout := payload.RepoRollout
	if rollout.Recommendation != "go" || rollout.CandidateGate != "enable-by-default" || !rollout.ReportContainsCandidateGate || !rollout.SectionContainsAcceptedCommits || !rollout.MarkdownContainsSummary || !rollout.TextContainsAcceptedCommits || !rollout.HTMLContainsSummary {
		t.Fatalf("unexpected repo rollout contract: %+v", rollout)
	}

	scheduler := payload.Scheduler
	if scheduler.HighRiskMedium != "vm" || scheduler.HighRiskApproved {
		t.Fatalf("unexpected high-risk scheduler decision: %+v", scheduler)
	}
	if scheduler.BrowserMedium != "browser" || !scheduler.BrowserApproved {
		t.Fatalf("unexpected browser scheduler decision: %+v", scheduler)
	}
	if scheduler.BudgetBrowserMedium != "docker" || !scheduler.BudgetBrowserApproved || !strings.Contains(scheduler.BudgetBrowserReason, "budget degraded browser route to docker") {
		t.Fatalf("unexpected budget browser scheduler decision: %+v", scheduler)
	}
	if scheduler.TinyBudgetMedium != "none" || scheduler.TinyBudgetApproved || scheduler.TinyBudgetReason != "paused: budget 5.0 below required docker budget 10.0" {
		t.Fatalf("unexpected tiny-budget scheduler decision: %+v", scheduler)
	}
}

type pythonRepoSmallContractsPayload struct {
	Memory struct {
		MatchedPreviousTask                bool `json:"matched_previous_task"`
		InheritedAcceptanceContains        bool `json:"inherited_acceptance_contains"`
		InheritedValidationContains        bool `json:"inherited_validation_contains"`
		PreservedCurrentAcceptanceContains bool `json:"preserved_current_acceptance_contains"`
	} `json:"memory"`
	RepoCollaboration struct {
		MergedExists      bool   `json:"merged_exists"`
		MergedSurface     string `json:"merged_surface"`
		CommentCount      int    `json:"comment_count"`
		DecisionCount     int    `json:"decision_count"`
		SecondCommentBody string `json:"second_comment_body"`
	} `json:"repo_collaboration"`
	RepoRegistry struct {
		ResolvedExists  bool   `json:"resolved_exists"`
		ResolvedRepo    string `json:"resolved_repo"`
		Channel         string `json:"channel"`
		AgentID         string `json:"agent_id"`
		RestoredExists  bool   `json:"restored_exists"`
		RestoredAgentID string `json:"restored_agent_id"`
	} `json:"repo_registry"`
	RepoRollout struct {
		Recommendation                 string `json:"recommendation"`
		CandidateGate                  string `json:"candidate_gate"`
		ReportContainsCandidateGate    bool   `json:"report_contains_candidate_gate"`
		SectionContainsAcceptedCommits bool   `json:"section_contains_accepted_commits"`
		MarkdownContainsSummary        bool   `json:"markdown_contains_summary"`
		TextContainsAcceptedCommits    bool   `json:"text_contains_accepted_commits"`
		HTMLContainsSummary            bool   `json:"html_contains_summary"`
	} `json:"repo_rollout"`
	Scheduler struct {
		HighRiskMedium        string `json:"high_risk_medium"`
		HighRiskApproved      bool   `json:"high_risk_approved"`
		BrowserMedium         string `json:"browser_medium"`
		BrowserApproved       bool   `json:"browser_approved"`
		BudgetBrowserMedium   string `json:"budget_browser_medium"`
		BudgetBrowserApproved bool   `json:"budget_browser_approved"`
		BudgetBrowserReason   string `json:"budget_browser_reason"`
		TinyBudgetMedium      string `json:"tiny_budget_medium"`
		TinyBudgetApproved    bool   `json:"tiny_budget_approved"`
		TinyBudgetReason      string `json:"tiny_budget_reason"`
	} `json:"scheduler"`
}

func runPythonRepoSmallContracts(t *testing.T, repoRoot string) pythonRepoSmallContractsPayload {
	t.Helper()

	code := `
import json
import tempfile
from pathlib import Path

from bigclaw.memory import TaskMemoryStore
from bigclaw.models import RiskLevel, Task
from bigclaw.scheduler import Scheduler
from bigclaw.collaboration import CollaborationComment, DecisionNote, build_collaboration_thread, merge_collaboration_threads
from bigclaw.repo_board import RepoDiscussionBoard
from bigclaw.repo_plane import RepoSpace
from bigclaw.repo_registry import RepoRegistry
from bigclaw.planning import EntryGateDecision, build_pilot_rollout_scorecard, evaluate_candidate_gate, render_pilot_rollout_gate_report
from bigclaw.reports import render_repo_narrative_exports, render_weekly_repo_evidence_section

with tempfile.TemporaryDirectory() as tmpdir:
    store = TaskMemoryStore(str(Path(tmpdir) / "memory" / "task-patterns.json"))
    previous = Task(
        task_id="BIG-501-prev",
        source="github",
        title="Previous queue rollout",
        description="",
        labels=["queue", "platform"],
        required_tools=["github", "browser"],
        acceptance_criteria=["report-shared"],
        validation_plan=["pytest", "smoke-test"],
    )
    store.remember_success(previous, summary="queue migration done")
    current = Task(
        task_id="BIG-501-new",
        source="github",
        title="New queue hardening",
        description="",
        labels=["queue"],
        required_tools=["github"],
        acceptance_criteria=["unit-tests"],
        validation_plan=["pytest"],
    )
    suggestion = store.suggest_rules(current)

native = build_collaboration_thread(
    "run",
    "run-165",
    comments=[CollaborationComment(comment_id="c1", author="ops", body="native note", created_at="2026-03-12T10:00:00Z")],
    decisions=[DecisionNote(decision_id="d1", author="lead", outcome="approved", summary="native decision", recorded_at="2026-03-12T10:05:00Z")],
)
board = RepoDiscussionBoard()
repo_post = board.create_post(
    channel="bigclaw-ope-165",
    author="repo-agent",
    body="repo board context",
    target_surface="run",
    target_id="run-165",
)
repo_thread = build_collaboration_thread(
    "repo-board",
    "run-165",
    comments=[repo_post.to_collaboration_comment()],
)
merged = merge_collaboration_threads(target_id="run-165", native_thread=native, repo_thread=repo_thread)

registry = RepoRegistry()
registry.register_space(
    RepoSpace(
        space_id="space-1",
        project_key="BIGCLAW",
        repo="OpenAGIs/BigClaw",
        sidecar_url="http://127.0.0.1:4041",
        health_state="healthy",
    )
)
task = Task(task_id="OPE-141", source="linear", title="repo registry", description="")
resolved = registry.resolve_space("BIGCLAW")
channel = registry.resolve_default_channel("BIGCLAW", task)
agent = registry.resolve_agent("native cloud", role="reviewer")
serialized = registry.to_dict()
restored = RepoRegistry.from_dict(serialized)

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

scheduler = Scheduler()
high_risk = scheduler.decide(Task(task_id="x", source="jira", title="prod op", description="", risk_level=RiskLevel.HIGH))
browser = scheduler.decide(Task(task_id="y", source="github", title="ui test", description="", required_tools=["browser"]))
budget_browser = scheduler.decide(Task(task_id="z", source="github", title="budgeted ui test", description="", required_tools=["browser"], budget=15.0))
tiny_budget = scheduler.decide(Task(task_id="b", source="linear", title="tiny budget", description="", budget=5.0))

print(json.dumps({
    "memory": {
        "matched_previous_task": "BIG-501-prev" in suggestion["matched_task_ids"],
        "inherited_acceptance_contains": "report-shared" in suggestion["acceptance_criteria"],
        "inherited_validation_contains": "smoke-test" in suggestion["validation_plan"],
        "preserved_current_acceptance_contains": "unit-tests" in suggestion["acceptance_criteria"],
    },
    "repo_collaboration": {
        "merged_exists": merged is not None,
        "merged_surface": merged.surface if merged is not None else "",
        "comment_count": len(merged.comments) if merged is not None else 0,
        "decision_count": len(merged.decisions) if merged is not None else 0,
        "second_comment_body": merged.comments[1].body if merged is not None and len(merged.comments) > 1 else "",
    },
    "repo_registry": {
        "resolved_exists": resolved is not None,
        "resolved_repo": resolved.repo if resolved is not None else "",
        "channel": channel,
        "agent_id": agent.repo_agent_id,
        "restored_exists": restored.resolve_space("BIGCLAW") is not None,
        "restored_agent_id": restored.resolve_agent("native cloud").repo_agent_id,
    },
    "repo_rollout": {
        "recommendation": scorecard["recommendation"],
        "candidate_gate": result["candidate_gate"],
        "report_contains_candidate_gate": "Candidate gate" in report,
        "section_contains_accepted_commits": "Accepted Commits: 7" in section,
        "markdown_contains_summary": "Repo Evidence Summary" in exports["markdown"],
        "text_contains_accepted_commits": "Accepted Commits: 7" in exports["text"],
        "html_contains_summary": "<section><h2>Repo Evidence Summary</h2>" in exports["html"],
    },
    "scheduler": {
        "high_risk_medium": high_risk.medium,
        "high_risk_approved": high_risk.approved,
        "browser_medium": browser.medium,
        "browser_approved": browser.approved,
        "budget_browser_medium": budget_browser.medium,
        "budget_browser_approved": budget_browser.approved,
        "budget_browser_reason": budget_browser.reason,
        "tiny_budget_medium": tiny_budget.medium,
        "tiny_budget_approved": tiny_budget.approved,
        "tiny_budget_reason": tiny_budget.reason,
    },
}))
`

	cmd := exec.Command("python3", "-c", code)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "PYTHONPATH="+filepath.Join(repoRoot, "src"))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run python repo small contracts: %v\n%s", err, output)
	}

	var payload pythonRepoSmallContractsPayload
	if err := json.Unmarshal(output, &payload); err != nil {
		t.Fatalf("decode python repo small contracts payload: %v\n%s", err, output)
	}
	return payload
}
