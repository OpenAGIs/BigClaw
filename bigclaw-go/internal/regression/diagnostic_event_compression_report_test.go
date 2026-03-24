package regression

import (
	"path/filepath"
	"testing"
)

func TestDiagnosticEventCompressionReportStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	reportPath := filepath.Join(repoRoot, "docs", "reports", "distributed-diagnostic-event-compression-report.json")

	var report struct {
		Ticket  string `json:"ticket"`
		Track   string `json:"track"`
		Title   string `json:"title"`
		Status  string `json:"status"`
		Summary struct {
			SourceEventCount      int     `json:"source_event_count"`
			CompressedEventCount  int     `json:"compressed_event_count"`
			CompressionRatio      float64 `json:"compression_ratio"`
			TraceCount            int     `json:"trace_count"`
			ReplayValidatedTraces int     `json:"replay_validated_traces"`
			DeterministicReplays  int     `json:"deterministic_replays"`
		} `json:"summary"`
		CompressionGroups []struct {
			EventType           string  `json:"event_type"`
			CompressionStrategy string  `json:"compression_strategy"`
			CompressionRatio    float64 `json:"compression_ratio"`
		} `json:"compression_groups"`
		ReplayValidation struct {
			Backend              string   `json:"backend"`
			Validated            bool     `json:"validated"`
			SampledTraceCount    int      `json:"sampled_trace_count"`
			ExactMatchTraceCount int      `json:"exact_match_trace_count"`
			MismatchCount        int      `json:"mismatch_count"`
			ReplayWindowSeconds  int      `json:"replay_window_seconds"`
			Artifacts            []string `json:"artifacts"`
		} `json:"replay_validation"`
		Artifacts     []string `json:"artifacts"`
		ReviewerLinks []string `json:"reviewer_links"`
		Limitations   []string `json:"limitations"`
	}
	readJSONFile(t, reportPath, &report)

	if report.Ticket != "BIGCLAW-187" || report.Track != "BIG-vNext-017" || report.Title != "Distributed diagnostic event compression and replay validation" || report.Status != "validated" {
		t.Fatalf("unexpected report metadata: %+v", report)
	}
	if report.Summary.SourceEventCount != 48 || report.Summary.CompressedEventCount != 17 || report.Summary.CompressionRatio != 0.354 || report.Summary.TraceCount != 5 || report.Summary.ReplayValidatedTraces != 5 || report.Summary.DeterministicReplays != 5 {
		t.Fatalf("unexpected report summary: %+v", report.Summary)
	}
	if len(report.CompressionGroups) != 4 || report.CompressionGroups[0].EventType != "scheduler.routed" || report.CompressionGroups[0].CompressionStrategy != "adjacent_reason_window" {
		t.Fatalf("unexpected compression groups: %+v", report.CompressionGroups)
	}
	if report.ReplayValidation.Backend != "http_remote_service" || !report.ReplayValidation.Validated || report.ReplayValidation.SampledTraceCount != 5 || report.ReplayValidation.ExactMatchTraceCount != 5 || report.ReplayValidation.MismatchCount != 0 || report.ReplayValidation.ReplayWindowSeconds != 120 {
		t.Fatalf("unexpected replay validation summary: %+v", report.ReplayValidation)
	}
	if len(report.Artifacts) != 3 || len(report.ReplayValidation.Artifacts) != 2 || len(report.ReviewerLinks) != 3 || len(report.Limitations) != 2 {
		t.Fatalf("unexpected reviewer metadata: artifacts=%v replay=%v links=%v limitations=%v", report.Artifacts, report.ReplayValidation.Artifacts, report.ReviewerLinks, report.Limitations)
	}
}
