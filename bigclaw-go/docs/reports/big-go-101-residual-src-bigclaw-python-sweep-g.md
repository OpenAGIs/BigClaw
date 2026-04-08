# BIG-GO-101 Residual `src/bigclaw` Python Sweep G

## Scope

This refill lane records exact Go replacement evidence for the largest retired
`src/bigclaw` reporting, operations, policy, governance, operator, product,
bootstrap, and sync modules that still merit explicit replacement tracking in
this checkout:

- `src/bigclaw/observability.py`
- `src/bigclaw/reports.py`
- `src/bigclaw/evaluation.py`
- `src/bigclaw/operations.py`
- `src/bigclaw/risk.py`
- `src/bigclaw/governance.py`
- `src/bigclaw/execution_contract.py`
- `src/bigclaw/audit_events.py`
- `src/bigclaw/issue_archive.py`
- `src/bigclaw/run_detail.py`
- `src/bigclaw/dashboard_run_contract.py`
- `src/bigclaw/saved_views.py`
- `src/bigclaw/console_ia.py`
- `src/bigclaw/design_system.py`
- `src/bigclaw/github_sync.py`
- `src/bigclaw/workspace_bootstrap.py`
- `src/bigclaw/workspace_bootstrap_cli.py`
- `src/bigclaw/workspace_bootstrap_validation.py`
- `src/bigclaw/parallel_refill.py`
- `src/bigclaw/service.py`
- `src/bigclaw/__main__.py`

## Before And After Counts

- Repository-wide physical Python file count before lane changes: `0`
- Repository-wide physical Python file count after lane changes: `0`
- Focused `src/bigclaw` sweep-G physical Python file count before lane changes: `0`
- Focused `src/bigclaw` sweep-G physical Python file count after lane changes: `0`

This branch was already Python-free before the lane started, so the delivered
work is replacement-evidence hardening rather than an in-branch file deletion
batch.

## Exact Deleted-File Ledger

Deleted files in this lane: `[]`

Focused sweep-G ledger: `[]`

## Retired Python Surface

- `src/bigclaw`: directory not present, so residual Python files = `0`
- `src/bigclaw/observability.py`
- `src/bigclaw/reports.py`
- `src/bigclaw/evaluation.py`
- `src/bigclaw/operations.py`
- `src/bigclaw/risk.py`
- `src/bigclaw/governance.py`
- `src/bigclaw/execution_contract.py`
- `src/bigclaw/audit_events.py`
- `src/bigclaw/issue_archive.py`
- `src/bigclaw/run_detail.py`
- `src/bigclaw/dashboard_run_contract.py`
- `src/bigclaw/saved_views.py`
- `src/bigclaw/console_ia.py`
- `src/bigclaw/design_system.py`
- `src/bigclaw/github_sync.py`
- `src/bigclaw/workspace_bootstrap.py`
- `src/bigclaw/workspace_bootstrap_cli.py`
- `src/bigclaw/workspace_bootstrap_validation.py`
- `src/bigclaw/parallel_refill.py`
- `src/bigclaw/service.py`
- `src/bigclaw/__main__.py`

## Structured Replacement Ledger

This lane adds the structured replacement registry at
`bigclaw-go/internal/migration/legacy_reporting_ops_modules.go`.

It also adds the structured replacement registry at
`bigclaw-go/internal/migration/legacy_policy_governance_modules.go`.

It also adds the structured replacement registry at
`bigclaw-go/internal/migration/legacy_operator_product_modules.go`.

It also adds the structured replacement registry at
`bigclaw-go/internal/migration/legacy_bootstrap_sync_modules.go`.

### `src/bigclaw/observability.py`

- Replacement kind: `go-observability-runtime`
- Go replacements:
  - `bigclaw-go/internal/observability/recorder.go`
  - `bigclaw-go/internal/observability/task_run.go`
  - `bigclaw-go/internal/observability/audit.go`
- Evidence:
  - `bigclaw-go/internal/observability/recorder_test.go`
  - `bigclaw-go/internal/observability/task_run_test.go`
  - `docs/go-mainline-cutover-issue-pack.md`

### `src/bigclaw/reports.py`

- Replacement kind: `go-reporting-surface`
- Go replacements:
  - `bigclaw-go/internal/reporting/reporting.go`
  - `bigclaw-go/internal/reportstudio/reportstudio.go`
