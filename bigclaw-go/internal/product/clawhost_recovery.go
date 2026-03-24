package product

import (
	"fmt"
	"sort"
	"strings"
)

type ClawHostRecoveryLifecycleAction struct {
	Action           string   `json:"action"`
	Supported        bool     `json:"supported"`
	RecoveryCheck    string   `json:"recovery_check"`
	TakeoverTriggers []string `json:"takeover_triggers,omitempty"`
	Evidence         []string `json:"evidence,omitempty"`
}

type ClawHostBotRecoveryScore struct {
	BotID             string   `json:"bot_id"`
	AppID             string   `json:"app_id"`
	TenantID          string   `json:"tenant_id"`
	Name              string   `json:"name"`
	Status            string   `json:"status"`
	PodIsolation      bool     `json:"pod_isolation"`
	ServiceIsolation  bool     `json:"service_isolation"`
	SupportedActions  []string `json:"supported_actions,omitempty"`
	TakeoverTriggers  []string `json:"takeover_triggers,omitempty"`
	RecoveryEvidence  []string `json:"recovery_evidence,omitempty"`
	RecoveryReadiness string   `json:"recovery_readiness"`
	Warnings          []string `json:"warnings,omitempty"`
}

type ClawHostLifecycleRecoverySummary struct {
	BotCount             int `json:"bot_count"`
	IsolatedBots         int `json:"isolated_bots"`
	RecoverableBots      int `json:"recoverable_bots"`
	DegradedBots         int `json:"degraded_bots"`
	TakeoverCoveredBots  int `json:"takeover_covered_bots"`
	FullyCoveredActions  int `json:"fully_covered_actions"`
	EvidenceArtifactRefs int `json:"evidence_artifact_refs"`
}

type ClawHostLifecycleRecoveryScorecard struct {
	ScorecardID      string                            `json:"scorecard_id"`
	Version          string                            `json:"version"`
	SourceRepository string                            `json:"source_repository"`
	Filters          map[string]string                 `json:"filters,omitempty"`
	ControlPlane     ClawHostControlPlane              `json:"control_plane"`
	Lifecycle        []ClawHostRecoveryLifecycleAction `json:"lifecycle,omitempty"`
	Bots             []ClawHostBotRecoveryScore        `json:"bots,omitempty"`
	Summary          ClawHostLifecycleRecoverySummary  `json:"summary"`
}

type ClawHostLifecycleRecoveryAudit struct {
	ScorecardID             string   `json:"scorecard_id"`
	Version                 string   `json:"version"`
	MissingLifecycleActions []string `json:"missing_lifecycle_actions,omitempty"`
	BotsMissingIsolation    []string `json:"bots_missing_isolation,omitempty"`
	BotsMissingTakeover     []string `json:"bots_missing_takeover,omitempty"`
	BotsMissingEvidence     []string `json:"bots_missing_evidence,omitempty"`
	DegradedBots            []string `json:"degraded_bots,omitempty"`
	ReadinessScore          float64  `json:"readiness_score"`
	ReleaseReady            bool     `json:"release_ready"`
}

