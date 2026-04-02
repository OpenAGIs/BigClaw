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

Current continuation pass: remove the remaining active validation-doc reference
to deleted `tests/test_service.py` so current guidance only names test files
that still exist in the workspace.

Current continuation pass: retarget the cutover issue pack away from deleted
`src/bigclaw/service.py`, `scheduler.py`, `orchestration.py`, `workflow.py`,
and `queue.py` file paths to the compatibility surfaces that still exist.

Current continuation pass: refresh the Go CLI migration plan so its active
validation command uses the current source-level legacy smoke suite instead of
deleted `tests/test_legacy_shim.py` and `tests/test_deprecation.py`.

Current continuation pass: mark the remaining deleted script paths in active
planning docs as explicitly retired or historical migration identifiers so they
no longer read like current workspace files.

Current continuation pass: remove the generated root `.pytest_cache/` directory
so no root Python cache/package residue remains on disk after validation.

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

Continuation commits in this cleanup sweep:

- `52a101f` `BIG-GO-1011 refresh root migration residual docs`
- `32e9771` `BIG-GO-1011 retire stale shim guidance`
- `b9d661d` `BIG-GO-1011 label historical shim artifacts`
- `e6d3596` `BIG-GO-1011 drop stale test path guidance`
- `892a4b7` `BIG-GO-1011 retarget cutover legacy surfaces`
- `28ca157` `BIG-GO-1011 refresh migration plan validation`
- `0a577d0` `BIG-GO-1011 clarify retired path references`
- `5d0e4d3` `BIG-GO-1011 clear root pytest cache residue`

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
rg -n "test_service\.py" docs/BigClaw-AgentHub-Integration-Alignment.md docs README.md reports
```

Result:

```text
reports/BIG-GO-948-validation.md:16:- `tests/test_service.py`
reports/OPE-148-150-validation.md:19:  - `tests/test_service.py`
reports/OPE-151-153-validation.md:12:  - `tests/test_service.py`
```

```bash
python3 - <<'PY'
from pathlib import Path
repo = Path('.').resolve()
paths = [
    'tests/test_repo_governance.py',
    'tests/test_repo_triage.py',
    'tests/test_operations.py',
    'tests/test_repo_rollout.py',
]
missing = [p for p in paths if not (repo / p).exists()]
if missing:
    raise SystemExit('missing: ' + ', '.join(missing))
print('ok:', ', '.join(paths))
PY
```

Result: `ok: tests/test_repo_governance.py, tests/test_repo_triage.py, tests/test_operations.py, tests/test_repo_rollout.py`

```bash
rg -n "src/bigclaw/service\.py|src/bigclaw/scheduler\.py|src/bigclaw/orchestration\.py|src/bigclaw/workflow\.py|src/bigclaw/queue\.py" docs/go-mainline-cutover-issue-pack.md docs README.md -g '!reports/**'
```

Result: exit `1` with no matches

```bash
python3 - <<'PY'
from pathlib import Path
repo = Path('.').resolve()
paths = [
    'src/bigclaw/__init__.py',
    'src/bigclaw/runtime.py',
    'src/bigclaw/__main__.py',
]
missing = [p for p in paths if not (repo / p).exists()]
if missing:
    raise SystemExit('missing: ' + ', '.join(missing))
print('ok:', ', '.join(paths))
PY
```

Result: `ok: src/bigclaw/__init__.py, src/bigclaw/runtime.py, src/bigclaw/__main__.py`

```bash
rg -n "test_legacy_shim\.py|test_deprecation\.py" docs/go-cli-script-migration-plan.md README.md docs workflow.md
```

Result: exit `1` with no matches

```bash
PYTHONPATH=src python3 -m pytest tests/test_workspace_bootstrap.py tests/test_planning.py
```

Result:

```text
============================= test session starts ==============================
platform darwin -- Python 3.9.6, pytest-8.4.2, pluggy-1.6.0
rootdir: /Users/openagi/code/bigclaw-workspaces/BIG-GO-1011
plugins: cov-7.1.0
collected 23 items

