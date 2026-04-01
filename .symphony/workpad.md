# BIG-GO-1053

## Plan
- Inspect `bigclaw-go/scripts/e2e` and repo references to identify tranche-2 helper remnants and current Go entrypoints.
- Remove stale Python-helper references and replace them with Go-native `bigclawctl automation e2e ...` commands or existing shell wrappers where appropriate.
- Add/adjust regression coverage so `bigclaw-go/scripts/e2e` stays Python-free and closeout surfaces point at Go-only entrypoints.
- Run targeted tests plus repository checks for `.py` count / reference removal, then commit and push.

## Acceptance
- `bigclaw-go/scripts/e2e` contains no Python helper files for tranche 2.
- README / docs / workflows / hooks / CI do not instruct users to invoke removed tranche-2 Python helpers.
- Validation and regression tests pass for the updated entrypoints.
- Repo evidence shows no remaining tracked `bigclaw-go/scripts/e2e/*.py` files and no stale references to the removed tranche-2 helper paths.

## Validation
- `find bigclaw-go/scripts/e2e -maxdepth 1 -type f | sort`
- `rg -n "bigclaw-go/scripts/e2e/.*\.py|scripts/e2e/.*\.py" README.md bigclaw-go .github .husky .git/hooks`
- `cd bigclaw-go && go test ./cmd/bigclawctl/... ./internal/regression/...`
