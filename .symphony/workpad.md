## BIG-GO-12 Workpad

### Plan

- Identify a bounded batch of `tests/*.py` files whose behavior is already covered by `bigclaw-go` tests.
- Remove only those Python residual tests and leave planner-driven Python test references alone for this issue.
- Run targeted Go validation for the replacement coverage packages.
- Commit the scoped diff on `BIG-GO-12` and push to `origin`.

### Acceptance

- The selected Python residual tests are removed from `tests/`.
- Equivalent Go coverage remains present for each removed area.
- Targeted `go test` commands for the touched Go packages pass.
- The final diff stays scoped to this issue slice.

### Validation

- `cd bigclaw-go && go test ./internal/intake ./internal/product ./internal/workflow ./internal/queue ./internal/scheduler ./internal/githubsync ./internal/risk ./internal/governance ./internal/events`
- `git status --short`

### Results

- `cd bigclaw-go && go test ./internal/intake ./internal/product ./internal/workflow ./internal/queue ./internal/scheduler ./internal/githubsync ./internal/risk ./internal/governance ./internal/events`
  - `ok  	bigclaw-go/internal/intake	3.328s`
  - `ok  	bigclaw-go/internal/product	3.244s`
  - `ok  	bigclaw-go/internal/workflow	3.158s`
  - `ok  	bigclaw-go/internal/queue	28.781s`
  - `ok  	bigclaw-go/internal/scheduler	1.725s`
  - `ok  	bigclaw-go/internal/githubsync	4.966s`
  - `ok  	bigclaw-go/internal/risk	2.141s`
  - `ok  	bigclaw-go/internal/governance	1.891s`
  - `ok  	bigclaw-go/internal/events	2.273s`
- `git status --short`
  - `M .symphony/workpad.md`
  - `D tests/test_connectors.py`
  - `D tests/test_dashboard_run_contract.py`
  - `D tests/test_event_bus.py`
  - `D tests/test_github_sync.py`
  - `D tests/test_governance.py`
  - `D tests/test_mapping.py`
  - `D tests/test_models.py`
  - `D tests/test_queue.py`
  - `D tests/test_risk.py`
  - `D tests/test_scheduler.py`

### Notes

- Scoped batch for this issue: `test_mapping.py`, `test_connectors.py`, `test_dashboard_run_contract.py`, `test_models.py`, `test_queue.py`, `test_scheduler.py`, `test_github_sync.py`, `test_risk.py`, `test_governance.py`, and `test_event_bus.py`.
- Intentionally excluding Python tests still referenced by active planner/docs flows in this slice, such as `test_saved_views.py`, `test_orchestration.py`, and the repo-surface clusters.
