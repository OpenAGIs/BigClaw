package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1606RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1606DeletedCompatibilityArtifactsStayAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	deletedArtifacts := []string{
		"bigclaw-go/internal/migration/legacy_model_runtime_modules.go",
		"bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json",
	}

	for _, relativePath := range deletedArtifacts {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected deleted compatibility artifact to stay absent: %s", relativePath)
		}
	}
}

func TestBIGGO1606GoMainlineEntryPathsExist(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	mainlinePaths := []string{
		"bigclaw-go/internal/worker/runtime.go",
		"bigclaw-go/internal/worker/runtime_runonce.go",
		"bigclaw-go/internal/service/server.go",
		"bigclaw-go/internal/scheduler/scheduler.go",
		"bigclaw-go/internal/workflow/engine.go",
		"bigclaw-go/internal/workflow/orchestration.go",
		"bigclaw-go/cmd/bigclawd/main.go",
	}

	for _, relativePath := range mainlinePaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go mainline entry path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1606LaneReportCapturesMainlineCutover(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1606-runtime-workflow-mainline-cutover.md")

	for _, needle := range []string{
		"BIG-GO-1606",
		"Repository-wide Python file count: `0`.",
		"`bigclaw-go/internal/migration/legacy_model_runtime_modules.go`",
		"`bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json`",
		"`bigclaw-go/internal/worker/runtime.go`",
		"`bigclaw-go/internal/worker/runtime_runonce.go`",
		"`bigclaw-go/internal/service/server.go`",
		"`bigclaw-go/internal/scheduler/scheduler.go`",
		"`bigclaw-go/internal/workflow/engine.go`",
		"`bigclaw-go/internal/workflow/orchestration.go`",
		"`bigclaw-go/cmd/bigclawd/main.go`",
		"`find . -path '*/.git' -prune -o -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' \\) -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1606(RepositoryHasNoPythonFiles|DeletedCompatibilityArtifactsStayAbsent|GoMainlineEntryPathsExist|LaneReportCapturesMainlineCutover)$'`",
		"direct Go-owned runtime, service, scheduler, and workflow entrypaths",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
