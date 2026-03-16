# Production Corpus Migration Coverage Digest

## Scope

This digest consolidates the remaining production-corpus migration coverage caveats for `OPE-268` / `BIG-PAR-079`.

## Current Repo-Backed Evidence

- `docs/reports/migration-readiness-report.md` captures the current shadow comparison readiness and remaining migration gaps.
- `docs/reports/shadow-matrix-report.json` captures fixture-backed parity evidence across multiple sample tasks.
- `docs/reports/shadow-compare-report.json` captures one shared-`trace_id` shadow comparison sample.
- `docs/migration-shadow.md` documents the current single-run and matrix comparison workflow.
- `docs/reports/issue-coverage.md` records where migration evidence exists today and where the remaining caveats live.

## Reviewer Digest

- The current migration matrix is still built from curated sample tasks rather than a real production issue/task corpus.
- Fixture-backed shadow evidence is useful for protocol and state-machine parity, but it does not prove readiness across real customer/task distributions.
- There is no repo-native evidence yet for corpus slices such as tenant skew, large issue sets, or historical long-tail task mixes.
- Production-corpus readiness therefore remains a documentation gap between local sample parity and honest cutover confidence.

## Current Blockers

- No production issue/task export is wired into the current shadow matrix flow.
- No anonymized corpus replay pack exists for migration review.
- No coverage scorecard maps current fixture tasks to real production volume or task-shape distribution.
- No ongoing report ties live or archived production corpus drift back into the migration evidence bundle.

## Lightweight Consistency Check

- Keep this digest aligned with `docs/reports/migration-readiness-report.md`, `docs/migration-shadow.md`, and the shadow JSON reports.
- Repeat the `fixture-backed evidence only` and `no real production issue/task corpus coverage` caveats anywhere migration coverage is summarized.
- When production-corpus evidence lands, update this digest, `docs/reports/review-readiness.md`, and `docs/reports/issue-coverage.md` together.
