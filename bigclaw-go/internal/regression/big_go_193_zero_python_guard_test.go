package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO193RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO193ResidualTestReplacementEvidenceExists(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	requiredPaths := []string{
		"reports/BIG-GO-948-validation.md",
		"reports/BIG-GO-193-validation.md",
		"bigclaw-go/docs/reports/big-go-193-residual-tests-python-sweep-ad.md",
		"bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go",
		"bigclaw-go/internal/regression/python_test_tranche14_removal_test.go",
		"bigclaw-go/internal/regression/python_test_tranche17_removal_test.go",
		"bigclaw-go/internal/regression/big_go_13_legacy_test_contract_sweep_d_test.go",
		"bigclaw-go/internal/regression/big_go_1365_legacy_test_contract_sweep_b_test.go",
		"bigclaw-go/internal/regression/big_go_163_legacy_test_contract_sweep_x_test.go",
		"bigclaw-go/internal/regression/big_go_152_zero_python_guard_test.go",
		"bigclaw-go/internal/regression/big_go_1577_zero_python_guard_test.go",
		"bigclaw-go/internal/regression/deprecation_contract_test.go",
		"bigclaw-go/internal/regression/roadmap_contract_test.go",
		"bigclaw-go/internal/regression/live_shadow_bundle_surface_test.go",
		"bigclaw-go/internal/refill/queue_repo_fixture_test.go",
		"bigclaw-go/cmd/bigclawctl/legacy_shim_help_test.go",
		"bigclaw-go/internal/service/server_test.go",
		"bigclaw-go/internal/pilot/report_test.go",
		"bigclaw-go/internal/issuearchive/archive_test.go",
	}

	for _, relativePath := range requiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected residual-test replacement evidence path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO193LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-193-residual-tests-python-sweep-ad.md")

	for _, needle := range []string{
		"BIG-GO-193",
		"`reports/BIG-GO-948-validation.md`",
		"`bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go`",
		"`bigclaw-go/internal/regression/python_test_tranche14_removal_test.go`",
		"`bigclaw-go/internal/regression/python_test_tranche17_removal_test.go`",
		"`bigclaw-go/internal/regression/big_go_13_legacy_test_contract_sweep_d_test.go`",
		"`bigclaw-go/internal/regression/big_go_1365_legacy_test_contract_sweep_b_test.go`",
		"`bigclaw-go/internal/regression/big_go_163_legacy_test_contract_sweep_x_test.go`",
		"`bigclaw-go/internal/regression/big_go_152_zero_python_guard_test.go`",
		"`bigclaw-go/internal/regression/big_go_1577_zero_python_guard_test.go`",
		"`bigclaw-go/internal/regression/deprecation_contract_test.go`",
		"`bigclaw-go/internal/regression/roadmap_contract_test.go`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO193",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("sweep report missing substring %q", needle)
		}
	}
}
