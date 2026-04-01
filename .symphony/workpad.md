# BIG-GO-1075 Workpad

## Plan
- Confirm the live git-hook execution path for GitHub sync and identify any remaining non-Go default hop in the hook/install flow.
- Move `.githooks/post-commit` and `.githooks/post-rewrite` to invoke the Go github-sync entrypoint directly from `bigclaw-go/cmd/bigclawctl`.
- Teach the Go github-sync installer to materialize the canonical hook scripts so the repo-default and regenerated hooks stay aligned on the same Go-only path.
- Add regression coverage for hook installation/content so the Python-era or wrapper-era hook path does not come back.
- Run targeted validation, capture exact commands/results, then commit and push the branch.

## Acceptance
- `.githooks/post-commit` and `.githooks/post-rewrite` no longer depend on a Python sync path or on `scripts/ops/bigclawctl` as their default execution hop.
- `bigclawctl github-sync install` writes or refreshes those hook scripts with the canonical Go-only content.
- Regression tests pin the generated hook content and install behavior.
- Validation proves the Go-only hook path works and records the exact commands/results.

## Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/githubsync ./cmd/bigclawctl`
- `bash .githooks/post-commit`
- `bash .githooks/post-rewrite`
- `bash scripts/ops/bigclawctl github-sync status --json`