- Evidence:
  - `bigclaw-go/internal/reporting/reporting_test.go`
  - `bigclaw-go/internal/reportstudio/reportstudio_test.go`
  - `docs/go-mainline-cutover-issue-pack.md`

### `src/bigclaw/evaluation.py`

- Replacement kind: `go-evaluation-benchmark`
- Go replacements:
  - `bigclaw-go/internal/evaluation/evaluation.go`
- Evidence:
  - `bigclaw-go/internal/evaluation/evaluation_test.go`
  - `docs/go-mainline-cutover-issue-pack.md`

### `src/bigclaw/operations.py`

- Replacement kind: `go-operations-control-plane`
- Go replacements:
  - `bigclaw-go/internal/product/dashboard_run_contract.go`
  - `bigclaw-go/internal/contract/execution.go`
  - `bigclaw-go/internal/control/controller.go`
- Evidence:
  - `bigclaw-go/internal/product/dashboard_run_contract_test.go`
  - `bigclaw-go/internal/contract/execution_test.go`
  - `bigclaw-go/docs/reports/v2-phase1-operations-foundation-report.md`

### `src/bigclaw/risk.py`

- Replacement kind: `go-risk-policy-surface`
- Go replacements:
  - `bigclaw-go/internal/risk/risk.go`
  - `bigclaw-go/internal/risk/assessment.go`
  - `bigclaw-go/internal/policy/policy.go`
- Evidence:
  - `bigclaw-go/internal/risk/risk_test.go`
  - `bigclaw-go/internal/risk/assessment_test.go`
  - `docs/go-mainline-cutover-issue-pack.md`

### `src/bigclaw/governance.py`

- Replacement kind: `go-governance-freeze`
- Go replacements:
  - `bigclaw-go/internal/governance/freeze.go`
- Evidence:
  - `bigclaw-go/internal/governance/freeze_test.go`
  - `docs/go-mainline-cutover-issue-pack.md`

### `src/bigclaw/execution_contract.py`

- Replacement kind: `go-execution-contract`
- Go replacements:
  - `bigclaw-go/internal/contract/execution.go`
  - `bigclaw-go/internal/api/policy_runtime.go`
- Evidence:
  - `bigclaw-go/internal/contract/execution_test.go`
  - `docs/go-mainline-cutover-issue-pack.md`

### `src/bigclaw/audit_events.py`

- Replacement kind: `go-audit-spec-surface`
- Go replacements:
  - `bigclaw-go/internal/observability/audit.go`
  - `bigclaw-go/internal/observability/audit_spec.go`
- Evidence:
  - `bigclaw-go/internal/observability/audit_test.go`
  - `bigclaw-go/internal/observability/audit_spec_test.go`
  - `docs/go-mainline-cutover-issue-pack.md`

### `src/bigclaw/issue_archive.py`

- Replacement kind: `go-issue-archive-surface`
- Go replacements:
  - `bigclaw-go/internal/issuearchive/archive.go`
- Evidence:
  - `bigclaw-go/internal/issuearchive/archive_test.go`
  - `docs/go-mainline-cutover-issue-pack.md`

### `src/bigclaw/run_detail.py`

- Replacement kind: `go-run-detail-surface`
- Go replacements:
  - `bigclaw-go/internal/observability/task_run.go`
- Evidence:
  - `bigclaw-go/internal/observability/task_run_test.go`
  - `docs/go-mainline-cutover-issue-pack.md`

### `src/bigclaw/dashboard_run_contract.py`

- Replacement kind: `go-dashboard-contract`
- Go replacements:
  - `bigclaw-go/internal/product/dashboard_run_contract.go`
- Evidence:
  - `bigclaw-go/internal/product/dashboard_run_contract_test.go`
  - `docs/go-mainline-cutover-issue-pack.md`

### `src/bigclaw/saved_views.py`

- Replacement kind: `go-saved-views-catalog`
- Go replacements:
  - `bigclaw-go/internal/product/saved_views.go`
- Evidence:
  - `bigclaw-go/internal/product/saved_views_test.go`
  - `docs/go-mainline-cutover-issue-pack.md`

### `src/bigclaw/console_ia.py`

