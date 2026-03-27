# BIG-GO-904 Workpad

## Plan
- Inventory control-plane and service-plane non-Go dependencies in bigclaw-go.
- Map each dependency to runtime/build/test/ops ownership and identify Go-only migration slices.
- Implement the first low-risk slice that reduces direct non-Go control-plane coupling without broad behavioral change.
- Add migration report documenting validation commands, regression surface, branch/PR proposal, and risk register.
- Run targeted validation, capture exact commands/results, then commit and push a dedicated branch.

## Acceptance
- Executable migration plan and first-batch implementation/change list are checked in.
- Validation commands and regression surface are explicit and reproducible.
- Branch/PR guidance and migration risks are documented.

## Validation
- go test ./... for impacted packages when feasible.
- Additional targeted grep or doc consistency checks for migrated control-plane surfaces.
- git status and git log verification before push.
