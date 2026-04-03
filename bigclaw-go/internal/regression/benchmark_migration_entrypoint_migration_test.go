package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBenchmarkAndMigrationScriptDirectoriesStayPythonFree(t *testing.T) {
	repoRoot := repoRoot(t)
	benchmarkDir := filepath.Join(repoRoot, "scripts", "benchmark")
	entries, err := os.ReadDir(benchmarkDir)
	if err != nil {
		t.Fatalf("read script directory %s: %v", benchmarkDir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".py") {
			t.Fatalf("expected no Python helper in %s, found %s", benchmarkDir, entry.Name())
		}
	}

	if _, err := os.Stat(filepath.Join(repoRoot, "scripts", "migration")); !os.IsNotExist(err) {
		t.Fatalf("expected scripts/migration to remain absent once the Python migration helpers were removed")
	}
}

func TestBenchmarkAndMigrationDocsListOnlyGoEntrypoints(t *testing.T) {
	goRepoRoot := repoRoot(t)
	workspaceRoot := filepath.Clean(filepath.Join(goRepoRoot, ".."))

	goDoc := readRepoFile(t, goRepoRoot, "docs/go-cli-script-migration.md")
	requiredGoDoc := []string{
		"`go run ./cmd/bigclawctl automation benchmark soak-local|run-matrix|capacity-certification ...`",
		"`go run ./cmd/bigclawctl automation migration shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle ...`",
		"go run ./cmd/bigclawctl automation benchmark capacity-certification --help",
		"go run ./cmd/bigclawctl automation migration export-live-shadow-bundle --help",
	}
	for _, needle := range requiredGoDoc {
		if !strings.Contains(goDoc, needle) {
			t.Fatalf("docs/go-cli-script-migration.md missing active entrypoint %q", needle)
		}
	}

	disallowedGoDoc := []string{
		"bigclaw-go/scripts/benchmark/capacity_certification.py",
		"bigclaw-go/scripts/benchmark/run_matrix.py",
		"bigclaw-go/scripts/benchmark/soak_local.py",
		"bigclaw-go/scripts/migration/shadow_compare.py",
		"bigclaw-go/scripts/migration/shadow_matrix.py",
		"bigclaw-go/scripts/migration/live_shadow_scorecard.py",
		"bigclaw-go/scripts/migration/export_live_shadow_bundle.py",
	}
	for _, needle := range disallowedGoDoc {
		if strings.Contains(goDoc, needle) {
			t.Fatalf("docs/go-cli-script-migration.md should not reference removed Python helper %q", needle)
		}
	}

	planDoc := readRepoFile(t, workspaceRoot, "docs/go-cli-script-migration-plan.md")
	requiredPlanDoc := []string{
		"- retired benchmark Python helpers; use `bigclawctl automation benchmark soak-local|run-matrix|capacity-certification`",
		"- retired migration Python helpers; use `bigclawctl automation migration shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle`",
		"- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-matrix --help`",
		"- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration live-shadow-scorecard --help`",
		"- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle --help`",
	}
	for _, needle := range requiredPlanDoc {
		if !strings.Contains(planDoc, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing active entrypoint guidance %q", needle)
		}
	}

	disallowedPlanDoc := []string{
		"bigclaw-go/scripts/migration/shadow_compare.py",
		"bigclaw-go/scripts/benchmark/capacity_certification.py",
		"bigclaw-go/scripts/migration/export_live_shadow_bundle.py",
		"Continue the remaining `bigclaw-go/scripts/*` migration helpers and E2E utilities after this",
	}
	for _, needle := range disallowedPlanDoc {
		if strings.Contains(planDoc, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md should not reference retired Python helper guidance %q", needle)
		}
	}
}
