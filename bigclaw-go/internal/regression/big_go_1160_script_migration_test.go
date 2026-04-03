package regression

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1160CandidatePythonFilesRemainDeleted(t *testing.T) {
	goRepoRoot := repoRoot(t)
	rootRepo := filepath.Clean(filepath.Join(goRepoRoot, ".."))

	candidates := []string{
		"bigclaw-go/scripts/benchmark/capacity_certification.py",
		"bigclaw-go/scripts/benchmark/capacity_certification_test.py",
		"bigclaw-go/scripts/benchmark/run_matrix.py",
		"bigclaw-go/scripts/benchmark/soak_local.py",
		"bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py",
		"bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py",
		"bigclaw-go/scripts/e2e/cross_process_coordination_surface.py",
		"bigclaw-go/scripts/e2e/export_validation_bundle.py",
		"bigclaw-go/scripts/e2e/export_validation_bundle_test.py",
		"bigclaw-go/scripts/e2e/external_store_validation.py",
		"bigclaw-go/scripts/e2e/mixed_workload_matrix.py",
		"bigclaw-go/scripts/e2e/multi_node_shared_queue.py",
		"bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py",
		"bigclaw-go/scripts/e2e/run_all_test.py",
		"bigclaw-go/scripts/e2e/run_task_smoke.py",
		"bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py",
		"bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py",
		"bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py",
		"bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py",
		"bigclaw-go/scripts/migration/export_live_shadow_bundle.py",
		"bigclaw-go/scripts/migration/live_shadow_scorecard.py",
		"bigclaw-go/scripts/migration/shadow_compare.py",
		"bigclaw-go/scripts/migration/shadow_matrix.py",
		"scripts/create_issues.py",
		"scripts/dev_smoke.py",
	}

	for _, relativePath := range candidates {
		_, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath)))
		if !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected BIG-GO-1160 candidate path to stay deleted: %s (err=%v)", relativePath, err)
		}
	}
}

func TestBIGGO1160MigrationDocsListGoReplacements(t *testing.T) {
	goRepoRoot := repoRoot(t)
	rootRepo := filepath.Clean(filepath.Join(goRepoRoot, ".."))

	goDoc := readRepoFile(t, goRepoRoot, "docs/go-cli-script-migration.md")
	rootDoc := readRepoFile(t, rootRepo, "docs/go-cli-script-migration-plan.md")

	requiredGoDoc := []string{
		"Issues: `BIG-GO-902`, `BIG-GO-1053`, `BIG-GO-1160`",
		"`go run ./cmd/bigclawctl automation benchmark soak-local ...`",
		"`go run ./cmd/bigclawctl automation benchmark run-matrix ...`",
		"`go run ./cmd/bigclawctl automation benchmark capacity-certification ...`",
		"`go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix ...`",
		"`go run ./cmd/bigclawctl automation e2e cross-process-coordination-surface ...`",
		"`go run ./cmd/bigclawctl automation e2e export-validation-bundle ...`",
		"`go run ./cmd/bigclawctl automation e2e external-store-validation ...`",
		"`go run ./cmd/bigclawctl automation e2e mixed-workload-matrix ...`",
		"`go run ./cmd/bigclawctl automation e2e multi-node-shared-queue ...`",
		"`go run ./cmd/bigclawctl automation e2e run-task-smoke ...`",
		"`go run ./cmd/bigclawctl automation e2e subscriber-takeover-fault-matrix ...`",
		"`go run ./cmd/bigclawctl automation e2e continuation-policy-gate ...`",
		"`go run ./cmd/bigclawctl automation e2e continuation-scorecard ...`",
		"`go run ./cmd/bigclawctl automation migration export-live-shadow-bundle ...`",
		"`go run ./cmd/bigclawctl automation migration live-shadow-scorecard ...`",
		"`go run ./cmd/bigclawctl automation migration shadow-compare ...`",
		"`go run ./cmd/bigclawctl automation migration shadow-matrix ...`",
		"`bash scripts/ops/bigclawctl create-issues ...`",
		"`bash scripts/ops/bigclawctl dev-smoke`",
		"Benchmark soak/matrix/capacity helpers and their Python-side tests",
		"E2E broker failover, coordination, bundle export, external-store, workload, shared-queue, smoke, takeover, and continuation sweep candidates",
	}
	for _, needle := range requiredGoDoc {
		if !strings.Contains(goDoc, needle) {
			t.Fatalf("bigclaw-go/docs/go-cli-script-migration.md missing BIG-GO-1160 replacement %q", needle)
		}
	}

	requiredRootDoc := []string{
		"`BIG-GO-1160` extends that migration evidence",
		"`bigclaw-go/scripts/migration/shadow_compare.py`",
		"`scripts/create_issues.py`",
		"`scripts/dev_smoke.py`",
		"`bigclawctl automation migration shadow-compare|shadow-matrix|live-shadow-scorecard|export-live-shadow-bundle`",
	}
	for _, needle := range requiredRootDoc {
		if !strings.Contains(rootDoc, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing BIG-GO-1160 coverage %q", needle)
		}
	}
}
