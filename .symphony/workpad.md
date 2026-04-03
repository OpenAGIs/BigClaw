# BIG-GO-1102

## Plan
- confirm the current `bigclaw-go/scripts` automation surface is already Python-free and locate live residual references to the removed script tranche
- update the migration/regression guardrails so they assert the current Go/shell-only script surface without carrying stale Python file manifests longer than needed
- remove a small self-contained cluster of legacy Python source assets that already have Go-native owners and are only retained as planning evidence
- update planning/docs references so they point at the Go-native ownership/tests instead of deleted Python source files
- run targeted validation for the affected Go planning/regression surfaces plus repository Python-file counts
- commit the scoped change set and push the branch

## Acceptance
- lane coverage is explicit: removed `bigclaw-go/scripts` Python automation files stay absent and live references are cleaned up
- the change deletes real Python source assets rather than only editing tracker/docs cosmetics
- `find . -name '*.py' | wc -l` decreases from the pre-change baseline of `17`
- exact validation commands and results are recorded below

## Validation
- `find bigclaw-go/scripts -name '*.py' | sort`
- `find . -name '*.py' | wc -l`
- `rg -n "src/bigclaw/(runtime|deprecation)\\.py|bigclaw-go/scripts/(benchmark|e2e|migration)/[A-Za-z0-9_./-]+\\.py|scripts/(benchmark|e2e|migration)/[A-Za-z0-9_./-]+\\.py" README.md bigclaw-go/docs docs scripts .github -g '!docs/go-mainline-cutover-issue-pack.md' -g '!bigclaw-go/internal/regression/*.go'`
- `cd bigclaw-go && go test ./internal/regression ./internal/legacyshim ./cmd/bigclawctl`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json`
- `git status --short`

## Validation Results
- pre-change `find . -name '*.py' | wc -l` -> `17`
- `find bigclaw-go/scripts -name '*.py' | sort` -> exit `0` with no output
- post-change `find . -name '*.py' | wc -l` -> `12`
- `rg -n "src/bigclaw/(runtime|deprecation)\\.py|bigclaw-go/scripts/(benchmark|e2e|migration)/[A-Za-z0-9_./-]+\\.py|scripts/(benchmark|e2e|migration)/[A-Za-z0-9_./-]+\\.py" README.md bigclaw-go/docs docs scripts .github -g '!docs/go-mainline-cutover-issue-pack.md' -g '!bigclaw-go/internal/regression/*.go'` -> exit `1` with no matches
- `cd bigclaw-go && go test ./internal/regression ./internal/legacyshim ./cmd/bigclawctl` -> `ok   bigclaw-go/internal/regression (cached)`; `ok   bigclaw-go/internal/legacyshim (cached)`; `ok   bigclaw-go/cmd/bigclawctl (cached)`
- `bash scripts/ops/bigclawctl legacy-python compile-check --json` -> exit `0`; JSON reported `status: ok`, `python: python3`, and the single checked file `/Users/openagi/code/bigclaw-workspaces/BIG-GO-1102/src/bigclaw/legacy_shim.py`
- `git status --short` after the pushed base commit showed only the follow-up issue edits in `.symphony/workpad.md`, `README.md`, `docs/go-cli-script-migration-plan.md`, `src/bigclaw/deprecation.py`, `src/bigclaw/runtime.py`, and `bigclaw-go/internal/regression/runtime_residue_purge_test.go`
