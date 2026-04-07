package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/migration"
)

func TestBIGGO1358LegacyModelRuntimeReplacementManifestMatchesRetiredModules(t *testing.T) {
	replacements := migration.LegacyModelRuntimeModuleReplacements()
	if len(replacements) != 2 {
		t.Fatalf("expected 2 legacy module replacements, got %d", len(replacements))
	}

	expected := map[string]struct {
		replacementKind string
		goReplacements  []string
		evidencePaths   []string
		statusNeedle    string
	}{
		"src/bigclaw/models.py": {
			replacementKind: "go-package-split",
			goReplacements: []string{
				"bigclaw-go/internal/domain/task.go",
				"bigclaw-go/internal/domain/priority.go",
				"bigclaw-go/internal/risk/assessment.go",
				"bigclaw-go/internal/triage/record.go",
				"bigclaw-go/internal/billing/statement.go",
				"bigclaw-go/internal/workflow/model.go",
			},
			evidencePaths: []string{
				"docs/go-domain-intake-parity-matrix.md",
				"bigclaw-go/internal/workflow/model_test.go",
			},
			statusNeedle: "split Go domain",
		},
		"src/bigclaw/runtime.py": {
			replacementKind: "go-runtime-mainline",
			goReplacements: []string{
				"bigclaw-go/internal/worker/runtime.go",
				"bigclaw-go/internal/worker/runtime_runonce.go",
				"bigclaw-go/internal/worker/runtime_test.go",
			},
			evidencePaths: []string{
				"docs/go-mainline-cutover-issue-pack.md",
				"bigclaw-go/docs/reports/worker-lifecycle-validation-report.md",
			},
			statusNeedle: "Go worker runtime mainline",
		},
	}

	for _, replacement := range replacements {
		want, ok := expected[replacement.RetiredPythonModule]
		if !ok {
			t.Fatalf("unexpected retired module in replacement registry: %+v", replacement)
		}
		if replacement.ReplacementKind != want.replacementKind {
			t.Fatalf("replacement kind for %s = %q, want %q", replacement.RetiredPythonModule, replacement.ReplacementKind, want.replacementKind)
		}
		assertExactStringSlice(t, replacement.GoReplacements, want.goReplacements, replacement.RetiredPythonModule+" go replacements")
		assertExactStringSlice(t, replacement.EvidencePaths, want.evidencePaths, replacement.RetiredPythonModule+" evidence paths")
		if !strings.Contains(replacement.Status, want.statusNeedle) {
			t.Fatalf("replacement status for %s missing %q: %q", replacement.RetiredPythonModule, want.statusNeedle, replacement.Status)
		}
	}
}

func TestBIGGO1358LegacyModelRuntimeReplacementReplacementPathsExist(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	for _, replacement := range migration.LegacyModelRuntimeModuleReplacements() {
		for _, relativePath := range replacement.GoReplacements {
			if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
				t.Fatalf("expected Go replacement path to exist for %s: %s (%v)", replacement.RetiredPythonModule, relativePath, err)
			}
		}
		for _, relativePath := range replacement.EvidencePaths {
			if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
				t.Fatalf("expected evidence path to exist for %s: %s (%v)", replacement.RetiredPythonModule, relativePath, err)
			}
		}
	}
}

func TestBIGGO1358LegacyModelRuntimeReplacementLaneReportCapturesReplacementState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1358-legacy-model-runtime-module-replacement.md")

	for _, needle := range []string{
		"BIG-GO-1358",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw/models.py`",
		"`src/bigclaw/runtime.py`",
		"`bigclaw-go/internal/migration/legacy_model_runtime_modules.go`",
		"`bigclaw-go/internal/domain/task.go`",
		"`bigclaw-go/internal/workflow/model.go`",
		"`bigclaw-go/internal/worker/runtime.go`",
		"`bigclaw-go/internal/worker/runtime_runonce.go`",
		"`docs/go-domain-intake-parity-matrix.md`",
		"`bigclaw-go/docs/reports/worker-lifecycle-validation-report.md`",
		"`find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1358LegacyModelRuntimeReplacement",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}

func assertExactStringSlice(t *testing.T, got []string, want []string, label string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s length = %d, want %d (%v)", label, len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("%s[%d] = %q, want %q", label, index, got[index], want[index])
		}
	}
}
