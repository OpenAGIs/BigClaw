# BIG-GO-943 Runtime/Service/Orchestration Lane

## Scope

This lane covers the remaining legacy Python core-runtime surfaces under
`src/bigclaw` that still describe runtime, service, scheduling, workflow,
orchestration, and queue behavior after Go mainline cutover.

## Lane File List

| Legacy Python file | Current status | Go replacement | Lane action |
| --- | --- | --- | --- |
| `src/bigclaw/runtime.py` | Frozen compatibility surface | `bigclaw-go/internal/worker/runtime.go` | Keep as legacy shim until Python package exports can be dropped |
| `src/bigclaw/service.py` | Frozen compatibility surface | `bigclaw-go/cmd/bigclawd/main.go`, `bigclaw-go/internal/api/server.go` | Keep as legacy shim for `python -m bigclaw serve`; delete when Python CLI compatibility is retired |
| `src/bigclaw/scheduler.py` | Frozen compatibility surface | `bigclaw-go/internal/scheduler/scheduler.go` | Keep as legacy shim until remaining Python tests/importers are removed |
| `src/bigclaw/workflow.py` | Frozen compatibility surface | `bigclaw-go/internal/workflow/engine.go`, `bigclaw-go/internal/workflow/closeout.go` | Keep as legacy shim until workflow acceptance/reporting callers move fully to Go |
| `src/bigclaw/orchestration.py` | Frozen compatibility surface | `bigclaw-go/internal/workflow/orchestration.go` | Keep as legacy shim until Python reporting helpers stop importing orchestration types |
| `src/bigclaw/queue.py` | Frozen compatibility surface | `bigclaw-go/internal/queue/queue.go`, `bigclaw-go/internal/queue/file_queue.go`, `bigclaw-go/internal/queue/sqlite_queue.go`, `bigclaw-go/internal/queue/memory_queue.go` | Keep as legacy shim until Python operations/tests are deleted or redirected |
| `src/bigclaw/__main__.py` | Python entrypoint shim | `bigclaw-go/cmd/bigclawd/main.go`, `bigclaw-go/cmd/bigclawctl/main.go` | Keep temporarily; delete with Python package entrypoint cutover |
| `src/bigclaw/legacy_shim.py` | Legacy guidance shim | `bigclaw-go/internal/legacyshim/compilecheck.go` | Keep temporarily as cutover guidance surface |

## Repo Changes In This Lane

- `internal/legacyshim/compilecheck.go` now syntax-checks the full frozen
  runtime/service/scheduler/workflow/orchestration/queue shim set instead of
  only the CLI shell.
- `internal/regression/big_go_943_docs_test.go` pins this lane report and the
  issue-coverage cross-link so the mapping stays reviewable in CI.
- `docs/reports/issue-coverage.md` now references `BIG-GO-943` as the parallel
  migration lane that owns the frozen Python core-runtime surfaces.

## Delete Plan

The Python files above are not feature-development targets anymore. They should
be deleted in the following order once downstream imports are removed:

1. Delete `tests/test_runtime.py`, `tests/test_scheduler.py`,
   `tests/test_workflow.py`, `tests/test_orchestration.py`, `tests/test_queue.py`,
   and other Python-only compatibility tests that assert legacy behavior instead
   of the Go control plane.
2. Remove `src/bigclaw/__init__.py` re-exports for runtime/scheduler/workflow/
   orchestration/queue once no Python callers import them.
3. Delete the frozen Python modules in this lane together with
   `src/bigclaw/__main__.py` and `src/bigclaw/legacy_shim.py`.

## Validation

- `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression`
- `cd bigclaw-go && go test ./cmd/bigclawctl`
- `cd bigclaw-go && go test ./internal/scheduler ./internal/workflow ./internal/queue ./internal/worker`

## Residual Risks

- The repo still contains Python tests and package exports that import these
  frozen shims directly, so immediate deletion would be a breaking change.
- Go replacements are semantically broader than some Python fixtures, but the
  compatibility layer still exists for documentation, not as the product
  mainline.
- `src/bigclaw/service.py` remains the Python CLI bridge for users who have not
  yet switched from `python -m bigclaw` to `bigclawd` / `bigclawctl`.
