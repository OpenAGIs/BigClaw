package api

import (
	"encoding/json"
	"fmt"
	"os"
)

const clawHostProxyAdminValidationLanePath = "docs/reports/clawhost-proxy-admin-validation-lane.json"

type clawHostProxyAdminValidationLane struct {
	ReportPath      string                               `json:"report_path"`
	GeneratedAt     string                               `json:"generated_at,omitempty"`
	Ticket          string                               `json:"ticket,omitempty"`
	Title           string                               `json:"title,omitempty"`
	Status          string                               `json:"status,omitempty"`
	Provider        string                               `json:"provider,omitempty"`
	ValidationLane  string                               `json:"validation_lane,omitempty"`
	EvidenceSources []string                             `json:"evidence_sources,omitempty"`
	ReviewerLinks   []string                             `json:"reviewer_links,omitempty"`
	Summary         clawHostProxyAdminValidationSummary  `json:"summary"`
	Bots            []clawHostProxyAdminValidationTarget `json:"bots,omitempty"`
	Limitations     []string                             `json:"limitations,omitempty"`
	Error           string                               `json:"error,omitempty"`
}

type clawHostProxyAdminValidationSummary struct {
	AppCount               int    `json:"app_count"`
	BotCount               int    `json:"bot_count"`
	HTTPReachableBots      int    `json:"http_reachable_bots"`
	WebsocketReachableBots int    `json:"websocket_reachable_bots"`
	SubdomainReadyBots     int    `json:"subdomain_ready_bots"`
	AdminReadyBots         int    `json:"admin_ready_bots"`
	DegradedBots           int    `json:"degraded_bots"`
	ParallelProbeWidth     int    `json:"parallel_probe_width"`
	ReviewerExportStatus   string `json:"reviewer_export_status,omitempty"`
}

type clawHostProxyAdminValidationTarget struct {
	AppID            string   `json:"app_id"`
	BotID            string   `json:"bot_id"`
	Tenant           string   `json:"tenant,omitempty"`
	Region           string   `json:"region,omitempty"`
	Subdomain        string   `json:"subdomain,omitempty"`
	ProxyRoute       string   `json:"proxy_route,omitempty"`
	HTTPStatus       string   `json:"http_status,omitempty"`
	WebsocketStatus  string   `json:"websocket_status,omitempty"`
	AdminStatus      string   `json:"admin_status,omitempty"`
	ValidationStatus string   `json:"validation_status,omitempty"`
	ProbeEvidence    []string `json:"probe_evidence,omitempty"`
	ReviewerNotes    []string `json:"reviewer_notes,omitempty"`
}

func clawHostProxyAdminValidationLanePayload() clawHostProxyAdminValidationLane {
	surface := clawHostProxyAdminValidationLane{ReportPath: clawHostProxyAdminValidationLanePath}
	reportPath := resolveRepoRelativePath(clawHostProxyAdminValidationLanePath)
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
		surface.Error = fmt.Sprintf("decode %s: %v", clawHostProxyAdminValidationLanePath, err)
		return surface
	}
	surface.ReportPath = clawHostProxyAdminValidationLanePath
	return surface
}
