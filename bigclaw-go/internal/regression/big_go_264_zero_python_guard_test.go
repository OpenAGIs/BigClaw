package regression

import (
	"strings"
	"testing"
)

func TestBIGGO264MixedWorkloadHelperAvoidsPythonEntrypoints(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	source := readRepoFile(t, rootRepo, "bigclaw-go/cmd/bigclawctl/automation_e2e_mixed_workload_command.go")

	for _, forbidden := range []string{
		`python -c "print('gpu via ray')"`,
		`python -c "print('required ray')"`,
	} {
		if strings.Contains(source, forbidden) {
			t.Fatalf("mixed workload helper should not retain Python command residue %q", forbidden)
		}
	}

	for _, required := range []string{
		`Entrypoint:              "echo gpu via ray"`,
		`Entrypoint:              "echo required ray"`,
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("mixed workload helper missing Go/native replacement command %q", required)
		}
	}
}

func TestBIGGO264LaneReportCapturesMixedWorkloadHelperSweep(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-264-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-264",
		"Repository-wide Python file count: `0`.",
		"`bigclaw-go/cmd/bigclawctl/automation_e2e_mixed_workload_command.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_e2e_mixed_workload_command_test.go`",
		"`python -c \\\"print('gpu via ray')\\\"` -> `echo gpu via ray`",
		"`python -c \\\"print('required ray')\\\"` -> `echo required ray`",
		"`cd bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestAutomationMixedWorkloadMatrixBuildsReport|TestDefaultMixedWorkloadTasksUseNoPythonEntrypoints'`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO264(MixedWorkloadHelperAvoidsPythonEntrypoints|LaneReportCapturesMixedWorkloadHelperSweep)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
