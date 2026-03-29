package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLane8PythonRiskContractStaysAligned(t *testing.T) {
	goRepoRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))
	scriptPath := filepath.Join(t.TempDir(), "risk_contract.py")
	script := `import json
import sys
import tempfile
from pathlib import Path

repo_root = Path(sys.argv[1])
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.models import Priority, Task
from bigclaw.observability import ObservabilityLedger
from bigclaw.risk import RiskScorer
from bigclaw.scheduler import Scheduler

low = RiskScorer().score_task(
    Task(task_id="BIG-902-low", source="linear", title="doc cleanup", description="")
)
mid = RiskScorer().score_task(
    Task(
        task_id="BIG-902-mid",
        source="linear",
        title="release verification",
        description="prod browser change",
        labels=["prod"],
        priority=Priority.P0,
        required_tools=["browser"],
    )
)

with tempfile.TemporaryDirectory() as td:
    ledger = ObservabilityLedger(str(Path(td) / "ledger.json"))
    task = Task(
        task_id="BIG-902-high",
        source="linear",
        title="security deploy",
        description="prod deploy",
        labels=["security", "prod"],
        priority=Priority.P0,
        required_tools=["deploy"],
    )
    record = Scheduler().execute(task, run_id="run-risk", ledger=ledger)
    entry = ledger.load()[0]
    high = {
        "score_total": record.risk_score.total,
        "score_level": record.risk_score.level,
        "medium": record.decision.medium,
        "approved": record.decision.approved,
        "has_risk_trace": any(trace["span"] == "risk.score" for trace in entry["traces"]),
        "has_risk_audit": any(audit["action"] == "risk.score" for audit in entry["audits"]),
    }

print(json.dumps({
    "low": {
        "total": low.total,
        "level": low.level,
        "requires_approval": low.requires_approval,
    },
    "mid": {
        "total": mid.total,
        "level": mid.level,
        "requires_approval": mid.requires_approval,
    },
    "high": high,
}))
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write risk contract script: %v", err)
	}

	cmd := exec.Command("python3", scriptPath, repoRoot)
	cmd.Dir = goRepoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run risk contract script: %v\n%s", err, string(output))
	}

	var decoded struct {
		Low struct {
			Total            int    `json:"total"`
			Level            string `json:"level"`
			RequiresApproval bool   `json:"requires_approval"`
		} `json:"low"`
		Mid struct {
			Total            int    `json:"total"`
			Level            string `json:"level"`
			RequiresApproval bool   `json:"requires_approval"`
		} `json:"mid"`
		High struct {
			ScoreTotal   int    `json:"score_total"`
			ScoreLevel   string `json:"score_level"`
			Medium       string `json:"medium"`
			Approved     bool   `json:"approved"`
			HasRiskTrace bool   `json:"has_risk_trace"`
			HasRiskAudit bool   `json:"has_risk_audit"`
		} `json:"high"`
	}
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("decode risk contract output: %v\n%s", err, string(output))
	}

	if decoded.Low.Total != 0 || decoded.Low.Level != "low" || decoded.Low.RequiresApproval {
		t.Fatalf("unexpected low-risk payload: %+v", decoded.Low)
	}
	if decoded.Mid.Total != 40 || decoded.Mid.Level != "medium" || decoded.Mid.RequiresApproval {
		t.Fatalf("unexpected medium-risk payload: %+v", decoded.Mid)
	}
	if decoded.High.ScoreTotal != 70 || decoded.High.ScoreLevel != "high" || decoded.High.Medium != "vm" || decoded.High.Approved || !decoded.High.HasRiskTrace || !decoded.High.HasRiskAudit {
		t.Fatalf("unexpected high-risk scheduler payload: %+v", decoded.High)
	}
}
