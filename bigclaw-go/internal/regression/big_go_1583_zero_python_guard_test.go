package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1583RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1583RootTestsDirectoryStaysAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	if _, err := os.Stat(filepath.Join(rootRepo, "tests")); !os.IsNotExist(err) {
		t.Fatalf("expected retired root tests directory to stay absent: %v", err)
	}
}

func TestBIGGO1583BucketATestPathsStayAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"tests/conftest.py",
		"tests/test_audit_events.py",
		"tests/test_connectors.py",
		"tests/test_console_ia.py",
		"tests/test_control_center.py",
		"tests/test_cost_control.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired bucket-A Python test path to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO1583GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/regression/regression.go",
		"bigclaw-go/internal/regression/regression_test.go",
		"bigclaw-go/internal/observability/audit_test.go",
		"bigclaw-go/internal/intake/connector_test.go",
		"bigclaw-go/internal/consoleia/consoleia_test.go",
		"bigclaw-go/internal/control/controller.go",
		"bigclaw-go/internal/control/controller_test.go",
		"bigclaw-go/internal/costcontrol/controller.go",
		"bigclaw-go/internal/costcontrol/controller_test.go",
		"bigclaw-go/internal/api/server.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1583LaneReportCapturesBucketASweep(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1583-tests-bucket-a-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1583",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused `tests/*.py` bucket-A physical Python file count before lane changes: `0`",
		"Focused `tests/*.py` bucket-A physical Python file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Focused bucket-A ledger: `[]`",
		"`tests`: directory not present, so residual Python files = `0`",
		"`tests/conftest.py`",
		"`tests/test_audit_events.py`",
		"`tests/test_connectors.py`",
		"`tests/test_console_ia.py`",
		"`tests/test_control_center.py`",
		"`tests/test_cost_control.py`",
		"`bigclaw-go/internal/regression/regression.go`",
		"`bigclaw-go/internal/regression/regression_test.go`",
		"`bigclaw-go/internal/observability/audit_test.go`",
		"`bigclaw-go/internal/intake/connector_test.go`",
		"`bigclaw-go/internal/consoleia/consoleia_test.go`",
		"`bigclaw-go/internal/control/controller.go`",
		"`bigclaw-go/internal/control/controller_test.go`",
		"`bigclaw-go/internal/costcontrol/controller.go`",
		"`bigclaw-go/internal/costcontrol/controller_test.go`",
		"`bigclaw-go/internal/api/server.go`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find tests -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1583",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
