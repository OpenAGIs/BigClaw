package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1580RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1580CandidatePathsRemainAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	candidatePaths := []string{
		"src/bigclaw/dsl.py",
		"src/bigclaw/observability.py",
		"src/bigclaw/repo_governance.py",
		"src/bigclaw/saved_views.py",
		"tests/test_audit_events.py",
		"tests/test_event_bus.py",
		"tests/test_memory.py",
		"tests/test_repo_board.py",
		"tests/test_roadmap.py",
		"tests/test_workflow.py",
		"bigclaw-go/scripts/benchmark/capacity_certification_test.py",
		"bigclaw-go/scripts/e2e/multi_node_shared_queue.py",
		"bigclaw-go/scripts/migration/shadow_matrix.py",
	}

	for _, relativePath := range candidatePaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected BIG-GO-1580 candidate path to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO1580GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/workflow/definition.go",
		"bigclaw-go/internal/workflow/engine.go",
		"bigclaw-go/internal/observability/audit.go",
		"bigclaw-go/internal/observability/recorder.go",
		"bigclaw-go/internal/repo/governance.go",
		"bigclaw-go/internal/product/saved_views.go",
		"bigclaw-go/internal/observability/audit_test.go",
		"bigclaw-go/internal/events/bus_test.go",
		"bigclaw-go/internal/policy/memory_test.go",
		"bigclaw-go/internal/repo/repo_surfaces_test.go",
		"bigclaw-go/internal/regression/roadmap_contract_test.go",
		"bigclaw-go/internal/workflow/engine_test.go",
		"bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_commands_test.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_multi_node_shared_queue_command.go",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_multi_node_shared_queue_command_test.go",
		"bigclaw-go/cmd/bigclawctl/migration_commands.go",
		"bigclaw-go/cmd/bigclawctl/migration_commands_test.go",
		"docs/go-mainline-cutover-handoff.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1580LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1580-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1580",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused `BIG-GO-1580` candidate-path physical Python file count before lane",
		"Focused `BIG-GO-1580` candidate-path physical Python file count after lane",
		"Deleted files in this lane: `[]`",
		"`src/bigclaw/dsl.py`",
		"`src/bigclaw/observability.py`",
		"`src/bigclaw/repo_governance.py`",
		"`src/bigclaw/saved_views.py`",
		"`tests/test_audit_events.py`",
		"`tests/test_event_bus.py`",
		"`tests/test_memory.py`",
		"`tests/test_repo_board.py`",
		"`tests/test_roadmap.py`",
		"`tests/test_workflow.py`",
		"`bigclaw-go/scripts/benchmark/capacity_certification_test.py`",
		"`bigclaw-go/scripts/e2e/multi_node_shared_queue.py`",
		"`bigclaw-go/scripts/migration/shadow_matrix.py`",
		"`bigclaw-go/internal/workflow/definition.go`",
		"`bigclaw-go/internal/observability/audit.go`",
		"`bigclaw-go/internal/repo/governance.go`",
		"`bigclaw-go/internal/product/saved_views.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_e2e_multi_node_shared_queue_command.go`",
		"`bigclaw-go/cmd/bigclawctl/migration_commands.go`",
		"`docs/go-mainline-cutover-handoff.md`",
		"`PYTHONPATH=src python3 - <<\"... legacy shim assertions ...\"`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src tests bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1580",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}

func TestBIGGO1580GoMainlineCutoverHandoffStaysGoOnly(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	handoff := readRepoFile(t, rootRepo, "docs/go-mainline-cutover-handoff.md")

	if strings.Contains(handoff, "PYTHONPATH=src python3") {
		t.Fatal("expected go-mainline cutover handoff to avoid Python validation commands")
	}
	if !strings.Contains(handoff, "`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`") {
		t.Fatal("expected go-mainline cutover handoff to record the zero-Python validation command")
	}
}
