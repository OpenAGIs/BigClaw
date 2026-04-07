package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAutomationBrokerFailoverStubMatrixCopiesCanonicalArtifacts(t *testing.T) {
	root := t.TempDir()
	mustMkdirAll := func(path string) {
		t.Helper()
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite := func(path, body string) {
		t.Helper()
		mustMkdirAll(filepath.Dir(path))
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	mustWrite(filepath.Join(root, brokerStubReportPath), `{"ticket":"OPE-272","summary":{"scenario_count":8,"passing_scenarios":8,"failing_scenarios":0},"proof_artifacts":{"checkpoint_fencing_summary":"bigclaw-go/docs/reports/broker-checkpoint-fencing-proof-summary.json","retention_boundary_summary":"bigclaw-go/docs/reports/broker-retention-boundary-proof-summary.json"}}`)
	mustWrite(filepath.Join(root, brokerStubCheckpointSummaryPath), `{"ticket":"OPE-230","proof_family":"checkpoint_fencing","summary":{"stale_write_rejections":1}}`)
	mustWrite(filepath.Join(root, brokerStubRetentionSummaryPath), `{"ticket":"OPE-230","proof_family":"retention_boundary","summary":{"retention_floor":3}}`)
	mustWrite(filepath.Join(root, brokerStubArtifactRoot, "BF-04", "checkpoint-transition-log.json"), `[{"owner_id":"consumer-a","transition":"fenced"}]`)
	mustWrite(filepath.Join(root, brokerStubArtifactRoot, "BF-08", "replay-capture.json"), `[{"event_id":"bf08-event-03","duplicate":true}]`)

	report, err := automationBrokerFailoverStubMatrix(root, "tmp/report.json", "tmp/artifacts", "tmp/checkpoint.json", "tmp/retention.json")
	if err != nil {
		t.Fatalf("copy stub matrix: %v", err)
	}
	if report["ticket"] != "OPE-272" {
		t.Fatalf("unexpected report: %+v", report)
	}
	for _, path := range []string{
		filepath.Join(root, "tmp/report.json"),
		filepath.Join(root, "tmp/checkpoint.json"),
		filepath.Join(root, "tmp/retention.json"),
		filepath.Join(root, "tmp/artifacts/BF-04/checkpoint-transition-log.json"),
		filepath.Join(root, "tmp/artifacts/BF-08/replay-capture.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected copied artifact %s: %v", path, err)
		}
	}
}
