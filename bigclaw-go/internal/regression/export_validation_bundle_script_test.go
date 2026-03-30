package regression

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportValidationBundleHelpersStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	scriptPath := filepath.Join(repoRoot, "scripts", "e2e", "export_validation_bundle.py")
	code := `
import importlib.util
import json
import pathlib
import tempfile

script_path = pathlib.Path(r"` + filepath.ToSlash(scriptPath) + `")
spec = importlib.util.spec_from_file_location("export_validation_bundle", script_path)
module = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(module)

summary = {
    "run_id": "20260316T140138Z",
    "generated_at": "2026-03-16T14:48:42.581505+00:00",
    "status": "succeeded",
    "bundle_path": "docs/reports/live-validation-runs/20260316T140138Z",
    "summary_path": "docs/reports/live-validation-runs/20260316T140138Z/summary.json",
    "closeout_commands": ["cd bigclaw-go && ./scripts/e2e/run_all.sh"],
    "local": {
        "enabled": True,
        "status": "succeeded",
        "bundle_report_path": "docs/reports/live-validation-runs/20260316T140138Z/sqlite-smoke-report.json",
        "canonical_report_path": "docs/reports/sqlite-smoke-report.json",
    },
    "kubernetes": {
        "enabled": False,
        "status": "skipped",
        "bundle_report_path": "docs/reports/live-validation-runs/20260316T140138Z/kubernetes-live-smoke-report.json",
        "canonical_report_path": "docs/reports/kubernetes-live-smoke-report.json",
    },
    "ray": {
        "enabled": False,
        "status": "skipped",
        "bundle_report_path": "docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json",
        "canonical_report_path": "docs/reports/ray-live-smoke-report.json",
    },
    "broker": {
        "enabled": False,
        "status": "skipped",
        "configuration_state": "not_configured",
        "reason": "not_configured",
        "bundle_summary_path": "docs/reports/live-validation-runs/20260316T140138Z/broker-validation-summary.json",
        "canonical_summary_path": "docs/reports/broker-validation-summary.json",
        "bundle_bootstrap_summary_path": "docs/reports/live-validation-runs/20260316T140138Z/broker-bootstrap-review-summary.json",
        "canonical_bootstrap_summary_path": "docs/reports/broker-bootstrap-review-summary.json",
        "validation_pack_path": "docs/reports/broker-failover-fault-injection-validation-pack.md",
        "backend": None,
        "bootstrap_ready": False,
        "runtime_posture": "contract_only",
        "live_adapter_implemented": False,
        "proof_boundary": "broker bootstrap readiness is a pre-adapter contract surface, not live broker durability proof",
        "config_completeness": {
            "driver": False,
            "urls": False,
            "topic": False,
            "consumer_group": False,
        },
        "validation_errors": ["broker event log config missing driver, urls, topic"],
    },
}
continuation_gate = {
    "path": "docs/reports/validation-bundle-continuation-policy-gate.json",
    "status": "policy-hold",
    "recommendation": "hold",
    "enforcement": {"mode": "hold", "outcome": "hold", "exit_code": 2},
    "summary": {"latest_run_id": "20260316T140138Z", "failing_check_count": 2, "workflow_exit_code": 2},
    "reviewer_path": {"digest_path": "docs/reports/validation-bundle-continuation-digest.md"},
    "next_actions": ["rerun cd bigclaw-go && ./scripts/e2e/run_all.sh to refresh the latest validation bundle"],
}
index_text = module.render_index(summary, [], continuation_gate, [], [])

with tempfile.TemporaryDirectory() as tmpdir:
    tmp_root = pathlib.Path(tmpdir)
    bundle_dir = tmp_root / "docs" / "reports" / "live-validation-runs" / "run-1"
    bundle_dir.mkdir(parents=True, exist_ok=True)
    bootstrap_summary_path = tmp_root / "docs" / "reports" / "broker-bootstrap-review-summary-source.json"
    bootstrap_summary_path.parent.mkdir(parents=True, exist_ok=True)
    bootstrap_summary_path.write_text(json.dumps({
        "ready": False,
        "runtime_posture": "contract_only",
        "live_adapter_implemented": False,
        "proof_boundary": "broker bootstrap readiness is a pre-adapter contract surface, not live broker durability proof",
        "config_completeness": {"driver": False, "urls": False, "topic": False, "consumer_group": False},
        "validation_errors": ["broker event log config missing driver, urls, topic"],
    }))
    broker_section = module.build_broker_section(
        enabled=False,
        backend="",
        root=tmp_root,
        bundle_dir=bundle_dir,
        bootstrap_summary_path=bootstrap_summary_path,
        report_path=None,
    )

with tempfile.TemporaryDirectory() as tmpdir:
    tmp_root = pathlib.Path(tmpdir)
    bundle_dir = tmp_root / "docs" / "reports" / "live-validation-runs" / "run-k8s"
    bundle_dir.mkdir(parents=True, exist_ok=True)
    report_path = tmp_root / "tmp" / "kubernetes-smoke-report.json"
    report_path.parent.mkdir(parents=True, exist_ok=True)
    report_path.write_text(json.dumps({
        "task": {"id": "kubernetes-smoke-failed", "required_executor": "kubernetes"},
        "status": {
            "state": "dead_letter",
            "latest_event": {
                "id": "evt-dead",
                "type": "task.dead_letter",
                "timestamp": "2026-03-23T11:02:00Z",
                "payload": {"message": "lease lost during replay"},
            },
        },
        "events": [
            {
                "id": "evt-routed",
                "type": "scheduler.routed",
                "timestamp": "2026-03-23T11:00:00Z",
                "payload": {"reason": "browser workloads default to kubernetes executor"},
            },
            {
                "id": "evt-dead",
                "type": "task.dead_letter",
                "timestamp": "2026-03-23T11:02:00Z",
                "payload": {"message": "lease lost during replay"},
            },
        ],
    }))
    stdout_path = tmp_root / "tmp" / "kubernetes.stdout.log"
    stderr_path = tmp_root / "tmp" / "kubernetes.stderr.log"
    stdout_path.write_text("starting kubernetes smoke\n")
    stderr_path.write_text("lease lost during replay\n")
    component_section = module.build_component_section(
        name="kubernetes",
        enabled=True,
        root=tmp_root,
        bundle_dir=bundle_dir,
        report_path=report_path,
        stdout_path=stdout_path,
        stderr_path=stderr_path,
    )

summary_with_matrix = {
    "run_id": "20260323T030000Z",
    "generated_at": "2026-03-23T03:10:00+00:00",
    "status": "failed",
    "bundle_path": "docs/reports/live-validation-runs/20260323T030000Z",
    "summary_path": "docs/reports/live-validation-runs/20260323T030000Z/summary.json",
    "closeout_commands": ["cd bigclaw-go && ./scripts/e2e/run_all.sh"],
    "local": {
        "enabled": True,
        "status": "succeeded",
        "bundle_report_path": "docs/reports/live-validation-runs/20260323T030000Z/sqlite-smoke-report.json",
        "canonical_report_path": "docs/reports/sqlite-smoke-report.json",
        "validation_matrix": {
            "lane": "local",
            "executor": "local",
            "enabled": True,
            "status": "succeeded",
            "bundle_report_path": "docs/reports/live-validation-runs/20260323T030000Z/sqlite-smoke-report.json",
        },
        "failure_root_cause": {
            "status": "not_triggered",
            "event_type": "task.completed",
            "message": "",
            "location": "docs/reports/live-validation-runs/20260323T030000Z/sqlite-smoke-report.json",
        },
    },
    "kubernetes": {
        "enabled": True,
        "status": "dead_letter",
        "bundle_report_path": "docs/reports/live-validation-runs/20260323T030000Z/kubernetes-live-smoke-report.json",
        "canonical_report_path": "docs/reports/kubernetes-live-smoke-report.json",
        "validation_matrix": {
            "lane": "k8s",
            "executor": "kubernetes",
            "enabled": True,
            "status": "dead_letter",
            "bundle_report_path": "docs/reports/live-validation-runs/20260323T030000Z/kubernetes-live-smoke-report.json",
            "root_cause_event_type": "task.dead_letter",
            "root_cause_location": "docs/reports/live-validation-runs/20260323T030000Z/kubernetes.stderr.log",
            "root_cause_message": "lease lost during replay",
        },
        "failure_root_cause": {
            "status": "captured",
            "event_type": "task.dead_letter",
            "message": "lease lost during replay",
            "location": "docs/reports/live-validation-runs/20260323T030000Z/kubernetes.stderr.log",
        },
    },
    "ray": {
        "enabled": True,
        "status": "succeeded",
        "bundle_report_path": "docs/reports/live-validation-runs/20260323T030000Z/ray-live-smoke-report.json",
        "canonical_report_path": "docs/reports/ray-live-smoke-report.json",
        "validation_matrix": {
            "lane": "ray",
            "executor": "ray",
            "enabled": True,
            "status": "succeeded",
            "bundle_report_path": "docs/reports/live-validation-runs/20260323T030000Z/ray-live-smoke-report.json",
        },
        "failure_root_cause": {
            "status": "not_triggered",
            "event_type": "task.completed",
            "message": "",
            "location": "docs/reports/live-validation-runs/20260323T030000Z/ray-live-smoke-report.json",
        },
    },
    "validation_matrix": [
        {
            "lane": "local",
            "executor": "local",
            "enabled": True,
            "status": "succeeded",
            "bundle_report_path": "docs/reports/live-validation-runs/20260323T030000Z/sqlite-smoke-report.json",
        },
        {
            "lane": "k8s",
            "executor": "kubernetes",
            "enabled": True,
            "status": "dead_letter",
            "bundle_report_path": "docs/reports/live-validation-runs/20260323T030000Z/kubernetes-live-smoke-report.json",
            "root_cause_event_type": "task.dead_letter",
            "root_cause_location": "docs/reports/live-validation-runs/20260323T030000Z/kubernetes.stderr.log",
            "root_cause_message": "lease lost during replay",
        },
        {
            "lane": "ray",
            "executor": "ray",
            "enabled": True,
            "status": "succeeded",
            "bundle_report_path": "docs/reports/live-validation-runs/20260323T030000Z/ray-live-smoke-report.json",
        },
    ],
}
matrix_index_text = module.render_index(summary_with_matrix, [], None, [], [])

print(json.dumps({
    "index_text": index_text,
    "broker_section": broker_section,
    "component_section": component_section,
    "matrix_index_text": matrix_index_text,
}))
`
	cmd := exec.Command("python3", "-c", code)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run export validation bundle helpers: %v\n%s", err, output)
	}

	var payload struct {
		IndexText       string `json:"index_text"`
		MatrixIndexText string `json:"matrix_index_text"`
		BrokerSection   struct {
			Status                 string          `json:"status"`
			ConfigurationState     string          `json:"configuration_state"`
			Reason                 string          `json:"reason"`
			BundleSummaryPath      string          `json:"bundle_summary_path"`
			RuntimePosture         string          `json:"runtime_posture"`
			LiveAdapterImplemented bool            `json:"live_adapter_implemented"`
			ConfigCompleteness     map[string]bool `json:"config_completeness"`
		} `json:"broker_section"`
		ComponentSection struct {
			Status           string `json:"status"`
			RoutingReason    string `json:"routing_reason"`
			ValidationMatrix struct {
				Lane     string `json:"lane"`
				Executor string `json:"executor"`
			} `json:"validation_matrix"`
			FailureRootCause struct {
				Status    string `json:"status"`
				EventType string `json:"event_type"`
				Message   string `json:"message"`
				Location  string `json:"location"`
			} `json:"failure_root_cause"`
		} `json:"component_section"`
	}
	if err := json.Unmarshal(output, &payload); err != nil {
		t.Fatalf("decode export validation bundle payload: %v\n%s", err, output)
	}

	for _, needle := range []string{
		"- Workflow mode: `hold`",
		"- Workflow outcome: `hold`",
		"- Workflow exit code on current evidence: `2`",
		"### broker",
		"- Configuration state: `not_configured`",
		"- Runtime posture: `contract_only`",
		"- Live adapter implemented: `False`",
		"- Validation error: `broker event log config missing driver, urls, topic`",
		"- Reason: `not_configured`",
	} {
		if !strings.Contains(payload.IndexText, needle) {
			t.Fatalf("index text missing %q\n%s", needle, payload.IndexText)
		}
	}

	if payload.BrokerSection.Status != "skipped" || payload.BrokerSection.ConfigurationState != "not_configured" || payload.BrokerSection.Reason != "not_configured" {
		t.Fatalf("unexpected broker section identity: %+v", payload.BrokerSection)
	}
	if payload.BrokerSection.BundleSummaryPath != "docs/reports/live-validation-runs/run-1/broker-validation-summary.json" {
		t.Fatalf("unexpected broker summary path: %+v", payload.BrokerSection)
	}
	if payload.BrokerSection.RuntimePosture != "contract_only" || payload.BrokerSection.LiveAdapterImplemented {
		t.Fatalf("unexpected broker runtime posture: %+v", payload.BrokerSection)
	}
	if payload.BrokerSection.ConfigCompleteness["driver"] {
		t.Fatalf("expected broker driver config to be incomplete: %+v", payload.BrokerSection.ConfigCompleteness)
	}

	if payload.ComponentSection.Status != "dead_letter" {
		t.Fatalf("unexpected component status: %+v", payload.ComponentSection)
	}
	if payload.ComponentSection.ValidationMatrix.Lane != "k8s" || payload.ComponentSection.ValidationMatrix.Executor != "kubernetes" {
		t.Fatalf("unexpected component validation matrix: %+v", payload.ComponentSection.ValidationMatrix)
	}
	if payload.ComponentSection.RoutingReason != "browser workloads default to kubernetes executor" {
		t.Fatalf("unexpected routing reason: %+v", payload.ComponentSection)
	}
	if payload.ComponentSection.FailureRootCause.Status != "captured" ||
		payload.ComponentSection.FailureRootCause.EventType != "task.dead_letter" ||
		payload.ComponentSection.FailureRootCause.Message != "lease lost during replay" ||
		payload.ComponentSection.FailureRootCause.Location != "docs/reports/live-validation-runs/run-k8s/kubernetes.stderr.log" {
		t.Fatalf("unexpected failure root cause: %+v", payload.ComponentSection.FailureRootCause)
	}

	for _, needle := range []string{
		"## Validation matrix",
		"- Lane `k8s` executor=`kubernetes` status=`dead_letter` enabled=`True` report=`docs/reports/live-validation-runs/20260323T030000Z/kubernetes-live-smoke-report.json`",
		"- Lane `k8s` root cause: event=`task.dead_letter` location=`docs/reports/live-validation-runs/20260323T030000Z/kubernetes.stderr.log` message=`lease lost during replay`",
		"- Failure root cause: status=`captured` event=`task.dead_letter` location=`docs/reports/live-validation-runs/20260323T030000Z/kubernetes.stderr.log`",
	} {
		if !strings.Contains(payload.MatrixIndexText, needle) {
			t.Fatalf("matrix index text missing %q\n%s", needle, payload.MatrixIndexText)
		}
	}
}
