package product

import (
	"fmt"
	"sort"
	"strings"

	"bigclaw-go/internal/domain"
)

type ClawHostRolloutWave struct {
	WaveID            string   `json:"wave_id"`
	Name              string   `json:"name"`
	Strategy          string   `json:"strategy"`
	TargetTenants     []string `json:"target_tenants,omitempty"`
	TargetApps        []string `json:"target_apps,omitempty"`
	LifecycleActions  []string `json:"lifecycle_actions,omitempty"`
	MaxParallelBots   int      `json:"max_parallel_bots"`
	RequiresApproval  bool     `json:"requires_approval"`
	HealthChecks      []string `json:"health_checks,omitempty"`
	TakeoverTriggers  []string `json:"takeover_triggers,omitempty"`
	PromotionCriteria []string `json:"promotion_criteria,omitempty"`
	RollbackActions   []string `json:"rollback_actions,omitempty"`
}

type ClawHostRolloutPlanner struct {
	PlanID              string                 `json:"plan_id"`
	Version             string                 `json:"version"`
	Source              string                 `json:"source"`
	Filters             map[string]string      `json:"filters,omitempty"`
	Summary             ClawHostRolloutSummary `json:"summary"`
	ConcurrencyGuards   []string               `json:"concurrency_guards,omitempty"`
	ValidationEvidence  []string               `json:"validation_evidence,omitempty"`
	SharedControlPlanes []string               `json:"shared_control_planes,omitempty"`
	Waves               []ClawHostRolloutWave  `json:"waves,omitempty"`
}

type ClawHostRolloutSummary struct {
	WaveCount             int `json:"wave_count"`
	TenantCount           int `json:"tenant_count"`
	AppCount              int `json:"app_count"`
	RequiresApprovalWaves int `json:"requires_approval_waves"`
	CanaryWaves           int `json:"canary_waves"`
	MaxParallelBots       int `json:"max_parallel_bots"`
}

type ClawHostRolloutPlannerAudit struct {
	PlanID                 string   `json:"plan_id"`
	Version                string   `json:"version"`
	DuplicateWaveIDs       []string `json:"duplicate_wave_ids,omitempty"`
	WavesMissingChecks     []string `json:"waves_missing_checks,omitempty"`
	WavesMissingRollback   []string `json:"waves_missing_rollback,omitempty"`
	InvalidParallelism     []string `json:"invalid_parallelism,omitempty"`
	MissingTakeoverSignals []string `json:"missing_takeover_signals,omitempty"`
	ReadinessScore         float64  `json:"readiness_score"`
	ReleaseReady           bool     `json:"release_ready"`
}

