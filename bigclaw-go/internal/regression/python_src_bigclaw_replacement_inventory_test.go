package regression

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSrcBigClawGoReplacementInventory(t *testing.T) {
	goRoot := repoRoot(t)
	repoRoot := filepath.Clean(filepath.Join(goRoot, ".."))

	deletedPythonFiles := []string{
		"src/bigclaw/console_ia.py",
		"src/bigclaw/connectors.py",
		"src/bigclaw/cost_control.py",
		"src/bigclaw/dashboard_run_contract.py",
		"src/bigclaw/design_system.py",
		"src/bigclaw/dsl.py",
		"src/bigclaw/event_bus.py",
		"src/bigclaw/issue_archive.py",
		"src/bigclaw/mapping.py",
		"src/bigclaw/pilot.py",
		"src/bigclaw/repo_board.py",
		"src/bigclaw/repo_commits.py",
		"src/bigclaw/repo_gateway.py",
		"src/bigclaw/repo_governance.py",
		"src/bigclaw/repo_registry.py",
		"src/bigclaw/repo_triage.py",
		"src/bigclaw/saved_views.py",
	}
	for _, relativePath := range deletedPythonFiles {
		path := filepath.Join(repoRoot, relativePath)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("expected deleted Python file to stay absent: %s", relativePath)
		}
	}

	goOwners := []string{
		"bigclaw-go/internal/costcontrol/controller.go",
		"bigclaw-go/internal/events/bus.go",
		"bigclaw-go/internal/governance/freeze.go",
		"bigclaw-go/internal/intake/connector.go",
		"bigclaw-go/internal/intake/mapping.go",
		"bigclaw-go/internal/issuearchive/archive.go",
		"bigclaw-go/internal/pilot/report.go",
		"bigclaw-go/internal/product/console.go",
		"bigclaw-go/internal/product/console_test.go",
		"bigclaw-go/internal/product/dashboard_run_contract.go",
		"bigclaw-go/internal/product/saved_views.go",
		"bigclaw-go/internal/repo/board.go",
		"bigclaw-go/internal/repo/commits.go",
		"bigclaw-go/internal/repo/gateway.go",
		"bigclaw-go/internal/repo/governance.go",
		"bigclaw-go/internal/repo/registry.go",
		"bigclaw-go/internal/repo/triage.go",
		"bigclaw-go/internal/workflow/definition.go",
	}
	for _, relativePath := range goOwners {
		path := filepath.Join(repoRoot, relativePath)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("expected Go owner file to exist: %s: %v", relativePath, err)
		}
		if info.IsDir() {
			t.Fatalf("expected Go owner path to be a file: %s", relativePath)
		}
	}
}
