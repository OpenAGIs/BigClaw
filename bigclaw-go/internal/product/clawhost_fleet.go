package product

import (
	"fmt"
	"sort"
	"strings"
)

var validClawHostBotStatuses = map[string]struct{}{
	"created":  {},
	"starting": {},
	"running":  {},
	"stopped":  {},
	"error":    {},
}

type ClawHostControlPlane struct {
	Name                 string   `json:"name"`
	Mode                 string   `json:"mode"`
	BackingStore         string   `json:"backing_store"`
	KubernetesNative     bool     `json:"kubernetes_native"`
	ProxyModes           []string `json:"proxy_modes,omitempty"`
	SubdomainRouting     bool     `json:"subdomain_routing"`
	PerBotIngressNeeded  bool     `json:"per_bot_ingress_needed"`
	ConfigBidirectional  bool     `json:"config_bidirectional_sync"`
	MultiTenantSupported bool     `json:"multi_tenant_supported"`
}

type ClawHostAppInventory struct {
	AppID                    string `json:"app_id"`
	Name                     string `json:"name"`
	TenantID                 string `json:"tenant_id"`
	Owner                    string `json:"owner"`
	APIAccessTokenManaged    bool   `json:"api_access_token_managed"`
	BotCount                 int    `json:"bot_count"`
	RunningBotCount          int    `json:"running_bot_count"`
	SupportsUserBotOwnership bool   `json:"supports_user_bot_ownership"`
}

type ClawHostBotInventory struct {
	BotID                 string   `json:"bot_id"`
	AppID                 string   `json:"app_id"`
	UserID                string   `json:"user_id"`
	Name                  string   `json:"name"`
	Slug                  string   `json:"slug"`
	Status                string   `json:"status"`
	Endpoint              string   `json:"endpoint,omitempty"`
	Subdomain             string   `json:"subdomain,omitempty"`
	PodIsolation          bool     `json:"pod_isolation"`
	ServiceIsolation      bool     `json:"service_isolation"`
	Channels              []string `json:"channels,omitempty"`
	ModelProviders        []string `json:"model_providers,omitempty"`
	DeviceAutoApproval    bool     `json:"device_auto_approval"`
	SupportsDynamicSkills bool     `json:"supports_dynamic_skills"`
}

type ClawHostInventoryFacets struct {
	ByStatus         map[string]int `json:"by_status,omitempty"`
	ByProvider       map[string]int `json:"by_provider,omitempty"`
	ByChannel        map[string]int `json:"by_channel,omitempty"`
	ByTenant         map[string]int `json:"by_tenant,omitempty"`
	ByLifecycleState map[string]int `json:"by_lifecycle_state,omitempty"`
}

type ClawHostFleetSummary struct {
	AppCount             int `json:"app_count"`
	BotCount             int `json:"bot_count"`
	RunningBots          int `json:"running_bots"`
	ErrorBots            int `json:"error_bots"`
	MultiTenantApps      int `json:"multi_tenant_apps"`
	BotsWithProxyDomain  int `json:"bots_with_proxy_domain"`
	BotsWithProviders    int `json:"bots_with_providers"`
	AutoApprovalEnabled  int `json:"auto_approval_enabled"`
	IsolationReady       int `json:"isolation_ready"`
	ParallelRolloutReady int `json:"parallel_rollout_ready"`
}

type ClawHostFleetInventory struct {
	SurfaceID        string                  `json:"surface_id"`
	Version          string                  `json:"version"`
	SourceRepository string                  `json:"source_repository"`
	ControlPlane     ClawHostControlPlane    `json:"control_plane"`
	LifecycleActions []string                `json:"lifecycle_actions,omitempty"`
	Apps             []ClawHostAppInventory  `json:"apps,omitempty"`
	Bots             []ClawHostBotInventory  `json:"bots,omitempty"`
	Summary          ClawHostFleetSummary    `json:"summary"`
	Facets           ClawHostInventoryFacets `json:"facets"`
}

