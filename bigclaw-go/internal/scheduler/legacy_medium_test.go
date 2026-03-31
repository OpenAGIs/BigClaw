package scheduler

import (
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestLegacyMediumDecision(t *testing.T) {
	t.Run("high risk requires approval", func(t *testing.T) {
		decision := DecideLegacyMedium(domain.Task{ID: "x", RiskLevel: domain.RiskHigh})
		if decision.Medium != "vm" || decision.Approved {
			t.Fatalf("expected vm pending approval, got %+v", decision)
		}
	})

	t.Run("browser routes to browser", func(t *testing.T) {
		decision := DecideLegacyMedium(domain.Task{ID: "y", RequiredTools: []string{"browser"}})
		if decision.Medium != "browser" || !decision.Approved {
			t.Fatalf("expected approved browser route, got %+v", decision)
		}
	})

	t.Run("budget degrades browser to docker", func(t *testing.T) {
		decision := DecideLegacyMedium(domain.Task{ID: "z", RequiredTools: []string{"browser"}, BudgetCents: 1500})
		if decision.Medium != "docker" || !decision.Approved || !strings.Contains(decision.Reason, "budget degraded browser route to docker") {
			t.Fatalf("expected docker degradation, got %+v", decision)
		}
	})

	t.Run("low budget pauses task", func(t *testing.T) {
		decision := DecideLegacyMedium(domain.Task{ID: "b", BudgetCents: 500})
		if decision.Medium != "none" || decision.Approved || decision.Reason != "paused: budget 5.0 below required docker budget 10.0" {
			t.Fatalf("expected paused low-budget decision, got %+v", decision)
		}
	})
}
