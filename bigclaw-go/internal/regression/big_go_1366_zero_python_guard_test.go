package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1366RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1366PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	priorityDirs := []string{
		"src/bigclaw",
		"tests",
		"scripts",
		"bigclaw-go/scripts",
	}

	for _, relativeDir := range priorityDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected priority residual directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO1366E2EReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/scripts/e2e/run_all.sh",
		"bigclaw-go/scripts/e2e/kubernetes_smoke.sh",
		"bigclaw-go/scripts/e2e/ray_smoke.sh",
		"bigclaw-go/scripts/e2e/broker_bootstrap_summary.go",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go",
	}
	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected native replacement path to exist: %s (%v)", relativePath, err)
		}
	}

	for _, tc := range []struct {
		path       string
		required   []string
		disallowed []string
	}{
		{
			path: "bigclaw-go/scripts/e2e/run_all.sh",
			required: []string{
				`go run "$ROOT/scripts/e2e/broker_bootstrap_summary.go"`,
				`go run "$ROOT/cmd/bigclawctl" automation e2e export-validation-bundle`,
				`go run "$ROOT/cmd/bigclawctl" automation e2e continuation-scorecard`,
				`go run "$ROOT/cmd/bigclawctl" automation e2e continuation-policy-gate`,
			},
			disallowed: []string{".py", "python ", "python3"},
		},
		{
			path: "bigclaw-go/scripts/e2e/kubernetes_smoke.sh",
			required: []string{
				`go run "$ROOT/cmd/bigclawctl" automation e2e run-task-smoke`,
			},
			disallowed: []string{".py", "python ", "python3"},
		},
		{
			path: "bigclaw-go/scripts/e2e/ray_smoke.sh",
			required: []string{
				`go run "$ROOT/cmd/bigclawctl" automation e2e run-task-smoke`,
				`ENTRYPOINT="${BIGCLAW_RAY_SMOKE_ENTRYPOINT:-sh -c 'echo hello from ray'}"`,
			},
			disallowed: []string{".py", "python ", "python3", "python -c"},
		},
		{
			path: "bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go",
			required: []string{
				"#!/usr/bin/env bash",
				"set -euo pipefail",
				"printf '{\"gate_exists\":%s",
			},
			disallowed: []string{"#!/usr/bin/env python3", "import json, pathlib, sys"},
		},
	} {
		content := readRepoFile(t, rootRepo, tc.path)
		for _, needle := range tc.required {
			if !strings.Contains(content, needle) {
				t.Fatalf("%s missing native replacement evidence %q", tc.path, needle)
			}
		}
		for _, needle := range tc.disallowed {
			if strings.Contains(content, needle) {
				t.Fatalf("%s should not contain retired Python helper evidence %q", tc.path, needle)
			}
		}
	}
}

func TestBIGGO1366LaneReportCapturesNativeReplacement(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1366-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1366",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw`: `0` Python files",
		"`tests`: `0` Python files",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"`bigclaw-go/scripts/e2e/run_all.sh` orchestrates the e2e sweep through `go run` invocations of `bigclawctl automation e2e ...` plus the retained shell smoke wrappers",
		"`bigclaw-go/scripts/e2e/kubernetes_smoke.sh` and `bigclaw-go/scripts/e2e/ray_smoke.sh` stay shell-native wrappers around `go run ./cmd/bigclawctl automation e2e run-task-smoke ...`",
		"`bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go` now fakes the `go` binary with a bash stub instead of an embedded `python3` helper",
		"`find . -name '*.py' | wc -l`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1366",
		"`cd bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestAutomationE2EScriptsStayGoOnly|TestAutomationE2EScriptRunAllUsesNativeEntrypoints|TestRunAllUsesGoBundleCommandsAndDefaultsHoldMode$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
