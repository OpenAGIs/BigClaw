package main

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

type runAllHarness struct {
	t    *testing.T
	root string
}

func newRunAllHarness(t *testing.T) *runAllHarness {
	t.Helper()
	root := t.TempDir()
	h := &runAllHarness{t: t, root: root}
	h.installRunAllScript()
	return h
}

func e2eFile(t *testing.T, name string) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller path")
	}
	return filepath.Join(filepath.Dir(filename), name)
}

func (h *runAllHarness) installRunAllScript() {
	h.t.Helper()
	scriptsDir := filepath.Join(h.root, "scripts", "e2e")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		h.t.Fatalf("mkdir scripts dir: %v", err)
	}

	sourcePath := e2eFile(h.t, "run_all.sh")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		h.t.Fatalf("read source run_all.sh: %v", err)
	}

	targetPath := filepath.Join(scriptsDir, "run_all.sh")
	if err := os.WriteFile(targetPath, source, 0o755); err != nil {
		h.t.Fatalf("write temp run_all.sh: %v", err)
	}
}

func (h *runAllHarness) writeFile(relPath, content string, executable bool) {
	h.t.Helper()
	path := filepath.Join(h.root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		h.t.Fatalf("mkdir for %s: %v", relPath, err)
	}
	mode := os.FileMode(0o644)
	if executable {
		mode = 0o755
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		h.t.Fatalf("write %s: %v", relPath, err)
	}
}

func (h *runAllHarness) installStubs() {
	h.t.Helper()
	h.writeFile("bin/go", `#!/usr/bin/env python3
import json
import pathlib
import sys

args = sys.argv[1:]
if args[:4] == ['run', './cmd/bigclawctl', 'automation', 'e2e'] or (
    len(args) >= 4 and args[0] == 'run' and args[1].endswith('/cmd/bigclawctl') and args[2] == 'automation' and args[3] == 'e2e'
):
    subcommand = args[4]
    if subcommand == 'run-task-smoke':
        report_path = pathlib.Path(args[args.index('--report-path') + 1])
        report_path.parent.mkdir(parents=True, exist_ok=True)
        report_path.write_text(json.dumps({'status': 'succeeded', 'all_ok': True}), encoding='utf-8')
        sys.exit(0)
    if subcommand == 'validation-bundle-continuation-scorecard':
        output = pathlib.Path(args[args.index('--output') + 1])
        output.parent.mkdir(parents=True, exist_ok=True)
        output.write_text(json.dumps({'summary': {}, 'shared_queue_companion': {'available': True}}), encoding='utf-8')
        sys.exit(0)
    if subcommand == 'validation-bundle-continuation-policy-gate':
        mode = args[args.index('--enforcement-mode') + 1]
        output = pathlib.Path(args[args.index('--output') + 1])
        output.parent.mkdir(parents=True, exist_ok=True)
        output.write_text(json.dumps({'status': 'policy-go', 'recommendation': 'go', 'enforcement': {'mode': mode, 'outcome': 'pass', 'exit_code': 0}}), encoding='utf-8')
        sys.exit(0)
if args[:2] == ['run', './scripts/e2e/broker_bootstrap_summary.go'] or (
    len(args) >= 2 and args[0] == 'run' and args[1].endswith('/scripts/e2e/broker_bootstrap_summary.go')
):
    output = pathlib.Path(args[args.index('--output') + 1])
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text('{"ready":false,"runtime_posture":"contract_only","live_adapter_implemented":false}\n', encoding='utf-8')
    sys.exit(0)
raise SystemExit(f'unexpected go args: {args}')
`, true)

	h.writeFile("scripts/e2e/export_validation_bundle.py", `#!/usr/bin/env python3
import json
import pathlib
import sys

args = sys.argv[1:]
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
`, true)
}

func (h *runAllHarness) run(env map[string]string) *exec.Cmd {
	h.t.Helper()
	cmd := exec.Command(filepath.Join(h.root, "scripts", "e2e", "run_all.sh"))
	cmd.Dir = h.root

	base := os.Environ()
	with := make([]string, 0, len(base)+len(env))
	with = append(with, base...)
	for k, v := range env {
		with = append(with, k+"="+v)
	}
	cmd.Env = with
	return cmd
}

func (h *runAllHarness) readCalls() []map[string]any {
	h.t.Helper()
	path := filepath.Join(h.root, "calls.jsonl")
	file, err := os.Open(path)
	if err != nil {
		h.t.Fatalf("open calls.jsonl: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var calls []map[string]any
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			h.t.Fatalf("decode call line: %v", err)
		}
		calls = append(calls, payload)
	}
	if err := scanner.Err(); err != nil {
		h.t.Fatalf("scan calls.jsonl: %v", err)
	}
	return calls
}

