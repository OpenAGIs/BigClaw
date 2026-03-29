# BIG-GO-943 Validation Report

Date: 2026-03-29

## Scope

Issue: `BIG-GO-943`

Title: `Lane3 Core runtime/service/orchestration modules`

This lane closes the file-level migration audit for the frozen Python core
runtime surfaces under `src/bigclaw` and pins their checked-in Go ownership.

## Delivered

- Added the lane report:
  - `bigclaw-go/docs/reports/big-go-943-runtime-service-orchestration-lane.md`
- Updated issue coverage so `BIG-GO-943` is discoverable from the main Go
  migration evidence index:
  - `bigclaw-go/docs/reports/issue-coverage.md`
- Added regression coverage that locks the lane report and coverage cross-link:
  - `bigclaw-go/internal/regression/big_go_943_docs_test.go`
- Expanded the frozen compatibility compile-check shim set to include:
  - `src/bigclaw/runtime.py`
  - `src/bigclaw/service.py`
  - `src/bigclaw/scheduler.py`
  - `src/bigclaw/workflow.py`
  - `src/bigclaw/orchestration.py`
  - `src/bigclaw/queue.py`
  - `src/bigclaw/__main__.py`
  - `src/bigclaw/legacy_shim.py`
- Updated the CLI regression that exercises `bigclawctl legacy-python
  compile-check` so the expanded shim set stays covered.

## Lane File Mapping

| Legacy Python file | Go replacement |
| --- | --- |
| `src/bigclaw/runtime.py` | `bigclaw-go/internal/worker/runtime.go` |
| `src/bigclaw/service.py` | `bigclaw-go/cmd/bigclawd/main.go`, `bigclaw-go/internal/api/server.go` |
| `src/bigclaw/scheduler.py` | `bigclaw-go/internal/scheduler/scheduler.go` |
| `src/bigclaw/workflow.py` | `bigclaw-go/internal/workflow/engine.go`, `bigclaw-go/internal/workflow/closeout.go` |
| `src/bigclaw/orchestration.py` | `bigclaw-go/internal/workflow/orchestration.go` |
| `src/bigclaw/queue.py` | `bigclaw-go/internal/queue/queue.go`, `bigclaw-go/internal/queue/file_queue.go`, `bigclaw-go/internal/queue/sqlite_queue.go`, `bigclaw-go/internal/queue/memory_queue.go` |
| `src/bigclaw/__main__.py` | `bigclaw-go/cmd/bigclawd/main.go`, `bigclaw-go/cmd/bigclawctl/main.go` |
| `src/bigclaw/legacy_shim.py` | `bigclaw-go/internal/legacyshim/compilecheck.go` |

Delete/defer conditions remain documented in
`bigclaw-go/docs/reports/big-go-943-runtime-service-orchestration-lane.md`.

## Validation

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-943/bigclaw-go && go test ./internal/legacyshim ./internal/regression
```

Result:

```text
ok  	bigclaw-go/internal/legacyshim	0.762s
ok  	bigclaw-go/internal/regression	0.776s
```

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-943/bigclaw-go && go test ./cmd/bigclawctl
```

Result:

```text
ok  	bigclaw-go/cmd/bigclawctl	2.560s
```

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-943/bigclaw-go && go test ./internal/scheduler ./internal/workflow ./internal/queue ./internal/worker
```

Result:

```text
ok  	bigclaw-go/internal/scheduler	1.829s
ok  	bigclaw-go/internal/workflow	0.761s
ok  	bigclaw-go/internal/queue	27.166s
ok  	bigclaw-go/internal/worker	2.962s
```

## Git

- Branch: `big-go-943-runtime-service-orchestration`
- Commit: `5b493c1fd6d72d4a692611184630d5af667eeb29`
- Push: `git push -u origin big-go-943-runtime-service-orchestration` -> success

## Residual Risks

- The Python modules in this lane are still imported by legacy tests and package
  re-exports, so immediate deletion would be a breaking change.
- The lane establishes frozen Go ownership and delete conditions, but it does
  not itself remove the remaining Python tests that still anchor these shims.
- Full removal depends on follow-up lanes that migrate or delete the remaining
  Python test and package surfaces.
