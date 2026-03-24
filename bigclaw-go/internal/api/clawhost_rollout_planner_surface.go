package api

import (
	"encoding/json"
	"fmt"
	"os"
)

const clawHostRolloutPlannerSurfacePath = "docs/reports/clawhost-rollout-planner-surface.json"

type clawHostRolloutPlannerSurface struct {
	ReportPath      string                        `json:"report_path"`
	GeneratedAt     string                        `json:"generated_at,omitempty"`
	Ticket          string                        `json:"ticket,omitempty"`
	Title           string                        `json:"title,omitempty"`
	Status          string                        `json:"status,omitempty"`
	Provider        string                        `json:"provider,omitempty"`
	PlannerMode     string                        `json:"planner_mode,omitempty"`
	EvidenceSources []string                      `json:"evidence_sources,omitempty"`
	ReviewerLinks   []string                      `json:"reviewer_links,omitempty"`
	Summary         clawHostRolloutPlannerSummary `json:"summary"`
	Waves           []clawHostRolloutWave         `json:"waves,omitempty"`
	TakeoverHooks   []clawHostRolloutTakeoverHook `json:"takeover_hooks,omitempty"`
	Limitations     []string                      `json:"limitations,omitempty"`
	Error           string                        `json:"error,omitempty"`
}

type clawHostRolloutPlannerSummary struct {
	AppCount           int    `json:"app_count"`
	BotCount           int    `json:"bot_count"`
	TotalWaves         int    `json:"total_waves"`
	CanaryWaves        int    `json:"canary_waves"`
	MaxParallelism     int    `json:"max_parallelism"`
	CanaryBotCount     int    `json:"canary_bot_count"`
	TakeoverProtected  int    `json:"takeover_protected_waves"`
	EvidenceReadyWaves int    `json:"evidence_ready_waves"`
	BlockedWaves       int    `json:"blocked_waves"`
	ExecutionReadiness string `json:"execution_readiness,omitempty"`
}

type clawHostRolloutWave struct {
	WaveID           string   `json:"wave_id"`
	Action           string   `json:"action,omitempty"`
	Scope            string   `json:"scope,omitempty"`
	TargetBots       []string `json:"target_bots,omitempty"`
	MaxParallelism   int      `json:"max_parallelism,omitempty"`
	Canary           bool     `json:"canary"`
	CanaryBotIDs     []string `json:"canary_bot_ids,omitempty"`
	TakeoverRequired bool     `json:"takeover_required"`
	SuccessGate      string   `json:"success_gate,omitempty"`
	RollbackGate     string   `json:"rollback_gate,omitempty"`
	ValidationStatus string   `json:"validation_status,omitempty"`
	EvidencePaths    []string `json:"evidence_paths,omitempty"`
}

type clawHostRolloutTakeoverHook struct {
	HookID        string   `json:"hook_id"`
	Trigger       string   `json:"trigger,omitempty"`
	Owner         string   `json:"owner,omitempty"`
	Reviewer      string   `json:"reviewer,omitempty"`
	RequiredSteps []string `json:"required_steps,omitempty"`
}

func clawHostRolloutPlannerSurfacePayload() clawHostRolloutPlannerSurface {
	surface := clawHostRolloutPlannerSurface{ReportPath: clawHostRolloutPlannerSurfacePath}
	reportPath := resolveRepoRelativePath(clawHostRolloutPlannerSurfacePath)
	if reportPath == "" {
		surface.Status = "unavailable"
		surface.Error = "report path could not be resolved"
		return surface
	}
	contents, err := os.ReadFile(reportPath)
	if err != nil {
		surface.Status = "unavailable"
		surface.Error = err.Error()
		return surface
	}
	if err := json.Unmarshal(contents, &surface); err != nil {
		surface.Status = "invalid"
		surface.Error = fmt.Sprintf("decode %s: %v", clawHostRolloutPlannerSurfacePath, err)
		return surface
	}
	surface.ReportPath = clawHostRolloutPlannerSurfacePath
	return surface
}
