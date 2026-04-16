package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1605RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1605AssignedPythonAssetsStayAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"src/bigclaw/observability.py",
		"src/bigclaw/reports.py",
		"src/bigclaw/evaluation.py",
		"src/bigclaw/operations.py",
		"tests/test_observability.py",
		"tests/test_reports.py",
		"tests/test_evaluation.py",
		"tests/test_operations.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected assigned Python asset to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO1605GoReplacementPathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/cmd/bigclawctl/reporting_commands.go",
		"bigclaw-go/cmd/bigclawctl/reporting_commands_test.go",
		"bigclaw-go/internal/api/expansion.go",
		"bigclaw-go/internal/api/expansion_test.go",
		"bigclaw-go/internal/observability/recorder.go",
		"bigclaw-go/internal/observability/audit.go",
		"bigclaw-go/internal/reporting/reporting.go",
		"bigclaw-go/internal/reporting/reporting_test.go",
		"bigclaw-go/internal/evaluation/evaluation.go",
		"bigclaw-go/internal/evaluation/evaluation_test.go",
		"bigclaw-go/internal/regression/regression.go",
		"bigclaw-go/docs/reports/go-control-plane-observability-report.md",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go replacement path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1605LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1605-go-only-sweep-refill.md")

	for _, needle := range []string{
		"BIG-GO-1605",
		"Repository-wide Python file count before lane changes: `0`.",
		"Repository-wide Python file count after lane changes: `0`.",
		"`src/bigclaw/observability.py` -> `bigclaw-go/internal/observability/recorder.go`",
		"`src/bigclaw/reports.py` -> `bigclaw-go/internal/reporting/reporting.go`",
		"`src/bigclaw/evaluation.py` -> `bigclaw-go/internal/evaluation/evaluation.go`",
		"`src/bigclaw/operations.py` -> `bigclaw-go/cmd/bigclawctl/reporting_commands.go`",
		"`tests/test_observability.py` -> `bigclaw-go/internal/observability/recorder_test.go`",
		"`tests/test_reports.py` -> `bigclaw-go/internal/reporting/reporting_test.go`",
		"`tests/test_evaluation.py` -> `bigclaw-go/internal/evaluation/evaluation_test.go`",
		"`tests/test_operations.py` -> `bigclaw-go/cmd/bigclawctl/reporting_commands_test.go`",
		"`bigclawctl reporting weekly`",
		"`GET /v2/reports/weekly`",
		"`GET /v2/reports/weekly/export`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`for path in src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/evaluation.py src/bigclaw/operations.py tests/test_observability.py tests/test_reports.py tests/test_evaluation.py tests/test_operations.py; do test ! -e \"$path\" && printf 'absent %s\\n' \"$path\"; done`",
		"`cd bigclaw-go && go test -count=1 ./cmd/bigclawctl ./internal/reporting ./internal/api`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run TestBIGGO1605`",
		"Residual risk: this checkout already started with zero physical Python files, so BIG-GO-1605 hardens that Go-only reporting/observability baseline rather than lowering the numeric file count further.",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
