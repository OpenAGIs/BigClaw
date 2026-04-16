package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1611RetiredPythonTestAssetsStayAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	if _, err := os.Stat(filepath.Join(rootRepo, "tests")); !os.IsNotExist(err) {
		t.Fatalf("expected retired root Python test tree to stay absent: %v", err)
	}

	retiredPaths := []string{
		"tests/conftest.py",
		"tests/test_parallel_refill.py",
		"tests/test_parallel_validation_bundle.py",
		"tests/test_queue.py",
		"tests/test_repo_board.py",
		"tests/test_repo_collaboration.py",
		"tests/test_repo_gateway.py",
		"tests/test_repo_governance.py",
		"tests/test_repo_links.py",
		"tests/test_repo_registry.py",
		"tests/test_repo_rollout.py",
		"tests/test_repo_triage.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected retired Python test path to stay absent: %s", relativePath)
		}
	}

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1611RefillAndReplacementCoverageStaysAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/internal/regression/big_go_253_zero_python_guard_test.go",
		"bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go",
		"bigclaw-go/internal/refill/local_store_test.go",
		"bigclaw-go/internal/refill/queue_markdown_test.go",
		"bigclaw-go/internal/refill/queue_repo_fixture_test.go",
		"bigclaw-go/internal/refill/queue_test.go",
		"bigclaw-go/internal/queue/sqlite_queue_test.go",
		"bigclaw-go/internal/repo/repo_surfaces_test.go",
		"bigclaw-go/internal/collaboration/thread_test.go",
		"bigclaw-go/internal/triage/repo_test.go",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1611SweepReportCapturesContract(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1611-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1611",
		"Repository-wide Python file count: `0`.",
		"`tests`: absent",
		"`tests/conftest.py`",
		"`tests/test_parallel_refill.py`",
		"`tests/test_repo_registry.py`",
		"`bigclaw-go/internal/regression/big_go_253_zero_python_guard_test.go`",
		"`bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`",
		"`bigclaw-go/internal/refill/local_store_test.go`",
		"`bigclaw-go/internal/refill/queue_markdown_test.go`",
		"`bigclaw-go/internal/refill/queue_repo_fixture_test.go`",
		"`bigclaw-go/internal/refill/queue_test.go`",
		"`bigclaw-go/internal/queue/sqlite_queue_test.go`",
		"`bigclaw-go/internal/repo/repo_surfaces_test.go`",
		"`bigclaw-go/internal/collaboration/thread_test.go`",
		"`bigclaw-go/internal/triage/repo_test.go`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`find tests bigclaw-go/internal/refill bigclaw-go/internal/regression -type f -name '*.py' 2>/dev/null | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1611(RetiredPythonTestAssetsStayAbsent|RefillAndReplacementCoverageStaysAvailable|SweepReportCapturesContract)$'`",
		"BIG-GO-1611 lands as a final guard-tightening pass because this checkout is already physically Python-free.",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("sweep report missing substring %q", needle)
		}
	}
}
