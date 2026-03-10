from dataclasses import dataclass
from typing import Optional

from .models import Task


@dataclass
class BudgetPolicy:
    max_cost: Optional[float] = None
    max_tokens: Optional[int] = None
    max_runtime_seconds: Optional[int] = None
    max_workers: Optional[int] = None


@dataclass
class BudgetDecision:
    allowed: bool
    reason: str
    dimension: str = ""


class BudgetGuard:
    def __init__(self, policy: Optional[BudgetPolicy] = None):
        self.policy = policy or BudgetPolicy()

    def evaluate(self, task: Task) -> BudgetDecision:
        if task.budget < 0:
            return BudgetDecision(False, "invalid budget", "cost")
        if task.estimated_tokens < 0:
            return BudgetDecision(False, "invalid token estimate", "tokens")
        if task.estimated_runtime_seconds < 0:
            return BudgetDecision(False, "invalid runtime estimate", "runtime")
        if task.required_workers <= 0:
            return BudgetDecision(False, "invalid worker request", "workers")

        checks = [
            (self.policy.max_cost, task.budget, "cost", "budget limit exceeded"),
            (self.policy.max_tokens, task.estimated_tokens, "tokens", "token budget exceeded"),
            (
                self.policy.max_runtime_seconds,
                task.estimated_runtime_seconds,
                "runtime",
                "runtime budget exceeded",
            ),
            (self.policy.max_workers, task.required_workers, "workers", "worker budget exceeded"),
        ]
        for limit, actual, dimension, reason in checks:
            if limit is not None and actual > limit:
                return BudgetDecision(False, f"{reason}: {actual} > {limit}", dimension)

        return BudgetDecision(True, "within budget")
