package product

import (
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
