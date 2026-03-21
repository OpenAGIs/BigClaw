from bigclaw.cost_control import CostController
from bigclaw.models import Task


def test_big503_cost_controller_degrades_when_high_medium_over_budget():
    controller = CostController()
    task = Task(task_id="BIG-503-1", source="local", title="budget", description="", budget=0.7)

    decision = controller.evaluate(task, medium="browser", duration_minutes=15)

    assert decision.status == "degrade"
    assert "degrade to docker" in decision.reason
    assert decision.remaining_budget >= 0


def test_big503_cost_controller_pauses_when_even_docker_exceeds_budget():
    controller = CostController()
    task = Task(task_id="BIG-503-2", source="local", title="budget", description="", budget=0.2)

    decision = controller.evaluate(task, medium="browser", duration_minutes=30)

    assert decision.status == "pause"
    assert decision.reason == "budget exceeded"


def test_big503_cost_controller_respects_budget_override_amount():
    controller = CostController()
    task = Task(
        task_id="BIG-503-3",
        source="local",
        title="budget",
        description="",
        budget=0.2,
        budget_override_amount=0.6,
    )

    decision = controller.evaluate(task, medium="browser", duration_minutes=10)

    assert decision.status in {"allow", "degrade"}
