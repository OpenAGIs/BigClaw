package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO122ResidualPythonHeavyTestDirectoriesStayAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	targetDirs := []string{
		"tests",
		"test",
		"testing",
		"fixtures",
		"bigclaw-go/tests",
		"bigclaw-go/test",
		"bigclaw-go/testing",
		"bigclaw-go/fixtures",
	}

	for _, relativeDir := range targetDirs {
		info, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if !os.IsNotExist(err) {
			t.Fatalf("expected residual Python-heavy test directory to stay absent: %s (info=%v err=%v)", relativeDir, info, err)
		}
	}
}

func TestBIGGO122RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO122GoReplacementCoveragePathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacements := []string{
		"bigclaw-go/internal/observability/audit_test.go",
		"bigclaw-go/internal/intake/connector_test.go",
		"bigclaw-go/internal/control/controller_test.go",
		"bigclaw-go/internal/workflow/orchestration_test.go",
		"bigclaw-go/cmd/bigclawctl/automation_commands_test.go",
	}

	for _, relativePath := range replacements {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected repo-native replacement coverage path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO122LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-122-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-122",
		"`tests`",
		"`test`",
		"`testing`",
		"`fixtures`",
		"`bigclaw-go/tests`",
		"`bigclaw-go/test`",
		"`bigclaw-go/testing`",
		"`bigclaw-go/fixtures`",
		"Residual Python-heavy test directory count: `0`.",
		"Repository-wide `*.py` file count: `0`.",
		"`bigclaw-go/internal/observability/audit_test.go`",
		"`bigclaw-go/internal/intake/connector_test.go`",
		"`bigclaw-go/internal/control/controller_test.go`",
		"`bigclaw-go/internal/workflow/orchestration_test.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_commands_test.go`",
		"`find . -type d \\( -name tests -o -name test -o -name testing -o -name fixtures \\) | sort`",
		"`find . -type f -name '*.py' | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO122",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