type ClawHostFleetAudit struct {
	SurfaceID                    string   `json:"surface_id"`
	Version                      string   `json:"version"`
	AppCount                     int      `json:"app_count"`
	BotCount                     int      `json:"bot_count"`
	AppsWithoutBots              []string `json:"apps_without_bots,omitempty"`
	OrphanBots                   []string `json:"orphan_bots,omitempty"`
	BotsMissingOwnership         []string `json:"bots_missing_ownership,omitempty"`
	BotsMissingProxyEndpoint     []string `json:"bots_missing_proxy_endpoint,omitempty"`
	BotsMissingSubdomain         []string `json:"bots_missing_subdomain,omitempty"`
	BotsMissingProviders         []string `json:"bots_missing_providers,omitempty"`
	BotsWithoutIsolation         []string `json:"bots_without_isolation,omitempty"`
	BotsWithoutLifecycleCoverage []string `json:"bots_without_lifecycle_coverage,omitempty"`
	UnknownStatuses              []string `json:"unknown_statuses,omitempty"`
	ReadinessScore               float64  `json:"readiness_score"`
	ControlPlaneReady            bool     `json:"control_plane_ready"`
}

func BuildDefaultClawHostFleetSurface() ClawHostFleetInventory {
	apps := []ClawHostAppInventory{
		{
			AppID:                    "app-platform",
			Name:                     "Platform Agents",
			TenantID:                 "tenant-platform",
			Owner:                    "platform-ops",
			APIAccessTokenManaged:    true,
			SupportsUserBotOwnership: true,
		},
		{
			AppID:                    "app-growth",
			Name:                     "Growth Assistants",
			TenantID:                 "tenant-growth",
			Owner:                    "growth-ops",
			APIAccessTokenManaged:    true,
			SupportsUserBotOwnership: true,
		},
	}
	bots := []ClawHostBotInventory{
		{
			BotID:                 "bot-platform-1",
			AppID:                 "app-platform",
			UserID:                "user-001",
			Name:                  "platform-release-bot",
			Slug:                  "platform-release-bot",
			Status:                "running",
			Endpoint:              "http://clawhost.local/proxy/bot-platform-1/",
			Subdomain:             "platform-release-bot.clawhost.loc",
			PodIsolation:          true,
			ServiceIsolation:      true,
			Channels:              []string{"slack", "telegram"},
			ModelProviders:        []string{"anthropic", "openai"},
			DeviceAutoApproval:    true,
			SupportsDynamicSkills: true,
		},
		{
			BotID:                 "bot-growth-1",
			AppID:                 "app-growth",
			UserID:                "user-002",
			Name:                  "growth-campaign-bot",
			Slug:                  "growth-campaign-bot",
			Status:                "starting",
			Endpoint:              "http://clawhost.local/proxy/bot-growth-1/",
			Subdomain:             "growth-campaign-bot.clawhost.loc",
			PodIsolation:          true,
			ServiceIsolation:      true,
			Channels:              []string{"discord", "teams"},
			ModelProviders:        []string{"openai", "minimax"},
			DeviceAutoApproval:    true,
			SupportsDynamicSkills: true,
		},
	}
	return BuildClawHostFleetSurface(apps, bots)
}