func BuildDefaultClawHostLifecycleRecoveryScorecard(team, project string) ClawHostLifecycleRecoveryScorecard {
	inventory := FilterClawHostFleetSurface(BuildDefaultClawHostFleetSurface(), team, project)
	appTenant := map[string]string{}
	for _, app := range inventory.Apps {
		appTenant[app.AppID] = app.TenantID
	}
	lifecycle := []ClawHostRecoveryLifecycleAction{
		{
			Action:           "create",
			Supported:        true,
			RecoveryCheck:    "new bot reaches running state with proxy route and tenant ownership intact",
			TakeoverTriggers: []string{"bot remains pending after bootstrap window", "ownership metadata is missing after create"},
			Evidence:         []string{"POST /bot/api/v1/bots", "GET /bot/api/v1/bots/:id/status", "/v2/clawhost/fleet"},
		},
		{
			Action:           "start",
			Supported:        true,
			RecoveryCheck:    "stopped bot returns to ready without losing channels or provider defaults",
			TakeoverTriggers: []string{"start attempt stalls in created", "provider session fails to restore"},
			Evidence:         []string{"POST /bot/api/v1/bots/:id/start", "GET /bot/api/v1/bots/:id/connect", "/v2/control-center"},
		},
		{
			Action:           "stop",
			Supported:        true,
			RecoveryCheck:    "bot drains traffic and exits cleanly before destructive actions",
			TakeoverTriggers: []string{"proxy keeps routing to draining bot", "stop action leaks open sessions"},
			Evidence:         []string{"POST /bot/api/v1/bots/:id/stop", "GET /proxy/:bot_id/", "/v2/reports/distributed"},
		},
		{
			Action:           "restart",
			Supported:        true,
			RecoveryCheck:    "bot restarts into a fresh pod and service pair without tenant drift",
			TakeoverTriggers: []string{"restart loops exceed retry budget", "service endpoint changes without readiness"},
			Evidence:         []string{"POST /bot/api/v1/bots/:id/restart", "GET /bot/api/v1/bots/:id/status", "/debug/status"},
		},
		{
			Action:           "upgrade",
			Supported:        true,
			RecoveryCheck:    "image update keeps subdomain, provider defaults, and reviewer gates aligned",
			TakeoverTriggers: []string{"upgrade leaves bot on stale image", "subdomain or websocket checks regress after rollout"},
			Evidence:         []string{"POST /bot/api/v1/bots/:id/upgrade", "/v2/clawhost/rollout-planner", "/v2/clawhost/workflows"},
		},
		{
			Action:           "delete",
			Supported:        true,
			RecoveryCheck:    "bot removal clears proxy routes and tenant inventory without orphans",
			TakeoverTriggers: []string{"app inventory still reports deleted bot", "delete path leaves orphaned service or volume"},
			Evidence:         []string{"DELETE /bot/api/v1/bots/:id", "/v2/clawhost/fleet/export", "/v2/reports/distributed"},
		},
	}

	bots := make([]ClawHostBotRecoveryScore, 0, len(inventory.Bots))
	for _, bot := range inventory.Bots {
		takeoverTriggers := []string{
			"proxy or websocket health regresses during lifecycle action",
			"tenant fairness window is exceeded while bot is mutating",
		}
		recoveryEvidence := []string{
			fmt.Sprintf("GET /bot/api/v1/bots/%s/status", bot.BotID),
			fmt.Sprintf("GET /proxy/%s/", bot.BotID),
			"/v2/reports/distributed",
		}
		warnings := make([]string, 0, 2)
		readiness := "ready"
		if !bot.PodIsolation || !bot.ServiceIsolation {
			warnings = append(warnings, "bot is missing dedicated pod or service isolation")
			readiness = "degraded"
		}
		if len(takeoverTriggers) == 0 {
			warnings = append(warnings, "bot is missing takeover triggers")
			readiness = "degraded"
		}
		if len(recoveryEvidence) == 0 {
			warnings = append(warnings, "bot is missing recovery evidence")
			readiness = "degraded"
		}
		bots = append(bots, ClawHostBotRecoveryScore{
			BotID:             bot.BotID,
			AppID:             bot.AppID,
			TenantID:          appTenant[bot.AppID],
			Name:              bot.Name,
			Status:            normalizedClawHostStatus(bot.Status),
			PodIsolation:      bot.PodIsolation,
			ServiceIsolation:  bot.ServiceIsolation,
			SupportedActions:  []string{"create", "start", "stop", "restart", "upgrade", "delete"},
			TakeoverTriggers:  takeoverTriggers,
			RecoveryEvidence:  recoveryEvidence,
			RecoveryReadiness: readiness,
			Warnings:          warnings,
		})
	}
	sort.SliceStable(bots, func(i, j int) bool { return bots[i].BotID < bots[j].BotID })

	scorecard := ClawHostLifecycleRecoveryScorecard{
		ScorecardID:      "BIG-PAR-292",
		Version:          "go-v1",
		SourceRepository: inventory.SourceRepository,
		Filters: map[string]string{
			"team":    strings.TrimSpace(team),
			"project": strings.TrimSpace(project),
		},
		ControlPlane: inventory.ControlPlane,
		Lifecycle:    lifecycle,
		Bots:         bots,
	}
	for _, action := range lifecycle {
		if action.Supported && len(action.Evidence) > 0 && len(action.TakeoverTriggers) > 0 {
			scorecard.Summary.FullyCoveredActions++
		}
		scorecard.Summary.EvidenceArtifactRefs += len(action.Evidence)
	}
	for _, bot := range bots {
		scorecard.Summary.BotCount++
		scorecard.Summary.EvidenceArtifactRefs += len(bot.RecoveryEvidence)
		if bot.PodIsolation && bot.ServiceIsolation {
			scorecard.Summary.IsolatedBots++
		}
		if len(bot.TakeoverTriggers) > 0 {
			scorecard.Summary.TakeoverCoveredBots++
		}
		if bot.RecoveryReadiness == "ready" {
			scorecard.Summary.RecoverableBots++
		} else {
			scorecard.Summary.DegradedBots++
		}
	}
	return scorecard
}

