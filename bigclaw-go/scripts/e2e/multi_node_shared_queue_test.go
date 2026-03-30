package main

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func testScriptDirForMultiNode(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		t.Fatal("failed to resolve caller file")
	}
	return filepath.Dir(file)
}

func runPythonJSONForMultiNode(t *testing.T, code string) map[string]any {
	t.Helper()
	cmd := exec.Command("python3", "-c", code)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("python3 failed: %v\n%s", err, string(out))
	}
	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("failed to parse python JSON output: %v\n%s", err, string(out))
	}
	return payload
}

func TestBuildLiveTakeoverReportSummarizesSchemaParity(t *testing.T) {
	scriptPath := filepath.Join(testScriptDirForMultiNode(t), "multi_node_shared_queue.py")
	code := `
import importlib.util
import json
import pathlib

path = pathlib.Path(r"""` + scriptPath + `""")
spec = importlib.util.spec_from_file_location("multi_node_shared_queue", path)
if spec is None or spec.loader is None:
    raise RuntimeError(f"Unable to load module from {path}")
module = importlib.util.module_from_spec(spec)
spec.loader.exec_module(module)

scenarios = [
    {
        "id": "scenario-a",
        "all_assertions_passed": True,
        "duplicate_delivery_count": 1,
        "stale_write_rejections": 1,
    },
    {
        "id": "scenario-b",
        "all_assertions_passed": False,
        "duplicate_delivery_count": 2,
        "stale_write_rejections": 0,
    },
]
report = module.build_live_takeover_report(
    scenarios,
    "docs/reports/multi-node-shared-queue-report.json",
)
print(json.dumps({
    "ticket": report["ticket"],
    "status": report["status"],
    "scenario_count": report["summary"]["scenario_count"],
    "passing_scenarios": report["summary"]["passing_scenarios"],
    "failing_scenarios": report["summary"]["failing_scenarios"],
    "duplicate_delivery_count": report["summary"]["duplicate_delivery_count"],
    "stale_write_rejections": report["summary"]["stale_write_rejections"],
    "has_required_report_sections": "required_report_sections" in report,
    "has_remaining_gaps": "remaining_gaps" in report,
    "shared_queue_evidence_second": report["current_primitives"]["shared_queue_evidence"][1],
}))
`
	payload := runPythonJSONForMultiNode(t, code)
	if got, ok := payload["ticket"].(string); !ok || got != "OPE-260" {
		t.Fatalf("unexpected ticket: %v", payload["ticket"])
	}
	if got, ok := payload["status"].(string); !ok || got != "live-multi-node-proof" {
		t.Fatalf("unexpected status: %v", payload["status"])
	}
	if got, ok := payload["scenario_count"].(float64); !ok || got != 2 {
		t.Fatalf("unexpected scenario_count: %v", payload["scenario_count"])
	}
	if got, ok := payload["passing_scenarios"].(float64); !ok || got != 1 {
		t.Fatalf("unexpected passing_scenarios: %v", payload["passing_scenarios"])
	}
	if got, ok := payload["failing_scenarios"].(float64); !ok || got != 1 {
		t.Fatalf("unexpected failing_scenarios: %v", payload["failing_scenarios"])
	}
	if got, ok := payload["duplicate_delivery_count"].(float64); !ok || got != 3 {
		t.Fatalf("unexpected duplicate_delivery_count: %v", payload["duplicate_delivery_count"])
	}
	if got, ok := payload["stale_write_rejections"].(float64); !ok || got != 1 {
		t.Fatalf("unexpected stale_write_rejections: %v", payload["stale_write_rejections"])
	}
	if got, ok := payload["has_required_report_sections"].(bool); !ok || !got {
		t.Fatalf("expected required_report_sections to exist: %v", payload["has_required_report_sections"])
	}
	if got, ok := payload["has_remaining_gaps"].(bool); !ok || !got {
		t.Fatalf("expected remaining_gaps to exist: %v", payload["has_remaining_gaps"])
	}
	if got, ok := payload["shared_queue_evidence_second"].(string); !ok || got != "docs/reports/multi-node-shared-queue-report.json" {
		t.Fatalf("unexpected shared_queue_evidence_second: %v", payload["shared_queue_evidence_second"])
	}
}
