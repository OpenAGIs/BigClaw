# BIG-GO-1089 Workpad

## Plan

1. Inspect the remaining Python operator entrypoints that still front Go-backed workflows and identify the narrowest removable tranche.
2. Delete the selected Python shim files and update any direct references to use the existing `scripts/ops/bigclawctl ...` shell or Go entrypoints instead.
3. Keep the e2e migration guarantees intact by avoiding any reintroduction of Python under `bigclaw-go/scripts/e2e/`.
4. Run targeted validation for the updated operator entrypoints, plus checks that the repository `.py` count decreases and no deleted paths remain on active docs/test surfaces.
5. Commit and push the branch-specific changes.

## Acceptance

- Repository `.py` file count decreases from the baseline captured before edits.
- Deleted Python entrypoints are no longer present in the repository.
- Default operator guidance and validation surfaces point to `scripts/ops/bigclawctl ...` or Go-native commands instead of the deleted Python shims.
- Existing `bigclaw-go/scripts/e2e/` remains Python-free.

## Validation

- `find . -name '*.py' | wc -l`
- `rg -n "scripts/ops/.*\\.py|python3? .*scripts/ops/.*\\.py" README.md docs bigclaw-go .github scripts workflow.md .symphony`
- `bash scripts/ops/bigclawctl refill --help`
- `bash scripts/ops/bigclawctl workspace bootstrap --help`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `cd bigclaw-go && go test ./internal/regression -run TestE2E -count=1`
