package regression

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1165CandidatePythonFilesRemainDeleted(t *testing.T) {
	repoRoot := repoRoot(t)

	candidates := []string{
		"bigclaw-go/scripts/benchmark/capacity_certification.py",
		"bigclaw-go/scripts/benchmark/capacity_certification_test.py",
		"bigclaw-go/scripts/benchmark/run_matrix.py",
		"bigclaw-go/scripts/benchmark/soak_local.py",
		"bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py",
		"bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py",
		"bigclaw-go/scripts/e2e/cross_process_coordination_surface.py",
		"bigclaw-go/scripts/e2e/export_validation_bundle_test.py",
		"bigclaw-go/scripts/e2e/external_store_validation.py",
		"bigclaw-go/scripts/e2e/mixed_workload_matrix.py",
		"bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py",
		"bigclaw-go/scripts/e2e/run_all_test.py",
		"bigclaw-go/scripts/e2e/run_task_smoke.py",
		"bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py",
		"bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py",
		"bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py",
		"bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py",
		"bigclaw-go/scripts/migration/export_live_shadow_bundle.py",
		"bigclaw-go/scripts/migration/live_shadow_scorecard.py",
		"bigclaw-go/scripts/migration/shadow_compare.py",
		"bigclaw-go/scripts/migration/shadow_matrix.py",
		"scripts/create_issues.py",
		"scripts/dev_smoke.py",
		"tests/test_parallel_validation_bundle.py",
		"tests/test_validation_bundle_continuation_policy_gate.py",
		"tests/test_validation_bundle_continuation_scorecard.py",
		"tests/test_subscriber_takeover_harness.py",
	}

	for _, relativePath := range candidates {
		_, err := os.Stat(filepath.Join(repoRoot, relativePath))
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected BIG-GO-1165 candidate path to stay deleted: %s (err=%v)", relativePath, err)
		}
	}
}

func TestBIGGO1165MigrationPlanDocumentsReplacementPaths(t *testing.T) {
	repoRoot := repoRoot(t)

	migrationPlan := readRepoFile(t, repoRoot, "../docs/go-cli-script-migration-plan.md")
	requiredPlanEntries := []string{
		"`scripts/dev_smoke.py` -> `cd bigclaw-go && go test ./...` and `cd bigclaw-go && go run ./cmd/bigclawd`",
		"`scripts/create_issues.py` -> `cd bigclaw-go && go run ./cmd/bigclawctl local-issues --help`",
		"`bigclaw-go/scripts/benchmark/run_matrix.py` -> `cd bigclaw-go && go test -bench . ./internal/queue ./internal/scheduler`",
		"`bigclaw-go/scripts/e2e/external_store_validation.py` -> `cd bigclaw-go && go test ./internal/regression -run TestExternalStoreValidationReportStaysAligned -count=1`",
		"`bigclaw-go/scripts/e2e/cross_process_coordination_surface.py` -> `cd bigclaw-go && go test ./internal/regression -run TestCrossProcessCoordinationReadinessDocsStayAligned -count=1`",
		"`bigclaw-go/scripts/e2e/run_task_smoke.py` -> `cd bigclaw-go && go run ./cmd/bigclawctl e2e run-task-smoke --help`",
		"`bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py` -> `cd bigclaw-go && go run ./cmd/bigclawctl e2e subscriber-takeover-fault-matrix --pretty`",
		"`bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py` -> `cd bigclaw-go && go run ./cmd/bigclawctl e2e validation-bundle-continuation-scorecard --pretty`",
		"`bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py` -> `cd bigclaw-go && go run ./cmd/bigclawctl e2e validation-bundle-continuation-policy-gate --pretty`",
		"`bigclaw-go/scripts/migration/shadow_compare.py` -> checked-in `bigclaw-go/docs/reports/shadow-compare-report.json`",
		"`bigclaw-go/scripts/migration/export_live_shadow_bundle.py` -> checked-in `bigclaw-go/docs/reports/live-shadow-summary.json` and `bigclaw-go/docs/reports/live-shadow-index.json`",
		"`138`",
	}
	for _, needle := range requiredPlanEntries {
		if !strings.Contains(migrationPlan, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing BIG-GO-1165 sweep guidance %q", needle)
		}
	}
}
