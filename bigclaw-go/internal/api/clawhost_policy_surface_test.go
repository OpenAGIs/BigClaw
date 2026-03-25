package api

import (
	"reflect"
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/policy"
)

func TestClawHostPolicySurfacePayloadHandlesEmptyScope(t *testing.T) {
	surface := clawHostPolicySurfacePayload([]domain.Task{
		{
			ID:     "non-clawhost-task",
			Source: "github",
		},
	}, "", "")

	if surface.Status != "catalog_only" {
		t.Fatalf("expected catalog-only empty surface, got %+v", surface)
	}
	if surface.Filters["team"] != "" || surface.Filters["project"] != "" {
		t.Fatalf("expected empty policy surface filters, got %+v", surface.Filters)
	}
	if surface.Summary.ActivePolicies != 0 || surface.Summary.ActiveTenants != 0 || surface.Summary.ActiveApps != 0 || surface.Summary.ReviewRequired != 0 || surface.Summary.TakeoverRequired != 0 || surface.Summary.OutOfPolicyDefaults != 0 || surface.Summary.BlockedDefaults != 0 {
		t.Fatalf("expected zeroed policy summary, got %+v", surface.Summary)
	}
	if len(surface.ReviewQueue) != 0 || len(surface.ObservedProviders) != 0 {
		t.Fatalf("expected no policy findings in empty surface, got %+v", surface)
	}
	if surface.Catalog.Integration != "clawhost" || !surface.Catalog.ParallelSafe {
		t.Fatalf("expected catalog metadata to persist in empty surface, got %+v", surface.Catalog)
	}
}

func TestRenderClawHostPolicySurfaceReportHandlesEmptyFilters(t *testing.T) {
	report := renderClawHostPolicySurfaceReport(clawHostPolicySurface{
		Integration: "clawhost",
		Status:      "active",
		Filters: map[string]string{
			"team":    "",
			"project": "",
		},
		Summary: clawHostPolicySurfaceSummary{
			ActivePolicies:      1,
			ActiveTenants:       1,
			ActiveApps:          1,
			ReviewRequired:      0,
			TakeoverRequired:    0,
			OutOfPolicyDefaults: 0,
			BlockedDefaults:     0,
		},
		ObservedProviders: []string{"openai"},
		ReviewQueue: []policy.ClawHostTenantPolicy{
			{
				TaskID:          "clawhost-policy-1",
				TenantID:        "tenant-a",
				AppID:           "sales-app",
				ProviderDefault: "openai",
				DriftStatus:     "aligned",
				Reason:          "provider default remains aligned with the shared app policy",
			},
		},
	})

	for _, want := range []string{
		"# ClawHost Policy Surface",
		"## Filters",
		"- Team: `none`",
		"- Project: `none`",
		"- Observed providers: `openai`",
		"- `clawhost-policy-1` tenant `tenant-a` app `sales-app` provider `openai` drift `aligned`",
		"Reason: provider default remains aligned with the shared app policy",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in policy report, got %s", want, report)
		}
	}
}

func TestFilterClawHostPolicyTasks(t *testing.T) {
	tasks := []domain.Task{
		{ID: "task-a", Metadata: map[string]string{"team": "platform", "project": "apollo"}},
		{ID: "task-b", Metadata: map[string]string{"team": "platform", "project": "beta"}},
		{ID: "task-c", Metadata: map[string]string{"team": "growth", "project": "apollo"}},
	}

	t.Run("team and project", func(t *testing.T) {
		filtered := filterClawHostPolicyTasks(tasks, "platform", "apollo")
		if len(filtered) != 1 || filtered[0].ID != "task-a" {
			t.Fatalf("expected only scoped policy task, got %+v", filtered)
		}
	})

	t.Run("project only", func(t *testing.T) {
		filtered := filterClawHostPolicyTasks(tasks, "", "apollo")
		if len(filtered) != 2 || filtered[0].ID != "task-a" || filtered[1].ID != "task-c" {
			t.Fatalf("expected project-only policy filter to preserve matching tasks, got %+v", filtered)
		}
	})
}

func TestClawHostPolicyHelpers(t *testing.T) {
	t.Run("drift rank", func(t *testing.T) {
		if clawHostDriftRank("blocked") != 0 || clawHostDriftRank("out_of_policy") != 1 || clawHostDriftRank("review_required") != 2 || clawHostDriftRank("aligned") != 3 {
			t.Fatalf("unexpected policy drift ranking")
		}
	})

	t.Run("sorted keys", func(t *testing.T) {
		got := sortedKeys(map[string]struct{}{"openai": {}, "anthropic": {}, "minimax": {}})
		want := []string{"anthropic", "minimax", "openai"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("expected sorted provider keys %v, got %v", want, got)
		}
	})

	t.Run("fallback", func(t *testing.T) {
		if got := clawHostPolicyFallback("", "none"); got != "none" {
			t.Fatalf("expected empty policy fallback to use default, got %q", got)
		}
		if got := clawHostPolicyFallback("openai", "none"); got != "openai" {
			t.Fatalf("expected non-empty policy fallback to preserve value, got %q", got)
		}
	})
}
