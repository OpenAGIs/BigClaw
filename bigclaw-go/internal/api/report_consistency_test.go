package api

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestOperationsFoundationReportReferencesRegisteredEndpoints(t *testing.T) {
	serverSource := mustReadAPIRelative(t, "server.go")
	report := mustReadProjectRelative(t, filepath.Join("docs", "reports", "v2-phase1-operations-foundation-report.md"))

	endpoints := []string{
		"/v2/dashboard/engineering",
		"/v2/dashboard/operations",
		"/v2/triage/center",
		"/v2/regression/center",
		"/v2/control-center",
		"/v2/control-center/audit",
		"/v2/control-center/actions",
		"/v2/control-center/policy",
		"/v2/control-center/policy/reload",
		"/v2/runs/",
	}
	for _, endpoint := range endpoints {
		if !strings.Contains(serverSource, endpoint) {
			t.Fatalf("server registration missing endpoint %q", endpoint)
		}
	}

	reportRefs := []string{
		"`GET /v2/dashboard/engineering`",
		"`GET /v2/dashboard/operations`",
		"`GET /v2/triage/center`",
		"`GET /v2/regression/center`",
		"`GET /v2/control-center`",
		"`GET /v2/control-center/audit`",
		"`POST /v2/control-center/actions`",
		"`GET /v2/control-center/policy`",
		"`POST /v2/control-center/policy/reload`",
		"`GET /v2/runs/{task_id}`",
		"`GET /v2/runs/{task_id}/audit`",
		"`GET /v2/runs/{task_id}/report`",
	}
	for _, ref := range reportRefs {
		if !strings.Contains(report, ref) {
			t.Fatalf("operations foundation report missing reference %q", ref)
		}
	}
}

func TestReviewReadinessReferencesOperationsFoundationEvidencePack(t *testing.T) {
	readiness := mustReadProjectRelative(t, filepath.Join("docs", "reports", "review-readiness.md"))
	for _, ref := range []string{
		"`OPE-255`",
		"`docs/reports/v2-phase1-operations-foundation-report.md`",
		"`docs/reports/go-control-plane-observability-report.md`",
	} {
		if !strings.Contains(readiness, ref) {
			t.Fatalf("review readiness missing reference %q", ref)
		}
	}
}

func mustReadAPIRelative(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller path")
	}
	return mustReadFile(t, filepath.Join(filepath.Dir(file), name))
}

func mustReadProjectRelative(t *testing.T, rel string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	return mustReadFile(t, filepath.Join(root, rel))
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
