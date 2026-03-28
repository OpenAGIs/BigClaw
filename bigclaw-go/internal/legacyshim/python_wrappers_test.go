package legacyshim

import (
	"encoding/json"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"bigclaw-go/internal/testharness"
)

func TestLegacyPythonShimHelperFunctions(t *testing.T) {
	cmd := testharness.PythonCommand(t, "-c", `
import json
from pathlib import Path

from bigclaw.legacy_shim import (
    LEGACY_PYTHON_WRAPPER_NOTICE,
    append_missing_flag,
    build_github_sync_args,
    build_refill_args,
    build_workspace_bootstrap_args,
    build_workspace_runtime_bootstrap_args,
    build_workspace_validate_args,
    repo_root_from_script,
    translate_workspace_validate_args,
)

repo_root = Path("/repo")
payload = {
    "append_repo_url": append_missing_flag(
        ["--repo-url", "ssh://example/repo.git"],
        "--repo-url",
        "git@github.com:OpenAGIs/BigClaw.git",
    ),
    "append_cache_key": append_missing_flag(
        ["--cache-key=openagis-bigclaw"],
        "--cache-key",
        "other",
    ),
    "workspace_bootstrap": build_workspace_bootstrap_args(repo_root, ["--workspace", "/tmp/ws"]),
    "workspace_validate_translate": translate_workspace_validate_args([
        "--repo-url", "git@github.com:OpenAGIs/BigClaw.git",
        "--workspace-root", "/tmp/ws",
        "--issues", "BIG-1", "BIG-2",
        "--report-file", "/tmp/report.md",
        "--no-cleanup",
        "--json",
    ]),
    "workspace_validate": build_workspace_validate_args(repo_root, ["--issues", "BIG-1", "BIG-2"]),
    "github_sync": build_github_sync_args(repo_root, ["status", "--json"]),
    "refill": build_refill_args(repo_root, ["--apply"]),
    "runtime_bootstrap": build_workspace_runtime_bootstrap_args(repo_root, ["bootstrap", "--json"]),
    "repo_root_from_script": str(repo_root_from_script("/repo/scripts/ops/bigclaw_refill_queue.py")),
    "notice": LEGACY_PYTHON_WRAPPER_NOTICE,
}
print(json.dumps(payload, sort_keys=True))
`)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("legacy shim helper python snippet failed: %v (%s)", err, string(output))
	}

	var payload map[string]any
	if err := json.Unmarshal(output, &payload); err != nil {
		t.Fatalf("decode legacy shim helper payload: %v (%s)", err, string(output))
	}

	if got := toStringSlice(t, payload["append_repo_url"]); !reflect.DeepEqual(got, []string{"--repo-url", "ssh://example/repo.git"}) {
		t.Fatalf("unexpected append_repo_url: %v", got)
	}
	if got := toStringSlice(t, payload["append_cache_key"]); !reflect.DeepEqual(got, []string{"--cache-key=openagis-bigclaw"}) {
		t.Fatalf("unexpected append_cache_key: %v", got)
	}
	if got := toStringSlice(t, payload["workspace_bootstrap"]); !reflect.DeepEqual(got, []string{
		"bash", "/repo/scripts/ops/bigclawctl", "workspace", "bootstrap",
		"--workspace", "/tmp/ws",
		"--repo-url", "git@github.com:OpenAGIs/BigClaw.git",
		"--cache-key", "openagis-bigclaw",
	}) {
		t.Fatalf("unexpected workspace_bootstrap args: %v", got)
	}
	if got := toStringSlice(t, payload["workspace_validate_translate"]); !reflect.DeepEqual(got, []string{
		"--repo-url", "git@github.com:OpenAGIs/BigClaw.git",
		"--workspace-root", "/tmp/ws",
		"--issues", "BIG-1,BIG-2",
		"--report", "/tmp/report.md",
		"--cleanup=false",
		"--json",
	}) {
		t.Fatalf("unexpected workspace_validate_translate args: %v", got)
	}
	if got := toStringSlice(t, payload["workspace_validate"]); !reflect.DeepEqual(got, []string{
		"bash", "/repo/scripts/ops/bigclawctl", "workspace", "validate",
		"--issues", "BIG-1,BIG-2",
	}) {
		t.Fatalf("unexpected workspace_validate args: %v", got)
	}
	if got := toStringSlice(t, payload["github_sync"]); !reflect.DeepEqual(got, []string{
		"bash", "/repo/scripts/ops/bigclawctl", "github-sync", "status", "--json",
	}) {
		t.Fatalf("unexpected github_sync args: %v", got)
	}
	if got := toStringSlice(t, payload["refill"]); !reflect.DeepEqual(got, []string{
		"bash", "/repo/scripts/ops/bigclawctl", "refill", "--apply",
	}) {
		t.Fatalf("unexpected refill args: %v", got)
	}
	if got := toStringSlice(t, payload["runtime_bootstrap"]); !reflect.DeepEqual(got, []string{
		"bash", "/repo/scripts/ops/bigclawctl", "workspace", "bootstrap", "--json",
	}) {
		t.Fatalf("unexpected runtime_bootstrap args: %v", got)
	}
	if got, ok := payload["repo_root_from_script"].(string); !ok || got != "/repo" {
		t.Fatalf("unexpected repo_root_from_script: %+v", payload["repo_root_from_script"])
	}
	if got, ok := payload["notice"].(string); !ok || !strings.Contains(got, "compatibility shim during migration") {
		t.Fatalf("unexpected notice: %+v", payload["notice"])
	}
}

