package reporting

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRetentionExternalStoreFollowUpDigestReferencesRequiredReports(t *testing.T) {
	root := filepath.Join("..", "..")
	digestPath := filepath.Join(root, "docs", "reports", "retention-external-store-follow-up-digest.md")
	body, err := os.ReadFile(digestPath)
	if err != nil {
		t.Fatalf("read digest: %v", err)
	}

	requiredRefs := []string{
		"docs/reports/replay-retention-semantics-report.md",
		"docs/reports/event-bus-reliability-report.md",
		"docs/reports/review-readiness.md",
		"docs/reports/issue-coverage.md",
		"docs/openclaw-parallel-gap-analysis.md",
	}
	content := string(body)
	for _, ref := range requiredRefs {
		if !strings.Contains(content, ref) {
			t.Fatalf("digest missing required reference %q", ref)
		}
	}
}
