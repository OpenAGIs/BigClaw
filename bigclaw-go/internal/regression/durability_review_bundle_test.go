package regression

import (
	"path/filepath"
	"strings"
	"testing"
)

type durabilityReviewScorecard struct {
	Status            string `json:"status"`
	RolloutReady      bool   `json:"rollout_ready"`
	CurrentBackend    string `json:"current_backend"`
	TargetBackend     string `json:"target_backend"`
	ReplicationFactor int    `json:"replication_factor"`
	ReadyChecks       int    `json:"ready_checks"`
	BlockedChecks     int    `json:"blocked_checks"`
	ReadyEvidence     int    `json:"ready_evidence"`
	BlockedEvidence   int    `json:"blocked_evidence"`
	Evidence          []struct {
		Name      string   `json:"name"`
		Status    string   `json:"status"`
		Artifacts []string `json:"artifacts"`
		Detail    string   `json:"detail"`
	} `json:"evidence"`
	Checks []struct {
		Name               string   `json:"name"`
		Status             string   `json:"status"`
		SupportingEvidence []string `json:"supporting_evidence"`
		Blockers           []string `json:"blockers"`
	} `json:"checks"`
	Blockers    []string `json:"blockers"`
	NextActions []string `json:"next_actions"`
}

type durabilityBrokerValidationSummary struct {
	BundleSummaryPath             string `json:"bundle_summary_path"`
	CanonicalSummaryPath          string `json:"canonical_summary_path"`
	BundleBootstrapSummaryPath    string `json:"bundle_bootstrap_summary_path"`
	CanonicalBootstrapSummaryPath string `json:"canonical_bootstrap_summary_path"`
	ValidationPackPath            string `json:"validation_pack_path"`
	ConfigurationState            string `json:"configuration_state"`
	BootstrapReady                bool   `json:"bootstrap_ready"`
	RuntimePosture                string `json:"runtime_posture"`
	LiveAdapterImplemented        bool   `json:"live_adapter_implemented"`
	ProofBoundary                 string `json:"proof_boundary"`
	Status                        string `json:"status"`
	Reason                        string `json:"reason"`
	ConfigCompleteness            struct {
		Driver        bool `json:"driver"`
		URLs          bool `json:"urls"`
		Topic         bool `json:"topic"`
		ConsumerGroup bool `json:"consumer_group"`
	} `json:"config_completeness"`
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
			Ready bool `json:"ready"`
		} `json:"broker_bootstrap"`
		ValidationErrors []string `json:"validation_errors"`
	} `json:"bootstrap_summary"`
	ValidationErrors []string `json:"validation_errors"`
}

type durabilityProofSummary struct {
	Ticket               string `json:"ticket"`
	Status               string `json:"status"`
	ProofFamily          string `json:"proof_family"`
	SummarySchemaVersion string `json:"summary_schema_version"`
	FocusScenarios       []struct {
		ScenarioID string `json:"scenario_id"`
	} `json:"focus_scenarios"`
	Summary map[string]any `json:"summary"`
}

