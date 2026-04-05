# BIG-GO-1358 Legacy Model/Runtime Module Replacement

`BIG-GO-1358` closes out the legacy model/runtime module replacement slice with a
Go-native ownership registry instead of a Python compatibility layer.

## Baseline

- Repository-wide Python file count: `0`.
- The active branch cannot lower the `.py` count further, so acceptance for this
  lane is a concrete Go/native replacement artifact in git.

## Go-Native Replacement Artifact

- Registry path: `bigclaw-go/internal/migration/legacy_model_runtime_modules.go`
- Purpose: record the retired Python module names, the active Go owners that
  replaced them, and the checked-in evidence that keeps the replacement auditable.

## Replacement Mapping

### `src/bigclaw/models.py`

- Replacement kind: `go-package-split`
- Active Go owners:
  - `bigclaw-go/internal/domain/task.go`
  - `bigclaw-go/internal/domain/priority.go`
  - `bigclaw-go/internal/risk/assessment.go`
  - `bigclaw-go/internal/triage/record.go`
  - `bigclaw-go/internal/billing/statement.go`
  - `bigclaw-go/internal/workflow/model.go`
- Checked-in evidence:
  - `docs/go-domain-intake-parity-matrix.md`
  - `bigclaw-go/internal/workflow/model_test.go`

### `src/bigclaw/runtime.py`

- Replacement kind: `go-runtime-mainline`
- Active Go owners:
  - `bigclaw-go/internal/worker/runtime.go`
  - `bigclaw-go/internal/worker/runtime_runonce.go`
  - `bigclaw-go/internal/worker/runtime_test.go`
- Checked-in evidence:
  - `docs/go-mainline-cutover-issue-pack.md`
  - `bigclaw-go/docs/reports/worker-lifecycle-validation-report.md`

## Regression Guard

- `bigclaw-go/internal/regression/big_go_1358_legacy_model_runtime_replacement_test.go`
  verifies the replacement registry contents, the referenced Go paths, and this
  lane report.

## Validation Commands

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1358LegacyModelRuntimeReplacement(ManifestMatchesRetiredModules|ReplacementPathsExist|LaneReportCapturesReplacementState)$'`
