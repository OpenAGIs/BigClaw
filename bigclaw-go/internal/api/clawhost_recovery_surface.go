package api

import (
	"sort"
	"strings"

	"bigclaw-go/internal/domain"
)

type clawHostRecoverySurface struct {
	Integration      string                   `json:"integration"`
	Status           string                   `json:"status"`
	SupportedActions []string                 `json:"supported_actions"`
	Summary          clawHostRecoverySummary  `json:"summary"`
	Targets          []clawHostRecoveryTarget `json:"targets,omitempty"`
}

type clawHostRecoverySummary struct {
	Targets            int `json:"targets"`
	RecoverableTargets int `json:"recoverable_targets"`
	DegradedTargets    int `json:"degraded_targets"`
	IsolatedTargets    int `json:"isolated_targets"`
	TakeoverRequired   int `json:"takeover_required"`
	EvidenceArtifacts  int `json:"evidence_artifacts"`
}

type clawHostRecoveryTarget struct {
	TaskID           string   `json:"task_id"`
	TenantID         string   `json:"tenant_id"`
	ClawID           string   `json:"claw_id"`
	ClawName         string   `json:"claw_name"`
	LifecycleActions []string `json:"lifecycle_actions,omitempty"`
	PodIsolation     bool     `json:"pod_isolation"`
	ServiceIsolation bool     `json:"service_isolation"`
	TakeoverRequired bool     `json:"takeover_required"`
	TakeoverTriggers []string `json:"takeover_triggers,omitempty"`
	RecoveryEvidence []string `json:"recovery_evidence,omitempty"`
	RecoveryStatus   string   `json:"recovery_status"`
	Warnings         []string `json:"warnings,omitempty"`
}

func clawHostRecoverySurfacePayload(tasks []domain.Task) clawHostRecoverySurface {
	surface := clawHostRecoverySurface{
		Integration:      "clawhost",
		Status:           "idle",
		SupportedActions: []string{"create", "start", "stop", "restart", "upgrade", "delete"},
	}
	targets := make([]clawHostRecoveryTarget, 0, len(tasks))
	for _, task := range tasks {
		target, ok := clawHostRecoveryTargetFromTask(task)
		if !ok {
			continue
		}
		surface.Status = "active"
		targets = append(targets, target)
		surface.Summary.Targets++
		surface.Summary.EvidenceArtifacts += len(target.RecoveryEvidence)
		if target.PodIsolation && target.ServiceIsolation {
			surface.Summary.IsolatedTargets++
		}
		if target.TakeoverRequired {
			surface.Summary.TakeoverRequired++
		}
		if target.RecoveryStatus == "ready" {
			surface.Summary.RecoverableTargets++
		} else {
			surface.Summary.DegradedTargets++
		}
	}
	sort.SliceStable(targets, func(i, j int) bool {
		if targets[i].RecoveryStatus != targets[j].RecoveryStatus {
			return targets[i].RecoveryStatus < targets[j].RecoveryStatus
		}
		return targets[i].ClawName < targets[j].ClawName
	})
	surface.Targets = targets
	return surface
}

func clawHostRecoveryTargetFromTask(task domain.Task) (clawHostRecoveryTarget, bool) {
	if !strings.EqualFold(strings.TrimSpace(task.Metadata["control_plane"]), "clawhost") &&
		!strings.EqualFold(strings.TrimSpace(task.Source), "clawhost") {
		return clawHostRecoveryTarget{}, false
	}
	actions := splitCSVClawHostRecovery(firstNonEmpty(task.Metadata["clawhost_lifecycle_actions"], task.Metadata["clawhost_rollout_action"]))
	triggers := splitCSVClawHostRecovery(task.Metadata["clawhost_takeover_triggers"])
	evidence := splitCSVClawHostRecovery(task.Metadata["clawhost_recovery_evidence"])
	podIsolation := parseBoolClawHost(firstNonEmpty(task.Metadata["clawhost_pod_isolation"], task.Metadata["pod_isolation"]))
	serviceIsolation := parseBoolClawHost(firstNonEmpty(task.Metadata["clawhost_service_isolation"], task.Metadata["service_isolation"]))
	takeoverRequired := parseBoolClawHost(task.Metadata["clawhost_takeover_required"])
	warnings := make([]string, 0, 4)
	recoveryStatus := "ready"
	if !podIsolation || !serviceIsolation {
		warnings = append(warnings, "per-bot pod/service isolation is incomplete")
		recoveryStatus = "degraded"
	}
	if len(actions) == 0 {
		warnings = append(warnings, "lifecycle action coverage is missing")
		recoveryStatus = "degraded"
	}
	if len(triggers) == 0 {
		warnings = append(warnings, "takeover triggers are not defined")
		recoveryStatus = "degraded"
	}
	if len(evidence) == 0 {
		warnings = append(warnings, "recovery evidence is missing")
		recoveryStatus = "degraded"
	}
	return clawHostRecoveryTarget{
		TaskID:           strings.TrimSpace(task.ID),
		TenantID:         firstNonEmpty(strings.TrimSpace(task.TenantID), task.Metadata["tenant_id"], task.Metadata["owner_user_id"]),
		ClawID:           firstNonEmpty(task.Metadata["claw_id"], strings.TrimSpace(task.ID)),
		ClawName:         firstNonEmpty(task.Metadata["claw_name"], strings.TrimSpace(task.Title)),
		LifecycleActions: actions,
		PodIsolation:     podIsolation,
		ServiceIsolation: serviceIsolation,
		TakeoverRequired: takeoverRequired,
		TakeoverTriggers: triggers,
		RecoveryEvidence: evidence,
		RecoveryStatus:   recoveryStatus,
		Warnings:         warnings,
	}, true
}

func splitCSVClawHostRecovery(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	sort.Strings(out)
	return out
}
