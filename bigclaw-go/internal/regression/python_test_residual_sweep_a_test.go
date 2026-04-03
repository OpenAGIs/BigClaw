package regression

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPythonTestResidualSweepA(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	_, err := os.Stat(filepath.Join(repoRoot, "tests"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected deleted Python tests directory to stay absent, stat err=%v", err)
	}

	goReplacementFiles := []string{
		"bigclaw-go/internal/observability/audit_test.go",
		"bigclaw-go/internal/intake/connector_test.go",
		"bigclaw-go/internal/consoleia/consoleia_test.go",
		"bigclaw-go/internal/reporting/reporting_test.go",
		"bigclaw-go/internal/costcontrol/controller_test.go",
		"bigclaw-go/internal/regression/cross_process_coordination_docs_test.go",
		"bigclaw-go/internal/product/dashboard_run_contract_test.go",
		"bigclaw-go/internal/designsystem/designsystem_test.go",
		"bigclaw-go/internal/workflow/definition_test.go",
		"bigclaw-go/internal/evaluation/evaluation_test.go",
		"bigclaw-go/internal/events/bus_test.go",
		"bigclaw-go/internal/contract/execution_test.go",
		"bigclaw-go/internal/workflow/engine_test.go",
		"bigclaw-go/internal/reporting/reporting_test.go",
		"bigclaw-go/internal/githubsync/sync_test.go",
		"bigclaw-go/internal/governance/freeze_test.go",
		"bigclaw-go/internal/issuearchive/archive_test.go",
		"bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go",
		"bigclaw-go/internal/regression/live_validation_summary_test.go",
		"bigclaw-go/internal/intake/mapping_test.go",
		"bigclaw-go/internal/policy/memory_test.go",
		"bigclaw-go/internal/workflow/model_test.go",
		"bigclaw-go/internal/observability/recorder_test.go",
		"bigclaw-go/internal/product/console_test.go",
		"bigclaw-go/internal/workflow/orchestration_test.go",
		"bigclaw-go/internal/refill/queue_test.go",
		"bigclaw-go/internal/regression/parallel_validation_matrix_docs_test.go",
		"bigclaw-go/internal/pilot/report_test.go",
		"bigclaw-go/internal/planning/planning_test.go",
		"bigclaw-go/internal/queue/memory_queue_test.go",
		"bigclaw-go/internal/triage/repo_test.go",
		"bigclaw-go/internal/collaboration/thread_test.go",
		"bigclaw-go/internal/repo/repo_surfaces_test.go",
		"bigclaw-go/internal/repo/governance_test.go",
		"bigclaw-go/internal/control/clawhost_rollout_test.go",
		"bigclaw-go/internal/triage/triage_test.go",
		"bigclaw-go/internal/reportstudio/reportstudio_test.go",
	}

	for _, relativePath := range goReplacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}
}
