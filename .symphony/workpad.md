Issue: BIG-GO-1030

Plan
- Fold the residual Python `connectors` compatibility structs/stubs into `src/bigclaw/models.py` and retire `src/bigclaw/connectors.py`.
- Fold the residual Python `dsl` workflow-definition structs into `src/bigclaw/models.py` and retire `src/bigclaw/dsl.py`.
- Keep `bigclaw.connectors` and `bigclaw.dsl` import paths working by installing legacy surface modules from `src/bigclaw/__init__.py` instead of keeping standalone files.
- Merge the dedicated DSL regression coverage into an existing workflow/runtime-oriented pytest file, then delete `tests/test_dsl.py` to reduce physical `.py` count without dropping assertions.
- Update any directly coupled documentation references that still claim `src/bigclaw/connectors.py` or `src/bigclaw/dsl.py` are standalone residual assets.
- Measure `.py` / `.go` / `pyproject` / `setup` counts before and after, run targeted pytest coverage for the migrated surfaces, then commit and push.

Acceptance
- The repository physical `.py` file count decreases.
- `src/bigclaw/connectors.py`, `src/bigclaw/dsl.py`, and `tests/test_dsl.py` are removed from the tree.
- `bigclaw.connectors` and `bigclaw.dsl` imports still resolve through package-level compatibility shims.
- Workflow-definition and connector data structures continue to work from their migrated owner module.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `find . -type f \\( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \\) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\\.py$/{py++} /\\.go$/{go++} /pyproject\\.toml$/{pp++} /(setup\\.py|setup\\.cfg)$/{setup++} END{printf("py=%d\\ngo=%d\\npyproject=%d\\nsetup=%d\\n",py,go,pp,setup)}'`
- `PYTHONPATH=src python3 -m pytest tests/test_runtime_matrix.py tests/test_workspace_bootstrap.py -q`
- `git status --short`
- `git diff --stat`
