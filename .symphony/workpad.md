Issue: BIG-GO-1030

Plan
- Fold `src/bigclaw/execution_contract.py` into `src/bigclaw/operations.py`.
- Preserve compatibility for `bigclaw.execution_contract` imports through the package legacy-surface shim in `src/bigclaw/__init__.py`.
- Re-run targeted operations pytest coverage plus a direct import-compatibility check, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `src/bigclaw/execution_contract.py` is removed from the tree.
- Execution-contract behaviors still pass from `src/bigclaw/operations.py` and `tests/test_operations.py`.
- `from bigclaw.execution_contract import ...` continues to resolve through the compatibility surface.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `PYTHONPATH=src python3 -m pytest tests/test_operations.py -q`
- `PYTHONPATH=src python3 - <<'PY'\nfrom bigclaw.execution_contract import ExecutionContractLibrary, render_execution_contract_report\nprint(ExecutionContractLibrary.__name__)\nprint(render_execution_contract_report.__name__)\nPY`
- `find . -type f \( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\.py$/{py++} /\.go$/{go++} /pyproject\.toml$/{pp++} /(setup\.py|setup\.cfg)$/{setup++} END{printf("py=%d\ngo=%d\npyproject=%d\nsetup=%d\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
