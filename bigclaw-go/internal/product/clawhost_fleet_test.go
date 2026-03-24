package product

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildDefaultClawHostFleetInventoryIsControlPlaneReady(t *testing.T) {
	inventory := BuildDefaultClawHostFleetSurface()
	audit := AuditClawHostFleetSurface(inventory)
	if inventory.SurfaceID != "BIG-PAR-287" || inventory.Version != "go-v1" {
		t.Fatalf("unexpected fleet metadata: %+v", inventory)
	}
	if inventory.Filters["team"] != "" || inventory.Filters["project"] != "" {
		t.Fatalf("expected default fleet filters to stay empty, got %+v", inventory.Filters)
	}
	if inventory.Summary.AppCount != 2 || inventory.Summary.BotCount != 2 || inventory.Summary.RunningBots != 1 {
		t.Fatalf("unexpected fleet summary: %+v", inventory.Summary)
	}
	if inventory.ControlPlane.Name != "ClawHost" || !inventory.ControlPlane.KubernetesNative || !inventory.ControlPlane.SubdomainRouting || inventory.ControlPlane.PerBotIngressNeeded {
		t.Fatalf("unexpected control-plane posture: %+v", inventory.ControlPlane)
	}
	if !audit.ControlPlaneReady {
		t.Fatalf("expected default fleet inventory to be control-plane ready, got %+v", audit)
	}
}

func TestBuildDefaultClawHostFleetInventoryCompatibilityAlias(t *testing.T) {
	aliasedInventory := BuildDefaultClawHostFleetInventory()
	surfaceInventory := BuildDefaultClawHostFleetSurface()
	if !reflect.DeepEqual(aliasedInventory, surfaceInventory) {
		t.Fatalf("expected default fleet alias builder to match surface builder, got alias=%+v surface=%+v", aliasedInventory, surfaceInventory)
	}

	aliasedAudit := AuditClawHostFleetSurface(aliasedInventory)
	surfaceAudit := AuditClawHostFleetSurface(surfaceInventory)
	if !reflect.DeepEqual(aliasedAudit, surfaceAudit) {
		t.Fatalf("expected default fleet alias audit to match surface audit, got alias=%+v surface=%+v", aliasedAudit, surfaceAudit)
	}
}

func TestClawHostFleetInventoryCompatibilityAliases(t *testing.T) {
	apps := []ClawHostAppInventory{
		{AppID: "app-zeta", Name: "zeta", TenantID: "tenant-z", Team: "platform", Project: "apollo"},
		{AppID: "app-alpha", Name: "alpha", TenantID: "tenant-a", Team: "growth", Project: "campaigns"},
	}
	bots := []ClawHostBotInventory{
		{BotID: "bot-zeta", AppID: "app-zeta", Name: "zeta-bot", UserID: "user-z", Status: "running", Endpoint: "http://clawhost.local/proxy/bot-zeta/", Subdomain: "zeta.clawhost.loc", PodIsolation: true, ServiceIsolation: true, ModelProviders: []string{"openai"}},
		{BotID: "bot-alpha", AppID: "app-alpha", Name: "alpha-bot", UserID: "user-a", Status: "starting", Endpoint: "http://clawhost.local/proxy/bot-alpha/", Subdomain: "alpha.clawhost.loc", PodIsolation: true, ServiceIsolation: true, ModelProviders: []string{"anthropic"}},
	}

	aliasedInventory := BuildClawHostFleetInventory(apps, bots)
	surfaceInventory := BuildClawHostFleetSurface(apps, bots)
	if aliasedInventory.SurfaceID != surfaceInventory.SurfaceID || aliasedInventory.Version != surfaceInventory.Version {
		t.Fatalf("expected fleet alias builder metadata to match surface builder, got alias=%+v surface=%+v", aliasedInventory, surfaceInventory)
	}
	if !reflect.DeepEqual(aliasedInventory.Apps, surfaceInventory.Apps) {
		t.Fatalf("expected fleet alias builder apps to match surface builder, got alias=%+v surface=%+v", aliasedInventory.Apps, surfaceInventory.Apps)
	}
	if !reflect.DeepEqual(aliasedInventory.Bots, surfaceInventory.Bots) {
		t.Fatalf("expected fleet alias builder bots to match surface builder, got alias=%+v surface=%+v", aliasedInventory.Bots, surfaceInventory.Bots)
	}
	if aliasedInventory.Summary != surfaceInventory.Summary {
		t.Fatalf("expected fleet alias builder summary to match surface builder, got alias=%+v surface=%+v", aliasedInventory.Summary, surfaceInventory.Summary)
	}

	aliasedAudit := AuditClawHostFleetInventory(aliasedInventory)
	surfaceAudit := AuditClawHostFleetSurface(surfaceInventory)
	if !reflect.DeepEqual(aliasedAudit, surfaceAudit) {
		t.Fatalf("expected fleet alias audit to match surface audit, got alias=%+v surface=%+v", aliasedAudit, surfaceAudit)
	}
}

