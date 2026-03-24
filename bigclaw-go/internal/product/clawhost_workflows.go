package product

import (
	"fmt"
	"sort"
	"strings"

	"bigclaw-go/internal/domain"
)

var (
	validClawHostWorkflowStages = map[string]struct{}{
		"inventory":     {},
		"planning":      {},
		"execution":     {},
		"approval":      {},
		"observability": {},
	}
	validClawHostAutomationBoundaries = map[string]struct{}{
		"advisory_only":  {},
		"human_gated":    {},
		"auto_with_gate": {},
	}
)

type ClawHostParallelBatch struct {
	Name              string   `json:"name"`
	MaxConcurrency    int      `json:"max_concurrency"`
	CanarySize        int      `json:"canary_size"`
	Selector          string   `json:"selector"`
	RequiredApprovals []string `json:"required_approvals,omitempty"`
}

type ClawHostWorkflowLane struct {
	LaneID                string                `json:"lane_id"`
	Name                  string                `json:"name"`
	Stage                 string                `json:"stage"`
	Route                 string                `json:"route"`
	Owner                 string                `json:"owner"`
	ParallelBatch         ClawHostParallelBatch `json:"parallel_batch"`
	AutomationBoundary    string                `json:"automation_boundary"`
	TokenSessionGating    bool                  `json:"token_session_gating"`
	DeviceAutoApproval    bool                  `json:"device_auto_approval"`
	SupportsHumanTakeover bool                  `json:"supports_human_takeover"`
	SkillAware            bool                  `json:"skill_aware"`
	ChannelAware          bool                  `json:"channel_aware"`
	ProviderAware         bool                  `json:"provider_aware"`
	HandoffTeams          []string              `json:"handoff_teams,omitempty"`
	Notes                 []string              `json:"notes,omitempty"`
}

type ClawHostWorkflowSurface struct {
	Name               string                 `json:"name"`
	Version            string                 `json:"version"`
	SourceRepo         string                 `json:"source_repo"`
	ReferenceRevision  string                 `json:"reference_revision"`
	Filters            map[string]string      `json:"filters,omitempty"`
	Summary            map[string]any         `json:"summary"`
	OperationalSignals map[string]int         `json:"operational_signals,omitempty"`
	Lanes              []ClawHostWorkflowLane `json:"lanes,omitempty"`
}

type ClawHostWorkflowSurfaceAudit struct {
	Name                             string   `json:"name"`
	Version                          string   `json:"version"`
	LaneCount                        int      `json:"lane_count"`
	MissingRouteLanes                []string `json:"missing_route_lanes,omitempty"`
	MissingOwnerLanes                []string `json:"missing_owner_lanes,omitempty"`
	LanesWithoutTakeover             []string `json:"lanes_without_takeover,omitempty"`
	LanesWithoutTokenSessionGating   []string `json:"lanes_without_token_session_gating,omitempty"`
	LanesWithoutRequiredApprovals    []string `json:"lanes_without_required_approvals,omitempty"`
	LanesWithInvalidStage            []string `json:"lanes_with_invalid_stage,omitempty"`
	LanesWithInvalidAutomationPolicy []string `json:"lanes_with_invalid_automation_policy,omitempty"`
	ReadinessScore                   float64  `json:"readiness_score"`
}

