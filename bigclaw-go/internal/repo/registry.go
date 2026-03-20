package repo

type Registry struct {
	SpacesByProject map[string]RepoSpace `json:"spaces_by_project,omitempty"`
	AgentsByActor   map[string]RepoAgent `json:"agents_by_actor,omitempty"`
}

func NewRegistry() Registry {
	return Registry{
		SpacesByProject: map[string]RepoSpace{},
		AgentsByActor:   map[string]RepoAgent{},
	}
}

func (r *Registry) RegisterSpace(space RepoSpace) {
	if r.SpacesByProject == nil {
		r.SpacesByProject = map[string]RepoSpace{}
	}
	r.SpacesByProject[space.ProjectKey] = space
}

func (r Registry) ResolveSpace(projectKey string) (RepoSpace, bool) {
	space, ok := r.SpacesByProject[projectKey]
	return space, ok
}

func (r Registry) ResolveDefaultChannel(projectKey string, taskID string) string {
	if space, ok := r.ResolveSpace(projectKey); ok {
		return space.DefaultChannelForTask(taskID)
	}
	return lowerASCII(projectKey) + "-" + slugify(taskID)
}

func (r *Registry) ResolveAgent(actor string, role string) RepoAgent {
	if r.AgentsByActor == nil {
		r.AgentsByActor = map[string]RepoAgent{}
	}
	if existing, ok := r.AgentsByActor[actor]; ok {
		return existing
	}
	agent := RepoAgent{
		Actor:       actor,
		RepoAgentID: "agent-" + slugify(actor),
		DisplayName: actor,
		Roles:       []string{role},
	}
	r.AgentsByActor[actor] = agent
	return agent
}

func NormalizeRegistry(payload map[string]any) Registry {
	registry := NewRegistry()
	for key, value := range mapValue(payload["spaces_by_project"]) {
		registry.SpacesByProject[key] = NormalizeRepoSpace(mapValue(value))
	}
	for key, value := range mapValue(payload["agents_by_actor"]) {
		registry.AgentsByActor[key] = NormalizeRepoAgent(mapValue(value))
	}
	return registry
}
