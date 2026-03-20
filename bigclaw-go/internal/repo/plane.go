package repo

type RepoSpace struct {
	SpaceID                string         `json:"space_id"`
	ProjectKey             string         `json:"project_key"`
	Repo                   string         `json:"repo"`
	DefaultBranch          string         `json:"default_branch,omitempty"`
	SidecarURL             string         `json:"sidecar_url,omitempty"`
	SidecarEnabled         bool           `json:"sidecar_enabled,omitempty"`
	HealthState            string         `json:"health_state,omitempty"`
	DefaultChannelStrategy string         `json:"default_channel_strategy,omitempty"`
	Metadata               map[string]any `json:"metadata,omitempty"`
}

func (s RepoSpace) DefaultChannelForTask(taskID string) string {
	return s.ProjectKeyLower() + "-" + slugify(taskID)
}

func (s RepoSpace) ProjectKeyLower() string {
	if text := stringValue(s.ProjectKey); text != "" {
		return lowerASCII(text)
	}
	return ""
}

func NormalizeRepoSpace(payload map[string]any) RepoSpace {
	return RepoSpace{
		SpaceID:                stringValue(payload["space_id"]),
		ProjectKey:             stringValue(payload["project_key"]),
		Repo:                   stringValue(payload["repo"]),
		DefaultBranch:          defaultString(payload["default_branch"], "main"),
		SidecarURL:             stringValue(payload["sidecar_url"]),
		SidecarEnabled:         defaultBool(payload["sidecar_enabled"], true),
		HealthState:            defaultString(payload["health_state"], "unknown"),
		DefaultChannelStrategy: defaultString(payload["default_channel_strategy"], "task"),
		Metadata:               mapValue(payload["metadata"]),
	}
}

type RepoAgent struct {
	Actor       string   `json:"actor"`
	RepoAgentID string   `json:"repo_agent_id"`
	DisplayName string   `json:"display_name,omitempty"`
	Roles       []string `json:"roles,omitempty"`
}

func NormalizeRepoAgent(payload map[string]any) RepoAgent {
	return RepoAgent{
		Actor:       stringValue(payload["actor"]),
		RepoAgentID: stringValue(payload["repo_agent_id"]),
		DisplayName: stringValue(payload["display_name"]),
		Roles:       stringSliceValue(payload["roles"]),
	}
}

type RunCommitLink struct {
	RunID       string         `json:"run_id"`
	CommitHash  string         `json:"commit_hash"`
	Role        string         `json:"role"`
	RepoSpaceID string         `json:"repo_space_id"`
	Actor       string         `json:"actor,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

func NormalizeRunCommitLink(payload map[string]any) RunCommitLink {
	return RunCommitLink{
		RunID:       stringValue(payload["run_id"]),
		CommitHash:  stringValue(payload["commit_hash"]),
		Role:        stringValue(payload["role"]),
		RepoSpaceID: stringValue(payload["repo_space_id"]),
		Actor:       stringValue(payload["actor"]),
		Metadata:    mapValue(payload["metadata"]),
	}
}