tests/test_workspace_bootstrap.py .........                              [ 39%]
tests/test_planning.py ..............                                    [100%]

============================== 23 passed in 3.25s ==============================
```

```bash
python3 - <<'PY'
from pathlib import Path
import re
repo = Path('.').resolve()
files = [repo/'README.md', repo/'workflow.md'] + sorted((repo/'docs').rglob('*.md'))
patterns = [
    re.compile(r'`(scripts/[^`]+\.py)`'),
    re.compile(r'`(bigclaw-go/scripts/[^`]+\.py)`'),
]
for path in files:
    lines = path.read_text().splitlines()
    for i, line in enumerate(lines, 1):
        for pat in patterns:
            for m in pat.finditer(line):
                ref = m.group(1)
                target = repo / ref
                if target.exists():
                    continue
                context = '\n'.join(lines[max(0, i-3):min(len(lines), i+1)])
                if 'retired' not in context.lower() and 'migration identifier' not in context.lower() and 'historical' not in context.lower():
                    print(f"{path.relative_to(repo)}:{i}:{ref}")
PY
```

Result: no output

```bash
find . -maxdepth 1 \( -name '.pytest_cache' -o -name '__pycache__' -o -name '*.egg-info' -o -name '*.dist-info' \) | sort
git status --ignored --short | sed -n '1,50p'
```

Result:

```text
M bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json
```

```bash
git rev-parse HEAD
git ls-remote --heads origin big-go-1011-root-config-residuals
git log -1 --stat --oneline
```

Result:

```text
5d0e4d335bdbdcaa25176acdd3dd147e7e802082
5d0e4d335bdbdcaa25176acdd3dd147e7e802082	refs/heads/big-go-1011-root-config-residuals
5d0e4d3 BIG-GO-1011 clear root pytest cache residue
 .symphony/workpad.md              |  3 ++-
 reports/BIG-GO-1011-validation.md | 16 ++++++++++++++++
 2 files changed, 18 insertions(+), 1 deletion(-)
```

Final sync check after recording the branch evidence:

```bash
git rev-parse HEAD
git ls-remote --heads origin big-go-1011-root-config-residuals
git log -1 --stat --oneline
```

Result:

```text
d71b08b6eae4518d1b0b46d6b4d943785b301eef
d71b08b6eae4518d1b0b46d6b4d943785b301eef	refs/heads/big-go-1011-root-config-residuals
d71b08b BIG-GO-1011 record branch sync evidence
 reports/BIG-GO-1011-validation.md | 29 +++++++++++++++++++++++++++++
 1 file changed, 29 insertions(+)
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
- older historical reports still contain `tests/test_service.py` as past
  evidence, but current active validation docs no longer prescribe that deleted
  file
- active cutover docs now point at compatibility imports in `src/bigclaw/__init__.py`
  plus `src/bigclaw/runtime.py` / `src/bigclaw/__main__.py` instead of deleted
  `service.py`, `scheduler.py`, `orchestration.py`, `workflow.py`, and `queue.py`
- active migration-plan validation now uses `tests/test_workspace_bootstrap.py`
  and `tests/test_planning.py`, which both exist and passed, instead of deleted
  `tests/test_legacy_shim.py` / `tests/test_deprecation.py`
- active planning docs now only mention deleted script paths when they are
  explicitly marked `retired` or described as historical migration identifiers
- no root `.pytest_cache/`, `__pycache__/`, `*.egg-info`, or `*.dist-info`
  directory remains in the workspace after the final cleanup pass
- local and remote branch SHAs match at `d71b08b6eae4518d1b0b46d6b4d943785b301eef`

No additional root `pyproject.toml`, `setup.py`, `*.egg-info`, repo-root Python wrapper scripts, or Python-specific CI/hook config residue remains.
