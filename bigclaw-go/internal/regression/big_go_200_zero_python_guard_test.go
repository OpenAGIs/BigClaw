package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO200RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO200CommandAndReportIndexSurfacesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	surfaces := []string{
		"bigclaw-go/cmd",
		"scripts/ops",
		"bigclaw-go/docs/reports",
	}

	for _, relativeDir := range surfaces {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected command/report-index surface to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO200GoNativeEntryPointsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	goNativePaths := []string{
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawctl/automation_commands.go",
		"bigclaw-go/cmd/bigclawctl/migration_commands.go",
		"bigclaw-go/cmd/bigclawd/main.go",
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
		"bigclaw-go/docs/reports/issue-coverage.md",
		"bigclaw-go/docs/reports/parallel-follow-up-index.md",
		"bigclaw-go/docs/reports/parallel-validation-matrix.md",
		"bigclaw-go/docs/reports/review-readiness.md",
		"bigclaw-go/docs/reports/linear-project-sync-summary.md",
		"bigclaw-go/docs/reports/epic-closure-readiness-report.md",
	}

	for _, relativePath := range goNativePaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native entrypoint or report index to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO200LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-200-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-200",
		"Repository-wide Python file count: `0`.",
		"`bigclaw-go/cmd`: `0` Python files",
		"`scripts/ops`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_commands.go`",
		"`bigclaw-go/cmd/bigclawctl/migration_commands.go`",
		"`bigclaw-go/cmd/bigclawd/main.go`",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`bigclaw-go/docs/reports/issue-coverage.md`",
		"`bigclaw-go/docs/reports/parallel-follow-up-index.md`",
		"`bigclaw-go/docs/reports/parallel-validation-matrix.md`",
		"`bigclaw-go/docs/reports/review-readiness.md`",
		"`bigclaw-go/docs/reports/linear-project-sync-summary.md`",
		"`bigclaw-go/docs/reports/epic-closure-readiness-report.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find bigclaw-go/cmd scripts/ops bigclaw-go/docs/reports -maxdepth 2 -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO200(RepositoryHasNoPythonFiles|CommandAndReportIndexSurfacesStayPythonFree|GoNativeEntryPointsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
