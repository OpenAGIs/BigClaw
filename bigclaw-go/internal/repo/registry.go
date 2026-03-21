package repo

import "bigclaw-go/internal/domain"

type RepoRegistry struct {
	SpacesByProject map[string]RepoSpace `json:"spaces_by_project,omitempty"`
	AgentsByActor   map[string]RepoAgent `json:"agents_by_actor,omitempty"`
}

func (r *RepoRegistry) RegisterSpace(space RepoSpace) {
	if r.SpacesByProject == nil {
		r.SpacesByProject = map[string]RepoSpace{}
	}
	r.SpacesByProject[space.ProjectKey] = space
}

func (r RepoRegistry) ResolveSpace(projectKey string) (RepoSpace, bool) {
	space, ok := r.SpacesByProject[projectKey]
	return space, ok
}

func (r RepoRegistry) ResolveDefaultChannel(projectKey string, task domain.Task) string {
	if space, ok := r.ResolveSpace(projectKey); ok {
		return space.DefaultChannelForTask(task.ID)
	}
	return projectKey + "-" + slug(task.ID)
}

func (r *RepoRegistry) ResolveAgent(actor, role string) RepoAgent {
	if r.AgentsByActor == nil {
		r.AgentsByActor = map[string]RepoAgent{}
	}
	if agent, ok := r.AgentsByActor[actor]; ok {
		return agent
	}
	agent := RepoAgent{
		Actor:       actor,
		RepoAgentID: "agent-" + slug(actor),
		DisplayName: actor,
		Roles:       []string{role},
	}
	r.AgentsByActor[actor] = agent
	return agent
}
