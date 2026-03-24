package api

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	parallelDiagnosticsLiveValidationIndexPath = "docs/reports/live-validation-index.json"
	parallelDiagnosticsLiveShadowIndexPath     = "docs/reports/live-shadow-index.json"
)

type parallelDiagnosticsEvidenceBundleIndex struct {
	GeneratedAt string                                  `json:"generated_at,omitempty"`
	Summary     parallelDiagnosticsEvidenceIndexSummary `json:"summary"`
	Bundles     []parallelDiagnosticsEvidenceBundle     `json:"bundles"`
}

type parallelDiagnosticsEvidenceIndexSummary struct {
	TotalBundles    int      `json:"total_bundles"`
	TotalArtifacts  int      `json:"total_artifacts"`
	BundleKinds     []string `json:"bundle_kinds,omitempty"`
	Statuses        []string `json:"statuses,omitempty"`
	Lanes           []string `json:"lanes,omitempty"`
	ReviewerLinkHit int      `json:"reviewer_link_count"`
}

type parallelDiagnosticsEvidenceBundle struct {
	ID            string   `json:"id"`
	Kind          string   `json:"kind"`
	Title         string   `json:"title"`
	Status        string   `json:"status,omitempty"`
	GeneratedAt   string   `json:"generated_at,omitempty"`
	BundlePath    string   `json:"bundle_path,omitempty"`
	SummaryPath   string   `json:"summary_path,omitempty"`
	ReportPath    string   `json:"report_path,omitempty"`
	Severity      string   `json:"severity,omitempty"`
	Lanes         []string `json:"lanes,omitempty"`
	ArtifactPaths []string `json:"artifact_paths,omitempty"`
	ReviewerLinks []string `json:"reviewer_links,omitempty"`
}

type parallelDiagnosticsEvidenceSearchResponse struct {
	Query   string                                   `json:"query,omitempty"`
	Status  string                                   `json:"status,omitempty"`
	Lane    string                                   `json:"lane,omitempty"`
	Path    string                                   `json:"path,omitempty"`
	Limit   int                                      `json:"limit"`
	Summary parallelDiagnosticsEvidenceSearchSummary `json:"summary"`
	Results []parallelDiagnosticsEvidenceSearchHit   `json:"results"`
}

type parallelDiagnosticsEvidenceSearchSummary struct {
	IndexedBundles  int `json:"indexed_bundles"`
	MatchedBundles  int `json:"matched_bundles"`
	ReturnedBundles int `json:"returned_bundles"`
}

type parallelDiagnosticsEvidenceSearchHit struct {
	Bundle        parallelDiagnosticsEvidenceBundle `json:"bundle"`
	MatchedFields []string                          `json:"matched_fields,omitempty"`
}