func BuildDefaultClawHostRolloutPlanner(tasks []domain.Task, team, project string) ClawHostRolloutPlanner {
	tenants := clawHostSortedValues(tasks, func(task domain.Task) string {
		return clawHostFirstNonEmpty(task.TenantID, task.Metadata["tenant"], task.Metadata["team"])
	})
	if len(tenants) == 0 {
		tenants = []string{"platform", "support", "customer-success"}
	}
	apps := clawHostSortedValues(tasks, func(task domain.Task) string {
		return clawHostFirstNonEmpty(task.Metadata["app"], task.Metadata["app_id"], task.Metadata["project"], project)
	})
	if len(apps) == 0 {
		apps = []string{emptyFallback(strings.TrimSpace(project), "clawhost-app")}
	}

	waves := []ClawHostRolloutWave{
		{
			WaveID:            "clawhost-canary-upgrade",
			Name:              "Canary Upgrade Wave",
			Strategy:          "canary",
			TargetTenants:     tenants[:1],
			TargetApps:        apps[:1],
			LifecycleActions:  []string{"status", "restart", "upgrade"},
			MaxParallelBots:   1,
			RequiresApproval:  true,
			HealthChecks:      []string{"GET /bot/api/v1/bots/:id/status", "GET /bot/api/v1/bots/:id/connect", "GET /proxy/:bot_id/"},
			TakeoverTriggers:  []string{"bot stays in error state", "proxy handshake fails", "device approval queue spikes"},
			PromotionCriteria: []string{"pod and service become ready", "subdomain traffic reaches the upgraded bot", "provider defaults remain intact"},
			RollbackActions:   []string{"stop bot", "restart previous image", "hold tenant segment in BigClaw control center"},
		},
		{
			WaveID:            "clawhost-tenant-ring-1",
			Name:              "Tenant Ring 1",
			Strategy:          "tenant-batch",
			TargetTenants:     append([]string(nil), tenants...),
			TargetApps:        apps[:clawHostMin(2, len(apps))],
			LifecycleActions:  []string{"start", "restart", "upgrade"},
			MaxParallelBots:   3,
			RequiresApproval:  true,
			HealthChecks:      []string{"GET /bot/api/v1/bots/:id/status", "GET /bot/api/v1/bots/:id/channels", "GET /bot/api/v1/bots/:id/devices"},
			TakeoverTriggers:  []string{"tenant fairness throttle trips", "channel pairing backlog grows", "manual approver rejects rollout"},
			PromotionCriteria: []string{"per-tenant backlog remains bounded", "pairing approvals stay below manual threshold", "no blocked provider drift review"},
			RollbackActions:   []string{"pause tenant wave", "release active takeovers after review", "re-run previous bot image on affected tenants"},
		},
		{
			WaveID:            "clawhost-app-fanout",
			Name:              "App Fanout Wave",
			Strategy:          "app-fanout",
			TargetTenants:     append([]string(nil), tenants...),
			TargetApps:        append([]string(nil), apps...),
			LifecycleActions:  []string{"upgrade", "status"},
			MaxParallelBots:   clawHostMax(5, len(apps)*2),
			RequiresApproval:  false,
			HealthChecks:      []string{"GET /health", "GET /bot/api/v1/admin/apps", "GET /proxy/:bot_id/"},
			TakeoverTriggers:  []string{"admin UI or proxy control plane degrades", "subdomain routing mismatch detected"},
			PromotionCriteria: []string{"app inventory remains queryable", "bot domains resolve through the proxy", "upgrade cadence stays under fleet budget"},
			RollbackActions:   []string{"requeue failed app segment", "demote to tenant-ring execution", "surface reviewer packet in distributed diagnostics"},
		},
	}

	summary := ClawHostRolloutSummary{
		WaveCount:             len(waves),
		TenantCount:           len(tenants),
		AppCount:              len(apps),
		RequiresApprovalWaves: countClawHostApprovalWaves(waves),
		CanaryWaves:           countClawHostStrategy(waves, "canary"),
		MaxParallelBots:       clawHostMaxParallel(waves),
	}

	return ClawHostRolloutPlanner{
		PlanID:  "BIG-PAR-288",
		Version: "go-v1",
		Source:  "ClawHost lifecycle orchestration surface",
		Filters: map[string]string{
			"team":    strings.TrimSpace(team),
			"project": strings.TrimSpace(project),
		},
		Summary: summary,
		ConcurrencyGuards: []string{
			"respect BigClaw fairness windows per tenant before promoting the next ClawHost wave",
			"cap canary wave to one bot until status, connect, and proxy routes all pass",
			"hold tenant-ring rollout when manual device or pairing approvals exceed reviewer budget",
			"route upgrade retries through BigClaw takeover when app-level proxy health regresses",
		},
		ValidationEvidence: []string{
			"/v2/control-center",
			"/v2/reports/distributed",
			"/v2/runs/{task_id}",
			"GET /bot/api/v1/bots/:id/status",
			"GET /bot/api/v1/bots/:id/connect",
			"GET /proxy/:bot_id/",
		},
		SharedControlPlanes: []string{
			"ClawHost REST API",
			"ClawHost admin UI",
			"BigClaw control center",
			"BigClaw distributed diagnostics",
		},
		Waves: waves,
	}
}

func AuditClawHostRolloutPlanner(plan ClawHostRolloutPlanner) ClawHostRolloutPlannerAudit {
	audit := ClawHostRolloutPlannerAudit{
		PlanID:  plan.PlanID,
		Version: plan.Version,
	}
	waveIDs := make([]string, 0, len(plan.Waves))
	for _, wave := range plan.Waves {
		waveIDs = append(waveIDs, wave.WaveID)
		if wave.MaxParallelBots <= 0 {
			audit.InvalidParallelism = append(audit.InvalidParallelism, wave.WaveID)
		}
		if len(wave.HealthChecks) == 0 {
			audit.WavesMissingChecks = append(audit.WavesMissingChecks, wave.WaveID)
		}
		if len(wave.RollbackActions) == 0 {
			audit.WavesMissingRollback = append(audit.WavesMissingRollback, wave.WaveID)
		}
		if len(wave.TakeoverTriggers) == 0 {
			audit.MissingTakeoverSignals = append(audit.MissingTakeoverSignals, wave.WaveID)
		}
	}
	audit.DuplicateWaveIDs = duplicateStrings(waveIDs)
	penalties := len(audit.DuplicateWaveIDs) + len(audit.WavesMissingChecks) + len(audit.WavesMissingRollback) + len(audit.InvalidParallelism) + len(audit.MissingTakeoverSignals)
	if len(plan.Waves) == 0 {
		audit.ReadinessScore = 0
		return audit
	}
	audit.ReadinessScore = round1(maxFloat(0, 100-(float64(penalties)*100/float64(len(plan.Waves)))))
	audit.ReleaseReady = penalties == 0
	return audit
}

