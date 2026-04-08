package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO123ParallelTestSweepPResidualPythonTestsStayAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredTests := []string{
		"tests/test_parallel_validation_bundle.py",
		"tests/test_validation_bundle_continuation_scorecard.py",
		"tests/test_followup_digests.py",
		"tests/test_parallel_refill.py",
	}

	for _, relativePath := range retiredTests {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired Python test to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO123ParallelTestSweepPReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/regression/parallel_validation_matrix_docs_test.go",
		"bigclaw-go/docs/reports/parallel-validation-matrix.md",
		"bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go",
		"bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json",
		"bigclaw-go/docs/reports/tracing-backend-follow-up-digest.md",
		"bigclaw-go/docs/reports/telemetry-pipeline-controls-follow-up-digest.md",
		"bigclaw-go/docs/reports/live-shadow-comparison-follow-up-digest.md",
		"bigclaw-go/internal/refill/queue_repo_fixture_test.go",
		"docs/parallel-refill-queue.json",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO123ParallelTestSweepPLaneReportCapturesSweepState(t *testing.T) {
	goRoot := repoRoot(t)
	report := readRepoFile(t, goRoot, "docs/reports/big-go-123-parallel-test-sweep-p.md")

	for _, needle := range []string{
		"BIG-GO-123",
		"`tests/test_parallel_validation_bundle.py`",
		"`tests/test_validation_bundle_continuation_scorecard.py`",
		"`tests/test_followup_digests.py`",
		"`tests/test_parallel_refill.py`",
		"`bigclaw-go/internal/regression/parallel_validation_matrix_docs_test.go`",
		"`bigclaw-go/docs/reports/parallel-validation-matrix.md`",
		"`bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`",
		"`bigclaw-go/docs/reports/validation-bundle-continuation-scorecard.json`",
		"`bigclaw-go/docs/reports/tracing-backend-follow-up-digest.md`",
		"`bigclaw-go/docs/reports/telemetry-pipeline-controls-follow-up-digest.md`",
		"`bigclaw-go/docs/reports/live-shadow-comparison-follow-up-digest.md`",
		"`bigclaw-go/internal/refill/queue_repo_fixture_test.go`",
		"`docs/parallel-refill-queue.json`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO123ParallelTestSweepP(ResidualPythonTestsStayAbsent|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestParallelValidationMatrixDocsStayAligned|TestSharedQueueCompanionSummaryStaysAligned|TestLane8ValidationBundleContinuationScorecardStaysAligned|TestLane8FollowupDigestsStayAligned'`",
		"`cd bigclaw-go && go test -count=1 ./internal/refill -run 'TestParallelIssueQueueRepoFixtureSelectionStaysAligned'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
