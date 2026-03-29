# BIG-GO-942

## Scope
- Issue: `BIG-GO-942`
- Title: `Lane2 Root scripts to Go CLI`
- Lane file list for this slice:
  - `scripts/create_issues.py`
  - `scripts/dev_smoke.py`
  - `scripts/ops/bigclaw_github_sync.py`
  - `scripts/ops/bigclaw_refill_queue.py`
  - `scripts/ops/bigclaw_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_validate.py`
- Existing implementation state:
  - lane-scoped Python implementations are already replaced by shell wrappers over `scripts/ops/bigclawctl`
  - behavior ownership lives in `bigclaw-go/cmd/bigclawctl`
- Out of scope:
  - `bigclaw-go/scripts/**`
  - `scripts/dev_bootstrap.sh`
  - unrelated report-sync or repo-wide documentation cleanup

## Plan
1. Reconfirm the lane file inventory and current wrapper behavior.
2. Run targeted validation for the Go CLI and retained compatibility wrappers.
3. Refresh issue-scoped artifacts with exact commands, results, current head commit, and residual risks.
4. Commit and push only the metadata refresh required for this unattended execution.

## Acceptance
- The lane file list is explicit and limited to the root `scripts/**` slice above.
- Go-backed replacement or retained wrapper behavior is documented for each lane file.
- Validation records exact commands and observed results.
- Residual risks are captured without expanding scope.
- Changes stay scoped to `BIG-GO-942`.

## Validation
- `cd bigclaw-go && go test ./cmd/bigclawctl`
- `python3 -m pytest tests/test_legacy_shim.py tests/test_deprecation.py`
- `bash scripts/dev_smoke.py`
- `bash scripts/create_issues.py --help`
- `bash scripts/ops/bigclaw_refill_queue.py --help`
- `bash scripts/ops/bigclaw_github_sync.py status --json`
- `BIGCLAW_BOOTSTRAP_REPO_URL=<tmp seeded bare repo with main> BIGCLAW_BOOTSTRAP_CACHE_KEY=compat-cache bash scripts/ops/bigclaw_workspace_bootstrap.py bootstrap --workspace <tmp>/workspaces/COMPAT-BOOT-1 --issue COMPAT-BOOT-1 --cache-base <tmp>/cache --json`
- `BIGCLAW_BOOTSTRAP_REPO_URL=<tmp seeded bare repo with main> BIGCLAW_BOOTSTRAP_CACHE_KEY=compat-cache bash scripts/ops/bigclaw_workspace_bootstrap.py cleanup --workspace <tmp>/workspaces/COMPAT-BOOT-1 --issue COMPAT-BOOT-1 --cache-base <tmp>/cache --json`
- `bash scripts/ops/symphony_workspace_validate.py --repo-url <tmp seeded bare repo with main> --workspace-root <tmp>/validate --issues COMPAT-VAL-1 COMPAT-VAL-2 --report-file <tmp>/report.json --no-cleanup --json`

## Notes
- The migration report path named in the issue text is not present in this checkout, so the lane file list is derived from the repo inventory and the issue title.
- This run is a verification and artifact-refresh pass on top of the existing lane implementation already present on branch `symphony/BIG-GO-942`.
