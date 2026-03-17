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
			TaskSucceeded           bool   `json:"task_succeeded"`
			RemoteReplayBackend     string `json:"remote_replay_backend"`
			CheckpointAcknowledged  bool   `json:"checkpoint_acknowledged"`
			CheckpointResetRecorded bool   `json:"checkpoint_reset_recorded"`
			RetentionBoundaryVisible bool  `json:"retention_boundary_visible"`
			TakeoverConflictRejected bool  `json:"takeover_conflict_rejected"`
			TakeoverAfterExpiry      bool  `json:"takeover_after_expiry"`
			StaleWriterRejected      bool  `json:"stale_writer_rejected"`
		} `json:"summary"`
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
	if report.Ticket != "BIG-PAR-097" || report.Status != "validated" {
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
			substrings: []string{"external_store_validation.py", "external-store-validation-report.json", "remote HTTP service boundary"},
		},
		{
			path: "docs/reports/replay-retention-semantics-report.md",
			substrings: []string{"external-store-validation-report.json", "external-store service boundary"},
		},
		{
			path: "docs/reports/epic-closure-readiness-report.md",
			substrings: []string{"external-store-validation-report.json", "remote HTTP event-log replay"},
		},
		{
			path: "docs/reports/review-readiness.md",
			substrings: []string{"external-store-validation-report.json", "remote HTTP event-log service boundary"},
		},
		{
			path: "docs/reports/issue-coverage.md",
			substrings: []string{"external-store-validation-report.json", "remote HTTP event-log service boundary"},
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
