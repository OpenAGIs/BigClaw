package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1587RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1587MigrationBucketStaysAbsentAndPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	migrationDir := filepath.Join(rootRepo, "bigclaw-go", "scripts", "migration")

	if _, err := os.Stat(migrationDir); !os.IsNotExist(err) {
		t.Fatalf("expected migration bucket to stay absent: %s (err=%v)", migrationDir, err)
	}

	pythonFiles := collectPythonFiles(t, migrationDir)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected migration bucket to remain Python-free: %v", pythonFiles)
	}
}

func TestBIGGO1587GoMigrationReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawctl/automation_commands.go",
		"bigclaw-go/docs/go-cli-script-migration.md",
		"docs/go-cli-script-migration-plan.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected Go migration replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1587LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1587-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1587",
		"Repository-wide Python file count: `0`.",
		"`bigclaw-go`: `0` Python files",
		"`bigclaw-go/scripts/migration`: `0` Python files",
		"Directory absent on disk: `yes`.",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_commands.go`",
		"`bigclaw-go/docs/go-cli-script-migration.md`",
		"`docs/go-cli-script-migration-plan.md`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find bigclaw-go/scripts/migration -type f -name '*.py' 2>/dev/null | sort`",
		"`test ! -d bigclaw-go/scripts/migration`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1587",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
