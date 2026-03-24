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

func TestRenderClawHostFleetReport(t *testing.T) {
	inventory := BuildDefaultClawHostFleetSurface()
	audit := AuditClawHostFleetSurface(inventory)
	report := RenderClawHostFleetReport(inventory, audit)
	for _, want := range []string{
		"# ClawHost Fleet Inventory & Control Plane Report",
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

func TestClawHostFleetSurfaceTracksErrorBots(t *testing.T) {
	inventory := BuildClawHostFleetSurface(
		[]ClawHostAppInventory{{AppID: "app-1", Name: "Platform", TenantID: "tenant-1", Owner: "ops"}},
		[]ClawHostBotInventory{{BotID: "bot-1", AppID: "app-1", UserID: "user-1", Name: "platform-bot", Status: "error", Endpoint: "http://proxy/bot-1", Subdomain: "bot.example", PodIsolation: true, ServiceIsolation: true}},
	)

	if inventory.Summary.ErrorBots != 1 {
		t.Fatalf("expected error bot summary to increment, got %+v", inventory.Summary)
	}
	if inventory.Facets.ByStatus["error"] != 1 {
		t.Fatalf("expected error status facet, got %+v", inventory.Facets.ByStatus)
	}
}

func TestRenderClawHostFleetReportEmptySections(t *testing.T) {
	inventory := ClawHostFleetInventory{
		SurfaceID:        "BIG-PAR-287",
		Version:          "go-v1",
		SourceRepository: "https://github.com/fastclaw-ai/clawhost",
		ControlPlane: ClawHostControlPlane{
			Name:                 "ClawHost",
			Mode:                 "kubernetes-native bot fleet hosting",
			BackingStore:         "postgresql",
			KubernetesNative:     true,
			SubdomainRouting:     true,
			MultiTenantSupported: true,
		},
	}
	report := RenderClawHostFleetReport(inventory, ClawHostFleetAudit{})

	for _, want := range []string{
		"## Lifecycle Actions\n\n- none",
		"## App Inventory\n\n- none",
		"## Bot Inventory\n\n- none",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected empty-state block %q in report, got %s", want, report)
		}
	}
}

func TestClawHostFleetInventoryAliasesMatchSurfaceHelpers(t *testing.T) {
	apps := []ClawHostAppInventory{
		{AppID: "app-2", Name: "Growth", TenantID: "tenant-2", Owner: "growth"},
		{AppID: "app-1", Name: "Platform", TenantID: "tenant-1", Owner: "platform"},
	}
	bots := []ClawHostBotInventory{
		{BotID: "bot-2", AppID: "app-2", UserID: "user-2", Name: "growth-bot", Status: "starting", Endpoint: "http://proxy/bot-2", Subdomain: "growth.example", PodIsolation: true, ServiceIsolation: true, Channels: []string{"discord", "discord"}, ModelProviders: []string{"openai", "openai"}},
		{BotID: "bot-1", AppID: "app-1", UserID: "user-1", Name: "platform-bot", Status: "running", Endpoint: "http://proxy/bot-1", Subdomain: "platform.example", PodIsolation: true, ServiceIsolation: true, Channels: []string{"slack"}, ModelProviders: []string{"anthropic"}},
	}

	if got, want := BuildDefaultClawHostFleetInventory(), BuildDefaultClawHostFleetSurface(); !reflect.DeepEqual(got, want) {
		t.Fatalf("default inventory alias mismatch: got %+v want %+v", got, want)
	}

	inventory := BuildClawHostFleetInventory(apps, bots)
	if want := BuildClawHostFleetSurface(apps, bots); !reflect.DeepEqual(inventory, want) {
		t.Fatalf("inventory alias mismatch: got %+v want %+v", inventory, want)
	}

	if got, want := AuditClawHostFleetInventory(inventory), AuditClawHostFleetSurface(inventory); !reflect.DeepEqual(got, want) {
		t.Fatalf("audit alias mismatch: got %+v want %+v", got, want)
	}
}

func TestClawHostFleetHelpers(t *testing.T) {
	if got := normalizedClawHostStatus("  RUNNING "); got != "running" {
		t.Fatalf("normalized running status = %q, want %q", got, "running")
	}
	if got := normalizedClawHostStatus("   "); got != "unknown" {
		t.Fatalf("blank status fallback = %q, want %q", got, "unknown")
	}
	if got := strings.Join(dedupeNonEmptyStrings([]string{" OpenAI ", "anthropic", "openai", "", "Anthropic"}), ","); got != "anthropic,openai" {
		t.Fatalf("unexpected deduped values: %s", got)
	}
	if got := renderFleetFacetMap(nil); got != "none" {
		t.Fatalf("nil facet map = %q, want %q", got, "none")
	}
	if got := renderFleetFacetMap(map[string]int{"zulu": 1, "alpha": 2}); got != "alpha=2; zulu=1" {
		t.Fatalf("unexpected sorted facet map: %s", got)
	}
	if got := fleetMaxInt(2, 5); got != 5 {
		t.Fatalf("fleetMaxInt(2, 5) = %d, want 5", got)
	}
	if got := fleetMaxInt(7, 3); got != 7 {
		t.Fatalf("fleetMaxInt(7, 3) = %d, want 7", got)
	}
}