func BuildDefaultClawHostWorkflowSurface(tasks []domain.Task, actor, team, project string) ClawHostWorkflowSurface {
	owner := normalizedWorkflowOwner(actor)
	lanes := []ClawHostWorkflowLane{
		{
			LaneID:             "clawhost-fleet-inventory",
			Name:               "Fleet Inventory & Ownership Baseline",
			Stage:              "inventory",
			Route:              buildSavedViewRoute("/v2/control-center", team, project),
			Owner:              owner,
			AutomationBoundary: "advisory_only",
			ParallelBatch: ClawHostParallelBatch{
				Name:              "inventory-refresh",
				MaxConcurrency:    20,
				CanarySize:        5,
				Selector:          "app_id,user_id,status",
				RequiredApprovals: []string{"platform-ops"},
			},
			SupportsHumanTakeover: true,
			ProviderAware:         true,
			HandoffTeams:          []string{"platform", "operations"},
			Notes: []string{
				"captures app and bot ownership slices before any rollout lane executes",
				"aligns with ClawHost multi-tenant app/user ownership model",
			},
		},
		{
			LaneID:             "clawhost-skills-rollout",
			Name:               "Skills and Agent Config Rollout",
			Stage:              "planning",
			Route:              buildSavedViewRoute("/v2/control-center/audit", team, project),
			Owner:              owner,
			AutomationBoundary: "human_gated",
			ParallelBatch: ClawHostParallelBatch{
				Name:              "skills-wave",
				MaxConcurrency:    8,
				CanarySize:        2,
				Selector:          "bot_slug,team",
				RequiredApprovals: []string{"product-ops", "ai-safety"},
			},
			SupportsHumanTakeover: true,
			SkillAware:            true,
			HandoffTeams:          []string{"product", "operations"},
			Notes: []string{
				"supports dynamic skill updates for running bots",
				"keeps risky skill or prompt changes under reviewer gates",
			},
		},
		{
			LaneID:             "clawhost-im-channels-and-devices",
			Name:               "IM Channels and Device Approval Workflows",
			Stage:              "execution",
			Route:              buildSavedViewRoute("/v2/triage/center", team, project),
			Owner:              owner,
			AutomationBoundary: "auto_with_gate",
			ParallelBatch: ClawHostParallelBatch{
				Name:              "channel-device-wave",
				MaxConcurrency:    10,
				CanarySize:        3,
				Selector:          "channel,account",
				RequiredApprovals: []string{"support-lead"},
			},
			DeviceAutoApproval:    true,
			SupportsHumanTakeover: true,
			SkillAware:            true,
			ChannelAware:          true,
			HandoffTeams:          []string{"support", "operations"},
			Notes: []string{
				"tracks multi-account channel operations and device approvals",
				"uses takeover hooks when device pairing or channel auth drifts",
			},
		},
		{
			LaneID:             "clawhost-proxy-session-gate",
			Name:               "Proxy Auth and Session Gating",
			Stage:              "approval",
			Route:              buildSavedViewRoute("/v2/runs", team, project),
			Owner:              owner,
			AutomationBoundary: "human_gated",
			ParallelBatch: ClawHostParallelBatch{
				Name:              "proxy-gate-wave",
				MaxConcurrency:    12,
				CanarySize:        3,
				Selector:          "domain,bot_id",
				RequiredApprovals: []string{"security-review"},
			},
			TokenSessionGating:    true,
			SupportsHumanTakeover: true,
			ChannelAware:          true,
			HandoffTeams:          []string{"security", "operations"},
			Notes: []string{
				"requires token or signed session cookie before forwarding to bot UI",
				"validates websocket and http proxy posture before broad rollout",
			},
		},
		{
			LaneID:             "clawhost-parallel-rollout-control",
			Name:               "Parallel Rollout and Incident Takeover",
			Stage:              "observability",
			Route:              buildSavedViewRoute("/v2/reports/distributed", team, project),
			Owner:              owner,
			AutomationBoundary: "human_gated",
			ParallelBatch: ClawHostParallelBatch{
				Name:              "lifecycle-rollout-wave",
				MaxConcurrency:    maxIntWorkflow(4, inferParallelLimit(tasks)),
				CanarySize:        2,
				Selector:          "status,provider",
				RequiredApprovals: []string{"release-manager", "platform-ops"},
			},
			TokenSessionGating:    true,
			DeviceAutoApproval:    true,
			SupportsHumanTakeover: true,
			SkillAware:            true,
			ChannelAware:          true,
			ProviderAware:         true,
			HandoffTeams:          []string{"release", "operations", "support"},
			Notes: []string{
				"covers start, stop, restart, and upgrade waves with canary policy",
				"keeps rollback and takeover signals visible on distributed diagnostics",
			},
		},
	}
	sort.SliceStable(lanes, func(i, j int) bool { return lanes[i].LaneID < lanes[j].LaneID })

	surface := ClawHostWorkflowSurface{
		Name:              "clawhost-workflow-surface",
		Version:           "go-v1",
		SourceRepo:        "https://github.com/fastclaw-ai/clawhost",
		ReferenceRevision: "ddd7c1dd960cc1769a39301968f10327f24978e1",
		Filters: map[string]string{
			"team":    strings.TrimSpace(team),
			"project": strings.TrimSpace(project),
			"actor":   owner,
		},
		OperationalSignals: clawhostWorkflowSignals(tasks),
		Lanes:              lanes,
	}
	surface.Summary = map[string]any{
		"lane_count":                     len(surface.Lanes),
		"supports_parallel_batches":      len(surface.Lanes),
		"token_session_gated_lanes":      countClawHostLanes(surface.Lanes, func(lane ClawHostWorkflowLane) bool { return lane.TokenSessionGating }),
		"device_auto_approval_lanes":     countClawHostLanes(surface.Lanes, func(lane ClawHostWorkflowLane) bool { return lane.DeviceAutoApproval }),
		"human_takeover_enabled_lanes":   countClawHostLanes(surface.Lanes, func(lane ClawHostWorkflowLane) bool { return lane.SupportsHumanTakeover }),
		"provider_aware_lanes":           countClawHostLanes(surface.Lanes, func(lane ClawHostWorkflowLane) bool { return lane.ProviderAware }),
		"skills_and_channel_aware_lanes": countClawHostLanes(surface.Lanes, func(lane ClawHostWorkflowLane) bool { return lane.SkillAware || lane.ChannelAware }),
	}
	return surface
}

