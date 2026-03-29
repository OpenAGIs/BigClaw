package regression

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestExternalStoreValidationReportStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	reportPath := filepath.Join(repoRoot, "docs", "reports", "external-store-validation-report.json")

	var report struct {
		Ticket string `json:"ticket"`
		Title  string `json:"title"`
		Status string `json:"status"`
		Lane   struct {
			ServiceBackend         string `json:"service_backend"`
			RuntimeEventLogBackend string `json:"runtime_event_log_backend"`
			QueueBackend           string `json:"queue_backend"`
			SubscriberLeaseBackend string `json:"subscriber_lease_backend"`
			NodeCount              int    `json:"node_count"`
		} `json:"lane"`
		Summary struct {
			TaskSucceeded            bool   `json:"task_succeeded"`
			RemoteReplayBackend      string `json:"remote_replay_backend"`
			CheckpointAcknowledged   bool   `json:"checkpoint_acknowledged"`
			CheckpointResetRecorded  bool   `json:"checkpoint_reset_recorded"`
			RetentionBoundaryVisible bool   `json:"retention_boundary_visible"`
			TakeoverConflictRejected bool   `json:"takeover_conflict_rejected"`
			TakeoverAfterExpiry      bool   `json:"takeover_after_expiry"`
			StaleWriterRejected      bool   `json:"stale_writer_rejected"`
		} `json:"summary"`
		BackendMatrix struct {
			StatusDefinitions map[string]string `json:"status_definitions"`
			Summary           struct {
				LiveValidatedLanes int `json:"live_validated_lanes"`
				NotConfiguredLanes int `json:"not_configured_lanes"`
				ContractOnlyLanes  int `json:"contract_only_lanes"`
			} `json:"summary"`
			Lanes []struct {
				Backend                 string   `json:"backend"`
				Role                    string   `json:"role"`
				ValidationStatus        string   `json:"validation_status"`
				ConfigurationState      string   `json:"configuration_state"`
				ProofKind               string   `json:"proof_kind"`
				ReplayBackend           string   `json:"replay_backend"`
				CheckpointBackend       string   `json:"checkpoint_backend"`
				RetentionBoundaryVisible bool    `json:"retention_boundary_visible"`
				TakeoverBackend         string   `json:"takeover_backend"`
				Reason                  string   `json:"reason"`
				ReportLinks             []string `json:"report_links"`
			} `json:"lanes"`
		} `json:"backend_matrix"`
		ReplayValidation struct {
			Backend string `json:"backend"`
			Durable bool   `json:"durable"`
		} `json:"replay_validation"`
		CheckpointValidation struct {
			SubscriberID        string `json:"subscriber_id"`
			ResetHistoryEntries int    `json:"reset_history_entries"`
		} `json:"checkpoint_validation"`
		RetentionValidation struct {
			HistoryTruncated    bool   `json:"history_truncated"`
			PersistedBoundary   bool   `json:"persisted_boundary"`
			TrimmedThroughEvent string `json:"trimmed_through_event_id"`
		} `json:"retention_validation"`
		TakeoverValidation struct {
			ConflictStatus    int    `json:"conflict_status"`
			TakeoverConsumer  string `json:"takeover_consumer"`
			TakeoverEpoch     int    `json:"takeover_epoch"`
			StaleWriterStatus int    `json:"stale_writer_status"`
			FinalConsumer     string `json:"final_lease_consumer"`
		} `json:"takeover_validation"`
	}
	readJSONFile(t, reportPath, &report)
	if report.Ticket != "BIG-PAR-102" || report.Title != "External-store validation backend matrix and broker placeholders" || report.Status != "validated" {
		t.Fatalf("unexpected external-store report metadata: %+v", report)
	}
	if report.Lane.ServiceBackend != "sqlite_event_log_service" || report.Lane.RuntimeEventLogBackend != "http_remote_service" || report.Lane.QueueBackend != "sqlite" || report.Lane.SubscriberLeaseBackend != "sqlite_shared" || report.Lane.NodeCount != 3 {
		t.Fatalf("unexpected external-store lane metadata: %+v", report.Lane)
	}
	if !report.Summary.TaskSucceeded || report.Summary.RemoteReplayBackend != "http" || !report.Summary.CheckpointAcknowledged || !report.Summary.CheckpointResetRecorded || !report.Summary.RetentionBoundaryVisible || !report.Summary.TakeoverConflictRejected || !report.Summary.TakeoverAfterExpiry || !report.Summary.StaleWriterRejected {
		t.Fatalf("unexpected external-store summary: %+v", report.Summary)
	}
	if report.ReplayValidation.Backend != "http" || !report.ReplayValidation.Durable {
		t.Fatalf("unexpected replay validation payload: %+v", report.ReplayValidation)
	}
	if report.BackendMatrix.Summary.LiveValidatedLanes != 1 || report.BackendMatrix.Summary.NotConfiguredLanes != 1 || report.BackendMatrix.Summary.ContractOnlyLanes != 1 {
		t.Fatalf("unexpected backend-matrix summary: %+v", report.BackendMatrix.Summary)
	}
	if len(report.BackendMatrix.StatusDefinitions) != 3 {
		t.Fatalf("unexpected backend-matrix status definitions: %+v", report.BackendMatrix.StatusDefinitions)
	}
	if len(report.BackendMatrix.Lanes) != 3 {
		t.Fatalf("unexpected backend-matrix lanes: %+v", report.BackendMatrix.Lanes)
	}
	if report.BackendMatrix.Lanes[0].Backend != "http_remote_service" || report.BackendMatrix.Lanes[0].ValidationStatus != "live_validated" || report.BackendMatrix.Lanes[0].ReplayBackend != "http" || !report.BackendMatrix.Lanes[0].RetentionBoundaryVisible || report.BackendMatrix.Lanes[0].TakeoverBackend != "sqlite_shared_lease" {
		t.Fatalf("unexpected http backend-matrix lane: %+v", report.BackendMatrix.Lanes[0])
	}
	if report.BackendMatrix.Lanes[1].Backend != "broker_replicated" || report.BackendMatrix.Lanes[1].ValidationStatus != "not_configured" || report.BackendMatrix.Lanes[1].Reason != "not_configured" {
		t.Fatalf("unexpected broker backend-matrix lane: %+v", report.BackendMatrix.Lanes[1])
	}
	if report.BackendMatrix.Lanes[2].Backend != "quorum_replicated" || report.BackendMatrix.Lanes[2].ValidationStatus != "contract_only" || report.BackendMatrix.Lanes[2].Reason != "contract_only" {
		t.Fatalf("unexpected quorum backend-matrix lane: %+v", report.BackendMatrix.Lanes[2])
	}
	if report.CheckpointValidation.SubscriberID != "subscriber-external-store" || report.CheckpointValidation.ResetHistoryEntries < 1 {
		t.Fatalf("unexpected checkpoint validation payload: %+v", report.CheckpointValidation)
	}
	if !report.RetentionValidation.HistoryTruncated || !report.RetentionValidation.PersistedBoundary || report.RetentionValidation.TrimmedThroughEvent != "evt-external-retention-old" {
		t.Fatalf("unexpected retention validation payload: %+v", report.RetentionValidation)
	}
	if report.TakeoverValidation.ConflictStatus != 409 || report.TakeoverValidation.TakeoverConsumer != "node-b" || report.TakeoverValidation.TakeoverEpoch != 2 || report.TakeoverValidation.StaleWriterStatus != 409 || report.TakeoverValidation.FinalConsumer != "node-b" {
		t.Fatalf("unexpected takeover validation payload: %+v", report.TakeoverValidation)
	}

	for _, tc := range []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/e2e-validation.md",
			substrings: []string{"external_store_validation.go", "external-store-validation-report.json", "backend_matrix", "not_configured", "contract_only"},
		},
		{
			path: "docs/reports/replay-retention-semantics-report.md",
			substrings: []string{"external-store-validation-report.json", "backend matrix", "http_remote_service"},
		},
		{
			path: "docs/reports/epic-closure-readiness-report.md",
			substrings: []string{"external-store-validation-report.json", "backend matrix", "broker_replicated"},
		},
		{
			path: "docs/reports/review-readiness.md",
			substrings: []string{"external-store-validation-report.json", "backend matrix", "contract_only"},
		},
		{
			path: "docs/reports/issue-coverage.md",
			substrings: []string{"external-store-validation-report.json", "backend matrix", "not_configured"},
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
