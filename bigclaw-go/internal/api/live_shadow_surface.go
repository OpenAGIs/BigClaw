package api

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	liveShadowSummaryPath                  = "docs/reports/live-shadow-summary.json"
	liveShadowMirrorScorecardPath          = "docs/reports/live-shadow-mirror-scorecard.json"
	migrationReadinessReportPath           = "docs/reports/migration-readiness-report.md"
	liveShadowIndexPath                    = "docs/reports/live-shadow-index.md"
	liveShadowComparisonFollowUpDigestPath = "docs/reports/live-shadow-comparison-follow-up-digest.md"
	rollbackTriggerSurfacePath             = "docs/reports/rollback-trigger-surface.json"
)

type liveShadowMirrorSurface struct {
	ReportPath              string                           `json:"report_path"`
	CanonicalSummaryPath    string                           `json:"canonical_summary_path,omitempty"`
	GeneratedAt             string                           `json:"generated_at,omitempty"`
	Ticket                  string                           `json:"ticket,omitempty"`
	Title                   string                           `json:"title,omitempty"`
	Status                  string                           `json:"status,omitempty"`
	Severity                string                           `json:"severity,omitempty"`
	BundlePath              string                           `json:"bundle_path,omitempty"`
	SummaryPath             string                           `json:"summary_path,omitempty"`
	LatestEvidenceTimestamp string                           `json:"latest_evidence_timestamp,omitempty"`
	ReviewerLinks           []string                         `json:"reviewer_links,omitempty"`
	Summary                 liveShadowMirrorSummary          `json:"summary"`
	Freshness               []liveShadowEvidenceFreshness    `json:"freshness,omitempty"`
	CutoverCheckpoints      []liveShadowCutoverCheckpoint    `json:"cutover_checkpoints,omitempty"`
	RollbackTriggerSurface  liveShadowRollbackTriggerSurface `json:"rollback_trigger_surface"`
	Limitations             []string                         `json:"limitations,omitempty"`
	FutureWork              []string                         `json:"future_work,omitempty"`
	Error                   string                           `json:"error,omitempty"`
}

type liveShadowMirrorSummary struct {
	TotalEvidenceRuns         int    `json:"total_evidence_runs"`
	ParityOKCount             int    `json:"parity_ok_count"`
	DriftDetectedCount        int    `json:"drift_detected_count"`
	MatrixTotal               int    `json:"matrix_total"`
	MatrixMatched             int    `json:"matrix_matched"`
	MatrixMismatched          int    `json:"matrix_mismatched"`
	FreshInputs               int    `json:"fresh_inputs"`
	StaleInputs               int    `json:"stale_inputs"`
	CorpusCoveragePresent     bool   `json:"corpus_coverage_present"`
	CorpusUncoveredSliceCount int    `json:"corpus_uncovered_slice_count"`
	LatestEvidenceTimestamp   string `json:"latest_evidence_timestamp,omitempty"`
}

type liveShadowEvidenceFreshness struct {
	Name                    string  `json:"name"`
	ReportPath              string  `json:"report_path,omitempty"`
	LatestEvidenceTimestamp string  `json:"latest_evidence_timestamp,omitempty"`
	AgeHours                float64 `json:"age_hours,omitempty"`
	FreshnessSLOHours       int     `json:"freshness_slo_hours,omitempty"`
	Status                  string  `json:"status,omitempty"`
}

type liveShadowCutoverCheckpoint struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

type liveShadowRollbackTriggerSurface struct {
	Status                   string                         `json:"status,omitempty"`
	AutomationBoundary       string                         `json:"automation_boundary,omitempty"`
	AutomatedRollbackTrigger bool                           `json:"automated_rollback_trigger"`
	Distinctions             liveShadowRollbackDistinctions `json:"distinctions"`
	SummaryPath              string                         `json:"summary_path,omitempty"`
}

type liveShadowRollbackDistinctions struct {
	Blockers        int `json:"blockers"`
	Warnings        int `json:"warnings"`
	ManualOnlyPaths int `json:"manual_only_paths"`
}

