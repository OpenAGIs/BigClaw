# BIG-GO-1061 Validation

Date: 2026-04-02

## Scope

Issue: `BIG-GO-1061`

Title: `src/bigclaw runtime/orchestration residual sweep`

This lane closes the remaining package-entry Python residue from the suggested
`src/bigclaw` tranche that still existed in this checkout:

- `src/bigclaw/__main__.py`
- `src/bigclaw/deprecation.py`

The lane also tightens the Go-owned validation path so the frozen legacy Python
compile-check reflects the actual residual shim surface instead of deleted files.

## Delivered

- deleted the legacy Python entrypoint `src/bigclaw/__main__.py`, removing the
  `python -m bigclaw` execution path from the active repo surface
- deleted `src/bigclaw/deprecation.py` and inlined its tiny warning helper into
  `src/bigclaw/runtime.py`
- updated `src/bigclaw/__init__.py` package metadata to describe the remaining
  `bigclaw.service` compatibility surface without referencing deleted files
- narrowed `bigclaw-go/internal/legacyshim/compilecheck.go` and its tests to the
  surviving frozen shim list:
  - `src/bigclaw/__init__.py`
  - `src/bigclaw/legacy_shim.py`
  - `src/bigclaw/runtime.py`
- refreshed Go-mainline guidance in:
  - `README.md`
  - `bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json`
  - `docs/go-mainline-cutover-handoff.md`
- added `bigclaw-go/internal/regression/top_level_module_purge_tranche14_test.go`
  to keep `src/bigclaw/__main__.py` and `src/bigclaw/deprecation.py` deleted

## Validation

### Python regression sweep

Command:

```bash
PYTHONPATH=src python3 -m pytest tests/test_top_level_module_shims.py tests/test_repo_collaboration.py tests/test_observability.py tests/test_planning.py tests/test_evaluation.py tests/test_operations.py tests/test_design_system.py tests/test_console_ia.py -q
```

Result:

```text
76 passed in 0.28s
```

### Go legacy-shim and regression coverage

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1061/bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl -count=1
```

Result:

```text
ok  	bigclaw-go/internal/legacyshim	1.316s
ok  	bigclaw-go/internal/regression	1.569s
ok  	bigclaw-go/cmd/bigclawctl	4.678s
```

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1061/bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche14 -count=1
```

Result:

```text
ok  	bigclaw-go/internal/regression	1.893s
```

### Go-owned legacy Python compile-check

Command:

```bash
bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-1061/scripts/ops/bigclawctl legacy-python compile-check --json
```

Result:

```json
{
  "files": [
    "/Users/openagi/code/bigclaw-workspaces/BIG-GO-1061/src/bigclaw/__init__.py",
    "/Users/openagi/code/bigclaw-workspaces/BIG-GO-1061/src/bigclaw/legacy_shim.py",
    "/Users/openagi/code/bigclaw-workspaces/BIG-GO-1061/src/bigclaw/runtime.py"
  ],
  "python": "python3",
  "repo": "/Users/openagi/code/bigclaw-workspaces/BIG-GO-1061",
  "status": "ok"
}
```

### Branch sync verification

Command:

```bash
bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-1061/scripts/ops/bigclawctl github-sync status --json
```

Result:

```json
{
  "ahead": 0,
  "behind": 0,
  "branch": "big-go-1061-residual-sweep",
  "detached": false,
  "dirty": false,
  "diverged": false,
  "local_sha": "78f957908a6415f7d87ad8f84a670f9ad3d2fc7b",
  "pushed": true,
  "relation_known": true,
  "remote_exists": true,
  "remote_sha": "78f957908a6415f7d87ad8f84a670f9ad3d2fc7b",
  "status": "ok",
  "synced": true
}
```

### Python file-count impact

Command:

```bash
find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1061 -name '*.py' | wc -l
```

Result:

```text
38
```

Command:

```bash
rg --files /Users/openagi/code/bigclaw-workspaces/BIG-GO-1061/src/bigclaw | rg '\.py$' | wc -l
```

Result:

```text
11
```

Measured impact for this lane:

- repo-wide `.py` count: `40 -> 38` (`-2`)
- `src/bigclaw` `.py` count: `13 -> 11` (`-2`)

## Commit And Push

- Commit: `78f957908a6415f7d87ad8f84a670f9ad3d2fc7b`
- Message: `BIG-GO-1061: purge residual package entry shims`
- Push: `git push origin big-go-1061-residual-sweep` succeeded

## Residual Risk

- The broader `src/bigclaw` package still contains retained migration-only Python
  compatibility files such as `runtime.py`, `legacy_shim.py`, and `__init__.py`;
  this lane only removed the redundant package-entry residue in scope.
- Historical reports and tracker comments may still mention deleted
  `src/bigclaw/__main__.py` or `src/bigclaw/deprecation.py`, but active code,
  validation paths, and operator guidance now point at the Go-first replacements.
