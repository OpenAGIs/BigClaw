# BIG-GO-1079 Workpad

## Plan
- Confirm the remaining live Python entry surface under `src/bigclaw` and keep the change scoped to an actual executable path rather than passive docs.
- Remove `src/bigclaw/__main__.py` so `python -m bigclaw` no longer exists as a default Python entrypoint.
- Update the Go-owned legacy compile-check and regression coverage so the frozen compatibility surface no longer expects the deleted Python entry file.
- Run targeted validation that proves the Python file count dropped, the deleted entrypoint is absent, and the remaining Go-owned validation path still passes.
- Commit the scoped change and push the branch.

## Acceptance
- `src/bigclaw/__main__.py` is deleted from the repo.
- The Go compile-check path no longer references `src/bigclaw/__main__.py`.
- Regression coverage pins the removed top-level module so the Python CLI entrypoint cannot return unnoticed.
- Validation records an actual decrease in repository `.py` count and confirms the remaining Go-owned checks still pass.

## Validation
- `find . -name '*.py' | wc -l`
- `test ! -f src/bigclaw/__main__.py`
- `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json`