type liveShadowSummaryDocument struct {
	RunID                   string `json:"run_id"`
	GeneratedAt             string `json:"generated_at"`
	Status                  string `json:"status"`
	Severity                string `json:"severity"`
	BundlePath              string `json:"bundle_path"`
	SummaryPath             string `json:"summary_path"`
	LatestEvidenceTimestamp string `json:"latest_evidence_timestamp"`
	Artifacts               struct {
		ShadowCompareReportPath    string `json:"shadow_compare_report_path"`
		ShadowMatrixReportPath     string `json:"shadow_matrix_report_path"`
		LiveShadowScorecardPath    string `json:"live_shadow_scorecard_path"`
		RollbackTriggerSurfacePath string `json:"rollback_trigger_surface_path"`
	} `json:"artifacts"`
	Summary            liveShadowMirrorSummary          `json:"summary"`
	Freshness          []liveShadowEvidenceFreshness    `json:"freshness"`
	RollbackTrigger    liveShadowRollbackTriggerSurface `json:"rollback_trigger_surface"`
	CutoverCheckpoints []liveShadowCutoverCheckpoint    `json:"cutover_checkpoints"`
}

type liveShadowScorecardDocument struct {
	GeneratedAt    string `json:"generated_at"`
	Ticket         string `json:"ticket"`
	Title          string `json:"title"`
	Status         string `json:"status"`
	EvidenceInputs struct {
		ShadowCompareReportPath string `json:"shadow_compare_report_path"`
		ShadowMatrixReportPath  string `json:"shadow_matrix_report_path"`
		GeneratorScript         string `json:"generator_script"`
	} `json:"evidence_inputs"`
	Summary            liveShadowMirrorSummary       `json:"summary"`
	Freshness          []liveShadowEvidenceFreshness `json:"freshness"`
	CutoverCheckpoints []liveShadowCutoverCheckpoint `json:"cutover_checkpoints"`
	Limitations        []string                      `json:"limitations"`
	FutureWork         []string                      `json:"future_work"`
}

type migrationReviewPack struct {
	Status                    string                  `json:"status"`
	ReadinessReportPath       string                  `json:"readiness_report_path"`
	ScorecardPath             string                  `json:"scorecard_path"`
	CanonicalSummaryPath      string                  `json:"canonical_summary_path"`
	SummaryPath               string                  `json:"summary_path"`
	IndexPath                 string                  `json:"index_path"`
	FollowUpDigestPath        string                  `json:"follow_up_digest_path"`
	RollbackTriggerPath       string                  `json:"rollback_trigger_path"`
	ReviewerLinks             []string                `json:"reviewer_links,omitempty"`
	LiveShadowMirrorScorecard liveShadowMirrorSurface `json:"live_shadow_mirror_scorecard"`
}

func liveShadowMirrorPayload() liveShadowMirrorSurface {
	surface := liveShadowMirrorSurface{
		ReportPath:           liveShadowMirrorScorecardPath,
		CanonicalSummaryPath: liveShadowSummaryPath,
	}
	summaryPath := resolveRepoRelativePath(liveShadowSummaryPath)
	if summaryPath == "" {
		surface.Status = "unavailable"
		surface.Error = "summary path could not be resolved"
		return surface
	}
	scorecardPath := resolveRepoRelativePath(liveShadowMirrorScorecardPath)
	if scorecardPath == "" {
		surface.Status = "unavailable"
		surface.Error = "scorecard path could not be resolved"
		return surface
	}

	summaryContents, err := os.ReadFile(summaryPath)
	if err != nil {
		surface.Status = "unavailable"
		surface.Error = err.Error()
		return surface
	}
	var summaryDoc liveShadowSummaryDocument
	if err := json.Unmarshal(summaryContents, &summaryDoc); err != nil {
		surface.Status = "invalid"
		surface.Error = fmt.Sprintf("decode %s: %v", liveShadowSummaryPath, err)
		return surface
	}

	scorecardContents, err := os.ReadFile(scorecardPath)
	if err != nil {
		surface.Status = "unavailable"
		surface.Error = err.Error()
		return surface
	}
	var scorecardDoc liveShadowScorecardDocument
	if err := json.Unmarshal(scorecardContents, &scorecardDoc); err != nil {
		surface.Status = "invalid"
		surface.Error = fmt.Sprintf("decode %s: %v", liveShadowMirrorScorecardPath, err)
		return surface
	}

	surface.GeneratedAt = firstNonEmpty(scorecardDoc.GeneratedAt, summaryDoc.GeneratedAt)
	surface.Ticket = scorecardDoc.Ticket
	surface.Title = scorecardDoc.Title
	surface.Status = firstNonEmpty(summaryDoc.Status, scorecardDoc.Status)
	surface.Severity = summaryDoc.Severity
	surface.BundlePath = summaryDoc.BundlePath
	surface.SummaryPath = summaryDoc.SummaryPath
	surface.LatestEvidenceTimestamp = firstNonEmpty(summaryDoc.LatestEvidenceTimestamp, scorecardDoc.Summary.LatestEvidenceTimestamp)
	surface.Summary = scorecardDoc.Summary
	surface.Summary.LatestEvidenceTimestamp = firstNonEmpty(scorecardDoc.Summary.LatestEvidenceTimestamp, summaryDoc.LatestEvidenceTimestamp)
	if surface.Summary.TotalEvidenceRuns == 0 {
		surface.Summary.TotalEvidenceRuns = summaryDoc.Summary.TotalEvidenceRuns
	}
	if surface.Summary.ParityOKCount == 0 {
		surface.Summary.ParityOKCount = summaryDoc.Summary.ParityOKCount
	}
	if surface.Summary.DriftDetectedCount == 0 {
		surface.Summary.DriftDetectedCount = summaryDoc.Summary.DriftDetectedCount
	}
	if surface.Summary.MatrixTotal == 0 {
		surface.Summary.MatrixTotal = summaryDoc.Summary.MatrixTotal
	}
	if surface.Summary.MatrixMismatched == 0 {
		surface.Summary.MatrixMismatched = summaryDoc.Summary.MatrixMismatched
	}
	if surface.Summary.FreshInputs == 0 {
		surface.Summary.FreshInputs = summaryDoc.Summary.FreshInputs
	}
	if surface.Summary.StaleInputs == 0 {
		surface.Summary.StaleInputs = summaryDoc.Summary.StaleInputs
	}
	surface.Freshness = append([]liveShadowEvidenceFreshness(nil), scorecardDoc.Freshness...)
	if len(surface.Freshness) == 0 {
		surface.Freshness = append([]liveShadowEvidenceFreshness(nil), summaryDoc.Freshness...)
	}
	surface.CutoverCheckpoints = append([]liveShadowCutoverCheckpoint(nil), scorecardDoc.CutoverCheckpoints...)
	if len(surface.CutoverCheckpoints) == 0 {
		surface.CutoverCheckpoints = append([]liveShadowCutoverCheckpoint(nil), summaryDoc.CutoverCheckpoints...)
	}
	surface.RollbackTriggerSurface = summaryDoc.RollbackTrigger
	surface.Limitations = append([]string(nil), scorecardDoc.Limitations...)
	surface.FutureWork = append([]string(nil), scorecardDoc.FutureWork...)
	surface.ReviewerLinks = liveShadowReviewerLinks(surface, summaryDoc, scorecardDoc)
	return surface
}

