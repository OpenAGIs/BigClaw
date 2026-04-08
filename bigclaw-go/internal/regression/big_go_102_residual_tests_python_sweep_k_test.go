package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO102ResidualPythonTestsStayAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredTests := []string{
		"tests/test_cost_control.py",
		"tests/test_mapping.py",
		"tests/test_repo_board.py",
		"tests/test_repo_collaboration.py",
		"tests/test_roadmap.py",
	}
	for _, relativePath := range retiredTests {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected retired Python test to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO102ReplacementSurfacesRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacements := []string{
		"bigclaw-go/internal/costcontrol/controller_test.go",
		"bigclaw-go/internal/intake/mapping_test.go",
		"bigclaw-go/internal/repo/board.go",
		"bigclaw-go/internal/collaboration/thread_test.go",
		"bigclaw-go/internal/regression/roadmap_contract_test.go",
		"bigclaw-go/internal/planning/planning_test.go",
	}
	for _, relativePath := range replacements {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected replacement surface to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO102LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-102-residual-tests-python-sweep-k.md")

	for _, needle := range []string{
		"BIG-GO-102",
		"`tests/test_cost_control.py`",
		"`tests/test_mapping.py`",
		"`tests/test_repo_board.py`",
		"`tests/test_repo_collaboration.py`",
		"`tests/test_roadmap.py`",
		"`bigclaw-go/internal/costcontrol/controller_test.go`",
		"`bigclaw-go/internal/intake/mapping_test.go`",
		"`bigclaw-go/internal/repo/board.go`",
		"`bigclaw-go/internal/collaboration/thread_test.go`",
		"`bigclaw-go/internal/regression/roadmap_contract_test.go`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO102(ResidualPythonTestsStayAbsent|ReplacementSurfacesRemainAvailable|LaneReportCapturesSweepState)$'`",
		"`BIG-GO-1577`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
