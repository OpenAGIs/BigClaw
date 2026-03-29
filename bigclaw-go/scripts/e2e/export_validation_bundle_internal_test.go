package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderIndexSurfacesContinuationWorkflowModeAndOutcome(t *testing.T) {
	summary := map[string]any{
		"run_id":       "20260316T140138Z",
		"generated_at": "2026-03-16T14:48:42.581505+00:00",
		"status":       "succeeded",
		"bundle_path":  "docs/reports/live-validation-runs/20260316T140138Z",
		"summary_path": "docs/reports/live-validation-runs/20260316T140138Z/summary.json",
		"closeout_commands": []any{
			"cd bigclaw-go && ./scripts/e2e/run_all.sh",
		},
		"local": map[string]any{
			"enabled":               true,
			"status":                "succeeded",
			"bundle_report_path":    "docs/reports/live-validation-runs/20260316T140138Z/sqlite-smoke-report.json",
			"canonical_report_path": "docs/reports/sqlite-smoke-report.json",
		},
		"kubernetes": map[string]any{
			"enabled":               false,
			"status":                "skipped",
			"bundle_report_path":    "docs/reports/live-validation-runs/20260316T140138Z/kubernetes-live-smoke-report.json",
			"canonical_report_path": "docs/reports/kubernetes-live-smoke-report.json",
		},
		"ray": map[string]any{
			"enabled":               false,
			"status":                "skipped",
			"bundle_report_path":    "docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json",
			"canonical_report_path": "docs/reports/ray-live-smoke-report.json",
		},
		"broker": map[string]any{
			"enabled":                           false,
			"status":                            "skipped",
			"configuration_state":               "not_configured",
			"reason":                            "not_configured",
			"bundle_summary_path":               "docs/reports/live-validation-runs/20260316T140138Z/broker-validation-summary.json",
			"canonical_summary_path":            "docs/reports/broker-validation-summary.json",
			"bundle_bootstrap_summary_path":     "docs/reports/live-validation-runs/20260316T140138Z/broker-bootstrap-review-summary.json",
			"canonical_bootstrap_summary_path":  "docs/reports/broker-bootstrap-review-summary.json",
			"validation_pack_path":              "docs/reports/broker-failover-fault-injection-validation-pack.md",
			"backend":                           nil,
			"bootstrap_ready":                   false,
			"runtime_posture":                   "contract_only",
			"live_adapter_implemented":          false,
			"proof_boundary":                    "broker bootstrap readiness is a pre-adapter contract surface, not live broker durability proof",
			"config_completeness":               map[string]any{"driver": false, "urls": false, "topic": false, "consumer_group": false},
			"validation_errors":                 []any{"broker event log config missing driver, urls, topic"},
		},
	}
	continuationGate := map[string]any{
		"path":           "docs/reports/validation-bundle-continuation-policy-gate.json",
		"status":         "policy-hold",
		"recommendation": "hold",
		"enforcement":    map[string]any{"mode": "hold", "outcome": "hold", "exit_code": 2},
		"summary":        map[string]any{"latest_run_id": "20260316T140138Z", "failing_check_count": 2, "workflow_exit_code": 2},
		"reviewer_path":  map[string]any{"digest_path": "docs/reports/validation-bundle-continuation-digest.md"},
		"next_actions":   []any{"rerun `cd bigclaw-go && ./scripts/e2e/run_all.sh` to refresh the latest validation bundle"},
	}

	indexText := renderIndex(summary, nil, continuationGate, nil, nil)

	for _, needle := range []string{
		"- Workflow mode: `hold`",
		"- Workflow outcome: `hold`",
		"- Workflow exit code on current evidence: `2`",
		"### broker",
		"- Configuration state: `not_configured`",
		"- Runtime posture: `contract_only`",
		"- Live adapter implemented: `false`",
		"- Validation error: `broker event log config missing driver, urls, topic`",
		"- Reason: `not_configured`",
	} {
		if !strings.Contains(indexText, needle) {
			t.Fatalf("index missing %q\n%s", needle, indexText)
		}
	}
}

