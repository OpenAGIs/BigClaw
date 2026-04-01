# BIG-GO-1068 Validation

Date: 2026-04-01

## Scope

Issue: `BIG-GO-1068`

Title: `bigclaw-go scripts e2e residual sweep B`

This lane verifies and closes the remaining Python-entrypoint sweep for the
following nominated assets:

- `bigclaw-go/scripts/e2e/multi_node_shared_queue_test.py`
- `bigclaw-go/scripts/e2e/run_all_test.py`
- `bigclaw-go/scripts/e2e/run_task_smoke.py`
- `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
- `bigclaw-go/scripts/migration/export_live_shadow_bundle.py`
- `bigclaw-go/scripts/migration/live_shadow_scorecard.py`
- `bigclaw-go/scripts/migration/shadow_compare.py`
- `bigclaw-go/scripts/migration/shadow_matrix.py`

The nominated script files are already absent in the checked-out baseline. The
remaining live residue in this checkout was one stale Python test,
`tests/test_live_shadow_bundle.py`, which still executed the deleted
`bigclaw-go/scripts/migration/export_live_shadow_bundle.py` path. This change
deletes that test because equivalent Go-native coverage already exists in:

- `bigclaw-go/cmd/bigclawctl/automation_commands_test.go`
- `bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go`

## Delivered

- Confirmed `bigclaw-go/scripts/e2e/` contains no `.py` files.
- Confirmed `bigclaw-go/scripts/migration/` no longer exists as a Python script
  surface; the migration entrypoints resolve through
  `bigclawctl automation migration ...`.
- Removed stale Python test:
  - `tests/test_live_shadow_bundle.py`
- Confirmed active Go replacement surfaces remain documented in
  `bigclaw-go/docs/go-cli-script-migration.md`:
  - `go run ./cmd/bigclawctl automation e2e run-task-smoke ...`
  - `go run ./cmd/bigclawctl automation e2e continuation-scorecard ...`
  - `go run ./cmd/bigclawctl automation e2e continuation-policy-gate ...`
  - `go run ./cmd/bigclawctl automation e2e subscriber-takeover-fault-matrix ...`
  - `go run ./cmd/bigclawctl automation migration shadow-compare ...`
  - `go run ./cmd/bigclawctl automation migration shadow-matrix ...`
  - `go run ./cmd/bigclawctl automation migration live-shadow-scorecard ...`
  - `go run ./cmd/bigclawctl automation migration export-live-shadow-bundle ...`

## Validation

### Python file counts

Command:

```bash
find bigclaw-go/scripts/e2e -maxdepth 1 -name '*.py' | wc -l
```

Result:

```text
0
```

Command:

```bash
find bigclaw-go/scripts/migration -maxdepth 1 -name '*.py' | wc -l
```

Result:

```text
find: bigclaw-go/scripts/migration: No such file or directory
0
```

Command:

```bash
find . -name '*.py' | wc -l
```

Result before this change:

```text
43
```

Result after deleting `tests/test_live_shadow_bundle.py`:

```text
42
```

Impact:

- Repo-wide Python file count decreased by `1` in this lane.
- All `11` nominated `bigclaw-go/scripts/*` Python assets are absent in the final
  checkout.

### Residual reference scan

Command:

```bash
rg -n "export_live_shadow_bundle\.py|live_shadow_scorecard\.py|shadow_compare\.py|shadow_matrix\.py|validation_bundle_continuation_scorecard\.py|validation_bundle_continuation_policy_gate(_test)?\.py|run_task_smoke\.py|subscriber_takeover_fault_matrix\.py|multi_node_shared_queue_test\.py|run_all_test\.py" tests bigclaw-go docs README.md workflow.md .github . -g '!reports/**' -g '!.symphony/workpad.md'
```

Result:

```text
docs/go-cli-script-migration-plan.md:30:- `bigclaw-go/scripts/migration/shadow_compare.py` -> `bigclawctl automation migration shadow-compare`
bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go:46:		"bigclaw-go/scripts/e2e/run_task_smoke.py",
bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go:48:		"bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py",
bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go:49:		"bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py",
bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go:53:		"bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py"
./bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go:46:		"bigclaw-go/scripts/e2e/run_task_smoke.py",
./bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go:48:		"bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py",
./bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go:49:		"bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py",
./bigclaw-go/internal/regression/e2e_entrypoint_migration_test.go:53:		"bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py",
./docs/go-cli-script-migration-plan.md:30:- `bigclaw-go/scripts/migration/shadow_compare.py` -> `bigclawctl automation migration shadow-compare`
./local-issues.json:378: historical issue comment text
./local-issues.json:401: historical issue comment text
./local-issues.json:424: historical issue comment text
./local-issues.json:447: historical issue comment text
./local-issues.json:493: historical issue comment text
```

Interpretation:

- No active Python test or runtime entry surface remains for the nominated deleted
  scripts.
- The remaining matches are intentional:
  - one historical migration-plan mapping entry
  - regression guard literals that prevent deleted helper paths from reappearing
  - historical tracker comments in `local-issues.json`

### Targeted Go tests

Command:

```bash
cd bigclaw-go && go test ./cmd/bigclawctl/... ./internal/regression/...
```

Result:

```text
ok  	bigclaw-go/cmd/bigclawctl	8.345s
ok  	bigclaw-go/internal/regression	5.987s
```

### Go replacement help checks

Commands:

```bash
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help | head -n 1
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e subscriber-takeover-fault-matrix --help | head -n 1
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-scorecard --help | head -n 1
cd bigclaw-go && go run ./cmd/bigclawctl automation e2e continuation-policy-gate --help | head -n 1
cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-compare --help | head -n 1
cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-matrix --help | head -n 1
cd bigclaw-go && go run ./cmd/bigclawctl automation migration live-shadow-scorecard --help | head -n 1
cd bigclaw-go && go run ./cmd/bigclawctl automation migration export-live-shadow-bundle --help | head -n 1
```

Result:

```text
usage: bigclawctl automation e2e run-task-smoke [flags]
usage: bigclawctl automation e2e subscriber-takeover-fault-matrix [flags]
usage: bigclawctl automation e2e continuation-scorecard [flags]
usage: bigclawctl automation e2e continuation-policy-gate [flags]
usage: bigclawctl automation migration shadow-compare [flags]
usage: bigclawctl automation migration shadow-matrix [flags]
usage: bigclawctl automation migration live-shadow-scorecard [flags]
usage: bigclawctl automation migration export-live-shadow-bundle [flags]
```

## Residual Risk

- `docs/go-cli-script-migration-plan.md` still contains a historical mapping line for
  `bigclaw-go/scripts/migration/shadow_compare.py`. It does not execute code and is
  retained as migration history, but it is the only non-regression reference left in
  live docs outside `reports/`.
- This lane does not add new Go behavior; it removes the stale Python test residue and
  records the issue-scoped validation evidence for already-landed Go replacements.
