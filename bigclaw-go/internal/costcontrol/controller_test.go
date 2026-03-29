package costcontrol

import (
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestControllerDegradesWhenHighMediumGoesOverBudget(t *testing.T) {
	controller := New()
	task := domain.Task{ID: "BIG-503-1", Source: "local", Title: "budget", BudgetCents: 70}

	decision := controller.Evaluate(task, "browser", 15, 0)

	if decision.Status != "degrade" {
		t.Fatalf("expected degrade decision, got %+v", decision)
	}
	if !strings.Contains(decision.Reason, "degrade to docker") {
		t.Fatalf("expected degrade reason, got %+v", decision)
	}
	if decision.RemainingBudget < 0 {
		t.Fatalf("expected non-negative remaining budget, got %+v", decision)
	}
}

func TestControllerPausesWhenEvenDockerExceedsBudget(t *testing.T) {
	controller := New()
	task := domain.Task{ID: "BIG-503-2", Source: "local", Title: "budget", BudgetCents: 20}

	decision := controller.Evaluate(task, "browser", 30, 0)

	if decision.Status != "pause" || decision.Reason != "budget exceeded" {
		t.Fatalf("expected pause because budget exceeded, got %+v", decision)
	}
}

func TestControllerRespectsBudgetOverrideAmount(t *testing.T) {
	controller := New()
	task := domain.Task{
		ID:                   "BIG-503-3",
		Source:               "local",
		Title:                "budget",
		BudgetCents:          20,
		BudgetOverrideAmount: 0.6,
	}

	decision := controller.Evaluate(task, "browser", 10, 0)
	if decision.Status != "allow" && decision.Status != "degrade" {
		t.Fatalf("expected allow or degrade with budget override, got %+v", decision)
	}
}
