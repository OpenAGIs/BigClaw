package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunAllUsesGoAutomationCommands(t *testing.T) {
	root := t.TempDir()
	scriptPath := filepath.Join(root, "scripts", "e2e", "run_all.sh")
	sourcePath := filepath.Join(repoRoot(t), "scripts", "e2e", "run_all.sh")
	body, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source run_all.sh: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scriptPath, body, 0o755); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(root, "scripts", "e2e", "kubernetes_smoke.sh"), "#!/usr/bin/env bash\nexit 0\n")
	writeExecutable(t, filepath.Join(root, "scripts", "e2e", "ray_smoke.sh"), "#!/usr/bin/env bash\nexit 0\n")

	goStub := `#!/usr/bin/env bash
set -euo pipefail
root="$(pwd)"
calls="$root/calls.jsonl"
if [[ "$1" == "run" && "$2" == "$root/scripts/e2e/broker_bootstrap_summary.go" ]]; then
  shift 2
  output=""
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --output) output="$2"; shift 2 ;;
      *) shift ;;
    esac
  done
  mkdir -p "$(dirname "$output")"
  printf '{"ready":false,"runtime_posture":"contract_only","live_adapter_implemented":false}\n' > "$output"
  exit 0
fi
if [[ "$1" == "build" ]]; then
  shift
  output=""
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -o) output="$2"; shift 2 ;;
      ./cmd/bigclawctl) shift ;;
      *) shift ;;
    esac
  done
  cat > "$output" <<'SCRIPT'
#!/usr/bin/env bash
set -euo pipefail
root="$(pwd)"
calls="$root/calls.jsonl"
if [[ "$1" != "automation" || "$2" != "e2e" ]]; then
  echo "unexpected bigclawctl command: $*" >&2
  exit 1
fi
subcommand="$3"
shift 3
python3 - "$root" "$calls" "$subcommand" "$@" <<'PY'
import json
import pathlib
import sys

root = pathlib.Path(sys.argv[1])
calls = pathlib.Path(sys.argv[2])
subcommand = sys.argv[3]
args = sys.argv[4:]

def arg_value(flag, default=""):
    if flag in args:
        return args[args.index(flag) + 1]
    return default

entry = {"subcommand": subcommand, "args": args}
with calls.open("a", encoding="utf-8") as handle:
    handle.write(json.dumps(entry) + "\n")

if subcommand == "run-task-smoke":
    report = root / arg_value("--report-path")
    report.parent.mkdir(parents=True, exist_ok=True)
    report.write_text(json.dumps({"status": {"state": "succeeded"}, "task": {"id": "local-smoke"}}), encoding="utf-8")
elif subcommand == "export-validation-bundle":
    gate = root / "docs/reports/validation-bundle-continuation-policy-gate.json"
    bundle_dir = root / arg_value("--bundle-dir")
    bundle_dir.mkdir(parents=True, exist_ok=True)
    call = {
        "gate_exists": gate.exists(),
        "run_broker": arg_value("--run-broker"),
        "broker_backend": arg_value("--broker-backend"),
        "broker_report_path": arg_value("--broker-report-path"),
        "broker_bootstrap_summary_path": arg_value("--broker-bootstrap-summary-path"),
    }
    export_calls = root / "export_calls.jsonl"
    with export_calls.open("a", encoding="utf-8") as handle:
        handle.write(json.dumps(call) + "\n")
elif subcommand == "validation-bundle-continuation-scorecard":
    output = root / arg_value("--output")
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps({"summary": {}, "shared_queue_companion": {"available": True}}), encoding="utf-8")
elif subcommand == "validation-bundle-continuation-policy-gate":
    output = root / arg_value("--output")
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps({"enforcement": {"mode": arg_value("--enforcement-mode")}, "status": "policy-go", "recommendation": "go"}), encoding="utf-8")
PY
SCRIPT
  chmod +x "$output"
  exit 0
fi
echo "unexpected go command: $*" >&2
exit 1
`
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeExecutable(t, filepath.Join(binDir, "go"), goStub)

	cmd := exec.Command(scriptPath)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"BIGCLAW_E2E_RUN_KUBERNETES=0",
		"BIGCLAW_E2E_RUN_RAY=0",
		"BIGCLAW_E2E_RUN_LOCAL=1",
		"BIGCLAW_E2E_RUN_BROKER=1",
		"BIGCLAW_E2E_BROKER_BACKEND=stub",
		"BIGCLAW_E2E_BROKER_REPORT_PATH=docs/reports/broker-failover-stub-report.json",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run_all.sh failed: %v\n%s", err, string(output))
	}

	exportCallsBody, err := os.ReadFile(filepath.Join(root, "export_calls.jsonl"))
	if err != nil {
		t.Fatalf("read export calls: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(exportCallsBody)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 export calls, got %d: %s", len(lines), string(exportCallsBody))
	}

	var firstCall, secondCall map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &firstCall); err != nil {
		t.Fatalf("decode first export call: %v", err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &secondCall); err != nil {
		t.Fatalf("decode second export call: %v", err)
	}
	if firstCall["gate_exists"] != false || secondCall["gate_exists"] != true {
		t.Fatalf("unexpected gate sequencing: first=%+v second=%+v", firstCall, secondCall)
	}
	if firstCall["run_broker"] != "1" || firstCall["broker_backend"] != "stub" {
		t.Fatalf("unexpected broker args: %+v", firstCall)
	}
}

func writeExecutable(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
