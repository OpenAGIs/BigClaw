package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAllScriptContinuationFlowStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	sourceRunAll := filepath.Join(repoRoot, "scripts", "e2e", "run_all.sh")

	type callRecord struct {
		GateExists                 bool   `json:"gate_exists"`
		RunBroker                  string `json:"run_broker"`
		BrokerBackend              string `json:"broker_backend"`
		BrokerReportPath           string `json:"broker_report_path"`
		BrokerBootstrapSummaryPath string `json:"broker_bootstrap_summary_path"`
	}

	makeRoot := func(t *testing.T) string {
		t.Helper()
		root := t.TempDir()
		runAllPath := filepath.Join(root, "scripts", "e2e", "run_all.sh")
		if err := os.MkdirAll(filepath.Dir(runAllPath), 0o755); err != nil {
			t.Fatalf("mkdir run_all dir: %v", err)
		}
		contents, err := os.ReadFile(sourceRunAll)
		if err != nil {
			t.Fatalf("read source run_all: %v", err)
		}
		if err := os.WriteFile(runAllPath, contents, 0o755); err != nil {
			t.Fatalf("write run_all: %v", err)
		}

		writeFile := func(relpath, content string) {
			path := filepath.Join(root, relpath)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatalf("mkdir %s: %v", relpath, err)
			}
			if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
				t.Fatalf("write %s: %v", relpath, err)
			}
		}

		writeFile("bin/go", `#!/usr/bin/env python3
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
`)

		writeFile("scripts/e2e/export_validation_bundle.py", `#!/usr/bin/env python3
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
`)

		writeFile("scripts/e2e/validation_bundle_continuation_scorecard.py", `#!/usr/bin/env python3
import json
import pathlib
import sys

args = sys.argv[1:]
output = pathlib.Path(args[args.index('--output') + 1])
output.parent.mkdir(parents=True, exist_ok=True)
output.write_text(json.dumps({'summary': {}, 'shared_queue_companion': {'available': True}}), encoding='utf-8')
`)

		writeFile("scripts/e2e/validation-bundle-continuation-policy-gate", `#!/usr/bin/env bash
set -euo pipefail
args=("$@")
mode=hold
for ((i = 0; i < ${#args[@]}; i++)); do
  if [[ "${args[$i]}" == "--enforcement-mode" ]]; then
    mode="${args[$((i + 1))]}"
  fi
  if [[ "${args[$i]}" == "--output" ]]; then
    output="${args[$((i + 1))]}"
  fi
done
mkdir -p "$(dirname "$output")"
printf '{"status":"policy-go","recommendation":"go","enforcement":{"mode":"%s","outcome":"pass","exit_code":0}}' "$mode" >"$output"
`)

		return root
	}

	runScript := func(t *testing.T, root string, extraEnv map[string]string) []callRecord {
		t.Helper()
		env := append([]string{}, os.Environ()...)
		env = append(env, "PATH="+filepath.Join(root, "bin")+":"+os.Getenv("PATH"))
		for key, value := range extraEnv {
			env = append(env, key+"="+value)
		}
		cmd := exec.Command(filepath.Join(root, "scripts", "e2e", "run_all.sh"))
		cmd.Dir = root
		cmd.Env = env
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("run_all.sh failed: %v\n%s", err, output)
		}
		body, err := os.ReadFile(filepath.Join(root, "calls.jsonl"))
		if err != nil {
			t.Fatalf("read calls.jsonl: %v", err)
		}
		lines := strings.Split(strings.TrimSpace(string(body)), "\n")
		records := make([]callRecord, 0, len(lines))
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			var record callRecord
			if err := json.Unmarshal([]byte(line), &record); err != nil {
				t.Fatalf("decode call record: %v", err)
			}
			records = append(records, record)
		}
		return records
	}

	t.Run("rerenders bundle after gate refresh", func(t *testing.T) {
		root := makeRoot(t)
		calls := runScript(t, root, map[string]string{
			"BIGCLAW_E2E_RUN_KUBERNETES":     "0",
			"BIGCLAW_E2E_RUN_RAY":            "0",
			"BIGCLAW_E2E_RUN_LOCAL":          "1",
			"BIGCLAW_E2E_RUN_BROKER":         "1",
			"BIGCLAW_E2E_BROKER_BACKEND":     "stub",
			"BIGCLAW_E2E_BROKER_REPORT_PATH": "docs/reports/broker-failover-stub-report.json",
		})
		if len(calls) != 2 {
			t.Fatalf("expected 2 export calls, got %+v", calls)
		}
		if calls[0].GateExists || !calls[1].GateExists {
			t.Fatalf("unexpected gate existence sequence: %+v", calls)
		}
		if calls[0].RunBroker != "1" || calls[0].BrokerBackend != "stub" {
			t.Fatalf("unexpected broker invocation: %+v", calls[0])
		}
		if calls[0].BrokerReportPath != "docs/reports/broker-failover-stub-report.json" {
			t.Fatalf("unexpected broker report path: %+v", calls[0])
		}
		if calls[0].BrokerBootstrapSummaryPath != "docs/reports/broker-bootstrap-review-summary.json" {
			t.Fatalf("unexpected bootstrap summary path: %+v", calls[0])
		}
	})

	t.Run("defaults to hold mode", func(t *testing.T) {
		root := makeRoot(t)
		gateStub := `#!/usr/bin/env bash
set -euo pipefail
args=("$@")
mode=hold
for ((i = 0; i < ${#args[@]}; i++)); do
  if [[ "${args[$i]}" == "--enforcement-mode" ]]; then
    mode="${args[$((i + 1))]}"
  fi
  if [[ "${args[$i]}" == "--output" ]]; then
    output="${args[$((i + 1))]}"
  fi
done
mkdir -p "$(dirname "$output")"
printf '{"enforcement":{"mode":"%s"}}' "$mode" >"$output"
`
		if err := os.WriteFile(filepath.Join(root, "scripts", "e2e", "validation-bundle-continuation-policy-gate"), []byte(gateStub), 0o755); err != nil {
			t.Fatalf("override gate stub: %v", err)
		}
		runScript(t, root, map[string]string{
			"BIGCLAW_E2E_RUN_KUBERNETES": "0",
			"BIGCLAW_E2E_RUN_RAY":        "0",
			"BIGCLAW_E2E_RUN_LOCAL":      "1",
		})
		var gate struct {
			Enforcement struct {
				Mode string `json:"mode"`
			} `json:"enforcement"`
		}
		readJSONFile(t, filepath.Join(root, "bigclaw-go", "docs", "reports", "validation-bundle-continuation-policy-gate.json"), &gate)
		if gate.Enforcement.Mode != "hold" {
			t.Fatalf("unexpected hold mode gate: %+v", gate)
		}
	})

	t.Run("legacy enforce alias maps to fail mode", func(t *testing.T) {
		root := makeRoot(t)
		gateStub := `#!/usr/bin/env bash
set -euo pipefail
args=("$@")
mode=hold
for ((i = 0; i < ${#args[@]}; i++)); do
  if [[ "${args[$i]}" == "--enforcement-mode" ]]; then
    mode="${args[$((i + 1))]}"
  fi
  if [[ "${args[$i]}" == "--output" ]]; then
    output="${args[$((i + 1))]}"
  fi
done
mkdir -p "$(dirname "$output")"
printf '{"enforcement":{"mode":"%s"}}' "$mode" >"$output"
`
		if err := os.WriteFile(filepath.Join(root, "scripts", "e2e", "validation-bundle-continuation-policy-gate"), []byte(gateStub), 0o755); err != nil {
			t.Fatalf("override gate stub: %v", err)
		}
		runScript(t, root, map[string]string{
			"BIGCLAW_E2E_RUN_KUBERNETES":            "0",
			"BIGCLAW_E2E_RUN_RAY":                   "0",
			"BIGCLAW_E2E_RUN_LOCAL":                 "1",
			"BIGCLAW_E2E_ENFORCE_CONTINUATION_GATE": "1",
		})
		var gate struct {
			Enforcement struct {
				Mode string `json:"mode"`
			} `json:"enforcement"`
		}
		readJSONFile(t, filepath.Join(root, "bigclaw-go", "docs", "reports", "validation-bundle-continuation-policy-gate.json"), &gate)
		if gate.Enforcement.Mode != "fail" {
			t.Fatalf("unexpected fail mode gate: %+v", gate)
		}
	})
}