func AuditClawHostLifecycleRecoveryScorecard(scorecard ClawHostLifecycleRecoveryScorecard) ClawHostLifecycleRecoveryAudit {
	audit := ClawHostLifecycleRecoveryAudit{
		ScorecardID: scorecard.ScorecardID,
		Version:     scorecard.Version,
	}
	requiredActions := map[string]struct{}{
		"create": {}, "start": {}, "stop": {}, "restart": {}, "upgrade": {}, "delete": {},
	}
	for _, action := range scorecard.Lifecycle {
		if !action.Supported || len(action.Evidence) == 0 || len(action.TakeoverTriggers) == 0 {
			audit.MissingLifecycleActions = append(audit.MissingLifecycleActions, action.Action)
		}
		delete(requiredActions, strings.TrimSpace(action.Action))
	}
	for action := range requiredActions {
		audit.MissingLifecycleActions = append(audit.MissingLifecycleActions, action)
	}
	sort.Strings(audit.MissingLifecycleActions)
	for _, bot := range scorecard.Bots {
		if !bot.PodIsolation || !bot.ServiceIsolation {
			audit.BotsMissingIsolation = append(audit.BotsMissingIsolation, bot.BotID)
		}
		if len(bot.TakeoverTriggers) == 0 {
			audit.BotsMissingTakeover = append(audit.BotsMissingTakeover, bot.BotID)
		}
		if len(bot.RecoveryEvidence) == 0 {
			audit.BotsMissingEvidence = append(audit.BotsMissingEvidence, bot.BotID)
		}
		if bot.RecoveryReadiness != "ready" {
			audit.DegradedBots = append(audit.DegradedBots, bot.BotID)
		}
	}
	sort.Strings(audit.BotsMissingIsolation)
	sort.Strings(audit.BotsMissingTakeover)
	sort.Strings(audit.BotsMissingEvidence)
	sort.Strings(audit.DegradedBots)
	penalties := len(audit.MissingLifecycleActions) + len(audit.BotsMissingIsolation) + len(audit.BotsMissingTakeover) + len(audit.BotsMissingEvidence) + len(audit.DegradedBots)
	totalChecks := len(scorecard.Lifecycle) + len(scorecard.Bots)
	if totalChecks == 0 {
		return audit
	}
	audit.ReadinessScore = round1(maxFloat(0, 100-(float64(penalties)*100/float64(totalChecks))))
	audit.ReleaseReady = penalties == 0
	return audit
}

