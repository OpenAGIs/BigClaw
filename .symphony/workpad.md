# BIG-GO-1072 Workpad

## Plan
- Locate active docs, CI, and wrapper surfaces that still treat Python refill/build outputs as part of the default path.
- Delete the remaining Python refill shim and remove dead helper coverage tied to that shim.
- Update the active README and migration doc so refill points directly at `bash scripts/ops/bigclawctl refill`.
- Remove CI's Python coverage artifact lane and make the default validation path Go-first with direct Go entrypoint checks.
- Run targeted validation, capture exact commands and results, then commit and push the branch.

## Acceptance
- `scripts/ops/bigclaw_refill_queue.py` is deleted.
- The refill default path is Go-only in active docs and CI.
- CI no longer installs Python or uploads Python coverage artifacts as the default build/test output.
- Legacy helper code only covers the remaining workspace compatibility shims.
- `.py` file count decreases from the branch baseline.

## Validation
- Capture pre/post `.py` file counts with `rg --files -g '*.py' | wc -l`.
- Run `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl`.
- Run `bash scripts/ops/bigclawctl refill --help`.
- Run `bash scripts/ops/bigclawctl workspace validate --help`.
- Run `make build`.
