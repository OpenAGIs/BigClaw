package scheduler

import (
	"fmt"
	"math"

	"bigclaw-go/internal/domain"
)

type BudgetDecision struct {
	Status          string
	EstimatedCost   float64
	RemainingBudget float64
	Reason          string
}

type CostController struct {
	mediumHourlyCosts map[string]float64
}

func NewCostController(mediumHourlyCosts map[string]float64) CostController {
	if mediumHourlyCosts == nil {
		mediumHourlyCosts = map[string]float64{
			"docker":  2.0,
			"browser": 4.0,
			"vm":      8.0,
			"none":    0.0,
		}
	}
	return CostController{mediumHourlyCosts: mediumHourlyCosts}
}

func (c CostController) EstimateCost(medium string, durationMinutes int) float64 {
	hourly := c.mediumHourlyCosts[medium]
	return round2(hourly * (math.Max(0, float64(durationMinutes)) / 60.0))
}

func (c CostController) Evaluate(task domain.Task, medium string, durationMinutes int, spentSoFar float64) BudgetDecision {
	estimated := c.EstimateCost(medium, durationMinutes)
	effectiveBudget := float64(task.BudgetCents)/100 + task.BudgetOverrideAmount
	remaining := round2(effectiveBudget - spentSoFar - estimated)

	if effectiveBudget <= 0 {
		return BudgetDecision{
			Status:          "allow",
			EstimatedCost:   estimated,
			RemainingBudget: remaining,
			Reason:          "budget not set",
		}
	}

	if remaining >= 0 {
		return BudgetDecision{
			Status:          "allow",
			EstimatedCost:   estimated,
			RemainingBudget: remaining,
			Reason:          "within budget",
		}
	}

	downgradedMedium := "none"
	if medium == "browser" || medium == "vm" {
		downgradedMedium = "docker"
	}
	if downgradedMedium != medium {
		downgradedEstimated := c.EstimateCost(downgradedMedium, durationMinutes)
		downgradedRemaining := round2(effectiveBudget - spentSoFar - downgradedEstimated)
		if downgradedRemaining >= 0 {
			return BudgetDecision{
				Status:          "degrade",
				EstimatedCost:   downgradedEstimated,
				RemainingBudget: downgradedRemaining,
				Reason:          fmt.Sprintf("degrade to %s to stay within budget", downgradedMedium),
			}
		}
	}

	return BudgetDecision{
		Status:          "pause",
		EstimatedCost:   estimated,
		RemainingBudget: remaining,
		Reason:          "budget exceeded",
	}
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
