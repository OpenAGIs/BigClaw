from dataclasses import dataclass
from typing import Dict, Optional

from .models import Task


@dataclass
class BudgetDecision:
    status: str
    estimated_cost: float
    remaining_budget: float
    reason: str


class CostController:
    def __init__(self, medium_hourly_costs: Optional[Dict[str, float]] = None):
        self.medium_hourly_costs = medium_hourly_costs or {
            "docker": 2.0,
            "browser": 4.0,
            "vm": 8.0,
            "none": 0.0,
        }

    def estimate_cost(self, medium: str, duration_minutes: int) -> float:
        hourly = self.medium_hourly_costs.get(medium, 0.0)
        return round(hourly * (max(0, duration_minutes) / 60.0), 2)

    def evaluate(self, task: Task, medium: str, duration_minutes: int, spent_so_far: float = 0.0) -> BudgetDecision:
        estimated = self.estimate_cost(medium, duration_minutes)
        effective_budget = float(task.budget + task.budget_override_amount)
        remaining = round(effective_budget - spent_so_far - estimated, 2)

        if effective_budget <= 0:
            return BudgetDecision(
                status="allow",
                estimated_cost=estimated,
                remaining_budget=remaining,
                reason="budget not set",
            )

        if remaining >= 0:
            return BudgetDecision(
                status="allow",
                estimated_cost=estimated,
                remaining_budget=remaining,
                reason="within budget",
            )

        downgraded_medium = "docker" if medium in {"browser", "vm"} else "none"
        if downgraded_medium != medium:
            downgraded_estimated = self.estimate_cost(downgraded_medium, duration_minutes)
            downgraded_remaining = round(effective_budget - spent_so_far - downgraded_estimated, 2)
            if downgraded_remaining >= 0:
                return BudgetDecision(
                    status="degrade",
                    estimated_cost=downgraded_estimated,
                    remaining_budget=downgraded_remaining,
                    reason=f"degrade to {downgraded_medium} to stay within budget",
                )

        return BudgetDecision(
            status="pause",
            estimated_cost=estimated,
            remaining_budget=remaining,
            reason="budget exceeded",
        )
