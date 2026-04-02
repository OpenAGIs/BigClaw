package regression

import (
	"strings"
	"testing"
)

func TestAgentHubIntegrationAlignmentDocUsesGoFirstValidation(t *testing.T) {
	root := regressionRepoRoot(t)
	contents := readRepoFile(t, root, "docs/BigClaw-AgentHub-Integration-Alignment.md")

	required := []string{
		"`cd bigclaw-go && go test ./internal/repo ./internal/collaboration ./internal/observability`",
		"`cd bigclaw-go && go test ./internal/reportstudio ./internal/governance ./internal/planning`",
		"`python3 -m py_compile src/bigclaw/__init__.py`",
		"`cd bigclaw-go && go test ./...`",
	}
	for _, needle := range required {
		if !strings.Contains(contents, needle) {
			t.Fatalf("docs/BigClaw-AgentHub-Integration-Alignment.md missing go-first validation command %q", needle)
		}
	}

	disallowed := []string{
		"PYTHONPATH=src python3 -m pytest",
		"tests/test_repo_registry.py",
		"tests/test_repo_gateway.py",
		"tests/test_repo_links.py",
		"tests/test_repo_board.py",
		"tests/test_repo_collaboration.py",
		"tests/test_observability.py",
		"tests/test_reports.py",
		"tests/test_repo_governance.py",
		"tests/test_repo_triage.py",
		"tests/test_service.py",
		"tests/test_operations.py",
		"tests/test_repo_rollout.py",
	}
	for _, needle := range disallowed {
		if strings.Contains(contents, needle) {
			t.Fatalf("docs/BigClaw-AgentHub-Integration-Alignment.md should not reference deleted repo-root Python validation command %q", needle)
		}
	}
}
