package product

import (
	"sort"
	"strconv"
	"strings"

	"bigclaw-go/internal/domain"
)

type ClawHostWorkflowItem struct {
	TaskID             string   `json:"task_id"`
	TenantID           string   `json:"tenant_id"`
	ClawID             string   `json:"claw_id"`
	ClawName           string   `json:"claw_name"`
	SkillsEnabled      int      `json:"skills_enabled"`
	AgentSkillCount    int      `json:"agent_skill_count"`
	Channels           []string `json:"channels,omitempty"`
	WhatsAppPairing    string   `json:"whatsapp_pairing"`
	CredentialsExposed bool     `json:"credentials_exposed"`
	CredentialsPath    string   `json:"credentials_path,omitempty"`
	TakeoverRequired   bool     `json:"takeover_required"`
	ReviewReason       string   `json:"review_reason"`
}

type ClawHostWorkflowSurface struct {
	Integration       string                  `json:"integration"`
	Status            string                  `json:"status"`
	SupportedChannels []string                `json:"supported_channels"`
	Summary           ClawHostWorkflowSummary `json:"summary"`
	ReviewQueue       []ClawHostWorkflowItem  `json:"review_queue,omitempty"`
}

type ClawHostWorkflowSummary struct {
	WorkflowItems     int `json:"workflow_items"`
	Tenants           int `json:"tenants"`
	PairingApprovals  int `json:"pairing_approvals"`
	CredentialReviews int `json:"credential_reviews"`
	TakeoverRequired  int `json:"takeover_required"`
	ChannelMutations  int `json:"channel_mutations"`
	SkillMutations    int `json:"skill_mutations"`
}

func BuildClawHostWorkflowSurface(tasks []domain.Task) ClawHostWorkflowSurface {
	surface := ClawHostWorkflowSurface{
		Integration:       "clawhost",
		Status:            "idle",
		SupportedChannels: []string{"whatsapp", "telegram", "discord", "slack", "signal"},
	}
	items := make([]ClawHostWorkflowItem, 0, len(tasks))
	tenants := map[string]struct{}{}
	for _, task := range tasks {
		item, ok := clawHostWorkflowItem(task)
		if !ok {
			continue
		}
		surface.Status = "active"
		items = append(items, item)
		tenants[item.TenantID] = struct{}{}
		surface.Summary.WorkflowItems++
		if item.WhatsAppPairing == "waiting" || item.WhatsAppPairing == "qr_ready" {
			surface.Summary.PairingApprovals++
		}
		if item.CredentialsExposed {
			surface.Summary.CredentialReviews++
		}
		if item.TakeoverRequired {
			surface.Summary.TakeoverRequired++
		}
		if len(item.Channels) > 0 {
			surface.Summary.ChannelMutations++
		}
		if item.SkillsEnabled > 0 || item.AgentSkillCount > 0 {
			surface.Summary.SkillMutations++
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].TakeoverRequired != items[j].TakeoverRequired {
			return items[i].TakeoverRequired
		}
		if items[i].WhatsAppPairing != items[j].WhatsAppPairing {
			return items[i].WhatsAppPairing < items[j].WhatsAppPairing
		}
		return items[i].ClawName < items[j].ClawName
	})
	surface.ReviewQueue = items
	surface.Summary.Tenants = len(tenants)
	return surface
}

func clawHostWorkflowItem(task domain.Task) (ClawHostWorkflowItem, bool) {
	if !strings.EqualFold(strings.TrimSpace(task.Metadata["control_plane"]), "clawhost") &&
		!strings.EqualFold(strings.TrimSpace(task.Source), "clawhost") {
		return ClawHostWorkflowItem{}, false
	}
	channels := splitCSVWorkflow(task.Metadata["channel_types"])
	pairing := firstNonEmptyWorkflow(task.Metadata["whatsapp_pairing_status"], "waiting")
	credentialsExposed := parseBoolWorkflow(task.Metadata["admin_credentials_exposed"])
	takeoverRequired := parseBoolWorkflow(task.Metadata["clawhost_takeover_required"]) || credentialsExposed || pairing == "waiting" || pairing == "qr_ready"
	return ClawHostWorkflowItem{
		TaskID:             strings.TrimSpace(task.ID),
		TenantID:           firstNonEmptyWorkflow(strings.TrimSpace(task.TenantID), task.Metadata["tenant_id"], task.Metadata["owner_user_id"], "unassigned"),
		ClawID:             firstNonEmptyWorkflow(task.Metadata["claw_id"], strings.TrimSpace(task.ID)),
		ClawName:           firstNonEmptyWorkflow(task.Metadata["claw_name"], strings.TrimSpace(task.Title)),
		SkillsEnabled:      parseIntWorkflow(task.Metadata["skill_count"]),
		AgentSkillCount:    parseIntWorkflow(task.Metadata["agent_skill_count"]),
		Channels:           channels,
		WhatsAppPairing:    pairing,
		CredentialsExposed: credentialsExposed,
		CredentialsPath:    strings.TrimSpace(task.Metadata["admin_surface_path"]),
		TakeoverRequired:   takeoverRequired,
		ReviewReason:       workflowReason(channels, pairing, credentialsExposed, takeoverRequired),
	}, true
}

func workflowReason(channels []string, pairing string, credentialsExposed bool, takeoverRequired bool) string {
	reasons := make([]string, 0, 4)
	if len(channels) > 0 {
		reasons = append(reasons, "channel config requires review across active IM integrations")
	}
	if pairing == "waiting" || pairing == "qr_ready" {
		reasons = append(reasons, "WhatsApp pairing still needs human completion")
	}
	if credentialsExposed {
		reasons = append(reasons, "credential access should stay takeover-gated")
	}
	if takeoverRequired && len(reasons) == 0 {
		reasons = append(reasons, "workflow requires explicit human takeover before mutating bot config")
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "skills and channel posture are aligned")
	}
	return strings.Join(reasons, "; ")
}

func splitCSVWorkflow(value string) []string {
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

func firstNonEmptyWorkflow(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func parseBoolWorkflow(value string) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	return err == nil && parsed
}

func parseIntWorkflow(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return parsed
}
