package api

import (
	"fmt"
	"net/http"
	"strings"

	"bigclaw-go/internal/scheduler"
)

func (s *Server) schedulerPolicyStore() *scheduler.PolicyStore {
	if s.SchedulerPolicy == nil {
		s.SchedulerPolicy = scheduler.NewDefaultPolicyStore()
	}
	return s.SchedulerPolicy
}

func (s *Server) schedulerRuntime() *scheduler.Scheduler {
	if s.SchedulerRuntime == nil {
		s.SchedulerRuntime = scheduler.NewWithPolicyStore(s.schedulerPolicyStore())
	}
	return s.SchedulerRuntime
}

func (s *Server) handleV2ControlCenterPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	authorization := parseControlAuthorization(r, "", "", "")
	if err := authorization.validateScope(); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	team := strings.TrimSpace(r.URL.Query().Get("team"))
	project := strings.TrimSpace(r.URL.Query().Get("project"))
	if err := enforceScopedTeamFilter(authorization, &team); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	writeJSON(w, http.StatusOK, s.controlCenterPolicyPayload(r, authorization, team, project, false))
}

func (s *Server) handleV2ControlCenterPolicyReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	authorization := parseControlAuthorization(r, "", "", "")
	if err := authorization.validateScope(); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	if !canReloadSchedulerPolicy(authorization.Role) {
		http.Error(w, fmt.Sprintf("forbidden: role %s cannot reload scheduler policy", authorization.Role), http.StatusForbidden)
		return
	}
	store := s.schedulerPolicyStore()
	if !store.HasSource() {
		http.Error(w, "scheduler policy reload not configured", http.StatusNotImplemented)
		return
	}
	if err := store.Reload(); err != nil {
		http.Error(w, fmt.Sprintf("reload scheduler policy: %v", err), http.StatusInternalServerError)
		return
	}
	team := strings.TrimSpace(r.URL.Query().Get("team"))
	project := strings.TrimSpace(r.URL.Query().Get("project"))
	if err := enforceScopedTeamFilter(authorization, &team); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	writeJSON(w, http.StatusOK, s.controlCenterPolicyPayload(r, authorization, team, project, true))
}

func (s *Server) handleV2ControlCenterPolicyExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	authorization := parseControlAuthorization(r, "", "", "")
	if err := authorization.validateScope(); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	team := strings.TrimSpace(r.URL.Query().Get("team"))
	project := strings.TrimSpace(r.URL.Query().Get("project"))
	if err := enforceScopedTeamFilter(authorization, &team); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	surface := clawHostPolicySurfacePayload(filterClawHostPolicyTasks(s.clawHostPolicyTasks(r.Context()), team, project))
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="clawhost-policy-surface.md"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(renderClawHostPolicySurfaceReport(surface, team, project)))
}

func (s *Server) controlCenterPolicyPayload(r *http.Request, authorization ControlAuthorization, team, project string, reloaded bool) map[string]any {
	store := s.schedulerPolicyStore()
	surface := clawHostPolicySurfacePayload(filterClawHostPolicyTasks(s.clawHostPolicyTasks(r.Context()), team, project))
	payload := map[string]any{
		"authorization":     authorization,
		"filters":           map[string]any{"team": team, "project": project},
		"policy":            store.Snapshot(),
		"fairness":          s.schedulerRuntime().FairnessSnapshot(),
		"clawhost":          surface,
		"backend":           store.Backend(),
		"shared":            store.Shared(),
		"source_path":       store.SourcePath(),
		"shared_path":       store.SharedPath(),
		"updated_at":        store.UpdatedAt(),
		"reload_supported":  store.HasSource(),
		"reload_authorized": canReloadSchedulerPolicy(authorization.Role),
		"reload_url":        "/v2/control-center/policy/reload",
		"report": map[string]any{
			"markdown":   renderClawHostPolicySurfaceReport(surface, team, project),
			"export_url": clawHostExportURL("/v2/control-center/policy/export", team, project, ""),
		},
	}
	if reloaded {
		payload["reloaded"] = true
	}
	return payload
}

func canReloadSchedulerPolicy(role ControlRole) bool {
	return role == RolePlatformAdmin || role == RoleVPEng
}
