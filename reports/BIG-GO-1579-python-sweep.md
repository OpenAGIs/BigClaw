# BIG-GO-1579 Python Sweep

## Covered Files

Deleted in this sweep:
- `tests/test_mapping.py`
- `tests/test_queue.py`
- `tests/test_risk.py`
- `tests/test_validation_policy.py`

Retained as frozen compatibility surfaces:
- `src/bigclaw/design_system.py`
- `src/bigclaw/models.py`
- `src/bigclaw/repo_gateway.py`
- `src/bigclaw/runtime.py`
- `tests/conftest.py`
- `tests/test_evaluation.py`
- `bigclaw-go/scripts/benchmark/capacity_certification.py`
- `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
- `bigclaw-go/scripts/migration/shadow_compare.py`

## Why These Tests Were Deleted

- `tests/test_mapping.py`: covered by `bigclaw-go/internal/intake/mapping_test.go`.
- `tests/test_queue.py`: covered by `bigclaw-go/internal/queue/*_test.go`.
- `tests/test_risk.py`: covered by `bigclaw-go/internal/risk/risk_test.go` and scheduler tests.
- `tests/test_validation_policy.py`: legacy-only closeout policy check with no active Go mainline caller; removed as residual Python-only coverage.

## Residual Conditions

- `src/bigclaw/design_system.py`: remove after remaining Python UI/design-system imports/tests migrate to `bigclaw-go/internal/product`.
- `src/bigclaw/models.py`: remove after Python task/risk/workflow callers stop importing it and the Go domain package is the only maintained path.
- `src/bigclaw/repo_gateway.py`: remove after Python repo surfaces are retired and `bigclaw-go/internal/repo/gateway.go` is the sole implementation.
- `src/bigclaw/runtime.py`: remove after Python scheduler/runtime tests retire and `bigclaw-go/internal/worker/runtime.go` is the only maintained runtime.
- `tests/conftest.py`: remove when the remaining Python tests are gone.
- `tests/test_evaluation.py`: remove when `src/bigclaw/evaluation.py` is retired or replaced by Go-native coverage.
- `bigclaw-go/scripts/benchmark/capacity_certification.py`: remove after a Go CLI owns capacity-certification generation.
- `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`: remove after a Go CLI/e2e command owns mixed-workload routing validation.
- `bigclaw-go/scripts/migration/shadow_compare.py`: remove after a Go CLI owns live-shadow comparison.

## Validation

- `python3 -m py_compile src/bigclaw/design_system.py src/bigclaw/models.py src/bigclaw/repo_gateway.py src/bigclaw/runtime.py tests/conftest.py tests/test_evaluation.py bigclaw-go/scripts/benchmark/capacity_certification.py bigclaw-go/scripts/e2e/mixed_workload_matrix.py bigclaw-go/scripts/migration/shadow_compare.py` -> passed
- `cd bigclaw-go && go test ./internal/legacyshim ./internal/repo ./internal/intake ./internal/queue ./internal/risk ./cmd/bigclawctl` -> passed
- `PYTHONPATH=src python3 -m pytest tests/test_design_system.py tests/test_models.py tests/test_repo_gateway.py tests/test_runtime.py tests/test_evaluation.py -q` -> passed (`34 passed`)
- `bash scripts/ops/bigclawctl legacy-python compile-check --json` -> passed (`status: ok`)

## Residual Risk

- The retained Python files are still physical repo assets; this sweep bounds them as frozen compatibility surfaces, but they remain until the named Go entrypoints exist and docs/importers fully cut over.
- `tests/test_evaluation.py` remains because there is no Go-native evaluation replacement in-tree yet; deleting it now would remove the only direct regression coverage for the frozen Python evaluation helpers.
