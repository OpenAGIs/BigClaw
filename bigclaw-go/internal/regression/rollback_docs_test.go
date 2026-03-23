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
	if triggerSurface.ReviewerPath.IndexPath != "docs/reports/live-shadow-index.md" {
		t.Fatalf("unexpected reviewer index path: %s", triggerSurface.ReviewerPath.IndexPath)
	}
	if triggerSurface.ReviewerPath.DigestPath != "docs/reports/rollback-safeguard-follow-up-digest.md" {
		t.Fatalf("unexpected reviewer digest path: %s", triggerSurface.ReviewerPath.DigestPath)
	}
	if triggerSurface.ReviewerPath.DigestIssue.ID != "OPE-254" || triggerSurface.ReviewerPath.DigestIssue.Slug != "BIG-PAR-088" {
		t.Fatalf("unexpected reviewer digest issue: %+v", triggerSurface.ReviewerPath.DigestIssue)
	}

	liveShadowTriggerSurface := readLiveShadowRollbackTriggerSurface(t, repoRoot)
	if liveShadowTriggerSurface.ReviewerPath.IndexPath != "docs/reports/live-shadow-index.md" {
		t.Fatalf("unexpected live-shadow reviewer index path: %s", liveShadowTriggerSurface.ReviewerPath.IndexPath)
	}
	if liveShadowTriggerSurface.ReviewerPath.DigestPath != "docs/reports/rollback-safeguard-follow-up-digest.md" {
		t.Fatalf("unexpected live-shadow reviewer digest path: %s", liveShadowTriggerSurface.ReviewerPath.DigestPath)
	}
	if liveShadowTriggerSurface.ReviewerPath.DigestIssue.ID != "OPE-254" || liveShadowTriggerSurface.ReviewerPath.DigestIssue.Slug != "BIG-PAR-088" {
		t.Fatalf("unexpected live-shadow reviewer digest issue: %+v", liveShadowTriggerSurface.ReviewerPath.DigestIssue)
	}

	liveShadowIndexSummary := readLiveShadowSummary(t, repoRoot, "docs/reports/live-shadow-index.json")
	assertLiveShadowRollbackSummary(t, liveShadowIndexSummary)

	liveShadowSummary := readLiveShadowSummary(t, repoRoot, "docs/reports/live-shadow-summary.json")
	assertLiveShadowRollbackSummary(t, liveShadowSummary)

	liveShadowBundleSummary := readLiveShadowSummary(t, repoRoot, "docs/reports/live-shadow-runs/20260313T085655Z/summary.json")
	assertLiveShadowRollbackSummary(t, liveShadowBundleSummary)
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
	ReviewerPath struct {
		IndexPath   string `json:"index_path"`
		DigestPath  string `json:"digest_path"`
		DigestIssue struct {
			ID   string `json:"id"`
			Slug string `json:"slug"`
		} `json:"digest_issue"`
	} `json:"reviewer_path"`
	Warnings        []map[string]any `json:"warnings"`
	Blockers        []map[string]any `json:"blockers"`
	ManualOnlyPaths []map[string]any `json:"manual_only_paths"`
}

type liveShadowSummary struct {
	RollbackTriggerSurface struct {
		Status                   string `json:"status"`
		AutomationBoundary       string `json:"automation_boundary"`
		AutomatedRollbackTrigger bool   `json:"automated_rollback_trigger"`
		Distinctions             struct {
			Blockers        int `json:"blockers"`
			Warnings        int `json:"warnings"`
			ManualOnlyPaths int `json:"manual_only_paths"`
		} `json:"distinctions"`
		Issue struct {
			ID   string `json:"id"`
			Slug string `json:"slug"`
		} `json:"issue"`
		DigestPath string `json:"digest_path"`
		SummaryPath string `json:"summary_path"`
	} `json:"rollback_trigger_surface"`
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

func readLiveShadowRollbackTriggerSurface(t *testing.T, root string) rollbackTriggerSurface {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(root, "docs/reports/live-shadow-runs/20260313T085655Z/rollback-trigger-surface.json"))
	if err != nil {
		t.Fatalf("read live-shadow rollback trigger surface: %v", err)
	}
	var payload rollbackTriggerSurface
	if err := json.Unmarshal(contents, &payload); err != nil {
		t.Fatalf("parse live-shadow rollback trigger surface: %v", err)
	}
	return payload
}

func readLiveShadowSummary(t *testing.T, root string, relative string) liveShadowSummary {
	t.Helper()
	contents, err := os.ReadFile(filepath.Join(root, relative))
	if err != nil {
		t.Fatalf("read %s: %v", relative, err)
	}
	var payload liveShadowSummary
	if err := json.Unmarshal(contents, &payload); err != nil {
		t.Fatalf("parse %s: %v", relative, err)
	}
	if payload.RollbackTriggerSurface.Status == "" {
		var wrapped struct {
			Latest liveShadowSummary `json:"latest"`
		}
		if err := json.Unmarshal(contents, &wrapped); err != nil {
			t.Fatalf("parse wrapped %s: %v", relative, err)
		}
		if wrapped.Latest.RollbackTriggerSurface.Status != "" {
			return wrapped.Latest
		}
	}
	return payload
}

func assertLiveShadowRollbackSummary(t *testing.T, payload liveShadowSummary) {
	t.Helper()
	if payload.RollbackTriggerSurface.Status != "manual-review-required" {
		t.Fatalf("unexpected live-shadow rollback status: %s", payload.RollbackTriggerSurface.Status)
	}
	if payload.RollbackTriggerSurface.AutomationBoundary != "manual_only" {
		t.Fatalf("unexpected live-shadow rollback automation boundary: %s", payload.RollbackTriggerSurface.AutomationBoundary)
	}
	if payload.RollbackTriggerSurface.AutomatedRollbackTrigger {
		t.Fatal("live-shadow rollback summary must not claim automated rollback execution")
	}
	if payload.RollbackTriggerSurface.Distinctions.Blockers != 3 || payload.RollbackTriggerSurface.Distinctions.Warnings != 1 || payload.RollbackTriggerSurface.Distinctions.ManualOnlyPaths != 2 {
		t.Fatalf("unexpected live-shadow rollback distinctions: %+v", payload.RollbackTriggerSurface.Distinctions)
	}
	if payload.RollbackTriggerSurface.Issue.ID != "OPE-254" || payload.RollbackTriggerSurface.Issue.Slug != "BIG-PAR-088" {
		t.Fatalf("unexpected live-shadow rollback issue: %+v", payload.RollbackTriggerSurface.Issue)
	}
	if payload.RollbackTriggerSurface.DigestPath != "docs/reports/rollback-safeguard-follow-up-digest.md" {
		t.Fatalf("unexpected live-shadow rollback digest path: %s", payload.RollbackTriggerSurface.DigestPath)
	}
	if payload.RollbackTriggerSurface.SummaryPath != "docs/reports/rollback-trigger-surface.json" {
		t.Fatalf("unexpected live-shadow rollback summary path: %s", payload.RollbackTriggerSurface.SummaryPath)
	}
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
