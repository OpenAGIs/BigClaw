# BIG-GO-1095

## Plan
- confirm the current `bigclaw-go/scripts` tree and identify residual migration text that still advertises Python helper files or follow-up script tranches
- update migration docs and related repo guidance so `bigclaw-go/scripts` is described as a Go-first, Python-free surface
- widen regression coverage from the old e2e-only check to the full `bigclaw-go/scripts` tree and lock docs against stale Python script references
- run targeted validation, record exact commands and results, then commit and push the scoped branch changes

## Acceptance
- repo guidance no longer states that `bigclaw-go/scripts/*` migration work remains deferred when the tracked tree is already Python-free
- `bigclaw-go/docs/go-cli-script-migration.md` describes the full active Go/script surface without naming deleted Python helper paths as live entrypoints
- regression coverage fails if any `.py` file reappears anywhere under `bigclaw-go/scripts`
- regression coverage fails if the script-migration docs regress to stale deleted Python helper references or stale "remaining backlog" language for `bigclaw-go/scripts`
- targeted validation is captured with exact commands and outcomes

## Validation
- `find bigclaw-go/scripts -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/regression`
- `cd bigclaw-go && go test ./cmd/bigclawctl/...`
- `rg -n "bigclaw-go/scripts/.+\\.py|scripts/e2e/.+\\.py|scripts/benchmark/.+\\.py|scripts/migration/.+\\.py|remaining `bigclaw-go/scripts|deferred to a follow-up migration lane|Continue the remaining" README.md docs bigclaw-go/docs bigclaw-go/internal`

## Results
- `find bigclaw-go/scripts -name '*.py' | wc -l` -> `0`
- `cd bigclaw-go && go test ./internal/regression` -> `ok   bigclaw-go/internal/regression 0.838s`
- `cd bigclaw-go && go test ./cmd/bigclawctl/...` -> `ok   bigclaw-go/cmd/bigclawctl 3.952s`
- `rg -n 'bigclaw-go/scripts/.+\\.py|scripts/e2e/.+\\.py|scripts/benchmark/.+\\.py|scripts/migration/.+\\.py|Continue the remaining `bigclaw-go/scripts|deferred to a follow-up migration lane|follow-up batches without expanding the e2e compatibility layer again' README.md docs bigclaw-go/docs -g '!bigclaw-go/internal/**'` -> exit `1` with no matches
