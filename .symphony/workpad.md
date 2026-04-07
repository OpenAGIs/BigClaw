# BIG-GO-1579 Workpad

## Scope
- Candidate Python files in this sweep:
  - `src/bigclaw/design_system.py`
  - `src/bigclaw/models.py`
  - `src/bigclaw/repo_gateway.py`
  - `src/bigclaw/runtime.py`
  - `tests/conftest.py`
  - `tests/test_evaluation.py`
  - `tests/test_mapping.py`
  - `tests/test_queue.py`
  - `tests/test_risk.py`
  - `tests/test_validation_policy.py`
  - `bigclaw-go/scripts/benchmark/capacity_certification.py`
  - `bigclaw-go/scripts/e2e/mixed_workload_matrix.py`
  - `bigclaw-go/scripts/migration/shadow_compare.py`

## Plan
1. Confirm which candidate Python tests/scripts already have Go-native coverage or can be fronted by Go commands.
2. Delete redundant Python tests first where equivalent Go coverage already exists.
3. Freeze unavoidable Python runtime/model/design/repo surfaces as explicit compatibility shims with Go replacement targets and removal conditions.
4. Extend Go-side legacy compile-check coverage to the frozen shim files introduced/retained by this sweep.
5. Update docs/notes only where needed to record the sweep inventory, validation, and residual deletion conditions.
6. Run targeted Python and Go validation for touched areas, then commit and push branch `BIG-GO-1579`.

## Acceptance
- Enumerate the Python files covered by this sweep.
- Prefer deletion for redundant Python assets.
- Any Python file kept in scope must be clearly marked as a frozen compatibility layer with a concrete Go replacement path and deletion condition.
- Record exact validation commands and results.
- Keep changes scoped to `BIG-GO-1579`.

## Validation
- `python3 -m py_compile src/bigclaw/design_system.py src/bigclaw/models.py src/bigclaw/repo_gateway.py src/bigclaw/runtime.py tests/conftest.py tests/test_evaluation.py bigclaw-go/scripts/benchmark/capacity_certification.py bigclaw-go/scripts/e2e/mixed_workload_matrix.py bigclaw-go/scripts/migration/shadow_compare.py`
- `cd bigclaw-go && go test ./internal/legacyshim ./internal/repo ./internal/intake ./internal/queue ./internal/risk ./cmd/bigclawctl`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json`
- Additional targeted commands for any new Go tests or doc/regression checks introduced during the sweep.

## Notes
- Initial workspace bootstrap used the local bare mirror at `/Users/openagi/code/bigclaw-workspaces/.symphony/bigclaw-mirror.git` because direct remote materialization was flaky in this environment.
