package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO175RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO175ToolingSweepPathsStayShellNative(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	expectedSnippets := map[string][]string{
		"bigclaw-go/cmd/bigclawctl/automation_e2e_mixed_workload_command.go": {
			`Entrypoint:              "sh -c 'echo gpu via ray'"`,
			`Entrypoint:              "sh -c 'echo required ray'"`,
		},
		"bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go": {
			"#!/bin/sh",
			"write_json_file",
		},
		"bigclaw-go/internal/executor/ray_test.go": {
			`Entrypoint: "sh -c 'echo hello from ray'"`,
		},
	}

	for relativePath, snippets := range expectedSnippets {
		absPath := filepath.Join(rootRepo, filepath.FromSlash(relativePath))
		if _, err := os.Stat(absPath); err != nil {
			t.Fatalf("expected tooling sweep path to exist: %s (%v)", relativePath, err)
		}

		contents := readRepoFile(t, rootRepo, relativePath)
		if strings.Contains(contents, "python -c") || strings.Contains(contents, "#!/usr/bin/env python3") || strings.Contains(contents, "python app.py") {
			t.Fatalf("expected tooling sweep path to avoid Python execution: %s", relativePath)
		}
		for _, snippet := range snippets {
			if !strings.Contains(contents, snippet) {
				t.Fatalf("expected tooling sweep path %s to contain %q", relativePath, snippet)
			}
		}
	}
}

func TestBIGGO175LaneReportCapturesToolingSweep(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-175-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-175",
		"Repository-wide Python file count: `0`.",
		"`bigclaw-go/cmd/bigclawctl/automation_e2e_mixed_workload_command.go` now uses shell-native Ray sample entrypoints",
		"`bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go` now stubs `go` with `/bin/sh` instead of Python",
		"`bigclaw-go/internal/executor/ray_test.go` exercises the Ray runner with a shell-native entrypoint fixture",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestRunAllUsesGoBundleCommandsAndDefaultsHoldMode|TestAutomationMixedWorkloadMatrixBuildsReport'`",
		"`cd bigclaw-go && go test -count=1 ./internal/executor -run 'TestRayRunnerExecuteUsesJobsAPI|TestRayRunnerStopsJobOnCancellation'`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO175",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
