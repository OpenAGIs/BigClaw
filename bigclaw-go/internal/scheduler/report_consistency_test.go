package scheduler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSchedulerPolicyReportReferencesCurrentSurfaces(t *testing.T) {
	report := readRepoFile(t, filepath.Join("..", "..", "docs", "reports", "scheduler-policy-report.md"))
	server := readRepoFile(t, filepath.Join("..", "api", "server.go"))
	policyRuntime := readRepoFile(t, filepath.Join("..", "api", "policy_runtime.go"))
	benchmarkReport := readRepoFile(t, filepath.Join("..", "..", "docs", "reports", "benchmark-report.md"))

	for _, want := range []string{
		"/v2/control-center/policy",
		"/v2/control-center/policy/reload",
		"/internal/scheduler/fairness/{throttle,record,snapshot}",
		"docs/reports/benchmark-report.md",
		"BenchmarkSchedulerDecide-8",
		"required_executor",
		"tool_executors",
		"high_risk_executor",
		"default_executor",
		"memory",
		"SQLite",
		"HTTP",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("scheduler policy report missing %q", want)
		}
	}

	for _, endpoint := range []string{
		"/v2/control-center/policy",
		"/v2/control-center/policy/reload",
		"/internal/scheduler/fairness",
	} {
		if !strings.Contains(server, endpoint) {
			t.Fatalf("server routing missing %q", endpoint)
		}
	}

	if !strings.Contains(policyRuntime, `"/v2/control-center/policy/reload"`) {
		t.Fatal("policy runtime metadata missing reload_url reference")
	}
	if !strings.Contains(benchmarkReport, "BenchmarkSchedulerDecide-8") {
		t.Fatal("benchmark report missing scheduler benchmark label")
	}
}

func readRepoFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(content)
}
