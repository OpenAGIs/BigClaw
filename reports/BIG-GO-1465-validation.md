# Issue Validation Report

- Issue ID: BIG-GO-1465
- Title: Lane refill: eliminate package re-export/import glue in src/bigclaw and freeze delete order
- Test Environment: local-python3 + local-go
- Generated At: 2026-04-06T00:00:00Z

## Outcome

Replaced the `src/bigclaw` package-root re-export shim with a minimal package marker so the legacy Python tree no longer exposes a giant import-glue surface at package import time. The remaining in-repo consumer that depended on root re-exports now imports `bigclaw.roadmap` directly, and the refill queue now has a focused regression test that freezes recent-batch delete and reassignment ordering against canonical `IssueOrder`.

## Python File Reduction And Delete Conditions

- No additional Python file was deleted in this slice because `src/bigclaw/__init__.py` still has to exist as a package marker to beat conflicting installed `bigclaw` packages during local test/import resolution.
- `src/bigclaw/__init__.py` was reduced from the package-root re-export shim to a minimal marker.
  - Reason: the prior file was pure package re-export/import glue with no unique runtime behavior.
  - Delete condition: safe once local and CI environments no longer require a regular package marker to keep the repo checkout ahead of any conflicting installed `bigclaw` package.
  - Replacement: direct module imports such as `from bigclaw.roadmap import ...`.

## Freeze Conditions

- `src/bigclaw/__main__.py` remains frozen because `python -m bigclaw` is still the documented migration-only deprecation surface and is compile-checked by `bigclawctl legacy-python compile-check`.
- `src/bigclaw/service.py` remains frozen for the same migration-only service warning path.
- `src/bigclaw/legacy_shim.py` remains frozen because the Go `legacy-python compile-check` command explicitly validates that compatibility shim set.

## Validation Evidence

- `PYTHONPATH=src python3 -m py_compile src/bigclaw/__main__.py src/bigclaw/deprecation.py src/bigclaw/service.py src/bigclaw/roadmap.py tests/test_deprecation.py tests/test_roadmap.py`
- `cd bigclaw-go && go test ./internal/refill ./internal/legacyshim ./cmd/bigclawctl`
- `python3 - <<'PY'`
  `import pathlib`
  `print(sum(1 for _ in pathlib.Path('src/bigclaw').glob('*.py')))`
  `PY`
