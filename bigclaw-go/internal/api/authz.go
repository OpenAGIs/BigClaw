package api

import (
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
	AllowedActions []string    `json:"allowed_actions"`
	CanMutate      bool        `json:"can_mutate"`
	CanViewAudit   bool        `json:"can_view_audit"`
}

func parseControlAuthorization(r *http.Request, actorHint string, roleHint string) ControlAuthorization {
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
	allowed := allowedControlActions(role)
	return ControlAuthorization{
		Actor:          actor,
		Role:           role,
		AllowedActions: allowed,
		CanMutate:      len(allowed) > 0,
		CanViewAudit:   true,
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
		actions = []string{"annotate", "cancel", "pause", "release_takeover", "resume", "retry", "takeover", "transfer_to_human"}
	case RoleCrossTeamOperator:
		actions = []string{"annotate", "release_takeover", "retry", "takeover", "transfer_to_human"}
	case RoleEngLead:
		actions = []string{"annotate", "release_takeover", "takeover", "transfer_to_human"}
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
	default:
		return strings.ToLower(strings.TrimSpace(action))
	}
}
