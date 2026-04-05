package regression

import (
	"strings"
	"testing"
)

func TestBIGGO1468ActiveDocsStayGoFirst(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	bootstrapTemplate := readRepoFile(t, rootRepo, "docs/symphony-repo-bootstrap-template.md")
	for _, needle := range []string{
		"`scripts/ops/bigclawctl`",
		"`workflow.md`",
		"the repo's Go/native workspace bootstrap implementation behind `bigclawctl workspace ...`",
		"Go-only repositories should keep the repo-specific bootstrap",
	} {
		if !strings.Contains(bootstrapTemplate, needle) {
			t.Fatalf("bootstrap template missing Go-first guidance %q", needle)
		}
	}
	for _, needle := range []string{
		"`src/<your_package>/workspace_bootstrap.py`",
		"`src/<your_package>/workspace_bootstrap_cli.py`",
	} {
		if strings.Contains(bootstrapTemplate, needle) {
			t.Fatalf("bootstrap template should not require deleted Python compatibility files %q", needle)
		}
	}

	handoff := readRepoFile(t, rootRepo, "docs/go-mainline-cutover-handoff.md")
	for _, needle := range []string{
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1454RepositoryHasNoPythonFiles$'`",
		"retired Python contract and runtime",
	} {
		if !strings.Contains(handoff, needle) {
			t.Fatalf("cutover handoff missing Go-only validation guidance %q", needle)
		}
	}
	if strings.Contains(handoff, "PYTHONPATH=src python3") {
		t.Fatalf("cutover handoff should not advertise Python validation commands")
	}
}

func TestBIGGO1468MigrationPlanAvoidsDeletedPythonManifests(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	plan := readRepoFile(t, rootRepo, "docs/go-cli-script-migration-plan.md")

	for _, needle := range []string{
		"the deleted benchmark helper tranche formerly housed under `bigclaw-go/scripts/benchmark`",
		"the deleted e2e automation helper tranche formerly housed under `bigclaw-go/scripts/e2e`",
		"the deleted migration helper tranche formerly housed under `bigclaw-go/scripts/migration`",
		"the retired repo-root `create-issues` and `dev-smoke` Python shims",
		"`bigclawctl automation migration shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle`",
	} {
		if !strings.Contains(plan, needle) {
			t.Fatalf("migration plan missing sweep summary %q", needle)
		}
	}

	for _, needle := range []string{
		"`bigclaw-go/scripts/benchmark/capacity_certification.py`",
		"`bigclaw-go/scripts/e2e/run_task_smoke.py`",
		"`bigclaw-go/scripts/migration/shadow_compare.py`",
	} {
		if strings.Contains(plan, needle) {
			t.Fatalf("migration plan should not enumerate deleted Python helper path %q", needle)
		}
	}
}

func TestBIGGO1468LaneReportCapturesDocReferenceSweep(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1468-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1468",
		"Repository-wide Python file count: `0`.",
		"`docs/symphony-repo-bootstrap-template.md`",
		"`docs/go-cli-script-migration-plan.md`",
		"`docs/go-mainline-cutover-handoff.md`",
		"`scripts/ops/bigclawctl`",
		"`bigclawctl automation ...`",
		"`find . -path '*/.git' -prune -o -type f \\( -name '*.py' -o -name '*.pyi' -o -name '*.pyw' -o -name 'requirements*.txt' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' -o -name 'Pipfile' -o -name 'Pipfile.lock' \\) -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'Test(BIGGO1160ScriptMigrationDocsListGoReplacements|RootOpsMigrationDocsListOnlyGoEntrypoints|BIGGO1468",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
