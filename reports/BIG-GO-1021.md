# BIG-GO-1021 Autorefill sweep A: remaining root/config/python residuals

## Scope

- `README.md`
- `.github/workflows/ci.yml`
- `scripts/dev_bootstrap.sh`
- `tests/conftest.py` (deleted)

## Changes

- Rewrote the root migration guidance to state explicitly that repository-root
  Python packaging artifacts remain retired and that legacy validation must use
  `PYTHONPATH=src`.
- Removed stale README references to deleted `bigclaw-go/scripts/.../*.py`
  automation entrypoints and replaced them with the current `bigclawctl
  automation ...` commands.
- Updated the live migration plan doc to mark the deleted `bigclaw-go/scripts`
  Python files as retired paths rather than current batch entrypoints.
- Deleted the orphaned [src/bigclaw/pilot.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/src/bigclaw/pilot.py)
  residual after confirming it had no remaining in-repo consumers and that the
  active pilot implementation/report surface already lives in Go under
  `bigclaw-go/internal/pilot`.
- Migrated the single-consumer task memory store from Python into
  `bigclaw-go/internal/memory`, then deleted
  [memory.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/src/bigclaw/memory.py)
  and [test_memory.py](/Users/openagi/code/bigclaw-workspaces/BIG-GO-1021/tests/test_memory.py).
- Split CI into a Go-mainline job and a legacy-Python migration job, and moved
  the legacy test invocation to `python3 -m pytest`, so the root workflow no
  longer reads as a Python-first repository entrypoint.
- Tightened the bootstrap helper messaging to call out explicit `PYTHONPATH`
  usage for any migration-only Python validation.
- Deleted `tests/conftest.py`, removing implicit `src` path injection and
  reducing repository `.py` file count by one.

## File-count impact

- `.py`: `50 -> 46`
- `.go`: `282 -> 282`
- `pyproject.toml`: absent before, absent after
- `setup.py`: absent before, absent after
- `setup.cfg`: absent before, absent after
- `*.egg-info`: absent before, absent after

## Validation

- `find . -path './.git' -prune -o -name '*.py' -print | wc -l`
- `find . -path './.git' -prune -o -name '*.go' -print | wc -l`
- `find . -maxdepth 2 \( -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' -o -name '*.egg-info' \) -print`
- `PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py tests/test_planning.py -q`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./cmd/bigclawd`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation --help`
- `rg -n "bigclaw-go/scripts/.+\\.py" README.md docs/go-cli-script-migration-plan.md -S`
- `rg -n "PilotKPI|PilotImplementationResult|render_pilot_implementation_report|from bigclaw\\.pilot|bigclaw\\.pilot" src tests README.md docs reports scripts -S`
- `cd bigclaw-go && go test ./internal/pilot -run 'TestImplementationResultReadyWhenKPIsPassAndNoIncidents|TestRenderPilotImplementationReportContainsReadinessFields'`
- `rg -n "TaskMemoryStore|MemoryPattern|from bigclaw\\.memory|bigclaw\\.memory" src tests README.md docs reports scripts bigclaw-go -S`
- `cd bigclaw-go && go test ./internal/memory -run TestTaskStoreReusesHistoryAndInjectsRules`
- `python3 - <<'PY'\nfrom pathlib import Path\nci = Path('.github/workflows/ci.yml').read_text()\nassert 'PYTHONPATH=src python3 -m pytest' in ci\nassert 'PYTHONPATH=src pytest' not in ci\nPY`
- `rg -n "pyproject|setup.py|egg-info|pip install -e|python -m build|setuptools" -S README.md .github/workflows/ci.yml scripts/dev_bootstrap.sh reports/BIG-GO-1021.md`

## Residual risk

- The repository still contains legacy Python source and tests under `src/` and
  `tests/`; this lane only removes remaining root/config entrypoint residue and
  one implicit Python test bootstrap file.
