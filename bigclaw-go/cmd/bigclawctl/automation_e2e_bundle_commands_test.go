package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAutomationContinuationPolicyGateReturnsPolicyGoWhenInputsPass(t *testing.T) {
	root := t.TempDir()
	scorecardPath := filepath.Join(root, "docs/reports/validation-bundle-continuation-scorecard.json")
	if err := os.MkdirAll(filepath.Dir(scorecardPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scorecardPath, []byte(`{
  "summary": {
    "latest_run_id": "20260316T140138Z",
    "latest_bundle_age_hours": 1.0,
    "recent_bundle_count": 3,
    "latest_all_executor_tracks_succeeded": true,
    "recent_bundle_chain_has_no_failures": true,
    "all_executor_tracks_have_repeated_recent_coverage": true
  },
  "shared_queue_companion": {
    "available": true,
    "cross_node_completions": 99
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	report, exitCode, err := automationContinuationPolicyGate(automationContinuationPolicyGateOptions{
		GoRoot:                      root,
		ScorecardPath:               "docs/reports/validation-bundle-continuation-scorecard.json",
		OutputPath:                  "docs/reports/validation-bundle-continuation-policy-gate.json",
		MaxLatestAgeHours:           72,
		MinRecentBundles:            2,
		RequireRepeatedLaneCoverage: true,
	})
	if err != nil {
		t.Fatalf("build gate report: %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("expected zero exit code, got %d", exitCode)
	}
	if report["status"] != "policy-go" || report["recommendation"] != "go" {
		t.Fatalf("unexpected report: %+v", report)
	}
	enforcement, _ := report["enforcement"].(map[string]any)
	if enforcement["mode"] != "hold" || enforcement["outcome"] != "pass" {
		t.Fatalf("unexpected enforcement: %+v", enforcement)
	}
	nextActions, _ := report["next_actions"].([]any)
	if len(nextActions) == 0 || !strings.Contains(nextActions[0].(string), "BIGCLAW_E2E_CONTINUATION_GATE_MODE=review") {
		t.Fatalf("unexpected next actions: %+v", nextActions)
	}
}

func TestAutomationContinuationPolicyGateReturnsPolicyHoldWithFailures(t *testing.T) {
	root := t.TempDir()
	scorecardPath := filepath.Join(root, "docs/reports/validation-bundle-continuation-scorecard.json")
	if err := os.MkdirAll(filepath.Dir(scorecardPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scorecardPath, []byte(`{
  "summary": {
    "latest_run_id": "20260316T140138Z",
    "latest_bundle_age_hours": 96.0,
    "recent_bundle_count": 1,
    "latest_all_executor_tracks_succeeded": true,
    "recent_bundle_chain_has_no_failures": true,
    "all_executor_tracks_have_repeated_recent_coverage": false
  },
  "shared_queue_companion": {
    "available": false,
    "cross_node_completions": 0
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	report, _, err := automationContinuationPolicyGate(automationContinuationPolicyGateOptions{
		GoRoot:                      root,
		ScorecardPath:               "docs/reports/validation-bundle-continuation-scorecard.json",
		OutputPath:                  "docs/reports/validation-bundle-continuation-policy-gate.json",
		MaxLatestAgeHours:           72,
		MinRecentBundles:            2,
		RequireRepeatedLaneCoverage: true,
	})
	if err != nil {
		t.Fatalf("build gate report: %v", err)
	}
	if report["status"] != "policy-hold" {
		t.Fatalf("unexpected status: %+v", report)
	}
	failingChecks, _ := report["failing_checks"].([]any)
	if len(failingChecks) < 4 {
		t.Fatalf("expected multiple failing checks, got %+v", failingChecks)
	}
}

func TestAutomationExportValidationBundleBuildBrokerSectionDisabled(t *testing.T) {
	root := t.TempDir()
	bundleDir := filepath.Join(root, "docs/reports/live-validation-runs/run-1")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	bootstrapSummary := filepath.Join(root, "docs/reports/broker-bootstrap-review-summary-source.json")
	if err := os.MkdirAll(filepath.Dir(bootstrapSummary), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bootstrapSummary, []byte(`{
  "ready": false,
  "runtime_posture": "contract_only",
  "live_adapter_implemented": false,
  "proof_boundary": "broker bootstrap readiness is a pre-adapter contract surface, not live broker durability proof",
  "config_completeness": {"driver": false, "urls": false, "topic": false, "consumer_group": false},
  "validation_errors": ["broker event log config missing driver, urls, topic"]
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	section, err := e2eBuildBrokerSection(false, "", root, bundleDir, bootstrapSummary, "")
	if err != nil {
		t.Fatalf("build broker section: %v", err)
	}
	if section["status"] != "skipped" || section["reason"] != "not_configured" {
		t.Fatalf("unexpected broker section: %+v", section)
	}
	if section["runtime_posture"] != "contract_only" {
		t.Fatalf("unexpected runtime posture: %+v", section)
	}
	if _, err := os.Stat(filepath.Join(bundleDir, "broker-validation-summary.json")); err != nil {
		t.Fatalf("expected bundled summary: %v", err)
	}
}

func TestAutomationExportValidationBundleBuildComponentSectionCapturesRootCause(t *testing.T) {
	root := t.TempDir()
	bundleDir := filepath.Join(root, "docs/reports/live-validation-runs/run-k8s")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(root, "tmp/kubernetes-smoke-report.json")
	stdoutPath := filepath.Join(root, "tmp/kubernetes.stdout.log")
	stderrPath := filepath.Join(root, "tmp/kubernetes.stderr.log")
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		t.Fatal(err)
	}
	reportBody := `{
  "task": {"id": "kubernetes-smoke-failed", "required_executor": "kubernetes"},
  "status": {
    "state": "dead_letter",
    "latest_event": {
      "id": "evt-dead",
      "type": "task.dead_letter",
      "timestamp": "2026-03-23T11:02:00Z",
      "payload": {"message": "lease lost during replay"}
    }
  },
  "events": [
    {
      "id": "evt-routed",
      "type": "scheduler.routed",
      "timestamp": "2026-03-23T11:00:00Z",
      "payload": {"reason": "browser workloads default to kubernetes executor"}
    },
    {
      "id": "evt-dead",
      "type": "task.dead_letter",
      "timestamp": "2026-03-23T11:02:00Z",
      "payload": {"message": "lease lost during replay"}
    }
  ]
}`
	if err := os.WriteFile(reportPath, []byte(reportBody), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stdoutPath, []byte("starting kubernetes smoke\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stderrPath, []byte("lease lost during replay\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	section, err := e2eBuildComponentSection("kubernetes", true, root, bundleDir, reportPath, stdoutPath, stderrPath)
	if err != nil {
		t.Fatalf("build component section: %v", err)
	}
	if section["status"] != "dead_letter" {
		t.Fatalf("unexpected status: %+v", section)
	}
	matrix, _ := section["validation_matrix"].(map[string]any)
	if matrix["lane"] != "k8s" || matrix["executor"] != "kubernetes" {
		t.Fatalf("unexpected validation matrix: %+v", matrix)
	}
	rootCause, _ := section["failure_root_cause"].(map[string]any)
	if rootCause["event_type"] != "task.dead_letter" || rootCause["message"] != "lease lost during replay" {
		t.Fatalf("unexpected root cause: %+v", rootCause)
	}
}

func TestAutomationExportValidationBundleRenderIndexShowsContinuationGate(t *testing.T) {
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
			"validation_matrix":     map[string]any{"lane": "local"},
			"failure_root_cause":    map[string]any{"status": "not_triggered", "event_type": "task.completed", "location": "x"},
		},
		"kubernetes": map[string]any{"enabled": false, "status": "skipped", "bundle_report_path": "a", "canonical_report_path": "b", "validation_matrix": map[string]any{}, "failure_root_cause": map[string]any{"status": "not_triggered", "event_type": "", "location": "x"}},
		"ray":        map[string]any{"enabled": false, "status": "skipped", "bundle_report_path": "a", "canonical_report_path": "b", "validation_matrix": map[string]any{}, "failure_root_cause": map[string]any{"status": "not_triggered", "event_type": "", "location": "x"}},
		"broker": map[string]any{
			"enabled":                          false,
			"status":                           "skipped",
			"configuration_state":              "not_configured",
			"bundle_summary_path":              "docs/reports/live-validation-runs/20260316T140138Z/broker-validation-summary.json",
			"canonical_summary_path":           "docs/reports/broker-validation-summary.json",
			"bundle_bootstrap_summary_path":    "docs/reports/live-validation-runs/20260316T140138Z/broker-bootstrap-review-summary.json",
			"canonical_bootstrap_summary_path": "docs/reports/broker-bootstrap-review-summary.json",
			"validation_pack_path":             "docs/reports/broker-failover-fault-injection-validation-pack.md",
			"runtime_posture":                  "contract_only",
			"live_adapter_implemented":         false,
			"validation_errors":                []any{"broker event log config missing driver, urls, topic"},
			"reason":                           "not_configured",
		},
		"shared_queue_companion": map[string]any{
			"available":              true,
			"status":                 "succeeded",
			"bundle_summary_path":    "x",
			"canonical_summary_path": "y",
			"bundle_report_path":     "z",
			"canonical_report_path":  "r",
		},
	}
	continuationGate := map[string]any{
		"path":           "docs/reports/validation-bundle-continuation-policy-gate.json",
		"status":         "policy-hold",
		"recommendation": "hold",
		"enforcement":    map[string]any{"mode": "hold", "outcome": "hold"},
		"summary":        map[string]any{"workflow_exit_code": 2},
	}

	indexText := e2eRenderIndex(summary, nil, continuationGate, nil, nil)
	for _, needle := range []string{"- Workflow mode: `hold`", "- Workflow outcome: `hold`", "- Workflow exit code on current evidence: `2`", "### broker", "- Runtime posture: `contract_only`"} {
		if !strings.Contains(indexText, needle) {
			t.Fatalf("expected %q in index text:\n%s", needle, indexText)
		}
	}
}

func TestRunAllUsesGoBundleCommandsAndDefaultsHoldMode(t *testing.T) {
	root := t.TempDir()
	runAllPath := filepath.Join(root, "scripts/e2e/run_all.sh")
	if err := os.MkdirAll(filepath.Dir(runAllPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runAllPath, mustReadSourceRunAll(t), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	stubGo := `#!/usr/bin/env python3
import json, pathlib, sys
args = sys.argv[1:]
if args[:4] == ['run', './cmd/bigclawctl', 'automation', 'e2e'] or (len(args) >= 4 and args[0] == 'run' and args[1].endswith('/cmd/bigclawctl') and args[2] == 'automation' and args[3] == 'e2e'):
    sub = args[4]
    if sub == 'run-task-smoke':
        report_path = pathlib.Path(args[args.index('--report-path') + 1])
        report_path.parent.mkdir(parents=True, exist_ok=True)
        report_path.write_text(json.dumps({'status': 'succeeded', 'all_ok': True}), encoding='utf-8')
        sys.exit(0)
    if sub == 'export-validation-bundle':
        root = pathlib.Path(args[args.index('--go-root') + 1])
        bundle_dir = root / args[args.index('--bundle-dir') + 1]
        bundle_dir.mkdir(parents=True, exist_ok=True)
        calls_path = root / 'calls.jsonl'
        gate_path = root / 'bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json'
        payload = {
            'gate_exists': gate_path.exists(),
            'run_broker': args[args.index('--run-broker') + 1],
            'broker_backend': args[args.index('--broker-backend') + 1],
            'broker_report_path': args[args.index('--broker-report-path') + 1],
            'broker_bootstrap_summary_path': args[args.index('--broker-bootstrap-summary-path') + 1],
        }
        with calls_path.open('a', encoding='utf-8') as handle:
            handle.write(json.dumps(payload) + '\n')
        sys.exit(0)
    if sub == 'continuation-scorecard':
        output = pathlib.Path(args[args.index('--output') + 1])
        output.parent.mkdir(parents=True, exist_ok=True)
        output.write_text(json.dumps({'summary': {}, 'shared_queue_companion': {'available': True}}), encoding='utf-8')
        sys.exit(0)
    if sub == 'continuation-policy-gate':
        mode = args[args.index('--enforcement-mode') + 1]
        output = pathlib.Path(args[args.index('--output') + 1])
        output.parent.mkdir(parents=True, exist_ok=True)
        output.write_text(json.dumps({'enforcement': {'mode': mode}, 'status': 'policy-go', 'recommendation': 'go'}), encoding='utf-8')
        sys.exit(0)
if args[:2] == ['run', './scripts/e2e/broker_bootstrap_summary.go'] or (len(args) >= 2 and args[0] == 'run' and args[1].endswith('/scripts/e2e/broker_bootstrap_summary.go')):
    output = pathlib.Path(args[args.index('--output') + 1])
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text('{"ready":false,"runtime_posture":"contract_only","live_adapter_implemented":false}\n', encoding='utf-8')
    sys.exit(0)
raise SystemExit(f'unexpected go args: {args}')
`
	if err := os.WriteFile(filepath.Join(root, "bin/go"), []byte(stubGo), 0o755); err != nil {
		t.Fatal(err)
	}

	env := append(os.Environ(),
		"BIGCLAW_E2E_RUN_KUBERNETES=0",
		"BIGCLAW_E2E_RUN_RAY=0",
		"BIGCLAW_E2E_RUN_LOCAL=1",
		"BIGCLAW_E2E_RUN_BROKER=1",
		"BIGCLAW_E2E_BROKER_BACKEND=stub",
		"BIGCLAW_E2E_BROKER_REPORT_PATH=docs/reports/broker-failover-stub-report.json",
		"PATH="+filepath.Join(root, "bin")+":"+os.Getenv("PATH"),
	)
	cmd := exec.Command(runAllPath)
	cmd.Dir = root
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run run_all.sh: %v\n%s", err, string(output))
	}
	callsBody, err := os.ReadFile(filepath.Join(root, "calls.jsonl"))
	if err != nil {
		t.Fatalf("read calls: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(callsBody)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 export bundle calls, got %d", len(lines))
	}
	var firstCall, secondCall map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &firstCall); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &secondCall); err != nil {
		t.Fatal(err)
	}
	if firstCall["gate_exists"] != false || secondCall["gate_exists"] != true {
		t.Fatalf("unexpected gate existence sequence: %+v %+v", firstCall, secondCall)
	}
	gateBody, err := os.ReadFile(filepath.Join(root, "bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json"))
	if err != nil {
		t.Fatalf("read gate file: %v", err)
	}
	if !strings.Contains(string(gateBody), `"mode": "hold"`) {
		t.Fatalf("expected hold mode gate: %s", string(gateBody))
	}
}

func mustReadSourceRunAll(t *testing.T) []byte {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve current file")
	}
	sourcePath := filepath.Join(filepath.Dir(currentFile), "..", "..", "scripts", "e2e", "run_all.sh")
	body, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source run_all.sh: %v", err)
	}
	return body
}