type durabilityAmbiguousPublishProofSummary struct {
	Ticket                string `json:"ticket"`
	Track                 string `json:"track"`
	Status                string `json:"status"`
	ClassificationSummary []struct {
		Outcome string `json:"outcome"`
	} `json:"classification_summary"`
	ReviewerChecks []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"reviewer_checks"`
	ReviewSurfaces struct {
		DistributedExport  string `json:"distributed_export"`
		DistributedPayload string `json:"distributed_payload"`
		ControlCenter      string `json:"control_center"`
	} `json:"review_surfaces"`
}

func TestDurabilityReviewBundleStaysAligned(t *testing.T) {
	root := repoRoot(t)

	var scorecard durabilityReviewScorecard
	readJSONFile(t, filepath.Join(root, "docs", "reports", "broker-durability-rollout-scorecard.json"), &scorecard)

	var validationSummary durabilityBrokerValidationSummary
	readJSONFile(t, filepath.Join(root, "docs", "reports", "broker-validation-summary.json"), &validationSummary)

	var checkpointProof durabilityProofSummary
	readJSONFile(t, filepath.Join(root, "docs", "reports", "broker-checkpoint-fencing-proof-summary.json"), &checkpointProof)

	var retentionProof durabilityProofSummary
	readJSONFile(t, filepath.Join(root, "docs", "reports", "broker-retention-boundary-proof-summary.json"), &retentionProof)

	var ambiguousProof durabilityAmbiguousPublishProofSummary
	readJSONFile(t, filepath.Join(root, "docs", "reports", "ambiguous-publish-outcome-proof-summary.json"), &ambiguousProof)

	if scorecard.Status != "blocked" ||
		scorecard.RolloutReady ||
		scorecard.CurrentBackend != "memory" ||
		scorecard.TargetBackend != "broker_replicated" ||
		scorecard.ReplicationFactor != 3 ||
		scorecard.ReadyChecks != 0 ||
		scorecard.BlockedChecks != 4 ||
		scorecard.ReadyEvidence != 3 ||
		scorecard.BlockedEvidence != 1 {
		t.Fatalf("unexpected durability rollout scorecard: %+v", scorecard)
	}
	if len(scorecard.Evidence) != 4 || len(scorecard.Checks) != 4 || len(scorecard.Blockers) != 2 || len(scorecard.NextActions) != 2 {
		t.Fatalf("unexpected durability review bundle sizes: evidence=%d checks=%d blockers=%d actions=%d", len(scorecard.Evidence), len(scorecard.Checks), len(scorecard.Blockers), len(scorecard.NextActions))
	}
	evidenceByName := map[string]string{}
	for _, evidence := range scorecard.Evidence {
		evidenceByName[evidence.Name] = evidence.Status
		if len(evidence.Artifacts) == 0 || strings.TrimSpace(evidence.Detail) == "" {
			t.Fatalf("unexpected durability evidence payload: %+v", evidence)
		}
	}
	if evidenceByName["debug_and_control_plane_surface"] != "ready" ||
		evidenceByName["replay_and_failover_validation"] != "ready" ||
		evidenceByName["operator_rollout_contract"] != "ready" ||
		evidenceByName["broker_bootstrap_config"] != "blocked" {
		t.Fatalf("unexpected durability evidence statuses: %+v", evidenceByName)
	}
	for _, check := range scorecard.Checks {
		if check.Status != "blocked" || len(check.SupportingEvidence) == 0 || len(check.Blockers) != 2 {
			t.Fatalf("unexpected durability rollout check: %+v", check)
		}
	}
	if !containsDurabilitySnippet(scorecard.Blockers, "current backend memory does not yet match the replicated target broker_replicated") ||
		!containsDurabilitySnippet(scorecard.NextActions, "BIGCLAW_EVENT_LOG_BROKER_DRIVER") {
		t.Fatalf("unexpected durability blockers/actions: blockers=%+v actions=%+v", scorecard.Blockers, scorecard.NextActions)
	}

	if validationSummary.CanonicalSummaryPath != "docs/reports/broker-validation-summary.json" ||
		validationSummary.BundleSummaryPath != "docs/reports/live-validation-runs/20260316T140138Z/broker-validation-summary.json" ||
		validationSummary.CanonicalBootstrapSummaryPath != "docs/reports/broker-bootstrap-review-summary.json" ||
		validationSummary.BundleBootstrapSummaryPath != "docs/reports/live-validation-runs/20260316T140138Z/broker-bootstrap-review-summary.json" ||
		validationSummary.ValidationPackPath != "docs/reports/broker-failover-fault-injection-validation-pack.md" ||
		validationSummary.ConfigurationState != "not_configured" ||
		validationSummary.BootstrapReady ||
		validationSummary.RuntimePosture != "contract_only" ||
		validationSummary.LiveAdapterImplemented ||
		validationSummary.Status != "skipped" ||
		validationSummary.Reason != "not_configured" {
		t.Fatalf("unexpected broker validation summary: %+v", validationSummary)
	}
	if !strings.Contains(validationSummary.ProofBoundary, "pre-adapter contract surface") ||
		validationSummary.ConfigCompleteness.Driver ||
		validationSummary.ConfigCompleteness.URLs ||
		validationSummary.ConfigCompleteness.Topic ||
		validationSummary.ConfigCompleteness.ConsumerGroup ||
		len(validationSummary.ValidationErrors) == 0 {
		t.Fatalf("unexpected broker validation config posture: %+v", validationSummary)
	}
	if validationSummary.BootstrapSummary.EventLogBackend != "memory" ||
		validationSummary.BootstrapSummary.TargetBackend != "broker_replicated" ||
		validationSummary.BootstrapSummary.Ready ||
		validationSummary.BootstrapSummary.RuntimePosture != "contract_only" ||
		validationSummary.BootstrapSummary.LiveAdapterImplemented ||
		validationSummary.BootstrapSummary.BrokerBootstrap.Ready {
		t.Fatalf("unexpected nested broker bootstrap summary: %+v", validationSummary.BootstrapSummary)
	}

	assertDurabilityProofSummary(t, checkpointProof, "checkpoint_fencing", []string{"BF-03", "BF-04", "BF-08"})
	assertDurabilityProofSummary(t, retentionProof, "retention_boundary", []string{"BF-07"})

	if ambiguousProof.Ticket != "OPE-260" ||
		ambiguousProof.Track != "BIG-PAR-104" ||
		ambiguousProof.Status != "repo-proof-summary" ||
		len(ambiguousProof.ClassificationSummary) != 3 ||
		len(ambiguousProof.ReviewerChecks) != 3 ||
		ambiguousProof.ReviewSurfaces.DistributedExport != "/v2/reports/distributed/export" ||
		ambiguousProof.ReviewSurfaces.DistributedPayload != "/v2/reports/distributed" ||
		ambiguousProof.ReviewSurfaces.ControlCenter != "/v2/control-center" {
		t.Fatalf("unexpected ambiguous publish proof summary: %+v", ambiguousProof)
	}
	if ambiguousProof.ClassificationSummary[0].Outcome != "committed" ||
		ambiguousProof.ClassificationSummary[1].Outcome != "rejected" ||
		ambiguousProof.ClassificationSummary[2].Outcome != "unknown_commit" {
		t.Fatalf("unexpected ambiguous publish classification ordering: %+v", ambiguousProof.ClassificationSummary)
	}
	for _, check := range ambiguousProof.ReviewerChecks {
		if check.Status != "pass" {
			t.Fatalf("expected ambiguous publish reviewer checks to pass, got %+v", ambiguousProof.ReviewerChecks)
		}
	}
}

func TestDurabilityReviewBundleDocsStayAligned(t *testing.T) {
	root := repoRoot(t)

	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/reports/replicated-event-log-durability-rollout-contract.md",
			substrings: []string{
				"docs/reports/broker-validation-summary.json",
				"docs/reports/broker-durability-rollout-scorecard.json",
				"docs/reports/ambiguous-publish-outcome-proof-summary.json",
				"docs/reports/broker-checkpoint-fencing-proof-summary.json",
				"docs/reports/broker-retention-boundary-proof-summary.json",
				"GET /debug/status",
				"/metrics",
				"event_durability_rollout",
			},
		},
		{
			path: "docs/reports/event-bus-reliability-report.md",
			substrings: []string{
				"docs/reports/broker-durability-rollout-scorecard.json",
				"docs/reports/durability-rollout-scorecard.json",
				"docs/reports/broker-checkpoint-fencing-proof-summary.json",
				"docs/reports/broker-retention-boundary-proof-summary.json",
				"event_durability_rollout",
			},
		},
	}

	for _, tc := range cases {
		body := readRepoFile(t, root, tc.path)
		for _, needle := range tc.substrings {
			if !strings.Contains(body, needle) {
				t.Fatalf("%s missing substring %q", tc.path, needle)
			}
		}
	}
}

func assertDurabilityProofSummary(t *testing.T, summary durabilityProofSummary, family string, scenarios []string) {
	t.Helper()
	if summary.Ticket != "OPE-230" || summary.Status != "passed" || summary.ProofFamily != family || summary.SummarySchemaVersion != "2026-03-17" {
		t.Fatalf("unexpected durability proof summary: %+v", summary)
	}
	if len(summary.FocusScenarios) != len(scenarios) {
		t.Fatalf("unexpected focus scenario count for %s: %+v", family, summary.FocusScenarios)
	}
	for i, scenario := range scenarios {
		if summary.FocusScenarios[i].ScenarioID != scenario {
			t.Fatalf("unexpected focus scenario ordering for %s: %+v", family, summary.FocusScenarios)
		}
	}
}

func containsDurabilitySnippet(values []string, needle string) bool {
	for _, value := range values {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
