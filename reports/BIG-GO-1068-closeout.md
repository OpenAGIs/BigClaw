# BIG-GO-1068 Closeout Index

Issue: `BIG-GO-1068`

Title: `bigclaw-go scripts e2e residual sweep B`

Date: `2026-04-01`

## Branch

`symphony/BIG-GO-1068`

## Baseline

- Starting checkout: `d36a1c700480054955f46d8a02e4c25cf80d094b`
- The nominated `bigclaw-go/scripts/*` Python files were already absent at branch
  start.

## Outcome

- Verified the full nominated batch of `11` Python assets is absent from
  `bigclaw-go/scripts/e2e/` and `bigclaw-go/scripts/migration/`.
- Deleted the one remaining active stale Python test surface,
  `tests/test_live_shadow_bundle.py`, which still executed a removed migration helper.
- Kept validation anchored on the Go-native replacements under
  `bigclaw-go/cmd/bigclawctl` and `bigclaw-go/internal/regression`.
- Reduced repo-wide Python files from `43` to `42`.

## In-Repo Artifacts

- Validation report:
  - `reports/BIG-GO-1068-validation.md`
- Machine-readable status:
  - `reports/BIG-GO-1068-status.json`
- Migration matrix:
  - `bigclaw-go/docs/go-cli-script-migration.md`
- Regression guards:
  - `bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go`
  - `bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`
  - `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`
- Workpad:
  - `.symphony/workpad.md`

## Validation Commands

```bash
find bigclaw-go/scripts/e2e -maxdepth 1 -name '*.py' | wc -l
find bigclaw-go/scripts/migration -maxdepth 1 -name '*.py' | wc -l
find . -name '*.py' | wc -l
rg -n "export_live_shadow_bundle\.py|live_shadow_scorecard\.py|shadow_compare\.py|shadow_matrix\.py|validation_bundle_continuation_scorecard\.py|validation_bundle_continuation_policy_gate(_test)?\.py|run_task_smoke\.py|subscriber_takeover_fault_matrix\.py|multi_node_shared_queue_test\.py|run_all_test\.py" tests bigclaw-go docs README.md workflow.md .github . -g '!reports/**' -g '!.symphony/workpad.md'
cd bigclaw-go && go test ./cmd/bigclawctl/... ./internal/regression/...
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e subscriber-takeover-fault-matrix --help
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-scorecard --help
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-policy-gate --help
cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-compare --help
cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-matrix --help
cd bigclaw-go && go run ./cmd/bigclawctl automation migration live-shadow-scorecard --help
cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle --help
```

## Remaining Risk

No blocking repo work remains inside the issue scope.

The only remaining non-regression string match for the nominated Python asset list is a
historical mapping line in `docs/go-cli-script-migration-plan.md`. It is not an
executable entry surface.
