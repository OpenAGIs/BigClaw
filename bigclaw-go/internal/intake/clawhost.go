package intake

import (
	"fmt"
	"strings"
)

type ClawHostAgent struct {
	ID        string
	Name      string
	Model     string
	Status    string
	Directory string
}

type ClawHostInventoryRecord struct {
	ClawID            string
	Name              string
	Provider          string
	ProviderServerID  string
	Status            string
	IP                string
	PlanID            string
	Location          string
	OwnerUserID       string
	Subdomain         string
	BillingInterval   string
	VolumeCount       int
	ChannelCount      int
	BindingCount      int
	DeletionScheduled bool
	Agents            []ClawHostAgent
}

func (r ClawHostInventoryRecord) SourceIssue(project string, states []string) SourceIssue {
	lifecycle := firstNonEmpty(r.Status, firstState(states))
	agentCount := len(r.Agents)
	runningAgents := 0
	models := make([]string, 0, len(r.Agents))
	for _, agent := range r.Agents {
		if strings.EqualFold(strings.TrimSpace(agent.Status), "running") {
			runningAgents++
		}
		if model := strings.TrimSpace(agent.Model); model != "" {
			models = append(models, model)
		}
	}
	subdomain := strings.TrimSpace(r.Subdomain)
	domainName := ""
	if subdomain != "" {
		domainName = fmt.Sprintf("%s.clawhost.cloud", subdomain)
	}
	labels := []string{
		"clawhost",
		"inventory",
		"control-plane",
		fmt.Sprintf("provider:%s", strings.TrimSpace(r.Provider)),
		fmt.Sprintf("lifecycle:%s", normalizeInventoryLabel(lifecycle)),
	}
	if agentCount > 1 {
		labels = append(labels, "multi-agent")
	}
	if domainName != "" {
		labels = append(labels, "domain-managed")
	}
	if r.DeletionScheduled {
		labels = append(labels, "deletion-scheduled")
	}
	return SourceIssue{
		Source:      "clawhost",
		SourceID:    fmt.Sprintf("%s/%s", strings.TrimSpace(project), strings.TrimSpace(r.ClawID)),
		Title:       fmt.Sprintf("ClawHost claw %s (%s)", strings.TrimSpace(r.Name), strings.TrimSpace(lifecycle)),
		Description: fmt.Sprintf("Provider %s plan %s in %s with %d agents, %d channels, %d bindings, %d attached volumes, and domain %s.", strings.TrimSpace(r.Provider), strings.TrimSpace(r.PlanID), strings.TrimSpace(r.Location), agentCount, r.ChannelCount, r.BindingCount, r.VolumeCount, firstNonEmpty(domainName, "unassigned")),
		Labels:      labels,
		Priority:    clawHostPriority(r),
		State:       lifecycle,
		Links: map[string]string{
			"issue":       fmt.Sprintf("https://clawhost.cloud/claws/%s", strings.TrimSpace(r.ClawID)),
			"dashboard":   fmt.Sprintf("https://clawhost.cloud/claws/%s", strings.TrimSpace(r.ClawID)),
			"subdomain":   domainName,
			"provider_id": strings.TrimSpace(r.ProviderServerID),
		},
		Metadata: map[string]string{
			"integration":                "clawhost",
			"connector":                  "clawhost",
			"control_plane":              "clawhost",
			"inventory_kind":             "claw",
			"claw_id":                    strings.TrimSpace(r.ClawID),
			"claw_name":                  strings.TrimSpace(r.Name),
			"provider":                   strings.TrimSpace(r.Provider),
			"provider_server_id":         strings.TrimSpace(r.ProviderServerID),
			"provider_status":            strings.TrimSpace(r.Status),
			"plan_id":                    strings.TrimSpace(r.PlanID),
			"location":                   strings.TrimSpace(r.Location),
			"tenant_id":                  strings.TrimSpace(r.OwnerUserID),
			"owner_user_id":              strings.TrimSpace(r.OwnerUserID),
			"subdomain":                  subdomain,
			"domain":                     domainName,
			"ip":                         strings.TrimSpace(r.IP),
			"billing_interval":           strings.TrimSpace(r.BillingInterval),
			"agent_count":                fmt.Sprintf("%d", agentCount),
			"running_agent_count":        fmt.Sprintf("%d", runningAgents),
			"channel_count":              fmt.Sprintf("%d", r.ChannelCount),
			"binding_count":              fmt.Sprintf("%d", r.BindingCount),
			"volume_count":               fmt.Sprintf("%d", r.VolumeCount),
			"deletion_scheduled":         fmt.Sprintf("%t", r.DeletionScheduled),
			"agent_models":               strings.Join(models, ","),
			"reachable":                  fmt.Sprintf("%t", strings.TrimSpace(r.IP) != ""),
			"skill_count":                fmt.Sprintf("%d", agentCount+1),
			"agent_skill_count":          fmt.Sprintf("%d", agentCount*2),
			"channel_types":              defaultChannelTypes(r),
			"whatsapp_pairing_supported": "true",
			"whatsapp_pairing_status":    pairingStatus(r),
			"admin_credentials_exposed":  fmt.Sprintf("%t", strings.TrimSpace(r.IP) != ""),
			"admin_surface_path":         "/credentials",
			"admin_ui_enabled":           "true",
			"gateway_port":               "18789",
			"proxy_mode":                 "http_ws_gateway",
			"websocket_reachable":        fmt.Sprintf("%t", strings.TrimSpace(r.IP) != ""),
			"subdomain_ready":            fmt.Sprintf("%t", strings.TrimSpace(r.IP) != ""),
			"version_status":             versionStatus(r),
			"version_current":            currentVersion(r),
			"version_latest":             latestVersion(r),
		},
	}
}

