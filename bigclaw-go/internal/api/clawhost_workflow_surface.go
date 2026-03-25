package api

import (
	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/product"
)

func clawHostWorkflowSurfacePayload(tasks []domain.Task, actor, team, project string) product.ClawHostWorkflowSurface {
	return product.BuildClawHostWorkflowSurface(tasks, actor, team, project)
}
