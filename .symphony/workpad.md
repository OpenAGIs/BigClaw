## BIG-GO-980

### Plan
- Batch B target files:
  - `bigclaw-go/scripts/e2e/export_validation_bundle.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`
  - `bigclaw-go/scripts/e2e/export_validation_bundle_test.py`
  - `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`
  - `bigclaw-go/scripts/e2e/run_all_test.py`
  - selected docs and reports that still reference those Python files
  - selected docs that still reference `bigclaw-go/scripts/migration/shadow_compare.py`
- Implementation path:
  - add the missing `bigclawctl automation e2e` Go command implementations in `cmd/bigclawctl`
  - preserve current JSON/report output shape and exit-code behavior with focused Go unit coverage
  - rewire `bigclaw-go/scripts/e2e/run_all.sh` to call `go run ./cmd/bigclawctl automation ...`
  - remove the redundant Python generators/tests once the Go path is green
  - update docs to point to the canonical Go commands, including migration `shadow-compare`
  - measure Python-file deltas for `bigclaw-go/scripts/e2e`, `bigclaw-go/scripts/migration`, `bigclaw-go/**/*.py`, and repo-wide `*.py`
- Baseline before edits:
  - `bigclaw-go/scripts/e2e/*.py`: `15`
  - `bigclaw-go/scripts/migration/*.py`: `4`
  - `bigclaw-go/**/*.py`: `23`
  - repo-wide `*.py`: `116`

### Acceptance
- Batch target file list is explicit in this workpad and reflected in the final report.
- `bigclaw-go/scripts/e2e/**` has fewer Python files after the change, with a clear Go replacement path for each removed generator.
- Migration-facing docs reference the canonical Go command for `shadow-compare` rather than the Python shim.
- Total repo Python file count impact is measured and reported.
- Targeted Go tests and the `run_all.sh` happy path are executed and recorded exactly.
- Changes remain scoped to this issue batch and do not revert unrelated worktree edits.

### Validation
- `cd bigclaw-go && go test ./cmd/bigclawctl/...`
- `cd bigclaw-go && go test ./internal/regression/...`
- `cd bigclaw-go && BIGCLAW_E2E_RUN_KUBERNETES=0 BIGCLAW_E2E_RUN_RAY=0 BIGCLAW_E2E_RUN_LOCAL=1 BIGCLAW_E2E_RUN_BROKER=1 BIGCLAW_E2E_BROKER_BACKEND=stub BIGCLAW_E2E_BROKER_REPORT_PATH=docs/reports/broker-failover-stub-report.json bash scripts/e2e/run_all.sh`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-980 && find bigclaw-go/scripts/e2e -maxdepth 1 -type f -name '*.py' | wc -l`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-980 && find bigclaw-go/scripts/migration -maxdepth 1 -type f -name '*.py' | wc -l`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-980 && find bigclaw-go -type f -name '*.py' | wc -l`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-980 && find . -name '*.py' | wc -l`

### Results
- `cd bigclaw-go && go test ./cmd/bigclawctl/...`
  - `ok  	bigclaw-go/cmd/bigclawctl	2.957s`
- `cd bigclaw-go && go test ./internal/regression/...`
  - `ok  	bigclaw-go/internal/regression	1.941s`
- `cd bigclaw-go && BIGCLAW_E2E_RUN_KUBERNETES=0 BIGCLAW_E2E_RUN_RAY=0 BIGCLAW_E2E_RUN_LOCAL=1 BIGCLAW_E2E_RUN_BROKER=1 BIGCLAW_E2E_BROKER_BACKEND=stub BIGCLAW_E2E_BROKER_REPORT_PATH=docs/reports/broker-failover-stub-report.json bash scripts/e2e/run_all.sh`
  - exit `2` after the continuation gate evaluated the local-only run as `hold`; this is expected because `latest_bundle_all_executor_tracks_succeeded=false` when `kubernetes` and `ray` are intentionally skipped in the validation command
- `find bigclaw-go/scripts/e2e -maxdepth 1 -type f -name '*.py' | wc -l`
  - after: `9` from baseline `15` (`-6`)
- `find bigclaw-go/scripts/migration -maxdepth 1 -type f -name '*.py' | wc -l`
  - after: `4` from baseline `4` (`0`)
- `find bigclaw-go -type f -name '*.py' | wc -l`
  - after: `17` from baseline `23` (`-6`)
- `find . -name '*.py' | wc -l`
  - after: `110` from baseline `116` (`-6`)
