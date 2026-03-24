package api

import (
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
