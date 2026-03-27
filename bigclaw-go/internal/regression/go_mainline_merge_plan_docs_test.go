package regression

import (
	"strings"
	"testing"
)

func TestGoMainlineMergePlanDoc(t *testing.T) {
	repoRoot := repoRoot(t)
	contents := readRepoFile(t, repoRoot, "../docs/go-mainline-merge-plan.md")

	requiredSubstrings := []string{
		"# BigClaw Go Mainline Merge Plan",
		"`BIG-GO-910`",
		"`symphony/BIG-GOM-302`",
		"PR `#138`",
		"`BIG-GOM-301` through `BIG-GOM-309` as the first nine implementation and",
		"`BIG-GOM-310` as the handoff",
		"closeout slice that records final merge readiness",
		"`BIG-GOM-301` through `BIG-GOM-303`",
		"`BIG-GOM-304` through `BIG-GOM-306`",
		"`BIG-GOM-307` through `BIG-GOM-309`",
		"`cd bigclaw-go && go test ./...`",
		"`bash scripts/ops/bigclawctl legacy-python compile-check --json`",
		"`PYTHONPATH=src python3 -m pytest -q tests/test_legacy_shim.py`",
		"`bash scripts/ops/bigclawctl github-sync status --json`",
		"`bash scripts/ops/bigclawctl refill --local-issues local-issues.json`",
		"`docs/reports/parallel-validation-matrix.md`",
		"`docs/reports/parallel-follow-up-index.md`",
		"`symphony/BIG-GO-910`",
		"`BIG-GO-910: publish Go mainline merge and compatibility landing plan`",
	}

	for _, needle := range requiredSubstrings {
		if !strings.Contains(contents, needle) {
			t.Fatalf("../docs/go-mainline-merge-plan.md missing substring %q", needle)
		}
	}
}
