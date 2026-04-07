package executor

import (
	"context"
	"time"

	"bigclaw-go/internal/domain"
)

type Capability struct {
	Kind            domain.ExecutorKind
	MaxConcurrency  int
	SupportsGPU     bool
	SupportsBrowser bool
	SupportsShell   bool
}

type Assignment struct {
	Executor domain.ExecutorKind
	Reason   string
}

type Result struct {
	Success     bool
	ShouldRetry bool
	DeadLetter  bool
	Message     string
	Artifacts   []string
	FinishedAt  time.Time
}

type Runner interface {
	Kind() domain.ExecutorKind
	Capability() Capability
	Execute(context.Context, domain.Task) Result
}

type Registry struct {
	runners map[domain.ExecutorKind]Runner
}

func NewRegistry(runners ...Runner) *Registry {
	registry := &Registry{runners: make(map[domain.ExecutorKind]Runner)}
	for _, runner := range runners {
		registry.runners[runner.Kind()] = runner
	}
	return registry
}

func (r *Registry) Get(kind domain.ExecutorKind) (Runner, bool) {
	runner, ok := r.runners[kind]
	return runner, ok
}

func (r *Registry) Kinds() []domain.ExecutorKind {
	kinds := make([]domain.ExecutorKind, 0, len(r.runners))
	for kind := range r.runners {
		kinds = append(kinds, kind)
	}
	return kinds
}

func CapabilityForKind(kind domain.ExecutorKind) Capability {
	switch kind {
	case domain.ExecutorLocal:
		return LocalRunner{}.Capability()
	case domain.ExecutorKubernetes:
		return (&KubernetesRunner{}).Capability()
	case domain.ExecutorRay:
		return (&RayRunner{}).Capability()
	default:
		return Capability{Kind: kind}
	}
}
