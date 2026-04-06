# BIG-GO-1525 Validation

## Python residual deletion sweep

- Before count: `138` Python files
- Removed files:
  - `bigclaw-go/scripts/benchmark/capacity_certification_test.py`
  - `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
- After count: `135` Python files

## Validation commands

- `python3 -m pytest tests/test_parallel_validation_bundle.py tests/test_validation_bundle_continuation_policy_gate.py`
- `python3 bigclaw-go/scripts/e2e/export_validation_bundle.py --help`
- `python3 bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py --help`
- `find . -type f -name '*.py' | wc -l`

## Results

- `python3 -m pytest tests/test_parallel_validation_bundle.py tests/test_validation_bundle_continuation_policy_gate.py`
  - Result: failed
  - Detail: `1 failed, 4 passed in 0.08s`
  - Failure: `tests/test_parallel_validation_bundle.py::test_export_validation_bundle_generates_latest_reports_and_index`
  - Failure cause: `bigclaw-go/scripts/e2e/export_validation_bundle.py` uses `Path | None` syntax and raises `TypeError` under local `Python 3.9.6`
- `python3 bigclaw-go/scripts/e2e/export_validation_bundle.py --help`
  - Result: failed
  - Detail: `TypeError: unsupported operand type(s) for |: 'type' and 'NoneType'`
- `python3 bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py --help`
  - Result: passed
- `find . -type f -name '*.py' | wc -l`
  - Result: passed
  - Detail: `135`