func RenderClawHostLifecycleRecoveryReport(scorecard ClawHostLifecycleRecoveryScorecard, audit ClawHostLifecycleRecoveryAudit) string {
	lines := []string{
		"# ClawHost Lifecycle Recovery Scorecard",
		"",
		fmt.Sprintf("- Scorecard ID: %s", scorecard.ScorecardID),
		fmt.Sprintf("- Version: %s", scorecard.Version),
		fmt.Sprintf("- Source Repository: %s", scorecard.SourceRepository),
		fmt.Sprintf("- Release Ready: %s", boolText(audit.ReleaseReady)),
		fmt.Sprintf("- Readiness Score: %.1f", audit.ReadinessScore),
		fmt.Sprintf("- Recoverable Bots: %d/%d", scorecard.Summary.RecoverableBots, scorecard.Summary.BotCount),
		fmt.Sprintf("- Isolated Bots: %d/%d", scorecard.Summary.IsolatedBots, scorecard.Summary.BotCount),
		"",
		"## Filters",
		"",
	}
	filterKeys := make([]string, 0, len(scorecard.Filters))
	for key := range scorecard.Filters {
		filterKeys = append(filterKeys, key)
	}
	sort.Strings(filterKeys)
	if len(filterKeys) == 0 {
		lines = append(lines, "- none")
	} else {
		for _, key := range filterKeys {
			lines = append(lines, fmt.Sprintf("- %s: %s", key, emptyFallback(scorecard.Filters[key], "none")))
		}
	}
	lines = append(lines,
		"",
		"## Lifecycle Coverage",
		"",
	)
	for _, action := range scorecard.Lifecycle {
		lines = append(lines, fmt.Sprintf("- %s: supported=%t recovery_check=%s evidence=%s takeover_triggers=%s",
			action.Action,
			action.Supported,
			action.RecoveryCheck,
			emptyFallback(strings.Join(action.Evidence, "; "), "none"),
			emptyFallback(strings.Join(action.TakeoverTriggers, "; "), "none"),
		))
	}
	lines = append(lines, "", "## Per-Bot Isolation", "")
	for _, bot := range scorecard.Bots {
		lines = append(lines, fmt.Sprintf("- %s (%s): tenant=%s status=%s pod_isolation=%t service_isolation=%t readiness=%s actions=%s",
			bot.Name,
			bot.BotID,
			emptyFallback(bot.TenantID, "unassigned"),
			bot.Status,
			bot.PodIsolation,
			bot.ServiceIsolation,
			bot.RecoveryReadiness,
			emptyFallback(strings.Join(bot.SupportedActions, ", "), "none"),
		))
		lines = append(lines, fmt.Sprintf("  takeover_triggers=%s", emptyFallback(strings.Join(bot.TakeoverTriggers, "; "), "none")))
		lines = append(lines, fmt.Sprintf("  recovery_evidence=%s", emptyFallback(strings.Join(bot.RecoveryEvidence, "; "), "none")))
		if len(bot.Warnings) > 0 {
			lines = append(lines, fmt.Sprintf("  warnings=%s", strings.Join(bot.Warnings, " | ")))
		}
	}
	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, fmt.Sprintf("- Missing lifecycle coverage: %s", fallbackJoin(audit.MissingLifecycleActions)))
	lines = append(lines, fmt.Sprintf("- Bots missing isolation: %s", fallbackJoin(audit.BotsMissingIsolation)))
	lines = append(lines, fmt.Sprintf("- Bots missing takeover: %s", fallbackJoin(audit.BotsMissingTakeover)))
	lines = append(lines, fmt.Sprintf("- Bots missing evidence: %s", fallbackJoin(audit.BotsMissingEvidence)))
	lines = append(lines, fmt.Sprintf("- Degraded bots: %s", fallbackJoin(audit.DegradedBots)))
	return strings.Join(lines, "\n") + "\n"
}
