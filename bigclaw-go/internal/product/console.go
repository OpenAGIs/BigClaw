package product

import (
	"fmt"
	"sort"
	"strings"

	"bigclaw-go/internal/domain"
)

type NavItem struct {
	Key   string   `json:"key"`
	Label string   `json:"label"`
	Path  string   `json:"path"`
	Roles []string `json:"roles,omitempty"`
}

type NavSection struct {
	Key   string    `json:"key"`
	Label string    `json:"label"`
	Items []NavItem `json:"items"`
}

type HomeCard struct {
	Key      string `json:"key"`
	Title    string `json:"title"`
	Value    int    `json:"value"`
	Subtitle string `json:"subtitle,omitempty"`
	Path     string `json:"path,omitempty"`
}

type Home struct {
	Role  string     `json:"role"`
	Cards []HomeCard `json:"cards"`
}

type ComponentSpec struct {
	Key        string   `json:"key"`
	Category   string   `json:"category"`
	States     []string `json:"states,omitempty"`
	Responsive bool     `json:"responsive"`
	DarkMode   bool     `json:"dark_mode"`
}

type DesignSystem struct {
	Tokens     map[string]string `json:"tokens"`
	Components []ComponentSpec   `json:"components"`
	DarkMode   bool              `json:"dark_mode"`
	Responsive bool              `json:"responsive"`
}

func Navigation() []NavSection {
	return []NavSection{
		{Key: "overview", Label: "Overview", Items: []NavItem{{Key: "home", Label: "Home", Path: "/v2/home"}, {Key: "dashboard", Label: "Engineering Dashboard", Path: "/v2/dashboard/engineering"}, {Key: "operations", Label: "Operations Dashboard", Path: "/v2/dashboard/operations"}}},
		{Key: "operations", Label: "Operations", Items: []NavItem{{Key: "runs", Label: "Runs", Path: "/v2/runs"}, {Key: "scheduler", Label: "Scheduler", Path: "/v2/control-center", Roles: []string{"eng_lead", "platform_admin", "cross_team_operator"}}, {Key: "triage", Label: "Triage", Path: "/v2/triage/center"}, {Key: "regression", Label: "Regression", Path: "/v2/regression/center", Roles: []string{"eng_lead", "platform_admin", "vp_eng"}}}},
		{Key: "delivery", Label: "Flows", Items: []NavItem{{Key: "flows", Label: "Flows", Path: "/v2/flows/overview", Roles: []string{"platform_admin", "cross_team_operator"}}, {Key: "canvas", Label: "Canvas", Path: "/v2/flows/templates", Roles: []string{"platform_admin", "cross_team_operator"}}, {Key: "reports", Label: "Weekly Reports", Path: "/v2/reports/weekly"}}},
		{Key: "business", Label: "Business", Items: []NavItem{{Key: "billing", Label: "Billing", Path: "/v2/billing/usage", Roles: []string{"platform_admin", "vp_eng", "cross_team_operator"}}, {Key: "entitlements", Label: "Entitlements", Path: "/v2/billing/entitlements", Roles: []string{"platform_admin", "vp_eng", "cross_team_operator"}}, {Key: "settings", Label: "Settings", Path: "/v2/design-system"}}},
	}
}

func NavigationForRole(role string) []NavSection {
	role = normalizeRole(role)
	sections := Navigation()
	filtered := make([]NavSection, 0, len(sections))
	for _, section := range sections {
		items := make([]NavItem, 0, len(section.Items))
		for _, item := range section.Items {
			if navItemAllowedForRole(item, role) {
				items = append(items, item)
			}
		}
		if len(items) == 0 {
			continue
		}
		filtered = append(filtered, NavSection{Key: section.Key, Label: section.Label, Items: items})
	}
	return filtered
}

