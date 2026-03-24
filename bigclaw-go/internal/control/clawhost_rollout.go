package control

import (
	"sort"
	"strconv"
	"strings"

	"bigclaw-go/internal/domain"
)

type ClawHostRolloutTarget struct {
	TaskID         string `json:"task_id"`
	TraceID        string `json:"trace_id,omitempty"`
	TenantID       string `json:"tenant_id"`
	ClawID         string `json:"claw_id"`
	ClawName       string `json:"claw_name"`
	Provider       string `json:"provider,omitempty"`
	CurrentStatus  string `json:"current_status,omitempty"`
	Subdomain      string `json:"subdomain,omitempty"`
	Domain         string `json:"domain,omitempty"`
	AgentCount     int    `json:"agent_count,omitempty"`
	BindingCount   int    `json:"binding_count,omitempty"`
	ChannelCount   int    `json:"channel_count,omitempty"`
	VolumeCount    int    `json:"volume_count,omitempty"`
	TakeoverNeeded bool   `json:"takeover_needed,omitempty"`
}

type ClawHostRolloutWave struct {
	Index             int                     `json:"index"`
	Kind              string                  `json:"kind"`
	ConcurrencyLimit  int                     `json:"concurrency_limit"`
	TakeoverRequired  bool                    `json:"takeover_required"`
	EvidenceArtifacts []string                `json:"evidence_artifacts,omitempty"`
	Targets           []ClawHostRolloutTarget `json:"targets"`
}

type ClawHostRolloutPlan struct {
	Key               string                `json:"key"`
	Action            string                `json:"action"`
	TenantID          string                `json:"tenant_id"`
	ConcurrencyLimit  int                   `json:"concurrency_limit"`
	TargetCount       int                   `json:"target_count"`
	WaveCount         int                   `json:"wave_count"`
	CanaryCount       int                   `json:"canary_count"`
	TakeoverHook      string                `json:"takeover_hook"`
	EvidenceArtifacts []string              `json:"evidence_artifacts,omitempty"`
	Warnings          []string              `json:"warnings,omitempty"`
	Waves             []ClawHostRolloutWave `json:"waves"`
}

func BuildClawHostRolloutPlans(tasks []domain.Task) []ClawHostRolloutPlan {
	type groupedTarget struct {
		action      string
		concurrency int
		target      ClawHostRolloutTarget
		warnings    []string
	}
	grouped := map[string][]groupedTarget{}
	for _, task := range tasks {
		target, action, concurrency, warnings, ok := clawHostRolloutTarget(task)
		if !ok {
			continue
		}
		key := strings.TrimSpace(target.TenantID) + ":" + action
		grouped[key] = append(grouped[key], groupedTarget{
			action:      action,
			concurrency: concurrency,
			target:      target,
			warnings:    warnings,
		})
	}
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	plans := make([]ClawHostRolloutPlan, 0, len(keys))
	for _, key := range keys {
		items := grouped[key]
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].target.TakeoverNeeded != items[j].target.TakeoverNeeded {
				return items[i].target.TakeoverNeeded
			}
			if items[i].target.CurrentStatus == items[j].target.CurrentStatus {
				return items[i].target.ClawName < items[j].target.ClawName
			}
			return items[i].target.CurrentStatus < items[j].target.CurrentStatus
		})
		action := items[0].action
		concurrency := items[0].concurrency
		if concurrency <= 0 {
			concurrency = 1
		}
		targets := make([]ClawHostRolloutTarget, 0, len(items))
		planWarnings := make([]string, 0, len(items))
		warningSeen := map[string]struct{}{}
		takeoverRequired := false
		for _, item := range items {
			targets = append(targets, item.target)
			if item.target.TakeoverNeeded {
				takeoverRequired = true
			}
			for _, warning := range item.warnings {
				if _, ok := warningSeen[warning]; ok {
					continue
				}
				warningSeen[warning] = struct{}{}
				planWarnings = append(planWarnings, warning)
			}
		}
		waves := buildClawHostWaves(targets, action, concurrency, takeoverRequired)
		plan := ClawHostRolloutPlan{
			Key:               key,
			Action:            action,
			TenantID:          items[0].target.TenantID,
			ConcurrencyLimit:  concurrency,
			TargetCount:       len(targets),
			WaveCount:         len(waves),
			CanaryCount:       canaryCount(waves),
			TakeoverHook:      takeoverHook(takeoverRequired),
			EvidenceArtifacts: evidenceArtifacts(action),
			Warnings:          planWarnings,
			Waves:             waves,
		}
		plans = append(plans, plan)
	}
	return plans
}

