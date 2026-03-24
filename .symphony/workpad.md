# BIGCLAW-176

## Plan
- Inspect the existing Python orchestration and reporting surfaces that already summarize run state, handoffs, and queue actions.
- Add a lifecycle fanout batch model for `start` / `stop` / `restart` / `upgrade` on the legacy orchestration surface so one plan can render multiple lifecycle operations together.
- Reuse the existing reporting layer for batch lifecycle summaries instead of creating a new subsystem.
- Extend takeover queue requests with blocked runtime task linkage and blocked task counts so operators can see which waiting work is attached to a manual takeover.
- Cover batch lifecycle rendering and takeover linkage summaries with focused regression tests in `tests/test_orchestration.py` and `tests/test_reports.py`.
- Run only targeted tests for the touched surfaces, inspect the scoped diff, commit, and push `symphony/BIGCLAW-176`.

## Acceptance
- Lifecycle fanout plans for `start`, `stop`, `restart`, and `upgrade` can be rendered as one batch report with per-operation and aggregate summaries.
- Takeover queue entries can link to blocked runtime tasks so an operator can see which tasks are waiting on manual takeover and how many are blocked.
- Regression coverage asserts the bulk lifecycle summary text and takeover queue linkage data without widening scope outside the current orchestration/reporting surfaces.

## Validation
- Run `pytest tests/test_orchestration.py tests/test_reports.py`.
- If needed while iterating, run file-scoped subsets such as `pytest tests/test_orchestration.py -k lifecycle` and `pytest tests/test_reports.py -k takeover`.
- Record the exact test commands and their results in the final report.
- Run `git status --short`, `git log -1 --stat`, and `git push origin symphony/BIGCLAW-176` before closeout.
