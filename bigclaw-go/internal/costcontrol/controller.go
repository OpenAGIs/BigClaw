package costcontrol

import (
	"math"

	"bigclaw-go/internal/domain"
)

type BudgetDecision struct {
	Status          string  `json:"status"`
	EstimatedCost   float64 `json:"estimated_cost"`
	RemainingBudget float64 `json:"remaining_budget"`
	Reason          string  `json:"reason"`
}

type Controller struct {
	mediumHourlyCosts map[string]float64
}

func New() *Controller {
	return &Controller{
		mediumHourlyCosts: map[string]float64{
			"docker":  2.0,
			"browser": 4.0,
			"vm":      8.0,
			"none":    0.0,
		},
	}
}

func (c *Controller) EstimateCost(medium string, durationMinutes int) float64 {
	if c == nil {
		c = New()
	}
	hourly := c.mediumHourlyCosts[medium]
	minutes := maxInt(durationMinutes, 0)
	return round2(hourly * (float64(minutes) / 60.0))
}

func (c *Controller) Evaluate(task domain.Task, medium string, durationMinutes int, spentSoFar float64) BudgetDecision {
	if c == nil {
		c = New()
	}
	estimated := c.EstimateCost(medium, durationMinutes)
	effectiveBudget := (float64(task.BudgetCents) / 100.0) + task.BudgetOverrideAmount
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

	downgradedMedium := downgradedMediumFor(medium)
	if downgradedMedium != medium {
		downgradedEstimated := c.EstimateCost(downgradedMedium, durationMinutes)
		downgradedRemaining := round2(effectiveBudget - spentSoFar - downgradedEstimated)
		if downgradedRemaining >= 0 {
			return BudgetDecision{
				Status:          "degrade",
				EstimatedCost:   downgradedEstimated,
				RemainingBudget: downgradedRemaining,
				Reason:          "degrade to " + downgradedMedium + " to stay within budget",
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

func downgradedMediumFor(medium string) string {
	switch medium {
	case "browser", "vm":
		return "docker"
	default:
		return "none"
	}
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
