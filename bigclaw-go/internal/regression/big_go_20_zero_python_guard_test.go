package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO20RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO20LiveDocsRemainGoOnly(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	requiredDocs := map[string][]string{
		"docs/symphony-repo-bootstrap-template.md": {
			"`scripts/ops/bigclawctl`",
			"`bigclawctl workspace bootstrap`",
			"`bigclawctl workspace validate`",
			"Go-native workspace lifecycle",
		},
		"docs/go-mainline-cutover-handoff.md": {
			"`cd bigclaw-go && go test ./...`",
			"`cd bigclaw-go && go test ./internal/domain ./internal/intake ./internal/workflow ./internal/risk ./internal/triage ./internal/billing`",
			"`cd bigclaw-go && go test ./internal/regression -run 'TestBIGGO19",
			"default mainline posture is Go-first",
		},
	}

	disallowedDocs := map[string][]string{
		"docs/symphony-repo-bootstrap-template.md": {
			"workspace_bootstrap.py",
			"workspace_bootstrap_cli.py",
			"Python compatibility package path is repo-specific",
		},
		"docs/go-mainline-cutover-handoff.md": {
			"PYTHONPATH=src python3",
			"legacy shim assertions",
		},
	}

	for relativePath, needles := range requiredDocs {
		contents := readRepoFile(t, rootRepo, relativePath)
		for _, needle := range needles {
			if !strings.Contains(contents, needle) {
				t.Fatalf("%s missing required Go-only guidance %q", relativePath, needle)
			}
		}
		for _, needle := range disallowedDocs[relativePath] {
			if strings.Contains(contents, needle) {
				t.Fatalf("%s should not include retired Python guidance %q", relativePath, needle)
			}
		}
	}
}

func TestBIGGO20ReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"scripts/ops/bigclawctl",
		"scripts/dev_bootstrap.sh",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/internal/regression/big_go_19_zero_python_guard_test.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO20LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-20-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-20",
		"Repository-wide Python file count: `0`.",
		"`docs/symphony-repo-bootstrap-template.md`",
		"`docs/go-mainline-cutover-handoff.md`",
		"`scripts/ops/bigclawctl`",
		"`scripts/dev_bootstrap.sh`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`bigclaw-go/internal/regression/big_go_19_zero_python_guard_test.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO20",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
