package billing

import (
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestBuildUsageAggregatesTeamsSeatsAndAlerts(t *testing.T) {
	tasks := []domain.Task{
		{
			State:       domain.TaskRunning,
			BudgetCents: 15000,
			Metadata: map[string]string{
				"team":       "platform",
				"plan":       " premium ",
				"owner":      "alice",
				"reviewer":   "bob",
				"created_by": "alice",
				"assignee":   "carol",
			},
		},
		{
			State:       domain.TaskBlocked,
			BudgetCents: 12000,
			Metadata: map[string]string{
				"team":       "growth",
				"plan":       "standard",
				"owner":      "dora",
				"reviewer":   "erin",
				"created_by": "frank",
				"assignee":   "erin",
			},
		},
		{
			State:       domain.TaskQueued,
			BudgetCents: 8000,
			Metadata: map[string]string{
				"team":       "platform",
				"plan":       "premium",
				"owner":      "alice",
				"reviewer":   "george",
				"created_by": "harry",
				"assignee":   "carol",
			},
		},
	}

	usage := BuildUsage(tasks, " OpenAGI ", "")
	if usage.Organization != "OpenAGI" || usage.Tier != "growth" {
		t.Fatalf("unexpected usage identity: %+v", usage)
	}
	if usage.SeatCount != 8 || usage.ActiveSeats != 8 {
		t.Fatalf("unexpected seat counts: %+v", usage)
	}
	if usage.BudgetCentsTotal != 35000 || usage.PremiumRuns != 2 || usage.StandardRuns != 1 {
		t.Fatalf("unexpected budget or run counters: %+v", usage)
	}
	if len(usage.ByTeam) != 2 {
		t.Fatalf("expected two team buckets, got %+v", usage.ByTeam)
	}
	if usage.ByTeam[0].Key != "platform" || usage.ByTeam[0].BudgetCentsTotal != 23000 || usage.ByTeam[0].SeatCount != 5 || usage.ByTeam[0].PremiumRuns != 2 {
		t.Fatalf("unexpected primary team usage bucket: %+v", usage.ByTeam[0])
	}
	if usage.ByTeam[1].Key != "growth" || usage.ByTeam[1].BudgetCentsTotal != 12000 || usage.ByTeam[1].SeatCount != 3 || usage.ByTeam[1].PremiumRuns != 0 {
		t.Fatalf("unexpected secondary team usage bucket: %+v", usage.ByTeam[1])
	}
	if got := strings.Join(usage.Alerts, " | "); got != "Premium orchestration usage is the majority of current workload. | Observed spend exceeds the current tier budget cap." {
		t.Fatalf("unexpected alerts: %q", got)
	}
}

func TestBuildUsageSortsEqualBudgetTeamsByKeyAndDefaultsOrganization(t *testing.T) {
	tasks := []domain.Task{
		{BudgetCents: 5000, Metadata: map[string]string{"team": "zeta", "owner": "alice"}},
		{BudgetCents: 5000, Metadata: map[string]string{"team": "alpha", "owner": "bob"}},
	}

	usage := BuildUsage(tasks, " ", "enterprise")
	if usage.Organization != "openagi" || usage.Tier != "enterprise" {
		t.Fatalf("unexpected default organization or tier: %+v", usage)
	}
	if len(usage.ByTeam) != 2 || usage.ByTeam[0].Key != "alpha" || usage.ByTeam[1].Key != "zeta" {
		t.Fatalf("expected equal-budget teams to sort by key, got %+v", usage.ByTeam)
	}
}

func TestEntitlementsForTierAndHelpers(t *testing.T) {
	t.Run("enterprise entitlements", func(t *testing.T) {
		ent := EntitlementsForTier(" Enterprise ")
		if ent.Tier != "enterprise" {
			t.Fatalf("expected normalized enterprise tier, got %+v", ent)
		}
		if !ent.Features["vm_pool"] || !ent.Features["flow_canvas"] || ent.Limits["concurrency_limit"] != 32 {
			t.Fatalf("unexpected enterprise entitlements: %+v", ent)
		}
		if got := strings.Join(ent.EnabledDashboards, ","); got != "billing,engineering,flows,operations,regression,triage" {
			t.Fatalf("expected sorted enterprise dashboards, got %q", got)
		}
	})

	t.Run("default helpers", func(t *testing.T) {
		if got := normalizeTier(" "); got != "growth" {
			t.Fatalf("expected blank tier to fall back to growth, got %q", got)
		}
		if got := firstNonEmpty("", "  ops ", "fallback"); got != "ops" {
			t.Fatalf("expected first non-empty trimmed value, got %q", got)
		}
		if got := firstNonEmpty("", " ", "\t"); got != "" {
			t.Fatalf("expected empty helper inputs to stay empty, got %q", got)
		}
		if got := strings.Join(collectUsers(domain.Task{Metadata: map[string]string{
			"owner":      "alice",
			"reviewer":   "bob",
			"created_by": "alice",
			"assignee":   "carol",
		}}), ","); got != "alice,bob,carol" {
			t.Fatalf("unexpected unique billing users: %q", got)
		}
	})

	t.Run("default entitlements", func(t *testing.T) {
		ent := EntitlementsForTier("starter")
		if ent.Tier != "starter" {
			t.Fatalf("expected unknown tier to stay normalized but unchanged, got %+v", ent)
		}
		if ent.Features["premium_orchestration"] || !ent.Features["billing_console"] || ent.Limits["max_agents"] != 2 {
			t.Fatalf("unexpected default entitlements: %+v", ent)
		}
		if got := strings.Join(ent.EnabledDashboards, ","); got != "engineering,operations" {
			t.Fatalf("expected default dashboards only, got %q", got)
		}
	})
}

func TestBuildAlertsIncludesNoSeatsFallback(t *testing.T) {
	alerts := buildAlerts(UsageSummary{Tier: "enterprise"})
	if got := strings.Join(alerts, " | "); got != "No billable seats detected in the current snapshot." {
		t.Fatalf("expected no-seat alert fallback, got %q", got)
	}
}
