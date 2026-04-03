## BIGCLAW-182 Workpad

### Plan

- Audit the queue control center model, builder, renderer, and targeted tests to keep the slice scoped to this issue.
- Extend the queue control center with operator-facing batch retry readiness, failure attribution rollups, and manual takeover entry metadata.
- Update targeted contract and rendering tests to lock the new behavior.
- Run targeted validation, record exact commands and results, then commit and push the branch.

### Acceptance

- The queue control center distinguishes tasks eligible for batch retry from tasks blocked by approval, ownership, or takeover constraints.
- The queue control center exposes failure attribution in a form operators can use to understand both grouped tasks and cause counts.
- The queue control center exposes manual takeover entry points with clear reasons for each queued task that requires human ownership.
- Existing queue control center and contract behavior stays intact outside the new retry, attribution, and takeover surfaces.

### Validation

- `python3 -m pytest tests/test_control_center.py tests/test_execution_contract.py`
- `python3 -m pytest tests/test_console_ia.py`

### Validation Results

- `python3 -m pytest tests/test_control_center.py tests/test_execution_contract.py` -> passed: `13 passed in 0.10s`
- `python3 -m pytest tests/test_console_ia.py` -> passed: `11 passed in 0.06s`

### Status

- Branch: `bigclaw-182-control-center`
- Scope: queue control center batch retry, failure attribution, manual takeover
