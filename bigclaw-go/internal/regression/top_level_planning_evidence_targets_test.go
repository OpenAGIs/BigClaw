package regression

import (
	"strings"
	"testing"
)

func TestTopLevelPlanningEvidenceTargetsAvoidDeletedPythonFiles(t *testing.T) {
	root := regressionRepoRoot(t)

	paths := []string{
		"src/bigclaw/__init__.py",
		"bigclaw-go/internal/planning/planning.go",
	}
	for _, relative := range paths {
		contents := readRepoFile(t, root, relative)
		for _, deletedTarget := range []string{
			"src/bigclaw/workflow.py",
			"src/bigclaw/orchestration.py",
			"src/bigclaw/execution_contract.py",
			"src/bigclaw/saved_views.py",
		} {
			if strings.Contains(contents, deletedTarget) {
				t.Fatalf("%s should not reference deleted planning evidence target %q", relative, deletedTarget)
			}
		}
	}
}
