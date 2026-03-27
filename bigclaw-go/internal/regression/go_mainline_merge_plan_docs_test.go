package regression

import (
	"strings"
	"testing"
)

func TestGoMainlineMergePlanDoc(t *testing.T) {
	repoRoot := repoRoot(t)
	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "../docs/go-mainline-merge-plan.md",
			substrings: []string{
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
				"`origin/BIG-GO-903`",
				"`origin/big-go-905`",
				"`origin/codex/BIG-GO-906-runtime-scheduler-orchestration-migration`",
				"`origin/symphony/BIG-GO-908`",
				"`origin/symphony/BIG-GO-909`",
				"`BIG-GO-901`, `BIG-GO-902`, `BIG-GO-904`, and `BIG-GO-907` were",
				"`symphony/BIG-GO-910`",
				"`BIG-GO-910: publish Go mainline merge and compatibility landing plan`",
				"`.github/workflows/ci.yml` currently runs Python lint/test/build only",
			},
		},
		{
			path: "../docs/go-mainline-cutover-handoff.md",
			substrings: []string{
				"`docs/go-mainline-merge-plan.md` records the forward-only compatibility",
			},
		},
		{
			path: "../docs/go-mainline-cutover-issue-pack.md",
			substrings: []string{
				"`docs/go-mainline-merge-plan.md` as the current executable compatibility",
			},
		},
	}

	for _, tc := range cases {
		contents := readRepoFile(t, repoRoot, tc.path)
		for _, needle := range tc.substrings {
			if !strings.Contains(contents, needle) {
				t.Fatalf("%s missing substring %q", tc.path, needle)
			}
		}
	}
}
