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

### Validation Results

- `pytest tests/test_control_center.py` -> failed in workspace because `pytest` executable was unavailable: `zsh:1: command not found: pytest`
- `python3 -m pytest tests/test_control_center.py` -> passed: `3 passed in 0.07s`
- `python3 -m pytest tests/test_execution_contract.py` -> passed: `7 passed in 0.07s`
- `python3 -m pytest tests/test_ui_review.py` -> passed: `28 passed in 0.21s`
- `python3 -m pytest tests/test_console_ia.py` -> passed: `11 passed in 0.07s`
- `python3 -m pytest tests/test_control_center.py tests/test_execution_contract.py tests/test_ui_review.py` -> passed: `38 passed in 0.09s`
- `python3 -m pytest tests/test_operations.py` -> passed: `20 passed in 0.10s`
- `python3 -m pytest tests/test_control_center.py` -> passed: `3 passed in 0.06s`
- `python3 -m pytest tests/test_planning.py` -> passed: `14 passed in 0.07s`
- `python3 -m pytest tests/test_control_center.py tests/test_operations.py` -> passed: `23 passed in 0.08s`
- `python3 -m pytest tests/test_control_center.py tests/test_execution_contract.py tests/test_ui_review.py tests/test_console_ia.py tests/test_operations.py tests/test_planning.py` -> passed: `83 passed in 0.12s`

### Status

- Branch: `bigclaw-182-control-center`
- Latest commit: `5e7d46dfaaa1763ae2de846fe641688854133bc0`
