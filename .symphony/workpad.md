# BIG-GO-1084

## Plan
- inspect the current Python shim, active repo references, and the Go replacement entrypoint
- delete `scripts/ops/bigclaw_refill_queue.py`
- update active documentation and tests to reference `bash scripts/ops/bigclawctl refill` instead of the deleted Python shim
- run targeted validation covering reference cleanup, Go refill command behavior, and Python file-count reduction
- commit and push the scoped change set

## Acceptance
- `scripts/ops/bigclaw_refill_queue.py` is removed from the repository
- active repo guidance no longer tells users or tests to execute `scripts/ops/bigclaw_refill_queue.py`
- the repository `.py` file count decreases from the pre-change baseline
- targeted validation passes and records exact commands plus results

## Validation
- `rg -n "scripts/ops/bigclaw_refill_queue\\.py|python3 scripts/ops/bigclaw_refill_queue\\.py|bigclaw_refill_queue" README.md docs bigclaw-go scripts`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim`
- `bash scripts/ops/bigclawctl refill --help`
- `find . -name '*.py' | wc -l`

## Validation Results
- `rg -n "scripts/ops/bigclaw_refill_queue\\.py|python3 scripts/ops/bigclaw_refill_queue\\.py|bigclaw_refill_queue" README.md docs bigclaw-go scripts` -> exit `1` with no matches
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/legacyshim` -> `ok   bigclaw-go/cmd/bigclawctl 4.295s`; `ok   bigclaw-go/internal/legacyshim 1.892s`
- `bash scripts/ops/bigclawctl refill --help` -> exit `0`; printed `usage: bigclawctl refill [flags]` and the `seed` subcommand help
- `find . -name '*.py' | wc -l` -> `22` after deletion, down from the pre-change baseline `23`
