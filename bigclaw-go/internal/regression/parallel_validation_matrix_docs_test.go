package regression

import (
	"strings"
	"testing"
)

func TestParallelValidationMatrixDocsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	contents := readRepoFile(t, repoRoot, "docs/reports/parallel-validation-matrix.md")

	requiredSubstrings := []string{
		"# Parallel Validation Matrix (Local/Kubernetes/Ray)",
		"Baseline run: `20260316T140138Z`",
		"`cd bigclaw-go && BIGCLAW_E2E_RUN_LOCAL=1 BIGCLAW_E2E_RUN_KUBERNETES=0 BIGCLAW_E2E_RUN_RAY=0 ./scripts/e2e/run_all.sh`",
		"`cd bigclaw-go && BIGCLAW_E2E_RUN_LOCAL=0 BIGCLAW_E2E_RUN_KUBERNETES=1 BIGCLAW_E2E_RUN_RAY=0 ./scripts/e2e/run_all.sh`; `cd bigclaw-go && ./scripts/e2e/kubernetes_smoke.sh`",
		"`cd bigclaw-go && BIGCLAW_E2E_RUN_LOCAL=0 BIGCLAW_E2E_RUN_KUBERNETES=0 BIGCLAW_E2E_RUN_RAY=1 ./scripts/e2e/run_all.sh`; `cd bigclaw-go && ./scripts/e2e/ray_smoke.sh`",
		"`bigclaw-go/docs/reports/sqlite-smoke-report.json`; `bigclaw-go/docs/reports/live-validation-summary.json`; `bigclaw-go/docs/reports/live-validation-index.md`",
		"`bigclaw-go/docs/reports/kubernetes-live-smoke-report.json`; `bigclaw-go/docs/reports/kubernetes-live-resources.txt`; `bigclaw-go/docs/reports/live-validation-summary.json`",
		"`bigclaw-go/docs/reports/ray-live-smoke-report.json`; `bigclaw-go/docs/reports/ray-live-jobs.json`; `bigclaw-go/docs/reports/live-validation-summary.json`",
		"`bigclaw-go/docs/reports/multi-node-shared-queue-report.json`",
		"`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-report.json`",
		"`bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json`",
		"`bigclaw-go/docs/reports/cross-process-coordination-boundary-digest.md`",
		"`bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json`",
		"docs/reports/parallel-follow-up-index.md",
		"remaining takeover, coordination, continuation, and broker-durability",
		"OPE-269` / `BIG-PAR-080",
		"OPE-261` / `BIG-PAR-085",
		"OPE-271` / `BIG-PAR-082",
		"OPE-222`",
	}

	for _, needle := range requiredSubstrings {
		if !strings.Contains(contents, needle) {
			t.Fatalf("docs/reports/parallel-validation-matrix.md missing substring %q", needle)
		}
	}
}
