package regression

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"bigclaw-go/internal/config"
	"bigclaw-go/internal/events"
)

func TestDurabilityRolloutScorecardReportStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	reportPaths := []string{
		filepath.Join(repoRoot, "docs", "reports", "broker-durability-rollout-scorecard.json"),
		filepath.Join(repoRoot, "docs", "reports", "durability-rollout-scorecard.json"),
	}

	cfg := config.Default()
	plan := events.NewDurabilityPlanWithBrokerConfig(
		cfg.EventLogBackend,
		cfg.EventLogTargetBackend,
		cfg.EventLogReplicationFactor,
		events.BrokerRuntimeConfig{
			Driver:             cfg.EventLogBrokerDriver,
			URLs:               cfg.EventLogBrokerURLs,
			Topic:              cfg.EventLogBrokerTopic,
			ConsumerGroup:      cfg.EventLogConsumerGroup,
			PublishTimeout:     cfg.EventLogPublishTimeout,
			ReplayLimit:        cfg.EventLogReplayLimit,
			CheckpointInterval: cfg.EventLogCheckpointInterval,
		},
	)
	want := plan.RolloutScorecard
	for _, reportPath := range reportPaths {
		var report events.RolloutScorecard
		readJSONFile(t, reportPath, &report)
		if !reflect.DeepEqual(report, want) {
			t.Fatalf("durability rollout scorecard drifted for %s\nreport=%+v\nwant=%+v", reportPath, report, want)
		}
	}

	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/reports/event-bus-reliability-report.md",
			substrings: []string{
				"event_durability_rollout",
				"broker-durability-rollout-scorecard.json",
				"durability-rollout-scorecard.json",
				"replicated-broker-durability-rollout-spike.md",
			},
		},
		{
			path: "docs/reports/replicated-event-log-durability-rollout-contract.md",
			substrings: []string{
				"event_durability_rollout",
				"broker-durability-rollout-scorecard.json",
				"durability-rollout-scorecard.json",
				"replicated-broker-durability-rollout-spike.md",
			},
		},
		{
			path: "docs/reports/replicated-broker-durability-rollout-spike.md",
			substrings: []string{
				"SQLite",
				"contract_only",
				"harness_proven",
				"follow-on implementation slices",
			},
		},
	}

	for _, tc := range cases {
		contents := readRepoFile(t, repoRoot, tc.path)
		for _, needle := range tc.substrings {
			if !strings.Contains(contents, needle) {
				t.Fatalf("%s missing substring %q", tc.path, needle)
			}
		}
	}
}

func TestBrokerProofSummariesStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)

	type proofSummary struct {
		Ticket               string `json:"ticket"`
		SourceReport         string `json:"source_report"`
		SummarySchemaVersion string `json:"summary_schema_version"`
		ProofFamily          string `json:"proof_family"`
		Status               string `json:"status"`
		RolloutGateStatuses  []struct {
			Name        string   `json:"name"`
			Status      string   `json:"status"`
			ScenarioIDs []string `json:"scenario_ids"`
		} `json:"rollout_gate_statuses"`
	}

	cases := []struct {
		path          string
		proofFamily   string
		requiredGates map[string]string
	}{
		{
			path:        filepath.Join(repoRoot, "docs", "reports", "broker-checkpoint-fencing-proof-summary.json"),
			proofFamily: "checkpoint_fencing",
			requiredGates: map[string]string{
				"durable_publish_ack":           "unknown",
				"replay_checkpoint_alignment":   "passed",
				"retention_boundary_visibility": "unknown",
				"live_fanout_isolation":         "unknown",
			},
		},
		{
			path:        filepath.Join(repoRoot, "docs", "reports", "broker-retention-boundary-proof-summary.json"),
			proofFamily: "retention_boundary",
			requiredGates: map[string]string{
				"durable_publish_ack":           "unknown",
				"replay_checkpoint_alignment":   "passed",
				"retention_boundary_visibility": "passed",
				"live_fanout_isolation":         "unknown",
			},
		},
	}

	for _, tc := range cases {
		var summary proofSummary
		readJSONFile(t, tc.path, &summary)
		if summary.Ticket != "OPE-230" {
			t.Fatalf("%s ticket = %q, want OPE-230", tc.path, summary.Ticket)
		}
		if summary.SourceReport != "bigclaw-go/docs/reports/broker-failover-stub-report.json" {
			t.Fatalf("%s source_report = %q", tc.path, summary.SourceReport)
		}
		if summary.SummarySchemaVersion == "" {
			t.Fatalf("%s missing summary_schema_version", tc.path)
		}
		if summary.ProofFamily != tc.proofFamily {
			t.Fatalf("%s proof_family = %q, want %q", tc.path, summary.ProofFamily, tc.proofFamily)
		}
		if summary.Status != "passed" {
			t.Fatalf("%s status = %q, want passed", tc.path, summary.Status)
		}
		actualGates := map[string]string{}
		for _, gate := range summary.RolloutGateStatuses {
			actualGates[gate.Name] = gate.Status
		}
		for gate, want := range tc.requiredGates {
			if actualGates[gate] != want {
				t.Fatalf("%s gate %s = %q, want %q", tc.path, gate, actualGates[gate], want)
			}
		}
	}

	docCases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/reports/event-bus-reliability-report.md",
			substrings: []string{
				"broker-checkpoint-fencing-proof-summary.json",
				"broker-retention-boundary-proof-summary.json",
			},
		},
		{
			path: "docs/reports/replicated-event-log-durability-rollout-contract.md",
			substrings: []string{
				"broker-checkpoint-fencing-proof-summary.json",
				"broker-retention-boundary-proof-summary.json",
			},
		},
		{
			path: "docs/reports/replay-retention-semantics-report.md",
			substrings: []string{
				"broker-retention-boundary-proof-summary.json",
			},
		},
	}

	for _, tc := range docCases {
		contents := readRepoFile(t, repoRoot, tc.path)
		for _, needle := range tc.substrings {
			if !strings.Contains(contents, needle) {
				t.Fatalf("%s missing substring %q", tc.path, needle)
			}
		}
	}
}
