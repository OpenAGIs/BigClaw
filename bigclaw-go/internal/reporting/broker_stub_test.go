package reporting

import (
	"path/filepath"
	"testing"
	"time"
)

func TestBuildBrokerFailoverStubReport(t *testing.T) {
	report := BuildBrokerFailoverStubReport(time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC))
	if asString(report["ticket"]) != "OPE-272" || asString(report["status"]) != "deterministic-harness" {
		t.Fatalf("unexpected broker stub report metadata: %+v", report)
	}
	if asInt(asMap(report["summary"])["scenario_count"]) != 8 || asInt(asMap(report["summary"])["passing_scenarios"]) != 8 {
		t.Fatalf("unexpected broker stub summary: %+v", report["summary"])
	}
	scenarios := anyToMapSlice(report["scenarios"])
	if len(scenarios) != 8 || asString(scenarios[0]["scenario_id"]) != "BF-01" {
		t.Fatalf("unexpected broker stub scenarios: %+v", scenarios)
	}
}

func TestWriteBrokerFailoverStubArtifacts(t *testing.T) {
	root := t.TempDir()
	if err := WriteBrokerFailoverStubArtifacts(root, BrokerStubOptions{}); err != nil {
		t.Fatalf("write broker stub artifacts: %v", err)
	}
	for _, path := range []string{
		"bigclaw-go/docs/reports/broker-failover-stub-report.json",
		"bigclaw-go/docs/reports/broker-checkpoint-fencing-proof-summary.json",
		"bigclaw-go/docs/reports/broker-retention-boundary-proof-summary.json",
		filepath.Join("bigclaw-go/docs/reports/broker-failover-stub-artifacts", "BF-01", "publish-attempt-ledger.json"),
	} {
		if !pathExists(filepath.Join(root, path)) {
			t.Fatalf("expected artifact %s", path)
		}
	}
}