func buildClawHostWaves(targets []ClawHostRolloutTarget, action string, concurrency int, takeoverRequired bool) []ClawHostRolloutWave {
	if len(targets) == 0 {
		return nil
	}
	waves := make([]ClawHostRolloutWave, 0, 1+(len(targets)-1+concurrency-1)/concurrency)
	canary := ClawHostRolloutWave{
		Index:             1,
		Kind:              "canary",
		ConcurrencyLimit:  1,
		TakeoverRequired:  takeoverRequired,
		EvidenceArtifacts: evidenceArtifacts(action),
		Targets:           append([]ClawHostRolloutTarget(nil), targets[:1]...),
	}
	waves = append(waves, canary)
	for start, waveIndex := 1, 2; start < len(targets); start, waveIndex = start+concurrency, waveIndex+1 {
		end := start + concurrency
		if end > len(targets) {
			end = len(targets)
		}
		waves = append(waves, ClawHostRolloutWave{
			Index:             waveIndex,
			Kind:              "batch",
			ConcurrencyLimit:  concurrency,
			TakeoverRequired:  takeoverRequired,
			EvidenceArtifacts: evidenceArtifacts(action),
			Targets:           append([]ClawHostRolloutTarget(nil), targets[start:end]...),
		})
	}
	return waves
}

func clawHostRolloutTarget(task domain.Task) (ClawHostRolloutTarget, string, int, []string, bool) {
	if !strings.EqualFold(strings.TrimSpace(task.Metadata["control_plane"]), "clawhost") &&
		!strings.EqualFold(strings.TrimSpace(task.Source), "clawhost") &&
		!strings.EqualFold(strings.TrimSpace(task.Metadata["integration"]), "clawhost") {
		return ClawHostRolloutTarget{}, "", 0, nil, false
	}
	if kind := strings.TrimSpace(task.Metadata["inventory_kind"]); kind != "" && !strings.EqualFold(kind, "claw") {
		return ClawHostRolloutTarget{}, "", 0, nil, false
	}
	action := recommendedRolloutAction(task)
	takeoverNeeded := metadataBool(task.Metadata["clawhost_takeover_required"]) || metadataBool(task.Metadata["clawhost_manual_review_required"])
	target := ClawHostRolloutTarget{
		TaskID:         strings.TrimSpace(task.ID),
		TraceID:        strings.TrimSpace(task.TraceID),
		TenantID:       firstNonEmptyControl(strings.TrimSpace(task.TenantID), strings.TrimSpace(task.Metadata["tenant_id"]), strings.TrimSpace(task.Metadata["owner_user_id"]), "unassigned"),
		ClawID:         firstNonEmptyControl(strings.TrimSpace(task.Metadata["claw_id"]), strings.TrimSpace(task.ID)),
		ClawName:       firstNonEmptyControl(strings.TrimSpace(task.Metadata["claw_name"]), strings.TrimSpace(task.Title)),
		Provider:       strings.TrimSpace(task.Metadata["provider"]),
		CurrentStatus:  firstNonEmptyControl(strings.TrimSpace(task.Metadata["provider_status"]), strings.TrimSpace(task.Metadata["source_state"]), string(task.State)),
		Subdomain:      strings.TrimSpace(task.Metadata["subdomain"]),
		Domain:         strings.TrimSpace(task.Metadata["domain"]),
		AgentCount:     metadataInt(task.Metadata["agent_count"]),
		BindingCount:   metadataInt(task.Metadata["binding_count"]),
		ChannelCount:   metadataInt(task.Metadata["channel_count"]),
		VolumeCount:    metadataInt(task.Metadata["volume_count"]),
		TakeoverNeeded: takeoverNeeded,
	}
	concurrency := rolloutConcurrencyLimit(task.Metadata, action)
	warnings := make([]string, 0, 2)
	if target.CurrentStatus == "stopped" && action == "upgrade" {
		warnings = append(warnings, "upgrade plan includes stopped claws that should be started or repaired before upgrade")
	}
	if takeoverNeeded {
		warnings = append(warnings, "plan contains claws that require human takeover before execution")
	}
	return target, action, concurrency, warnings, true
}

func recommendedRolloutAction(task domain.Task) string {
	if action := strings.ToLower(strings.TrimSpace(task.Metadata["clawhost_rollout_action"])); action != "" {
		return action
	}
	status := strings.ToLower(strings.TrimSpace(task.Metadata["provider_status"]))
	switch {
	case status == "stopped":
		return "start"
	case metadataBool(task.Metadata["clawhost_update_available"]):
		return "upgrade"
	case status == "running":
		return "restart"
	default:
		return "repair"
	}
}

func rolloutConcurrencyLimit(metadata map[string]string, action string) int {
	if override := metadataInt(metadata["clawhost_rollout_concurrency_limit"]); override > 0 {
		return override
	}
	switch action {
	case "upgrade":
		return 1
	case "restart":
		return 2
	case "start":
		return 3
	case "stop":
		return 4
	default:
		return 1
	}
}

func evidenceArtifacts(action string) []string {
	return []string{
		"provider_status",
		"agent_status",
		"subdomain_readiness",
		"control_action_audit",
		"rollout_action:" + action,
	}
}

func takeoverHook(required bool) string {
	if required {
		return "required"
	}
	return "optional"
}

func canaryCount(waves []ClawHostRolloutWave) int {
	count := 0
	for _, wave := range waves {
		if wave.Kind == "canary" {
			count += len(wave.Targets)
		}
	}
	return count
}

func metadataBool(value string) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	return err == nil && parsed
}

func metadataInt(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return parsed
}

func firstNonEmptyControl(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
