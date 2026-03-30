Issue: BIG-GO-1019

Plan
- Inspect the remaining `bigclaw-go/scripts/e2e/**` Python residue and pick a deterministic, repo-native migration slice with limited coupling.
- Replace `broker_failover_stub_matrix.py` with a Go-native `bigclawctl automation e2e broker-failover-stub-matrix` entrypoint while preserving canonical report, proof-summary, and artifact output surfaces.
- Update directly coupled docs/tests to call the Go entrypoint and remove the migrated Python script plus its Python test.
- Run targeted validation for the broker stub migration slice, capture exact commands and results, then commit and push the scoped change set.

Acceptance
- Changes stay scoped to `bigclaw-go/scripts/**` residual Python assets plus directly coupled tests/docs.
- `.py` file count under `bigclaw-go/scripts/e2e/**` is reduced for this tranche.
- The deterministic broker failover stub harness remains invokable through a Go-native CLI path.
- Final report states the impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

Validation
- `find bigclaw-go/scripts/e2e -maxdepth 1 \( -name '*.py' -o -name '*.go' -o -name '*.sh' \) | sort`
- `go test ./cmd/bigclawctl -run 'TestAutomationBrokerFailoverStubMatrixCopiesCanonicalArtifacts'`
- `go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix --help`
- `go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix --output docs/reports/broker-failover-stub-report.tmp.json --artifact-root docs/reports/broker-failover-stub-artifacts.tmp --checkpoint-fencing-summary-output docs/reports/broker-checkpoint-fencing-proof-summary.tmp.json --retention-boundary-summary-output docs/reports/broker-retention-boundary-proof-summary.tmp.json`
- `git diff --stat && git status --short`
