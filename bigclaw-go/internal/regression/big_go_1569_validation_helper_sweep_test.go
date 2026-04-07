package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1569RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1569ValidationHelperPathsStayDeleted(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	deletedPaths := []string{
		"scripts/ops/symphony_workspace_validate.py",
	}

	for _, relativePath := range deletedPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected retired validation helper to remain absent: %s", relativePath)
		}
	}
}

func TestBIGGO1569ValidationHelperGoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"scripts/ops/bigclawctl",
		"bigclaw-go/cmd/bigclawctl/main.go",
		"docs/go-cli-script-migration-plan.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}

	contents := readRepoFile(t, rootRepo, "docs/go-cli-script-migration-plan.md")
	required := []string{
		"retired `scripts/ops/symphony_workspace_validate.py`; use `bash scripts/ops/bigclawctl workspace validate`",
		"`bash scripts/ops/bigclawctl workspace validate --help`",
	}
	for _, needle := range required {
		if !strings.Contains(contents, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing validation helper replacement guidance %q", needle)
		}
	}
}

func TestBIGGO1569LaneReportCapturesExactLedger(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1569-validation-helper-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1569",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused `scripts`, `scripts/ops`, and `bigclaw-go/cmd/bigclawctl` physical",
		"Python file count before lane changes: `0`",
		"Python file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Focused validation-helper ledger: `[]`",
		"`scripts/ops/symphony_workspace_validate.py`",
		"`bash scripts/ops/bigclawctl workspace validate`",
		"`scripts/ops/bigclawctl`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`docs/go-cli-script-migration-plan.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find scripts scripts/ops bigclaw-go/cmd/bigclawctl -type f -name '*.py' 2>/dev/null | sort`",
		"`bash scripts/ops/bigclawctl workspace validate --help`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1569",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