type parallelDiagnosticsLiveValidationIndexDocument struct {
	Latest struct {
		RunID       string `json:"run_id"`
		GeneratedAt string `json:"generated_at"`
		Status      string `json:"status"`
		BundlePath  string `json:"bundle_path"`
		SummaryPath string `json:"summary_path"`
		Local       struct {
			Enabled             bool   `json:"enabled"`
			BundleReportPath    string `json:"bundle_report_path"`
			CanonicalReportPath string `json:"canonical_report_path"`
			Status              string `json:"status"`
			StdoutPath          string `json:"stdout_path"`
			StderrPath          string `json:"stderr_path"`
			AuditLogPath        string `json:"audit_log_path"`
			ServiceLogPath      string `json:"service_log_path"`
		} `json:"local"`
		Kubernetes struct {
			Enabled             bool   `json:"enabled"`
			BundleReportPath    string `json:"bundle_report_path"`
			CanonicalReportPath string `json:"canonical_report_path"`
			Status              string `json:"status"`
			StdoutPath          string `json:"stdout_path"`
			StderrPath          string `json:"stderr_path"`
			AuditLogPath        string `json:"audit_log_path"`
			ServiceLogPath      string `json:"service_log_path"`
		} `json:"kubernetes"`
		Ray struct {
			Enabled             bool   `json:"enabled"`
			BundleReportPath    string `json:"bundle_report_path"`
			CanonicalReportPath string `json:"canonical_report_path"`
			Status              string `json:"status"`
			StdoutPath          string `json:"stdout_path"`
			StderrPath          string `json:"stderr_path"`
			AuditLogPath        string `json:"audit_log_path"`
			ServiceLogPath      string `json:"service_log_path"`
		} `json:"ray"`
		Broker struct {
			BundleSummaryPath    string `json:"bundle_summary_path"`
			CanonicalSummaryPath string `json:"canonical_summary_path"`
			ValidationPackPath   string `json:"validation_pack_path"`
			Status               string `json:"status"`
		} `json:"broker"`
		SharedQueueCompanion struct {
			Available            bool   `json:"available"`
			CanonicalReportPath  string `json:"canonical_report_path"`
			CanonicalSummaryPath string `json:"canonical_summary_path"`
			BundleReportPath     string `json:"bundle_report_path"`
			BundleSummaryPath    string `json:"bundle_summary_path"`
			Status               string `json:"status"`
		} `json:"shared_queue_companion"`
	} `json:"latest"`
	ContinuationGate struct {
		Path         string `json:"path"`
		Status       string `json:"status"`
		ReviewerPath struct {
			IndexPath  string `json:"index_path"`
			DigestPath string `json:"digest_path"`
		} `json:"reviewer_path"`
	} `json:"continuation_gate"`
}

type parallelDiagnosticsLiveShadowIndexDocument struct {
	Latest struct {
		RunID       string `json:"run_id"`
		GeneratedAt string `json:"generated_at"`
		Status      string `json:"status"`
		Severity    string `json:"severity"`
		BundlePath  string `json:"bundle_path"`
		SummaryPath string `json:"summary_path"`
		Artifacts   struct {
			ShadowCompareReportPath    string `json:"shadow_compare_report_path"`
			ShadowMatrixReportPath     string `json:"shadow_matrix_report_path"`
			LiveShadowScorecardPath    string `json:"live_shadow_scorecard_path"`
			RollbackTriggerSurfacePath string `json:"rollback_trigger_surface_path"`
		} `json:"artifacts"`
		Freshness []struct {
			ReportPath string `json:"report_path"`
			Status     string `json:"status"`
		} `json:"freshness"`
		RollbackTriggerSurface struct {
			DigestPath  string `json:"digest_path"`
			SummaryPath string `json:"summary_path"`
		} `json:"rollback_trigger_surface"`
	} `json:"latest"`
	DriftRollup struct {
		Status string `json:"status"`
	} `json:"drift_rollup"`
}

func parallelDiagnosticsEvidenceBundleIndexPayload() parallelDiagnosticsEvidenceBundleIndex {
	bundles := buildParallelDiagnosticsEvidenceBundles()
	summary := parallelDiagnosticsEvidenceIndexSummary{
		TotalBundles:    len(bundles),
		TotalArtifacts:  0,
		BundleKinds:     collectBundleKinds(bundles),
		Statuses:        collectBundleStatuses(bundles),
		Lanes:           collectBundleLanes(bundles),
		ReviewerLinkHit: 0,
	}
	generatedAt := ""
	for _, bundle := range bundles {
		summary.TotalArtifacts += len(bundle.ArtifactPaths)
		summary.ReviewerLinkHit += len(bundle.ReviewerLinks)
		if compareTimestamp(bundle.GeneratedAt, generatedAt) > 0 {
			generatedAt = bundle.GeneratedAt
		}
	}
	return parallelDiagnosticsEvidenceBundleIndex{
		GeneratedAt: generatedAt,
		Summary:     summary,
		Bundles:     bundles,
	}
}

