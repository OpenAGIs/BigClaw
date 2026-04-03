package regression

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var bigGO1161CandidateReplacements = map[string][]string{
	"src/bigclaw/__init__.py":               {"bigclaw-go/cmd/bigclawctl/main.go"},
	"src/bigclaw/__main__.py":               {"bigclaw-go/cmd/bigclawctl/main.go", "bigclaw-go/cmd/bigclawd/main.go"},
	"src/bigclaw/audit_events.py":           {"bigclaw-go/internal/observability/audit.go", "bigclaw-go/internal/observability/audit_spec.go"},
	"src/bigclaw/collaboration.py":          {"bigclaw-go/internal/collaboration/thread.go"},
	"src/bigclaw/connectors.py":             {"bigclaw-go/internal/intake/connector.go", "bigclaw-go/internal/intake/connector_test.go"},
	"src/bigclaw/console_ia.py":             {"bigclaw-go/internal/consoleia/consoleia.go"},
	"src/bigclaw/cost_control.py":           {"bigclaw-go/internal/costcontrol/controller.go"},
	"src/bigclaw/dashboard_run_contract.py": {"bigclaw-go/internal/product/dashboard_run_contract.go"},
	"src/bigclaw/design_system.py":          {"bigclaw-go/internal/designsystem/designsystem.go"},
	"src/bigclaw/dsl.py":                    {"bigclaw-go/internal/workflow/definition.go", "bigclaw-go/internal/workflow/engine.go"},
	"src/bigclaw/evaluation.py":             {"bigclaw-go/internal/evaluation/evaluation.go"},
	"src/bigclaw/event_bus.py":              {"bigclaw-go/internal/events/transition_bus.go", "bigclaw-go/internal/events/transition_bus_test.go"},
	"src/bigclaw/execution_contract.py":     {"bigclaw-go/internal/contract/execution.go"},
	"src/bigclaw/github_sync.py":            {"bigclaw-go/internal/githubsync/sync.go"},
	"src/bigclaw/governance.py":             {"bigclaw-go/internal/governance/freeze.go"},
	"src/bigclaw/issue_archive.py":          {"bigclaw-go/internal/issuearchive/archive.go"},
	"src/bigclaw/mapping.py":                {"bigclaw-go/internal/intake/mapping.go"},
	"src/bigclaw/memory.py":                 {"bigclaw-go/internal/policy/memory.go", "bigclaw-go/internal/policy/memory_test.go"},
	"src/bigclaw/models.py":                 {"bigclaw-go/internal/domain/task.go"},
	"src/bigclaw/observability.py":          {"bigclaw-go/internal/observability/recorder.go"},
	"src/bigclaw/operations.py":             {"bigclaw-go/internal/product/dashboard_run_contract.go"},
	"src/bigclaw/orchestration.py":          {"bigclaw-go/internal/workflow/orchestration.go"},
	"src/bigclaw/parallel_refill.py":        {"bigclaw-go/internal/refill/queue.go"},
	"src/bigclaw/pilot.py":                  {"bigclaw-go/internal/pilot/report.go"},
	"src/bigclaw/planning.py":               {"bigclaw-go/internal/planning/planning.go"},
	"src/bigclaw/queue.py":                  {"bigclaw-go/internal/queue/queue.go"},
	"src/bigclaw/repo_board.py":             {"bigclaw-go/internal/repo/board.go"},
	"src/bigclaw/repo_commits.py":           {"bigclaw-go/internal/repo/commits.go"},
	"src/bigclaw/repo_gateway.py":           {"bigclaw-go/internal/repo/gateway.go"},
	"src/bigclaw/repo_governance.py":        {"bigclaw-go/internal/repo/governance.go"},
	"src/bigclaw/repo_links.py":             {"bigclaw-go/internal/repo/links.go"},
	"src/bigclaw/repo_plane.py":             {"bigclaw-go/internal/repo/plane.go"},
	"src/bigclaw/repo_registry.py":          {"bigclaw-go/internal/repo/registry.go"},
	"src/bigclaw/repo_triage.py":            {"bigclaw-go/internal/repo/triage.go"},
	"src/bigclaw/reports.py":                {"bigclaw-go/internal/reporting/reporting.go", "bigclaw-go/internal/reportstudio/reportstudio.go"},
	"src/bigclaw/risk.py":                   {"bigclaw-go/internal/risk/risk.go"},
	"src/bigclaw/roadmap.py":                {"bigclaw-go/internal/regression/roadmap_contract_test.go"},
	"src/bigclaw/run_detail.py":             {"bigclaw-go/internal/observability/task_run.go"},
	"src/bigclaw/runtime.py":                {"bigclaw-go/internal/worker/runtime.go"},
	"src/bigclaw/saved_views.py":            {"bigclaw-go/internal/product/saved_views.go"},
}

func TestBIGGO1161CandidatePythonFilesRemainDeleted(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	for relativePath := range bigGO1161CandidateReplacements {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python module to be absent: %s", relativePath)
		}
	}
}

func TestBIGGO1161GoReplacementPathsExist(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	for deletedPath, replacementPaths := range bigGO1161CandidateReplacements {
		for _, replacementPath := range replacementPaths {
			if _, err := os.Stat(filepath.Join(repoRoot, replacementPath)); err != nil {
				t.Fatalf("expected Go replacement for %s to exist at %s (%v)", deletedPath, replacementPath, err)
			}
		}
	}
}

func TestBIGGO1161RepositoryContainsNoPythonFiles(t *testing.T) {
	repoRoot := regressionRepoRoot(t)
	var pythonFiles []string

	err := filepath.WalkDir(repoRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if strings.HasSuffix(entry.Name(), ".py") {
			relativePath, err := filepath.Rel(repoRoot, path)
			if err != nil {
				return err
			}
			pythonFiles = append(pythonFiles, filepath.ToSlash(relativePath))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repository for Python files: %v", err)
	}

	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found: %v", pythonFiles)
	}
}
