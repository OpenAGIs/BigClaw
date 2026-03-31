from bigclaw.cost_control import CostController
from bigclaw.models import Task


def test_cost_controller_allows_when_budget_not_set() -> None:
    task = Task(task_id="BIG-budget-0", source="linear", title="No budget", description="", budget=0)

    decision = CostController().evaluate(task, medium="browser", duration_minutes=30)

    assert decision.status == "allow"
    assert decision.estimated_cost == 2.0
    assert decision.reason == "budget not set"


def test_cost_controller_degrades_before_pausing() -> None:
    task = Task(
        task_id="BIG-budget-1",
        source="linear",
        title="Tight budget",
        description="",
        budget=1.5,
    )

    decision = CostController().evaluate(task, medium="browser", duration_minutes=30)

    assert decision.status == "degrade"
    assert decision.estimated_cost == 1.0
    assert decision.remaining_budget == 0.5
    assert decision.reason == "degrade to docker to stay within budget"
