package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO103RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO103ResidualPythonTestPathsStayAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"tests/test_cost_control.py",
		"tests/test_mapping.py",
		"tests/test_repo_board.py",
		"tests/test_repo_collaboration.py",
		"tests/test_roadmap.py",
		"tests/test_design_system.py",
		"tests/test_live_shadow_bundle.py",
		"tests/test_pilot.py",
		"tests/test_repo_triage.py",
		"tests/test_subscriber_takeover_harness.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired residual Python test path to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO103GoReplacementTestsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/costcontrol/controller_test.go",
		"bigclaw-go/internal/designsystem/designsystem_test.go",
		"bigclaw-go/internal/events/subscriber_leases_test.go",
		"bigclaw-go/internal/intake/mapping_test.go",
		"bigclaw-go/internal/pilot/report_test.go",
		"bigclaw-go/internal/pilot/rollout_test.go",
		"bigclaw-go/internal/regression/live_multinode_takeover_proof_test.go",
		"bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go",
		"bigclaw-go/internal/regression/takeover_proof_surface_test.go",
		"bigclaw-go/internal/repo/governance_test.go",
		"bigclaw-go/internal/repo/repo_surfaces_test.go",
		"bigclaw-go/internal/triage/triage_test.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement test path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO103LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-103-residual-tests-python-sweep-l.md")

	for _, needle := range []string{
		"BIG-GO-103",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused residual Python test file count before lane changes: `0`",
		"Focused residual Python test file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"`tests/test_cost_control.py`",
		"`tests/test_mapping.py`",
		"`tests/test_repo_board.py`",
		"`tests/test_repo_collaboration.py`",
		"`tests/test_roadmap.py`",
		"`tests/test_design_system.py`",
		"`tests/test_live_shadow_bundle.py`",
		"`tests/test_pilot.py`",
		"`tests/test_repo_triage.py`",
		"`tests/test_subscriber_takeover_harness.py`",
		"`bigclaw-go/internal/costcontrol/controller_test.go`",
		"`bigclaw-go/internal/designsystem/designsystem_test.go`",
		"`bigclaw-go/internal/events/subscriber_leases_test.go`",
		"`bigclaw-go/internal/intake/mapping_test.go`",
		"`bigclaw-go/internal/pilot/report_test.go`",
		"`bigclaw-go/internal/pilot/rollout_test.go`",
		"`bigclaw-go/internal/regression/live_multinode_takeover_proof_test.go`",
		"`bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`",
		"`bigclaw-go/internal/regression/takeover_proof_surface_test.go`",
		"`bigclaw-go/internal/repo/governance_test.go`",
		"`bigclaw-go/internal/repo/repo_surfaces_test.go`",
		"`bigclaw-go/internal/triage/triage_test.go`",
		"`find tests bigclaw-go -type f \\( -name 'test_*.py' -o -name '*_test.py' \\) 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO103",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
