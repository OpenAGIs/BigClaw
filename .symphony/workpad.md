# BIG-GO-1068 Workpad

## Plan
- Confirm the nominated residual Python assets under `bigclaw-go/scripts/e2e/` and `bigclaw-go/scripts/migration/` are already absent from the current checkout.
- Remove any remaining live Python test or doc surfaces that still execute deleted `bigclaw-go/scripts/*/*.py` entrypoints.
- Record issue-scoped closeout artifacts with the exact asset list, Go replacement paths, validation commands, residual risks, and Python file-count impact.
- Run targeted validation covering the Go command replacements and existing regression surfaces.
- Commit and push the issue branch.

## Acceptance
- The batch asset list for `BIG-GO-1068` is captured in-repo.
- No live Python files remain for the nominated `bigclaw-go/scripts/e2e/` and `bigclaw-go/scripts/migration/` assets.
- Any stale active test surface that still shells deleted Python script paths is removed or replaced by Go-native coverage.
- Validation commands and exact results are recorded.
- The repo-wide `.py` file count is reduced or an explicit zero-delta explanation is documented.

## Validation
- `find bigclaw-go/scripts/e2e -maxdepth 1 -name '*.py' | wc -l`
- `find bigclaw-go/scripts/migration -maxdepth 1 -name '*.py' | wc -l`
- `find . -name '*.py' | wc -l`
- `rg -n "export_live_shadow_bundle\\.py|live_shadow_scorecard\\.py|shadow_compare\\.py|shadow_matrix\\.py|validation_bundle_continuation_scorecard\\.py|validation_bundle_continuation_policy_gate(_test)?\\.py|run_task_smoke\\.py|subscriber_takeover_fault_matrix\\.py|multi_node_shared_queue_test\\.py|run_all_test\\.py" tests bigclaw-go docs README.md workflow.md .github . -g '!reports/**' -g '!.symphony/workpad.md'`
- `cd bigclaw-go && go test ./cmd/bigclawctl/... ./internal/regression/...`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help | head -n 1`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e subscriber-takeover-fault-matrix --help | head -n 1`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-scorecard --help | head -n 1`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-policy-gate --help | head -n 1`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-compare --help | head -n 1`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-matrix --help | head -n 1`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration live-shadow-scorecard --help | head -n 1`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle --help | head -n 1`