func parallelDiagnosticsEvidenceSearchPayload(query, status, lane, path string, limit int) parallelDiagnosticsEvidenceSearchResponse {
	if limit <= 0 {
		limit = 20
	}
	index := parallelDiagnosticsEvidenceBundleIndexPayload()
	results := make([]parallelDiagnosticsEvidenceSearchHit, 0, len(index.Bundles))
	for _, bundle := range index.Bundles {
		matchedFields := matchParallelDiagnosticsEvidenceBundle(bundle, query, status, lane, path)
		if len(matchedFields) == 0 {
			continue
		}
		results = append(results, parallelDiagnosticsEvidenceSearchHit{
			Bundle:        bundle,
			MatchedFields: matchedFields,
		})
	}
	totalMatches := len(results)
	if len(results) > limit {
		results = results[:limit]
	}
	return parallelDiagnosticsEvidenceSearchResponse{
		Query:  strings.TrimSpace(query),
		Status: strings.TrimSpace(status),
		Lane:   strings.TrimSpace(lane),
		Path:   strings.TrimSpace(path),
		Limit:  limit,
		Summary: parallelDiagnosticsEvidenceSearchSummary{
			IndexedBundles:  len(index.Bundles),
			MatchedBundles:  totalMatches,
			ReturnedBundles: len(results),
		},
		Results: results,
	}
}

func buildParallelDiagnosticsEvidenceBundles() []parallelDiagnosticsEvidenceBundle {
	bundles := []parallelDiagnosticsEvidenceBundle{
		buildParallelDiagnosticsLiveValidationBundle(),
		buildParallelDiagnosticsLiveShadowBundle(),
		buildParallelDiagnosticsBrokerReviewBundle(),
		buildParallelDiagnosticsBrokerFanoutBundle(),
		buildParallelDiagnosticsProviderHandoffBundle(),
		buildParallelDiagnosticsContinuationBundle(),
	}
	sort.SliceStable(bundles, func(i, j int) bool {
		if compareTimestamp(bundles[i].GeneratedAt, bundles[j].GeneratedAt) == 0 {
			return bundles[i].ID < bundles[j].ID
		}
		return compareTimestamp(bundles[i].GeneratedAt, bundles[j].GeneratedAt) > 0
	})
	return bundles
}

func buildParallelDiagnosticsLiveValidationBundle() parallelDiagnosticsEvidenceBundle {
	bundle := parallelDiagnosticsEvidenceBundle{
		ID:         "live-validation",
		Kind:       "parallel_validation_bundle",
		Title:      "Parallel live validation evidence bundle",
		Status:     "unavailable",
		ReportPath: parallelDiagnosticsLiveValidationIndexPath,
	}
	indexPath := resolveRepoRelativePath(parallelDiagnosticsLiveValidationIndexPath)
	if indexPath == "" {
		return bundle
	}
	contents, err := os.ReadFile(indexPath)
	if err != nil {
		return bundle
	}
	var doc parallelDiagnosticsLiveValidationIndexDocument
	if err := json.Unmarshal(contents, &doc); err != nil {
		bundle.Status = "invalid"
		return bundle
	}
	bundle.Status = firstNonEmpty(doc.Latest.Status, doc.ContinuationGate.Status)
	bundle.GeneratedAt = doc.Latest.GeneratedAt
	bundle.BundlePath = doc.Latest.BundlePath
	bundle.SummaryPath = doc.Latest.SummaryPath
	bundle.Lanes = compactPaths(
		laneLabel(doc.Latest.Local.Enabled, "local"),
		laneLabel(doc.Latest.Kubernetes.Enabled, "kubernetes"),
		laneLabel(doc.Latest.Ray.Enabled, "ray"),
		laneLabel(doc.Latest.SharedQueueCompanion.Available, "shared-queue"),
		"continuation-gate",
	)
	bundle.ArtifactPaths = compactPaths(
		parallelDiagnosticsLiveValidationIndexPath,
		doc.Latest.BundlePath,
		doc.Latest.SummaryPath,
		doc.Latest.Local.BundleReportPath,
		doc.Latest.Local.CanonicalReportPath,
		doc.Latest.Local.StdoutPath,
		doc.Latest.Local.StderrPath,
		doc.Latest.Local.AuditLogPath,
		doc.Latest.Local.ServiceLogPath,
		doc.Latest.Kubernetes.BundleReportPath,
		doc.Latest.Kubernetes.CanonicalReportPath,
		doc.Latest.Kubernetes.StdoutPath,
		doc.Latest.Kubernetes.StderrPath,
		doc.Latest.Kubernetes.AuditLogPath,
		doc.Latest.Kubernetes.ServiceLogPath,
		doc.Latest.Ray.BundleReportPath,
		doc.Latest.Ray.CanonicalReportPath,
		doc.Latest.Ray.StdoutPath,
		doc.Latest.Ray.StderrPath,
		doc.Latest.Ray.AuditLogPath,
		doc.Latest.Ray.ServiceLogPath,
		doc.Latest.Broker.BundleSummaryPath,
		doc.Latest.Broker.CanonicalSummaryPath,
		doc.Latest.Broker.ValidationPackPath,
		doc.Latest.SharedQueueCompanion.BundleReportPath,
		doc.Latest.SharedQueueCompanion.BundleSummaryPath,
		doc.Latest.SharedQueueCompanion.CanonicalReportPath,
		doc.Latest.SharedQueueCompanion.CanonicalSummaryPath,
		doc.ContinuationGate.Path,
		doc.ContinuationGate.ReviewerPath.IndexPath,
		doc.ContinuationGate.ReviewerPath.DigestPath,
	)
	bundle.ReviewerLinks = compactPaths(
		doc.ContinuationGate.Path,
		doc.ContinuationGate.ReviewerPath.IndexPath,
		doc.ContinuationGate.ReviewerPath.DigestPath,
		doc.Latest.Broker.ValidationPackPath,
	)
	return bundle
}

