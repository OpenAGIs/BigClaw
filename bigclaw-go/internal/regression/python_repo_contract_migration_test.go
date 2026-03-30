package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPythonRepoContractMigration(t *testing.T) {
	repoRoot := repoRoot(t)
	payload := runPythonRepoContracts(t, repoRoot)

	validation := payload.ValidationPolicy
	if validation.BlockedAllowedToClose || validation.BlockedStatus != "blocked" || !containsExactString(validation.BlockedMissingReports, "benchmark-suite") {
		t.Fatalf("unexpected blocked validation policy result: %+v", validation)
	}
	if !validation.ReadyAllowedToClose || validation.ReadyStatus != "ready" {
		t.Fatalf("unexpected ready validation policy result: %+v", validation)
	}

	governance := payload.RepoGovernance
	if !governance.EngLeadCanPush || !governance.ReviewerCanAccept || governance.ExecutionAgentCanPush {
		t.Fatalf("unexpected repo governance permission matrix: %+v", governance)
	}
	if len(governance.MissingAuditFields) != 2 || governance.MissingAuditFields[0] != "accepted_commit_hash" || governance.MissingAuditFields[1] != "reviewer" {
		t.Fatalf("unexpected repo governance audit fields: %+v", governance.MissingAuditFields)
	}

	board := payload.RepoBoard
	if board.PostID != "post-1" || board.ReplyParentPostID != "post-1" {
		t.Fatalf("unexpected repo board ids: %+v", board)
	}
	if board.RunPostCount != 2 || board.RunPostChannel != "bigclaw-ope-164" || board.CommentAnchor != "run:run-164" || board.CommentBodyPrefix != "Need reviewer" {
		t.Fatalf("unexpected repo board contract: %+v", board)
	}

	links := payload.RepoLinks
	if links.BindingAcceptedCommitHash != "ccc333" || links.CloseoutAcceptedCommitHash != "ccc333" || links.RestoredCandidateRole != "candidate" {
		t.Fatalf("unexpected repo links contract: %+v", links)
	}

	triage := payload.RepoTriage
	if triage.NeedsApprovalAction != "approve" || triage.FailedAction != "replay" {
		t.Fatalf("unexpected repo triage recommendations: %+v", triage)
	}
	if triage.PacketAcceptedCommitHash != "def222" || triage.PacketCandidateCommitHash != "abc111" || triage.PacketLineageSummary != "candidate descends from accepted baseline" {
		t.Fatalf("unexpected repo triage packet: %+v", triage)
	}
}

type pythonRepoContractsPayload struct {
	ValidationPolicy struct {
		BlockedAllowedToClose bool     `json:"blocked_allowed_to_close"`
		BlockedStatus         string   `json:"blocked_status"`
		BlockedMissingReports []string `json:"blocked_missing_reports"`
		ReadyAllowedToClose   bool     `json:"ready_allowed_to_close"`
		ReadyStatus           string   `json:"ready_status"`
	} `json:"validation_policy"`
	RepoGovernance struct {
		EngLeadCanPush        bool     `json:"eng_lead_can_push"`
		ReviewerCanAccept     bool     `json:"reviewer_can_accept"`
		ExecutionAgentCanPush bool     `json:"execution_agent_can_push"`
		MissingAuditFields    []string `json:"missing_audit_fields"`
	} `json:"repo_governance"`
	RepoBoard struct {
		PostID            string `json:"post_id"`
		ReplyParentPostID string `json:"reply_parent_post_id"`
		RunPostCount      int    `json:"run_post_count"`
		RunPostChannel    string `json:"run_post_channel"`
		CommentAnchor     string `json:"comment_anchor"`
		CommentBodyPrefix string `json:"comment_body_prefix"`
	} `json:"repo_board"`
	RepoLinks struct {
		BindingAcceptedCommitHash  string `json:"binding_accepted_commit_hash"`
		CloseoutAcceptedCommitHash string `json:"closeout_accepted_commit_hash"`
		RestoredCandidateRole      string `json:"restored_candidate_role"`
	} `json:"repo_links"`
	RepoTriage struct {
		NeedsApprovalAction       string `json:"needs_approval_action"`
		FailedAction              string `json:"failed_action"`
		PacketAcceptedCommitHash  string `json:"packet_accepted_commit_hash"`
		PacketCandidateCommitHash string `json:"packet_candidate_commit_hash"`
		PacketLineageSummary      string `json:"packet_lineage_summary"`
	} `json:"repo_triage"`
}

