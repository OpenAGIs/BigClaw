package regression

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type liveMultiNodeTakeoverProofReport struct {
	Ticket            string `json:"ticket"`
	Status            string `json:"status"`
	CurrentPrimitives struct {
		LeaseAwareCheckpoints []string `json:"lease_aware_checkpoints"`
		SharedQueueEvidence   []string `json:"shared_queue_evidence"`
		LiveTakeoverHarness   []string `json:"live_takeover_harness"`
	} `json:"current_primitives"`
	Scenarios []struct {
		AuditLogPaths []string `json:"audit_log_paths"`
	} `json:"scenarios"`
}

func TestLiveMultiNodeSubscriberTakeoverProofReport(t *testing.T) {
	repoRoot := repoRoot(t)
	reportPath := filepath.Join(repoRoot, "docs", "reports", "live-multi-node-subscriber-takeover-report.json")
	contents, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read live multi-node takeover report: %v", err)
	}

	var report liveMultiNodeTakeoverProofReport
	if err := json.Unmarshal(contents, &report); err != nil {
		t.Fatalf("decode live multi-node takeover report: %v", err)
	}

	if report.Ticket != "OPE-260" || report.Status != "live-multi-node-proof" {
		t.Fatalf("unexpected live takeover report identity: %+v", report)
	}

	requiredPaths := []string{}
	requiredPaths = append(requiredPaths, report.CurrentPrimitives.LeaseAwareCheckpoints...)
	requiredPaths = append(requiredPaths, report.CurrentPrimitives.SharedQueueEvidence...)
	requiredPaths = append(requiredPaths, report.CurrentPrimitives.LiveTakeoverHarness...)
	for _, scenario := range report.Scenarios {
		requiredPaths = append(requiredPaths, scenario.AuditLogPaths...)
	}

	for _, candidate := range requiredPaths {
		if strings.TrimSpace(candidate) == "" {
			t.Fatal("live takeover report contains an empty path")
		}
		if _, err := os.Stat(resolveRepoPath(repoRoot, candidate)); err != nil {
			t.Fatalf("expected referenced path %q to exist: %v", candidate, err)
		}
	}

	digestPath := resolveRepoPath(repoRoot, "docs/reports/subscriber-takeover-executability-follow-up-digest.md")
	digestBody, err := os.ReadFile(digestPath)
	if err != nil {
		t.Fatalf("read follow-up digest: %v", err)
	}
	if !strings.Contains(string(digestBody), "live-multi-node-subscriber-takeover-report.json") {
		t.Fatalf("expected follow-up digest to reference the live takeover report")
	}
}
