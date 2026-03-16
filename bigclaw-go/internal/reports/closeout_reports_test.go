package reports

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const moduleRoot = "../.."

func TestCloseoutReportsReferenceExpectedEvidence(t *testing.T) {
	t.Parallel()

	cases := []struct {
		report string
		want   []string
	}{
		{
			report: "docs/reports/epic-concurrency-readiness-report.md",
			want: []string{
				"docs/reports/epic-closure-readiness-report.md",
				"docs/reports/long-duration-soak-report.md",
				"docs/reports/mixed-workload-validation-report.md",
				"docs/reports/multi-node-coordination-report.md",
				"docs/reports/queue-reliability-report.md",
				"docs/reports/soak-local-1000x24.json",
				"docs/reports/soak-local-2000x24.json",
				"docs/reports/mixed-workload-matrix-report.json",
				"docs/reports/multi-node-shared-queue-report.json",
				"docs/reports/live-validation-summary.json",
			},
		},
		{
			report: "docs/reports/event-bus-reliability-report.md",
			want: []string{
				"docs/reports/live-validation-index.md",
				"docs/reports/live-validation-index.json",
				"docs/reports/replay-retention-semantics-report.md",
				"docs/reports/multi-subscriber-takeover-validation-report.md",
				"docs/reports/replicated-event-log-durability-rollout-contract.md",
				"docs/reports/broker-failover-fault-injection-validation-pack.md",
			},
		},
		{
			report: "docs/reports/queue-reliability-report.md",
			want: []string{
				"docs/reports/lease-recovery-report.md",
				"docs/reports/multi-node-coordination-report.md",
				"docs/reports/multi-node-shared-queue-report.json",
				"docs/reports/live-validation-summary.json",
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.report, func(t *testing.T) {
			t.Parallel()

			content := mustReadReport(t, tc.report)
			for _, needle := range tc.want {
				if !strings.Contains(content, needle) {
					t.Fatalf("%s does not reference %s", tc.report, needle)
				}
				if _, err := os.Stat(filepath.Join(moduleRoot, filepath.FromSlash(needle))); err != nil {
					t.Fatalf("referenced artifact %s is missing: %v", needle, err)
				}
			}
		})
	}
}

func mustReadReport(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(moduleRoot, filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}
