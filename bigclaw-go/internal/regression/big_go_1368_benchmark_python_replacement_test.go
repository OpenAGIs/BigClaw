package regression

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/migration"
)

func TestBIGGO1368BenchmarkPythonReplacementManifestCapturesBenchmarkLane(t *testing.T) {
	replacements := migration.BenchmarkScriptReplacements()
	if len(replacements) != 4 {
		t.Fatalf("expected 4 benchmark script replacements, got %d", len(replacements))
	}

	expected := map[string]struct {
		replacementKind string
		activePaths     []string
		evidencePaths   []string
		statusNeedle    string
	}{
		"bigclaw-go/scripts/benchmark/soak_local.py": {
			replacementKind: "go-cli-subcommand",
			activePaths: []string{
				"bigclaw-go/cmd/bigclawctl/automation_commands.go",
				"bigclaw-go/cmd/bigclawctl/automation_commands_test.go",
				"bigclaw-go/scripts/benchmark/run_suite.sh",
			},
			evidencePaths: []string{
				"bigclaw-go/docs/benchmark-plan.md",
				"bigclaw-go/docs/reports/long-duration-soak-report.md",
			},
			statusNeedle: "automation benchmark soak-local",
		},
		"bigclaw-go/scripts/benchmark/run_matrix.py": {
			replacementKind: "go-cli-subcommand",
			activePaths: []string{
				"bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go",
				"bigclaw-go/cmd/bigclawctl/automation_commands_test.go",
				"bigclaw-go/scripts/benchmark/run_suite.sh",
			},
			evidencePaths: []string{
				"bigclaw-go/docs/benchmark-plan.md",
				"bigclaw-go/docs/reports/benchmark-matrix-report.json",
			},
			statusNeedle: "automation benchmark run-matrix",
		},
		"bigclaw-go/scripts/benchmark/capacity_certification.py": {
			replacementKind: "go-cli-subcommand",
			activePaths: []string{
				"bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go",
				"bigclaw-go/cmd/bigclawctl/automation_commands_test.go",
			},
			evidencePaths: []string{
				"bigclaw-go/docs/go-cli-script-migration.md",
				"bigclaw-go/docs/reports/capacity-certification-matrix.json",
				"bigclaw-go/docs/reports/capacity-certification-report.md",
			},
			statusNeedle: "automation benchmark capacity-certification",
		},
		"bigclaw-go/scripts/benchmark/capacity_certification_test.py": {
			replacementKind: "go-test-coverage",
			activePaths: []string{
				"bigclaw-go/cmd/bigclawctl/automation_commands_test.go",
				"bigclaw-go/internal/regression/big_go_1160_script_migration_test.go",
			},
			evidencePaths: []string{
				"bigclaw-go/docs/go-cli-script-migration.md",
			},
			statusNeedle: "Go command and regression tests",
		},
	}

	for _, replacement := range replacements {
		want, ok := expected[replacement.RetiredPythonPath]
		if !ok {
			t.Fatalf("unexpected retired benchmark path in replacement registry: %+v", replacement)
		}
		if replacement.ReplacementKind != want.replacementKind {
			t.Fatalf("replacement kind for %s = %q, want %q", replacement.RetiredPythonPath, replacement.ReplacementKind, want.replacementKind)
		}
		assertExactStringSlice(t, replacement.ActivePaths, want.activePaths, replacement.RetiredPythonPath+" active paths")
		assertExactStringSlice(t, replacement.EvidencePaths, want.evidencePaths, replacement.RetiredPythonPath+" evidence paths")
		if !strings.Contains(replacement.Status, want.statusNeedle) {
			t.Fatalf("replacement status for %s missing %q: %q", replacement.RetiredPythonPath, want.statusNeedle, replacement.Status)
		}
	}
}

func TestBIGGO1368BenchmarkPythonReplacementReplacementPathsExist(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	for _, replacement := range migration.BenchmarkScriptReplacements() {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(replacement.RetiredPythonPath))); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected retired benchmark Python path to stay deleted: %s (err=%v)", replacement.RetiredPythonPath, err)
		}
		for _, relativePath := range replacement.ActivePaths {
			if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
				t.Fatalf("expected active replacement path to exist for %s: %s (%v)", replacement.RetiredPythonPath, relativePath, err)
			}
		}
		for _, relativePath := range replacement.EvidencePaths {
			if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
				t.Fatalf("expected evidence path to exist for %s: %s (%v)", replacement.RetiredPythonPath, relativePath, err)
			}
		}
	}
}

func TestBIGGO1368BenchmarkPythonReplacementLaneReportCapturesReplacementState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1368-benchmark-python-replacement.md")

	for _, needle := range []string{
		"BIG-GO-1368",
		"Repository-wide Python file count: `0`.",
		"`bigclaw-go/scripts/benchmark/soak_local.py`",
		"`bigclaw-go/scripts/benchmark/run_matrix.py`",
		"`bigclaw-go/scripts/benchmark/capacity_certification.py`",
		"`bigclaw-go/scripts/benchmark/capacity_certification_test.py`",
		"`bigclaw-go/internal/migration/benchmark_script_replacements.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_commands.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_benchmark_commands.go`",
		"`bigclaw-go/cmd/bigclawctl/automation_commands_test.go`",
		"`bigclaw-go/scripts/benchmark/run_suite.sh`",
		"`bigclaw-go/docs/reports/benchmark-matrix-report.json`",
		"`bigclaw-go/docs/reports/capacity-certification-matrix.json`",
		"`find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1368 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`",
		"`cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1368/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1368BenchmarkPythonReplacement",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
