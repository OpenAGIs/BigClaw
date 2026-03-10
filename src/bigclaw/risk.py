from dataclasses import dataclass, field
from typing import List

from .models import Priority, RiskLevel, Task


@dataclass
class RiskFactor:
    name: str
    points: int
    reason: str


@dataclass
class RiskScore:
    level: RiskLevel
    total: int
    requires_approval: bool
    factors: List[RiskFactor] = field(default_factory=list)

    @property
    def summary(self) -> str:
        return ", ".join(f"{factor.name}={factor.points}" for factor in self.factors) or "baseline=0"


class RiskScorer:
    TOOL_POINTS = {
        "browser": 10,
        "terminal": 15,
        "github": 10,
        "deploy": 20,
        "sql": 15,
        "warehouse": 15,
        "bi": 10,
    }
    LABEL_POINTS = {
        "security": 20,
        "compliance": 20,
        "prod": 20,
        "release": 15,
        "ops": 10,
    }

    def score_task(self, task: Task) -> RiskScore:
        factors: List[RiskFactor] = []
        total = 0

        risk_points = {
            RiskLevel.LOW: 0,
            RiskLevel.MEDIUM: 30,
            RiskLevel.HIGH: 60,
        }[task.risk_level]
        total += risk_points
        if risk_points:
            factors.append(RiskFactor("risk-level", risk_points, f"declared risk level {task.risk_level.value}"))

        if task.priority == Priority.P0:
            total += 10
            factors.append(RiskFactor("priority", 10, "p0 task needs tighter controls"))

        labels = {label.lower() for label in task.labels}
        for label in sorted(labels):
            points = self.LABEL_POINTS.get(label)
            if points:
                total += points
                factors.append(RiskFactor(f"label:{label}", points, f"label {label} increases operational risk"))

        tools = {tool.lower() for tool in task.required_tools}
        for tool in sorted(tools):
            points = self.TOOL_POINTS.get(tool)
            if points:
                total += points
                factors.append(RiskFactor(f"tool:{tool}", points, f"tool {tool} expands execution surface"))

        if task.budget < 0:
            total += 20
            factors.append(RiskFactor("budget", 20, "invalid budget requires manual review"))

        level = self._level_for_total(total)
        return RiskScore(level=level, total=total, requires_approval=(level == RiskLevel.HIGH), factors=factors)

    def _level_for_total(self, total: int) -> RiskLevel:
        if total >= 60:
            return RiskLevel.HIGH
        if total >= 25:
            return RiskLevel.MEDIUM
        return RiskLevel.LOW
