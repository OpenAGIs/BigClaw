package regression

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var bigGO187LegacyPythonReferencePattern = regexp.MustCompile("src/bigclaw/[^`\\s]+|scripts/[^`\\s]+\\.py|bigclaw-go/scripts/[^`\\s]+\\.py|python3|\\.py")

func TestBIGGO187CompactedLegacyPythonDocsStayWithinBudget(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	docChecks := []struct {
		path      string
		maxCount  int
		required  []string
		forbidden []string
	}{
		{
			path:     "docs/go-cli-script-migration-plan.md",
			maxCount: 10,
			required: []string{
				"`bigclaw-go/scripts/benchmark/{capacity_certification,capacity_certification_test,run_matrix,soak_local}.py`",
				"`bigclaw-go/scripts/e2e/{broker_failover_stub_matrix,broker_failover_stub_matrix_test,cross_process_coordination_surface,export_validation_bundle,export_validation_bundle_test,external_store_validation,mixed_workload_matrix,multi_node_shared_queue,multi_node_shared_queue_test,run_all_test,run_task_smoke,subscriber_takeover_fault_matrix,validation_bundle_continuation_policy_gate,validation_bundle_continuation_policy_gate_test,validation_bundle_continuation_scorecard}.py`",
				"`bigclaw-go/scripts/migration/{export_live_shadow_bundle,live_shadow_scorecard,shadow_compare,shadow_matrix}.py`",
				"`scripts/{create_issues,dev_smoke}.py`",
			},
			forbidden: []string{
				"`bigclaw-go/scripts/e2e/run_task_smoke.py`",
				"`bigclaw-go/scripts/migration/shadow_compare.py`",
			},
		},
		{
			path:     "docs/go-mainline-cutover-issue-pack.md",
			maxCount: 24,
			required: []string{
				"`src/bigclaw/{models,connectors,mapping,dsl}.py`",
				"`src/bigclaw/{runtime,scheduler,orchestration,workflow,queue}.py`",
				"`src/bigclaw/{github_sync,parallel_refill,workspace_bootstrap,workspace_bootstrap_cli,workspace_bootstrap_validation,service,__main__}.py`",
			},
			forbidden: []string{
				"`src/bigclaw/connectors.py`",
				"`src/bigclaw/orchestration.py`",
				"`src/bigclaw/workspace_bootstrap_validation.py`",
			},
		},
		{
			path:     "docs/go-mainline-cutover-handoff.md",
			maxCount: 3,
			required: []string{
				"`PYTHONPATH=src python3 - <<\"... legacy shim assertions ...\"`",
				"`src/bigclaw/{models,connectors,mapping,dsl}.py`",
				"`src/bigclaw/{governance,observability,operations,orchestration,pilot}.py`",
			},
			forbidden: []string{
				"`src/bigclaw/models.py`",
				"`src/bigclaw/pilot.py`",
			},
		},
	}

	for _, doc := range docChecks {
		content := readRepoFile(t, rootRepo, doc.path)
		matchCount := len(bigGO187LegacyPythonReferencePattern.FindAllString(content, -1))
		if matchCount > doc.maxCount {
			t.Fatalf("expected %s to stay within the compacted legacy Python reference budget (%d), found %d", doc.path, doc.maxCount, matchCount)
		}
		for _, needle := range doc.required {
			if !strings.Contains(content, needle) {
				t.Fatalf("expected %s to contain grouped legacy reference %q", doc.path, needle)
			}
		}
		for _, needle := range doc.forbidden {
			if strings.Contains(content, needle) {
				t.Fatalf("expected %s to avoid expanded legacy reference %q", doc.path, needle)
			}
		}
	}
}

func TestBIGGO187LaneReportCapturesDocReferenceSweep(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-187-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-187",
		"`docs/go-cli-script-migration-plan.md`: `31 -> 10` legacy Python references",
		"`docs/go-mainline-cutover-issue-pack.md`: `83 -> 24` legacy Python references",
		"`docs/go-mainline-cutover-handoff.md`: `10 -> 3` legacy Python references",
		"`bigclaw-go/scripts/benchmark/{capacity_certification,capacity_certification_test,run_matrix,soak_local}.py`",
		"`src/bigclaw/{models,connectors,mapping,dsl}.py`",
		"`src/bigclaw/{governance,observability,operations,orchestration,pilot}.py`",
		"`cd bigclaw-go && go test ./internal/regression -run 'TestBIGGO187' -count=1`",
		"`rg -n \"python3|\\\\.py\\\\b|#!/usr/bin/env python|#!/usr/bin/python\" docs bigclaw-go/internal bigclaw-go/cmd --glob '!bigclaw-go/internal/regression/**' --glob '!bigclaw-go/docs/reports/**' | head -n 200`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}

	if _, err := filepath.Abs(filepath.Join(rootRepo, "docs/go-mainline-cutover-issue-pack.md")); err != nil {
		t.Fatalf("expected repo root to resolve for BIG-GO-187 doc sweep: %v", err)
	}
}
