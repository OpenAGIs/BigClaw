package regression

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBrokerBootstrapReviewSummaryStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	reportPath := filepath.Join(repoRoot, "docs", "reports", "broker-bootstrap-review-summary.json")

	var report struct {
		EventLogBackend        string `json:"event_log_backend"`
		TargetBackend          string `json:"target_backend"`
		Ready                  bool   `json:"ready"`
		RuntimePosture         string `json:"runtime_posture"`
		LiveAdapterImplemented bool   `json:"live_adapter_implemented"`
		ProofBoundary          string `json:"proof_boundary"`
		ConfigCompleteness     struct {
			Driver        bool `json:"driver"`
			URLs          bool `json:"urls"`
			Topic         bool `json:"topic"`
			ConsumerGroup bool `json:"consumer_group"`
		} `json:"config_completeness"`
		BrokerBootstrap struct {
			PublishTimeout     string   `json:"publish_timeout"`
			ReplayLimit        int      `json:"replay_limit"`
			CheckpointInterval string   `json:"checkpoint_interval"`
			Ready              bool     `json:"ready"`
			ValidationErrors   []string `json:"validation_errors"`
		} `json:"broker_bootstrap"`
		ValidationErrors []string `json:"validation_errors"`
	}
	readJSONFile(t, reportPath, &report)

	if report.EventLogBackend != "memory" || report.TargetBackend != "broker_replicated" || report.Ready || report.RuntimePosture != "contract_only" || report.LiveAdapterImplemented {
		t.Fatalf("unexpected broker bootstrap review posture: %+v", report)
	}
	if report.ProofBoundary != "broker bootstrap readiness is a pre-adapter contract surface, not live broker durability proof" {
		t.Fatalf("unexpected broker bootstrap proof boundary: %s", report.ProofBoundary)
	}
	if report.ConfigCompleteness.Driver || report.ConfigCompleteness.URLs || report.ConfigCompleteness.Topic || report.ConfigCompleteness.ConsumerGroup {
		t.Fatalf("unexpected broker bootstrap config completeness: %+v", report.ConfigCompleteness)
	}
	if report.BrokerBootstrap.PublishTimeout != "5s" || report.BrokerBootstrap.ReplayLimit != 500 || report.BrokerBootstrap.CheckpointInterval != "5s" || report.BrokerBootstrap.Ready {
		t.Fatalf("unexpected broker bootstrap runtime knobs: %+v", report.BrokerBootstrap)
	}
	if len(report.BrokerBootstrap.ValidationErrors) != 1 || report.BrokerBootstrap.ValidationErrors[0] != "broker event log config missing driver, urls, topic" {
		t.Fatalf("unexpected broker bootstrap nested validation errors: %+v", report.BrokerBootstrap.ValidationErrors)
	}
	if len(report.ValidationErrors) != 1 || report.ValidationErrors[0] != "broker event log config missing driver, urls, topic" {
		t.Fatalf("unexpected broker bootstrap validation errors: %+v", report.ValidationErrors)
	}

	for _, tc := range []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/reports/replicated-event-log-durability-rollout-contract.md",
			substrings: []string{
				"broker bootstrap readiness",
				"docs/reports/broker-validation-summary.json",
				"`contract_only`",
			},
		},
		{
			path: "docs/reports/event-bus-reliability-report.md",
			substrings: []string{
				"broker bootstrap readiness",
				"GET /debug/status",
				"replicated-event-log-durability-rollout-contract.md",
			},
		},
		{
			path: "docs/reports/broker-event-log-adapter-contract.md",
			substrings: []string{
				"BIGCLAW_EVENT_LOG_BROKER_URLS",
				"broker bootstrap endpoints",
				"BIGCLAW_EVENT_LOG_BROKER_TOPIC",
			},
		},
	} {
		contents := readRepoFile(t, repoRoot, tc.path)
		for _, needle := range tc.substrings {
			if !strings.Contains(contents, needle) {
				t.Fatalf("%s missing substring %q", tc.path, needle)
			}
		}
	}
}
