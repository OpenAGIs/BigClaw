package api

import (
	"strings"
	"testing"

	"bigclaw-go/internal/policy"
)

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
