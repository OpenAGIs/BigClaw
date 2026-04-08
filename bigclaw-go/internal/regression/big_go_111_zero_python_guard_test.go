package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO111RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO111SrcBigclawResidualAreaStaysAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	if _, err := os.Stat(filepath.Join(rootRepo, "src", "bigclaw")); !os.IsNotExist(err) {
		t.Fatalf("expected src/bigclaw residual surface to stay absent")
	}
}

func TestBIGGO111GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/consoleia/consoleia.go",
		"bigclaw-go/internal/issuearchive/archive.go",
		"bigclaw-go/internal/queue/queue.go",
		"bigclaw-go/internal/risk/risk.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/internal/planning/planning.go",
		"scripts/ops/bigclawctl",
		"docs/go-mainline-cutover-handoff.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO111LaneReportCapturesSweepLedger(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-111-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-111",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused `src/bigclaw` physical Python file count before lane changes: `0`",
		"Focused `src/bigclaw` physical Python file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Focused sweep-H ledger for `src/bigclaw`: `[]`",
		"`src/bigclaw`: directory not present, so residual Python files = `0`",
		"`bigclaw-go/internal/consoleia/consoleia.go`",
		"`bigclaw-go/internal/issuearchive/archive.go`",
		"`bigclaw-go/internal/queue/queue.go`",
		"`bigclaw-go/internal/risk/risk.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`bigclaw-go/internal/planning/planning.go`",
		"`scripts/ops/bigclawctl`",
		"`docs/go-mainline-cutover-handoff.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw bigclaw-go/internal/consoleia bigclaw-go/internal/issuearchive bigclaw-go/internal/queue bigclaw-go/internal/risk bigclaw-go/internal/bootstrap bigclaw-go/internal/planning -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO111",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
