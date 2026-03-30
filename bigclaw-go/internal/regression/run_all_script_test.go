package regression

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

type runAllExportCall struct {
	GateExists                 bool   `json:"gate_exists"`
	RunBroker                  string `json:"run_broker"`
	BrokerBackend              string `json:"broker_backend"`
	BrokerReportPath           string `json:"broker_report_path"`
	BrokerBootstrapSummaryPath string `json:"broker_bootstrap_summary_path"`
}

func TestRunAllRerendersBundleAfterGateRefresh(t *testing.T) {
	root := prepareRunAllFixture(t, runAllPolicyGateScript(`
mode = args[args.index('--enforcement-mode') + 1]
output.write_text(json.dumps({'status': 'policy-go', 'recommendation': 'go', 'enforcement': {'mode': mode, 'outcome': 'pass', 'exit_code': 0}}), encoding='utf-8')
`))
	result := runRunAll(t, root, map[string]string{
		"BIGCLAW_E2E_RUN_KUBERNETES":     "0",
		"BIGCLAW_E2E_RUN_RAY":            "0",
		"BIGCLAW_E2E_RUN_LOCAL":          "1",
		"BIGCLAW_E2E_RUN_BROKER":         "1",
		"BIGCLAW_E2E_BROKER_BACKEND":     "stub",
		"BIGCLAW_E2E_BROKER_REPORT_PATH": "docs/reports/broker-failover-stub-report.json",
	})
	if result.returnCode != 0 {
		t.Fatalf("run_all.sh failed: stdout=%s stderr=%s", result.stdout, result.stderr)
	}

	calls := readRunAllExportCalls(t, root)
	if len(calls) != 2 {
		t.Fatalf("expected two export calls, got %+v", calls)
	}
	if calls[0].GateExists {
		t.Fatalf("expected first export before gate render, got %+v", calls)
	}
	if !calls[1].GateExists {
		t.Fatalf("expected rerender after gate refresh, got %+v", calls)
	}
	if calls[0].RunBroker != "1" || calls[0].BrokerBackend != "stub" {
		t.Fatalf("unexpected broker export payload: %+v", calls[0])
	}
	if calls[0].BrokerReportPath != "docs/reports/broker-failover-stub-report.json" {
		t.Fatalf("unexpected broker report path: %+v", calls[0])
	}
	if calls[0].BrokerBootstrapSummaryPath != "docs/reports/broker-bootstrap-review-summary.json" {
		t.Fatalf("unexpected broker bootstrap summary path: %+v", calls[0])
	}
}

func TestRunAllDefaultsToHoldMode(t *testing.T) {
	root := prepareRunAllFixture(t, runAllPolicyGateScript(`
mode = args[args.index('--enforcement-mode') + 1]
output.write_text(json.dumps({'enforcement': {'mode': mode}}), encoding='utf-8')
`))
	result := runRunAll(t, root, map[string]string{
		"BIGCLAW_E2E_RUN_KUBERNETES": "0",
		"BIGCLAW_E2E_RUN_RAY":        "0",
		"BIGCLAW_E2E_RUN_LOCAL":      "1",
	})
	if result.returnCode != 0 {
		t.Fatalf("run_all.sh failed: stdout=%s stderr=%s", result.stdout, result.stderr)
	}

	var gate struct {
		Enforcement struct {
			Mode string `json:"mode"`
		} `json:"enforcement"`
	}
	readJSONFile(t, filepath.Join(root, "bigclaw-go", "docs", "reports", "validation-bundle-continuation-policy-gate.json"), &gate)
	if gate.Enforcement.Mode != "hold" {
		t.Fatalf("expected hold mode, got %+v", gate)
	}
}

func TestRunAllLegacyEnforceAliasMapsToFailMode(t *testing.T) {
	root := prepareRunAllFixture(t, runAllPolicyGateScript(`
mode = args[args.index('--enforcement-mode') + 1]
output.write_text(json.dumps({'enforcement': {'mode': mode}}), encoding='utf-8')
`))
	result := runRunAll(t, root, map[string]string{
		"BIGCLAW_E2E_RUN_KUBERNETES":            "0",
		"BIGCLAW_E2E_RUN_RAY":                   "0",
		"BIGCLAW_E2E_RUN_LOCAL":                 "1",
		"BIGCLAW_E2E_ENFORCE_CONTINUATION_GATE": "1",
	})
	if result.returnCode != 0 {
		t.Fatalf("run_all.sh failed: stdout=%s stderr=%s", result.stdout, result.stderr)
	}

	var gate struct {
		Enforcement struct {
			Mode string `json:"mode"`
		} `json:"enforcement"`
	}
	readJSONFile(t, filepath.Join(root, "bigclaw-go", "docs", "reports", "validation-bundle-continuation-policy-gate.json"), &gate)
	if gate.Enforcement.Mode != "fail" {
		t.Fatalf("expected fail mode, got %+v", gate)
	}
}