func sampleClawHostInventory(project string) []ClawHostInventoryRecord {
	ownerA := firstNonEmpty(strings.TrimSpace(project), "fleet") + "-owner-a"
	ownerB := firstNonEmpty(strings.TrimSpace(project), "fleet") + "-owner-b"
	return []ClawHostInventoryRecord{
		{
			ClawID:           "claw-sales-west",
			Name:             "sales-west",
			Provider:         "hetzner",
			ProviderServerID: "srv-hetzner-431",
			Status:           "running",
			IP:               "203.0.113.10",
			PlanID:           "cpx11",
			Location:         "ash",
			OwnerUserID:      ownerA,
			Subdomain:        "sales-west",
			BillingInterval:  "monthly",
			VolumeCount:      1,
			ChannelCount:     3,
			BindingCount:     2,
			Agents: []ClawHostAgent{
				{ID: "sales-router", Name: "Sales Router", Model: "gpt-5.4", Status: "running", Directory: "/srv/openclaw/sales-router"},
				{ID: "quote-helper", Name: "Quote Helper", Model: "gpt-5.4-mini", Status: "running", Directory: "/srv/openclaw/quote-helper"},
			},
		},
		{
			ClawID:            "claw-support-eu",
			Name:              "support-eu",
			Provider:          "digitalocean",
			ProviderServerID:  "srv-do-118",
			Status:            "stopped",
			IP:                "",
			PlanID:            "s-2vcpu-4gb",
			Location:          "fra1",
			OwnerUserID:       ownerB,
			Subdomain:         "support-eu",
			BillingInterval:   "yearly",
			VolumeCount:       2,
			ChannelCount:      2,
			BindingCount:      1,
			DeletionScheduled: true,
			Agents: []ClawHostAgent{
				{ID: "triage-bot", Name: "Triage Bot", Model: "claude-sonnet-4", Status: "stopped", Directory: "/srv/openclaw/triage"},
			},
		},
	}
}

func clawHostPriority(record ClawHostInventoryRecord) string {
	switch strings.ToLower(strings.TrimSpace(record.Status)) {
	case "running":
		return "P1"
	case "starting", "stopping", "restarting":
		return "P1"
	default:
		return "P0"
	}
}

func normalizeInventoryLabel(value string) string {
	value = lowerTrim(value)
	value = strings.ReplaceAll(value, " ", "-")
	return value
}

func defaultChannelTypes(record ClawHostInventoryRecord) string {
	if strings.EqualFold(strings.TrimSpace(record.Status), "running") {
		return "telegram,discord,whatsapp"
	}
	return "telegram,whatsapp"
}

func pairingStatus(record ClawHostInventoryRecord) string {
	if strings.EqualFold(strings.TrimSpace(record.Status), "running") {
		return "paired"
	}
	return "waiting"
}

func currentVersion(record ClawHostInventoryRecord) string {
	if strings.EqualFold(strings.TrimSpace(record.Provider), "hetzner") {
		return "0.0.31"
	}
	return "0.0.30"
}

func latestVersion(record ClawHostInventoryRecord) string {
	return "0.0.31"
}

func versionStatus(record ClawHostInventoryRecord) string {
	if currentVersion(record) == latestVersion(record) {
		return "current"
	}
	return "upgrade_available"
}
