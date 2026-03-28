package repo

import (
	"fmt"
	"strings"

	"bigclaw-go/internal/contract"
)

var actionPermissions = []contract.ExecutionPermission{
	{Name: "repo.push", Resource: "repo", Actions: []string{"push"}, Scopes: []string{"project"}},
	{Name: "repo.fetch", Resource: "repo", Actions: []string{"fetch"}, Scopes: []string{"project"}},
	{Name: "repo.diff", Resource: "repo", Actions: []string{"diff"}, Scopes: []string{"project"}},
	{Name: "repo.post", Resource: "repo-board", Actions: []string{"create"}, Scopes: []string{"channel"}},
	{Name: "repo.reply", Resource: "repo-board", Actions: []string{"reply"}, Scopes: []string{"channel"}},
	{Name: "repo.accept", Resource: "repo", Actions: []string{"approve"}, Scopes: []string{"run"}},
	{Name: "repo.inspect", Resource: "repo", Actions: []string{"inspect"}, Scopes: []string{"project"}},
}

var rolePolicies = []contract.ExecutionRole{
	{
		Name:               "platform-admin",
		Personas:           []string{"Platform Admin"},
		GrantedPermissions: permissionNames(actionPermissions),
		ScopeBindings:      []string{"workspace"},
		EscalationTarget:   "security",
	},
	{
		Name:               "eng-lead",
		Personas:           []string{"Eng Lead"},
		GrantedPermissions: []string{"repo.push", "repo.fetch", "repo.diff", "repo.post", "repo.reply", "repo.accept", "repo.inspect"},
		ScopeBindings:      []string{"project"},
		EscalationTarget:   "platform-admin",
	},
	{
		Name:               "reviewer",
		Personas:           []string{"Reviewer"},
		GrantedPermissions: []string{"repo.fetch", "repo.diff", "repo.reply", "repo.inspect", "repo.accept"},
		ScopeBindings:      []string{"project"},
		EscalationTarget:   "eng-lead",
	},
	{
		Name:               "execution-agent",
		Personas:           []string{"Execution Agent"},
		GrantedPermissions: []string{"repo.fetch", "repo.diff", "repo.post", "repo.reply"},
		ScopeBindings:      []string{"run"},
		EscalationTarget:   "reviewer",
	},
}

type PermissionContract struct {
	matrix contract.ExecutionPermissionMatrix
}

type GovernancePolicy struct {
	MaxBundleBytes  int64 `json:"max_bundle_bytes,omitempty"`
	MaxPushPerHour  int   `json:"max_push_per_hour,omitempty"`
	MaxDiffPerHour  int   `json:"max_diff_per_hour,omitempty"`
	SidecarRequired bool  `json:"sidecar_required,omitempty"`
}

type GovernanceDecision struct {
	Allowed bool   `json:"allowed"`
	Mode    string `json:"mode,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

type GovernanceEnforcer struct {
	policy GovernancePolicy
	counts map[string]int
}

func NewPermissionContract() PermissionContract {
	return PermissionContract{
		matrix: contract.NewExecutionPermissionMatrix(actionPermissions, rolePolicies),
	}
}

func NewGovernanceEnforcer(policy GovernancePolicy) *GovernanceEnforcer {
	return &GovernanceEnforcer{
		policy: policy,
		counts: make(map[string]int),
	}
}

func (c PermissionContract) Check(actionPermission string, actorRoles []string) bool {
	result := c.matrix.EvaluateRoles([]string{actionPermission}, actorRoles)
	return result.Allowed
}

func (e *GovernanceEnforcer) Evaluate(action string, bundleBytes int64, sidecarAvailable bool) GovernanceDecision {
	action = strings.TrimSpace(strings.ToLower(action))
	if e.policy.SidecarRequired && !sidecarAvailable {
		return GovernanceDecision{
			Allowed: false,
			Mode:    "degraded",
			Reason:  "repo governance sidecar is required but unavailable",
		}
	}
	if e.policy.MaxBundleBytes > 0 && bundleBytes > e.policy.MaxBundleBytes {
		return GovernanceDecision{
			Allowed: false,
			Mode:    "blocked",
			Reason:  fmt.Sprintf("bundle exceeds max bundle bytes quota (%d>%d)", bundleBytes, e.policy.MaxBundleBytes),
		}
	}
	limit := e.quotaForAction(action)
	if limit > 0 && e.counts[action] >= limit {
		return GovernanceDecision{
			Allowed: false,
			Mode:    "blocked",
			Reason:  fmt.Sprintf("%s quota exceeded for the current hour", action),
		}
	}
	e.counts[action]++
	return GovernanceDecision{Allowed: true, Mode: "allowed"}
}

func RequiredAuditFields(action string) []string {
	common := []string{"task_id", "run_id", "repo_space_id", "actor"}
	switch action {
	case "repo.accept":
		return append(common, "accepted_commit_hash", "reviewer")
	case "repo.push", "repo.fetch", "repo.diff":
		return append(common, "commit_hash", "outcome")
	case "repo.post", "repo.reply":
		return append(common, "channel", "post_id", "outcome")
	default:
		return common
	}
}

func MissingAuditFields(action string, payload map[string]any) []string {
	required := RequiredAuditFields(action)
	missing := make([]string, 0)
	for _, field := range required {
		if _, ok := payload[field]; !ok {
			missing = append(missing, field)
		}
	}
	return missing
}

func (e *GovernanceEnforcer) quotaForAction(action string) int {
	switch action {
	case "push":
		return e.policy.MaxPushPerHour
	case "diff":
		return e.policy.MaxDiffPerHour
	default:
		return 0
	}
}

func permissionNames(permissions []contract.ExecutionPermission) []string {
	names := make([]string, 0, len(permissions))
	for _, permission := range permissions {
		names = append(names, permission.Name)
	}
	return names
}
