# BIG-GO-1038 Workpad

## Plan

1. Delete the next Go-covered Python tests:
   `tests/test_models.py`, `tests/test_queue.py`, and `tests/test_scheduler.py`.
2. Extend `bigclaw-go/internal/queue/file_queue.go` and `bigclaw-go/internal/queue/file_queue_test.go`
   so the Go file-backed queue covers the remaining Python-only persistence behaviors, including
   parent-directory creation, payload preservation, and legacy list storage compatibility.
3. Reuse the existing Go scheduler, workflow, billing, triage, risk, and domain test coverage as the
   replacement surface for the deleted Python model and scheduler tests.
4. Run targeted Go validation for `./internal/queue`, `./internal/scheduler`, `./internal/domain`,
   `./internal/workflow`, `./internal/triage`, `./internal/risk`, and `./internal/billing`, plus
   repo-level file-count checks.
5. Commit the scoped migration changes and push the branch to the remote.

## Acceptance

- The number of Python files under `tests/` decreases in this issue scope.
- Any deleted Python test has a checked-in Go replacement test in `bigclaw-go/`.
- No new Python tests are introduced.
- `pyproject.toml` and `setup.py` remain absent.
- The final change can name the deleted Python files and the added or expanded Go test files.

## Validation

- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
- `cd bigclaw-go && go test ./internal/queue ./internal/scheduler ./internal/domain ./internal/workflow ./internal/triage ./internal/risk ./internal/billing`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
- `git status --short`

## Validation Results

- `cd bigclaw-go && go test ./internal/queue ./internal/scheduler ./internal/domain ./internal/workflow ./internal/triage ./internal/risk ./internal/billing`
  - `ok  	bigclaw-go/internal/queue	27.429s`
  - `ok  	bigclaw-go/internal/scheduler	0.612s`
  - `ok  	bigclaw-go/internal/domain	1.084s`
  - `ok  	bigclaw-go/internal/workflow	(cached)`
  - `ok  	bigclaw-go/internal/triage	1.483s`
  - `ok  	bigclaw-go/internal/risk	(cached)`
  - `ok  	bigclaw-go/internal/billing	1.882s`
- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
  - `20`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
  - no output
- `cd bigclaw-go && go test ./internal/repo ./internal/risk ./internal/workflow`
  - `ok  	bigclaw-go/internal/repo	1.151s`
  - `ok  	bigclaw-go/internal/risk	2.022s`
  - `ok  	bigclaw-go/internal/workflow	1.574s`
- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
  - `23`
- `cd bigclaw-go && go test ./internal/bootstrap`
  - `ok  	bigclaw-go/internal/bootstrap	4.862s`
- `cd bigclaw-go && go test ./internal/product`
  - `ok  	bigclaw-go/internal/product	2.728s`
- `cd bigclaw-go && go test ./internal/contract`
  - `ok  	bigclaw-go/internal/contract	1.370s`
- `cd bigclaw-go && go test ./internal/githubsync`
  - `ok  	bigclaw-go/internal/githubsync	3.702s`
- `cd bigclaw-go && go test ./internal/governance`
  - `ok  	bigclaw-go/internal/governance	0.534s`
- `cd bigclaw-go && go test ./internal/observability`
  - `ok  	bigclaw-go/internal/observability	1.891s`
- `PYTHONPATH=src python3 -m pytest tests/test_planning.py -q`
  - `14 passed in 0.18s`
- `find tests -maxdepth 1 -name '*.py' | sort | wc -l`
  - `31`
- `find . \\( -name pyproject.toml -o -name setup.py \\) -print | sort`
  - no output
