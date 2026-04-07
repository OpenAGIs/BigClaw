package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1578RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1578CandidatePathsStayAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"src/bigclaw/dashboard_run_contract.py",
		"src/bigclaw/memory.py",
		"src/bigclaw/repo_commits.py",
		"src/bigclaw/run_detail.py",
		"src/bigclaw/workspace_bootstrap_validation.py",
		"tests/test_dsl.py",
		"tests/test_live_shadow_scorecard.py",
		"tests/test_planning.py",
		"tests/test_reports.py",
		"tests/test_ui_review.py",
		"scripts/ops/symphony_workspace_validate.py",
		"bigclaw-go/scripts/e2e/external_store_validation.py",
		"bigclaw-go/scripts/migration/live_shadow_scorecard.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired Python path to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO1578GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/product/dashboard_run_contract.go",
		"bigclaw-go/internal/policy/memory.go",
		"bigclaw-go/internal/collaboration/thread.go",
		"bigclaw-go/internal/observability/task_run.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/internal/workflow/definition_test.go",
		"bigclaw-go/internal/planning/planning_test.go",
		"bigclaw-go/internal/reporting/reporting_test.go",
		"bigclaw-go/internal/uireview/uireview_test.go",
		"scripts/ops/bigclawctl",
		"bigclaw-go/cmd/bigclawctl/automation_e2e_external_store_validation_command.go",
		"bigclaw-go/cmd/bigclawctl/migration_commands.go",
		"docs/go-cli-script-migration-plan.md",
		"bigclaw-go/docs/migration-shadow.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1578LaneReportCapturesSweepLedger(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1578-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1578",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused candidate set physical Python file count before lane changes: `0`",
		"Focused candidate set physical Python file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Compatibility shims retained in this lane: `[]`",
		"`src/bigclaw/dashboard_run_contract.py` -> `bigclaw-go/internal/product/dashboard_run_contract.go`",
		"`src/bigclaw/memory.py` -> `bigclaw-go/internal/policy/memory.go`",
		"`src/bigclaw/repo_commits.py` -> `bigclaw-go/internal/collaboration/thread.go`",
		"`src/bigclaw/run_detail.py` -> `bigclaw-go/internal/observability/task_run.go`",
		"`src/bigclaw/workspace_bootstrap_validation.py` -> `bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`tests/test_dsl.py` -> `bigclaw-go/internal/workflow/definition_test.go`",
		"`tests/test_live_shadow_scorecard.py` -> `bigclaw-go/cmd/bigclawctl/migration_commands.go`",
		"`tests/test_planning.py` -> `bigclaw-go/internal/planning/planning_test.go`",
		"`tests/test_reports.py` -> `bigclaw-go/internal/reporting/reporting_test.go`",
		"`tests/test_ui_review.py` -> `bigclaw-go/internal/uireview/uireview_test.go`",
		"`scripts/ops/symphony_workspace_validate.py` -> `scripts/ops/bigclawctl`",
		"`bigclaw-go/scripts/e2e/external_store_validation.py` -> `bigclaw-go/cmd/bigclawctl/automation_e2e_external_store_validation_command.go`",
		"`bigclaw-go/scripts/migration/live_shadow_scorecard.py` -> `bigclaw-go/cmd/bigclawctl/migration_commands.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find src tests scripts/ops bigclaw-go/scripts/e2e bigclaw-go/scripts/migration -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1578",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
