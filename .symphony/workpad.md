# BIG-GO-1018

## Plan
- Migrate the next scoped residual `tests/**` tranche for the task-run observability surface.
- Port the small Python observability/report helpers into a dedicated Go package that covers task runs, ledger round-trip, repo-sync audit rendering, detail-page rendering, and collaboration extraction.
- Remove `tests/test_observability.py` after validating the new Go coverage.
- Keep scope limited to the repository-facing observability slice and avoid broad report-suite or UI-review migration.
- Run targeted Go tests for `bigclaw-go/internal/observabilitysurface`, capture exact commands and results, then commit and push the branch.

## Acceptance
- Changes stay scoped to this issue's residual `tests/**` tranche.
- The selected Python test behaviors are covered by Go tests against repository code, not tracker metadata.
- The number of repository `.py` files decreases.
- Final report includes impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

## Validation
- `go test ./internal/observabilitysurface`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `git status --short`

## Results
- Current tranche: `cd bigclaw-go && go test ./internal/observabilitysurface` -> `ok  	bigclaw-go/internal/observabilitysurface	0.952s`
- Current tranche: `find . -name '*.py' | wc -l` -> `75`
- Current tranche: `find . -name '*.go' | wc -l` -> `281`
- Current tranche: `git status --short` -> `M .symphony/workpad.md`; `D tests/test_observability.py`; `?? bigclaw-go/internal/observabilitysurface/`
- Current tranche impact: `py files` decreased from `76` to `75`; `go files` increased from `279` to `281`; `pyproject.toml` absent and unchanged; `setup.py` absent and unchanged
- Current tranche status: `.symphony/workpad.md` modified; `bigclaw-go/internal/observabilitysurface/observabilitysurface.go` and `bigclaw-go/internal/observabilitysurface/observabilitysurface_test.go` added; `tests/test_observability.py` deleted
- Current tranche: `cd bigclaw-go && go test ./internal/auditsurface` -> `ok  	bigclaw-go/internal/auditsurface	0.792s`
- Current tranche: `find . -name '*.py' | wc -l` -> `76`
- Current tranche: `find . -name '*.go' | wc -l` -> `279`
- Current tranche: `git status --short` -> `M .symphony/workpad.md`; `D tests/test_audit_events.py`; `?? bigclaw-go/internal/auditsurface/`
- Current tranche impact: `py files` decreased from `77` to `76`; `go files` increased from `277` to `279`; `pyproject.toml` absent and unchanged; `setup.py` absent and unchanged
- Current tranche status: `.symphony/workpad.md` modified; `bigclaw-go/internal/auditsurface/auditsurface.go` and `bigclaw-go/internal/auditsurface/auditsurface_test.go` added; `tests/test_audit_events.py` deleted
- Current tranche: `cd bigclaw-go && go test ./internal/evaluation` -> `ok  	bigclaw-go/internal/evaluation	1.099s`
- Current tranche: `find . -name '*.py' | wc -l` -> `77`
- Current tranche: `find . -name '*.go' | wc -l` -> `277`
- Current tranche: `git status --short` -> `M .symphony/workpad.md`; `D tests/test_evaluation.py`; `?? bigclaw-go/internal/evaluation/`
- Current tranche impact: `py files` decreased from `78` to `77`; `go files` increased from `275` to `277`; `pyproject.toml` absent and unchanged; `setup.py` absent and unchanged
- Current tranche status: `.symphony/workpad.md` modified; `bigclaw-go/internal/evaluation/evaluation.go` and `bigclaw-go/internal/evaluation/evaluation_test.go` added; `tests/test_evaluation.py` deleted
- Previous completed tranche: `cd bigclaw-go && go test ./internal/rollout` -> `ok  	bigclaw-go/internal/rollout	1.447s`
- Previous completed tranche: `find . -name '*.py' | wc -l` -> `78`
- Previous completed tranche: `find . -name '*.go' | wc -l` -> `275`
- Previous completed tranche status: `.symphony/workpad.md` modified; `bigclaw-go/internal/rollout/rollout.go` and `bigclaw-go/internal/rollout/rollout_test.go` added; `tests/test_repo_rollout.py` deleted
- Previous completed tranche: `cd bigclaw-go && go test ./internal/runbus` -> `ok  	bigclaw-go/internal/runbus	0.446s`
- Previous completed tranche: `find . -name '*.py' | wc -l` -> `79`
- Previous completed tranche: `find . -name '*.go' | wc -l` -> `273`
- Previous completed tranche status: `.symphony/workpad.md` modified; `bigclaw-go/internal/runbus/runbus.go` and `bigclaw-go/internal/runbus/runbus_test.go` added; `tests/test_event_bus.py` deleted
- Previous completed tranche: `cd bigclaw-go && go test ./internal/memory` -> `ok  	bigclaw-go/internal/memory	1.805s`
- Previous completed tranche: `find . -name '*.py' | wc -l` -> `80`
- Previous completed tranche: `find . -name '*.go' | wc -l` -> `271`
- Previous completed tranche status: `.symphony/workpad.md` modified; `bigclaw-go/internal/memory/store.go` and `bigclaw-go/internal/memory/store_test.go` added; `tests/test_memory.py` deleted
- Previous completed tranche: `cd bigclaw-go && go test ./internal/repo` -> `ok  	bigclaw-go/internal/repo	0.827s`
- Previous completed tranche: `find . -name '*.py' | wc -l` -> `81`
- Previous completed tranche: `find . -name '*.go' | wc -l` -> `269`
- Previous completed tranche status: `.symphony/workpad.md`, `bigclaw-go/internal/repo/board.go`, `bigclaw-go/internal/repo/repo_surfaces_test.go` modified; `bigclaw-go/internal/repo/collaboration.go` added; `tests/test_repo_collaboration.py` deleted
- Previous completed tranche: `cd bigclaw-go && go test ./internal/workflow` -> `ok  	bigclaw-go/internal/workflow	1.580s`
- Previous completed tranche: `find . -name '*.py' | wc -l` -> `82`
- Previous completed tranche: `find . -name '*.go' | wc -l` -> `268`
- Previous completed tranche status: `.symphony/workpad.md`, `bigclaw-go/internal/workflow/definition.go`, `bigclaw-go/internal/workflow/definition_test.go` modified; `tests/test_dsl.py` deleted
