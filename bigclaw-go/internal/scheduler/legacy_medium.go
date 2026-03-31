package scheduler

import (
	"fmt"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/risk"
)

type LegacyMediumDecision struct {
	Medium   string `json:"medium"`
	Approved bool   `json:"approved"`
	Reason   string `json:"reason"`
}

var legacyMediumBudgetFloors = map[string]int64{
	"docker":  1000,
	"browser": 2000,
	"vm":      4000,
}

func DecideLegacyMedium(task domain.Task) LegacyMediumDecision {
	score := risk.ScoreTask(task, nil)
	decision := LegacyMediumDecision{Medium: "docker", Approved: true, Reason: "default low risk path"}

	switch {
	case task.BudgetCents < 0:
		return LegacyMediumDecision{Medium: "none", Approved: false, Reason: "invalid budget"}
	case score.Level == domain.RiskHigh:
		decision = LegacyMediumDecision{Medium: "vm", Approved: false, Reason: "requires approval for high-risk task"}
	case requiresTool(task, "browser"):
		decision = LegacyMediumDecision{Medium: "browser", Approved: true, Reason: "browser automation task"}
	case score.Level == domain.RiskMedium:
		decision = LegacyMediumDecision{Medium: "docker", Approved: true, Reason: "medium risk in docker"}
	}

	effectiveBudget := task.BudgetCents
	if effectiveBudget <= 0 {
		return decision
	}
	requiredBudget := legacyMediumBudgetFloors[decision.Medium]
	if effectiveBudget >= requiredBudget {
		return decision
	}
	if decision.Medium == "browser" && score.Level != domain.RiskHigh && effectiveBudget >= legacyMediumBudgetFloors["docker"] {
		return LegacyMediumDecision{
			Medium:   "docker",
			Approved: true,
			Reason: fmt.Sprintf(
				"budget degraded browser route to docker (budget %.1f < required %.1f)",
				float64(effectiveBudget)/100,
				float64(requiredBudget)/100,
			),
		}
	}
	return LegacyMediumDecision{
		Medium:   "none",
		Approved: false,
		Reason: fmt.Sprintf(
			"paused: budget %.1f below required %s budget %.1f",
			float64(effectiveBudget)/100,
			decision.Medium,
			float64(requiredBudget)/100,
		),
	}
}
