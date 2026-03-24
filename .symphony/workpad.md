# BIGCLAW-177 Workpad

## Plan

1. Inspect the existing control center and distributed diagnostics payload builders in `bigclaw-go/internal/api` and keep the change isolated to those surfaces.
2. Add a ClawHost proxy/domain routing observability bridge that derives route, domain, bot, and executor evidence from task metadata plus task event payloads.
3. Expose the new bridge in the control center and distributed diagnostics JSON payloads with a summary and traceable mapping rows.
4. Render the same routing/domain bridge in the distributed diagnostics markdown/export output.
5. Extend targeted Go tests for payload and markdown coverage, then run the focused test set, commit, and push the branch.

## Acceptance

1. The control center and distributed diagnostics payloads include a route/domain summary surface.
2. Bot and executor to entry-domain mappings are traceable through the new payload contract.
3. Distributed diagnostics markdown/export renders the new routing/domain observability section and the related tests pass.

## Validation

- Run focused Go tests for `internal/api` control center and distributed report coverage.
- Verify the JSON payload contains the new route/domain summary and mapping rows.
- Verify the markdown/export contains the ClawHost routing observability section with route/domain mapping details.
- Record exact test commands and results for handoff before commit/push.
