# BIG-GO-945 Workpad

## Plan

1. Confirm the lane-5 file list from `docs/go-mainline-cutover-issue-pack.md` and map each Python module under `src/bigclaw` to an existing or missing Go owner.
2. Inspect the current Python and Go implementations for `console_ia`, `design_system`, `saved_views`, `ui_review`, and operator-facing `service` surfaces to find the smallest real migration slice still missing in Go.
3. Implement the missing Go replacement or, if the Go owner already exists, remove or deprecate the redundant Python surface with a scoped deletion plan and regression coverage.
4. Run targeted validation for the touched Go packages and any directly affected Python compatibility surface, recording exact commands and results.
5. Commit the scoped changes and push the branch to the remote tracking branch.

## Acceptance

- Explicitly identify the lane-5 Python module inventory for this issue.
- Land a Go replacement and/or deletion plan for the remaining lane-5 Python application modules.
- Keep the change scoped to lane-5 ownership surfaces.
- Record exact validation commands, results, and remaining migration risks.

## Validation

- `go test ./internal/product ./internal/api ./cmd/bigclawctl`
- Additional targeted `go test` commands for any extra Go package touched by the implementation.
- Targeted Python regression only if a retained compatibility shim or legacy import surface is edited.
