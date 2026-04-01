package regression

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestRepoWidePythonFileBan(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	allowedPythonFiles := []string{
		"scripts/ops/bigclaw_github_sync.py",
		"scripts/ops/bigclaw_refill_queue.py",
		"scripts/ops/bigclaw_workspace_bootstrap.py",
		"scripts/ops/symphony_workspace_bootstrap.py",
		"src/bigclaw/__init__.py",
		"src/bigclaw/__main__.py",
		"src/bigclaw/audit_events.py",
		"src/bigclaw/collaboration.py",
		"src/bigclaw/connectors.py",
		"src/bigclaw/console_ia.py",
		"src/bigclaw/deprecation.py",
		"src/bigclaw/design_system.py",
		"src/bigclaw/dsl.py",
		"src/bigclaw/evaluation.py",
		"src/bigclaw/event_bus.py",
		"src/bigclaw/governance.py",
		"src/bigclaw/legacy_shim.py",
		"src/bigclaw/memory.py",
		"src/bigclaw/models.py",
		"src/bigclaw/observability.py",
		"src/bigclaw/operations.py",
		"src/bigclaw/parallel_refill.py",
		"src/bigclaw/planning.py",
		"src/bigclaw/repo_links.py",
		"src/bigclaw/repo_plane.py",
		"src/bigclaw/reports.py",
		"src/bigclaw/risk.py",
		"src/bigclaw/roadmap.py",
		"src/bigclaw/run_detail.py",
		"src/bigclaw/runtime.py",
		"src/bigclaw/ui_review.py",
		"src/bigclaw/validation_policy.py",
		"src/bigclaw/workspace_bootstrap.py",
		"src/bigclaw/workspace_bootstrap_cli.py",
		"tests/conftest.py",
		"tests/test_console_ia.py",
		"tests/test_control_center.py",
		"tests/test_design_system.py",
		"tests/test_dsl.py",
		"tests/test_evaluation.py",
		"tests/test_event_bus.py",
		"tests/test_live_shadow_bundle.py",
		"tests/test_memory.py",
		"tests/test_models.py",
		"tests/test_observability.py",
		"tests/test_operations.py",
		"tests/test_orchestration.py",
		"tests/test_parallel_validation_bundle.py",
		"tests/test_planning.py",
		"tests/test_queue.py",
		"tests/test_repo_collaboration.py",
		"tests/test_repo_links.py",
		"tests/test_repo_rollout.py",
		"tests/test_reports.py",
		"tests/test_risk.py",
		"tests/test_runtime_matrix.py",
		"tests/test_scheduler.py",
		"tests/test_ui_review.py",
		"tests/test_validation_bundle_continuation_policy_gate.py",
		"tests/test_validation_policy.py",
	}

	actualPythonFiles := trackedPythonFiles(t, repoRoot)
	unexpected := make([]string, 0)
	for _, path := range actualPythonFiles {
		if !slices.Contains(allowedPythonFiles, path) {
			unexpected = append(unexpected, path)
		}
	}
	if len(unexpected) > 0 {
		t.Fatalf("unexpected tracked Python files found; delete them or replace them with Go before merging: %v", unexpected)
	}
}

func TestWorkspaceValidatePythonShimRemoved(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	deletedPythonFiles := []string{
		"scripts/ops/symphony_workspace_validate.py",
	}
	for _, relativePath := range deletedPythonFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python shim to be absent: %s", relativePath)
		}
	}

	goReplacementFiles := []string{
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/internal/legacyshim/wrappers.go",
	}
	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}

func trackedPythonFiles(t *testing.T, repoRoot string) []string {
	t.Helper()

	output, err := exec.Command("git", "-C", repoRoot, "ls-files", "--", "*.py").Output()
	if err != nil {
		t.Fatalf("list tracked Python files: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	files := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(repoRoot, line)); os.IsNotExist(err) {
			continue
		} else if err != nil {
			t.Fatalf("stat %s: %v", line, err)
		}
		files = append(files, filepath.ToSlash(line))
	}
	return files
}
