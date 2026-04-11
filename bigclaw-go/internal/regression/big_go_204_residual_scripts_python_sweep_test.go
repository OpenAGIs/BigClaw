package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO204RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO204PriorityResidualDirectoriesStayPythonFree(t *testing.T) {
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

func TestBIGGO204ActiveDocsStayGoOnly(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"docs/symphony-repo-bootstrap-template.md",
		"docs/go-mainline-cutover-handoff.md",
		"scripts/ops/bigclawctl",
		"bigclaw-go/cmd/bigclawctl/main.go",
	}
	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go or shell replacement path to exist: %s (%v)", relativePath, err)
		}
	}

	bootstrapTemplate := readRepoFile(t, rootRepo, "docs/symphony-repo-bootstrap-template.md")
	for _, needle := range []string{
		"`scripts/ops/bigclawctl`",
		"`bash scripts/ops/bigclawctl workspace bootstrap`",
		"`bash scripts/ops/bigclawctl workspace cleanup`",
		"Go or shell native",
	} {
		if !strings.Contains(bootstrapTemplate, needle) {
			t.Fatalf("bootstrap template missing Go-only guidance %q", needle)
		}
	}
	for _, forbidden := range []string{
		"workspace_bootstrap.py",
		"workspace_bootstrap_cli.py",
	} {
		if strings.Contains(bootstrapTemplate, forbidden) {
			t.Fatalf("bootstrap template should not reference retired Python helper %q", forbidden)
		}
	}

	cutoverHandoff := readRepoFile(t, rootRepo, "docs/go-mainline-cutover-handoff.md")
	for _, needle := range []string{
		"`find . -path '*/.git' -prune -o \\( -name '*.py' -o -name '*.pyw' \\) -type f -print | sort`",
		"`bash scripts/ops/bigclawctl workspace validate --help`",
		"Python-based",
	} {
		if !strings.Contains(cutoverHandoff, needle) {
			t.Fatalf("cutover handoff missing Go-only validation evidence %q", needle)
		}
	}
	if strings.Contains(cutoverHandoff, "PYTHONPATH=src python3") {
		t.Fatal("cutover handoff should not require python3 legacy shim assertions")
	}
}

func TestBIGGO204LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-204-residual-scripts-python-sweep-p.md")

	for _, needle := range []string{
		"BIG-GO-204",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw`: `0` Python files",
		"`tests`: `0` Python files",
		"`scripts`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"`docs/symphony-repo-bootstrap-template.md` now requires only `scripts/ops/bigclawctl` plus workflow hooks that invoke `workspace bootstrap` and `workspace cleanup`",
		"`docs/go-mainline-cutover-handoff.md` now records zero-Python inventory plus `bash scripts/ops/bigclawctl workspace validate --help` instead of a `python3` shim assertion",
		"`find . -path '*/.git' -prune -o \\( -name '*.py' -o -name '*.pyw' \\) -type f -print | sort`",
		"`rg -n --glob 'scripts/**' --glob 'bigclaw-go/scripts/**' \"python3|python |\\\\.py\\\\b|#!/usr/bin/env python|#!/usr/bin/python\" /Users/openagi/code/bigclaw-workspaces/BIG-GO-204`",
		"`cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-204/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO204",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