type runAllResult struct {
	returnCode int
	stdout     string
	stderr     string
}

func prepareRunAllFixture(t *testing.T, policyGateScript string) string {
	t.Helper()
	root := t.TempDir()
	runAllSource := filepath.Join(repoRoot(t), "scripts", "e2e", "run_all.sh")
	runAllTarget := filepath.Join(root, "scripts", "e2e", "run_all.sh")
	copyExecutableFile(t, runAllSource, runAllTarget)

	writeExecutableFile(t, filepath.Join(root, "bin", "go"), runAllGoStub)
	writeExecutableFile(t, filepath.Join(root, "scripts", "e2e", "export_validation_bundle.py"), runAllExportBundleStub)
	writeExecutableFile(t, filepath.Join(root, "scripts", "e2e", "validation_bundle_continuation_scorecard.py"), runAllScorecardStub)
	writeExecutableFile(t, filepath.Join(root, "scripts", "e2e", "validation_bundle_continuation_policy_gate.py"), policyGateScript)
	return root
}

func runRunAll(t *testing.T, root string, extraEnv map[string]string) runAllResult {
	t.Helper()
	cmd := exec.Command(filepath.Join(root, "scripts", "e2e", "run_all.sh"))
	cmd.Dir = root
	env := append([]string{}, os.Environ()...)
	env = append(env, "PATH="+filepath.Join(root, "bin")+":"+os.Getenv("PATH"))
	for key, value := range extraEnv {
		env = append(env, key+"="+value)
	}
	cmd.Env = env
	stdout, err := cmd.Output()
	if err == nil {
		return runAllResult{returnCode: 0, stdout: string(stdout)}
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("run_all.sh: %v", err)
	}
	return runAllResult{
		returnCode: exitErr.ExitCode(),
		stdout:     string(stdout),
		stderr:     string(exitErr.Stderr),
	}
}

func readRunAllExportCalls(t *testing.T, root string) []runAllExportCall {
	t.Helper()
	file, err := os.Open(filepath.Join(root, "calls.jsonl"))
	if err != nil {
		t.Fatalf("open calls.jsonl: %v", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	calls := []runAllExportCall{}
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var call runAllExportCall
		if err := json.Unmarshal(line, &call); err != nil {
			t.Fatalf("decode export call: %v", err)
		}
		calls = append(calls, call)
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan calls.jsonl: %v", err)
	}
	return calls
}

func copyExecutableFile(t *testing.T, source string, destination string) {
	t.Helper()
	body, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read %s: %v", source, err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", destination, err)
	}
	if err := os.WriteFile(destination, body, 0o755); err != nil {
		t.Fatalf("write %s: %v", destination, err)
	}
}

func writeExecutableFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func runAllPolicyGateScript(body string) string {
	return "#!/usr/bin/env python3\nimport json\nimport pathlib\nimport sys\n\nargs = sys.argv[1:]\noutput = pathlib.Path(args[args.index('--output') + 1])\noutput.parent.mkdir(parents=True, exist_ok=True)\n" + body + "\n"
}

const runAllGoStub = `#!/usr/bin/env python3
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
    raise SystemExit(0)
if args[:2] == ['run', './scripts/e2e/broker_bootstrap_summary.go'] or (
    len(args) >= 2 and args[0] == 'run' and args[1].endswith('/scripts/e2e/broker_bootstrap_summary.go')
):
    output = pathlib.Path(args[args.index('--output') + 1])
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text('{"ready":false,"runtime_posture":"contract_only","live_adapter_implemented":false}\n', encoding='utf-8')
    raise SystemExit(0)
raise SystemExit(f'unexpected go args: {args}')
`

const runAllExportBundleStub = `#!/usr/bin/env python3
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
`

const runAllScorecardStub = `#!/usr/bin/env python3
import json
import pathlib
import sys

args = sys.argv[1:]
output = pathlib.Path(args[args.index('--output') + 1])
output.parent.mkdir(parents=True, exist_ok=True)
output.write_text(json.dumps({'summary': {}, 'shared_queue_companion': {'available': True}}), encoding='utf-8')
`
