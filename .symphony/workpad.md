# BIG-GO-1575 Workpad

## Plan

1. Reconfirm the repository-wide physical Python baseline and verify that all
   issue candidate files are already absent in the current checkout.
2. Record the exact candidate-file coverage, zero-Python inventory, and current
   Go/native replacement paths in repo-native lane artifacts.
3. Add focused regression coverage so the candidate file set and repository-wide
   `.py` inventory stay at zero.
4. Run targeted validation, record exact commands and results, then commit and
   push `BIG-GO-1575`.

## Acceptance

- The lane records the exact candidate Python file list covered by this sweep.
- The lane confirms the candidate files are already removed in the current
  baseline rather than leaving undocumented residual shims.
- The lane names the active Go/native replacement paths for the removed
  surfaces.
- The lane records exact validation commands and outcomes.
- The change is committed and pushed on `BIG-GO-1575`.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `for f in src/bigclaw/connectors.py src/bigclaw/governance.py src/bigclaw/planning.py src/bigclaw/reports.py src/bigclaw/workflow.py tests/test_cross_process_coordination_surface.py tests/test_governance.py tests/test_parallel_refill.py tests/test_repo_registry.py tests/test_service.py scripts/ops/bigclaw_refill_queue.py bigclaw-go/scripts/e2e/cross_process_coordination_surface.py bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py; do [ -e "$f" ] && echo "EXISTS $f" || echo "MISSING $f"; done`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1575(RepositoryHasNoPythonFiles|CandidatePathsRemainAbsent|ReplacementPathsRemainAvailable|LaneReportCapturesCoverage)$'`
- `cd bigclaw-go && go test -count=1 ./cmd/bigclawctl ./internal/intake ./internal/governance ./internal/planning ./internal/reporting ./internal/workflow ./internal/repo ./internal/refill ./internal/service`

## GitHub

- Branch to push: `origin/BIG-GO-1575`
- Compare view after push: `https://github.com/OpenAGIs/BigClaw/compare/main...BIG-GO-1575?expand=1`
