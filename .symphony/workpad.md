## BIG-GO-1465 Workpad

### Plan

- [x] Audit the remaining `src/bigclaw` package-root re-export/import glue and isolate the refill mutation path whose ordering must be frozen.
- [x] Replace the root package re-export surface with a minimal package marker, convert the remaining imports to direct module imports, and document the delete condition for the marker file.
- [x] Add a targeted refill regression that locks recent-batch delete/reassignment behavior to canonical `IssueOrder`, then run focused validation.

### Acceptance Criteria

- [x] `src/bigclaw/__init__.py` no longer serves as package-root re-export glue, and no in-repo tests depend on root re-exported symbols such as `EpicMilestone`.
- [x] The refill lane has an explicit regression test proving delete/reassignment keeps recent-batch ordering aligned to canonical `IssueOrder`.
- [x] Repo documentation records the exact marker-file delete condition, the remaining frozen Python shims, and validation evidence for this slice.

### Validation

- [x] `PYTHONPATH=src python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/__main__.py src/bigclaw/deprecation.py src/bigclaw/service.py src/bigclaw/roadmap.py tests/test_deprecation.py tests/test_roadmap.py`
- [x] `PYTHONPATH=src python3 -m pytest -q tests/test_deprecation.py tests/test_roadmap.py`
- [x] `cd bigclaw-go && go test ./internal/refill ./internal/legacyshim ./cmd/bigclawctl`
- [x] `python3 - <<'PY'`
      `import pathlib`
      `print(sum(1 for _ in pathlib.Path('src/bigclaw').glob('*.py')))`
      `PY`

### Notes

- Scope stayed limited to root-package Python glue removal plus refill ordering freeze coverage.
- Direct deletion of `src/bigclaw/__init__.py` is deferred because the local environment contains another installed `bigclaw` package, and keeping a minimal marker file ensures the repo checkout wins import resolution during targeted tests.
