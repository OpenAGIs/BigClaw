package regression

import (
	"strings"
	"testing"
)

func TestE2EEntrypointDocsStayGoOnly(t *testing.T) {
	root := repoRoot(t)
	activeSurfaces := map[string]string{
		"repo README":             readRepoFile(t, root, "../README.md"),
		"bigclaw-go README":       readRepoFile(t, root, "README.md"),
		"e2e validation guide":    readRepoFile(t, root, "docs/e2e-validation.md"),
		"GitHub Actions workflow": readRepoFile(t, root, "../.github/workflows/ci.yml"),
	}

	retiredPythonEntrypoints := []string{
		"run_task_smoke.py",
		"export_validation_bundle.py",
		"validation_bundle_continuation_scorecard.py",
		"validation_bundle_continuation_policy_gate.py",
		"broker_failover_stub_matrix.py",
		"mixed_workload_matrix.py",
		"cross_process_coordination_surface.py",
		"subscriber_takeover_fault_matrix.py",
		"external_store_validation.py",
		"multi_node_shared_queue.py",
	}

	for surface, body := range activeSurfaces {
		for _, retired := range retiredPythonEntrypoints {
			if strings.Contains(body, retired) {
				t.Fatalf("%s must not reference retired e2e Python entrypoint %q", surface, retired)
			}
		}
	}
}
