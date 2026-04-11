package regression

import (
	"strings"
	"testing"
)

func TestBIGGO255RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO255AutomationBundleTestStubStaysShellOnly(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	contents := readRepoFile(t, rootRepo, "bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go")

	disallowed := []string{
		"#!/usr/bin/env python3",
		"import json, pathlib, sys",
		"raise SystemExit",
	}
	for _, needle := range disallowed {
		if strings.Contains(contents, needle) {
			t.Fatalf("automation_e2e_bundle_commands_test.go should not retain Python stub content %q", needle)
		}
	}

	required := []string{
		"#!/bin/sh",
		"find_arg() {",
		"printf '%s\\n' '{\"status\":\"succeeded\",\"all_ok\":true}' > \"$report_path\"",
		"printf 'unexpected go args: %s\\n' \"$*\" >&2",
	}
	for _, needle := range required {
		if !strings.Contains(contents, needle) {
			t.Fatalf("automation_e2e_bundle_commands_test.go missing shell stub content %q", needle)
		}
	}
}

func TestBIGGO255LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-255-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-255",
		"Repository-wide Python file count: `0`.",
		"`bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go` no longer embeds a Python fake `go` helper",
		"`#!/bin/sh`",
		"`find_arg()`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`rg -n '#!/bin/sh|find_arg\\\\(\\\\)|unexpected go args: \\\\$\\\\*' bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go`",
		"`cd bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestRunAllUsesGoBundleCommandsAndDefaultsHoldMode$'`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO255(RepositoryHasNoPythonFiles|AutomationBundleTestStubStaysShellOnly|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
