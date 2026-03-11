from bigclaw.models import Task
from bigclaw.scheduler import Scheduler


def main() -> None:
    task = Task(task_id="SMOKE-1", source="local", title="smoke", description="smoke test")
    scheduler = Scheduler()
    decision = scheduler.decide(task)
    assert decision.approved is True
    assert decision.medium in {"docker", "browser"}
    print("smoke_ok", decision.medium)


if __name__ == "__main__":
    main()