func runPythonRepoContracts(t *testing.T, repoRoot string) pythonRepoContractsPayload {
	t.Helper()

	code := `
import json
from bigclaw.validation_policy import enforce_validation_report_policy
from bigclaw.repo_governance import RepoPermissionContract, missing_repo_audit_fields
from bigclaw.repo_board import RepoDiscussionBoard
from bigclaw.models import Task
from bigclaw.observability import TaskRun
from bigclaw.repo_links import bind_run_commits
from bigclaw.repo_plane import RunCommitLink
from bigclaw.repo_triage import LineageEvidence, approval_evidence_packet, recommend_triage_action

blocked = enforce_validation_report_policy(["task-run", "replay"])
ready = enforce_validation_report_policy(["task-run", "replay", "benchmark-suite"])

contract = RepoPermissionContract()
missing = missing_repo_audit_fields(
    "repo.accept",
    {
        "task_id": "OPE-172",
        "run_id": "run-172",
        "repo_space_id": "space-1",
        "actor": "reviewer",
    },
)

board = RepoDiscussionBoard()
post = board.create_post(
    channel="bigclaw-ope-164",
    author="agent-a",
    body="Need reviewer on commit lineage",
    target_surface="run",
    target_id="run-164",
    metadata={"severity": "p1"},
)
reply = board.reply(parent_post_id=post.post_id, author="reviewer", body="I will review this now")
run_posts = board.list_posts(target_surface="run", target_id="run-164")
comment = run_posts[0].to_collaboration_comment()

task = Task(task_id="OPE-143", source="linear", title="run links", description="")
run = TaskRun.from_task(task, run_id="run-143", medium="docker")
links = [
    RunCommitLink(run_id=run.run_id, commit_hash="aaa111", role="source", repo_space_id="space-1"),
    RunCommitLink(run_id=run.run_id, commit_hash="bbb222", role="candidate", repo_space_id="space-1"),
    RunCommitLink(run_id=run.run_id, commit_hash="ccc333", role="accepted", repo_space_id="space-1"),
]
binding = bind_run_commits(links)
run.record_closeout(
    validation_evidence=["pytest tests/test_repo_links.py"],
    git_push_succeeded=True,
    git_log_stat_output="commit ccc333",
    run_commit_links=links,
)
restored = TaskRun.from_dict(run.to_dict())

approve = recommend_triage_action(
    status="needs-approval",
    evidence=LineageEvidence(candidate_commit="abc", accepted_ancestor="0001", similar_failure_count=0),
)
replay = recommend_triage_action(
    status="failed",
    evidence=LineageEvidence(candidate_commit="abc", similar_failure_count=3),
)
packet = approval_evidence_packet(
    run_id="run-170",
    links=[
        {"role": "candidate", "commit_hash": "abc111"},
        {"role": "accepted", "commit_hash": "def222"},
    ],
    lineage_summary="candidate descends from accepted baseline",
)

print(json.dumps({
    "validation_policy": {
        "blocked_allowed_to_close": blocked.allowed_to_close,
        "blocked_status": blocked.status,
        "blocked_missing_reports": blocked.missing_reports,
        "ready_allowed_to_close": ready.allowed_to_close,
        "ready_status": ready.status,
    },
    "repo_governance": {
        "eng_lead_can_push": contract.check(action_permission="repo.push", actor_roles=["eng-lead"]),
        "reviewer_can_accept": contract.check(action_permission="repo.accept", actor_roles=["reviewer"]),
        "execution_agent_can_push": contract.check(action_permission="repo.push", actor_roles=["execution-agent"]),
        "missing_audit_fields": missing,
    },
    "repo_board": {
        "post_id": post.post_id,
        "reply_parent_post_id": reply.parent_post_id,
        "run_post_count": len(run_posts),
        "run_post_channel": run_posts[0].channel,
        "comment_anchor": comment.anchor,
        "comment_body_prefix": comment.body[:13],
    },
    "repo_links": {
        "binding_accepted_commit_hash": binding.accepted_commit_hash,
        "closeout_accepted_commit_hash": run.closeout.accepted_commit_hash,
        "restored_candidate_role": restored.closeout.run_commit_links[1].role,
    },
    "repo_triage": {
        "needs_approval_action": approve.action,
        "failed_action": replay.action,
        "packet_accepted_commit_hash": packet["accepted_commit_hash"],
        "packet_candidate_commit_hash": packet["candidate_commit_hash"],
        "packet_lineage_summary": packet["lineage_summary"],
    },
}))
`

	cmd := exec.Command("python3", "-c", code)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "PYTHONPATH="+filepath.Join(repoRoot, "src"))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run python repo contracts: %v\n%s", err, output)
	}

	var payload pythonRepoContractsPayload
	if err := json.Unmarshal(output, &payload); err != nil {
		t.Fatalf("decode python repo contracts payload: %v\n%s", err, output)
	}
	return payload
}

func containsExactString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
