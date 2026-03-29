package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLane8PythonExecutionFlowContractStaysAligned(t *testing.T) {
	goRepoRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))
	scriptPath := filepath.Join(t.TempDir(), "execution_flow_contract.py")
	script := `import json
import sys
import tempfile
from pathlib import Path

repo_root = Path(sys.argv[1])
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.models import Priority, RiskLevel, Task
from bigclaw.observability import ObservabilityLedger
from bigclaw.queue import PersistentTaskQueue
from bigclaw.scheduler import Scheduler

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    queue = PersistentTaskQueue(str(td / "queue.json"))
    ledger = ObservabilityLedger(str(td / "ledger.json"))
    report_path = td / "reports" / "run-1.md"

    queue.enqueue(
        Task(
            task_id="BIG-502",
            source="linear",
            title="Record execution",
            description="full chain",
            priority=Priority.P0,
            risk_level=RiskLevel.MEDIUM,
            required_tools=["browser"],
        )
    )
    task = queue.dequeue_task()
    record = Scheduler().execute(task, run_id="run-1", ledger=ledger, report_path=str(report_path))
    entries = ledger.load()
    queue_chain = {
        "medium": record.decision.medium,
        "approved": record.decision.approved,
        "status": record.run.status,
        "report_exists": report_path.exists(),
        "page_exists": report_path.with_suffix(".html").exists(),
        "report_has_status": "Status: approved" in report_path.read_text(),
        "entry_count": len(entries),
        "trace_span": entries[0]["traces"][0]["span"],
        "artifact_kinds": [item["kind"] for item in entries[0]["artifacts"]],
        "audit_reason": entries[0]["audits"][0]["details"]["reason"],
    }

with tempfile.TemporaryDirectory() as td:
    ledger = ObservabilityLedger(str(Path(td) / "ledger.json"))
    task = Task(
        task_id="BIG-502-risk",
        source="jira",
        title="Prod change",
        description="manual review",
        risk_level=RiskLevel.HIGH,
    )
    record = Scheduler().execute(task, run_id="run-2", ledger=ledger)
    approval = {
        "approved": record.decision.approved,
        "status": record.run.status,
        "audit_outcome": ledger.load()[0]["audits"][0]["outcome"],
    }

print(json.dumps({
    "queue_chain": queue_chain,
    "approval": approval,
}))
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write execution flow contract script: %v", err)
	}

	cmd := exec.Command("python3", scriptPath, repoRoot)
	cmd.Dir = goRepoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run execution flow contract script: %v\n%s", err, string(output))
	}

	var decoded struct {
		QueueChain struct {
			Medium          string   `json:"medium"`
			Approved        bool     `json:"approved"`
			Status          string   `json:"status"`
			ReportExists    bool     `json:"report_exists"`
			PageExists      bool     `json:"page_exists"`
			ReportHasStatus bool     `json:"report_has_status"`
			EntryCount      int      `json:"entry_count"`
			TraceSpan       string   `json:"trace_span"`
			ArtifactKinds   []string `json:"artifact_kinds"`
			AuditReason     string   `json:"audit_reason"`
		} `json:"queue_chain"`
		Approval struct {
			Approved     bool   `json:"approved"`
			Status       string `json:"status"`
			AuditOutcome string `json:"audit_outcome"`
		} `json:"approval"`
	}
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("decode execution flow contract output: %v\n%s", err, string(output))
	}

	if decoded.QueueChain.Medium != "browser" || !decoded.QueueChain.Approved || decoded.QueueChain.Status != "approved" || !decoded.QueueChain.ReportExists || !decoded.QueueChain.PageExists || !decoded.QueueChain.ReportHasStatus || decoded.QueueChain.EntryCount != 1 || decoded.QueueChain.TraceSpan != "scheduler.decide" || len(decoded.QueueChain.ArtifactKinds) != 2 || decoded.QueueChain.ArtifactKinds[0] != "page" || decoded.QueueChain.ArtifactKinds[1] != "report" || decoded.QueueChain.AuditReason != "browser automation task" {
		t.Fatalf("unexpected queue-to-scheduler payload: %+v", decoded.QueueChain)
	}
	if decoded.Approval.Approved || decoded.Approval.Status != "needs-approval" || decoded.Approval.AuditOutcome != "pending" {
		t.Fatalf("unexpected high-risk approval payload: %+v", decoded.Approval)
	}
}
