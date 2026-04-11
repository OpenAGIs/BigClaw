package api

import (
	"encoding/json"
	"fmt"
	"os"
)

const clawHostFleetInventorySurfacePath = "docs/reports/clawhost-fleet-inventory-surface.json"

type clawHostFleetInventorySurface struct {
	ReportPath      string                        `json:"report_path"`
	GeneratedAt     string                        `json:"generated_at,omitempty"`
	Ticket          string                        `json:"ticket,omitempty"`
	Title           string                        `json:"title,omitempty"`
	Status          string                        `json:"status,omitempty"`
	Provider        string                        `json:"provider,omitempty"`
	SourceKind      string                        `json:"source_kind,omitempty"`
	EvidenceSources []string                      `json:"evidence_sources,omitempty"`
	ReviewerLinks   []string                      `json:"reviewer_links,omitempty"`
	Summary         clawHostFleetInventorySummary `json:"summary"`
	Apps            []clawHostFleetInventoryApp   `json:"apps,omitempty"`
	Bots            []clawHostFleetInventoryBot   `json:"bots,omitempty"`
	Limitations     []string                      `json:"limitations,omitempty"`
	Error           string                        `json:"error,omitempty"`
}

type clawHostFleetInventorySummary struct {
	AppCount          int `json:"app_count"`
	BotCount          int `json:"bot_count"`
	ActiveBots        int `json:"active_bots"`
	SuspendedBots     int `json:"suspended_bots"`
	DegradedBots      int `json:"degraded_bots"`
	TenantCount       int `json:"tenant_count"`
	DomainCount       int `json:"domain_count"`
	OwnershipTeams    int `json:"ownership_teams"`
	ReviewerReadyApps int `json:"reviewer_ready_apps"`
}

type clawHostFleetInventoryApp struct {
	AppID            string   `json:"app_id"`
	Tenant           string   `json:"tenant,omitempty"`
	Owner            string   `json:"owner,omitempty"`
	Reviewer         string   `json:"reviewer,omitempty"`
	DefaultDomain    string   `json:"default_domain,omitempty"`
	DefaultSubdomain string   `json:"default_subdomain,omitempty"`
	ProviderPolicy   string   `json:"provider_policy,omitempty"`
	BotCount         int      `json:"bot_count"`
	LifecycleStates  []string `json:"lifecycle_states,omitempty"`
}

type clawHostFleetInventoryBot struct {
	BotID          string   `json:"bot_id"`
	AppID          string   `json:"app_id,omitempty"`
	Tenant         string   `json:"tenant,omitempty"`
	LifecycleState string   `json:"lifecycle_state,omitempty"`
	Owner          string   `json:"owner,omitempty"`
	Reviewer       string   `json:"reviewer,omitempty"`
	RuntimeRegion  string   `json:"runtime_region,omitempty"`
	ServiceName    string   `json:"service_name,omitempty"`
	Domain         string   `json:"domain,omitempty"`
	Subdomain      string   `json:"subdomain,omitempty"`
	Capabilities   []string `json:"capabilities,omitempty"`
	Labels         []string `json:"labels,omitempty"`
}

func clawHostFleetInventorySurfacePayload() clawHostFleetInventorySurface {
	surface := clawHostFleetInventorySurface{ReportPath: clawHostFleetInventorySurfacePath}
	reportPath := resolveRepoRelativePath(clawHostFleetInventorySurfacePath)
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
		surface.Error = fmt.Sprintf("decode %s: %v", clawHostFleetInventorySurfacePath, err)
		return surface
	}
	surface.ReportPath = clawHostFleetInventorySurfacePath
	return surface
}
