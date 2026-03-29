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
	harness := newRunAllHarness(t)
	harness.installStubs()

	result := harness.run(map[string]string{
		"BIGCLAW_E2E_RUN_KUBERNETES":     "0",
		"BIGCLAW_E2E_RUN_RAY":            "0",
		"BIGCLAW_E2E_RUN_LOCAL":          "1",
		"BIGCLAW_E2E_RUN_BROKER":         "1",
		"BIGCLAW_E2E_BROKER_BACKEND":     "stub",
		"BIGCLAW_E2E_BROKER_REPORT_PATH": "docs/reports/broker-failover-stub-report.json",
	})
	if result.Err != nil {
		t.Fatalf("run_all.sh failed: %v\nstderr:\n%s", result.Err, result.Stderr)
	}

	var calls []map[string]any
	readJSONLines(t, filepath.Join(harness.root, "calls.jsonl"), &calls)
	if len(calls) != 2 {
		t.Fatalf("call count = %d, want 2", len(calls))
	}
	if asBoolForRunAll(calls[0]["gate_exists"]) {
		t.Fatalf("expected first export to run before gate refresh: %+v", calls[0])
	}
	if !asBoolForRunAll(calls[1]["gate_exists"]) {
		t.Fatalf("expected second export to observe gate artifact: %+v", calls[1])
	}
	if calls[0]["run_broker"] != "1" || calls[0]["broker_backend"] != "stub" || calls[0]["broker_report_path"] != "docs/reports/broker-failover-stub-report.json" || calls[0]["broker_bootstrap_summary_path"] != "docs/reports/broker-bootstrap-review-summary.json" {
		t.Fatalf("unexpected broker call payload: %+v", calls[0])
	}
}

func TestRunAllDefaultsToHoldMode(t *testing.T) {
	harness := newRunAllHarness(t)
	harness.installStubs()
	harness.writeFile("scripts/e2e/validation_bundle_continuation_policy_gate.go", runAllGateModeOnlyStub)

	result := harness.run(map[string]string{
		"BIGCLAW_E2E_RUN_KUBERNETES": "0",
		"BIGCLAW_E2E_RUN_RAY":        "0",
		"BIGCLAW_E2E_RUN_LOCAL":      "1",
	})
	if result.Err != nil {
		t.Fatalf("run_all.sh failed: %v\nstderr:\n%s", result.Err, result.Stderr)
	}

	var gate struct {
		Enforcement struct {
			Mode string `json:"mode"`
		} `json:"enforcement"`
	}
	readJSONFileForRunAll(t, harness.findGatePath(), &gate)
	if gate.Enforcement.Mode != "hold" {
		t.Fatalf("gate enforcement mode = %q, want hold", gate.Enforcement.Mode)
	}
}

func TestRunAllLegacyEnforceAliasStillMapsToFailMode(t *testing.T) {
	harness := newRunAllHarness(t)
	harness.installStubs()
	harness.writeFile("scripts/e2e/validation_bundle_continuation_policy_gate.go", runAllGateModeOnlyStub)

	result := harness.run(map[string]string{
		"BIGCLAW_E2E_RUN_KUBERNETES":            "0",
		"BIGCLAW_E2E_RUN_RAY":                   "0",
		"BIGCLAW_E2E_RUN_LOCAL":                 "1",
		"BIGCLAW_E2E_ENFORCE_CONTINUATION_GATE": "1",
	})
	if result.Err != nil {
		t.Fatalf("run_all.sh failed: %v\nstderr:\n%s", result.Err, result.Stderr)
	}

	var gate struct {
		Enforcement struct {
			Mode string `json:"mode"`
		} `json:"enforcement"`
	}
	readJSONFileForRunAll(t, harness.findGatePath(), &gate)
	if gate.Enforcement.Mode != "fail" {
		t.Fatalf("gate enforcement mode = %q, want fail", gate.Enforcement.Mode)
	}
}

type runAllHarness struct {
	root string
}

type runAllResult struct {
	Stdout string
	Stderr string
	Err    error
}

func newRunAllHarness(t *testing.T) runAllHarness {
	t.Helper()
	root := t.TempDir()
	sourcePath := filepath.Join(repoRootForRunAllTest(t), "scripts", "e2e", "run_all.sh")
	body, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source run_all.sh: %v", err)
	}
	scriptsDir := filepath.Join(root, "scripts", "e2e")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	targetPath := filepath.Join(scriptsDir, "run_all.sh")
	if err := os.WriteFile(targetPath, body, 0o755); err != nil {
		t.Fatalf("write run_all.sh: %v", err)
	}
	return runAllHarness{root: root}
}

func (h runAllHarness) installStubs() {
	h.writeFile("scripts/e2e/run_task_smoke.py", runAllTaskSmokeStub)
	if err := os.Chmod(filepath.Join(h.root, "scripts", "e2e", "run_task_smoke.py"), 0o755); err != nil {
		panic(err)
	}
	h.writeFile("scripts/e2e/broker_bootstrap_summary.go", runAllBrokerBootstrapStub)
	h.writeFile("scripts/e2e/export_validation_bundle.py", runAllExportBundleStub)
	if err := os.Chmod(filepath.Join(h.root, "scripts", "e2e", "export_validation_bundle.py"), 0o755); err != nil {
		panic(err)
	}
	h.writeFile("scripts/e2e/validation_bundle_continuation_scorecard.go", runAllScorecardStub)
	h.writeFile("scripts/e2e/validation_bundle_continuation_policy_gate.go", runAllGateSuccessStub)
}

