package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLane8PythonRuntimeMatrixContractStaysAligned(t *testing.T) {
	goRepoRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))
scriptPath := filepath.Join(t.TempDir(), "runtime_matrix_contract.py")
script := `import json
import sys
import tempfile
from pathlib import Path

repo_root = Path(sys.argv[1])
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.models import RiskLevel, Task
from bigclaw.observability import ObservabilityLedger, TaskRun
from bigclaw.runtime import ClawWorkerRuntime, ToolPolicy, ToolRuntime
from bigclaw.scheduler import Scheduler

task = Task(
    task_id="BIG-301-matrix",
    source="github",
    title="worker lifecycle matrix",
    description="validate stable lifecycle",
    required_tools=["github", "browser"],
)
run = TaskRun.from_task(task, run_id="run-big301-matrix", medium="docker")
runtime = ToolRuntime(
    handlers={
        "github": lambda action, payload: f"{action}:{payload.get('repo', 'none')}",
        "browser": lambda action, payload: f"{action}:{payload.get('url', 'none')}",
    }
)
worker = ClawWorkerRuntime(tool_runtime=runtime)
result = worker.execute(
    task,
    decision=type("Decision", (), {"medium": "docker", "approved": True, "reason": "ok"})(),
    run=run,
    tool_payloads={"github": {"repo": "OpenAGIs/BigClaw"}, "browser": {"url": "https://example.com"}},
)

scheduler = Scheduler()
low = scheduler.decide(Task(task_id="low", source="local", title="low", description="", risk_level=RiskLevel.LOW))
high = scheduler.decide(Task(task_id="high", source="local", title="high", description="", risk_level=RiskLevel.HIGH))
browser = scheduler.decide(
    Task(task_id="browser", source="local", title="browser", description="", required_tools=["browser"], risk_level=RiskLevel.MEDIUM)
)
with tempfile.TemporaryDirectory() as td:
    budget = scheduler.execute(
        Task(
            task_id="budget",
            source="local",
            title="budget pause",
            description="budget should stop execution",
            budget=5.0,
            required_tools=["github"],
        ),
        run_id="run-budget-pause",
        ledger=ObservabilityLedger(str(Path(td) / "ledger.json")),
    )

task = Task(task_id="BIG-303-matrix", source="local", title="tool policy", description="", required_tools=["github", "browser"])
run2 = TaskRun.from_task(task, run_id="run-big303-matrix", medium="docker")
runtime2 = ToolRuntime(
    policy=ToolPolicy(allowed_tools=["github"], blocked_tools=["browser"]),
    handlers={"github": lambda action, payload: "ok"},
)
allow = runtime2.invoke("github", action="execute", payload={"repo": "OpenAGIs/BigClaw"}, run=run2)
block = runtime2.invoke("browser", action="execute", payload={"url": "https://example.com"}, run=run2)

print(json.dumps({
    "matrix301": {
        "tool_results": len(result.tool_results),
        "all_success": all(item.success for item in result.tool_results),
        "last_action": run.audits[-1].action,
        "last_outcome": run.audits[-1].outcome,
    },
    "matrix302": {
        "low_medium": low.medium,
        "high_medium": high.medium,
        "browser_medium": browser.medium,
        "budget_medium": budget.decision.medium,
        "budget_approved": budget.decision.approved,
        "budget_status": budget.run.status,
        "budget_tool_results": len(budget.tool_results),
        "budget_last_action": budget.run.audits[-1].action,
        "budget_last_outcome": budget.run.audits[-1].outcome,
    },
    "matrix303": {
        "allow_success": allow.success,
        "block_success": block.success,
        "outcomes": [audit.outcome for audit in run2.audits if audit.action == "tool.invoke"],
    },
}))
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write runtime matrix contract script: %v", err)
	}

	cmd := exec.Command("python3", scriptPath, repoRoot)
	cmd.Dir = goRepoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run runtime matrix contract script: %v\n%s", err, string(output))
	}

	var decoded struct {
		Matrix301 struct {
			ToolResults int    `json:"tool_results"`
			AllSuccess  bool   `json:"all_success"`
			LastAction  string `json:"last_action"`
			LastOutcome string `json:"last_outcome"`
		} `json:"matrix301"`
		Matrix302 struct {
			LowMedium        string `json:"low_medium"`
			HighMedium       string `json:"high_medium"`
			BrowserMedium    string `json:"browser_medium"`
			BudgetMedium     string `json:"budget_medium"`
			BudgetApproved   bool   `json:"budget_approved"`
			BudgetStatus     string `json:"budget_status"`
			BudgetToolResults int   `json:"budget_tool_results"`
			BudgetLastAction string `json:"budget_last_action"`
			BudgetLastOutcome string `json:"budget_last_outcome"`
		} `json:"matrix302"`
		Matrix303 struct {
			AllowSuccess bool     `json:"allow_success"`
			BlockSuccess bool     `json:"block_success"`
			Outcomes     []string `json:"outcomes"`
		} `json:"matrix303"`
	}
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("decode runtime matrix contract output: %v\n%s", err, string(output))
	}

	if decoded.Matrix301.ToolResults != 2 || !decoded.Matrix301.AllSuccess || decoded.Matrix301.LastAction != "worker.lifecycle" || decoded.Matrix301.LastOutcome != "completed" {
		t.Fatalf("unexpected worker lifecycle payload: %+v", decoded.Matrix301)
	}
	if decoded.Matrix302.LowMedium != "docker" || decoded.Matrix302.HighMedium != "vm" {
		t.Fatalf("unexpected scheduler medium mapping: %+v", decoded.Matrix302)
	}
	if decoded.Matrix302.BrowserMedium != "browser" && decoded.Matrix302.BrowserMedium != "docker" {
		t.Fatalf("unexpected browser scheduler medium: %+v", decoded.Matrix302)
	}
	if decoded.Matrix302.BudgetMedium != "none" || decoded.Matrix302.BudgetApproved || decoded.Matrix302.BudgetStatus != "paused" || decoded.Matrix302.BudgetToolResults != 0 || decoded.Matrix302.BudgetLastAction != "worker.lifecycle" || decoded.Matrix302.BudgetLastOutcome != "paused" {
		t.Fatalf("unexpected budget pause payload: %+v", decoded.Matrix302)
	}
	if !decoded.Matrix303.AllowSuccess || decoded.Matrix303.BlockSuccess {
		t.Fatalf("unexpected tool policy result payload: %+v", decoded.Matrix303)
	}
	if len(decoded.Matrix303.Outcomes) != 2 || decoded.Matrix303.Outcomes[0] != "success" || decoded.Matrix303.Outcomes[1] != "blocked" {
		t.Fatalf("unexpected tool policy audit outcomes: %+v", decoded.Matrix303.Outcomes)
	}
}
