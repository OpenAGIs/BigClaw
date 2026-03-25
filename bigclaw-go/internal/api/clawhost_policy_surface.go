package api

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/policy"
	"bigclaw-go/internal/queue"
)

type clawHostPolicySurface struct {
	Integration       string                        `json:"integration"`
	Status            string                        `json:"status"`
	Filters           map[string]string             `json:"filters,omitempty"`
	Catalog           policy.ClawHostCatalog        `json:"catalog"`
	Summary           clawHostPolicySurfaceSummary  `json:"summary"`
	ObservedProviders []string                      `json:"observed_providers,omitempty"`
	ReviewQueue       []policy.ClawHostTenantPolicy `json:"review_queue,omitempty"`
}

type clawHostPolicySurfaceSummary struct {
	ActivePolicies      int `json:"active_policies"`
	ActiveTenants       int `json:"active_tenants"`
	ActiveApps          int `json:"active_apps"`
	ReviewRequired      int `json:"review_required"`
	TakeoverRequired    int `json:"takeover_required"`
	OutOfPolicyDefaults int `json:"out_of_policy_defaults"`
	BlockedDefaults     int `json:"blocked_defaults"`
}

func (s *Server) clawHostPolicyTasks(ctx context.Context) []domain.Task {
	seen := map[string]struct{}{}
	tasks := make([]domain.Task, 0, 8)
	appendTask := func(task domain.Task) {
		if task.ID == "" {
			return
		}
		if _, ok := seen[task.ID]; ok {
			return
		}
		seen[task.ID] = struct{}{}
		tasks = append(tasks, task)
	}
	if inspector, ok := s.Queue.(queue.TaskInspector); ok {
		snapshots, err := inspector.ListTasks(ctx, 0)
		if err == nil {
			for _, snapshot := range snapshots {
				appendTask(snapshot.Task)
			}
		}
	}
	for _, task := range s.Recorder.Tasks(0) {
		appendTask(task)
	}
	return tasks
}

func filterClawHostPolicyTasks(tasks []domain.Task, team, project string) []domain.Task {
	if team == "" && project == "" {
		return tasks
	}
	filtered := make([]domain.Task, 0, len(tasks))
	for _, task := range tasks {
		if team != "" && !strings.EqualFold(strings.TrimSpace(task.Metadata["team"]), team) {
			continue
		}
		if project != "" && !strings.EqualFold(strings.TrimSpace(task.Metadata["project"]), project) {
			continue
		}
		filtered = append(filtered, task)
	}
	return filtered
}

func clawHostPolicySurfacePayload(tasks []domain.Task, team, project string) clawHostPolicySurface {
	surface := clawHostPolicySurface{
		Integration: "clawhost",
		Status:      "catalog_only",
		Filters: map[string]string{
			"team":    strings.TrimSpace(team),
			"project": strings.TrimSpace(project),
		},
		Catalog:     policy.ClawHostCatalogInfo(),
	}
	reviewQueue := make([]policy.ClawHostTenantPolicy, 0, len(tasks))
	tenants := map[string]struct{}{}
	apps := map[string]struct{}{}
	providers := map[string]struct{}{}
	for _, task := range tasks {
		resolved, ok := policy.ResolveClawHostTenantPolicy(task)
		if !ok {
			continue
		}
		surface.Status = "active"
		reviewQueue = append(reviewQueue, resolved)
		surface.Summary.ActivePolicies++
		tenants[resolved.TenantID] = struct{}{}
		apps[resolved.AppID] = struct{}{}
		if resolved.ProviderDefault != "" {
			providers[resolved.ProviderDefault] = struct{}{}
		}
		if resolved.ManualReviewRequired {
			surface.Summary.ReviewRequired++
		}
		if resolved.TakeoverRequired {
			surface.Summary.TakeoverRequired++
		}
		switch resolved.DriftStatus {
		case "out_of_policy":
			surface.Summary.OutOfPolicyDefaults++
		case "blocked":
			surface.Summary.BlockedDefaults++
		}
	}
	surface.Summary.ActiveTenants = len(tenants)
	surface.Summary.ActiveApps = len(apps)
	surface.ObservedProviders = sortedKeys(providers)
	sort.SliceStable(reviewQueue, func(i, j int) bool {
		left := clawHostDriftRank(reviewQueue[i].DriftStatus)
		right := clawHostDriftRank(reviewQueue[j].DriftStatus)
		if left == right {
			if reviewQueue[i].TenantID == reviewQueue[j].TenantID {
				return reviewQueue[i].TaskID < reviewQueue[j].TaskID
			}
			return reviewQueue[i].TenantID < reviewQueue[j].TenantID
		}
		return left < right
	})
	surface.ReviewQueue = reviewQueue
	return surface
}

func clawHostDriftRank(status string) int {
	switch status {
	case "blocked":
		return 0
	case "out_of_policy":
		return 1
	case "review_required":
		return 2
	default:
		return 3
	}
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func renderClawHostPolicySurfaceReport(surface clawHostPolicySurface) string {
	var out strings.Builder
	out.WriteString("# ClawHost Policy Surface\n\n")
	out.WriteString(fmt.Sprintf("- Status: `%s`\n", surface.Status))
	out.WriteString(fmt.Sprintf("- Active policies: `%d`\n", surface.Summary.ActivePolicies))
	out.WriteString(fmt.Sprintf("- Active tenants: `%d`\n", surface.Summary.ActiveTenants))
	out.WriteString(fmt.Sprintf("- Active apps: `%d`\n", surface.Summary.ActiveApps))
	out.WriteString(fmt.Sprintf("- Review required: `%d`\n", surface.Summary.ReviewRequired))
	out.WriteString(fmt.Sprintf("- Takeover required: `%d`\n", surface.Summary.TakeoverRequired))
	out.WriteString(fmt.Sprintf("- Out-of-policy defaults: `%d`\n", surface.Summary.OutOfPolicyDefaults))
	out.WriteString(fmt.Sprintf("- Blocked defaults: `%d`\n", surface.Summary.BlockedDefaults))
	if len(surface.ObservedProviders) > 0 {
		out.WriteString(fmt.Sprintf("- Observed providers: `%s`\n", strings.Join(surface.ObservedProviders, "`, `")))
	}
	out.WriteString("\n## Filters\n\n")
	out.WriteString(fmt.Sprintf("- Team: `%s`\n", clawHostPolicyFallback(strings.TrimSpace(surface.Filters["team"]), "none")))
	out.WriteString(fmt.Sprintf("- Project: `%s`\n", clawHostPolicyFallback(strings.TrimSpace(surface.Filters["project"]), "none")))
	out.WriteString("\n## Review Queue\n\n")
	if len(surface.ReviewQueue) == 0 {
		out.WriteString("No active ClawHost tenant policy reviews.\n")
		return out.String()
	}
	for _, item := range surface.ReviewQueue {
		out.WriteString(fmt.Sprintf("- `%s` tenant `%s` app `%s` provider `%s` drift `%s`\n", item.TaskID, item.TenantID, item.AppID, item.ProviderDefault, item.DriftStatus))
		if item.Reason != "" {
			out.WriteString(fmt.Sprintf("  - Reason: %s\n", item.Reason))
		}
	}
	return out.String()
}

func clawHostPolicyFallback(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
