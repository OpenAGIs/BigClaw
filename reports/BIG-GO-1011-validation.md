# BIG-GO-1011 Validation

## Scope

Refill sweep A for remaining repository-root/config/python packaging residuals.
This pass removes stale root README references to deleted Python shim files and
replaces missing legacy file-path references with the compatibility import
surfaces that still exist in-tree.

Continuation pass: align active cutover and reviewer-facing docs so they no
longer describe retired repo-root Python shim files as current compatibility
surfaces.

Final continuation pass: label the remaining BIG-GO-902 reviewer artifacts as
historical cutover evidence so the repo no longer presents those retired Python
shim paths as live operator entrypoints.

## Branch

- branch: `big-go-1011-root-config-residuals`

## Repo impact

- `py files`: `101`
- `go files`: `267`
- `pyproject.toml`: `0`
- `setup.py`: `0`

Counts were unchanged in this pass; the repo-root cleanup was documentation and
validation-surface tightening rather than source-file deletion.

## Cleanup history

- `cd902ee` `BIG-GO-1011 remove root python config residuals`
- `63f6db5` `BIG-GO-1011 retire root python wrapper scripts`
- `fce74ce` `BIG-GO-1011 align cutover docs with wrapper retirement`
- `93f8495` `BIG-GO-1011 drop python hook env residue`
- `0ef27ab` `BIG-GO-1011 remove egg-info ignore residue`
- `57aefe0` `BIG-GO-1011 ignore root pytest cache residue`
- `49603e7` `BIG-GO-1011 record root residue scope boundary`
- `c8102e6` `BIG-GO-1011 add validation evidence report`
- `8de0ce9` `BIG-GO-1011 refresh validation report head`
- `48dc372` `BIG-GO-1011 finalize validation report metadata`

Report-maintenance commits after the evidence file was introduced are intentionally not tracked
here one-by-one when they only stabilize this report's metadata.

## Validation commands

```bash
find . -maxdepth 2 \( -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' -o -name 'MANIFEST.in' -o -name '*.egg-info' -o -name '*.dist-info' -o -name '.pre-commit-config.yaml' -o -name 'requirements*.txt' -o -name 'tox.ini' -o -name 'pytest.ini' -o -name '.python-version' -o -name '.coveragerc' -o -name 'Pipfile' -o -name 'poetry.lock' -o -name 'uv.lock' \) | sort
```

Result: no matches

```bash
rg -n "run_task_smoke\.py|shadow_compare\.py|src/bigclaw/scheduler\.py|src/bigclaw/workflow\.py|src/bigclaw/orchestration\.py|src/bigclaw/queue\.py|src/bigclaw/service\.py" README.md
```

Result: exit `1` with no matches

```bash
python3 - <<'PY'
from pathlib import Path
repo = Path('.').resolve()
paths = [
    repo / 'src/bigclaw/runtime.py',
    repo / 'src/bigclaw/__init__.py',
    repo / 'bigclaw-go/docs/go-cli-script-migration.md',
    repo / 'bigclaw-go/cmd/bigclawd/main.go',
]
missing = [str(p.relative_to(repo)) for p in paths if not p.exists()]
if missing:
    raise SystemExit('missing: ' + ', '.join(missing))
print('ok:', ', '.join(str(p.relative_to(repo)) for p in paths))
PY
```

Result: `ok: src/bigclaw/runtime.py, src/bigclaw/__init__.py, bigclaw-go/docs/go-cli-script-migration.md, bigclaw-go/cmd/bigclawd/main.go`

```bash
rg -n "remain available as compatibility shims|Compatibility shims now dispatch|compatibility shims during cutover" docs/go-cli-script-migration-plan.md docs/go-mainline-cutover-handoff.md reports/BIG-GO-902-closeout.md reports/BIG-GO-902-pr.md
```

Result: no matches

```bash
rg -n "Historical note|historical_note" reports/BIG-GO-902-validation.md reports/BIG-GO-902-closeout.md reports/BIG-GO-902-pr.md reports/BIG-GO-902-status.json
```

Result:

```text
reports/BIG-GO-902-validation.md:16:Historical note: the Python shim file paths referenced below are part of the
reports/BIG-GO-902-closeout.md:63:Historical note: command lines and shim file names listed in this closeout
reports/BIG-GO-902-pr.md:20:Historical note: the Python shim file paths listed in this draft reflect the
reports/BIG-GO-902-status.json:52:    "historical_note": "The Python shim file paths in these recorded BIG-GO-902 validation commands reflect the original cutover branch state. Later cleanup passes retired the repo-root Python shim files from the active operator path.",
```

```bash
make build
```

Result: passed

```bash
make test
```

Result: failed in existing Go regression:

```text
bigclaw-go/internal/regression
TestContinuationPolicyGateReviewerMetadata
runtime_report_followup_docs_test.go:144: unexpected policy gate reviewer digest issue: {ID: Slug:}
```

```bash
bash scripts/ops/bigclawctl github-sync --help
bash scripts/ops/bigclawctl refill --help
bash scripts/ops/bigclawctl workspace validate --help
```

Result: all exited `0`

```bash
git diff --stat origin/main...HEAD
```

Result:

```text
15 files changed, 66 insertions(+), 190 deletions(-)
```

## Scope boundary

Remaining root-level Python mentions are intentional migration-only validation surfaces:

- `scripts/dev_bootstrap.sh`
- source-level `PYTHONPATH=src python3 -m pytest ...` guidance
- root cache ignores for `__pycache__/`, `*.py[cod]`, and `.pytest_cache/`
- compatibility import surfaces exposed through `src/bigclaw/__init__.py`
  and `src/bigclaw/runtime.py`
- historical migration reports may still mention retired shim files as past
  implementation details, but active root/cutover guidance no longer presents
  them as current repo surfaces

No additional root `pyproject.toml`, `setup.py`, `*.egg-info`, repo-root Python wrapper scripts, or Python-specific CI/hook config residue remains.
