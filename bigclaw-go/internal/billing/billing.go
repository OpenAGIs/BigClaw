package billing

import (
	"sort"
	"strings"

	"bigclaw-go/internal/domain"
)

type TeamUsage struct {
	Key              string `json:"key"`
	SeatCount        int    `json:"seat_count"`
	BudgetCentsTotal int64  `json:"budget_cents_total"`
	PremiumRuns      int    `json:"premium_runs"`
}

type UsageSummary struct {
	Organization     string      `json:"organization"`
	Tier             string      `json:"tier"`
	SeatCount        int         `json:"seat_count"`
	ActiveSeats      int         `json:"active_seats"`
	BudgetCentsTotal int64       `json:"budget_cents_total"`
	PremiumRuns      int         `json:"premium_runs"`
	StandardRuns     int         `json:"standard_runs"`
	Alerts           []string    `json:"alerts,omitempty"`
	ByTeam           []TeamUsage `json:"by_team"`
}

type Entitlements struct {
	Tier              string           `json:"tier"`
	Features          map[string]bool  `json:"features"`
	Limits            map[string]int64 `json:"limits"`
	EnabledDashboards []string         `json:"enabled_dashboards"`
}

func BuildUsage(tasks []domain.Task, organization string, tier string) UsageSummary {
	usage := UsageSummary{Organization: firstNonEmpty(organization, "openagi"), Tier: normalizeTier(tier)}
	teamSeats := make(map[string]map[string]struct{})
	byTeam := make(map[string]*TeamUsage)
	globalSeats := make(map[string]struct{})
	activeSeats := make(map[string]struct{})
	for _, task := range tasks {
		team := firstNonEmpty(task.Metadata["team"], "unassigned")
		entry := byTeam[team]
		if entry == nil {
			entry = &TeamUsage{Key: team}
			byTeam[team] = entry
		}
		entry.BudgetCentsTotal += task.BudgetCents
		usage.BudgetCentsTotal += task.BudgetCents
		if strings.EqualFold(strings.TrimSpace(task.Metadata["plan"]), "premium") {
			entry.PremiumRuns++
			usage.PremiumRuns++
		} else {
			usage.StandardRuns++
		}
		if teamSeats[team] == nil {
			teamSeats[team] = make(map[string]struct{})
		}
		for _, user := range collectUsers(task) {
			globalSeats[user] = struct{}{}
			teamSeats[team][user] = struct{}{}
			if domain.IsActiveTaskState(task.State) {
				activeSeats[user] = struct{}{}
			}
		}
	}
	usage.SeatCount = len(globalSeats)
	usage.ActiveSeats = len(activeSeats)
	for team, seats := range teamSeats {
		if entry := byTeam[team]; entry != nil {
			entry.SeatCount = len(seats)
		}
	}
	for _, entry := range byTeam {
		usage.ByTeam = append(usage.ByTeam, *entry)
	}
	sort.SliceStable(usage.ByTeam, func(i, j int) bool {
		if usage.ByTeam[i].BudgetCentsTotal == usage.ByTeam[j].BudgetCentsTotal {
			return usage.ByTeam[i].Key < usage.ByTeam[j].Key
		}
		return usage.ByTeam[i].BudgetCentsTotal > usage.ByTeam[j].BudgetCentsTotal
	})
	usage.Alerts = buildAlerts(usage)
	return usage
}

func EntitlementsForTier(tier string) Entitlements {
	tier = normalizeTier(tier)
	ent := Entitlements{Tier: tier, Features: map[string]bool{}, Limits: map[string]int64{}, EnabledDashboards: []string{"engineering", "operations"}}
	switch tier {
	case "enterprise":
		ent.Features = map[string]bool{"premium_orchestration": true, "advanced_approval": true, "browser_pool": true, "vm_pool": true, "regression_center": true, "billing_console": true, "flow_canvas": true}
		ent.Limits = map[string]int64{"max_agents": 8, "concurrency_limit": 32, "queue_depth_limit": 256, "budget_cap_cents": 50000}
		ent.EnabledDashboards = append(ent.EnabledDashboards, "triage", "regression", "billing", "flows")
	case "growth":
		ent.Features = map[string]bool{"premium_orchestration": true, "advanced_approval": true, "browser_pool": true, "vm_pool": false, "regression_center": true, "billing_console": true, "flow_canvas": true}
		ent.Limits = map[string]int64{"max_agents": 4, "concurrency_limit": 16, "queue_depth_limit": 128, "budget_cap_cents": 20000}
		ent.EnabledDashboards = append(ent.EnabledDashboards, "triage", "regression", "billing", "flows")
	default:
		ent.Features = map[string]bool{"premium_orchestration": false, "advanced_approval": false, "browser_pool": false, "vm_pool": false, "regression_center": false, "billing_console": true, "flow_canvas": false}
		ent.Limits = map[string]int64{"max_agents": 2, "concurrency_limit": 8, "queue_depth_limit": 64, "budget_cap_cents": 10000}
	}
	sort.Strings(ent.EnabledDashboards)
	return ent
}

func collectUsers(task domain.Task) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, key := range []string{"owner", "reviewer", "created_by", "assignee"} {
		if value := strings.TrimSpace(task.Metadata[key]); value != "" {
			if _, ok := seen[value]; ok {
				continue
			}
			seen[value] = struct{}{}
			out = append(out, value)
		}
	}
	return out
}

func buildAlerts(usage UsageSummary) []string {
	alerts := make([]string, 0)
	if usage.PremiumRuns > usage.StandardRuns && usage.PremiumRuns > 0 {
		alerts = append(alerts, "Premium orchestration usage is the majority of current workload.")
	}
	if usage.BudgetCentsTotal > EntitlementsForTier(usage.Tier).Limits["budget_cap_cents"] {
		alerts = append(alerts, "Observed spend exceeds the current tier budget cap.")
	}
	if usage.SeatCount == 0 {
		alerts = append(alerts, "No billable seats detected in the current snapshot.")
	}
	return alerts
}

func normalizeTier(tier string) string {
	tier = strings.ToLower(strings.TrimSpace(tier))
	if tier == "" {
		return "growth"
	}
	return tier
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
