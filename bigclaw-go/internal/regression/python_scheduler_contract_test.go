package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLane8PythonSchedulerContractStaysAligned(t *testing.T) {
	goRepoRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))
	scriptPath := filepath.Join(t.TempDir(), "scheduler_contract.py")
	script := `import json
import sys
from pathlib import Path

repo_root = Path(sys.argv[1])
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.models import RiskLevel, Task
from bigclaw.scheduler import Scheduler

scheduler = Scheduler()
cases = {
    "high": scheduler.decide(Task(task_id="x", source="jira", title="prod op", description="", risk_level=RiskLevel.HIGH)),
    "browser": scheduler.decide(Task(task_id="y", source="github", title="ui test", description="", required_tools=["browser"])),
    "budgeted_browser": scheduler.decide(Task(task_id="z", source="github", title="budgeted ui test", description="", required_tools=["browser"], budget=15.0)),
    "tiny_budget": scheduler.decide(Task(task_id="b", source="linear", title="tiny budget", description="", budget=5.0)),
}
print(json.dumps({
    name: {"medium": decision.medium, "approved": decision.approved, "reason": decision.reason}
    for name, decision in cases.items()
}))
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write scheduler contract script: %v", err)
	}

	cmd := exec.Command("python3", scriptPath, repoRoot)
	cmd.Dir = goRepoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run scheduler contract script: %v\n%s", err, string(output))
	}

	var decoded map[string]struct {
		Medium   string `json:"medium"`
		Approved bool   `json:"approved"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("decode scheduler contract output: %v\n%s", err, string(output))
	}

	if decoded["high"].Medium != "vm" || decoded["high"].Approved {
		t.Fatalf("unexpected high-risk decision: %+v", decoded["high"])
	}
	if decoded["browser"].Medium != "browser" || !decoded["browser"].Approved {
		t.Fatalf("unexpected browser decision: %+v", decoded["browser"])
	}
	if decoded["budgeted_browser"].Medium != "docker" || !decoded["budgeted_browser"].Approved {
		t.Fatalf("unexpected degraded browser decision: %+v", decoded["budgeted_browser"])
	}
	if decoded["budgeted_browser"].Reason != "budget degraded browser route to docker (budget 15.0 < required 20.0)" {
		t.Fatalf("unexpected degraded browser reason: %+v", decoded["budgeted_browser"])
	}
	if decoded["tiny_budget"].Medium != "none" || decoded["tiny_budget"].Approved {
		t.Fatalf("unexpected tiny-budget decision: %+v", decoded["tiny_budget"])
	}
	if decoded["tiny_budget"].Reason != "paused: budget 5.0 below required docker budget 10.0" {
		t.Fatalf("unexpected tiny-budget reason: %+v", decoded["tiny_budget"])
	}
}
