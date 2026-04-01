## Plan

1. Purge `src/bigclaw/deprecation.py` by moving the retained Python legacy-runtime guidance helpers into `src/bigclaw/runtime.py`.
2. Keep the `bigclaw.deprecation` import path working by exporting that moved surface from `src/bigclaw/__init__.py` and installing a synthetic compatibility module there.
3. Repoint retained Python callers in `__main__.py` and `runtime.py` to the runtime-owned deprecation surface.
4. Add tranche 17 Go regression coverage proving the deleted Python file is gone and the Go deprecation replacement files exist.
5. Run focused Python and Go validation plus the repository Python file count check.
6. Commit with the deleted Python file and added Go test file explicitly listed, then push to `origin/BIG-GO-1041`.

## Acceptance

- `src/bigclaw/deprecation.py` is deleted.
- `src/bigclaw/runtime.py` provides the retained Python deprecation surface previously owned by `deprecation.py`.
- `src/bigclaw/__init__.py` no longer imports from `src/bigclaw/deprecation.py`, and `import bigclaw.deprecation` still resolves through package compatibility wiring.
- Retained Python callers use the runtime-owned deprecation surface.
- `bigclaw-go/internal/regression/top_level_module_purge_tranche17_test.go` asserts the Python deletion and Go replacement files.
- `find . -name '*.py' | wc -l` decreases from the current baseline of `42`.
- Focused Python and Go tests pass.
- Changes are committed and pushed to `origin/BIG-GO-1041`.

## Validation

- `find . -name '*.py' | wc -l`
- `PYTHONPATH=src python3 -m pytest tests/test_observability.py tests/test_repo_rollout.py -q`
- `cd bigclaw-go && go test ./internal/regression -run 'TestLegacyMainlineCompatibilityManifestStaysAligned|TestTopLevelModulePurgeTranche(1|2|3|4|5|6|7|8|9|10|11|12|13|14|15|16|17)'`
- `git status --short`
- `git log -1 --stat`
