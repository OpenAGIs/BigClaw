package regression

import (
	"strings"
	"testing"
)

func TestBIGGO1483MigrationPlanListsOnlyGoOrShellBigClawScriptEntrypoints(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	rootDoc := readRepoFile(t, rootRepo, "docs/go-cli-script-migration-plan.md")

	required := []string{
		"`BIG-GO-1483` removes the",
		"`bigclaw-go/scripts/` are shell or Go entrypoints only",
		"`benchmark/run_suite.sh`",
		"`e2e/run_all.sh`",
		"`e2e/kubernetes_smoke.sh`",
		"`e2e/ray_smoke.sh`",
		"`e2e/broker_bootstrap_summary.go`",
		"`bigclawctl automation e2e ...`",
		"`bigclawctl automation benchmark soak-local|run-matrix|capacity-certification`",
		"`bigclawctl automation migration shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle`",
	}
	for _, needle := range required {
		if !strings.Contains(rootDoc, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing BIG-GO-1483 caller cutover text %q", needle)
		}
	}

	disallowed := []string{
		"bigclaw-go/scripts/benchmark/capacity_certification.py",
		"bigclaw-go/scripts/benchmark/run_matrix.py",
		"bigclaw-go/scripts/benchmark/soak_local.py",
		"bigclaw-go/scripts/e2e/run_task_smoke.py",
		"bigclaw-go/scripts/e2e/export_validation_bundle.py",
		"bigclaw-go/scripts/migration/shadow_compare.py",
	}
	for _, needle := range disallowed {
		if strings.Contains(rootDoc, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md should not reference retired Python caller text %q", needle)
		}
	}
}

func TestBIGGO1483LaneReportCapturesCallerCutoverState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1483-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1483",
		"Before update checked-in caller references to retired `bigclaw-go/scripts` Python paths: `23`.",
		"After update checked-in caller references to retired `bigclaw-go/scripts` Python paths: `0`.",
		"Before update physical Python files under `bigclaw-go/scripts`: `0`.",
		"After update physical Python files under `bigclaw-go/scripts`: `0`.",
		"`benchmark/run_suite.sh`",
		"`e2e/run_all.sh`",
		"`e2e/kubernetes_smoke.sh`",
		"`e2e/ray_smoke.sh`",
		"`e2e/broker_bootstrap_summary.go`",
		"`rg -n --glob '!reports/**' --glob '!bigclaw-go/docs/reports/**' --glob '!local-issues.json' --glob '!bigclaw-go/internal/regression/**' --glob '!.symphony/**' 'bigclaw-go/scripts/.*\\\\.py' README.md docs scripts .github bigclaw-go | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1160|TestBIGGO1483|TestE2E'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
