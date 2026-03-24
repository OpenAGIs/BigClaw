package api

import (
	"encoding/json"
	"fmt"
	"os"
)

const diagnosticEventCompressionSurfacePath = "docs/reports/distributed-diagnostic-event-compression-report.json"

type diagnosticEventCompressionSurface struct {
	ReportPath        string                            `json:"report_path"`
	GeneratedAt       string                            `json:"generated_at,omitempty"`
	Ticket            string                            `json:"ticket,omitempty"`
	Track             string                            `json:"track,omitempty"`
	Title             string                            `json:"title,omitempty"`
	Status            string                            `json:"status,omitempty"`
	Summary           diagnosticEventCompressionSummary `json:"summary"`
	CompressionGroups []diagnosticCompressionGroup      `json:"compression_groups,omitempty"`
	ReplayValidation  diagnosticReplayValidationSummary `json:"replay_validation"`
	Artifacts         []string                          `json:"artifacts,omitempty"`
	ReviewerLinks     []string                          `json:"reviewer_links,omitempty"`
	Limitations       []string                          `json:"limitations,omitempty"`
	Error             string                            `json:"error,omitempty"`
}

type diagnosticEventCompressionSummary struct {
	SourceEventCount      int     `json:"source_event_count"`
	CompressedEventCount  int     `json:"compressed_event_count"`
	CompressionRatio      float64 `json:"compression_ratio"`
	TraceCount            int     `json:"trace_count"`
	ReplayValidatedTraces int     `json:"replay_validated_traces"`
	DeterministicReplays  int     `json:"deterministic_replays"`
}

type diagnosticCompressionGroup struct {
	EventType           string  `json:"event_type"`
	SourceCount         int     `json:"source_count"`
	CompressedCount     int     `json:"compressed_count"`
	CompressionRatio    float64 `json:"compression_ratio"`
	CompressionStrategy string  `json:"compression_strategy"`
}

type diagnosticReplayValidationSummary struct {
	Backend                string   `json:"backend"`
	Validated              bool     `json:"validated"`
	SampledTraceCount      int      `json:"sampled_trace_count"`
	ExactMatchTraceCount   int      `json:"exact_match_trace_count"`
	MismatchCount          int      `json:"mismatch_count"`
	ReplayWindowSeconds    int      `json:"replay_window_seconds"`
	LatestValidatedEventID string   `json:"latest_validated_event_id,omitempty"`
	Artifacts              []string `json:"artifacts,omitempty"`
}

func diagnosticEventCompressionSurfacePayload() diagnosticEventCompressionSurface {
	surface := diagnosticEventCompressionSurface{ReportPath: diagnosticEventCompressionSurfacePath}
	reportPath := resolveRepoRelativePath(diagnosticEventCompressionSurfacePath)
	if reportPath == "" {
		surface.Status = "unavailable"
		surface.Error = "report path could not be resolved"
		return surface
	}
	contents, err := os.ReadFile(reportPath)
	if err != nil {
		surface.Status = "unavailable"
		surface.Error = err.Error()
		return surface
	}
	if err := json.Unmarshal(contents, &surface); err != nil {
		surface.Status = "invalid"
		surface.Error = fmt.Sprintf("decode %s: %v", diagnosticEventCompressionSurfacePath, err)
		return surface
	}
	surface.ReportPath = diagnosticEventCompressionSurfacePath
	return surface
}
