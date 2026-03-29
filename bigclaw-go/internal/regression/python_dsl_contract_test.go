package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLane8PythonDSLContractStaysAligned(t *testing.T) {
	goRepoRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))
	scriptPath := filepath.Join(t.TempDir(), "dsl_contract.py")
	script := `import json
import sys
import tempfile
from pathlib import Path

repo_root = Path(sys.argv[1])
sys.path.insert(0, str(repo_root / "src"))

from bigclaw.dsl import WorkflowDefinition
from bigclaw.models import RiskLevel, Task
from bigclaw.observability import ObservabilityLedger
from bigclaw.workflow import WorkflowEngine

definition = WorkflowDefinition.from_json(
    "{"
    "\"name\": \"release-closeout\", "
    "\"steps\": [{\"name\": \"execute\", \"kind\": \"scheduler\"}], "
    "\"report_path_template\": \"reports/{task_id}/{run_id}.md\", "
    "\"journal_path_template\": \"journals/{workflow}/{run_id}.json\", "
    "\"validation_evidence\": [\"pytest\"], "
    "\"approvals\": [\"ops-review\"]"
    "}"
)
task = Task(task_id="BIG-401", source="linear", title="DSL", description="")
parsed = {
    "step_name": definition.steps[0].name,
    "report_path": definition.render_report_path(task, "run-1"),
    "journal_path": definition.render_journal_path(task, "run-1"),
}

with tempfile.TemporaryDirectory() as td:
    td = Path(td)
    definition = WorkflowDefinition.from_dict(
        {
            "name": "acceptance-closeout",
            "steps": [{"name": "execute", "kind": "scheduler"}],
            "report_path_template": str(td / "reports" / "{task_id}" / "{run_id}.md"),
            "journal_path_template": str(td / "journals" / "{workflow}" / "{run_id}.json"),
            "validation_evidence": ["pytest", "report-shared"],
        }
    )
    task = Task(
        task_id="BIG-401-flow",
        source="linear",
        title="Run workflow definition",
        description="dsl execution",
        acceptance_criteria=["report-shared"],
        validation_plan=["pytest"],
    )
    result = WorkflowEngine().run_definition(
        task,
        definition=definition,
        run_id="run-dsl-1",
        ledger=ObservabilityLedger(str(td / "ledger.json")),
    )
    executed = {
        "acceptance_status": result.acceptance.status,
        "report_exists": Path(definition.render_report_path(task, "run-dsl-1")).exists(),
        "journal_exists": Path(definition.render_journal_path(task, "run-dsl-1")).exists(),
    }

with tempfile.TemporaryDirectory() as td:
    definition = WorkflowDefinition.from_dict(
        {
            "name": "broken-flow",
            "steps": [{"name": "hack", "kind": "unknown-kind"}],
        }
    )
    task = Task(task_id="BIG-401-invalid", source="local", title="invalid", description="")
    invalid_error = ""
    try:
        WorkflowEngine().run_definition(
            task,
            definition=definition,
            run_id="run-dsl-invalid",
            ledger=ObservabilityLedger(str(Path(td) / "ledger.json")),
        )
    except ValueError as exc:
        invalid_error = str(exc)

with tempfile.TemporaryDirectory() as td:
    definition = WorkflowDefinition.from_dict(
        {
            "name": "prod-approval",
            "steps": [{"name": "review", "kind": "approval"}],
            "validation_evidence": ["rollback-plan", "integration-test"],
            "approvals": ["release-manager"],
        }
    )
    task = Task(
        task_id="BIG-403-dsl",
        source="linear",
        title="Prod rollout",
        description="needs manual closure",
        risk_level=RiskLevel.HIGH,
        acceptance_criteria=["rollback-plan"],
        validation_plan=["integration-test"],
    )
    result = WorkflowEngine().run_definition(
        task,
        definition=definition,
        run_id="run-dsl-2",
        ledger=ObservabilityLedger(str(Path(td) / "ledger.json")),
    )
    approval = {
        "run_status": result.execution.run.status,
        "acceptance_status": result.acceptance.status,
        "approvals": result.acceptance.approvals,
    }

print(json.dumps({
    "parsed": parsed,
    "executed": executed,
    "invalid_error": invalid_error,
    "approval": approval,
}))
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatalf("write dsl contract script: %v", err)
	}

	cmd := exec.Command("python3", scriptPath, repoRoot)
	cmd.Dir = goRepoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run dsl contract script: %v\n%s", err, string(output))
	}

	var decoded struct {
		Parsed struct {
			StepName    string `json:"step_name"`
			ReportPath  string `json:"report_path"`
			JournalPath string `json:"journal_path"`
		} `json:"parsed"`
		Executed struct {
			AcceptanceStatus string `json:"acceptance_status"`
			ReportExists     bool   `json:"report_exists"`
			JournalExists    bool   `json:"journal_exists"`
		} `json:"executed"`
		InvalidError string `json:"invalid_error"`
		Approval     struct {
			RunStatus        string   `json:"run_status"`
			AcceptanceStatus string   `json:"acceptance_status"`
			Approvals        []string `json:"approvals"`
		} `json:"approval"`
	}
	if err := json.Unmarshal(output, &decoded); err != nil {
		t.Fatalf("decode dsl contract output: %v\n%s", err, string(output))
	}

	if decoded.Parsed.StepName != "execute" || decoded.Parsed.ReportPath != "reports/BIG-401/run-1.md" || decoded.Parsed.JournalPath != "journals/release-closeout/run-1.json" {
		t.Fatalf("unexpected parsed workflow definition payload: %+v", decoded.Parsed)
	}
	if decoded.Executed.AcceptanceStatus != "accepted" || !decoded.Executed.ReportExists || !decoded.Executed.JournalExists {
		t.Fatalf("unexpected run_definition payload: %+v", decoded.Executed)
	}
	if decoded.InvalidError != "invalid workflow step kind(s): unknown-kind" {
		t.Fatalf("unexpected invalid workflow error: %q", decoded.InvalidError)
	}
	if decoded.Approval.RunStatus != "needs-approval" || decoded.Approval.AcceptanceStatus != "accepted" || len(decoded.Approval.Approvals) != 1 || decoded.Approval.Approvals[0] != "release-manager" {
		t.Fatalf("unexpected approval workflow payload: %+v", decoded.Approval)
	}
}
