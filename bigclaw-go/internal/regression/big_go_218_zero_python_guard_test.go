package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO218RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO218ActiveBootstrapDocsStayGoOnly(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	template := readRepoFile(t, rootRepo, "docs/symphony-repo-bootstrap-template.md")
	templateRequired := []string{
		"`scripts/ops/bigclawctl`",
		"`bigclawctl workspace bootstrap`",
		"`bigclawctl workspace cleanup`",
		"`bigclawctl workspace validate`",
	}
	for _, needle := range templateRequired {
		if !strings.Contains(template, needle) {
			t.Fatalf("bootstrap template missing Go/native workspace guidance %q", needle)
		}
	}
	for _, needle := range []string{
		"`src/<your_package>/workspace_bootstrap.py`",
		"`src/<your_package>/workspace_bootstrap_cli.py`",
		"Python compatibility package path",
	} {
		if strings.Contains(template, needle) {
			t.Fatalf("bootstrap template should not reference Python bootstrap guidance %q", needle)
		}
	}

	handoff := readRepoFile(t, rootRepo, "docs/go-mainline-cutover-handoff.md")
	if !strings.Contains(handoff, "`bash scripts/ops/bigclawctl workspace validate --help >/dev/null`") {
		t.Fatalf("cutover handoff missing Go validation evidence")
	}
	for _, needle := range []string{
		"`PYTHONPATH=src python3 - <<\"... legacy shim assertions ...\"`",
		"PYTHONPATH=src python3",
	} {
		if strings.Contains(handoff, needle) {
			t.Fatalf("cutover handoff should not retain Python validation evidence %q", needle)
		}
	}
}

func TestBIGGO218ReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"scripts/ops/bigclawctl",
		"scripts/dev_bootstrap.sh",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"docs/symphony-repo-bootstrap-template.md",
		"docs/go-mainline-cutover-handoff.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO218LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-218-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-218",
		"Repository-wide Python file count: `0`.",
		"`docs/symphony-repo-bootstrap-template.md`",
		"`docs/go-mainline-cutover-handoff.md`",
		"`scripts/ops/bigclawctl`",
		"`scripts/dev_bootstrap.sh`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`bash scripts/ops/bigclawctl workspace validate --help >/dev/null`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`rg -n \"workspace_bootstrap\\\\.py|workspace_bootstrap_cli\\\\.py|PYTHONPATH=src python3\" docs/symphony-repo-bootstrap-template.md docs/go-mainline-cutover-handoff.md`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO218(RepositoryHasNoPythonFiles|ActiveBootstrapDocsStayGoOnly|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
