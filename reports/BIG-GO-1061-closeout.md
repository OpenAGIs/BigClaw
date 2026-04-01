# BIG-GO-1061 Closeout Index

Issue: `BIG-GO-1061`

Title: `src/bigclaw runtime/orchestration residual sweep`

Date: `2026-04-02`

## Branch

`big-go-1061-residual-sweep`

## Latest Code Migration Commit

`78f957908a6415f7d87ad8f84a670f9ad3d2fc7b`

## In-Repo Artifacts

- Validation report:
  - `reports/BIG-GO-1061-validation.md`
- Machine-readable status:
  - `reports/BIG-GO-1061-status.json`
- Workpad:
  - `.symphony/workpad.md`

## Outcome

- `src/bigclaw/__main__.py` and `src/bigclaw/deprecation.py` were removed from
  the repository.
- The surviving runtime compatibility surface in `src/bigclaw/runtime.py` now
  owns the small legacy warning helper directly.
- The Go-owned `legacy-python compile-check` now validates the actual retained
  shim set instead of referencing deleted Python files.
- The repo now has regression coverage that keeps the deleted package-entry
  modules absent.
- This lane reduced Python file counts by `-2` repo-wide and `-2` under
  `src/bigclaw`.

## Validation Commands

```bash
PYTHONPATH=src python3 -m pytest tests/test_top_level_module_shims.py tests/test_repo_collaboration.py tests/test_observability.py tests/test_planning.py tests/test_evaluation.py tests/test_operations.py tests/test_design_system.py tests/test_console_ia.py -q
cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl -count=1
cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche14 -count=1
bash scripts/ops/bigclawctl legacy-python compile-check --json
bash scripts/ops/bigclawctl github-sync status --json
find . -name '*.py' | wc -l
rg --files src/bigclaw | rg '\.py$' | wc -l
```

## Remaining Risk

No blocking repo action remains for `BIG-GO-1061`.

The remaining risk is limited to other retained migration-only Python
compatibility surfaces outside this lane's deleted package-entry files.

## Final Repo Check

- `bash scripts/ops/bigclawctl github-sync status --json` reported `status: ok`
  and `synced: true` for `origin/big-go-1061-residual-sweep`.
- `git rev-parse HEAD` matched `git rev-parse origin/big-go-1061-residual-sweep`
  at `9df504144ab1e1dc0ca026e6992b6b6459a56b73` when this closeout was recorded.
- `.symphony/workpad.md` records the handled asset list, exact validation
  commands, and the Python file-count impact for this tranche.
