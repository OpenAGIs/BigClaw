# Root And Ops Script Final Sweep

Issue: `BIG-GO-982`

## Scope

This final sweep is limited to Python files under:

- `scripts/*.py`
- `scripts/ops/*.py`

## Batch Inventory

### `scripts/*.py`

No files remain. The repository currently has no Python files directly under `scripts/`.

### `scripts/ops/*.py`

No files remain. The repository currently has no `scripts/ops/` directory, so there are no Python files in this batch scope.

## Keep / Replace / Delete Basis

- Keep: shell entrypoints such as `scripts/e2e/run_all.sh`, `scripts/e2e/kubernetes_smoke.sh`, `scripts/e2e/ray_smoke.sh`, and `scripts/benchmark/run_suite.sh` remain because they are shell orchestration wrappers, not Python implementation files.
- Replace: root and ops script behavior has already been migrated away from Python in this slice. The active CLI replacement surface lives in `cmd/bigclawctl`, including `automation e2e run-task-smoke`, `automation benchmark soak-local`, and `automation migration shadow-compare`.
- Delete: there are no remaining batch-scope Python files to delete in this issue. The final sweep result for the target directories is already zero.
- Out of scope: the remaining repository Python files live under `scripts/benchmark/`, `scripts/e2e/`, and `scripts/migration/`; they are tracked by other migration batches and are not root-level or ops-level script files.

## Python Count Impact

- Repository-wide Python file count before this sweep: `23`
- Repository-wide Python file count after this sweep: `23`
- Net change from this issue: `0`

The count is unchanged because the target directories were already clean when this batch started.

## Validation Commands

```bash
cd bigclaw-go
find scripts -maxdepth 1 -name '*.py' | sort
find scripts/ops -maxdepth 1 -name '*.py' 2>/dev/null | sort
find . -name '*.py' | sed 's#^./##' | sort | wc -l
go test ./internal/regression -run 'TestRootAndOpsScriptPythonSweepStaysEmpty$'
```