func HomeForRole(role string, tasks []domain.Task, activeTakeovers int) Home {
	role = normalizeRole(role)
	counts := aggregate(tasks)
	home := Home{Role: role}
	switch role {
	case "eng_lead":
		home.Cards = []HomeCard{{Key: "blockers", Title: "Blockers", Value: counts["blocked"], Subtitle: "Open blocked engineering runs", Path: "/v2/dashboard/engineering"}, {Key: "takeovers", Title: "Takeovers", Value: activeTakeovers, Subtitle: "Runs requiring human ownership", Path: "/v2/control-center"}, {Key: "regressions", Title: "Regressions", Value: counts["regression"], Subtitle: "Regression findings this week", Path: "/v2/regression/center"}}
	case "platform_admin":
		home.Cards = []HomeCard{{Key: "queue", Title: "Queue Depth", Value: counts["active"], Subtitle: "Queued or running work", Path: "/v2/control-center"}, {Key: "deadletters", Title: "Dead Letters", Value: counts["dead_letter"], Subtitle: "Runs requiring replay", Path: "/v2/control-center"}, {Key: "cost", Title: "Premium Runs", Value: counts["premium"], Subtitle: "Premium orchestration usage", Path: "/v2/billing/usage"}}
	case "vp_eng":
		home.Cards = []HomeCard{{Key: "throughput", Title: "Completed Runs", Value: counts["succeeded"], Subtitle: "Completed delivery runs", Path: "/v2/dashboard/operations"}, {Key: "risk", Title: "High Risk", Value: counts["high_risk"], Subtitle: "High-risk tasks under review", Path: "/v2/dashboard/operations"}, {Key: "spend", Title: "Tracked Spend", Value: counts["budget"], Subtitle: "Budget cents represented in current snapshot", Path: "/v2/billing/usage"}}
	default:
		home.Cards = []HomeCard{{Key: "active", Title: "Active Runs", Value: counts["active"], Subtitle: "Currently active delivery work", Path: "/v2/dashboard/engineering"}, {Key: "flows", Title: "Flow Nodes", Value: counts["flow"], Subtitle: "Cross-functional flow tasks", Path: "/v2/flows/overview"}, {Key: "support", Title: "Support Handoffs", Value: counts["support"], Subtitle: "Support readiness tasks", Path: "/v2/support/handoff"}}
	}
	return home
}

func DefaultDesignSystem() DesignSystem {
	components := []ComponentSpec{
		{Key: "status-badge", Category: "feedback", States: []string{"default", "success", "warning", "critical"}, Responsive: true, DarkMode: true},
		{Key: "timeline", Category: "data-display", States: []string{"compact", "expanded"}, Responsive: true, DarkMode: true},
		{Key: "flow-canvas", Category: "workflow", States: []string{"readonly", "editable"}, Responsive: true, DarkMode: true},
		{Key: "metric-card", Category: "dashboard", States: []string{"default", "empty"}, Responsive: true, DarkMode: true},
	}
	sort.SliceStable(components, func(i, j int) bool { return components[i].Key < components[j].Key })
	return DesignSystem{
		Tokens:     map[string]string{"bg.surface": "#101418", "bg.canvas": "#0b0f13", "fg.default": "#f5f7fa", "fg.muted": "#94a3b8", "accent.primary": "#4f46e5", "accent.success": "#10b981", "accent.warning": "#f59e0b", "accent.critical": "#ef4444"},
		Components: components,
		DarkMode:   true,
		Responsive: true,
	}
}

func aggregate(tasks []domain.Task) map[string]int {
	out := map[string]int{"blocked": 0, "active": 0, "dead_letter": 0, "premium": 0, "high_risk": 0, "regression": 0, "succeeded": 0, "budget": 0, "flow": 0, "support": 0}
	for _, task := range tasks {
		switch task.State {
		case domain.TaskBlocked:
			out["blocked"]++
		case domain.TaskQueued, domain.TaskLeased, domain.TaskRunning, domain.TaskRetrying:
			out["active"]++
		case domain.TaskDeadLetter:
			out["dead_letter"]++
		case domain.TaskSucceeded:
			out["succeeded"]++
		}
		if strings.EqualFold(strings.TrimSpace(task.Metadata["plan"]), "premium") {
			out["premium"]++
		}
		if task.RiskLevel == domain.RiskHigh {
			out["high_risk"]++
		}
		if strings.TrimSpace(task.Metadata["regression_count"]) != "" || strings.EqualFold(strings.TrimSpace(task.Metadata["regression"]), "true") {
			out["regression"]++
		}
		out["budget"] += int(task.BudgetCents)
		if strings.TrimSpace(task.Metadata["flow_id"]) != "" {
			out["flow"]++
		}
		if strings.EqualFold(strings.TrimSpace(task.Metadata["department"]), "support") {
			out["support"]++
		}
	}
	return out
}

func normalizeRole(role string) string {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		return "viewer"
	}
	return role
}

func SummaryText(home Home) string {
	parts := make([]string, 0, len(home.Cards))
	for _, card := range home.Cards {
		parts = append(parts, fmt.Sprintf("%s=%d", card.Key, card.Value))
	}
	return strings.Join(parts, ", ")
}

func navItemAllowedForRole(item NavItem, role string) bool {
	if len(item.Roles) == 0 {
		return true
	}
	for _, allowed := range item.Roles {
		if normalizeRole(allowed) == role {
			return true
		}
	}
	return false
}
