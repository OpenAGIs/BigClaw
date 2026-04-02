package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/planning"
)

func TestV3PlanningBacklogUsesGoReplacementsForRemovedPythonTests(t *testing.T) {
	repoRoot := regressionRepoRoot(t)
	backlog := planning.BuildV3CandidateBacklog()

	replacementFiles := []string{
		"bigclaw-go/internal/designsystem/designsystem_test.go",
		"bigclaw-go/internal/consoleia/consoleia_test.go",
		"bigclaw-go/internal/uireview/uireview_test.go",
		"bigclaw-go/internal/contract/execution_test.go",
		"bigclaw-go/internal/product/console_test.go",
		"bigclaw-go/internal/product/saved_views_test.go",
		"bigclaw-go/internal/evaluation/evaluation_test.go",
		"bigclaw-go/internal/collaboration/thread_test.go",
		"bigclaw-go/internal/pilot/rollout_test.go",
		"bigclaw-go/internal/reportstudio/reportstudio_test.go",
	}
	for _, relativePath := range replacementFiles {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); err != nil {
			t.Fatalf("expected Go replacement file to exist: %s (%v)", relativePath, err)
		}
	}

	removedPythonTests := []string{
		"tests/test_design_system.py",
		"tests/test_console_ia.py",
		"tests/test_operations.py",
		"tests/test_reports.py",
	}
	for _, relativePath := range removedPythonTests {
		if _, err := os.Stat(filepath.Join(repoRoot, relativePath)); !os.IsNotExist(err) {
			t.Fatalf("expected removed Python test to stay absent: %s", relativePath)
		}
	}

	pythonPlanningSource, err := os.ReadFile(filepath.Join(repoRoot, "src", "bigclaw", "planning.py"))
	if err != nil {
		t.Fatalf("read src/bigclaw/planning.py: %v", err)
	}
	for _, disallowed := range []string{
		"pytest",
		"tests/test_design_system.py",
		"tests/test_console_ia.py",
		"tests/test_ui_review.py",
		"tests/test_control_center.py",
		"tests/test_operations.py",
		"tests/test_evaluation.py",
		"tests/test_orchestration.py",
		"tests/test_reports.py",
	} {
		if strings.Contains(string(pythonPlanningSource), disallowed) {
			t.Fatalf("src/bigclaw/planning.py still references removed Python test asset %q", disallowed)
		}
	}

	for _, candidate := range backlog.Candidates {
		if strings.Contains(candidate.ValidationCommand, "pytest") || strings.Contains(candidate.ValidationCommand, "tests/test_") {
			t.Fatalf("candidate %s still references removed Python tests in validation command: %s", candidate.CandidateID, candidate.ValidationCommand)
		}
		if !strings.Contains(candidate.ValidationCommand, "go test ./internal/") {
			t.Fatalf("candidate %s validation command is not Go-native: %s", candidate.CandidateID, candidate.ValidationCommand)
		}
		for _, link := range candidate.EvidenceLinks {
			if strings.Contains(link.Target, "tests/test_") {
				t.Fatalf("candidate %s still references removed Python test evidence target: %+v", candidate.CandidateID, link)
			}
		}
	}
}