- Replacement kind: `go-console-ia-surface`
- Go replacements:
  - `bigclaw-go/internal/consoleia/consoleia.go`
  - `bigclaw-go/internal/product/console.go`
- Evidence:
  - `bigclaw-go/internal/consoleia/consoleia_test.go`
  - `bigclaw-go/internal/product/console_test.go`
  - `docs/go-mainline-cutover-issue-pack.md`

### `src/bigclaw/design_system.py`

- Replacement kind: `go-design-system-surface`
- Go replacements:
  - `bigclaw-go/internal/designsystem/designsystem.go`
  - `bigclaw-go/internal/product/console.go`
- Evidence:
  - `bigclaw-go/internal/designsystem/designsystem_test.go`
  - `bigclaw-go/internal/product/console_test.go`
  - `docs/go-mainline-cutover-issue-pack.md`

### `src/bigclaw/github_sync.py`

- Replacement kind: `go-github-sync`
- Go replacements:
  - `bigclaw-go/internal/githubsync/sync.go`
- Evidence:
  - `bigclaw-go/internal/githubsync/sync_test.go`
  - `docs/go-mainline-cutover-issue-pack.md`

### `src/bigclaw/workspace_bootstrap.py`

- Replacement kind: `go-workspace-bootstrap`
- Go replacements:
  - `bigclaw-go/internal/bootstrap/bootstrap.go`
  - `bigclaw-go/cmd/bigclawctl/main.go`
- Evidence:
  - `bigclaw-go/internal/bootstrap/bootstrap_test.go`
  - `docs/go-mainline-cutover-issue-pack.md`

### `src/bigclaw/workspace_bootstrap_cli.py`

- Replacement kind: `go-bootstrap-cli`
- Go replacements:
  - `bigclaw-go/internal/bootstrap/bootstrap.go`
  - `bigclaw-go/cmd/bigclawctl/main.go`
- Evidence:
  - `bigclaw-go/internal/bootstrap/bootstrap_test.go`
  - `bigclaw-go/cmd/bigclawctl/main_test.go`
  - `docs/go-mainline-cutover-issue-pack.md`

### `src/bigclaw/workspace_bootstrap_validation.py`

- Replacement kind: `go-bootstrap-validation`
- Go replacements:
  - `bigclaw-go/internal/bootstrap/bootstrap.go`
- Evidence:
  - `bigclaw-go/internal/bootstrap/bootstrap_test.go`
  - `docs/go-mainline-cutover-issue-pack.md`

### `src/bigclaw/parallel_refill.py`

- Replacement kind: `go-refill-queue`
- Go replacements:
  - `bigclaw-go/internal/refill/queue.go`
  - `bigclaw-go/internal/refill/local_store.go`
  - `bigclaw-go/cmd/bigclawctl/main.go`
- Evidence:
  - `bigclaw-go/internal/refill/queue_test.go`
  - `bigclaw-go/internal/refill/local_store_test.go`
  - `docs/go-mainline-cutover-issue-pack.md`

### `src/bigclaw/service.py`

- Replacement kind: `go-mainline-service`
- Go replacements:
  - `bigclaw-go/cmd/bigclawd/main.go`
  - `bigclaw-go/cmd/bigclawctl/main.go`
- Evidence:
  - `bigclaw-go/cmd/bigclawd/main_test.go`
  - `bigclaw-go/cmd/bigclawctl/main_test.go`
  - `docs/go-mainline-cutover-issue-pack.md`

### `src/bigclaw/__main__.py`

- Replacement kind: `go-mainline-entrypoint`
- Go replacements:
  - `bigclaw-go/cmd/bigclawd/main.go`
  - `bigclaw-go/cmd/bigclawctl/main.go`
- Evidence:
  - `bigclaw-go/cmd/bigclawd/main_test.go`
  - `bigclaw-go/cmd/bigclawctl/main_test.go`
  - `docs/go-mainline-cutover-issue-pack.md`

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find src/bigclaw -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the focused `src/bigclaw` sweep-G surface remained absent.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO101(RepositoryHasNoPythonFiles|ResidualSrcBigClawSweepGStaysAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesReplacementEvidence)$'`
  Result: `ok  	bigclaw-go/internal/regression	5.910s`
- `cd bigclaw-go && go test -count=1 ./internal/migration`
  Result: `?   	bigclaw-go/internal/migration	[no test files]`
