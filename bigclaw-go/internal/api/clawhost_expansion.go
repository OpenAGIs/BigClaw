package api

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"bigclaw-go/internal/product"
)

func (s *Server) handleV2ClawHostFleet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	inventory := product.BuildDefaultClawHostFleetSurface()
	audit := product.AuditClawHostFleetSurface(inventory)
	writeJSON(w, http.StatusOK, map[string]any{
		"inventory": inventory,
		"audit":     audit,
		"report": map[string]any{
			"markdown":   product.RenderClawHostFleetReport(inventory, audit),
			"export_url": "/v2/clawhost/fleet/export",
		},
	})
}

func (s *Server) handleV2ClawHostFleetExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	inventory := product.BuildDefaultClawHostFleetSurface()
	audit := product.AuditClawHostFleetSurface(inventory)
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="clawhost-fleet.md"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(product.RenderClawHostFleetReport(inventory, audit)))
}

func (s *Server) handleV2ClawHostRolloutPlanner(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	team, project, _ := clawHostScopeFilters(r)
	plan := product.BuildDefaultClawHostRolloutPlanner(s.filteredTasks(team, project, "", time.Time{}, time.Time{}), team, project)
	audit := product.AuditClawHostRolloutPlanner(plan)
	writeJSON(w, http.StatusOK, map[string]any{
		"plan":  plan,
		"audit": audit,
		"report": map[string]any{
			"markdown":   product.RenderClawHostRolloutPlannerReport(plan, audit),
			"export_url": clawHostExportURL("/v2/clawhost/rollout-planner/export", team, project, ""),
		},
	})
}

func (s *Server) handleV2ClawHostRolloutPlannerExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	team, project, _ := clawHostScopeFilters(r)
	plan := product.BuildDefaultClawHostRolloutPlanner(s.filteredTasks(team, project, "", time.Time{}, time.Time{}), team, project)
	audit := product.AuditClawHostRolloutPlanner(plan)
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="clawhost-rollout-planner.md"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(product.RenderClawHostRolloutPlannerReport(plan, audit)))
}

func (s *Server) handleV2ClawHostWorkflows(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	team, project, actor := clawHostScopeFilters(r)
	surface := product.BuildDefaultClawHostWorkflowSurface(s.filteredTasks(team, project, "", time.Time{}, time.Time{}), actor, team, project)
	audit := product.AuditClawHostWorkflowSurface(surface)
	writeJSON(w, http.StatusOK, map[string]any{
		"surface": surface,
		"audit":   audit,
		"report": map[string]any{
			"markdown":   product.RenderClawHostWorkflowReport(surface, audit),
			"export_url": clawHostExportURL("/v2/clawhost/workflows/export", team, project, actor),
		},
	})
}

func (s *Server) handleV2ClawHostWorkflowsExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	team, project, actor := clawHostScopeFilters(r)
	surface := product.BuildDefaultClawHostWorkflowSurface(s.filteredTasks(team, project, "", time.Time{}, time.Time{}), actor, team, project)
	audit := product.AuditClawHostWorkflowSurface(surface)
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="clawhost-workflows.md"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(product.RenderClawHostWorkflowReport(surface, audit)))
}

func clawHostScopeFilters(r *http.Request) (string, string, string) {
	team := strings.TrimSpace(r.URL.Query().Get("team"))
	project := strings.TrimSpace(r.URL.Query().Get("project"))
	actor := firstNonEmpty(r.URL.Query().Get("actor"), r.Header.Get("X-BigClaw-Actor"))
	return team, project, actor
}

func clawHostExportURL(base, team, project, actor string) string {
	values := url.Values{}
	if team = strings.TrimSpace(team); team != "" {
		values.Set("team", team)
	}
	if project = strings.TrimSpace(project); project != "" {
		values.Set("project", project)
	}
	if actor = strings.TrimSpace(actor); actor != "" {
		values.Set("actor", actor)
	}
	if len(values) == 0 {
		return base
	}
	return base + "?" + values.Encode()
}
