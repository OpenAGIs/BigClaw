package scheduler

import (
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestCostControllerDegradesWhenBrowserMediumExceedsBudget(t *testing.T) {
	controller := NewCostController(nil)
	task := domain.Task{ID: "BIG-503-1", BudgetCents: 70}

	decision := controller.Evaluate(task, "browser", 15, 0)

	if decision.Status != "degrade" {
		t.Fatalf("expected degrade status, got %+v", decision)
	}
	if !strings.Contains(decision.Reason, "degrade to docker") {
		t.Fatalf("expected degrade reason, got %+v", decision)
	}
	if decision.RemainingBudget < 0 {
		t.Fatalf("expected non-negative remaining budget, got %+v", decision)
	}
}

func TestCostControllerPausesWhenEvenDockerExceedsBudget(t *testing.T) {
	controller := NewCostController(nil)
	task := domain.Task{ID: "BIG-503-2", BudgetCents: 20}

	decision := controller.Evaluate(task, "browser", 30, 0)

	if decision.Status != "pause" {
		t.Fatalf("expected pause status, got %+v", decision)
	}
	if decision.Reason != "budget exceeded" {
		t.Fatalf("expected budget exceeded reason, got %+v", decision)
	}
}

func TestCostControllerRespectsBudgetOverrideAmount(t *testing.T) {
	controller := NewCostController(nil)
	task := domain.Task{
		ID:                   "BIG-503-3",
		BudgetCents:          20,
		BudgetOverrideAmount: 0.6,
	}

	decision := controller.Evaluate(task, "browser", 10, 0)

	if decision.Status != "allow" && decision.Status != "degrade" {
		t.Fatalf("expected allow or degrade status, got %+v", decision)
	}
}
