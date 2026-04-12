package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1607RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1607GoFirstMaintenanceSurfacesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	residualDirs := []string{
		"docs",
		"reports",
		"bigclaw-go/docs/reports",
		"bigclaw-go/internal/migration",
		"bigclaw-go/internal/planning",
	}

	for _, relativeDir := range residualDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected Go-first maintenance surface to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO1607ReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"docs/issue-plan.md",
		"docs/local-tracker-automation.md",
		"bigclaw-go/internal/planning/planning.go",
		"bigclaw-go/internal/planning/planning_test.go",
		"bigclaw-go/internal/migration/legacy_test_contract_sweep_b.go",
		"bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go",
		"bigclaw-go/internal/migration/legacy_test_contract_sweep_x.go",
		"bigclaw-go/docs/reports/parallel-follow-up-index.md",
		"bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json",
		"bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json",
		"bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/static maintenance replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1607LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1607-go-first-maintenance-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1607",
		"Repository-wide Python file count before lane changes: `0`.",
		"Repository-wide Python file count after lane changes: `0`.",
		"Explicit remaining Python asset list: none.",
		"`docs`: `0` Python files",
		"`reports`: `0` Python files",
		"`bigclaw-go/docs/reports`: `0` Python files",
		"`bigclaw-go/internal/migration`: `0` Python files",
		"`bigclaw-go/internal/planning`: `0` Python files",
		"`docs/issue-plan.md`",
		"`docs/local-tracker-automation.md`",
		"`bigclaw-go/internal/planning/planning.go`",
		"`bigclaw-go/internal/migration/legacy_test_contract_sweep_d.go`",
		"`bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json`",
		"`bigclaw-go/docs/reports/live-shadow-mirror-scorecard.json`",
		"`find . -path '*/.git' -prune -o -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \\) -print | sort`",
		"`find docs reports bigclaw-go/docs/reports bigclaw-go/internal/migration bigclaw-go/internal/planning -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \\) 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1607(RepositoryHasNoPythonFiles|GoFirstMaintenanceSurfacesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
		"Residual risk: this checkout already started with zero physical Python files, so BIG-GO-1607 hardens the Go-first maintenance baseline rather than lowering the numeric file count further.",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
