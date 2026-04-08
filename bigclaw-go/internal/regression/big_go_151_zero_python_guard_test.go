package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO151RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO151WorkflowDefinitionTrancheLStaysAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"src/bigclaw/models.py",
		"src/bigclaw/connectors.py",
		"src/bigclaw/mapping.py",
		"src/bigclaw/dsl.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired tranche-L Python path to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO151GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/domain/task.go",
		"bigclaw-go/internal/domain/priority.go",
		"bigclaw-go/internal/intake/connector.go",
		"bigclaw-go/internal/intake/mapping.go",
		"bigclaw-go/internal/workflow/definition.go",
		"bigclaw-go/internal/workflow/model.go",
		"bigclaw-go/internal/prd/intake.go",
		"docs/go-mainline-cutover-issue-pack.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO151LaneReportCapturesReplacementEvidence(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-151-src-bigclaw-tranche-l.md")

	for _, needle := range []string{
		"BIG-GO-151",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused `src/bigclaw` tranche-L physical Python file count before lane changes: `0`",
		"Focused `src/bigclaw` tranche-L physical Python file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Focused tranche-L ledger: `[]`",
		"`src/bigclaw`: directory not present, so residual Python files = `0`",
		"`src/bigclaw/models.py`",
		"`src/bigclaw/connectors.py`",
		"`src/bigclaw/mapping.py`",
		"`src/bigclaw/dsl.py`",
		"`bigclaw-go/internal/domain/task.go`",
		"`bigclaw-go/internal/domain/priority.go`",
		"`bigclaw-go/internal/intake/connector.go`",
		"`bigclaw-go/internal/intake/mapping.go`",
		"`bigclaw-go/internal/workflow/definition.go`",
		"`bigclaw-go/internal/workflow/model.go`",
		"`bigclaw-go/internal/prd/intake.go`",
		"`docs/go-mainline-cutover-issue-pack.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src/bigclaw -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO151",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
