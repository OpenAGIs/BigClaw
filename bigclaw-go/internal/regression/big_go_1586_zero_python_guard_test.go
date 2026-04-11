package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1586RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1586BenchmarkBucketStaysPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, filepath.Join(rootRepo, "bigclaw-go", "scripts", "benchmark"))
	if len(pythonFiles) != 0 {
		t.Fatalf("expected benchmark bucket to remain Python-free, found %v", pythonFiles)
	}
}

func TestBIGGO1586RetiredBenchmarkPythonHelpersRemainAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"bigclaw-go/scripts/benchmark/capacity_certification.py",
		"bigclaw-go/scripts/benchmark/capacity_certification_test.py",
		"bigclaw-go/scripts/benchmark/run_matrix.py",
		"bigclaw-go/scripts/benchmark/soak_local.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected retired benchmark Python helper to remain absent: %s", relativePath)
		}
	}
}

func TestBIGGO1586BenchmarkReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/scripts/benchmark/run_suite.sh",
		"bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go",
		"bigclaw-go/cmd/bigclawctl/automation_commands_test.go",
		"bigclaw-go/internal/queue/benchmark_test.go",
		"bigclaw-go/internal/scheduler/benchmark_test.go",
		"bigclaw-go/docs/reports/benchmark-report.md",
		"bigclaw-go/docs/reports/benchmark-matrix-report.json",
		"bigclaw-go/docs/reports/long-duration-soak-report.md",
		"bigclaw-go/docs/reports/soak-local-50x8.json",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected benchmark replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1586LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1586-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1586",
		"Repository-wide Python file count: `0`.",
		"`bigclaw-go/scripts/benchmark`: `0` Python files",
		"`bigclaw-go/scripts/benchmark/capacity_certification.py`",
		"`bigclaw-go/scripts/benchmark/capacity_certification_test.py`",
		"`bigclaw-go/scripts/benchmark/run_matrix.py`",
		"`bigclaw-go/scripts/benchmark/soak_local.py`",
		"`bigclaw-go/scripts/benchmark/run_suite.sh`",
		"`bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go`",
		"`bigclaw-go/internal/queue/benchmark_test.go`",
		"`bigclaw-go/internal/scheduler/benchmark_test.go`",
		"`bigclaw-go/docs/reports/benchmark-report.md`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`find bigclaw-go/scripts/benchmark -type f -name '*.py' -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1586(RepositoryHasNoPythonFiles|BenchmarkBucketStaysPythonFree|RetiredBenchmarkPythonHelpersRemainAbsent|BenchmarkReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