func BuildClawHostWorkflowSurface(tasks []domain.Task, actor, team, project string) ClawHostWorkflowSurface {
	return BuildDefaultClawHostWorkflowSurface(tasks, actor, team, project)
}

func AuditClawHostWorkflowSurface(surface ClawHostWorkflowSurface) ClawHostWorkflowSurfaceAudit {
	audit := ClawHostWorkflowSurfaceAudit{
		Name:      surface.Name,
		Version:   surface.Version,
		LaneCount: len(surface.Lanes),
	}
	for _, lane := range surface.Lanes {
		if strings.TrimSpace(lane.Route) == "" {
			audit.MissingRouteLanes = append(audit.MissingRouteLanes, lane.LaneID)
		}
		if strings.TrimSpace(lane.Owner) == "" {
			audit.MissingOwnerLanes = append(audit.MissingOwnerLanes, lane.LaneID)
		}
		if !lane.SupportsHumanTakeover {
			audit.LanesWithoutTakeover = append(audit.LanesWithoutTakeover, lane.LaneID)
		}
		if !lane.TokenSessionGating {
			audit.LanesWithoutTokenSessionGating = append(audit.LanesWithoutTokenSessionGating, lane.LaneID)
		}
		if len(lane.ParallelBatch.RequiredApprovals) == 0 {
			audit.LanesWithoutRequiredApprovals = append(audit.LanesWithoutRequiredApprovals, lane.LaneID)
		}
		if _, ok := validClawHostWorkflowStages[lane.Stage]; !ok {
			audit.LanesWithInvalidStage = append(audit.LanesWithInvalidStage, lane.LaneID)
		}
		if _, ok := validClawHostAutomationBoundaries[lane.AutomationBoundary]; !ok {
			audit.LanesWithInvalidAutomationPolicy = append(audit.LanesWithInvalidAutomationPolicy, lane.LaneID)
		}
	}
	sort.Strings(audit.MissingRouteLanes)
	sort.Strings(audit.MissingOwnerLanes)
	sort.Strings(audit.LanesWithoutTakeover)
	sort.Strings(audit.LanesWithoutTokenSessionGating)
	sort.Strings(audit.LanesWithoutRequiredApprovals)
	sort.Strings(audit.LanesWithInvalidStage)
	sort.Strings(audit.LanesWithInvalidAutomationPolicy)

	penalties := len(audit.MissingRouteLanes) +
		len(audit.MissingOwnerLanes) +
		len(audit.LanesWithoutTakeover) +
		len(audit.LanesWithoutTokenSessionGating) +
		len(audit.LanesWithoutRequiredApprovals) +
		len(audit.LanesWithInvalidStage) +
		len(audit.LanesWithInvalidAutomationPolicy)
	if len(surface.Lanes) == 0 {
		audit.ReadinessScore = 0
		return audit
	}
	audit.ReadinessScore = round1(maxFloat(0, 100-(float64(penalties)*100/float64(len(surface.Lanes)))))
	return audit
}