func buildParallelDiagnosticsLiveShadowBundle() parallelDiagnosticsEvidenceBundle {
	bundle := parallelDiagnosticsEvidenceBundle{
		ID:         "live-shadow",
		Kind:       "parallel_shadow_bundle",
		Title:      "Parallel live shadow evidence bundle",
		Status:     "unavailable",
		ReportPath: parallelDiagnosticsLiveShadowIndexPath,
	}
	indexPath := resolveRepoRelativePath(parallelDiagnosticsLiveShadowIndexPath)
	if indexPath == "" {
		return bundle
	}
	contents, err := os.ReadFile(indexPath)
	if err != nil {
		return bundle
	}
	var doc parallelDiagnosticsLiveShadowIndexDocument
	if err := json.Unmarshal(contents, &doc); err != nil {
		bundle.Status = "invalid"
		return bundle
	}
	bundle.Status = firstNonEmpty(doc.Latest.Status, doc.DriftRollup.Status)
	bundle.GeneratedAt = doc.Latest.GeneratedAt
	bundle.BundlePath = doc.Latest.BundlePath
	bundle.SummaryPath = doc.Latest.SummaryPath
	bundle.Severity = doc.Latest.Severity
	bundle.Lanes = compactPaths("shadow-compare", "shadow-matrix", "migration")
	bundle.ArtifactPaths = compactPaths(
		parallelDiagnosticsLiveShadowIndexPath,
		doc.Latest.BundlePath,
		doc.Latest.SummaryPath,
		doc.Latest.Artifacts.ShadowCompareReportPath,
		doc.Latest.Artifacts.ShadowMatrixReportPath,
		doc.Latest.Artifacts.LiveShadowScorecardPath,
		doc.Latest.Artifacts.RollbackTriggerSurfacePath,
		doc.Latest.RollbackTriggerSurface.DigestPath,
		doc.Latest.RollbackTriggerSurface.SummaryPath,
	)
	for _, item := range doc.Latest.Freshness {
		bundle.ArtifactPaths = compactPaths(append(bundle.ArtifactPaths, normalizeReportPath(item.ReportPath))...)
	}
	bundle.ReviewerLinks = compactPaths(
		doc.Latest.RollbackTriggerSurface.DigestPath,
		doc.Latest.RollbackTriggerSurface.SummaryPath,
		parallelDiagnosticsLiveShadowIndexPath,
	)
	return bundle
}

