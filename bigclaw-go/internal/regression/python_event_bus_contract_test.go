package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLane8PythonEventBusContractStaysAligned(t *testing.T) {
	goRepoRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))
	scriptPath := filepath.Join(t.TempDir(), "event_bus_contract.py")
	script := `import json
import sys
import tempfile
from pathlib import Path

repo_root = Path(sys.argv[1])
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.event_bus import CI_COMPLETED_EVENT, PULL_REQUEST_COMMENT_EVENT, TASK_FAILED_EVENT, BusEvent, EventBus
from bigclaw.models import Task
from bigclaw.observability import ObservabilityLedger, TaskRun

with tempfile.TemporaryDirectory() as td:
    td = Path(td)

    ledger = ObservabilityLedger(str(td / "ledger1.json"))
    task = Task(task_id="BIG-203-pr", source="github", title="PR approval", description="")
    run = TaskRun.from_task(task, run_id="run-pr-1", medium="vm")
    run.finalize("needs-approval", "waiting for reviewer comment")
    ledger.append(run)
    bus = EventBus(ledger=ledger)
    seen = []
    bus.subscribe(PULL_REQUEST_COMMENT_EVENT, lambda _event, current: seen.append(current.status))
    updated = bus.publish(BusEvent(
        event_type=PULL_REQUEST_COMMENT_EVENT,
        run_id=run.run_id,
        actor="reviewer",
        details={"decision": "approved", "body": "LGTM, merge when green.", "mentions": ["ops"]},
    ))
    persisted = ledger.load()[0]
    pr = {
        "status": updated.status,
        "summary": updated.summary,
        "seen": seen,
        "persisted_status": persisted["status"],
        "has_comment": any(audit["action"] == "collaboration.comment" for audit in persisted["audits"]),
        "has_transition": any(audit["action"] == "event_bus.transition" and audit["details"]["previous_status"] == "needs-approval" for audit in persisted["audits"]),
    }

    ledger = ObservabilityLedger(str(td / "ledger2.json"))
    task = Task(task_id="BIG-203-ci", source="github", title="CI completion", description="")
    run = TaskRun.from_task(task, run_id="run-ci-1", medium="docker")
    run.finalize("approved", "waiting for CI")
    ledger.append(run)
    bus = EventBus(ledger=ledger)
    updated = bus.publish(BusEvent(
        event_type=CI_COMPLETED_EVENT,
        run_id=run.run_id,
        actor="github-actions",
        details={"workflow": "pytest", "conclusion": "success"},
    ))
    persisted = ledger.load()[0]
    ci = {
        "status": updated.status,
        "summary": updated.summary,
        "persisted_status": persisted["status"],
        "has_event": any(audit["action"] == "event_bus.event" and audit["details"]["event_type"] == CI_COMPLETED_EVENT for audit in persisted["audits"]),
    }

    ledger = ObservabilityLedger(str(td / "ledger3.json"))
    task = Task(task_id="BIG-203-fail", source="scheduler", title="Task failure", description="")
    run = TaskRun.from_task(task, run_id="run-fail-1", medium="docker")
    ledger.append(run)
    bus = EventBus(ledger=ledger)
    updated = bus.publish(BusEvent(
        event_type=TASK_FAILED_EVENT,
        run_id=run.run_id,
        actor="worker",
        details={"error": "sandbox command exited 137"},
    ))
    persisted = ledger.load()[0]
    failed = {
        "status": updated.status,
        "summary": updated.summary,
        "persisted_status": persisted["status"],
        "has_transition": any(audit["action"] == "event_bus.transition" and audit["details"]["status"] == "failed" for audit in persisted["audits"]),
    }

    print(json.dumps({"pr": pr, "ci": ci, "fail": failed}))
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write event bus contract script: %v", err)
	}

	cmd := exec.Command("python3", scriptPath, repoRoot)
	cmd.Dir = goRepoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run event bus contract script: %v\n%s", err, string(output))
	}

	var decoded struct {
		PR struct {
			Status          string   `json:"status"`
			Summary         string   `json:"summary"`
			Seen            []string `json:"seen"`
			PersistedStatus string   `json:"persisted_status"`
			HasComment      bool     `json:"has_comment"`
			HasTransition   bool     `json:"has_transition"`
		} `json:"pr"`
		CI struct {
			Status          string `json:"status"`
			Summary         string `json:"summary"`
			PersistedStatus string `json:"persisted_status"`
			HasEvent        bool   `json:"has_event"`
		} `json:"ci"`
		Fail struct {
			Status          string `json:"status"`
			Summary         string `json:"summary"`
			PersistedStatus string `json:"persisted_status"`
			HasTransition   bool   `json:"has_transition"`
		} `json:"fail"`
	}
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("decode event bus contract output: %v\n%s", err, string(output))
	}

	if decoded.PR.Status != "approved" || decoded.PR.Summary != "LGTM, merge when green." || decoded.PR.PersistedStatus != "approved" || !decoded.PR.HasComment || !decoded.PR.HasTransition {
		t.Fatalf("unexpected pr-comment payload: %+v", decoded.PR)
	}
	if len(decoded.PR.Seen) != 1 || decoded.PR.Seen[0] != "approved" {
		t.Fatalf("unexpected pr-comment subscriber statuses: %+v", decoded.PR.Seen)
	}
	if decoded.CI.Status != "completed" || decoded.CI.Summary != "CI workflow pytest completed with success" || decoded.CI.PersistedStatus != "completed" || !decoded.CI.HasEvent {
		t.Fatalf("unexpected ci-completed payload: %+v", decoded.CI)
	}
	if decoded.Fail.Status != "failed" || decoded.Fail.Summary != "sandbox command exited 137" || decoded.Fail.PersistedStatus != "failed" || !decoded.Fail.HasTransition {
		t.Fatalf("unexpected task-failed payload: %+v", decoded.Fail)
	}
}
