# BigClaw Go Domain and Intake Parity Matrix

This matrix captures the current `BIG-GOM-301` field and ownership split while the repo continues moving from the legacy Python surface to the Go-only mainline.

## Current status

- `BigClaw` is not yet `100%` Go.
- `BigClaw/bigclaw-go` is the only implementation mainline for new work.
- The Python domain/intake/DSL layer is now partially ported into dedicated Go ownership areas instead of a monolithic `models.go`.

## Python to Go ownership

### `src/bigclaw/models.py`

- `Task` -> `bigclaw-go/internal/domain/task.go`
  - canonical Go runtime task shape
  - now accepts legacy `task_id` input for migration compatibility while preserving canonical Go `id`
  - now accepts legacy Python `budget` payloads and round-trips them alongside canonical Go `budget_cents`
  - now normalizes legacy Python task states (`Todo`, `In Progress`, `Done`, `Blocked`, `Failed`) into canonical Go runtime states on ingest
  - task JSON output now preserves the Python `to_dict()` core field set, including default state/risk/budget values and empty list fields for labels, required tools, acceptance criteria, and validation plan
  - now carries Python budget override fields (`budget_override_actor`, `budget_override_reason`, `budget_override_amount`) on the canonical task contract
  - execution lifecycle remains `queued/running/succeeded/...`
- `Priority` -> `bigclaw-go/internal/domain/priority.go`
- `RiskLevel` -> `bigclaw-go/internal/domain/task.go`
- `RiskSignal`, `RiskAssessment` -> `bigclaw-go/internal/risk/assessment.go`
  - risk JSON encode/decode now preserves the Python `to_dict()` / `from_dict()` field set, including default low-level output and empty signal/mitigation metadata collections
- `TriageLabel`, `TriageRecord` -> `bigclaw-go/internal/triage/record.go`
  - triage JSON encode/decode now preserves the Python `to_dict()` / `from_dict()` field set, including default `open` status, default `default` queue, label confidence/source defaults, and empty labels/actions collections
- `BillingInterval`, `BillingRate`, `UsageRecord`, `BillingSummary` -> `bigclaw-go/internal/billing/statement.go`
  - billing usage metadata now preserves Python-style `Dict[str, Any]` payloads instead of narrowing to string-only values
  - billing JSON output now preserves the Python `to_dict()` field set for usage records and summaries, including empty metadata/rates/usage collections and default numeric totals
- `FlowTrigger`, `FlowRunStatus`, `FlowStepStatus`, `FlowTemplate*`, `FlowRun*` -> `bigclaw-go/internal/workflow/model.go`
  - workflow JSON decode now defaults missing Python list/map fields to empty collections so `from_dict` behavior stays aligned for steps, tags, outputs, approvals, and step metadata
  - workflow JSON output now preserves the Python `to_dict()` field set for templates and runs, including default trigger/status values and empty collection fields

### Former `src/bigclaw/connectors.py`

- `SourceIssue` -> `bigclaw-go/internal/intake/types.go`
- `Connector` protocol -> `bigclaw-go/internal/intake/connector.go`
- `GitHubConnector`, `LinearConnector`, `JiraConnector` -> `bigclaw-go/internal/intake/connector.go`

### Former `src/bigclaw/mapping.py`

- `map_priority` -> `bigclaw-go/internal/intake/mapping.go`
- `map_state` -> `bigclaw-go/internal/intake/mapping.go`
- `map_source_issue_to_task` -> `bigclaw-go/internal/intake/mapping.go`

### Former `src/bigclaw/dsl.py`

- `WorkflowStep` -> `bigclaw-go/internal/workflow/definition.go`
- `WorkflowDefinition` -> `bigclaw-go/internal/workflow/definition.go`
- `from_json` -> `bigclaw-go/internal/workflow/definition.go`
- `render_report_path` / `render_journal_path` -> `bigclaw-go/internal/workflow/definition.go`
  - workflow-definition JSON decode now defaults missing Python list/map fields to empty collections so `from_dict` behavior stays aligned for step metadata, steps, validation evidence, and approvals
  - workflow-definition JSON output now preserves the Python `to_dict()` field set, including empty steps, validation evidence, approvals, and blank template paths

## Key design decisions

- Python source-board status is intentionally separate from Go runtime execution state.
  - Source status lives in `bigclaw-go/internal/intake/status.go`.
  - Runtime task state remains in `bigclaw-go/internal/domain/task.go`.
- Python `models.py` is being split by responsibility into existing Go package families rather than copied into a single compatibility file.
- New API surfaces for the migrated intake/definition layer live under:
  - `/v2/intake/connectors/...`
  - `/v2/intake/issues/map`
  - `/v2/workflows/definitions/render`
- The active API and contract mainline now uses `internal/intake` and `internal/workflow` as the sole Go owners for this migration slice.

## Remaining gaps

- Some legacy Python modules under `src/bigclaw` still exist and are not yet fully retired, but the intake connector / mapping / DSL slice has been deleted after Go takeover.
- Python tests under `BigClaw/tests` still remain as migration references.
- Follow-on `BIG-GOM-302` migration work has started in Go with:
  - `bigclaw-go/internal/governance/freeze.go`
  - `bigclaw-go/internal/contract/execution.go`
  - `bigclaw-go/internal/observability/audit_spec.go`
  but the broader Python runtime/reporting/orchestration surface still remains.
- The workflow execution engine and broader runtime/reporting surfaces still need follow-on Go migrations from the `BIG-GOM-302` to `BIG-GOM-306` slices.
