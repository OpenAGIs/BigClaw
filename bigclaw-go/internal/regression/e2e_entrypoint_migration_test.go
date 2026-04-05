package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestE2EScriptDirectoryStaysPythonFree(t *testing.T) {
	repoRoot := repoRoot(t)
	e2eDir := filepath.Join(repoRoot, "scripts", "e2e")

	entries, err := os.ReadDir(e2eDir)
	if err != nil {
		t.Fatalf("read e2e script directory: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".py") {
			t.Fatalf("expected no Python helper in scripts/e2e, found %s", entry.Name())
		}
	}
}

func TestE2EMigrationDocListsOnlyActiveEntrypoints(t *testing.T) {
	repoRoot := repoRoot(t)
	contents := readRepoFile(t, repoRoot, "docs/go-cli-script-migration.md")

	required := []string{
		"`go run ./cmd/bigclawctl automation e2e run-task-smoke ...`",
		"`go run ./cmd/bigclawctl automation e2e export-validation-bundle ...`",
		"`./scripts/e2e/run_all.sh`",
		"`./scripts/e2e/kubernetes_smoke.sh`",
		"`./scripts/e2e/ray_smoke.sh`",
	}
	for _, needle := range required {
		if !strings.Contains(contents, needle) {
			t.Fatalf("docs/go-cli-script-migration.md missing active entrypoint %q", needle)
		}
	}

	disallowed := []string{
		"bigclaw-go/scripts/e2e/run_task_smoke.py",
		"bigclaw-go/scripts/e2e/export_validation_bundle.py",
		"bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py",
		"bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py",
		"bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py",
		"bigclaw-go/scripts/e2e/mixed_workload_matrix.py",
		"bigclaw-go/scripts/e2e/cross_process_coordination_surface.py",
		"bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py",
		"bigclaw-go/scripts/e2e/external_store_validation.py",
		"bigclaw-go/scripts/e2e/multi_node_shared_queue.py",
	}
	for _, needle := range disallowed {
		if strings.Contains(contents, needle) {
			t.Fatalf("docs/go-cli-script-migration.md should not reference removed Python helper %q", needle)
		}
	}
}

func TestRaySmokeEntrypointStaysNative(t *testing.T) {
	repoRoot := repoRoot(t)
	script := readRepoFile(t, repoRoot, "scripts/e2e/ray_smoke.sh")
	if !strings.Contains(script, `BIGCLAW_RAY_SMOKE_ENTRYPOINT:-echo hello from ray`) {
		t.Fatalf("scripts/e2e/ray_smoke.sh should default to a shell-native Ray smoke entrypoint")
	}
	if strings.Contains(script, `python -c "print('hello from ray')"`) {
		t.Fatalf("scripts/e2e/ray_smoke.sh should not default to a Python entrypoint")
	}

	doc := readRepoFile(t, repoRoot, "docs/e2e-validation.md")
	if !strings.Contains(doc, `export BIGCLAW_RAY_SMOKE_ENTRYPOINT='echo custom ray validation'`) {
		t.Fatalf("docs/e2e-validation.md should document the shell-native Ray smoke override")
	}
	if strings.Contains(doc, `export BIGCLAW_RAY_SMOKE_ENTRYPOINT='python -c "print(123)"'`) {
		t.Fatalf("docs/e2e-validation.md should not advertise a Python Ray smoke override")
	}
}

func TestActiveRayValidationEvidenceAvoidsPythonEntrypoints(t *testing.T) {
	repoRoot := repoRoot(t)
	paths := []string{
		"docs/reports/ray-live-jobs.json",
		"docs/reports/ray-live-smoke-report.json",
		"docs/reports/live-validation-index.json",
		"docs/reports/live-validation-summary.json",
		"docs/reports/live-validation-runs/20260316T140138Z/ray-live-smoke-report.json",
		"docs/reports/live-validation-runs/20260316T140138Z/summary.json",
		"docs/reports/live-validation-runs/20260316T140138Z/ray.stdout.log",
		"docs/reports/live-validation-runs/20260316T140138Z/ray.audit.jsonl",
	}

	for _, relativePath := range paths {
		contents := readRepoFile(t, repoRoot, relativePath)
		if strings.Contains(contents, `python -c "print('hello from ray')"`) {
			t.Fatalf("%s should not retain the removed Python Ray smoke entrypoint", relativePath)
		}
		if !strings.Contains(contents, "echo hello from ray") {
			t.Fatalf("%s should record the native Ray smoke entrypoint", relativePath)
		}
	}
}
