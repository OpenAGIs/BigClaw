## BIG-GO-1593 Workpad

### Plan

- [x] Inspect the assigned Python modules and nearby tests to find the smallest coherent slice that can be removed or migrated without broad repo churn.
- [x] Implement scoped repo-visible changes that reduce the Python file count and keep behavior covered by targeted validation.
- [x] Run focused validation, record exact commands and results here, then commit and push `BIG-GO-1593`.

### Acceptance

- Remove or migrate the assigned Python assets toward Go-owned surfaces.
- Reduce the repository Python file count with scoped repo-visible changes.
- Record exact validation commands and residual risks.

### Validation

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593/bigclaw-go && go test ./internal/refill ./internal/repo`
  - Result: passed
  - Output: `ok   bigclaw-go/internal/refill 3.271s`; `ok   bigclaw-go/internal/repo 3.269s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && python3 -m pytest tests/test_control_center.py tests/test_followup_digests.py tests/test_operations.py -q`
  - Result: passed
  - Output: `25 passed in 0.20s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1593 && rg --files | rg '\.py$' | wc -l`
  - Result: passed
  - Output: `134`

### Notes

- Initial focus from issue: `src/bigclaw/audit_events.py`, `src/bigclaw/execution_contract.py`, `src/bigclaw/parallel_refill.py`, `src/bigclaw/repo_registry.py`, `src/bigclaw/ui_review.py`, `tests/test_control_center.py`, `tests/test_followup_digests.py`, `tests/test_operations.py`, plus nearby Python assets if they form a cleaner removable slice.
- Constraint: keep changes scoped to this refill issue and avoid unrelated tracker/documentation churn.
- Removed the isolated Python refill and repo-registry surfaces plus their dedicated tests: `src/bigclaw/parallel_refill.py`, `src/bigclaw/repo_registry.py`, `tests/test_parallel_refill.py`, `tests/test_repo_registry.py`.
- Python file count changed from `138` to `134`.
- Residual risk: some docs and historical validation reports still mention the deleted Python files and tests; this change intentionally leaves those references untouched because the issue prioritized physical Python asset removal over documentation cleanup.
