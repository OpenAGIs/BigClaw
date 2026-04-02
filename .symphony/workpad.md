# BIG-GO-1097

## Plan

1. Audit root README, workflow CI, and migration docs for Python references that still present active or default repo entrypoints.
2. Replace those references with the current Go-first commands already implemented in `bigclawctl` and the Go test/build flow.
3. Preserve historical inventory references where the docs are explicitly describing migrated Python assets or completed cutover evidence.
4. Run targeted validation on the affected Go CLI/workflow surfaces, then record exact commands and results.
5. Commit and push the scoped sweep to the workspace branch.

## Acceptance

- README and repo docs no longer present Python as the default root validation or operator path where Go equivalents already exist.
- `.github/workflows/ci.yml` validates the repo through Go-first commands instead of Python setup/test steps.
- Historical migration docs still keep Python file/path mentions only when they are describing completed migration inventory or evidence.
- The sweep provides clear Go replacement evidence for each removed default Python entrypoint reference.

## Validation

- `cd bigclaw-go && go test ./cmd/bigclawctl/...`
- `bash scripts/ops/bigclawctl dev-smoke`
- `bash scripts/ops/bigclawctl workspace validate --help`
- `make test`
- `make build`
- `rg -n "python|\\.py\\b|python3|pip install|pytest" README.md docs .github/workflows`

## Validation Results

- `cd bigclaw-go && go test ./cmd/bigclawctl/...` -> exit `0`; `ok bigclaw-go/cmd/bigclawctl 5.129s`
- `bash scripts/ops/bigclawctl dev-smoke` -> exit `0`; `smoke_ok local`
- `bash scripts/ops/bigclawctl workspace validate --help` -> exit `0`; printed `usage: bigclawctl workspace validate [flags]`
- `make test` -> exit `0`; `go test ./...` passed across `bigclaw-go`, including `internal/queue 31.959s`, `internal/regression 6.447s`, `internal/service 6.572s`, and `internal/workflow 4.045s`
- `make build` -> exit `0`; built `./cmd/bigclawd` and `./cmd/bigclawctl`
- `rg -n "python|\\.py\\b|python3|pip install|pytest" README.md docs .github/workflows` -> no workflow hits remain; remaining matches are limited to legacy-marked README notes and historical migration inventory docs such as `docs/go-mainline-cutover-issue-pack.md` and `docs/go-mainline-cutover-handoff.md`
