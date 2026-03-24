package api

import (
	"context"
	"sort"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/policy"
	"bigclaw-go/internal/queue"
)

type clawHostPolicySurface struct {
	Integration       string                        `json:"integration"`
	Status            string                        `json:"status"`
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

func clawHostPolicySurfacePayload(tasks []domain.Task) clawHostPolicySurface {
	surface := clawHostPolicySurface{
		Integration: "clawhost",
		Status:      "catalog_only",
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