func (h runAllHarness) writeFile(relPath, body string) {
	target := filepath.Join(h.root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		panic(err)
	}
	if err := os.WriteFile(target, []byte(strings.TrimLeft(body, "\n")), 0o644); err != nil {
		panic(err)
	}
}

func (h runAllHarness) run(env map[string]string) runAllResult {
	cmd := exec.Command(filepath.Join(h.root, "scripts", "e2e", "run_all.sh"))
	cmd.Dir = h.root
	cmd.Env = os.Environ()
	for key, value := range env {
		cmd.Env = append(cmd.Env, key+"="+value)
	}
	stdout, err := cmd.Output()
	if err == nil {
		return runAllResult{Stdout: string(stdout)}
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return runAllResult{Stdout: string(stdout), Stderr: string(exitErr.Stderr), Err: err}
	}
	return runAllResult{Stdout: string(stdout), Err: err}
}

func (h runAllHarness) findGatePath() string {
	var match string
	_ = filepath.Walk(h.root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if filepath.Base(path) == "validation-bundle-continuation-policy-gate.json" {
			match = path
			return filepath.SkipAll
		}
		return nil
	})
	if match == "" {
		panic("expected continuation gate artifact to be generated")
	}
	return match
}

func repoRootForRunAllTest(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

func readJSONLines(t *testing.T, path string, target *[]map[string]any) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	items := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var item map[string]any
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			t.Fatalf("Unmarshal JSONL line: %v", err)
		}
		items = append(items, item)
	}
	*target = items
}

func readJSONFileForRunAll(t *testing.T, path string, target any) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("Unmarshal(%s): %v", path, err)
	}
}

func asBoolForRunAll(value any) bool {
	cast, ok := value.(bool)
	return ok && cast
}

const runAllTaskSmokeStub = `
#!/usr/bin/env python3
import json
import pathlib
import sys

args = sys.argv[1:]
report_path = pathlib.Path(args[args.index('--report-path') + 1])
report_path.parent.mkdir(parents=True, exist_ok=True)
report_path.write_text(json.dumps({'status': 'succeeded', 'all_ok': True}), encoding='utf-8')
`

const runAllBrokerBootstrapStub = `
package main

import (
    "flag"
    "os"
)

func main() {
    output := flag.String("output", "", "output")
    flag.Parse()
    if err := os.WriteFile(*output, []byte("{\"ready\":false,\"runtime_posture\":\"contract_only\",\"live_adapter_implemented\":false}\n"), 0o644); err != nil {
        panic(err)
    }
}
`

const runAllExportBundleStub = `
#!/usr/bin/env python3
import json
import pathlib
import sys

args = sys.argv[1:]
root = pathlib.Path(args[args.index('--go-root') + 1])
bundle_dir = root / args[args.index('--bundle-dir') + 1]
bundle_dir.mkdir(parents=True, exist_ok=True)
calls_path = root / 'calls.jsonl'
gate_path = next(root.rglob('validation-bundle-continuation-policy-gate.json'), None)
payload = {
    'gate_exists': gate_path is not None,
    'run_broker': args[args.index('--run-broker') + 1],
    'broker_backend': args[args.index('--broker-backend') + 1],
    'broker_report_path': args[args.index('--broker-report-path') + 1],
    'broker_bootstrap_summary_path': args[args.index('--broker-bootstrap-summary-path') + 1],
}
with calls_path.open('a', encoding='utf-8') as handle:
    handle.write(json.dumps(payload) + '\n')
`

const runAllScorecardStub = `
package main

import (
    "encoding/json"
    "flag"
    "os"
    "path/filepath"
)

func main() {
    output := flag.String("output", "", "output")
    flag.Parse()
    if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
        panic(err)
    }
    body, err := json.Marshal(map[string]any{
        "summary": map[string]any{},
        "shared_queue_companion": map[string]any{"available": true},
    })
    if err != nil {
        panic(err)
    }
    if err := os.WriteFile(*output, body, 0o644); err != nil {
        panic(err)
    }
}
`

const runAllGateSuccessStub = `
package main

import (
    "encoding/json"
    "flag"
    "os"
    "path/filepath"
)

func main() {
    _ = flag.String("scorecard", "", "scorecard")
    mode := flag.String("enforcement-mode", "", "mode")
    output := flag.String("output", "", "output")
    flag.Parse()
    if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
        panic(err)
    }
    body, err := json.Marshal(map[string]any{
        "status": "policy-go",
        "recommendation": "go",
        "enforcement": map[string]any{
            "mode": *mode,
            "outcome": "pass",
            "exit_code": 0,
        },
    })
    if err != nil {
        panic(err)
    }
    if err := os.WriteFile(*output, body, 0o644); err != nil {
        panic(err)
    }
}
`

const runAllGateModeOnlyStub = `
package main

import (
    "encoding/json"
    "flag"
    "os"
    "path/filepath"
)

func main() {
    _ = flag.String("scorecard", "", "scorecard")
    mode := flag.String("enforcement-mode", "", "mode")
    output := flag.String("output", "", "output")
    flag.Parse()
    if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
        panic(err)
    }
    body, err := json.Marshal(map[string]any{
        "enforcement": map[string]any{"mode": *mode},
    })
    if err != nil {
        panic(err)
    }
    if err := os.WriteFile(*output, body, 0o644); err != nil {
        panic(err)
    }
}
`
