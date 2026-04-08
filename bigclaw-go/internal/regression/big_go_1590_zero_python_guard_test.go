package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1590RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1590RepoWidePhysicalReductionBucketStaysPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	residualDirs := []string{
		"docs",
		"scripts",
		"tests",
		"bigclaw-go/internal",
		"bigclaw-go/docs/reports",
		"reports",
	}

	for _, relativeDir := range residualDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected repo-wide physical reduction bucket to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO1590GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"scripts/ops/bigclawctl",
		"scripts/dev_bootstrap.sh",
		"docs/go-cli-script-migration-plan.md",
		"docs/symphony-repo-bootstrap-template.md",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/internal/planning/planning.go",
		"bigclaw-go/internal/githubsync/sync.go",
		"bigclaw-go/internal/regression/big_go_1174_zero_python_guard_test.go",
		"bigclaw-go/docs/reports/migration-readiness-report.md",
		"reports/BIG-GO-902-validation.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1590LaneReportCapturesExactLedger(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1590-physical-reduction-bucket.md")

	for _, needle := range []string{
		"BIG-GO-1590",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Repo-wide physical reduction bucket count before lane changes: `0`",
		"Repo-wide physical reduction bucket count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Focused ledger for the repo-wide physical reduction bucket: `[]`",
		"`docs`: `0` Python files",
		"`scripts`: `0` Python files",
		"`tests`: `0` Python files",
		"`bigclaw-go/internal`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files",
		"`reports`: `0` Python files",
		"`scripts/ops/bigclawctl`",
		"`scripts/dev_bootstrap.sh`",
		"`docs/go-cli-script-migration-plan.md`",
		"`docs/symphony-repo-bootstrap-template.md`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`bigclaw-go/internal/planning/planning.go`",
		"`bigclaw-go/internal/githubsync/sync.go`",
		"`bigclaw-go/internal/regression/big_go_1174_zero_python_guard_test.go`",
		"`bigclaw-go/docs/reports/migration-readiness-report.md`",
		"`reports/BIG-GO-902-validation.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find docs scripts tests bigclaw-go/internal bigclaw-go/docs/reports reports -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1590(RepositoryHasNoPythonFiles|RepoWidePhysicalReductionBucketStaysPythonFree|GoReplacementPathsRemainAvailable|LaneReportCapturesExactLedger)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