func TestLegacyPythonWrapperScriptsRunWithoutPythonPath(t *testing.T) {
	pythonBin := testharness.PythonExecutable(t)
	projectRoot := testharness.ProjectRoot(t)
	cases := []struct {
		name       string
		script     string
		args       []string
		wantOutput string
	}{
		{
			name:       "dev-smoke",
			script:     "scripts/dev_smoke.py",
			wantOutput: "smoke_ok local",
		},
		{
			name:       "refill-help",
			script:     "scripts/ops/bigclaw_refill_queue.py",
			args:       []string{"--help"},
			wantOutput: "usage: bigclawctl refill [flags]",
		},
		{
			name:       "create-issues-help",
			script:     "scripts/create_issues.py",
			args:       []string{"--help"},
			wantOutput: "usage: bigclawctl create-issues [flags]",
		},
		{
			name:       "github-sync-help",
			script:     "scripts/ops/bigclaw_github_sync.py",
			args:       []string{"--help"},
			wantOutput: "usage: bigclawctl github-sync <install|status|sync> [flags]",
		},
		{
			name:       "workspace-bootstrap-help",
			script:     "scripts/ops/bigclaw_workspace_bootstrap.py",
			args:       []string{"--help"},
			wantOutput: "usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]",
		},
		{
			name:       "symphony-workspace-bootstrap-help",
			script:     "scripts/ops/symphony_workspace_bootstrap.py",
			args:       []string{"--help"},
			wantOutput: "usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]",
		},
		{
			name:       "symphony-workspace-validate-help",
			script:     "scripts/ops/symphony_workspace_validate.py",
			args:       []string{"--help"},
			wantOutput: "usage: bigclawctl workspace validate [flags]",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{tc.script}, tc.args...)
			cmd := exec.Command(pythonBin, args...)
			cmd.Dir = projectRoot
			cmd.Env = envWithoutPythonPath(os.Environ())
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("run %s without PYTHONPATH: %v (%s)", tc.script, err, string(output))
			}
			if !strings.Contains(string(output), tc.wantOutput) {
				t.Fatalf("expected %q in output, got %q", tc.wantOutput, string(output))
			}
		})
	}
}

func envWithoutPythonPath(env []string) []string {
	filtered := make([]string, 0, len(env))
	for _, item := range env {
		if strings.HasPrefix(item, "PYTHONPATH=") {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func toStringSlice(t *testing.T, raw any) []string {
	t.Helper()
	items, ok := raw.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", raw)
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		text, ok := item.(string)
		if !ok {
			t.Fatalf("expected string item, got %T", item)
		}
		out = append(out, text)
	}
	return out
}