func buildParallelDiagnosticsBrokerReviewBundle() parallelDiagnosticsEvidenceBundle {
	surface := brokerReviewBundleSurfacePayload()
	return parallelDiagnosticsEvidenceBundle{
		ID:          "broker-review",
		Kind:        "broker_review_bundle",
		Title:       "Broker review evidence bundle",
		Status:      firstNonEmpty(surface.Status, "unknown"),
		ReportPath:  surface.StubReportPath,
		SummaryPath: surface.CanonicalSummaryPath,
		Lanes:       compactPaths("broker", "validation"),
		ArtifactPaths: compactPaths(
			surface.CanonicalSummaryPath,
			surface.CanonicalBootstrapSummaryPath,
			surface.ValidationPackPath,
			surface.StubReportPath,
			surface.ArtifactDirectory,
			surface.LiveValidationIndexPath,
			surface.ReviewReadinessPath,
			surface.OperatorGuidePath,
			surface.AmbiguousPublishProof.Path,
		),
		ReviewerLinks: append([]string(nil), surface.ReviewerLinks...),
	}
}

func buildParallelDiagnosticsBrokerFanoutBundle() parallelDiagnosticsEvidenceBundle {
	surface := brokerStubFanoutIsolationPayload()
	paths := compactPaths(surface.ReportPath)
	paths = compactPaths(append(paths, surface.EvidenceSources...)...)
	for _, scenario := range surface.Scenarios {
		paths = compactPaths(append(paths, scenario.SourceTests...)...)
		paths = compactPaths(append(paths, scenario.ReplayPath, scenario.LivePath)...)
	}
	return parallelDiagnosticsEvidenceBundle{
		ID:            "broker-fanout-isolation",
		Kind:          "broker_fanout_evidence_pack",
		Title:         firstNonEmpty(surface.Title, "Broker stub live fanout isolation evidence pack"),
		Status:        surface.Status,
		GeneratedAt:   surface.GeneratedAt,
		ReportPath:    surface.ReportPath,
		Lanes:         compactPaths("broker", "fanout", "shared-queue"),
		ArtifactPaths: paths,
		ReviewerLinks: append([]string(nil), surface.ReviewerLinks...),
	}
}

func buildParallelDiagnosticsProviderHandoffBundle() parallelDiagnosticsEvidenceBundle {
	surface := providerLiveHandoffIsolationPayload()
	paths := compactPaths(surface.ReportPath)
	paths = compactPaths(append(paths, surface.EvidenceSources...)...)
	for _, scenario := range surface.Scenarios {
		paths = compactPaths(append(paths, scenario.SourceTests...)...)
		paths = compactPaths(append(paths, scenario.ReplayPath, scenario.LivePath)...)
	}
	lanes := compactPaths("provider", "handoff")
	if trimmed := strings.TrimSpace(surface.ValidationLane); trimmed != "" {
		lanes = compactPaths(append(lanes, trimmed)...)
	}
	return parallelDiagnosticsEvidenceBundle{
		ID:            "provider-live-handoff",
		Kind:          "provider_handoff_evidence_pack",
		Title:         firstNonEmpty(surface.Title, "Provider-backed live handoff isolation evidence pack"),
		Status:        surface.Status,
		GeneratedAt:   surface.GeneratedAt,
		ReportPath:    surface.ReportPath,
		Lanes:         lanes,
		ArtifactPaths: paths,
		ReviewerLinks: append([]string(nil), surface.ReviewerLinks...),
	}
}

func buildParallelDiagnosticsContinuationBundle() parallelDiagnosticsEvidenceBundle {
	surface := validationBundleContinuationGatePayload()
	return parallelDiagnosticsEvidenceBundle{
		ID:          "validation-continuation-gate",
		Kind:        "validation_continuation_gate",
		Title:       firstNonEmpty(surface.Title, "Validation bundle continuation gate"),
		Status:      surface.Status,
		GeneratedAt: surface.GeneratedAt,
		ReportPath:  surface.ReportPath,
		SummaryPath: surface.ScorecardPath,
		Lanes:       compactPaths("continuation-gate", "local", "kubernetes", "ray", "shared-queue"),
		ArtifactPaths: compactPaths(
			surface.ReportPath,
			surface.ScorecardPath,
			surface.DigestPath,
		),
		ReviewerLinks: append([]string(nil), surface.ReviewerLinks...),
	}
}

