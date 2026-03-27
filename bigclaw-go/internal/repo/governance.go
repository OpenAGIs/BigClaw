package repo

import "bigclaw-go/internal/contract"

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

func NewPermissionContract() PermissionContract {
	return PermissionContract{
		matrix: contract.NewExecutionPermissionMatrix(actionPermissions, rolePolicies),
	}
}

func (c PermissionContract) Check(actionPermission string, actorRoles []string) bool {
	result := c.matrix.EvaluateRoles([]string{actionPermission}, actorRoles)
	return result.Allowed
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

func ActionPermissions() []contract.ExecutionPermission {
	permissions := make([]contract.ExecutionPermission, 0, len(actionPermissions))
	for _, permission := range actionPermissions {
		permissions = append(permissions, contract.ExecutionPermission{
			Name:     permission.Name,
			Resource: permission.Resource,
			Actions:  append([]string(nil), permission.Actions...),
			Scopes:   append([]string(nil), permission.Scopes...),
		})
	}
	return permissions
}

func RolePolicies() []contract.ExecutionRole {
	roles := make([]contract.ExecutionRole, 0, len(rolePolicies))
	for _, role := range rolePolicies {
		roles = append(roles, contract.ExecutionRole{
			Name:               role.Name,
			Personas:           append([]string(nil), role.Personas...),
			GrantedPermissions: append([]string(nil), role.GrantedPermissions...),
			ScopeBindings:      append([]string(nil), role.ScopeBindings...),
			EscalationTarget:   role.EscalationTarget,
		})
	}
	return roles
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

func permissionNames(permissions []contract.ExecutionPermission) []string {
	names := make([]string, 0, len(permissions))
	for _, permission := range permissions {
		names = append(names, permission.Name)
	}
	return names
}
