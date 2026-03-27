# BIG-GO-902

## Plan
- Audit repo-root `scripts/*.py`, `scripts/ops/*`, and `bigclaw-go/cmd/bigclawctl` to confirm the migrated surface and identify any remaining repo-root gaps.
- Re-validate the delivered Go CLI migration slice with focused Go tests, Python regression tests, syntax checks, and command-level shim invocations.
- Refresh issue artifacts so the migration plan, validated commit, command results, branch guidance, and risk notes all match the current branch head.
- Commit the report-sync delta and push `feat/BIG-GO-902-go-cli-script-migration`.

## Acceptance
- Repo-root automation entrypoints remain owned by `bigclawctl` subcommands, with legacy Python/Bash names preserved only as compatibility shims.
- The repo contains an executable migration plan plus a first-batch entrypoint list, validation commands, regression surface, branch/PR suggestion, and risks.
- Validation artifacts reference the current branch head and the exact commands/results from this execution pass.

## Validation
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-902/bigclaw-go && go test ./cmd/bigclawctl ./internal/refill`
  - Result: `ok  	bigclaw-go/cmd/bigclawctl	2.651s`; `ok  	bigclaw-go/internal/refill	(cached)`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-902 && python3 -m pytest tests/test_legacy_shim.py tests/test_deprecation.py`
  - Result: `17 passed in 1.76s`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-902 && python3 -m py_compile src/bigclaw/legacy_shim.py scripts/ops/bigclaw_github_sync.py scripts/ops/bigclaw_refill_queue.py scripts/ops/bigclaw_workspace_bootstrap.py scripts/ops/symphony_workspace_bootstrap.py scripts/ops/symphony_workspace_validate.py scripts/create_issues.py scripts/dev_smoke.py`
  - Result: exit code `0`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-902 && bash scripts/ops/bigclawctl dev-smoke`
  - Result: `smoke_ok local`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-902 && python3 scripts/dev_smoke.py`
  - Result: deprecation warning emitted, then `smoke_ok local`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-902 && python3 scripts/create_issues.py --help`
  - Result: usage for `bigclawctl create-issues`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-902 && bash scripts/ops/bigclawctl issue --help`
  - Result: usage for `bigclawctl issue`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-902 && python3 scripts/ops/bigclaw_github_sync.py --help`
  - Result: usage for `bigclawctl github-sync`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-902 && python3 scripts/ops/bigclaw_workspace_bootstrap.py --help`
  - Result: usage for `bigclawctl workspace <bootstrap|cleanup|validate>`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-902 && python3 scripts/ops/symphony_workspace_bootstrap.py --help`
  - Result: usage for `bigclawctl workspace <bootstrap|cleanup|validate>`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-902 && python3 scripts/ops/bigclaw_refill_queue.py --help`
  - Result: usage for `bigclawctl refill`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-902 && python3 scripts/ops/symphony_workspace_validate.py --help`
  - Result: usage for `bigclawctl workspace validate`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-902 && python3 scripts/ops/bigclaw_github_sync.py status --json`
  - Result: branch `feat/BIG-GO-902-go-cli-script-migration`, `synced=true`, `dirty=false`, `local_sha=7bf0f59f3c8649328cabaca1e619136fbf114a30`