func TestBuildBrokerSectionWritesNotConfiguredSummaryWhenDisabled(t *testing.T) {
	root := t.TempDir()
	bundleDir := filepath.Join(root, "docs", "reports", "live-validation-runs", "run-1")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	bootstrapSummaryPath := filepath.Join(root, "docs", "reports", "broker-bootstrap-review-summary-source.json")
	if err := os.MkdirAll(filepath.Dir(bootstrapSummaryPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(bootstrapSummaryPath, []byte(`{"ready": false, "runtime_posture": "contract_only", "live_adapter_implemented": false, "proof_boundary": "broker bootstrap readiness is a pre-adapter contract surface, not live broker durability proof", "config_completeness": {"driver": false, "urls": false, "topic": false, "consumer_group": false}, "validation_errors": ["broker event log config missing driver, urls, topic"]}`), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	section, err := buildBrokerSection(brokerSectionInput{
		Enabled:              false,
		Backend:              "",
		Root:                 root,
		BundleDir:            bundleDir,
		BootstrapSummaryPath: bootstrapSummaryPath,
	})
	if err != nil {
		t.Fatalf("buildBrokerSection: %v", err)
	}

	if section["status"] != "skipped" || section["configuration_state"] != "not_configured" || section["reason"] != "not_configured" {
		t.Fatalf("unexpected section: %+v", section)
	}
	if section["bundle_summary_path"] != "docs/reports/live-validation-runs/run-1/broker-validation-summary.json" {
		t.Fatalf("unexpected bundle_summary_path: %+v", section)
	}
	if section["runtime_posture"] != "contract_only" || asExportBool(section["live_adapter_implemented"]) || asExportBool(asExportMap(section["config_completeness"])["driver"]) {
		t.Fatalf("unexpected runtime posture section: %+v", section)
	}
	for _, path := range []string{
		filepath.Join(bundleDir, "broker-validation-summary.json"),
		filepath.Join(root, "docs", "reports", "broker-validation-summary.json"),
		filepath.Join(bundleDir, "broker-bootstrap-review-summary.json"),
		filepath.Join(root, "docs", "reports", "broker-bootstrap-review-summary.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s: %v", path, err)
		}
	}
}

func TestBuildComponentSectionEmitsK8sMatrixAndFailureRootCause(t *testing.T) {
	root := t.TempDir()
	bundleDir := filepath.Join(root, "docs", "reports", "live-validation-runs", "run-k8s")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	reportPath := filepath.Join(root, "tmp", "kubernetes-smoke-report.json")
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	reportBody := `{
  "task": {
    "id": "kubernetes-smoke-failed",
    "required_executor": "kubernetes"
  },
  "status": {
    "state": "dead_letter",
    "latest_event": {
      "id": "evt-dead",
      "type": "task.dead_letter",
      "timestamp": "2026-03-23T11:02:00Z",
      "payload": {
        "message": "lease lost during replay"
      }
    }
  },
  "events": [
    {
      "id": "evt-routed",
      "type": "scheduler.routed",
      "timestamp": "2026-03-23T11:00:00Z",
      "payload": {
        "reason": "browser workloads default to kubernetes executor"
      }
    },
    {
      "id": "evt-dead",
      "type": "task.dead_letter",
      "timestamp": "2026-03-23T11:02:00Z",
      "payload": {
        "message": "lease lost during replay"
      }
    }
  ]
}`
	if err := os.WriteFile(reportPath, []byte(reportBody), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	stdoutPath := filepath.Join(root, "tmp", "kubernetes.stdout.log")
	stderrPath := filepath.Join(root, "tmp", "kubernetes.stderr.log")
	if err := os.WriteFile(stdoutPath, []byte("starting kubernetes smoke\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(stderrPath, []byte("lease lost during replay\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	section, err := buildComponentSection(componentSectionInput{
		Name:       "kubernetes",
		Enabled:    true,
		Root:       root,
		BundleDir:  bundleDir,
		ReportPath: reportPath,
		StdoutPath: stdoutPath,
		StderrPath: stderrPath,
	})
	if err != nil {
		t.Fatalf("buildComponentSection: %v", err)
	}

	if section["status"] != "dead_letter" {
		t.Fatalf("status = %v", section["status"])
	}
	matrix := asExportMap(section["validation_matrix"])
	rootCause := asExportMap(section["failure_root_cause"])
	if matrix["lane"] != "k8s" || matrix["executor"] != "kubernetes" {
		t.Fatalf("unexpected validation matrix: %+v", matrix)
	}
	if section["routing_reason"] != "browser workloads default to kubernetes executor" {
		t.Fatalf("routing_reason = %v", section["routing_reason"])
	}
	if rootCause["status"] != "captured" || rootCause["event_type"] != "task.dead_letter" || rootCause["message"] != "lease lost during replay" {
		t.Fatalf("unexpected root cause: %+v", rootCause)
	}
	if rootCause["location"] != "docs/reports/live-validation-runs/run-k8s/kubernetes.stderr.log" {
		t.Fatalf("unexpected root cause location: %+v", rootCause)
	}
}

func TestRenderIndexSurfacesValidationMatrixAndFailureRootCause(t *testing.T) {
	summary := map[string]any{
		"run_id":       "20260323T030000Z",
		"generated_at": "2026-03-23T03:10:00+00:00",
		"status":       "failed",
		"bundle_path":  "docs/reports/live-validation-runs/20260323T030000Z",
		"summary_path": "docs/reports/live-validation-runs/20260323T030000Z/summary.json",
		"closeout_commands": []any{
			"cd bigclaw-go && ./scripts/e2e/run_all.sh",
		},
		"local": map[string]any{
			"enabled":               true,
			"status":                "succeeded",
			"bundle_report_path":    "docs/reports/live-validation-runs/20260323T030000Z/sqlite-smoke-report.json",
			"canonical_report_path": "docs/reports/sqlite-smoke-report.json",
			"validation_matrix": map[string]any{
				"lane":               "local",
				"executor":           "local",
				"enabled":            true,
				"status":             "succeeded",
				"bundle_report_path": "docs/reports/live-validation-runs/20260323T030000Z/sqlite-smoke-report.json",
			},
			"failure_root_cause": map[string]any{
				"status":    "not_triggered",
				"event_type": "task.completed",
				"message":   "",
				"location":  "docs/reports/live-validation-runs/20260323T030000Z/sqlite-smoke-report.json",
			},
		},
		"kubernetes": map[string]any{
			"enabled":               true,
			"status":                "dead_letter",
			"bundle_report_path":    "docs/reports/live-validation-runs/20260323T030000Z/kubernetes-live-smoke-report.json",
			"canonical_report_path": "docs/reports/kubernetes-live-smoke-report.json",
			"validation_matrix": map[string]any{
				"lane":                  "k8s",
				"executor":              "kubernetes",
				"enabled":               true,
				"status":                "dead_letter",
				"bundle_report_path":    "docs/reports/live-validation-runs/20260323T030000Z/kubernetes-live-smoke-report.json",
				"root_cause_event_type": "task.dead_letter",
				"root_cause_location":   "docs/reports/live-validation-runs/20260323T030000Z/kubernetes.stderr.log",
				"root_cause_message":    "lease lost during replay",
			},
			"failure_root_cause": map[string]any{
				"status":     "captured",
				"event_type": "task.dead_letter",
				"message":    "lease lost during replay",
				"location":   "docs/reports/live-validation-runs/20260323T030000Z/kubernetes.stderr.log",
			},
		},
		"ray": map[string]any{
			"enabled":               true,
			"status":                "succeeded",
			"bundle_report_path":    "docs/reports/live-validation-runs/20260323T030000Z/ray-live-smoke-report.json",
			"canonical_report_path": "docs/reports/ray-live-smoke-report.json",
			"validation_matrix": map[string]any{
				"lane":               "ray",
				"executor":           "ray",
				"enabled":            true,
				"status":             "succeeded",
				"bundle_report_path": "docs/reports/live-validation-runs/20260323T030000Z/ray-live-smoke-report.json",
			},
			"failure_root_cause": map[string]any{
				"status":     "not_triggered",
				"event_type": "task.completed",
				"message":    "",
				"location":   "docs/reports/live-validation-runs/20260323T030000Z/ray-live-smoke-report.json",
			},
		},
		"validation_matrix": []any{
			map[string]any{
				"lane":               "local",
				"executor":           "local",
				"enabled":            true,
				"status":             "succeeded",
				"bundle_report_path": "docs/reports/live-validation-runs/20260323T030000Z/sqlite-smoke-report.json",
			},
			map[string]any{
				"lane":                  "k8s",
				"executor":              "kubernetes",
				"enabled":               true,
				"status":                "dead_letter",
				"bundle_report_path":    "docs/reports/live-validation-runs/20260323T030000Z/kubernetes-live-smoke-report.json",
				"root_cause_event_type": "task.dead_letter",
				"root_cause_location":   "docs/reports/live-validation-runs/20260323T030000Z/kubernetes.stderr.log",
				"root_cause_message":    "lease lost during replay",
			},
			map[string]any{
				"lane":               "ray",
				"executor":           "ray",
				"enabled":            true,
				"status":             "succeeded",
				"bundle_report_path": "docs/reports/live-validation-runs/20260323T030000Z/ray-live-smoke-report.json",
			},
		},
	}

	indexText := renderIndex(summary, nil, nil, nil, nil)

	for _, needle := range []string{
		"## Validation matrix",
		"- Lane `k8s` executor=`kubernetes` status=`dead_letter` enabled=`true` report=`docs/reports/live-validation-runs/20260323T030000Z/kubernetes-live-smoke-report.json`",
		"- Lane `k8s` root cause: event=`task.dead_letter` location=`docs/reports/live-validation-runs/20260323T030000Z/kubernetes.stderr.log` message=`lease lost during replay`",
		"- Failure root cause: status=`captured` event=`task.dead_letter` location=`docs/reports/live-validation-runs/20260323T030000Z/kubernetes.stderr.log`",
	} {
		if !strings.Contains(indexText, needle) {
			t.Fatalf("index missing %q\n%s", needle, indexText)
		}
	}
}