func BuildClawHostFleetSurface(apps []ClawHostAppInventory, bots []ClawHostBotInventory) ClawHostFleetInventory {
	surface := ClawHostFleetInventory{
		SurfaceID:        "BIG-PAR-287",
		Version:          "go-v1",
		SourceRepository: "https://github.com/fastclaw-ai/clawhost",
		ControlPlane: ClawHostControlPlane{
			Name:                 "ClawHost",
			Mode:                 "kubernetes-native bot fleet hosting",
			BackingStore:         "postgresql",
			KubernetesNative:     true,
			ProxyModes:           []string{"http", "websocket"},
			SubdomainRouting:     true,
			PerBotIngressNeeded:  false,
			ConfigBidirectional:  true,
			MultiTenantSupported: true,
		},
		LifecycleActions: []string{"create", "start", "stop", "restart", "upgrade", "delete"},
		Apps:             append([]ClawHostAppInventory(nil), apps...),
		Bots:             append([]ClawHostBotInventory(nil), bots...),
		Facets: ClawHostInventoryFacets{
			ByStatus:         map[string]int{},
			ByProvider:       map[string]int{},
			ByChannel:        map[string]int{},
			ByTenant:         map[string]int{},
			ByLifecycleState: map[string]int{},
		},
	}
	sort.SliceStable(surface.Apps, func(i, j int) bool { return surface.Apps[i].AppID < surface.Apps[j].AppID })
	sort.SliceStable(surface.Bots, func(i, j int) bool { return surface.Bots[i].BotID < surface.Bots[j].BotID })

	appIndex := map[string]int{}
	tenantSet := map[string]struct{}{}
	for index, app := range surface.Apps {
		app.BotCount = 0
		app.RunningBotCount = 0
		surface.Apps[index] = app
		appIndex[app.AppID] = index
		if tenant := strings.TrimSpace(app.TenantID); tenant != "" {
			surface.Facets.ByTenant[tenant]++
			tenantSet[tenant] = struct{}{}
		}
	}

	for _, bot := range surface.Bots {
		status := normalizedClawHostStatus(bot.Status)
		surface.Facets.ByStatus[status]++
		surface.Facets.ByLifecycleState[status]++
		if strings.TrimSpace(bot.Subdomain) != "" || strings.TrimSpace(bot.Endpoint) != "" {
			surface.Summary.BotsWithProxyDomain++
		}
		if len(bot.ModelProviders) > 0 {
			surface.Summary.BotsWithProviders++
		}
		if bot.DeviceAutoApproval {
			surface.Summary.AutoApprovalEnabled++
		}
		if bot.PodIsolation && bot.ServiceIsolation {
			surface.Summary.IsolationReady++
		}
		if bot.Status == "running" || bot.Status == "starting" {
			surface.Summary.ParallelRolloutReady++
		}
		if status == "running" {
			surface.Summary.RunningBots++
		}
		if status == "error" {
			surface.Summary.ErrorBots++
		}
		for _, provider := range dedupeNonEmptyStrings(bot.ModelProviders) {
			surface.Facets.ByProvider[provider]++
		}
		for _, channel := range dedupeNonEmptyStrings(bot.Channels) {
			surface.Facets.ByChannel[channel]++
		}
		if index, ok := appIndex[bot.AppID]; ok {
			surface.Apps[index].BotCount++
			if status == "running" {
				surface.Apps[index].RunningBotCount++
			}
		}
	}

	surface.Summary.AppCount = len(surface.Apps)
	surface.Summary.BotCount = len(surface.Bots)
	surface.Summary.MultiTenantApps = len(tenantSet)
	return surface
}

func AuditClawHostFleetSurface(inventory ClawHostFleetInventory) ClawHostFleetAudit {
	audit := ClawHostFleetAudit{
		SurfaceID:       inventory.SurfaceID,
		Version:         inventory.Version,
		AppCount:        len(inventory.Apps),
		BotCount:        len(inventory.Bots),
		UnknownStatuses: []string{},
	}
	appIndex := map[string]ClawHostAppInventory{}
	for _, app := range inventory.Apps {
		appIndex[app.AppID] = app
		if app.BotCount == 0 {
			audit.AppsWithoutBots = append(audit.AppsWithoutBots, app.AppID)
		}
	}

	lifecycleCoverage := map[string]struct{}{}
	for _, action := range inventory.LifecycleActions {
		lifecycleCoverage[strings.TrimSpace(strings.ToLower(action))] = struct{}{}
	}
	for _, required := range []string{"create", "start", "stop", "restart", "upgrade", "delete"} {
		if _, ok := lifecycleCoverage[required]; !ok {
			audit.BotsWithoutLifecycleCoverage = append(audit.BotsWithoutLifecycleCoverage, required)
		}
	}

	for _, bot := range inventory.Bots {
		if _, ok := appIndex[bot.AppID]; !ok {
			audit.OrphanBots = append(audit.OrphanBots, bot.BotID)
		}
		if strings.TrimSpace(bot.UserID) == "" {
			audit.BotsMissingOwnership = append(audit.BotsMissingOwnership, bot.BotID)
		}
		if strings.TrimSpace(bot.Endpoint) == "" {
			audit.BotsMissingProxyEndpoint = append(audit.BotsMissingProxyEndpoint, bot.BotID)
		}
		if strings.TrimSpace(bot.Subdomain) == "" {
			audit.BotsMissingSubdomain = append(audit.BotsMissingSubdomain, bot.BotID)
		}
		if len(dedupeNonEmptyStrings(bot.ModelProviders)) == 0 {
			audit.BotsMissingProviders = append(audit.BotsMissingProviders, bot.BotID)
		}
		if !bot.PodIsolation || !bot.ServiceIsolation {
			audit.BotsWithoutIsolation = append(audit.BotsWithoutIsolation, bot.BotID)
		}
		if _, ok := validClawHostBotStatuses[normalizedClawHostStatus(bot.Status)]; !ok {
			audit.UnknownStatuses = append(audit.UnknownStatuses, bot.Status)
		}
	}

	sort.Strings(audit.AppsWithoutBots)
	sort.Strings(audit.OrphanBots)
	sort.Strings(audit.BotsMissingOwnership)
	sort.Strings(audit.BotsMissingProxyEndpoint)
	sort.Strings(audit.BotsMissingSubdomain)
	sort.Strings(audit.BotsMissingProviders)
	sort.Strings(audit.BotsWithoutIsolation)
	sort.Strings(audit.BotsWithoutLifecycleCoverage)
	sort.Strings(audit.UnknownStatuses)

	penalties := len(audit.AppsWithoutBots) +
		len(audit.OrphanBots) +
		len(audit.BotsMissingOwnership) +
		len(audit.BotsMissingProxyEndpoint) +
		len(audit.BotsMissingSubdomain) +
		len(audit.BotsMissingProviders) +
		len(audit.BotsWithoutIsolation) +
		len(audit.BotsWithoutLifecycleCoverage) +
		len(audit.UnknownStatuses)
	denominator := maxFloat(1, float64(fleetMaxInt(1, len(inventory.Bots))+len(inventory.Apps)))
	audit.ReadinessScore = round1(maxFloat(0, 100-(float64(penalties)*100/denominator)))
	audit.ControlPlaneReady = penalties == 0 &&
		inventory.ControlPlane.KubernetesNative &&
		inventory.ControlPlane.SubdomainRouting &&
		!inventory.ControlPlane.PerBotIngressNeeded &&
		inventory.ControlPlane.MultiTenantSupported
	return audit
}