func RenderClawHostRolloutPlannerReport(plan ClawHostRolloutPlanner, audit ClawHostRolloutPlannerAudit) string {
	lines := []string{
		"# ClawHost Rollout Planner",
		"",
		fmt.Sprintf("- Plan ID: %s", plan.PlanID),
		fmt.Sprintf("- Version: %s", plan.Version),
		fmt.Sprintf("- Source: %s", plan.Source),
		fmt.Sprintf("- Release Ready: %s", boolText(audit.ReleaseReady)),
		fmt.Sprintf("- Readiness Score: %.1f", audit.ReadinessScore),
		fmt.Sprintf("- Tenants: %d", plan.Summary.TenantCount),
		fmt.Sprintf("- Apps: %d", plan.Summary.AppCount),
		fmt.Sprintf("- Max Parallel Bots: %d", plan.Summary.MaxParallelBots),
		"",
		"## Concurrency Guards",
		"",
	}
	for _, item := range plan.ConcurrencyGuards {
		lines = append(lines, "- "+item)
	}
	lines = append(lines, "", "## Waves", "")
	for _, wave := range plan.Waves {
		lines = append(lines,
			fmt.Sprintf("- %s: strategy=%s tenants=%s apps=%s max_parallel_bots=%d requires_approval=%t actions=%s",
				wave.Name,
				wave.Strategy,
				emptyFallback(strings.Join(wave.TargetTenants, ", "), "none"),
				emptyFallback(strings.Join(wave.TargetApps, ", "), "none"),
				wave.MaxParallelBots,
				wave.RequiresApproval,
				emptyFallback(strings.Join(wave.LifecycleActions, ", "), "none"),
			),
			fmt.Sprintf("  health_checks=%s", emptyFallback(strings.Join(wave.HealthChecks, "; "), "none")),
			fmt.Sprintf("  takeover_triggers=%s", emptyFallback(strings.Join(wave.TakeoverTriggers, "; "), "none")),
			fmt.Sprintf("  promotion_criteria=%s", emptyFallback(strings.Join(wave.PromotionCriteria, "; "), "none")),
			fmt.Sprintf("  rollback_actions=%s", emptyFallback(strings.Join(wave.RollbackActions, "; "), "none")),
		)
	}
	lines = append(lines, "", "## Validation Evidence", "")
	for _, item := range plan.ValidationEvidence {
		lines = append(lines, "- "+item)
	}
	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, fmt.Sprintf("- Duplicate wave IDs: %s", emptyFallback(strings.Join(audit.DuplicateWaveIDs, ", "), "none")))
	lines = append(lines, fmt.Sprintf("- Waves missing checks: %s", emptyFallback(strings.Join(audit.WavesMissingChecks, ", "), "none")))
	lines = append(lines, fmt.Sprintf("- Waves missing rollback: %s", emptyFallback(strings.Join(audit.WavesMissingRollback, ", "), "none")))
	lines = append(lines, fmt.Sprintf("- Invalid parallelism: %s", emptyFallback(strings.Join(audit.InvalidParallelism, ", "), "none")))
	lines = append(lines, fmt.Sprintf("- Missing takeover signals: %s", emptyFallback(strings.Join(audit.MissingTakeoverSignals, ", "), "none")))
	return strings.Join(lines, "\n") + "\n"
}

func clawHostSortedValues(tasks []domain.Task, getter func(domain.Task) string) []string {
	seen := map[string]struct{}{}
	values := make([]string, 0)
	for _, task := range tasks {
		value := strings.TrimSpace(getter(task))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func clawHostFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func countClawHostApprovalWaves(waves []ClawHostRolloutWave) int {
	count := 0
	for _, wave := range waves {
		if wave.RequiresApproval {
			count++
		}
	}
	return count
}

func countClawHostStrategy(waves []ClawHostRolloutWave, strategy string) int {
	count := 0
	for _, wave := range waves {
		if strings.EqualFold(strings.TrimSpace(wave.Strategy), strategy) {
			count++
		}
	}
	return count
}

func clawHostMaxParallel(waves []ClawHostRolloutWave) int {
	maxParallel := 0
	for _, wave := range waves {
		if wave.MaxParallelBots > maxParallel {
			maxParallel = wave.MaxParallelBots
		}
	}
	return maxParallel
}

func clawHostMin(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func clawHostMax(left, right int) int {
	if left > right {
		return left
	}
	return right
}
