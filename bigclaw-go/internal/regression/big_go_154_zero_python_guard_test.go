package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO154RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO154ResidualScriptAreasStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	residualDirs := []string{
		"scripts",
		"scripts/ops",
		"bigclaw-go/scripts",
	}

	for _, relativeDir := range residualDirs {
		pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, filepath.FromSlash(relativeDir)))
		if len(pythonFiles) != 0 {
			t.Fatalf("expected residual script area to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
	}
}

func TestBIGGO154SupportedRootHelpersRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	requiredPaths := []string{
		"scripts/dev_bootstrap.sh",
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
		"docs/go-cli-script-migration-plan.md",
		"bigclaw-go/docs/go-cli-script-migration.md",
	}

	for _, relativePath := range requiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected supported root helper path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO154RootHelperInventoryMatchesContract(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	rootScripts, err := os.ReadDir(filepath.Join(rootRepo, "scripts"))
	if err != nil {
		t.Fatalf("read scripts directory: %v", err)
	}
	if len(rootScripts) != 2 {
		t.Fatalf("expected scripts directory to contain exactly dev_bootstrap.sh and ops, found %d entries", len(rootScripts))
	}
	expectedRoot := map[string]bool{
		"dev_bootstrap.sh": true,
		"ops":              true,
	}
	for _, entry := range rootScripts {
		if !expectedRoot[entry.Name()] {
			t.Fatalf("unexpected scripts entry: %s", entry.Name())
		}
	}

	opsEntries, err := os.ReadDir(filepath.Join(rootRepo, "scripts", "ops"))
	if err != nil {
		t.Fatalf("read scripts/ops directory: %v", err)
	}
	expectedOps := map[string]bool{
		"bigclawctl":       true,
		"bigclaw-issue":    true,
		"bigclaw-panel":    true,
		"bigclaw-symphony": true,
	}
	if len(opsEntries) != len(expectedOps) {
		t.Fatalf("expected scripts/ops to contain exactly %d helper files, found %d", len(expectedOps), len(opsEntries))
	}
	for _, entry := range opsEntries {
		if entry.IsDir() {
			t.Fatalf("expected flat scripts/ops helper inventory, found directory %s", entry.Name())
		}
		if !expectedOps[entry.Name()] {
			t.Fatalf("unexpected scripts/ops helper: %s", entry.Name())
		}
	}
}

func TestBIGGO154LaneReportCapturesExactLedger(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-154-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-154",
		"Repository-wide physical Python file count before lane changes: `0`",
		"Repository-wide physical Python file count after lane changes: `0`",
		"Focused `scripts/scripts/ops/bigclaw-go/scripts` physical Python file count before lane changes: `0`",
		"Focused `scripts/scripts/ops/bigclaw-go/scripts` physical Python file count after lane changes: `0`",
		"Deleted files in this lane: `[]`",
		"Focused ledger for `scripts/scripts/ops/bigclaw-go/scripts`: `[]`",
		"`scripts`: `0` Python files",
		"`scripts/ops`: `0` Python files",
		"`bigclaw-go/scripts`: `0` Python files",
		"`scripts/dev_bootstrap.sh`",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`docs/go-cli-script-migration-plan.md`",
		"`bigclaw-go/docs/go-cli-script-migration.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find scripts scripts/ops bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO154",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
