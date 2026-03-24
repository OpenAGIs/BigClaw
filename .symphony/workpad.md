# BIGCLAW-184 Workpad

## Plan
- Inspect existing distributed diagnostics and evidence-bundle surfaces in `bigclaw-go/internal/api` to match current API and report conventions.
- Add a repo-native parallel diagnostics evidence-bundle index payload plus a search payload that can filter bundle artifacts by query, status, lane, and evidence path matches.
- Expose the new payloads from the active Go API surface and cover them with targeted API tests.
- Run focused Go tests, record the exact commands and results, then commit and push the branch for `BIGCLAW-184`.

## Acceptance
- The Go API exposes a deterministic parallel diagnostics evidence-bundle index surface sourced from checked-in repo artifacts.
- The Go API exposes a deterministic search surface for the same evidence bundle corpus with stable filtering semantics.
- The implementation stays scoped to the distributed diagnostics / evidence bundle surface and includes targeted regression coverage.
- Validation commands and results are captured before closeout.

## Validation
- `cd bigclaw-go && go test ./internal/api`

## Results
- `cd bigclaw-go && go test ./internal/api -run 'TestV2DistributedEvidenceBundlesIndex|TestV2DistributedEvidenceBundleSearch|TestV2DistributedEvidenceBundleSearchRejectsInvalidLimit'` -> `ok  	bigclaw-go/internal/api	0.350s`
- `cd bigclaw-go && go test ./internal/api` -> `ok  	bigclaw-go/internal/api	3.363s`
