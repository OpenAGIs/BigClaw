package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAllRerendersBundleAfterGateRefresh(t *testing.T) {
	root := setupRunAllFixture(t)
	env := append(os.Environ(),
		"BIGCLAW_E2E_RUN_KUBERNETES=0",
		"BIGCLAW_E2E_RUN_RAY=0",
		"BIGCLAW_E2E_RUN_LOCAL=1",
		"BIGCLAW_E2E_RUN_BROKER=1",
		"BIGCLAW_E2E_BROKER_BACKEND=stub",
		"BIGCLAW_E2E_BROKER_REPORT_PATH=docs/reports/broker-failover-stub-report.json",
		"PATH="+filepath.Join(root, "bin")+":"+os.Getenv("PATH"),
	)

	run := exec.Command(filepath.Join(root, "scripts", "e2e", "run_all.sh"))
	run.Dir = root
	run.Env = env
	output, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run_all.sh failed: %v\n%s", err, output)
	}

	body, err := os.ReadFile(filepath.Join(root, "calls.jsonl"))
	if err != nil {
		t.Fatalf("read calls.jsonl: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 export calls, got %d: %q", len(lines), body)
	}

	var first, second map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("decode first export call: %v", err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("decode second export call: %v", err)
	}

	if first["gate_exists"] != false || second["gate_exists"] != true {
		t.Fatalf("unexpected gate existence sequence: first=%v second=%v", first["gate_exists"], second["gate_exists"])
	}
	if first["run_broker"] != "1" || first["broker_backend"] != "stub" {
		t.Fatalf("unexpected broker arguments: %+v", first)
	}
	if first["broker_report_path"] != "docs/reports/broker-failover-stub-report.json" {
		t.Fatalf("unexpected broker report path: %+v", first)
	}
	if first["broker_bootstrap_summary_path"] != "docs/reports/broker-bootstrap-review-summary.json" {
		t.Fatalf("unexpected broker bootstrap summary path: %+v", first)
	}
}

func TestRunAllDefaultsToHoldMode(t *testing.T) {
	root := setupRunAllFixture(t)
	writeTestFile(t, root, "scripts/e2e/validation_bundle_continuation_policy_gate.py", `#!/usr/bin/env python3
import json
import pathlib
import sys

args = sys.argv[1:]
mode = args[args.index('--enforcement-mode') + 1]
output = pathlib.Path(args[args.index('--output') + 1])
output.parent.mkdir(parents=True, exist_ok=True)
output.write_text(json.dumps({'enforcement': {'mode': mode}}), encoding='utf-8')
`, true)

	env := append(os.Environ(),
		"BIGCLAW_E2E_RUN_KUBERNETES=0",
		"BIGCLAW_E2E_RUN_RAY=0",
		"BIGCLAW_E2E_RUN_LOCAL=1",
		"PATH="+filepath.Join(root, "bin")+":"+os.Getenv("PATH"),
	)

	run := exec.Command(filepath.Join(root, "scripts", "e2e", "run_all.sh"))
	run.Dir = root
	run.Env = env
	output, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run_all.sh failed: %v\n%s", err, output)
	}

	gate := readJSONMap(t, filepath.Join(root, "bigclaw-go", "docs", "reports", "validation-bundle-continuation-policy-gate.json"))
	enforcement, ok := gate["enforcement"].(map[string]any)
	if !ok || enforcement["mode"] != "hold" {
		t.Fatalf("unexpected gate enforcement: %+v", gate)
	}
}

func TestRunAllLegacyEnforceAliasStillMapsToFailMode(t *testing.T) {
	root := setupRunAllFixture(t)
	writeTestFile(t, root, "scripts/e2e/validation_bundle_continuation_policy_gate.py", `#!/usr/bin/env python3
import json
import pathlib
import sys

args = sys.argv[1:]
mode = args[args.index('--enforcement-mode') + 1]
output = pathlib.Path(args[args.index('--output') + 1])
output.parent.mkdir(parents=True, exist_ok=True)
output.write_text(json.dumps({'enforcement': {'mode': mode}}), encoding='utf-8')
`, true)

	env := append(os.Environ(),
		"BIGCLAW_E2E_RUN_KUBERNETES=0",
		"BIGCLAW_E2E_RUN_RAY=0",
		"BIGCLAW_E2E_RUN_LOCAL=1",
		"BIGCLAW_E2E_ENFORCE_CONTINUATION_GATE=1",
		"PATH="+filepath.Join(root, "bin")+":"+os.Getenv("PATH"),
	)

	run := exec.Command(filepath.Join(root, "scripts", "e2e", "run_all.sh"))
	run.Dir = root
	run.Env = env
	output, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("run_all.sh failed: %v\n%s", err, output)
	}

	gate := readJSONMap(t, filepath.Join(root, "bigclaw-go", "docs", "reports", "validation-bundle-continuation-policy-gate.json"))
	enforcement, ok := gate["enforcement"].(map[string]any)
	if !ok || enforcement["mode"] != "fail" {
		t.Fatalf("unexpected gate enforcement: %+v", gate)
	}
}

func setupRunAllFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(repoRootForScriptTests(t), "scripts", "e2e", "run_all.sh")
	body, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read source run_all.sh: %v", err)
	}
	writeTestFile(t, root, "scripts/e2e/run_all.sh", string(body), true)
	writeTestFile(t, root, "bin/go", `#!/usr/bin/env python3
import json
import pathlib
import sys

args = sys.argv[1:]
if args[:4] == ['run', './cmd/bigclawctl', 'automation', 'e2e'] or (
    len(args) >= 4 and args[0] == 'run' and args[1].endswith('/cmd/bigclawctl') and args[2] == 'automation' and args[3] == 'e2e'
):
    report_path = pathlib.Path(args[args.index('--report-path') + 1])
    report_path.parent.mkdir(parents=True, exist_ok=True)
    report_path.write_text(json.dumps({'status': 'succeeded', 'all_ok': True}), encoding='utf-8')
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
	writeTestFile(t, root, "scripts/e2e/export_validation_bundle.py", `#!/usr/bin/env python3
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
	writeTestFile(t, root, "scripts/e2e/validation_bundle_continuation_scorecard.py", `#!/usr/bin/env python3
import json
import pathlib
import sys

args = sys.argv[1:]
output = pathlib.Path(args[args.index('--output') + 1])
output.parent.mkdir(parents=True, exist_ok=True)
output.write_text(json.dumps({'summary': {}, 'shared_queue_companion': {'available': True}}), encoding='utf-8')
`, true)
	writeTestFile(t, root, "scripts/e2e/validation_bundle_continuation_policy_gate.py", `#!/usr/bin/env python3
import json
import pathlib
import sys

args = sys.argv[1:]
mode = args[args.index('--enforcement-mode') + 1]
output = pathlib.Path(args[args.index('--output') + 1])
output.parent.mkdir(parents=True, exist_ok=True)
output.write_text(json.dumps({'status': 'policy-go', 'recommendation': 'go', 'enforcement': {'mode': mode, 'outcome': 'pass', 'exit_code': 0}}), encoding='utf-8')
`, true)
	return root
}

func repoRootForScriptTests(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func writeTestFile(t *testing.T, root, rel, body string, executable bool) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
	if executable {
		if err := os.Chmod(path, 0o755); err != nil {
			t.Fatalf("chmod %s: %v", rel, err)
		}
	}
}

func readJSONMap(t *testing.T, path string) map[string]any {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return payload
}
