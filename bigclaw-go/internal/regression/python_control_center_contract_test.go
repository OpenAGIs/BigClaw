package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLane8PythonControlCenterContractStaysAligned(t *testing.T) {
	goRepoRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))
	scriptPath := filepath.Join(t.TempDir(), "control_center_contract.py")
	script := `import json
import sys
import tempfile
from pathlib import Path

repo_root = Path(sys.argv[1])
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.models import Priority, RiskLevel, Task
from bigclaw.operations import OperationsAnalytics, render_queue_control_center
from bigclaw.queue import PersistentTaskQueue
from bigclaw.reports import SharedViewContext, SharedViewFilter

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    queue = PersistentTaskQueue(str(td / "queue.json"))
    queue.enqueue(Task(task_id="p2", source="linear", title="low", description="", priority=Priority.P2))
    queue.enqueue(Task(task_id="p0", source="linear", title="top", description="", priority=Priority.P0))
    queue.enqueue(Task(task_id="p1", source="linear", title="mid", description="", priority=Priority.P1))
    ordering = {"task_ids": [task.task_id for task in queue.peek_tasks()]}

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    queue = PersistentTaskQueue(str(td / "queue.json"))
    queue.enqueue(Task(task_id="BIG-802-1", source="linear", title="top", description="", priority=Priority.P0, risk_level=RiskLevel.HIGH))
    queue.enqueue(Task(task_id="BIG-802-2", source="linear", title="mid", description="", priority=Priority.P1, risk_level=RiskLevel.MEDIUM))
    queue.enqueue(Task(task_id="BIG-802-3", source="linear", title="low", description="", priority=Priority.P2, risk_level=RiskLevel.LOW))
    center = OperationsAnalytics().build_queue_control_center(
        queue,
        runs=[
            {"task_id": "BIG-802-1", "status": "needs-approval", "medium": "vm"},
            {"task_id": "BIG-802-2", "status": "approved", "medium": "browser"},
            {"task_id": "BIG-802-4", "status": "approved", "medium": "docker"},
        ],
    )
    report = render_queue_control_center(center)
    summary = {
        "queue_depth": center.queue_depth,
        "queued_by_priority": center.queued_by_priority,
        "queued_by_risk": center.queued_by_risk,
        "execution_media": center.execution_media,
        "waiting_approval_runs": center.waiting_approval_runs,
        "blocked_tasks": center.blocked_tasks,
        "queued_tasks": center.queued_tasks,
        "action_ids": [action.action_id for action in center.actions["BIG-802-1"]],
        "escalate_enabled": center.actions["BIG-802-1"][3].enabled,
        "retry_enabled": center.actions["BIG-802-1"][4].enabled,
        "pause_enabled": center.actions["BIG-802-1"][5].enabled,
        "has_title": "# Queue Control Center" in report,
        "has_waiting": "- Waiting Approval Runs: 1" in report,
        "has_task": "- BIG-802-1" in report,
        "has_drill_down": "BIG-802-1: Drill Down [drill-down]" in report,
        "has_escalate": "Escalate [escalate] state=enabled" in report,
        "has_pause_reason": "Pause [pause] state=disabled target=BIG-802-1 reason=approval-blocked tasks should be escalated instead of paused" in report,
    }

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    queue = PersistentTaskQueue(str(td / "queue.json"))
    center = OperationsAnalytics().build_queue_control_center(queue, runs=[])
    report = render_queue_control_center(
        center,
        view=SharedViewContext(
            filters=[SharedViewFilter(label="Team", value="operations")],
            result_count=0,
            empty_message="No queued work for the selected team.",
        ),
    )
    empty = {
        "has_view_state": "## View State" in report,
        "has_empty_state": "- State: empty" in report,
        "has_summary": "- Summary: No queued work for the selected team." in report,
        "has_filter": "- Team: operations" in report,
    }

print(json.dumps({
    "ordering": ordering,
    "summary": summary,
    "empty": empty,
}))
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write control center contract script: %v", err)
	}

	cmd := exec.Command("python3", scriptPath, repoRoot)
	cmd.Dir = goRepoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run control center contract script: %v\n%s", err, string(output))
	}

	var decoded struct {
		Ordering struct {
			TaskIDs []string `json:"task_ids"`
		} `json:"ordering"`
		Summary struct {
			QueueDepth          int            `json:"queue_depth"`
			QueuedByPriority    map[string]int `json:"queued_by_priority"`
			QueuedByRisk        map[string]int `json:"queued_by_risk"`
			ExecutionMedia      map[string]int `json:"execution_media"`
			WaitingApprovalRuns int            `json:"waiting_approval_runs"`
			BlockedTasks        []string       `json:"blocked_tasks"`
			QueuedTasks         []string       `json:"queued_tasks"`
			ActionIDs           []string       `json:"action_ids"`
			EscalateEnabled     bool           `json:"escalate_enabled"`
			RetryEnabled        bool           `json:"retry_enabled"`
			PauseEnabled        bool           `json:"pause_enabled"`
			HasTitle            bool           `json:"has_title"`
			HasWaiting          bool           `json:"has_waiting"`
			HasTask             bool           `json:"has_task"`
			HasDrillDown        bool           `json:"has_drill_down"`
			HasEscalate         bool           `json:"has_escalate"`
			HasPauseReason      bool           `json:"has_pause_reason"`
		} `json:"summary"`
		Empty struct {
			HasViewState bool `json:"has_view_state"`
			HasEmptyState bool `json:"has_empty_state"`
			HasSummary   bool `json:"has_summary"`
			HasFilter    bool `json:"has_filter"`
		} `json:"empty"`
	}
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("decode control center contract output: %v\n%s", err, string(output))
	}

	if len(decoded.Ordering.TaskIDs) != 3 || decoded.Ordering.TaskIDs[0] != "p0" || decoded.Ordering.TaskIDs[1] != "p1" || decoded.Ordering.TaskIDs[2] != "p2" {
		t.Fatalf("unexpected queue ordering payload: %+v", decoded.Ordering)
	}
	if decoded.Summary.QueueDepth != 3 ||
		decoded.Summary.QueuedByPriority["P0"] != 1 ||
		decoded.Summary.QueuedByPriority["P1"] != 1 ||
		decoded.Summary.QueuedByPriority["P2"] != 1 ||
		decoded.Summary.QueuedByRisk["low"] != 1 ||
		decoded.Summary.QueuedByRisk["medium"] != 1 ||
		decoded.Summary.QueuedByRisk["high"] != 1 ||
		decoded.Summary.ExecutionMedia["vm"] != 1 ||
		decoded.Summary.ExecutionMedia["browser"] != 1 ||
		decoded.Summary.ExecutionMedia["docker"] != 1 ||
		decoded.Summary.WaitingApprovalRuns != 1 ||
		len(decoded.Summary.BlockedTasks) != 1 || decoded.Summary.BlockedTasks[0] != "BIG-802-1" ||
		len(decoded.Summary.QueuedTasks) != 3 || decoded.Summary.QueuedTasks[0] != "BIG-802-1" || decoded.Summary.QueuedTasks[1] != "BIG-802-2" || decoded.Summary.QueuedTasks[2] != "BIG-802-3" {
		t.Fatalf("unexpected queue summary payload: %+v", decoded.Summary)
	}
	wantActions := []string{"drill-down", "export", "add-note", "escalate", "retry", "pause", "reassign", "audit"}
	if len(decoded.Summary.ActionIDs) != len(wantActions) {
		t.Fatalf("unexpected action ids: %+v", decoded.Summary.ActionIDs)
	}
	for i, want := range wantActions {
		if decoded.Summary.ActionIDs[i] != want {
			t.Fatalf("unexpected action ids: %+v", decoded.Summary.ActionIDs)
		}
	}
	if !decoded.Summary.EscalateEnabled || !decoded.Summary.RetryEnabled || decoded.Summary.PauseEnabled || !decoded.Summary.HasTitle || !decoded.Summary.HasWaiting || !decoded.Summary.HasTask || !decoded.Summary.HasDrillDown || !decoded.Summary.HasEscalate || !decoded.Summary.HasPauseReason {
		t.Fatalf("unexpected rendered control center payload: %+v", decoded.Summary)
	}
	if !decoded.Empty.HasViewState || !decoded.Empty.HasEmptyState || !decoded.Empty.HasSummary || !decoded.Empty.HasFilter {
		t.Fatalf("unexpected empty-state payload: %+v", decoded.Empty)
	}
}
