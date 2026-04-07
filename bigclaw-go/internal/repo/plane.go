package repo

import (
	"strings"
)

type RepoSpace struct {
	SpaceID                string         `json:"space_id"`
	ProjectKey             string         `json:"project_key"`
	Repo                   string         `json:"repo"`
	DefaultBranch          string         `json:"default_branch,omitempty"`
	SidecarURL             string         `json:"sidecar_url,omitempty"`
	SidecarEnabled         bool           `json:"sidecar_enabled"`
	HealthState            string         `json:"health_state,omitempty"`
	DefaultChannelStrategy string         `json:"default_channel_strategy,omitempty"`
	Metadata               map[string]any `json:"metadata,omitempty"`
}

func (s RepoSpace) DefaultChannelForTask(taskID string) string {
	normalized := slug(taskID)
	return strings.ToLower(strings.TrimSpace(s.ProjectKey)) + "-" + normalized
}

type RepoAgent struct {
	Actor       string   `json:"actor"`
	RepoAgentID string   `json:"repo_agent_id"`
	DisplayName string   `json:"display_name,omitempty"`
	Roles       []string `json:"roles,omitempty"`
}

type RunCommitLink struct {
	RunID       string         `json:"run_id"`
	CommitHash  string         `json:"commit_hash"`
	Role        string         `json:"role"`
	RepoSpaceID string         `json:"repo_space_id"`
	Actor       string         `json:"actor,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

func slug(value string) string {
	var builder strings.Builder
	lastDash := false
	for _, ch := range strings.ToLower(strings.TrimSpace(value)) {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			builder.WriteRune(ch)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(builder.String(), "-")
	if out == "" {
		return "agent"
	}
	return out
}
