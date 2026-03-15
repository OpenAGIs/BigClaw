package api

import (
	"fmt"
	"net/http"

	"bigclaw-go/internal/scheduler"
)

func (s *Server) schedulerPolicyStore() *scheduler.PolicyStore {
	if s.SchedulerPolicy == nil {
		s.SchedulerPolicy = scheduler.NewDefaultPolicyStore()
	}
	return s.SchedulerPolicy
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
	store := s.schedulerPolicyStore()
	writeJSON(w, http.StatusOK, map[string]any{
		"authorization":     authorization,
		"policy":            store.Snapshot(),
		"source_path":       store.SourcePath(),
		"reload_supported":  store.HasSource(),
		"reload_authorized": canReloadSchedulerPolicy(authorization.Role),
		"reload_url":        "/v2/control-center/policy/reload",
	})
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
	writeJSON(w, http.StatusOK, map[string]any{
		"authorization": authorization,
		"reloaded":      true,
		"policy":        store.Snapshot(),
		"source_path":   store.SourcePath(),
	})
}

func canReloadSchedulerPolicy(role ControlRole) bool {
	return role == RolePlatformAdmin || role == RoleVPEng
}