func RenderClawHostWorkflowReport(surface ClawHostWorkflowSurface, audit ClawHostWorkflowSurfaceAudit) string {
	lines := []string{
		"# ClawHost Workflow Surface",
		"",
		fmt.Sprintf("- Name: %s", surface.Name),
		fmt.Sprintf("- Version: %s", surface.Version),
		fmt.Sprintf("- Source Repo: %s", surface.SourceRepo),
		fmt.Sprintf("- Reference Revision: %s", surface.ReferenceRevision),
		fmt.Sprintf("- Lane Count: %d", audit.LaneCount),
		fmt.Sprintf("- Readiness Score: %.1f", audit.ReadinessScore),
		"",
		"## Summary",
		"",
	}
	summaryKeys := make([]string, 0, len(surface.Summary))
	for key := range surface.Summary {
		summaryKeys = append(summaryKeys, key)
	}
	sort.Strings(summaryKeys)
	for _, key := range summaryKeys {
		lines = append(lines, fmt.Sprintf("- %s: %v", key, surface.Summary[key]))
	}
	lines = append(lines, "", "## Lanes", "")
	if len(surface.Lanes) == 0 {
		lines = append(lines, "- none")
	} else {
		for _, lane := range surface.Lanes {
			lines = append(lines, fmt.Sprintf("- %s (%s): stage=%s route=%s owner=%s boundary=%s batch=%s[%d/%d] approvals=%s token_session=%t device_auto_approval=%t takeover=%t skills=%t channels=%t providers=%t",
				lane.Name,
				lane.LaneID,
				lane.Stage,
				lane.Route,
				lane.Owner,
				lane.AutomationBoundary,
				lane.ParallelBatch.Name,
				lane.ParallelBatch.CanarySize,
				lane.ParallelBatch.MaxConcurrency,
				emptyFallback(strings.Join(lane.ParallelBatch.RequiredApprovals, ", "), "none"),
				lane.TokenSessionGating,
				lane.DeviceAutoApproval,
				lane.SupportsHumanTakeover,
				lane.SkillAware,
				lane.ChannelAware,
				lane.ProviderAware,
			))
			if len(lane.HandoffTeams) > 0 {
				lines = append(lines, fmt.Sprintf("  handoff_teams=%s", strings.Join(lane.HandoffTeams, ", ")))
			}
			if len(lane.Notes) > 0 {
				lines = append(lines, fmt.Sprintf("  notes=%s", strings.Join(lane.Notes, " | ")))
			}
		}
	}
	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, fmt.Sprintf("- Missing route lanes: %s", fallbackJoin(audit.MissingRouteLanes)))
	lines = append(lines, fmt.Sprintf("- Missing owner lanes: %s", fallbackJoin(audit.MissingOwnerLanes)))
	lines = append(lines, fmt.Sprintf("- Lanes without human takeover: %s", fallbackJoin(audit.LanesWithoutTakeover)))
	lines = append(lines, fmt.Sprintf("- Lanes without token/session gating: %s", fallbackJoin(audit.LanesWithoutTokenSessionGating)))
	lines = append(lines, fmt.Sprintf("- Lanes without required approvals: %s", fallbackJoin(audit.LanesWithoutRequiredApprovals)))
	lines = append(lines, fmt.Sprintf("- Lanes with invalid stage: %s", fallbackJoin(audit.LanesWithInvalidStage)))
	lines = append(lines, fmt.Sprintf("- Lanes with invalid automation policy: %s", fallbackJoin(audit.LanesWithInvalidAutomationPolicy)))
	return strings.Join(lines, "\n") + "\n"
}

func normalizedWorkflowOwner(actor string) string {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return "workflow-operator"
	}
	return actor
}

func clawhostWorkflowSignals(tasks []domain.Task) map[string]int {
	signals := map[string]int{
		"total_tasks":           len(tasks),
		"blocked_tasks":         0,
		"high_risk_tasks":       0,
		"channel_tagged_tasks":  0,
		"device_tagged_tasks":   0,
		"provider_tagged_tasks": 0,
	}
	for _, task := range tasks {
		if task.State == domain.TaskBlocked {
			signals["blocked_tasks"]++
		}
		if task.RiskLevel == domain.RiskHigh {
			signals["high_risk_tasks"]++
		}
		if value := strings.ToLower(strings.TrimSpace(task.Metadata["channel"])); value != "" {
			signals["channel_tagged_tasks"]++
		}
		if value := strings.ToLower(strings.TrimSpace(task.Metadata["device"])); value != "" {
			signals["device_tagged_tasks"]++
		}
		if value := strings.ToLower(strings.TrimSpace(task.Metadata["provider"])); value != "" {
			signals["provider_tagged_tasks"]++
		}
	}
	return signals
}

func inferParallelLimit(tasks []domain.Task) int {
	total := len(tasks)
	switch {
	case total >= 60:
		return 12
	case total >= 30:
		return 10
	case total >= 10:
		return 8
	case total > 0:
		return 6
	default:
		return 4
	}
}

func countClawHostLanes(lanes []ClawHostWorkflowLane, fn func(ClawHostWorkflowLane) bool) int {
	count := 0
	for _, lane := range lanes {
		if fn(lane) {
			count++
		}
	}
	return count
}

func maxIntWorkflow(left, right int) int {
	if left > right {
		return left
	}
	return right
}