func (h *runAllHarness) continuationGatePath() string {
	return filepath.Join(h.root, "bigclaw-go", "docs", "reports", "validation-bundle-continuation-policy-gate.json")
}

func TestRunAllRerendersBundleAfterGateRefresh(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is required for run_all.sh stubs")
	}

	h := newRunAllHarness(t)
	h.installStubs()

	env := map[string]string{
		"BIGCLAW_E2E_RUN_KUBERNETES":     "0",
		"BIGCLAW_E2E_RUN_RAY":            "0",
		"BIGCLAW_E2E_RUN_LOCAL":          "1",
		"BIGCLAW_E2E_RUN_BROKER":         "1",
		"BIGCLAW_E2E_BROKER_BACKEND":     "stub",
		"BIGCLAW_E2E_BROKER_REPORT_PATH": "docs/reports/broker-failover-stub-report.json",
		"PATH":                           filepath.Join(h.root, "bin") + string(os.PathListSeparator) + os.Getenv("PATH"),
	}
	cmd := h.run(env)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run_all.sh failed: %v\n%s", err, string(output))
	}

	calls := h.readCalls()
	if len(calls) != 2 {
		t.Fatalf("expected 2 export calls, got %d", len(calls))
	}
	if calls[0]["gate_exists"] != false || calls[1]["gate_exists"] != true {
		t.Fatalf("unexpected gate transition: %+v", calls)
	}
	if calls[0]["run_broker"] != "1" {
		t.Fatalf("unexpected run_broker: %+v", calls[0]["run_broker"])
	}
	if calls[0]["broker_backend"] != "stub" {
		t.Fatalf("unexpected broker_backend: %+v", calls[0]["broker_backend"])
	}
	if calls[0]["broker_report_path"] != "docs/reports/broker-failover-stub-report.json" {
		t.Fatalf("unexpected broker_report_path: %+v", calls[0]["broker_report_path"])
	}
	if calls[0]["broker_bootstrap_summary_path"] != "docs/reports/broker-bootstrap-review-summary.json" {
		t.Fatalf("unexpected broker_bootstrap_summary_path: %+v", calls[0]["broker_bootstrap_summary_path"])
	}
}

func TestRunAllDefaultsToHoldMode(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is required for run_all.sh stubs")
	}

	h := newRunAllHarness(t)
	h.installStubs()

	env := map[string]string{
		"BIGCLAW_E2E_RUN_KUBERNETES": "0",
		"BIGCLAW_E2E_RUN_RAY":        "0",
		"BIGCLAW_E2E_RUN_LOCAL":      "1",
		"PATH":                       filepath.Join(h.root, "bin") + string(os.PathListSeparator) + os.Getenv("PATH"),
	}
	cmd := h.run(env)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run_all.sh failed: %v\n%s", err, string(output))
	}

	body, err := os.ReadFile(h.continuationGatePath())
	if err != nil {
		t.Fatalf("read continuation gate: %v", err)
	}
	var gate map[string]any
	if err := json.Unmarshal(body, &gate); err != nil {
		t.Fatalf("decode gate json: %v", err)
	}
	enforcement, ok := gate["enforcement"].(map[string]any)
	if !ok {
		t.Fatalf("missing enforcement payload: %+v", gate)
	}
	if enforcement["mode"] != "hold" {
		t.Fatalf("expected hold mode, got %+v", enforcement["mode"])
	}
}

func TestLegacyEnforceAliasStillMapsToFailMode(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is required for run_all.sh stubs")
	}

	h := newRunAllHarness(t)
	h.installStubs()

	env := map[string]string{
		"BIGCLAW_E2E_RUN_KUBERNETES":            "0",
		"BIGCLAW_E2E_RUN_RAY":                   "0",
		"BIGCLAW_E2E_RUN_LOCAL":                 "1",
		"BIGCLAW_E2E_ENFORCE_CONTINUATION_GATE": "1",
		"PATH":                                  filepath.Join(h.root, "bin") + string(os.PathListSeparator) + os.Getenv("PATH"),
	}
	cmd := h.run(env)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run_all.sh failed: %v\n%s", err, string(output))
	}

	body, err := os.ReadFile(h.continuationGatePath())
	if err != nil {
		t.Fatalf("read continuation gate: %v", err)
	}
	var gate map[string]any
	if err := json.Unmarshal(body, &gate); err != nil {
		t.Fatalf("decode gate json: %v", err)
	}
	enforcement, ok := gate["enforcement"].(map[string]any)
	if !ok {
		t.Fatalf("missing enforcement payload: %+v", gate)
	}
	if enforcement["mode"] != "fail" {
		t.Fatalf("expected fail mode, got %+v", enforcement["mode"])
	}
}
