package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1575RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1575CandidatePathsRemainAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	candidatePaths := []string{
		"src/bigclaw/connectors.py",
		"src/bigclaw/governance.py",
		"src/bigclaw/planning.py",
		"src/bigclaw/reports.py",
		"src/bigclaw/workflow.py",
		"tests/test_cross_process_coordination_surface.py",
		"tests/test_governance.py",
		"tests/test_parallel_refill.py",
		"tests/test_repo_registry.py",
		"tests/test_service.py",
		"scripts/ops/bigclaw_refill_queue.py",
		"bigclaw-go/scripts/e2e/cross_process_coordination_surface.py",
		"bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py",
	}

	for _, relativePath := range candidatePaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected candidate Python path to stay absent: %s (err=%v)", relativePath, err)
		}
	}
}

func TestBIGGO1575ReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/intake/connector.go",
		"bigclaw-go/internal/governance/freeze.go",
		"bigclaw-go/internal/planning/planning.go",
		"bigclaw-go/internal/reporting/reporting.go",
		"bigclaw-go/internal/workflow/definition.go",
		"bigclaw-go/internal/service/server.go",
		"bigclaw-go/internal/repo/registry.go",
		"bigclaw-go/internal/refill/queue.go",
		"scripts/ops/bigclawctl",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_coordination_surface_command.go",
		"bigclaw-go/internal/api/validation_bundle_continuation_surface.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1575LaneReportCapturesCoverage(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1575-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1575",
		"`src/bigclaw/connectors.py`",
		"`src/bigclaw/governance.py`",
		"`src/bigclaw/planning.py`",
		"`src/bigclaw/reports.py`",
		"`src/bigclaw/workflow.py`",
		"`tests/test_cross_process_coordination_surface.py`",
		"`tests/test_governance.py`",
		"`tests/test_parallel_refill.py`",
		"`tests/test_repo_registry.py`",
		"`tests/test_service.py`",
		"`scripts/ops/bigclaw_refill_queue.py`",
		"`bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`",
		"`bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Candidate file count still present before lane changes: `0`",
		"Candidate file count still present after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Compatibility shims left in this lane: `[]`",
		"`bigclaw-go/internal/intake/connector.go`",
		"`bigclaw-go/internal/governance/freeze.go`",
		"`bigclaw-go/internal/planning/planning.go`",
		"`bigclaw-go/internal/reporting/reporting.go`",
		"`bigclaw-go/internal/workflow/definition.go`",
		"`bigclaw-go/internal/service/server.go`",
		"`bigclaw-go/internal/repo/registry.go`",
		"`bigclaw-go/internal/refill/queue.go`",
		"`scripts/ops/bigclawctl`",
		"`bigclaw-go/cmd/bigclawctl/automation_e2e_coordination_surface_command.go`",
		"`bigclaw-go/internal/api/validation_bundle_continuation_surface.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`for f in src/bigclaw/connectors.py src/bigclaw/governance.py",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1575",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
