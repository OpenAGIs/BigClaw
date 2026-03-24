package product

import (
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestNavigationIncludesCoreConsoleSections(t *testing.T) {
	sections := Navigation()
	if len(sections) != 4 {
		t.Fatalf("expected four top-level nav sections, got %+v", sections)
	}
	if sections[0].Key != "overview" || sections[1].Key != "operations" || sections[2].Key != "delivery" || sections[3].Key != "business" {
		t.Fatalf("unexpected nav section order: %+v", sections)
	}
	if sections[1].Items[1].Path != "/v2/control-center" || sections[1].Items[4].Path != "/v2/saved-views" {
		t.Fatalf("expected operations routes in nav, got %+v", sections[1].Items)
	}
}

func TestHomeForRoleUsesRoleSpecificCards(t *testing.T) {
	tasks := []domain.Task{
		{State: domain.TaskBlocked, RiskLevel: domain.RiskHigh, BudgetCents: 1250, Metadata: map[string]string{"plan": "premium", "regression": "true", "flow_id": "flow-1", "department": "support"}},
		{State: domain.TaskRunning, BudgetCents: 2300, Metadata: map[string]string{"plan": "premium", "regression_count": "2"}},
		{State: domain.TaskDeadLetter, BudgetCents: 800, Metadata: map[string]string{"flow_id": "flow-2"}},
		{State: domain.TaskSucceeded, BudgetCents: 650},
	}

	cases := []struct {
		role      string
		wantRole  string
		wantFirst string
		wantValue int
	}{
		{role: " ENG_LEAD ", wantRole: "eng_lead", wantFirst: "blockers", wantValue: 1},
		{role: "platform_admin", wantRole: "platform_admin", wantFirst: "queue", wantValue: 1},
		{role: "vp_eng", wantRole: "vp_eng", wantFirst: "throughput", wantValue: 1},
		{role: "", wantRole: "viewer", wantFirst: "active", wantValue: 1},
	}

	for _, tc := range cases {
		home := HomeForRole(tc.role, tasks)
		if home.Role != tc.wantRole {
			t.Fatalf("normalized role = %q, want %q", home.Role, tc.wantRole)
		}
		if len(home.Cards) != 3 || home.Cards[0].Key != tc.wantFirst || home.Cards[0].Value != tc.wantValue {
			t.Fatalf("unexpected home cards for %s: %+v", tc.role, home.Cards)
		}
	}
}

func TestDefaultDesignSystemAndConsoleHelpers(t *testing.T) {
	system := DefaultDesignSystem()
	if !system.DarkMode || !system.Responsive {
		t.Fatalf("expected console design system to be responsive dark mode, got %+v", system)
	}
	if system.Tokens["accent.primary"] != "#4f46e5" || system.Components[0].Key != "flow-canvas" || system.Components[len(system.Components)-1].Key != "timeline" {
		t.Fatalf("unexpected design system defaults: %+v", system)
	}

	tasks := []domain.Task{
		{State: domain.TaskBlocked, RiskLevel: domain.RiskHigh, BudgetCents: 100, Metadata: map[string]string{"plan": "premium", "regression_count": "1", "flow_id": "flow-1", "department": "support"}},
		{State: domain.TaskQueued, BudgetCents: 200, Metadata: map[string]string{"plan": "standard"}},
		{State: domain.TaskLeased, BudgetCents: 300},
		{State: domain.TaskRetrying, BudgetCents: 400},
		{State: domain.TaskDeadLetter, BudgetCents: 500},
		{State: domain.TaskSucceeded, BudgetCents: 600},
	}

	counts := aggregate(tasks)
	if counts["blocked"] != 1 || counts["active"] != 3 || counts["dead_letter"] != 1 || counts["premium"] != 1 || counts["high_risk"] != 1 || counts["regression"] != 1 || counts["succeeded"] != 1 || counts["budget"] != 2100 || counts["flow"] != 1 || counts["support"] != 1 {
		t.Fatalf("unexpected aggregate counts: %+v", counts)
	}

	if got := normalizeRole("  VP_ENG "); got != "vp_eng" {
		t.Fatalf("normalizeRole trims and lowers = %q, want %q", got, "vp_eng")
	}
	if got := normalizeRole("   "); got != "viewer" {
		t.Fatalf("normalizeRole blank fallback = %q, want %q", got, "viewer")
	}

	summary := SummaryText(Home{Cards: []HomeCard{{Key: "active", Value: 3}, {Key: "blocked", Value: 1}}})
	if summary != "active=3, blocked=1" {
		t.Fatalf("unexpected summary text: %s", summary)
	}
	if strings.Contains(summary, "  ") {
		t.Fatalf("unexpected spacing in summary text: %q", summary)
	}
}
