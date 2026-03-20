package api

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

type ControlRole string

const (
	RoleViewer            ControlRole = "viewer"
	RoleEngLead           ControlRole = "eng_lead"
	RoleCrossTeamOperator ControlRole = "cross_team_operator"
	RolePlatformAdmin     ControlRole = "platform_admin"
	RoleVPEng             ControlRole = "vp_eng"
)

type ControlAuthorization struct {
	Actor          string      `json:"actor,omitempty"`
	Role           ControlRole `json:"role"`
	ViewerTeam     string      `json:"viewer_team,omitempty"`
	AllowedActions []string    `json:"allowed_actions"`
	CanMutate      bool        `json:"can_mutate"`
	CanViewAudit   bool        `json:"can_view_audit"`
}

func parseControlAuthorization(r *http.Request, actorHint string, roleHint string, viewerTeamHint string) ControlAuthorization {
	actor := strings.TrimSpace(actorHint)
	if actor == "" {
		actor = strings.TrimSpace(r.Header.Get("X-BigClaw-Actor"))
	}
	if actor == "" {
		actor = strings.TrimSpace(r.URL.Query().Get("viewer_actor"))
	}
	role := normalizeControlRole(roleHint)
	if role == RoleViewer {
		role = normalizeControlRole(r.Header.Get("X-BigClaw-Role"))
	}
	if role == RoleViewer {
		role = normalizeControlRole(r.URL.Query().Get("viewer_role"))
	}
	viewerTeam := normalizeViewerTeam(viewerTeamHint)
	if viewerTeam == "" {
		viewerTeam = normalizeViewerTeam(r.Header.Get("X-BigClaw-Team"))
	}
	if viewerTeam == "" {
		viewerTeam = normalizeViewerTeam(r.URL.Query().Get("viewer_team"))
	}
	if viewerTeam == "" && requiresViewerTeam(role) {
		viewerTeam = normalizeViewerTeam(r.URL.Query().Get("team"))
	}
	allowed := allowedControlActions(role)
	canMutate := len(allowed) > 0
	canViewAudit := true
	if requiresViewerTeam(role) && viewerTeam == "" {
		canMutate = false
		canViewAudit = false
	}
	return ControlAuthorization{
		Actor:          actor,
		Role:           role,
		ViewerTeam:     viewerTeam,
		AllowedActions: allowed,
		CanMutate:      canMutate,
		CanViewAudit:   canViewAudit,
	}
}

func normalizeControlRole(raw string) ControlRole {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "platform_admin", "admin", "platform-admin":
		return RolePlatformAdmin
	case "cross_team_operator", "cross-team-operator":
		return RoleCrossTeamOperator
	case "eng_lead", "eng-lead":
		return RoleEngLead
	case "vp_eng", "vp-eng":
		return RoleVPEng
	default:
		return RoleViewer
	}
}

func allowedControlActions(role ControlRole) []string {
	var actions []string
	switch role {
	case RolePlatformAdmin:
		actions = []string{"annotate", "assign_owner", "assign_reviewer", "cancel", "override_budget", "pause", "record_approval", "release_takeover", "resume", "retry", "takeover", "transfer_to_human"}
	case RoleCrossTeamOperator:
		actions = []string{"annotate", "assign_owner", "assign_reviewer", "override_budget", "record_approval", "release_takeover", "retry", "takeover", "transfer_to_human"}
	case RoleEngLead:
		actions = []string{"annotate", "assign_owner", "assign_reviewer", "record_approval", "release_takeover", "takeover", "transfer_to_human"}
	default:
		actions = []string{}
	}
	sort.Strings(actions)
	return actions
}

func canPerformControlAction(role ControlRole, action string) bool {
	action = normalizeActionName(action)
	for _, allowed := range allowedControlActions(role) {
		if allowed == action {
			return true
		}
	}
	return false
}

func normalizeActionName(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "replay_deadletter":
		return "retry"
	case "transfer_to_human":
		return "takeover"
	case "release_to_automation":
		return "release_takeover"
	case "approve", "approval", "record-approval":
		return "record_approval"
	case "budget_override", "budget-override":
		return "override_budget"
	default:
		return strings.ToLower(strings.TrimSpace(action))
	}
}

func requiresViewerTeam(role ControlRole) bool {
	return role == RoleEngLead
}

func normalizeViewerTeam(raw string) string {
	return strings.TrimSpace(raw)
}

func (authorization ControlAuthorization) teamScoped() bool {
	return requiresViewerTeam(authorization.Role)
}

func (authorization ControlAuthorization) validateScope() error {
	if authorization.teamScoped() && authorization.ViewerTeam == "" {
		return fmt.Errorf("forbidden: role %s requires viewer_team", authorization.Role)
	}
	return nil
}

func (authorization ControlAuthorization) permitsTeam(team string) bool {
	if !authorization.teamScoped() {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(team), authorization.ViewerTeam)
}
