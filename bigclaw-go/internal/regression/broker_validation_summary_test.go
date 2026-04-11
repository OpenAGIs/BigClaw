package regression

import (
	"path/filepath"
	"testing"
)

func TestBrokerValidationSummaryStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	summaryPath := filepath.Join(repoRoot, "docs", "reports", "broker-validation-summary.json")
	bootstrapPath := filepath.Join(repoRoot, "docs", "reports", "broker-bootstrap-review-summary.json")

	var summary struct {
		Enabled                     bool   `json:"enabled"`
		Backend                     any    `json:"backend"`
		BundleSummaryPath           string `json:"bundle_summary_path"`
		CanonicalSummaryPath        string `json:"canonical_summary_path"`
		BundleBootstrapSummaryPath  string `json:"bundle_bootstrap_summary_path"`
		CanonicalBootstrapSummaryPath string `json:"canonical_bootstrap_summary_path"`
		ValidationPackPath          string `json:"validation_pack_path"`
		ConfigurationState          string `json:"configuration_state"`
		BootstrapReady             bool   `json:"bootstrap_ready"`
		RuntimePosture             string `json:"runtime_posture"`
		LiveAdapterImplemented     bool   `json:"live_adapter_implemented"`
		ProofBoundary              string `json:"proof_boundary"`
		ValidationErrors           []string `json:"validation_errors"`
		ConfigCompleteness         struct {
			Driver        bool `json:"driver"`
			URLs          bool `json:"urls"`
			Topic         bool `json:"topic"`
			ConsumerGroup bool `json:"consumer_group"`
		} `json:"config_completeness"`
		Status          string `json:"status"`
		Reason          string `json:"reason"`
		BootstrapSummary struct {
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
				PublishTimeout  string   `json:"publish_timeout"`
				ReplayLimit     int      `json:"replay_limit"`
				CheckpointInterval string `json:"checkpoint_interval"`
				Ready           bool     `json:"ready"`
				ValidationErrors []string `json:"validation_errors"`
			} `json:"broker_bootstrap"`
			ValidationErrors []string `json:"validation_errors"`
		} `json:"bootstrap_summary"`
	}
	readJSONFile(t, summaryPath, &summary)

	if summary.Enabled || summary.Backend != nil {
		t.Fatalf("unexpected broker summary enabled/backend fields: %+v", summary)
	}
	if summary.BundleSummaryPath != "docs/reports/live-validation-runs/20260316T140138Z/broker-validation-summary.json" || summary.CanonicalSummaryPath != "docs/reports/broker-validation-summary.json" {
		t.Fatalf("unexpected broker summary paths: %+v", summary)
	}
	if summary.BundleBootstrapSummaryPath != "docs/reports/live-validation-runs/20260316T140138Z/broker-bootstrap-review-summary.json" || summary.CanonicalBootstrapSummaryPath != "docs/reports/broker-bootstrap-review-summary.json" {
		t.Fatalf("unexpected broker bootstrap summary paths: %+v", summary)
	}
	if summary.ValidationPackPath != "docs/reports/broker-failover-fault-injection-validation-pack.md" || summary.ConfigurationState != "not_configured" || summary.Status != "skipped" || summary.Reason != "not_configured" {
		t.Fatalf("unexpected broker summary posture: %+v", summary)
	}
	if summary.BootstrapReady || summary.RuntimePosture != "contract_only" || summary.LiveAdapterImplemented {
		t.Fatalf("unexpected broker readiness booleans: %+v", summary)
	}
	if summary.ProofBoundary != "broker bootstrap readiness is a pre-adapter contract surface, not live broker durability proof" {
		t.Fatalf("unexpected broker proof boundary: %s", summary.ProofBoundary)
	}
	if len(summary.ValidationErrors) != 1 || summary.ValidationErrors[0] != "broker event log config missing driver, urls, topic" {
		t.Fatalf("unexpected broker validation errors: %+v", summary.ValidationErrors)
	}
	if summary.ConfigCompleteness.Driver || summary.ConfigCompleteness.URLs || summary.ConfigCompleteness.Topic || summary.ConfigCompleteness.ConsumerGroup {
		t.Fatalf("unexpected broker config completeness: %+v", summary.ConfigCompleteness)
	}

	if summary.BootstrapSummary.EventLogBackend != "memory" || summary.BootstrapSummary.TargetBackend != "broker_replicated" || summary.BootstrapSummary.Ready || summary.BootstrapSummary.RuntimePosture != "contract_only" || summary.BootstrapSummary.LiveAdapterImplemented {
		t.Fatalf("unexpected embedded bootstrap summary: %+v", summary.BootstrapSummary)
	}
	if summary.BootstrapSummary.ProofBoundary != summary.ProofBoundary {
		t.Fatalf("bootstrap proof boundary drift: %q != %q", summary.BootstrapSummary.ProofBoundary, summary.ProofBoundary)
	}
	if summary.BootstrapSummary.ConfigCompleteness.Driver || summary.BootstrapSummary.ConfigCompleteness.URLs || summary.BootstrapSummary.ConfigCompleteness.Topic || summary.BootstrapSummary.ConfigCompleteness.ConsumerGroup {
		t.Fatalf("unexpected embedded bootstrap config completeness: %+v", summary.BootstrapSummary.ConfigCompleteness)
	}
	if summary.BootstrapSummary.BrokerBootstrap.PublishTimeout != "5s" || summary.BootstrapSummary.BrokerBootstrap.ReplayLimit != 500 || summary.BootstrapSummary.BrokerBootstrap.CheckpointInterval != "5s" || summary.BootstrapSummary.BrokerBootstrap.Ready {
		t.Fatalf("unexpected broker bootstrap config: %+v", summary.BootstrapSummary.BrokerBootstrap)
	}
	if len(summary.BootstrapSummary.BrokerBootstrap.ValidationErrors) != 1 || summary.BootstrapSummary.BrokerBootstrap.ValidationErrors[0] != "broker event log config missing driver, urls, topic" {
		t.Fatalf("unexpected broker bootstrap validation errors: %+v", summary.BootstrapSummary.BrokerBootstrap.ValidationErrors)
	}
	if len(summary.BootstrapSummary.ValidationErrors) != 1 || summary.BootstrapSummary.ValidationErrors[0] != "broker event log config missing driver, urls, topic" {
		t.Fatalf("unexpected embedded bootstrap validation errors: %+v", summary.BootstrapSummary.ValidationErrors)
	}

	var bootstrap struct {
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
			PublishTimeout   string   `json:"publish_timeout"`
			ReplayLimit      int      `json:"replay_limit"`
			CheckpointInterval string `json:"checkpoint_interval"`
			Ready            bool     `json:"ready"`
			ValidationErrors []string `json:"validation_errors"`
		} `json:"broker_bootstrap"`
		ValidationErrors []string `json:"validation_errors"`
	} 
	readJSONFile(t, bootstrapPath, &bootstrap)

	if bootstrap.EventLogBackend != summary.BootstrapSummary.EventLogBackend || bootstrap.TargetBackend != summary.BootstrapSummary.TargetBackend || bootstrap.Ready != summary.BootstrapSummary.Ready || bootstrap.RuntimePosture != summary.BootstrapSummary.RuntimePosture || bootstrap.LiveAdapterImplemented != summary.BootstrapSummary.LiveAdapterImplemented || bootstrap.ProofBoundary != summary.BootstrapSummary.ProofBoundary {
		t.Fatalf("broker bootstrap summary drift: %+v vs %+v", bootstrap, summary.BootstrapSummary)
	}
	if bootstrap.BrokerBootstrap.PublishTimeout != summary.BootstrapSummary.BrokerBootstrap.PublishTimeout || bootstrap.BrokerBootstrap.ReplayLimit != summary.BootstrapSummary.BrokerBootstrap.ReplayLimit || bootstrap.BrokerBootstrap.CheckpointInterval != summary.BootstrapSummary.BrokerBootstrap.CheckpointInterval || bootstrap.BrokerBootstrap.Ready != summary.BootstrapSummary.BrokerBootstrap.Ready {
		t.Fatalf("broker bootstrap nested config drift: %+v vs %+v", bootstrap.BrokerBootstrap, summary.BootstrapSummary.BrokerBootstrap)
	}
}
