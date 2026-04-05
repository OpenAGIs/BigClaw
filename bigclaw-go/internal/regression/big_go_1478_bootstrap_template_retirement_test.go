package regression

import (
	"strings"
	"testing"
)

func TestBIGGO1478BootstrapTemplateStaysGoOnly(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	template := readRepoFile(t, rootRepo, "docs/symphony-repo-bootstrap-template.md")

	for _, forbidden := range []string{
		"workspace_bootstrap.py",
		"workspace_bootstrap_cli.py",
		"Python compatibility package path",
	} {
		if strings.Contains(template, forbidden) {
			t.Fatalf("bootstrap template reintroduced retired Python bootstrap guidance %q", forbidden)
		}
	}

	for _, required := range []string{
		"`scripts/ops/bigclawctl`",
		"`workflow.md`",
		"Go/shell-only",
		"`bigclaw-go/internal/bootstrap/*`",
	} {
		if !strings.Contains(template, required) {
			t.Fatalf("bootstrap template missing Go-only contract %q", required)
		}
	}
}

func TestBIGGO1478LaneReportCapturesTemplateRetirement(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1478-python-doc-manifest-retirement.md")

	for _, needle := range []string{
		"BIG-GO-1478",
		"tracked Python file count at branch start: `0`",
		"physical Python file count at branch start: `0`",
		"`docs/symphony-repo-bootstrap-template.md`",
		"`scripts/ops/bigclawctl`",
		"`bigclaw-go/internal/bootstrap/*`",
		"`workflow.md`",
		"`git ls-files '*.py'`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1478|TestBIGGO1299'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
