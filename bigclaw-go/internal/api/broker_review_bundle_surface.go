package api

import "strings"

type brokerReviewBundleSurface struct {
	Status                        string               `json:"status"`
	CanonicalSummaryPath          string               `json:"canonical_summary_path,omitempty"`
	CanonicalBootstrapSummaryPath string               `json:"canonical_bootstrap_summary_path,omitempty"`
	ValidationPackPath            string               `json:"validation_pack_path,omitempty"`
	StubReportPath                string               `json:"stub_report_path,omitempty"`
	ArtifactDirectory             string               `json:"artifact_directory,omitempty"`
	LiveValidationIndexPath       string               `json:"live_validation_index_path,omitempty"`
	ReviewReadinessPath           string               `json:"review_readiness_path,omitempty"`
	OperatorGuidePath             string               `json:"operator_guide_path,omitempty"`
	AmbiguousPublishProof         brokerProofReference `json:"ambiguous_publish_proof"`
	RuntimePosture                string               `json:"runtime_posture,omitempty"`
	BootstrapReady                bool                 `json:"bootstrap_ready"`
	LiveAdapterImplemented        bool                 `json:"live_adapter_implemented"`
	ProofBoundary                 string               `json:"proof_boundary,omitempty"`
	ReviewerLinks                 []string             `json:"reviewer_links,omitempty"`
}

func brokerReviewBundleSurfacePayload() brokerReviewBundleSurface {
	reviewPack := buildBrokerReviewPack()
	bootstrap := brokerBootstrapSurfacePayload()
	links := make([]string, 0, 8)
	appendUnique := func(values ...string) {
		for _, value := range values {
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				continue
			}
			exists := false
			for _, existing := range links {
				if existing == trimmed {
					exists = true
					break
				}
			}
			if !exists {
				links = append(links, trimmed)
			}
		}
	}
	appendUnique(
		reviewPack.SummaryPath,
		bootstrap.CanonicalSummaryPath,
		bootstrap.CanonicalBootstrapSummaryPath,
		reviewPack.ValidationPackPath,
		reviewPack.ReportPath,
		reviewPack.ArtifactDirectory,
		"docs/reports/live-validation-index.json",
		"docs/reports/review-readiness.md",
		brokerBootstrapOperatorGuidePath,
	)
	for _, link := range reviewPack.ReviewerLinks {
		appendUnique(link)
	}
	for _, link := range bootstrap.ConfigDiagnostics.ReferenceDocs {
		appendUnique(link)
	}
	appendUnique(reviewPack.AmbiguousPublishProof.Path)
	return brokerReviewBundleSurface{
		Status:                        reviewPack.Status,
		CanonicalSummaryPath:          firstNonEmpty(bootstrap.CanonicalSummaryPath, reviewPack.SummaryPath),
		CanonicalBootstrapSummaryPath: bootstrap.CanonicalBootstrapSummaryPath,
		ValidationPackPath:            firstNonEmpty(bootstrap.ValidationPackPath, reviewPack.ValidationPackPath),
		StubReportPath:                reviewPack.ReportPath,
		ArtifactDirectory:             reviewPack.ArtifactDirectory,
		LiveValidationIndexPath:       "docs/reports/live-validation-index.json",
		ReviewReadinessPath:           "docs/reports/review-readiness.md",
		OperatorGuidePath:             brokerBootstrapOperatorGuidePath,
		AmbiguousPublishProof:         reviewPack.AmbiguousPublishProof,
		RuntimePosture:                bootstrap.RuntimePosture,
		BootstrapReady:                bootstrap.BootstrapReady,
		LiveAdapterImplemented:        bootstrap.LiveAdapterImplemented,
		ProofBoundary:                 bootstrap.ProofBoundary,
		ReviewerLinks:                 links,
	}
}
