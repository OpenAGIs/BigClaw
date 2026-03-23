package regression

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRollbackDocsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/reports/rollback-safeguard-follow-up-digest.md",
			substrings: []string{
				"OPE-254` / `BIG-PAR-088",
				"## Tenant-Scoped Trigger Surface",
				"Pause the tenant rollout segment",
				"rollback-trigger-surface.json",
				"manual, evidence-backed operator action",
				"no tenant-scoped automated rollback trigger",
			},
		},
		{
			path: "docs/migration.md",
			substrings: []string{
				"tenant-scoped trigger surface",
				"rollback remains operator-driven",
				"visibility-only and does not execute rollback automatically",
			},
		},
		{
			path: "docs/reports/migration-plan-review-notes.md",
			substrings: []string{
				"tenant-scoped trigger surface",
				"rollback-trigger-surface.json",
				"OPE-254` / `BIG-PAR-088",
			},
		},
		{
			path: "docs/reports/migration-readiness-report.md",
			substrings: []string{
				"current trigger surface and manual rollback guardrails",
				"rollback-trigger-surface.json",
				"OPE-254` / `BIG-PAR-088",
				"GET /debug/status` rollback trigger payload",
				"GET /v2/control-center` migration review rollback trigger payload",
				"`rollback_trigger_surface`",
				"`distributed_diagnostics.migration_review_pack.rollback_trigger_surface`",
			},
		},
		{
			path: "docs/reports/review-readiness.md",
			substrings: []string{
				"rollback safeguard trigger surface",
				"rollback-trigger-surface.json",
				"OPE-254` / `BIG-PAR-088",
				"GET /debug/status",
				"distributed_diagnostics.migration_review_pack.rollback_trigger_surface",
			},
		},
		{
			path: "docs/reports/issue-coverage.md",
			substrings: []string{
				"rollback safeguard trigger surfaces",
				"rollback-trigger-surface.json",
				"OPE-254` / `BIG-PAR-088",
			},
		},
		{
			path: "../docs/openclaw-parallel-gap-analysis.md",
			substrings: []string{
				"Distributed diagnostics follow-up digests",
				"Migration follow-up digests",
				"OPE-254` / `BIG-PAR-088",
				"rollback-safeguard-follow-up-digest.md",
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

	triggerSurface := readRollbackTriggerSurface(t, repoRoot)
	if got := triggerSurface.Summary.Status; got != "manual-review-required" {
		t.Fatalf("unexpected rollback trigger status: %s", got)
	}
	if triggerSurface.Summary.AutomationBoundary != "manual_only" {
		t.Fatalf("unexpected automation boundary: %s", triggerSurface.Summary.AutomationBoundary)
	}
	if triggerSurface.Summary.AutomatedRollbackTrigger {
		t.Fatal("rollback trigger surface must not claim automated rollback execution")
	}
	if len(triggerSurface.Warnings) != 1 || len(triggerSurface.Blockers) != 3 || len(triggerSurface.ManualOnlyPaths) != 2 {
		t.Fatalf("unexpected trigger distinctions: warnings=%d blockers=%d manual_only=%d", len(triggerSurface.Warnings), len(triggerSurface.Blockers), len(triggerSurface.ManualOnlyPaths))
	}
	if triggerSurface.SharedGuardrailSummary.LiveShadowIndexPath != "docs/reports/live-shadow-index.md" {
		t.Fatalf("unexpected live shadow index path: %s", triggerSurface.SharedGuardrailSummary.LiveShadowIndexPath)
	}
	if triggerSurface.SharedGuardrailSummary.LiveShadowRollupPath != "docs/reports/live-shadow-drift-rollup.json" {
		t.Fatalf("unexpected live shadow rollup path: %s", triggerSurface.SharedGuardrailSummary.LiveShadowRollupPath)
	}
}

type rollbackTriggerSurface struct {
	Summary struct {
		Status                   string `json:"status"`
		AutomationBoundary       string `json:"automation_boundary"`
		AutomatedRollbackTrigger bool   `json:"automated_rollback_trigger"`
	} `json:"summary"`
	SharedGuardrailSummary struct {
		LiveShadowIndexPath  string `json:"live_shadow_index_path"`
		LiveShadowRollupPath string `json:"live_shadow_rollup_path"`
	} `json:"shared_guardrail_summary"`
	Warnings        []map[string]any `json:"warnings"`
	Blockers        []map[string]any `json:"blockers"`
	ManualOnlyPaths []map[string]any `json:"manual_only_paths"`
}

func readRollbackTriggerSurface(t *testing.T, root string) rollbackTriggerSurface {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(root, "docs/reports/rollback-trigger-surface.json"))
	if err != nil {
		t.Fatalf("read rollback trigger surface: %v", err)
	}
	var payload rollbackTriggerSurface
	if err := json.Unmarshal(contents, &payload); err != nil {
		t.Fatalf("parse rollback trigger surface: %v", err)
	}
	return payload
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

func readRepoFile(t *testing.T, root string, relative string) string {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(root, relative))
	if err != nil {
		t.Fatalf("read %s: %v", relative, err)
	}
	return string(contents)
}
