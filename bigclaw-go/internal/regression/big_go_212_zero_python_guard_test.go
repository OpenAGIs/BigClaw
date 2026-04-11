package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO212RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO212ResidualTestReplacementDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	auditedDirs := []string{
		"bigclaw-go/internal/billing",
		"bigclaw-go/internal/config",
		"bigclaw-go/internal/executor",
		"bigclaw-go/internal/flow",
		"bigclaw-go/internal/prd",
		"bigclaw-go/internal/reporting",
		"bigclaw-go/internal/reportstudio",
		"bigclaw-go/internal/service",
	}

	for _, relativeDir := range auditedDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected residual test replacement directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO212RepresentativeReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"reports/BIG-GO-948-validation.md",
		"bigclaw-go/internal/billing/billing_test.go",
		"bigclaw-go/internal/billing/statement_test.go",
		"bigclaw-go/internal/config/config_test.go",
		"bigclaw-go/internal/executor/executor.go",
		"bigclaw-go/internal/executor/kubernetes_test.go",
		"bigclaw-go/internal/executor/ray_test.go",
		"bigclaw-go/internal/flow/flow.go",
		"bigclaw-go/internal/prd/intake.go",
		"bigclaw-go/internal/reporting/reporting_test.go",
		"bigclaw-go/internal/reportstudio/reportstudio_test.go",
		"bigclaw-go/internal/service/server.go",
		"bigclaw-go/internal/service/server_test.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected representative Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO212LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-212-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-212",
		"Repository-wide Python file count: `0`.",
		"`bigclaw-go/internal/billing`: `0` Python files",
		"`bigclaw-go/internal/config`: `0` Python files",
		"`bigclaw-go/internal/executor`: `0` Python files",
		"`bigclaw-go/internal/flow`: `0` Python files",
		"`bigclaw-go/internal/prd`: `0` Python files",
		"`bigclaw-go/internal/reporting`: `0` Python files",
		"`bigclaw-go/internal/reportstudio`: `0` Python files",
		"`bigclaw-go/internal/service`: `0` Python files",
		"`reports/BIG-GO-948-validation.md`",
		"`bigclaw-go/internal/billing/billing_test.go`",
		"`bigclaw-go/internal/billing/statement_test.go`",
		"`bigclaw-go/internal/config/config_test.go`",
		"`bigclaw-go/internal/executor/executor.go`",
		"`bigclaw-go/internal/executor/kubernetes_test.go`",
		"`bigclaw-go/internal/executor/ray_test.go`",
		"`bigclaw-go/internal/flow/flow.go`",
		"`bigclaw-go/internal/prd/intake.go`",
		"`bigclaw-go/internal/reporting/reporting_test.go`",
		"`bigclaw-go/internal/reportstudio/reportstudio_test.go`",
		"`bigclaw-go/internal/service/server.go`",
		"`bigclaw-go/internal/service/server_test.go`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find bigclaw-go/internal/billing bigclaw-go/internal/config bigclaw-go/internal/executor bigclaw-go/internal/flow bigclaw-go/internal/prd bigclaw-go/internal/reporting bigclaw-go/internal/reportstudio bigclaw-go/internal/service -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO212(RepositoryHasNoPythonFiles|ResidualTestReplacementDirectoriesStayPythonFree|RepresentativeReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
