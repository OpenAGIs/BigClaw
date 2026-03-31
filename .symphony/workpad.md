Issue: BIG-GO-1035

Plan
- Remove a safe tranche of `src/bigclaw/**` Python modules that already have Go-native owners and are not on surviving Python runtime paths.
- Continue with a second safe tranche for `connectors.py` and `dsl.py`, which now only serve Python package exports/tests while their Go owners are already canonical.
- Delete the directly coupled Python tests for those removed modules, while preserving adjacent coverage by rewriting any remaining test that only used the deleted helpers incidentally.
- Trim `src/bigclaw/__init__.py` exports so the package no longer imports deleted modules.
- Add a Go regression test that asserts the deleted Python files stay gone and their canonical Go replacements remain present.
- Refresh migration docs that still list the deleted modules as remaining Python inventory.
- Remove `src/bigclaw/repo_plane.py` and `src/bigclaw/repo_links.py` by keeping a minimal compatibility surface in `src/bigclaw/__init__.py` while pointing canonical ownership at `bigclaw-go/internal/repo/{plane,links}.go`.
- Run targeted tests and inventory checks, then commit and push the scoped branch.

Acceptance
- `.py` file count under `src/bigclaw` decreases for this issue.
- Deleted Python modules are ones with existing Go-native replacements under `bigclaw-go/internal/**`.
- A new Go regression test is added to lock in the deletion set.
- `pyproject.toml` and `setup.py` remain absent from the repository root.
- Final report can name which Python files were deleted and which Go files/tests now cover them.
- `src/bigclaw/observability.py` and repo link tests continue to import successfully after the physical repo Python modules are deleted.

Validation
- `find src/bigclaw -maxdepth 1 -name '*.py' | sort | wc -l`
- `find src/bigclaw -maxdepth 1 -name '*.py' | sort`
- `gofmt -w bigclaw-go/internal/regression/python_src_bigclaw_replacement_inventory_test.go`
- `cd bigclaw-go && go test ./internal/regression -run TestSrcBigClawGoReplacementInventory`
- `PYTHONPATH=src python3 -c "import bigclaw; import bigclaw.collaboration; print('ok')"`
- `python3 -m pytest tests/test_repo_collaboration.py -q`
- `PYTHONPATH=src python3 -m py_compile src/bigclaw/__init__.py src/bigclaw/observability.py`
- `PYTHONPATH=src python3 -c "import bigclaw.repo_plane, bigclaw.repo_links, bigclaw.observability; print('ok')"`
- `PYTHONPATH=src python3 -m pytest tests/test_repo_links.py tests/test_observability.py -q`
- `find . -maxdepth 2 \\( -name 'pyproject.toml' -o -name 'setup.py' \\) | sort`
- `git status --short`
