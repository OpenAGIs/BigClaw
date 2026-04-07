package api

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type rollbackTriggerRuntimeSurface struct {
	ReportPath             string                     `json:"report_path"`
	Issue                  rollbackTriggerIssue       `json:"issue"`
	Summary                rollbackTriggerSummary     `json:"summary"`
	SharedGuardrailSummary rollbackGuardrailSummary   `json:"shared_guardrail_summary"`
	Warnings               []rollbackTriggerCondition `json:"warnings,omitempty"`
	Blockers               []rollbackTriggerCondition `json:"blockers,omitempty"`
	ManualOnlyPaths        []rollbackManualOnlyPath   `json:"manual_only_paths,omitempty"`
	ReviewerLinks          []string                   `json:"reviewer_links,omitempty"`
	Error                  string                     `json:"error,omitempty"`
}

type rollbackTriggerIssue struct {
	ID    string `json:"id,omitempty"`
	Slug  string `json:"slug,omitempty"`
	Title string `json:"title,omitempty"`
}

type rollbackTriggerSummary struct {
	Status                   string                      `json:"status,omitempty"`
	AutomationBoundary       string                      `json:"automation_boundary,omitempty"`
	AutomatedRollbackTrigger bool                        `json:"automated_rollback_trigger"`
	CutoverGate              string                      `json:"cutover_gate,omitempty"`
	Distinctions             rollbackTriggerDistinctions `json:"distinctions"`
}

type rollbackTriggerDistinctions struct {
	Blockers        int `json:"blockers"`
	Warnings        int `json:"warnings"`
	ManualOnlyPaths int `json:"manual_only_paths"`
}

type rollbackGuardrailSummary struct {
	DigestPath             string `json:"digest_path,omitempty"`
	MigrationReadinessPath string `json:"migration_readiness_path,omitempty"`
	LiveShadowIndexPath    string `json:"live_shadow_index_path,omitempty"`
	LiveShadowManifestPath string `json:"live_shadow_manifest_path,omitempty"`
	LiveShadowRollupPath   string `json:"live_shadow_rollup_path,omitempty"`
}

type rollbackTriggerCondition struct {
	Key                string   `json:"key,omitempty"`
	Condition          string   `json:"condition,omitempty"`
	EvidencePaths      []string `json:"evidence_paths,omitempty"`
	OperatorResponse   []string `json:"operator_response,omitempty"`
	AutomationBoundary string   `json:"automation_boundary,omitempty"`
}

type rollbackManualOnlyPath struct {
	Key          string   `json:"key,omitempty"`
	Path         []string `json:"path,omitempty"`
	CurrentState string   `json:"current_state,omitempty"`
}

func rollbackTriggerSurfacePayload() rollbackTriggerRuntimeSurface {
	surface := rollbackTriggerRuntimeSurface{ReportPath: rollbackTriggerSurfacePath}
	reportPath := resolveRepoRelativePath(rollbackTriggerSurfacePath)
	if reportPath == "" {
		surface.Summary.Status = "unavailable"
		surface.Error = "report path could not be resolved"
		return surface
	}
	contents, err := os.ReadFile(reportPath)
	if err != nil {
		surface.Summary.Status = "unavailable"
		surface.Error = err.Error()
		return surface
	}
	if err := json.Unmarshal(contents, &surface); err != nil {
		surface.Summary.Status = "invalid"
		surface.Error = fmt.Sprintf("decode %s: %v", rollbackTriggerSurfacePath, err)
		return surface
	}
	surface.ReportPath = rollbackTriggerSurfacePath
	surface.ReviewerLinks = rollbackTriggerReviewerLinks(surface)
	return surface
}

func rollbackTriggerReviewerLinks(surface rollbackTriggerRuntimeSurface) []string {
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
	appendLink(surface.ReportPath)
	appendLink(surface.SharedGuardrailSummary.DigestPath)
	appendLink(surface.SharedGuardrailSummary.MigrationReadinessPath)
	appendLink(surface.SharedGuardrailSummary.LiveShadowIndexPath)
	appendLink(surface.SharedGuardrailSummary.LiveShadowManifestPath)
	appendLink(surface.SharedGuardrailSummary.LiveShadowRollupPath)
	if len(links) <= 2 {
		return links
	}
	first := links[0]
	middle := append([]string(nil), links[1:]...)
	sort.Strings(middle)
	return append([]string{first}, middle...)
}
