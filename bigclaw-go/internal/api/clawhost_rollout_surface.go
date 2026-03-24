package api

import (
	"bigclaw-go/internal/control"
	"bigclaw-go/internal/domain"
)

type clawHostRolloutSurface struct {
	Integration      string                        `json:"integration"`
	Status           string                        `json:"status"`
	SupportedActions []string                      `json:"supported_actions"`
	Summary          clawHostRolloutSummary        `json:"summary"`
	Plans            []control.ClawHostRolloutPlan `json:"plans,omitempty"`
}

type clawHostRolloutSummary struct {
	ActivePlans       int `json:"active_plans"`
	TotalTargets      int `json:"total_targets"`
	CanaryTargets     int `json:"canary_targets"`
	TakeoverRequired  int `json:"takeover_required"`
	EvidenceArtifacts int `json:"evidence_artifacts"`
}

func clawHostRolloutSurfacePayload(tasks []domain.Task) clawHostRolloutSurface {
	plans := control.BuildClawHostRolloutPlans(tasks)
	surface := clawHostRolloutSurface{
		Integration:      "clawhost",
		Status:           "idle",
		SupportedActions: []string{"start", "stop", "restart", "upgrade", "repair"},
		Plans:            plans,
	}
	if len(plans) == 0 {
		return surface
	}
	surface.Status = "active"
	for _, plan := range plans {
		surface.Summary.ActivePlans++
		surface.Summary.TotalTargets += plan.TargetCount
		surface.Summary.CanaryTargets += plan.CanaryCount
		if plan.TakeoverHook == "required" {
			surface.Summary.TakeoverRequired++
		}
		surface.Summary.EvidenceArtifacts += len(plan.EvidenceArtifacts)
	}
	return surface
}
