package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/migration"
)

func TestBIGGO1363RuntimeServiceReplacementManifestMatchesRetiredPythonSurfaces(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	replacements := migration.LegacyRuntimeServiceSurfaceReplacements()

	if len(replacements) != 2 {
		t.Fatalf("expected 2 runtime/service replacements, got %d", len(replacements))
	}

	expected := map[string]struct {
		kind       string
		replacedBy []string
	}{
		"src/bigclaw/service.py": {
			kind: "python-service-entrypoint",
			replacedBy: []string{
				"bigclaw-go/internal/service/server.go",
				"bigclaw-go/internal/service/server_test.go",
			},
		},
		"tests/test_service.py": {
			kind: "python-service-regression-tests",
			replacedBy: []string{
				"bigclaw-go/internal/service/server_test.go",
			},
		},
	}

	for _, replacement := range replacements {
		want, ok := expected[replacement.RetiredPythonPath]
		if !ok {
			t.Fatalf("unexpected retired Python path in manifest: %+v", replacement)
		}
		if replacement.SurfaceKind != want.kind {
			t.Fatalf("unexpected surface kind for %s: %s", replacement.RetiredPythonPath, replacement.SurfaceKind)
		}
		if strings.TrimSpace(replacement.Status) == "" {
			t.Fatalf("replacement status must be populated for %s", replacement.RetiredPythonPath)
		}
		if len(replacement.GoReplacements) != len(want.replacedBy) {
			t.Fatalf("unexpected Go replacement count for %s: %v", replacement.RetiredPythonPath, replacement.GoReplacements)
		}
		for index, path := range want.replacedBy {
			if replacement.GoReplacements[index] != path {
				t.Fatalf("unexpected Go replacement at index %d for %s: got %s want %s", index, replacement.RetiredPythonPath, replacement.GoReplacements[index], path)
			}
		}
		if len(replacement.EvidencePaths) == 0 {
			t.Fatalf("expected evidence paths for %s", replacement.RetiredPythonPath)
		}
	}

	for retiredPath := range expected {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(retiredPath))); !os.IsNotExist(err) {
			t.Fatalf("expected retired Python surface to stay absent: %s", retiredPath)
		}
	}
}

func TestBIGGO1363RuntimeServiceReplacementReplacementPathsExist(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	requiredPaths := []string{
		"bigclaw-go/internal/service/server.go",
		"bigclaw-go/internal/service/server_test.go",
		"reports/OPE-148-150-validation.md",
		"reports/OPE-151-153-validation.md",
		"reports/BIG-GO-948-validation.md",
		"bigclaw-go/docs/reports/big-go-1363-runtime-service-sweep.md",
	}

	for _, relativePath := range requiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected replacement evidence path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1363RuntimeServiceReplacementLaneReportCapturesReplacementState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1363-runtime-service-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1363",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw/service.py`",
		"`tests/test_service.py`",
		"`bigclaw-go/internal/service/server.go`",
		"`bigclaw-go/internal/service/server_test.go`",
		"`reports/OPE-148-150-validation.md`",
		"`reports/OPE-151-153-validation.md`",
		"`reports/BIG-GO-948-validation.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/service -run 'TestRepoGovernanceEnforcerBlocksQuotaAndSidecarFailures|TestServerEntryHealthMetrics|TestEnsureStaticIndex'",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1363RuntimeServiceReplacement",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