func RenderClawHostFleetReport(inventory ClawHostFleetInventory, audit ClawHostFleetAudit) string {
	lines := []string{
		"# ClawHost Fleet Inventory & Control Plane Report",
		"",
		fmt.Sprintf("- Surface: %s", inventory.SurfaceID),
		fmt.Sprintf("- Version: %s", inventory.Version),
		fmt.Sprintf("- Source Repository: %s", inventory.SourceRepository),
		fmt.Sprintf("- App Count: %d", inventory.Summary.AppCount),
		fmt.Sprintf("- Bot Count: %d", inventory.Summary.BotCount),
		fmt.Sprintf("- Running Bots: %d", inventory.Summary.RunningBots),
		fmt.Sprintf("- Error Bots: %d", inventory.Summary.ErrorBots),
		fmt.Sprintf("- Readiness Score: %.1f", audit.ReadinessScore),
		fmt.Sprintf("- Control Plane Ready: %t", audit.ControlPlaneReady),
		"",
		"## Control Plane",
		"",
		fmt.Sprintf("- Name: %s", inventory.ControlPlane.Name),
		fmt.Sprintf("- Mode: %s", inventory.ControlPlane.Mode),
		fmt.Sprintf("- Backing Store: %s", inventory.ControlPlane.BackingStore),
		fmt.Sprintf("- Kubernetes Native: %t", inventory.ControlPlane.KubernetesNative),
		fmt.Sprintf("- Proxy Modes: %s", strings.Join(inventory.ControlPlane.ProxyModes, ", ")),
		fmt.Sprintf("- Subdomain Routing: %t", inventory.ControlPlane.SubdomainRouting),
		fmt.Sprintf("- Per-bot Ingress Needed: %t", inventory.ControlPlane.PerBotIngressNeeded),
		fmt.Sprintf("- Bidirectional Config Sync: %t", inventory.ControlPlane.ConfigBidirectional),
		fmt.Sprintf("- Multi-tenant Supported: %t", inventory.ControlPlane.MultiTenantSupported),
		"",
		"## Lifecycle Actions",
		"",
	}
	if len(inventory.LifecycleActions) == 0 {
		lines = append(lines, "- none")
	} else {
		lines = append(lines, fmt.Sprintf("- %s", strings.Join(inventory.LifecycleActions, ", ")))
	}

	lines = append(lines, "", "## App Inventory", "")
	if len(inventory.Apps) == 0 {
		lines = append(lines, "- none")
	} else {
		for _, app := range inventory.Apps {
			lines = append(lines, fmt.Sprintf("- %s (%s): tenant=%s owner=%s bots=%d running=%d token_managed=%t user_ownership=%t",
				app.Name, app.AppID, emptyFallback(app.TenantID, "unassigned"), emptyFallback(app.Owner, "unassigned"), app.BotCount, app.RunningBotCount, app.APIAccessTokenManaged, app.SupportsUserBotOwnership))
		}
	}

	lines = append(lines, "", "## Bot Inventory", "")
	if len(inventory.Bots) == 0 {
		lines = append(lines, "- none")
	} else {
		for _, bot := range inventory.Bots {
			lines = append(lines, fmt.Sprintf("- %s (%s): app=%s user=%s status=%s pod_isolation=%t service_isolation=%t subdomain=%s providers=%s channels=%s auto_approval=%t dynamic_skills=%t",
				bot.Name,
				bot.BotID,
				emptyFallback(bot.AppID, "unassigned"),
				emptyFallback(bot.UserID, "unassigned"),
				normalizedClawHostStatus(bot.Status),
				bot.PodIsolation,
				bot.ServiceIsolation,
				emptyFallback(bot.Subdomain, "none"),
				emptyFallback(strings.Join(dedupeNonEmptyStrings(bot.ModelProviders), ", "), "none"),
				emptyFallback(strings.Join(dedupeNonEmptyStrings(bot.Channels), ", "), "none"),
				bot.DeviceAutoApproval,
				bot.SupportsDynamicSkills,
			))
		}
	}

	lines = append(lines, "", "## Inventory Facets", "")
	lines = append(lines, fmt.Sprintf("- By Status: %s", renderFleetFacetMap(inventory.Facets.ByStatus)))
	lines = append(lines, fmt.Sprintf("- By Provider: %s", renderFleetFacetMap(inventory.Facets.ByProvider)))
	lines = append(lines, fmt.Sprintf("- By Channel: %s", renderFleetFacetMap(inventory.Facets.ByChannel)))
	lines = append(lines, fmt.Sprintf("- By Tenant: %s", renderFleetFacetMap(inventory.Facets.ByTenant)))

	lines = append(lines, "", "## Audit Gaps", "")
	lines = append(lines, fmt.Sprintf("- Apps without bots: %s", emptyFallback(strings.Join(audit.AppsWithoutBots, ", "), "none")))
	lines = append(lines, fmt.Sprintf("- Orphan bots: %s", emptyFallback(strings.Join(audit.OrphanBots, ", "), "none")))
	lines = append(lines, fmt.Sprintf("- Bots missing ownership: %s", emptyFallback(strings.Join(audit.BotsMissingOwnership, ", "), "none")))
	lines = append(lines, fmt.Sprintf("- Bots missing proxy endpoint: %s", emptyFallback(strings.Join(audit.BotsMissingProxyEndpoint, ", "), "none")))
	lines = append(lines, fmt.Sprintf("- Bots missing subdomain: %s", emptyFallback(strings.Join(audit.BotsMissingSubdomain, ", "), "none")))
	lines = append(lines, fmt.Sprintf("- Bots missing providers: %s", emptyFallback(strings.Join(audit.BotsMissingProviders, ", "), "none")))
	lines = append(lines, fmt.Sprintf("- Bots without isolation: %s", emptyFallback(strings.Join(audit.BotsWithoutIsolation, ", "), "none")))
	lines = append(lines, fmt.Sprintf("- Missing lifecycle actions: %s", emptyFallback(strings.Join(audit.BotsWithoutLifecycleCoverage, ", "), "none")))
	lines = append(lines, fmt.Sprintf("- Unknown statuses: %s", emptyFallback(strings.Join(audit.UnknownStatuses, ", "), "none")))
	return strings.Join(lines, "\n") + "\n"
}

// Backward-compatible aliases kept for parallel branches still calling the
// older inventory-first names.
func BuildDefaultClawHostFleetInventory() ClawHostFleetInventory {
	return BuildDefaultClawHostFleetSurface()
}

func BuildClawHostFleetInventory(apps []ClawHostAppInventory, bots []ClawHostBotInventory) ClawHostFleetInventory {
	return BuildClawHostFleetSurface(apps, bots)
}

func AuditClawHostFleetInventory(inventory ClawHostFleetInventory) ClawHostFleetAudit {
	return AuditClawHostFleetSurface(inventory)
}

func normalizedClawHostStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		return "unknown"
	}
	return status
}

func dedupeNonEmptyStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(strings.ToLower(value))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func renderFleetFacetMap(values map[string]int) string {
	if len(values) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, values[key]))
	}
	return strings.Join(parts, "; ")
}

func fleetMaxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
