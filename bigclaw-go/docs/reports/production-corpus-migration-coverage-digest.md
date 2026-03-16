# Production Corpus Migration Coverage Digest

## Scope

This digest consolidates the remaining production-corpus migration coverage caveats for `OPE-268` / `BIG-PAR-079`.

## Current Repo-Backed Evidence

- `docs/reports/migration-readiness-report.md` captures the current shadow comparison readiness and remaining migration gaps.
- `docs/reports/shadow-matrix-report.json` captures fixture-backed parity evidence across multiple sample tasks plus the new `corpus_coverage` scorecard.
- `docs/reports/shadow-compare-report.json` captures one shared-`trace_id` shadow comparison sample.
- `docs/migration-shadow.md` documents the current single-run, matrix, and corpus-manifest workflow.
- `examples/shadow-corpus-manifest.json` provides a repo-native anonymized replay-pack/coverage-manifest example.
- `docs/reports/issue-coverage.md` records where migration evidence exists today and where the remaining caveats live.

## Reviewer Digest

- The migration matrix now accepts an anonymized corpus manifest and emits a coverage scorecard that maps fixture task shapes to corpus slices.
- Fixture-backed evidence only remains the default checked-in execution path, so the scorecard should be read as a readiness overlay rather than standalone cutover proof.
- Fixture-backed shadow evidence is useful for protocol and state-machine parity, but it does not prove readiness across real customer/task distributions.
- There is still no real production issue/task corpus coverage checked into the repo; operators must supply anonymized manifests for tenant skew, large issue sets, or historical long-tail task mixes.
- Production-corpus readiness therefore remains a documentation gap between local sample parity and honest cutover confidence.

## Current Blockers

- No production issue/task export is wired into the current shadow matrix flow.
- The repo-native manifest example is anonymized and illustrative; it is not a live or automatically refreshed production export.
- Real tenant-weighted drift still depends on operators supplying updated anonymized manifests and reviewing uncovered slices over time.
- No ongoing report ties live or archived production corpus drift back into the migration evidence bundle.

## Lightweight Consistency Check

- Keep this digest aligned with `docs/reports/migration-readiness-report.md`, `docs/migration-shadow.md`, `examples/shadow-corpus-manifest.json`, and the shadow JSON reports.
- Repeat the `fixture-backed evidence only` and `no real production issue/task corpus coverage` caveats anywhere migration coverage is summarized.
- When production-corpus evidence lands, update this digest, `docs/reports/review-readiness.md`, and `docs/reports/issue-coverage.md` together.
