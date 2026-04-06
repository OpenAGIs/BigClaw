package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1504ScriptsAndOpsStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	for _, relativeDir := range []string{
		"scripts",
		"scripts/ops",
	} {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected %s to remain Python-free, found %d file(s): %v", relativeDir, len(pythonFiles), pythonFiles)
		}
	}
}

func TestBIGGO1504CompatibilityLaunchersRemainNonPython(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	expectedContents := map[string][]string{
		"scripts/ops/bigclawctl": {
			"#!/usr/bin/env bash",
			"exec go run ./cmd/bigclawctl",
		},
		"scripts/ops/bigclaw-issue": {
			"#!/usr/bin/env bash",
			"exec bash \"$script_dir/bigclawctl\" issue \"$@\"",
		},
		"scripts/ops/bigclaw-panel": {
			"#!/usr/bin/env bash",
			"exec bash \"$script_dir/bigclawctl\" panel \"$@\"",
		},
		"scripts/ops/bigclaw-symphony": {
			"#!/usr/bin/env bash",
			"exec bash \"$script_dir/bigclawctl\" symphony \"$@\"",
		},
	}

	for relativePath, needles := range expectedContents {
		fullPath := filepath.Join(rootRepo, filepath.FromSlash(relativePath))
		if _, err := os.Stat(fullPath); err != nil {
			t.Fatalf("expected compatibility launcher to exist: %s (%v)", relativePath, err)
		}
		contents := readRepoFile(t, rootRepo, filepath.ToSlash(relativePath))
		for _, needle := range needles {
			if !strings.Contains(contents, needle) {
				t.Fatalf("expected %s to contain %q", relativePath, needle)
			}
		}
		if strings.Contains(contents, ".py") || strings.Contains(contents, "python") {
			t.Fatalf("expected compatibility launcher to stay non-Python: %s", relativePath)
		}
	}
}

func TestBIGGO1504LaneReportCapturesRepoReality(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1504-script-wrapper-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1504",
		"`scripts/*.py` before count: `0`",
		"`scripts/ops/*.py` before count: `0`",
		"`scripts/*.py` after count: `0`",
		"`scripts/ops/*.py` after count: `0`",
		"Deleted physical `.py` files: `none`",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`rg --files | rg '^(scripts|scripts/ops)/.*\\.py$' | wc -l`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1504",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
