package api

import (
	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/product"
)

func clawHostWorkflowSurfacePayload(tasks []domain.Task) product.ClawHostWorkflowSurface {
	return product.BuildClawHostWorkflowSurface(tasks)
}
