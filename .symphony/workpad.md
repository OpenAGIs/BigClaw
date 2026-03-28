# BIG-GO-922 Workpad

## Plan

1. Inventory current `scripts/*.py`, `scripts/*.sh`, and `scripts/ops/*` assets and separate thin compatibility shims from scripts that still own behavior.
2. Migrate the remaining behavior-bearing repo script surfaces into `bigclaw-go/cmd/bigclawctl`.
3. Keep legacy script paths as compatibility shims where direct deletion would break operator entrypoints.
4. Update migration docs with the current asset inventory, first-batch replacement map, removal conditions, and regression commands.
5. Run targeted tests and CLI smoke checks, then commit and push the issue branch.

## Acceptance

- Current Python/non-Go script inventory is documented with ownership status.
- At least one additional repo-level script surface is implemented as a Go CLI/subcommand in this slice.
- Legacy script deletion conditions are explicit.
- Validation commands and exact results are recorded for this slice.

## Validation

- `cd bigclaw-go && go test ./cmd/bigclawctl`
- `cd bigclaw-go && go test ./cmd/bigclawctl/...`
- `cd bigclaw-go && go run ./cmd/bigclawctl dev bootstrap --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl compat exec --help`
- `bash scripts/dev_bootstrap.sh --help`
- `bash scripts/ops/bigclawctl --help`