func TestAuditClawHostFleetInventoryDetectsCoverageGaps(t *testing.T) {
	inventory := BuildDefaultClawHostFleetSurface()
	inventory.LifecycleActions = []string{"create", "start", "stop"}
	inventory.Bots[0].AppID = "missing-app"
	inventory.Bots[0].UserID = ""
	inventory.Bots[0].Endpoint = ""
	inventory.Bots[0].Subdomain = ""
	inventory.Bots[0].ModelProviders = nil
	inventory.Bots[0].PodIsolation = false
	inventory.Bots[0].Status = "paused"
	inventory.Apps[1].BotCount = 0
	audit := AuditClawHostFleetSurface(inventory)

	if audit.ControlPlaneReady {
		t.Fatalf("expected control-plane audit to fail, got %+v", audit)
	}
	if len(audit.OrphanBots) == 0 || len(audit.BotsMissingOwnership) == 0 || len(audit.BotsMissingProxyEndpoint) == 0 || len(audit.BotsMissingSubdomain) == 0 || len(audit.BotsMissingProviders) == 0 || len(audit.BotsWithoutIsolation) == 0 || len(audit.BotsWithoutLifecycleCoverage) == 0 || len(audit.UnknownStatuses) == 0 {
		t.Fatalf("expected comprehensive coverage gaps, got %+v", audit)
	}
	if audit.ReadinessScore >= 100 {
		t.Fatalf("expected readiness score penalty, got %+v", audit)
	}
}

func TestFilterClawHostFleetSurface(t *testing.T) {
	inventory := BuildDefaultClawHostFleetSurface()

	t.Run("team and project", func(t *testing.T) {
		filtered := FilterClawHostFleetSurface(inventory, "platform", "apollo")
		if filtered.Filters["team"] != "platform" || filtered.Filters["project"] != "apollo" {
			t.Fatalf("expected scoped fleet filters, got %+v", filtered.Filters)
		}
		if filtered.Summary.AppCount != 1 || filtered.Summary.BotCount != 1 || filtered.Summary.RunningBots != 1 {
			t.Fatalf("unexpected scoped summary: %+v", filtered.Summary)
		}
		if len(filtered.Apps) != 1 || filtered.Apps[0].AppID != "app-platform" {
			t.Fatalf("expected only platform app, got %+v", filtered.Apps)
		}
		if len(filtered.Bots) != 1 || filtered.Bots[0].BotID != "bot-platform-1" {
			t.Fatalf("expected only platform bot, got %+v", filtered.Bots)
		}
	})

	t.Run("project only", func(t *testing.T) {
		filtered := FilterClawHostFleetSurface(inventory, "", "campaigns")
		if filtered.Summary.AppCount != 1 || filtered.Summary.BotCount != 1 || filtered.Summary.RunningBots != 0 {
			t.Fatalf("unexpected project-only scoped summary: %+v", filtered.Summary)
		}
		if len(filtered.Apps) != 1 || filtered.Apps[0].AppID != "app-growth" {
			t.Fatalf("expected only growth app, got %+v", filtered.Apps)
		}
		if len(filtered.Bots) != 1 || filtered.Bots[0].BotID != "bot-growth-1" {
			t.Fatalf("expected only growth bot, got %+v", filtered.Bots)
		}
	})

	t.Run("no matches", func(t *testing.T) {
		filtered := FilterClawHostFleetSurface(inventory, "support", "phoenix")
		if filtered.Filters["team"] != "support" || filtered.Filters["project"] != "phoenix" {
			t.Fatalf("expected no-match fleet scope to persist filters, got %+v", filtered.Filters)
		}
		if filtered.Summary.AppCount != 0 || filtered.Summary.BotCount != 0 || filtered.Summary.RunningBots != 0 {
			t.Fatalf("expected empty scoped summary, got %+v", filtered.Summary)
		}
		if len(filtered.Apps) != 0 || len(filtered.Bots) != 0 {
			t.Fatalf("expected no scoped inventory, got apps=%+v bots=%+v", filtered.Apps, filtered.Bots)
		}
	})
}

