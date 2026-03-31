## BIG-GO-1023

### Plan
- Reduce `src/bigclaw` residual Python file count in a scoped tranche by removing low-coupling modules first.
- Preserve the legacy Python import contract for `bigclaw.audit_events`, `bigclaw.dsl`, and `bigclaw.deprecation` through package-level compatibility exports.
- Keep behavior aligned with existing `bigclaw-go` implementations where those already exist.

### Acceptance
- Changes stay scoped to remaining `src/bigclaw` Python assets for this tranche.
- `.py` file count under `src/bigclaw` decreases.
- Legacy Python imports and existing tests for audit specs, workflow definition parsing, and deprecation warnings still pass.
- Report the impact on Python/Go file counts and note any `pyproject`/`setup` impact.

### Validation
- `pytest tests/test_audit_events.py tests/test_dsl.py`
- `python -m pytest tests/test_legacy_shim.py`
- `cd bigclaw-go && go test ./internal/observability ./internal/workflow ./internal/regression`
- `find src/bigclaw -maxdepth 1 -name '*.py' | wc -l`
- `find bigclaw-go -type f -name '*.go' | wc -l`
