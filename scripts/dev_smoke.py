"""Legacy Python smoke check retained only for migration-reference validation."""

from bigclaw.models import Task
from bigclaw.scheduler import Scheduler
from bigclaw.service import warn_legacy_runtime_surface


def main() -> None:
    warn_legacy_runtime_surface("scripts/dev_smoke.py", "cd bigclaw-go && go test ./... && go run ./cmd/bigclawd")
    task = Task(task_id="SMOKE-1", source="local", title="smoke", description="smoke test")
    scheduler = Scheduler()
    decision = scheduler.decide(task)
    assert decision.approved is True
    assert decision.medium in {"docker", "browser"}
    print("smoke_ok", decision.medium)


if __name__ == "__main__":
    main()
