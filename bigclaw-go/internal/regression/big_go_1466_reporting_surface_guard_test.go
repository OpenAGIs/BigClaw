package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1466PythonReportingSurfacesStayRemoved(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	targetFiles := []string{
		"bigclaw-go/cmd/bigclawctl/automation_e2e_mixed_workload_command.go",
		"bigclaw-go/docs/reports/ray-live-smoke-report.json",
		"bigclaw-go/docs/reports/ray-live-jobs.json",
		"bigclaw-go/docs/reports/live-validation-summary.json",
		"bigclaw-go/docs/reports/live-validation-index.json",
		"bigclaw-go/docs/reports/mixed-workload-matrix-report.json",
		"bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/ray-live-smoke-report.json",
		"bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/ray.stdout.log",
		"bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/summary.json",
		"bigclaw-go/docs/reports/live-validation-runs/20260314T164647Z/ray.audit.jsonl",
		"bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json",
		"bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/ray.stdout.log",
		"bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json",
		"bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/ray.audit.jsonl",
	}

	for _, relativePath := range targetFiles {
		contents := readRepoFile(t, rootRepo, relativePath)
		for _, forbidden := range []string{
			"python -c",
			"import ray; ray.init",
		} {
			if strings.Contains(contents, forbidden) {
				t.Fatalf("%s should not contain %q", relativePath, forbidden)
			}
		}
	}
}

func TestBIGGO1466ShellNativeRayEvidenceRemainsPresent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/cmd/bigclawctl/automation_e2e_mixed_workload_command.go",
		"bigclaw-go/docs/reports/ray-live-smoke-report.json",
		"bigclaw-go/docs/reports/ray-live-jobs.json",
		"bigclaw-go/docs/reports/live-validation-summary.json",
		"bigclaw-go/docs/reports/live-validation-index.json",
		"bigclaw-go/docs/reports/mixed-workload-matrix-report.json",
		"bigclaw-go/docs/reports/big-go-1466-python-reporting-observability-sweep.md",
	}
	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected replacement path to exist: %s (%v)", relativePath, err)
		}
	}

	code := readRepoFile(t, rootRepo, "bigclaw-go/cmd/bigclawctl/automation_e2e_mixed_workload_command.go")
	for _, needle := range []string{
		`Entrypoint:              "sh -c 'echo gpu via ray'"`,
		`Entrypoint:              "sh -c 'echo required ray'"`,
	} {
		if !strings.Contains(code, needle) {
			t.Fatalf("mixed workload command missing shell-native ray entrypoint %q", needle)
		}
	}

	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1466-python-reporting-observability-sweep.md")
	for _, needle := range []string{
		"BIG-GO-1466",
		"Repository-wide Python file count remains `0` before and after this change.",
		"`bigclaw-go/docs/reports/ray-live-jobs.json`",
		"`bigclaw-go/docs/reports/ray-live-smoke-report.json`",
		"`bigclaw-go/docs/reports/live-validation-summary.json`",
		"`bigclaw-go/docs/reports/live-validation-index.json`",
		"`bigclaw-go/docs/reports/mixed-workload-matrix-report.json`",
		"`cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression -run 'TestBIGGO1466|TestLiveValidation|TestParallelValidationMatrixDocs'`",
		"`rg -n \"python -c|import ray; ray.init\" bigclaw-go --glob '!**/.git/**' --glob '!bigclaw-go/docs/reports/big-go-1466-python-reporting-observability-sweep.md' --glob '!bigclaw-go/internal/regression/big_go_1466_reporting_surface_guard_test.go' --glob '!bigclaw-go/internal/regression/big_go_1359_zero_python_guard_test.go'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
