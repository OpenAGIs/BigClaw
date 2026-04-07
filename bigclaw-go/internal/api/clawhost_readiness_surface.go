package api

import (
	"sort"
	"strconv"
	"strings"

	"bigclaw-go/internal/domain"
)

type clawHostReadinessSurface struct {
	Integration     string                    `json:"integration"`
	Status          string                    `json:"status"`
	Filters         map[string]string         `json:"filters,omitempty"`
	SupportedChecks []string                  `json:"supported_checks"`
	Summary         clawHostReadinessSummary  `json:"summary"`
	Targets         []clawHostReadinessTarget `json:"targets,omitempty"`
}

type clawHostReadinessSummary struct {
	Targets                 int `json:"targets"`
	ReadyTargets            int `json:"ready_targets"`
	DegradedTargets         int `json:"degraded_targets"`
	AdminReadyTargets       int `json:"admin_ready_targets"`
	WebSocketReadyTargets   int `json:"websocket_ready_targets"`
	SubdomainReadyTargets   int `json:"subdomain_ready_targets"`
	UpgradeAvailableTargets int `json:"upgrade_available_targets"`
}

type clawHostReadinessTarget struct {
	TaskID             string   `json:"task_id"`
	TenantID           string   `json:"tenant_id"`
	ClawID             string   `json:"claw_id"`
	ClawName           string   `json:"claw_name"`
	Domain             string   `json:"domain,omitempty"`
	ProxyMode          string   `json:"proxy_mode,omitempty"`
	GatewayPort        int      `json:"gateway_port,omitempty"`
	CurrentVersion     string   `json:"current_version,omitempty"`
	LatestVersion      string   `json:"latest_version,omitempty"`
	VersionStatus      string   `json:"version_status,omitempty"`
	AdminUIEnabled     bool     `json:"admin_ui_enabled"`
	WebSocketReachable bool     `json:"websocket_reachable"`
	SubdomainReady     bool     `json:"subdomain_ready"`
	Reachable          bool     `json:"reachable"`
	ReviewStatus       string   `json:"review_status"`
	Warnings           []string `json:"warnings,omitempty"`
}

func clawHostReadinessSurfacePayload(tasks []domain.Task, team, project string) clawHostReadinessSurface {
	surface := clawHostReadinessSurface{
		Integration:     "clawhost",
		Status:          "idle",
		Filters: map[string]string{
			"team":    team,
			"project": project,
		},
		SupportedChecks: []string{"gateway_port", "subdomain_ready", "websocket_reachable", "admin_ui_enabled", "version_status"},
	}
	targets := make([]clawHostReadinessTarget, 0, len(tasks))
	for _, task := range tasks {
		target, ok := clawHostReadinessTargetFromTask(task)
		if !ok {
			continue
		}
		surface.Status = "active"
		targets = append(targets, target)
		surface.Summary.Targets++
		if target.ReviewStatus == "ready" {
			surface.Summary.ReadyTargets++
		} else {
			surface.Summary.DegradedTargets++
		}
		if target.AdminUIEnabled && target.Reachable {
			surface.Summary.AdminReadyTargets++
		}
		if target.WebSocketReachable {
			surface.Summary.WebSocketReadyTargets++
		}
		if target.SubdomainReady {
			surface.Summary.SubdomainReadyTargets++
		}
		if target.VersionStatus == "upgrade_available" {
			surface.Summary.UpgradeAvailableTargets++
		}
	}
	sort.SliceStable(targets, func(i, j int) bool {
		if targets[i].ReviewStatus != targets[j].ReviewStatus {
			return targets[i].ReviewStatus < targets[j].ReviewStatus
		}
		return targets[i].ClawName < targets[j].ClawName
	})
	surface.Targets = targets
	return surface
}

func clawHostReadinessTargetFromTask(task domain.Task) (clawHostReadinessTarget, bool) {
	if !strings.EqualFold(strings.TrimSpace(task.Metadata["control_plane"]), "clawhost") &&
		!strings.EqualFold(strings.TrimSpace(task.Source), "clawhost") {
		return clawHostReadinessTarget{}, false
	}
	reachable := parseBoolClawHost(task.Metadata["reachable"])
	adminUIEnabled := parseBoolClawHost(task.Metadata["admin_ui_enabled"])
	websocketReachable := parseBoolClawHost(task.Metadata["websocket_reachable"])
	subdomainReady := parseBoolClawHost(task.Metadata["subdomain_ready"])
	versionStatus := strings.TrimSpace(task.Metadata["version_status"])
	target := clawHostReadinessTarget{
		TaskID:             strings.TrimSpace(task.ID),
		TenantID:           firstNonEmpty(strings.TrimSpace(task.TenantID), task.Metadata["tenant_id"], task.Metadata["owner_user_id"]),
		ClawID:             firstNonEmpty(task.Metadata["claw_id"], strings.TrimSpace(task.ID)),
		ClawName:           firstNonEmpty(task.Metadata["claw_name"], strings.TrimSpace(task.Title)),
		Domain:             strings.TrimSpace(task.Metadata["domain"]),
		ProxyMode:          strings.TrimSpace(task.Metadata["proxy_mode"]),
		GatewayPort:        parseIntClawHost(task.Metadata["gateway_port"]),
		CurrentVersion:     strings.TrimSpace(task.Metadata["version_current"]),
		LatestVersion:      strings.TrimSpace(task.Metadata["version_latest"]),
		VersionStatus:      versionStatus,
		AdminUIEnabled:     adminUIEnabled,
		WebSocketReachable: websocketReachable,
		SubdomainReady:     subdomainReady,
		Reachable:          reachable,
	}
	warnings := make([]string, 0, 3)
	if !reachable {
		warnings = append(warnings, "claw is not reachable from the control plane")
	}
	if !subdomainReady {
		warnings = append(warnings, "subdomain health check is not yet ready")
	}
	if !websocketReachable {
		warnings = append(warnings, "websocket/admin proxy is not reachable")
	}
	if versionStatus == "upgrade_available" {
		warnings = append(warnings, "installed OpenClaw version is behind latest")
	}
	target.Warnings = warnings
	if len(warnings) == 0 {
		target.ReviewStatus = "ready"
	} else {
		target.ReviewStatus = "degraded"
	}
	return target, true
}

func parseBoolClawHost(value string) bool {
	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	return err == nil && parsed
}

func parseIntClawHost(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return parsed
}