func matchParallelDiagnosticsEvidenceBundle(bundle parallelDiagnosticsEvidenceBundle, query, status, lane, path string) []string {
	query = strings.TrimSpace(strings.ToLower(query))
	status = strings.TrimSpace(strings.ToLower(status))
	lane = strings.TrimSpace(strings.ToLower(lane))
	path = strings.TrimSpace(strings.ToLower(path))

	if status != "" && strings.ToLower(bundle.Status) != status {
		return nil
	}
	if lane != "" && !bundleHasLane(bundle, lane) {
		return nil
	}
	if path != "" && !bundleHasPath(bundle, path) {
		return nil
	}
	if query == "" {
		return []string{"filters"}
	}
	fields := make([]string, 0, 4)
	if strings.Contains(strings.ToLower(bundle.ID), query) || strings.Contains(strings.ToLower(bundle.Kind), query) {
		fields = append(fields, "identity")
	}
	if strings.Contains(strings.ToLower(bundle.Title), query) {
		fields = append(fields, "title")
	}
	if strings.Contains(strings.ToLower(bundle.Status), query) || strings.Contains(strings.ToLower(bundle.Severity), query) {
		fields = append(fields, "status")
	}
	if bundleHasLane(bundle, query) {
		fields = append(fields, "lanes")
	}
	if bundleHasPath(bundle, query) {
		fields = append(fields, "artifact_paths")
	}
	if len(fields) == 0 {
		return nil
	}
	return uniqueStrings(fields)
}

func bundleHasLane(bundle parallelDiagnosticsEvidenceBundle, candidate string) bool {
	for _, item := range bundle.Lanes {
		if strings.Contains(strings.ToLower(item), candidate) {
			return true
		}
	}
	return false
}

func bundleHasPath(bundle parallelDiagnosticsEvidenceBundle, candidate string) bool {
	for _, item := range bundle.ArtifactPaths {
		if strings.Contains(strings.ToLower(item), candidate) {
			return true
		}
	}
	for _, item := range bundle.ReviewerLinks {
		if strings.Contains(strings.ToLower(item), candidate) {
			return true
		}
	}
	return false
}

func collectBundleKinds(bundles []parallelDiagnosticsEvidenceBundle) []string {
	values := make([]string, 0, len(bundles))
	for _, bundle := range bundles {
		values = append(values, bundle.Kind)
	}
	sort.Strings(values)
	return uniqueStrings(values)
}

func collectBundleStatuses(bundles []parallelDiagnosticsEvidenceBundle) []string {
	values := make([]string, 0, len(bundles))
	for _, bundle := range bundles {
		if strings.TrimSpace(bundle.Status) != "" {
			values = append(values, bundle.Status)
		}
	}
	sort.Strings(values)
	return uniqueStrings(values)
}

func collectBundleLanes(bundles []parallelDiagnosticsEvidenceBundle) []string {
	values := make([]string, 0, len(bundles))
	for _, bundle := range bundles {
		values = append(values, bundle.Lanes...)
	}
	sort.Strings(values)
	return uniqueStrings(values)
}

func laneLabel(enabled bool, value string) string {
	if !enabled {
		return ""
	}
	return value
}

func compareTimestamp(left, right string) int {
	leftTime, leftOK := parseComparableTime(left)
	rightTime, rightOK := parseComparableTime(right)
	switch {
	case leftOK && rightOK:
		if leftTime.Before(rightTime) {
			return -1
		}
		if leftTime.After(rightTime) {
			return 1
		}
		return 0
	case leftOK:
		return 1
	case rightOK:
		return -1
	default:
		return strings.Compare(strings.TrimSpace(left), strings.TrimSpace(right))
	}
}

func parseComparableTime(value string) (time.Time, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func parsePositiveIntQuery(value string, fallback int) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid integer %q", value)
	}
	if parsed <= 0 {
		return 0, fmt.Errorf("expected positive integer, got %d", parsed)
	}
	return parsed, nil
}
