package regression

import (
	"path/filepath"
	"reflect"
	"testing"
)

type brokerBootstrapReviewSummary struct {
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

type brokerValidationSummary struct {
	Enabled                       bool                         `json:"enabled"`
	Backend                       any                          `json:"backend"`
	BundleSummaryPath             string                       `json:"bundle_summary_path"`
	CanonicalSummaryPath          string                       `json:"canonical_summary_path"`
	BundleBootstrapSummaryPath    string                       `json:"bundle_bootstrap_summary_path"`
	CanonicalBootstrapSummaryPath string                       `json:"canonical_bootstrap_summary_path"`
	ValidationPackPath            string                       `json:"validation_pack_path"`
	ConfigurationState            string                       `json:"configuration_state"`
	BootstrapSummary              brokerBootstrapReviewSummary `json:"bootstrap_summary"`
	BootstrapReady                bool                         `json:"bootstrap_ready"`
	RuntimePosture                string                       `json:"runtime_posture"`
	LiveAdapterImplemented        bool                         `json:"live_adapter_implemented"`
	ProofBoundary                 string                       `json:"proof_boundary"`
	ValidationErrors              []string                     `json:"validation_errors"`
	ConfigCompleteness            struct {
		Driver        bool `json:"driver"`
		URLs          bool `json:"urls"`
		Topic         bool `json:"topic"`
		ConsumerGroup bool `json:"consumer_group"`
	} `json:"config_completeness"`
	Status string `json:"status"`
	Reason string `json:"reason"`
}

func TestBrokerBundleSummaryStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	canonicalSummaryPath := filepath.Join(repoRoot, "docs", "reports", "broker-validation-summary.json")
	bundleSummaryPath := filepath.Join(repoRoot, "docs", "reports", "live-validation-runs", "20260316T140138Z", "broker-validation-summary.json")
	canonicalBootstrapPath := filepath.Join(repoRoot, "docs", "reports", "broker-bootstrap-review-summary.json")
	bundleBootstrapPath := filepath.Join(repoRoot, "docs", "reports", "live-validation-runs", "20260316T140138Z", "broker-bootstrap-review-summary.json")

	var canonicalSummary brokerValidationSummary
	readJSONFile(t, canonicalSummaryPath, &canonicalSummary)
	var bundleSummary brokerValidationSummary
	readJSONFile(t, bundleSummaryPath, &bundleSummary)
	if !reflect.DeepEqual(bundleSummary, canonicalSummary) {
		t.Fatalf("bundled broker summary drifted from canonical summary: bundled=%+v canonical=%+v", bundleSummary, canonicalSummary)
	}

	var canonicalBootstrap brokerBootstrapReviewSummary
	readJSONFile(t, canonicalBootstrapPath, &canonicalBootstrap)
	var bundleBootstrap brokerBootstrapReviewSummary
	readJSONFile(t, bundleBootstrapPath, &bundleBootstrap)
	if !reflect.DeepEqual(bundleBootstrap, canonicalBootstrap) {
		t.Fatalf("bundled broker bootstrap summary drifted from canonical bootstrap summary: bundled=%+v canonical=%+v", bundleBootstrap, canonicalBootstrap)
	}

	if !reflect.DeepEqual(bundleSummary.BootstrapSummary, bundleBootstrap) {
		t.Fatalf("bundled broker summary embedded bootstrap drifted from bundled bootstrap summary: summary=%+v bootstrap=%+v", bundleSummary.BootstrapSummary, bundleBootstrap)
	}
}
