Issue: BIG-GO-1108

Plan
- confirm the live remaining Python asset list under `src/bigclaw` and keep `src/bigclaw/legacy_shim.py` as the only frozen compatibility shim
- delete the migrated `src/bigclaw/*.py` tranche that already has Go-owned replacements
- update active Go planning/docs surfaces so they stop pointing at deleted Python modules
- add/refresh Go regression coverage for the purge tranche
- run targeted validation, capture exact commands/results, then commit and push `symphony/BIG-GO-1108`

Acceptance
- lane coverage is explicit: the sweep targets the remaining migrated Python source files under `src/bigclaw`
- the work removes real Python assets instead of tracker-only/doc-only references
- `find . -name '*.py' | wc -l` decreases after the change
- validation commands and residual risks are recorded here and in the final handoff

Validation
- `find src/bigclaw -maxdepth 1 -type f -name '*.py' | sort`
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/planning ./internal/regression ./internal/legacyshim`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json`
- `git status --short`

Lane file list
- `src/bigclaw/audit_events.py`
- `src/bigclaw/collaboration.py`
- `src/bigclaw/console_ia.py`
- `src/bigclaw/deprecation.py`
- `src/bigclaw/design_system.py`
- `src/bigclaw/evaluation.py`
- `src/bigclaw/governance.py`
- `src/bigclaw/models.py`
- `src/bigclaw/observability.py`
- `src/bigclaw/operations.py`
- `src/bigclaw/planning.py`
- `src/bigclaw/reports.py`
- `src/bigclaw/risk.py`
- `src/bigclaw/run_detail.py`
- `src/bigclaw/runtime.py`
- `src/bigclaw/ui_review.py`
- retained shim: `src/bigclaw/legacy_shim.py`

Results
- `find src/bigclaw -maxdepth 1 -type f -name '*.py' | sort` -> `src/bigclaw/legacy_shim.py`
- `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l && find . -name '*.py' | wc -l` -> `17` tracked `.py` files in `HEAD`; `1` `.py` file in the worktree after this sweep
- `cd bigclaw-go && go test ./internal/planning ./internal/regression ./internal/legacyshim` -> passed (`ok   bigclaw-go/internal/planning`, `ok   bigclaw-go/internal/regression`, `ok   bigclaw-go/internal/legacyshim`)
- `bash scripts/ops/bigclawctl legacy-python compile-check --json` -> passed with `status: ok`; checked file list only contained `/Users/openagi/code/bigclaw-workspaces/BIG-GO-1108/src/bigclaw/legacy_shim.py`
- `git status --short` after commit -> clean
- `git rev-parse HEAD` -> `4442611e5405ca52f4d7cf70e967c0190b32adb1`
- `git rev-parse origin/symphony/BIG-GO-1108` -> `4442611e5405ca52f4d7cf70e967c0190b32adb1`
- `git log -1 --stat --oneline` -> `4442611e purge remaining legacy python modules`

Residual risk
- historical migration docs intentionally continue to mention retired Python paths as archived tranche inputs; this lane only corrected active guidance that still described them as live assets
- out-of-repo consumers that still import deleted `src/bigclaw/*` modules directly will now fail; the repository itself retains only the supported `legacy_shim.py` compatibility path
