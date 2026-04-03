## BIGCLAW-182 Workpad

### Plan

- Audit the current queue control center data model, renderer, and contract surface for issue-scoped changes only.
- Add bulk retry state, failure attribution rollups, and manual takeover entry points to the queue control center.
- Extend targeted control center tests to cover the new analytics and rendered output.
- Run targeted validation commands, capture exact commands and results, then commit and push the branch.

### Acceptance

- Queue control center exposes batch retry readiness for blocked or failed queue work, including eligible task IDs and blocked reasons when a task cannot join the batch retry set.
- Queue control center groups actionable failures into an attribution view so operators can distinguish approval, tool, repo sync, and unknown failure causes.
- Queue control center surfaces a manual takeover entry point for tasks that should move from retry/escalation into human ownership.
- Existing queue control center behavior remains intact for queue depth, priority/risk rollups, and per-task actions.

### Validation

- `pytest tests/test_control_center.py`
- If contract coverage changes: `pytest tests/test_execution_contract.py`
