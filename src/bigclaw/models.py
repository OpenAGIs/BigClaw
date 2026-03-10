from dataclasses import dataclass, field
from enum import Enum
from typing import List, Dict


class TaskState(str, Enum):
    TODO = "Todo"
    IN_PROGRESS = "In Progress"
    BLOCKED = "Blocked"
    DONE = "Done"
    FAILED = "Failed"


class RiskLevel(str, Enum):
    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"


class Priority(int, Enum):
    P0 = 0
    P1 = 1
    P2 = 2


@dataclass
class Task:
    task_id: str
    source: str
    title: str
    description: str
    labels: List[str] = field(default_factory=list)
    priority: Priority = Priority.P2
    state: TaskState = TaskState.TODO
    risk_level: RiskLevel = RiskLevel.LOW
    budget: float = 0.0
    required_tools: List[str] = field(default_factory=list)
    acceptance_criteria: List[str] = field(default_factory=list)
    validation_plan: List[str] = field(default_factory=list)

    def to_dict(self) -> Dict:
        return {
            "task_id": self.task_id,
            "source": self.source,
            "title": self.title,
            "description": self.description,
            "labels": self.labels,
            "priority": int(self.priority),
            "state": self.state.value,
            "risk_level": self.risk_level.value,
            "budget": self.budget,
            "required_tools": self.required_tools,
            "acceptance_criteria": self.acceptance_criteria,
            "validation_plan": self.validation_plan,
        }