func liveShadowMirrorScorecardPayload() liveShadowMirrorSurface {
	return liveShadowMirrorPayload()
}

func buildMigrationReviewPack() migrationReviewPack {
	surface := liveShadowMirrorPayload()
	return migrationReviewPack{
		Status:                    firstNonEmpty(surface.Status, "unavailable"),
		ReadinessReportPath:       migrationReadinessReportPath,
		ScorecardPath:             surface.ReportPath,
		CanonicalSummaryPath:      surface.CanonicalSummaryPath,
		SummaryPath:               surface.SummaryPath,
		IndexPath:                 liveShadowIndexPath,
		FollowUpDigestPath:        liveShadowComparisonFollowUpDigestPath,
		RollbackTriggerPath:       firstNonEmpty(surface.RollbackTriggerSurface.SummaryPath, rollbackTriggerSurfacePath),
		ReviewerLinks:             append([]string(nil), surface.ReviewerLinks...),
		LiveShadowMirrorScorecard: surface,
	}
}

func liveShadowReviewerLinks(surface liveShadowMirrorSurface, summaryDoc liveShadowSummaryDocument, scorecardDoc liveShadowScorecardDocument) []string {
	seen := make(map[string]struct{})
	links := make([]string, 0, 8)
	appendLink := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		links = append(links, value)
	}
	appendLink(surface.CanonicalSummaryPath)
	appendLink(surface.ReportPath)
	appendLink(migrationReadinessReportPath)
	appendLink(liveShadowIndexPath)
	appendLink(liveShadowComparisonFollowUpDigestPath)
	appendLink(summaryDoc.Artifacts.ShadowCompareReportPath)
	appendLink(summaryDoc.Artifacts.ShadowMatrixReportPath)
	appendLink(scorecardDoc.EvidenceInputs.GeneratorScript)
	appendLink(firstNonEmpty(surface.RollbackTriggerSurface.SummaryPath, summaryDoc.Artifacts.RollbackTriggerSurfacePath))
	if len(links) <= 2 {
		return links
	}
	first := links[0]
	last := links[len(links)-1]
	middle := append([]string(nil), links[1:len(links)-1]...)
	sort.Strings(middle)
	ordered := []string{first}
	ordered = append(ordered, middle...)
	ordered = append(ordered, last)
	return ordered
}
