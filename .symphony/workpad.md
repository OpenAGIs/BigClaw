# BIG-GO-942

## Scope
- Lane title indicates root `scripts` migration to Go CLI.
- Current root/ops Python surfaces in scope:
  - `scripts/create_issues.py`
  - `scripts/dev_smoke.py`
  - `scripts/ops/bigclaw_github_sync.py`
  - `scripts/ops/bigclaw_refill_queue.py`
  - `scripts/ops/bigclaw_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_bootstrap.py`
  - `scripts/ops/symphony_workspace_validate.py`
- Existing non-Python wrappers already present and retained:
  - `scripts/ops/bigclawctl`
  - `scripts/ops/bigclaw-issue`
  - `scripts/ops/bigclaw-panel`
  - `scripts/ops/bigclaw-symphony`
- The migration report path referenced in the issue text is not present in this checkout, so the concrete lane file list is derived from the repository inventory and issue title.

## Plan
1. Verify equivalent Go subcommands exist in `bigclaw-go/cmd/bigclawctl`.
2. Move any Python-only shim behavior into Go or shell wrappers.
3. Replace Python shims with shell entrypoints, or delete when redundant.
4. Run targeted Go tests and wrapper smoke checks.
5. Commit and push only lane-scoped changes.

## Acceptance
- Lane file list is explicit.
- Python scripts under root `scripts/**` for this lane are replaced by Go CLI-backed wrappers or removed with a clear deletion rationale.
- Validation includes exact commands and observed results.
- Residual risks are documented.
- Changes remain scoped to this issue.

## Validation
- `go test ./cmd/bigclawctl`
- Wrapper smoke checks for `create-issues`, `dev-smoke`, `github-sync`, `refill`, `workspace`, and `workspace validate` help/JSON paths as applicable.
- `git diff --stat`

## Results
- `cd bigclaw-go && go test ./cmd/bigclawctl` -> passed (`ok   bigclaw-go/cmd/bigclawctl 5.211s`)
- `python3 -m pytest tests/test_legacy_shim.py tests/test_deprecation.py` -> passed (`17 passed in 7.12s`)
- `bash scripts/dev_smoke.py` -> passed (`smoke_ok local`; legacy wrapper notice emitted on stderr)
- `bash scripts/create_issues.py --help` -> passed (`usage: bigclawctl create-issues [flags]`)
- `bash scripts/ops/bigclaw_refill_queue.py --help` -> passed (`usage: bigclawctl refill [flags]`)
- `bash scripts/ops/bigclaw_github_sync.py status --json` -> passed (`status: ok`, `synced: true`, `dirty: true` because of in-progress lane changes)
- `BIGCLAW_BOOTSTRAP_REPO_URL=<tmp bare repo> BIGCLAW_BOOTSTRAP_CACHE_KEY=compat-cache bash scripts/ops/bigclaw_workspace_bootstrap.py bootstrap ... --json` -> passed (`workspace_mode: worktree_created`)
- `BIGCLAW_BOOTSTRAP_REPO_URL=<tmp bare repo> BIGCLAW_BOOTSTRAP_CACHE_KEY=compat-cache bash scripts/ops/bigclaw_workspace_bootstrap.py cleanup ... --json` -> passed (`workspace_mode: cleanup`, `removed: true`)
- `bash scripts/ops/symphony_workspace_validate.py --repo-url <tmp bare repo> --workspace-root <tmp> --issues COMPAT-VAL-1 COMPAT-VAL-2 --report-file <tmp>/report.json --no-cleanup --json` -> passed (`workspace_count: 2`, report emitted)

## Residual Risks
- The legacy wrapper paths still end in `.py`, but they now require shell execution semantics rather than `python3`; external callers still hardcoding `python3 <wrapper>.py` will need to switch to `bash ...` or `scripts/ops/bigclawctl ...`.
- `scripts/ops/bigclawctl` still uses `go run`, so wrapper latency and local Go toolchain availability remain operator dependencies.

## Follow-up Artifacts
- `reports/BIG-GO-942-validation.md`
- `reports/BIG-GO-942-pr.md`
- `reports/BIG-GO-942-closeout.md`
- `reports/BIG-GO-942-status.json`
