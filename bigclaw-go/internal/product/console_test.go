package product

import (
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestNavigationProvidesExpectedSections(t *testing.T) {
	sections := Navigation()
	if len(sections) != 4 {
		t.Fatalf("expected 4 navigation sections, got %+v", sections)
	}
	if sections[0].Key != "overview" || sections[0].Items[0].Path != "/v2/home" {
		t.Fatalf("unexpected overview section: %+v", sections[0])
	}
	if sections[3].Key != "business" || sections[3].Items[len(sections[3].Items)-1].Path != "/v2/design-system" {
		t.Fatalf("unexpected business section: %+v", sections[3])
	}
}

func TestHomeForRoleUsesRoleSpecificCards(t *testing.T) {
	tasks := []domain.Task{
		{State: domain.TaskBlocked, RiskLevel: domain.RiskHigh, BudgetCents: 10, Metadata: map[string]string{"plan": "premium", "regression": "true", "flow_id": "flow-1", "department": "support"}},
		{State: domain.TaskRunning, BudgetCents: 20, Metadata: map[string]string{"regression_count": "2"}},
		{State: domain.TaskDeadLetter},
		{State: domain.TaskSucceeded},
	}

	if home := HomeForRole("eng_lead", tasks); home.Role != "eng_lead" || home.Cards[0].Key != "blockers" || home.Cards[0].Value != 1 {
		t.Fatalf("unexpected eng lead home: %+v", home)
	}
	if home := HomeForRole("platform_admin", tasks); home.Cards[0].Key != "queue" || home.Cards[0].Value != 1 || home.Cards[1].Value != 1 || home.Cards[2].Value != 1 {
		t.Fatalf("unexpected platform admin home: %+v", home)
	}
	if home := HomeForRole("vp_eng", tasks); home.Cards[0].Key != "throughput" || home.Cards[0].Value != 1 || home.Cards[1].Value != 1 || home.Cards[2].Value != 30 {
		t.Fatalf("unexpected vp eng home: %+v", home)
	}
	if home := HomeForRole("", tasks); home.Role != "viewer" || home.Cards[0].Key != "active" || home.Cards[0].Value != 1 || home.Cards[1].Value != 1 || home.Cards[2].Value != 1 {
		t.Fatalf("unexpected default viewer home: %+v", home)
	}
}

func TestDefaultDesignSystemIsSortedAndResponsive(t *testing.T) {
	system := DefaultDesignSystem()
	if !system.DarkMode || !system.Responsive {
		t.Fatalf("expected default design system flags, got %+v", system)
	}
	if len(system.Components) != 4 {
		t.Fatalf("expected 4 component specs, got %+v", system.Components)
	}
	if system.Components[0].Key != "flow-canvas" || system.Components[len(system.Components)-1].Key != "timeline" {
		t.Fatalf("expected component keys to be sorted, got %+v", system.Components)
	}
	if system.Tokens["accent.primary"] != "#4f46e5" || system.Tokens["fg.default"] != "#f5f7fa" {
		t.Fatalf("unexpected design tokens: %+v", system.Tokens)
	}
}

func TestConsoleHelpersAggregateNormalizeAndSummarize(t *testing.T) {
	tasks := []domain.Task{
		{State: domain.TaskBlocked, RiskLevel: domain.RiskHigh, BudgetCents: 11, Metadata: map[string]string{"plan": "premium", "regression": "true", "flow_id": "flow-1", "department": "support"}},
		{State: domain.TaskQueued, BudgetCents: 13},
		{State: domain.TaskLeased},
		{State: domain.TaskRunning},
		{State: domain.TaskRetrying},
		{State: domain.TaskDeadLetter},
		{State: domain.TaskSucceeded},
	}
	counts := aggregate(tasks)
	if counts["blocked"] != 1 || counts["active"] != 4 || counts["dead_letter"] != 1 || counts["premium"] != 1 || counts["high_risk"] != 1 || counts["regression"] != 1 || counts["succeeded"] != 1 || counts["budget"] != 24 || counts["flow"] != 1 || counts["support"] != 1 {
		t.Fatalf("unexpected aggregate counts: %+v", counts)
	}
	if got := normalizeRole(" VP_ENG "); got != "vp_eng" {
		t.Fatalf("expected normalized role vp_eng, got %q", got)
	}
	if got := normalizeRole("   "); got != "viewer" {
		t.Fatalf("expected blank role to normalize to viewer, got %q", got)
	}
	summary := SummaryText(Home{Cards: []HomeCard{{Key: "alpha", Value: 1}, {Key: "beta", Value: 2}}})
	if !strings.Contains(summary, "alpha=1") || !strings.Contains(summary, "beta=2") {
		t.Fatalf("unexpected summary text: %q", summary)
	}
}