func TestRenderClawHostFleetReport(t *testing.T) {
	inventory := BuildDefaultClawHostFleetSurface()
	audit := AuditClawHostFleetSurface(inventory)
	report := RenderClawHostFleetReport(inventory, audit)
	for _, want := range []string{
		"# ClawHost Fleet Inventory & Control Plane Report",
		"## Filters",
		"- none",
		"Source Repository: https://github.com/fastclaw-ai/clawhost",
		"Per-bot Ingress Needed: false",
		"platform-release-bot",
		"By Provider: anthropic=1; minimax=1; openai=2",
		"Control Plane Ready: true",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report, got %s", want, report)
		}
	}
}

func TestRenderClawHostFleetReportHandlesEmptyInventory(t *testing.T) {
	inventory := FilterClawHostFleetSurface(BuildDefaultClawHostFleetSurface(), "support", "phoenix")
	audit := AuditClawHostFleetSurface(inventory)
	report := RenderClawHostFleetReport(inventory, audit)

	for _, want := range []string{
		"# ClawHost Fleet Inventory & Control Plane Report",
		"## Filters",
		"- project: phoenix",
		"- team: support",
		"App Count: 0",
		"Bot Count: 0",
		"Running Bots: 0",
		"## App Inventory",
		"## Bot Inventory",
		"- none",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in empty fleet report, got %s", want, report)
		}
	}
}

func TestRenderClawHostFleetReportHandlesSparseInventoryAndEmptyFacets(t *testing.T) {
	inventory := ClawHostFleetInventory{
		SurfaceID:        "BIG-PAR-364",
		Version:          "go-v1",
		SourceRepository: "https://github.com/fastclaw-ai/clawhost",
		ControlPlane:     ClawHostControlPlane{},
		Apps: []ClawHostAppInventory{
			{AppID: "app-a", Name: "App A"},
		},
		Bots: []ClawHostBotInventory{
			{BotID: "bot-a", Name: "Bot A", Status: "", ModelProviders: nil, Channels: nil},
		},
		Facets: ClawHostInventoryFacets{},
		Summary: ClawHostFleetSummary{
			AppCount: 1,
			BotCount: 1,
		},
	}
	audit := ClawHostFleetAudit{
		SurfaceID:         inventory.SurfaceID,
		Version:           inventory.Version,
		ReadinessScore:    0,
		ControlPlaneReady: false,
	}

	report := RenderClawHostFleetReport(inventory, audit)
	for _, want := range []string{
		"## Filters",
		"- none",
		"## Control Plane",
		"- Proxy Modes: ",
		"## Lifecycle Actions",
		"- none",
		"App A (app-a): team=unassigned project=unassigned tenant=unassigned owner=unassigned",
		"Bot A (bot-a): app=unassigned team=unassigned project=unassigned user=unassigned status=unknown",
		"providers=none channels=none",
		"## Inventory Facets",
		"By Status: none",
		"By Provider: none",
		"By Channel: none",
		"By Tenant: none",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected fleet edge-case report to contain %q, got %s", want, report)
		}
	}
}

func TestNormalizedClawHostStatusFallbacks(t *testing.T) {
	for _, tc := range []struct {
		name   string
		input  string
		expect string
	}{
		{name: "trims and lowers", input: " Running ", expect: "running"},
		{name: "empty becomes unknown", input: "", expect: "unknown"},
		{name: "whitespace becomes unknown", input: "   ", expect: "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizedClawHostStatus(tc.input); got != tc.expect {
				t.Fatalf("expected normalized status %q, got %q", tc.expect, got)
			}
		})
	}
}

func TestDedupeNonEmptyStringsNormalizesAndSorts(t *testing.T) {
	got := dedupeNonEmptyStrings([]string{
		" OpenAI ",
		"",
		"anthropic",
		"OPENAI",
		"  ",
		"Anthropic ",
		"minimax",
	})

	want := []string{"anthropic", "minimax", "openai"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected deduped values %+v, got %+v", want, got)
	}
}
