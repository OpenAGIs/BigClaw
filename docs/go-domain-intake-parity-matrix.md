# BigClaw Go Domain and Intake Parity Matrix

This matrix captures the current `BIG-GOM-301` field and ownership split while the repo continues moving from the legacy Python surface to the Go-only mainline.

## Current status

- `BigClaw` now retains only one Python compatibility shim under `src/bigclaw/legacy_shim.py`.
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

### `src/bigclaw/connectors.py`

- `SourceIssue` -> `bigclaw-go/internal/intake/types.go`
- `Connector` protocol -> `bigclaw-go/internal/intake/connector.go`
- `GitHubConnector`, `LinearConnector`, `JiraConnector` -> `bigclaw-go/internal/intake/connector.go`

### `src/bigclaw/mapping.py`

- `map_priority` -> `bigclaw-go/internal/intake/mapping.go`
- `map_state` -> `bigclaw-go/internal/intake/mapping.go`
- `map_source_issue_to_task` -> `bigclaw-go/internal/intake/mapping.go`

### `src/bigclaw/dsl.py`

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

- The legacy domain/intake source modules covered here have been physically removed from `src/bigclaw`; only `legacy_shim.py` remains for operator-wrapper compatibility.
- Python tests under `BigClaw/tests` have already been retired in earlier sweeps.
- Repo tooling still carries a small number of Python wrapper scripts outside `src/bigclaw`; those wrappers remain migration compatibility surfaces.
- Follow-up work now lives in Go-native hardening lanes rather than Python ownership migration for the domain/intake surfaces captured in this matrix.
