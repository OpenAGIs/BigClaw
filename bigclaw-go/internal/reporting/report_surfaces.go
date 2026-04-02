package reporting

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"bigclaw-go/internal/collaboration"
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/workflow"
)

type DocumentationArtifact struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type LaunchChecklistItem struct {
	Name     string   `json:"name"`
	Evidence []string `json:"evidence,omitempty"`
}

type LaunchChecklist struct {
	IssueID              string                  `json:"issue_id"`
	Documentation        []DocumentationArtifact `json:"documentation,omitempty"`
	Items                []LaunchChecklistItem   `json:"items,omitempty"`
	DocumentationStatus  map[string]bool         `json:"documentation_status,omitempty"`
	CompletedItems       int                     `json:"completed_items"`
	MissingDocumentation []string                `json:"missing_documentation,omitempty"`
	Ready                bool                    `json:"ready"`
}

type FinalDeliveryChecklist struct {
	IssueID                           string                  `json:"issue_id"`
	RequiredOutputs                   []DocumentationArtifact `json:"required_outputs,omitempty"`
	RecommendedDocumentation          []DocumentationArtifact `json:"recommended_documentation,omitempty"`
	RequiredOutputStatus              map[string]bool         `json:"required_output_status,omitempty"`
	RecommendedDocumentationStatus    map[string]bool         `json:"recommended_documentation_status,omitempty"`
	GeneratedRequiredOutputs          int                     `json:"generated_required_outputs"`
	GeneratedRecommendedDocumentation int                     `json:"generated_recommended_documentation"`
	MissingRequiredOutputs            []string                `json:"missing_required_outputs,omitempty"`
	MissingRecommendedDocumentation   []string                `json:"missing_recommended_documentation,omitempty"`
	Ready                             bool                    `json:"ready"`
}

type IssueClosureDecision struct {
	Allowed    bool   `json:"allowed"`
	Reason     string `json:"reason"`
	ReportPath string `json:"report_path,omitempty"`
}

type NarrativeSection struct {
	Heading  string   `json:"heading"`
	Body     string   `json:"body"`
	Evidence []string `json:"evidence,omitempty"`
	Callouts []string `json:"callouts,omitempty"`
}

type ReportStudio struct {
	Name          string             `json:"name"`
	IssueID       string             `json:"issue_id"`
	Audience      string             `json:"audience"`
	Period        string             `json:"period"`
	Summary       string             `json:"summary"`
	Sections      []NarrativeSection `json:"sections,omitempty"`
	ActionItems   []string           `json:"action_items,omitempty"`
	SourceReports []string           `json:"source_reports,omitempty"`
}

func (s ReportStudio) Ready() bool {
	if strings.TrimSpace(s.Summary) == "" || len(s.Sections) == 0 {
		return false
	}
	for _, section := range s.Sections {
		if strings.TrimSpace(section.Heading) == "" || strings.TrimSpace(section.Body) == "" {
			return false
		}
	}
	return true
}

func (s ReportStudio) Recommendation() string {
	if s.Ready() {
		return "publish"
	}
	return "draft"
}

type ReportStudioBundle struct {
	MarkdownPath string `json:"markdown_path"`
	HTMLPath     string `json:"html_path"`
	TextPath     string `json:"text_path"`
}

type PilotMetric struct {
	Name           string  `json:"name"`
	Baseline       float64 `json:"baseline"`
	Current        float64 `json:"current"`
	Target         float64 `json:"target"`
	Unit           string  `json:"unit"`
	HigherIsBetter bool    `json:"higher_is_better,omitempty"`
}

func (m PilotMetric) Met() bool {
	if m.HigherIsBetter || !metricLowerIsBetterSet(m) {
		return m.Current >= m.Target
	}
	return m.Current <= m.Target
}

func metricLowerIsBetterSet(m PilotMetric) bool {
	return !m.HigherIsBetter
}

type PilotScorecard struct {
	IssueID            string        `json:"issue_id"`
	Customer           string        `json:"customer"`
	Period             string        `json:"period"`
	Metrics            []PilotMetric `json:"metrics,omitempty"`
	MonthlyBenefit     float64       `json:"monthly_benefit"`
	MonthlyCost        float64       `json:"monthly_cost"`
	ImplementationCost float64       `json:"implementation_cost"`
	BenchmarkScore     int           `json:"benchmark_score"`
	BenchmarkPassed    bool          `json:"benchmark_passed"`
}

func (s PilotScorecard) MetricsMet() int {
	count := 0
	for _, metric := range s.Metrics {
		if metric.Met() {
			count++
		}
	}
	return count
}

func (s PilotScorecard) MonthlyNetValue() float64 {
	return s.MonthlyBenefit - s.MonthlyCost
}

func (s PilotScorecard) PaybackMonths() *float64 {
	net := s.MonthlyNetValue()
	if net <= 0 {
		return nil
	}
	value := roundTenth(s.ImplementationCost / net)
	return &value
}

func (s PilotScorecard) AnnualizedROI() float64 {
	if s.ImplementationCost <= 0 {
		return 0
	}
	return roundTenth(((s.MonthlyNetValue() * 12) - s.ImplementationCost) / s.ImplementationCost * 100.0)
}

func (s PilotScorecard) Recommendation() string {
	if s.MonthlyNetValue() <= 0 || !s.BenchmarkPassed {
		return "hold"
	}
	if s.MetricsMet() == len(s.Metrics) && s.BenchmarkScore >= 95 {
		return "go"
	}
	return "iterate"
}

type PilotPortfolio struct {
	Name       string           `json:"name"`
	Period     string           `json:"period"`
	Scorecards []PilotScorecard `json:"scorecards,omitempty"`
}

func (p PilotPortfolio) TotalMonthlyNetValue() float64 {
	total := 0.0
	for _, scorecard := range p.Scorecards {
		total += scorecard.MonthlyNetValue()
	}
	return total
}

func (p PilotPortfolio) AverageROI() float64 {
	if len(p.Scorecards) == 0 {
		return 0
	}
	total := 0.0
	for _, scorecard := range p.Scorecards {
		total += scorecard.AnnualizedROI()
	}
	return roundTenth(total / float64(len(p.Scorecards)))
}

func (p PilotPortfolio) RecommendationCounts() map[string]int {
	counts := map[string]int{"go": 0, "iterate": 0, "hold": 0}
	for _, scorecard := range p.Scorecards {
		counts[scorecard.Recommendation()]++
	}
	return counts
}

func (p PilotPortfolio) Recommendation() string {
	counts := p.RecommendationCounts()
	if counts["hold"] > 0 {
		return "stabilize"
	}
	return "continue"
}

type TriageSimilarityEvidence struct {
	RelatedRunID string  `json:"related_run_id"`
	Score        float64 `json:"score"`
}

type TriageSuggestion struct {
	Label          string                     `json:"label"`
	Action         string                     `json:"action"`
	Confidence     float64                    `json:"confidence"`
	FeedbackStatus string                     `json:"feedback_status,omitempty"`
	Evidence       []TriageSimilarityEvidence `json:"evidence,omitempty"`
}

type TriageFeedbackRecord struct {
	RunID     string `json:"run_id"`
	Action    string `json:"action"`
	Decision  string `json:"decision"`
	Actor     string `json:"actor"`
	Notes     string `json:"notes,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

type AutoTriageFinding struct {
	RunID      string          `json:"run_id"`
	TaskID     string          `json:"task_id"`
	Severity   string          `json:"severity"`
	Owner      string          `json:"owner"`
	Status     string          `json:"status"`
	NextAction string          `json:"next_action"`
	Actions    []ConsoleAction `json:"actions,omitempty"`
}

type AutoTriageInboxItem struct {
	RunID       string             `json:"run_id"`
	TaskID      string             `json:"task_id"`
	Severity    string             `json:"severity"`
	Owner       string             `json:"owner"`
	Status      string             `json:"status"`
	Suggestions []TriageSuggestion `json:"suggestions,omitempty"`
}

type AutoTriageCenter struct {
	Name           string                `json:"name"`
	Period         string                `json:"period"`
	Findings       []AutoTriageFinding   `json:"findings,omitempty"`
	Inbox          []AutoTriageInboxItem `json:"inbox,omitempty"`
	FlaggedRuns    int                   `json:"flagged_runs"`
	InboxSize      int                   `json:"inbox_size"`
	SeverityCounts map[string]int        `json:"severity_counts,omitempty"`
	OwnerCounts    map[string]int        `json:"owner_counts,omitempty"`
	FeedbackCounts map[string]int        `json:"feedback_counts,omitempty"`
	Recommendation string                `json:"recommendation,omitempty"`
}

type OrchestrationPortfolio struct {
	Name                  string                `json:"name"`
	Period                string                `json:"period"`
	Canvases              []OrchestrationCanvas `json:"canvases,omitempty"`
	TakeoverQueue         *TakeoverQueue        `json:"takeover_queue,omitempty"`
	TotalRuns             int                   `json:"total_runs"`
	CollaborationModes    map[string]int        `json:"collaboration_modes,omitempty"`
	TierCounts            map[string]int        `json:"tier_counts,omitempty"`
	EntitlementCounts     map[string]int        `json:"entitlement_counts,omitempty"`
	BillingModelCounts    map[string]int        `json:"billing_model_counts,omitempty"`
	TotalEstimatedCostUSD float64               `json:"total_estimated_cost_usd"`
	TotalOverageCostUSD   float64               `json:"total_overage_cost_usd"`
	UpgradeRequiredCount  int                   `json:"upgrade_required_count"`
	ActiveHandoffs        int                   `json:"active_handoffs"`
	Recommendation        string                `json:"recommendation,omitempty"`
}

type BillingRunCharge struct {
	RunID               string   `json:"run_id"`
	TaskID              string   `json:"task_id"`
	EntitlementStatus   string   `json:"entitlement_status"`
	BillingModel        string   `json:"billing_model"`
	EstimatedCostUSD    float64  `json:"estimated_cost_usd"`
	IncludedUsageUnits  int      `json:"included_usage_units"`
	OverageUsageUnits   int      `json:"overage_usage_units"`
	OverageCostUSD      float64  `json:"overage_cost_usd"`
	BlockedCapabilities []string `json:"blocked_capabilities,omitempty"`
	HandoffTeam         string   `json:"handoff_team,omitempty"`
	Recommendation      string   `json:"recommendation,omitempty"`
}

type BillingEntitlementsPage struct {
	WorkspaceName           string             `json:"workspace_name"`
	PlanName                string             `json:"plan_name"`
	BillingPeriod           string             `json:"billing_period"`
	Charges                 []BillingRunCharge `json:"charges,omitempty"`
	RunCount                int                `json:"run_count"`
	TotalIncludedUsageUnits int                `json:"total_included_usage_units"`
	TotalOverageUsageUnits  int                `json:"total_overage_usage_units"`
	TotalEstimatedCostUSD   float64            `json:"total_estimated_cost_usd"`
	TotalOverageCostUSD     float64            `json:"total_overage_cost_usd"`
	UpgradeRequiredCount    int                `json:"upgrade_required_count"`
	EntitlementCounts       map[string]int     `json:"entitlement_counts,omitempty"`
	BillingModelCounts      map[string]int     `json:"billing_model_counts,omitempty"`
	BlockedCapabilities     []string           `json:"blocked_capabilities,omitempty"`
	Recommendation          string             `json:"recommendation,omitempty"`
}

func ensureTimestamp(ts string) string {
	if strings.TrimSpace(ts) != "" {
		return ts
	}
	return time.Now().UTC().Format(time.RFC3339)
}

func ValidationReportExists(path string) bool {
	body, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(body)) != ""
}

func RenderIssueValidationReport(issueID, version, environment, status string) string {
	generatedAt := time.Now().UTC().Format(time.RFC3339)
	return strings.Join([]string{
		"# Validation",
		"",
		fmt.Sprintf("- Issue: %s", issueID),
		fmt.Sprintf("- Version: %s", version),
		fmt.Sprintf("- Environment: %s", environment),
		fmt.Sprintf("- Status: %s", status),
		fmt.Sprintf("- 生成时间: %s", generatedAt),
		"",
	}, "\n")
}

func BuildLaunchChecklist(issueID string, documentation []DocumentationArtifact, items []LaunchChecklistItem) LaunchChecklist {
	status := make(map[string]bool, len(documentation))
	missing := make([]string, 0)
	for _, doc := range documentation {
		available := ValidationReportExists(doc.Path)
		status[doc.Name] = available
		if !available {
			missing = append(missing, doc.Name)
		}
	}
	completed := 0
	for _, item := range items {
		if launchChecklistItemComplete(item, status) {
			completed++
		}
	}
	return LaunchChecklist{
		IssueID:              issueID,
		Documentation:        append([]DocumentationArtifact(nil), documentation...),
		Items:                append([]LaunchChecklistItem(nil), items...),
		DocumentationStatus:  status,
		CompletedItems:       completed,
		MissingDocumentation: missing,
		Ready:                len(missing) == 0 && completed == len(items),
	}
}

func launchChecklistItemComplete(item LaunchChecklistItem, docs map[string]bool) bool {
	for _, evidence := range item.Evidence {
		if !docs[evidence] {
			return false
		}
	}
	return true
}

func BuildFinalDeliveryChecklist(issueID string, requiredOutputs []DocumentationArtifact, recommendedDocumentation []DocumentationArtifact) FinalDeliveryChecklist {
	requiredStatus := make(map[string]bool, len(requiredOutputs))
	recommendedStatus := make(map[string]bool, len(recommendedDocumentation))
	missingRequired := make([]string, 0)
	missingRecommended := make([]string, 0)
	generatedRequired := 0
	generatedRecommended := 0
	for _, doc := range requiredOutputs {
		available := ValidationReportExists(doc.Path)
		requiredStatus[doc.Name] = available
		if available {
			generatedRequired++
		} else {
			missingRequired = append(missingRequired, doc.Name)
		}
	}
	for _, doc := range recommendedDocumentation {
		available := ValidationReportExists(doc.Path)
		recommendedStatus[doc.Name] = available
		if available {
			generatedRecommended++
		} else {
			missingRecommended = append(missingRecommended, doc.Name)
		}
	}
	return FinalDeliveryChecklist{
		IssueID:                           issueID,
		RequiredOutputs:                   append([]DocumentationArtifact(nil), requiredOutputs...),
		RecommendedDocumentation:          append([]DocumentationArtifact(nil), recommendedDocumentation...),
		RequiredOutputStatus:              requiredStatus,
		RecommendedDocumentationStatus:    recommendedStatus,
		GeneratedRequiredOutputs:          generatedRequired,
		GeneratedRecommendedDocumentation: generatedRecommended,
		MissingRequiredOutputs:            missingRequired,
		MissingRecommendedDocumentation:   missingRecommended,
		Ready:                             len(missingRequired) == 0,
	}
}

func EvaluateIssueClosure(issueID, reportPath string, validationPassed bool, launchChecklist *LaunchChecklist, finalDeliveryChecklist *FinalDeliveryChecklist) IssueClosureDecision {
	if !ValidationReportExists(reportPath) {
		return IssueClosureDecision{Allowed: false, Reason: "validation report required before closing issue"}
	}
	if !validationPassed {
		return IssueClosureDecision{Allowed: false, Reason: "validation failed; issue must remain open"}
	}
	if finalDeliveryChecklist != nil {
		if !finalDeliveryChecklist.Ready {
			return IssueClosureDecision{Allowed: false, Reason: "final delivery checklist incomplete; required outputs missing"}
		}
		return IssueClosureDecision{Allowed: true, Reason: "validation report and final delivery checklist requirements satisfied; issue can be closed", ReportPath: reportPath}
	}
	if launchChecklist != nil && !launchChecklist.Ready {
		return IssueClosureDecision{Allowed: false, Reason: "launch checklist incomplete; linked documentation missing or empty"}
	}
	return IssueClosureDecision{Allowed: true, Reason: "validation report and launch checklist requirements satisfied; issue can be closed", ReportPath: reportPath}
}

func RenderLaunchChecklistReport(checklist LaunchChecklist) string {
	lines := []string{"# Launch Checklist"}
	for _, doc := range checklist.Documentation {
		lines = append(lines, fmt.Sprintf("%s: available=%t", doc.Name, checklist.DocumentationStatus[doc.Name]))
	}
	for _, item := range checklist.Items {
		lines = append(lines, fmt.Sprintf("%s: completed=%t evidence=%s", item.Name, launchChecklistItemComplete(item, checklist.DocumentationStatus), strings.Join(item.Evidence, ", ")))
	}
	return strings.Join(lines, "\n")
}

func RenderFinalDeliveryChecklistReport(checklist FinalDeliveryChecklist) string {
	lines := []string{
		"# Final Delivery Checklist",
		fmt.Sprintf("Required Outputs Generated: %d/%d", checklist.GeneratedRequiredOutputs, len(checklist.RequiredOutputs)),
		fmt.Sprintf("Recommended Docs Generated: %d/%d", checklist.GeneratedRecommendedDocumentation, len(checklist.RecommendedDocumentation)),
	}
	for _, doc := range checklist.RequiredOutputs {
		lines = append(lines, fmt.Sprintf("%s: available=%t", doc.Name, checklist.RequiredOutputStatus[doc.Name]))
	}
	for _, doc := range checklist.RecommendedDocumentation {
		lines = append(lines, fmt.Sprintf("%s: available=%t", doc.Name, checklist.RecommendedDocumentationStatus[doc.Name]))
	}
	return strings.Join(lines, "\n")
}

func RenderReportStudioReport(studio ReportStudio) string {
	lines := []string{"# Report Studio", "", fmt.Sprintf("Recommendation: %s", studio.Recommendation()), ""}
	for _, section := range studio.Sections {
		lines = append(lines, fmt.Sprintf("### %s", section.Heading), "", section.Body, "")
	}
	return strings.Join(lines, "\n")
}

func RenderReportStudioPlainText(studio ReportStudio) string {
	return fmt.Sprintf("%s\nRecommendation: %s\nSummary: %s\n", studio.Name, studio.Recommendation(), studio.Summary)
}

func RenderReportStudioHTML(studio ReportStudio) string {
	return fmt.Sprintf("<html><head><title>%s</title></head><body><h1>%s</h1><p>%s</p></body></html>", studio.Name, studio.Name, studio.Summary)
}

func WriteReportStudioBundle(root string, studio ReportStudio) (ReportStudioBundle, error) {
	slug := slugify(studio.Name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return ReportStudioBundle{}, err
	}
	bundle := ReportStudioBundle{
		MarkdownPath: filepath.Join(root, slug+".md"),
		HTMLPath:     filepath.Join(root, slug+".html"),
		TextPath:     filepath.Join(root, slug+".txt"),
	}
	if err := WriteReport(bundle.MarkdownPath, RenderReportStudioReport(studio)); err != nil {
		return ReportStudioBundle{}, err
	}
	if err := WriteReport(bundle.HTMLPath, RenderReportStudioHTML(studio)); err != nil {
		return ReportStudioBundle{}, err
	}
	if err := WriteReport(bundle.TextPath, RenderReportStudioPlainText(studio)); err != nil {
		return ReportStudioBundle{}, err
	}
	return bundle, nil
}

func RenderPilotScorecard(scorecard PilotScorecard) string {
	payback := "none"
	if months := scorecard.PaybackMonths(); months != nil {
		payback = fmt.Sprintf("%.1f", *months)
	}
	lines := []string{
		"# Pilot Scorecard",
		fmt.Sprintf("Recommendation: %s", scorecard.Recommendation()),
		fmt.Sprintf("Annualized ROI: %.1f%%", scorecard.AnnualizedROI()),
		fmt.Sprintf("Payback Months: %s", payback),
		fmt.Sprintf("Benchmark Score: %d", scorecard.BenchmarkScore),
	}
	for _, metric := range scorecard.Metrics {
		lines = append(lines, fmt.Sprintf("%s: baseline=%.0f current=%.0f target=%.0f", metric.Name, metric.Baseline, metric.Current, metric.Target))
	}
	return strings.Join(lines, "\n")
}

func RenderPilotPortfolioReport(portfolio PilotPortfolio) string {
	counts := portfolio.RecommendationCounts()
	lines := []string{
		"# Pilot Portfolio Report",
		fmt.Sprintf("Recommendation Mix: go=%d iterate=%d hold=%d", counts["go"], counts["iterate"], counts["hold"]),
	}
	for _, scorecard := range portfolio.Scorecards {
		lines = append(lines, fmt.Sprintf("%s: recommendation=%s", scorecard.Customer, scorecard.Recommendation()))
	}
	return strings.Join(lines, "\n")
}

func RenderSharedViewContext(view SharedViewContext) []string {
	lines := []string{"## Filters"}
	for _, filter := range view.Filters {
		lines = append(lines, fmt.Sprintf("- %s: %s", filter.Label, filter.Value))
	}
	if view.Collaboration != nil {
		lines = append(lines, "## Collaboration", fmt.Sprintf("Surface: %s", view.Collaboration.Surface))
		for _, comment := range view.Collaboration.Comments {
			lines = append(lines, comment.Body)
		}
		for _, decision := range view.Collaboration.Decisions {
			lines = append(lines, decision.Summary)
		}
	}
	return lines
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "-", "_", "-", "/", "-", ".", "-", "--", "-")
	value = replacer.Replace(value)
	value = strings.Trim(value, "-")
	if value == "" {
		return "report"
	}
	return value
}

func NewTriageFeedbackRecord(runID, action, decision, actor, notes string) TriageFeedbackRecord {
	return TriageFeedbackRecord{
		RunID:     runID,
		Action:    action,
		Decision:  decision,
		Actor:     actor,
		Notes:     notes,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func BuildAutoTriageCenter(runs []observability.TaskRun, name, period string, feedback []TriageFeedbackRecord) AutoTriageCenter {
	center := AutoTriageCenter{
		Name:           name,
		Period:         period,
		SeverityCounts: map[string]int{"critical": 0, "high": 0, "medium": 0},
		OwnerCounts:    map[string]int{"security": 0, "engineering": 0, "operations": 0},
		FeedbackCounts: map[string]int{"accepted": 0, "rejected": 0, "pending": 0},
	}
	feedbackIndex := map[string]TriageFeedbackRecord{}
	for _, item := range feedback {
		feedbackIndex[item.RunID] = item
	}

	for _, run := range runs {
		severity, owner, flagged, nextAction := triageClassification(run)
		if !flagged {
			continue
		}
		center.SeverityCounts[severity]++
		center.OwnerCounts[owner]++
		suggestion := TriageSuggestion{
			Label:      triageSuggestionLabel(run),
			Action:     nextAction,
			Confidence: triageConfidence(run),
		}
		if item, ok := feedbackIndex[run.RunID]; ok {
			suggestion.FeedbackStatus = item.Decision
			center.FeedbackCounts[item.Decision]++
		} else {
			center.FeedbackCounts["pending"]++
		}
		for _, other := range runs {
			if other.RunID == run.RunID {
				continue
			}
			if similarity := triageSimilarity(run, other); similarity >= 0.8 {
				suggestion.Evidence = append(suggestion.Evidence, TriageSimilarityEvidence{RelatedRunID: other.RunID, Score: similarity})
			}
		}

		actions := triageActions(run, owner)
		finding := AutoTriageFinding{
			RunID:      run.RunID,
			TaskID:     run.Task.ID,
			Severity:   severity,
			Owner:      owner,
			Status:     run.Status,
			NextAction: nextAction,
			Actions:    actions,
		}
		center.Findings = append(center.Findings, finding)
		center.Inbox = append(center.Inbox, AutoTriageInboxItem{
			RunID:       run.RunID,
			TaskID:      run.Task.ID,
			Severity:    severity,
			Owner:       owner,
			Status:      run.Status,
			Suggestions: []TriageSuggestion{suggestion},
		})
	}
	sort.SliceStable(center.Findings, func(i, j int) bool { return center.Findings[i].Severity < center.Findings[j].Severity })
	sort.SliceStable(center.Inbox, func(i, j int) bool { return center.Inbox[i].Severity < center.Inbox[j].Severity })
	center.FlaggedRuns = len(center.Findings)
	center.InboxSize = len(center.Inbox)
	if center.SeverityCounts["critical"] > 0 || center.SeverityCounts["high"] > 0 {
		center.Recommendation = "immediate-attention"
	}
	return center
}

func triageClassification(run observability.TaskRun) (severity string, owner string, flagged bool, nextAction string) {
	switch run.Status {
	case "failed":
		return "critical", "engineering", true, "replay run and inspect tool failures"
	case "needs-approval":
		return "high", "security", true, "request approval and queue security review"
	default:
		return "medium", "operations", false, ""
	}
}

func triageSuggestionLabel(run observability.TaskRun) string {
	if run.Status == "failed" {
		return "replay candidate"
	}
	return "approval candidate"
}

func triageConfidence(run observability.TaskRun) float64 {
	if run.Status == "failed" {
		return 0.9
	}
	return 0.6
}

func triageSimilarity(left, right observability.TaskRun) float64 {
	if left.Status == right.Status && left.Summary == right.Summary && left.Medium == right.Medium {
		return 0.9
	}
	if left.Status == right.Status {
		return 0.5
	}
	return 0
}

func triageActions(run observability.TaskRun, owner string) []ConsoleAction {
	return []ConsoleAction{
		{ActionID: "drill-down", Label: "Drill Down", Target: run.RunID, Enabled: true},
		{ActionID: "comment", Label: "Comment", Target: run.RunID, Enabled: true},
		{ActionID: "assign", Label: "Assign", Target: run.RunID, Enabled: true},
		{ActionID: "escalate", Label: "Escalate", Target: run.RunID, Enabled: owner != "operations"},
		{ActionID: "retry", Label: "Retry", Target: run.RunID, Enabled: run.Status == "failed", Reason: retryReason(run)},
		{ActionID: "pause", Label: "Pause", Target: run.RunID, Enabled: run.Status == "failed"},
		{ActionID: "approve", Label: "Approve", Target: run.RunID, Enabled: run.Status != "needs-approval", Reason: "approval available after security review"},
	}
}

func retryReason(run observability.TaskRun) string {
	if run.Status == "failed" {
		return ""
	}
	return "retry available after owner review"
}

func RenderAutoTriageCenterReport(center AutoTriageCenter, totalRuns int, view ...SharedViewContext) string {
	lines := []string{
		"# Auto Triage Center",
		fmt.Sprintf("Flagged Runs: %d", center.FlaggedRuns),
		fmt.Sprintf("Inbox Size: %d", center.InboxSize),
		fmt.Sprintf("Severity Mix: critical=%d high=%d medium=%d", center.SeverityCounts["critical"], center.SeverityCounts["high"], center.SeverityCounts["medium"]),
		fmt.Sprintf("Feedback Loop: accepted=%d rejected=%d pending=%d", center.FeedbackCounts["accepted"], center.FeedbackCounts["rejected"], center.FeedbackCounts["pending"]),
	}
	if len(view) > 0 {
		lines = append(lines, renderViewState(view[0])...)
	}
	lines = append(lines, "## Findings")
	for _, finding := range center.Findings {
		lines = append(lines, fmt.Sprintf("%s: severity=%s owner=%s status=%s actions=%s", finding.RunID, finding.Severity, finding.Owner, finding.Status, renderActionList(finding.Actions)))
		for _, action := range finding.Actions {
			lines = append(lines, renderConsoleAction(action))
		}
	}
	lines = append(lines, "## Inbox")
	for _, item := range center.Inbox {
		line := fmt.Sprintf("%s: severity=%s owner=%s status=%s", item.RunID, item.Severity, item.Owner, item.Status)
		if len(item.Suggestions) > 0 && len(item.Suggestions[0].Evidence) > 0 {
			line += fmt.Sprintf(" similar=%s:%.1f", item.Suggestions[0].Evidence[0].RelatedRunID, item.Suggestions[0].Evidence[0].Score)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func BuildOrchestrationCanvas(run observability.TaskRun, plan workflow.OrchestrationPlan, policy workflow.OrchestrationPolicyDecision, handoff *workflow.HandoffRequest) OrchestrationCanvas {
	canvas := OrchestrationCanvas{
		RunID:              run.RunID,
		TaskID:             run.Task.ID,
		Source:             run.Task.Source,
		Summary:            run.Summary,
		CollaborationMode:  plan.CollaborationMode,
		Departments:        plan.Departments(),
		RequiredApprovals:  plan.RequiredApprovals(),
		Tier:               policy.Tier,
		UpgradeRequired:    policy.UpgradeRequired,
		EntitlementStatus:  policy.EntitlementStatus,
		BillingModel:       policy.BillingModel,
		EstimatedCostUSD:   policy.EstimatedCostUSD,
		IncludedUsageUnits: policy.IncludedUsageUnits,
		OverageUsageUnits:  policy.OverageUsageUnits,
		OverageCostUSD:     policy.OverageCostUSD,
		BlockedDepartments: append([]string(nil), policy.BlockedDepartments...),
		ActiveTools:        runTools(run),
		Actions:            orchestrationActions(run.RunID, handoff),
	}
	if handoff != nil {
		canvas.HandoffTeam = handoff.TargetTeam
		canvas.HandoffStatus = handoff.Status
		if len(canvas.RequiredApprovals) == 0 {
			canvas.RequiredApprovals = append([]string(nil), handoff.RequiredApprovals...)
		}
	}
	canvas.Recommendation = orchestrationRecommendation(canvas)
	if len(run.Comments) > 0 || len(run.Decisions) > 0 {
		canvas.CollaborationThread = runThread(run, "run")
	}
	return canvas
}

func runTools(run observability.TaskRun) []string {
	seen := map[string]struct{}{}
	for _, audit := range run.Audits {
		if audit.Action != "tool.invoke" {
			continue
		}
		tool := ledgerString(audit.Details["tool"])
		if tool != "" {
			seen[tool] = struct{}{}
		}
	}
	return sortedKeys(seen)
}

func orchestrationActions(runID string, handoff *workflow.HandoffRequest) []ConsoleAction {
	enabled := handoff != nil
	return []ConsoleAction{
		{ActionID: "drill-down", Label: "Drill Down", Target: runID, Enabled: true},
		{ActionID: "view-policy", Label: "View Policy", Target: runID, Enabled: true},
		{ActionID: "handoff", Label: "Handoff", Target: runID, Enabled: enabled},
		{ActionID: "escalate", Label: "Escalate", Target: runID, Enabled: enabled},
		{ActionID: "retry", Label: "Retry", Target: runID, Enabled: false, Reason: "retry available after entitlement or handoff resolution"},
	}
}

func orchestrationRecommendation(canvas OrchestrationCanvas) string {
	if canvas.UpgradeRequired {
		return "resolve-entitlement-gap"
	}
	if canvas.CollaborationThread != nil && len(canvas.CollaborationThread.Comments) > 0 {
		return "resolve-flow-comments"
	}
	if canvas.HandoffTeam == "security" {
		return "review-security-takeover"
	}
	return "continue"
}

func RenderOrchestrationCanvas(canvas OrchestrationCanvas) string {
	lines := []string{
		"# Orchestration Canvas",
		fmt.Sprintf("- Tier: %s", canvas.Tier),
		fmt.Sprintf("- Entitlement Status: %s", canvas.EntitlementStatus),
		fmt.Sprintf("- Billing Model: %s", canvas.BillingModel),
		fmt.Sprintf("- Estimated Cost (USD): %.2f", canvas.EstimatedCostUSD),
		fmt.Sprintf("- Handoff Team: %s", canvas.HandoffTeam),
		fmt.Sprintf("- Recommendation: %s", canvas.Recommendation),
		"## Actions",
	}
	for _, action := range canvas.Actions {
		lines = append(lines, renderConsoleAction(action))
	}
	if canvas.CollaborationThread != nil {
		lines = append(lines, "## Collaboration")
		for _, comment := range canvas.CollaborationThread.Comments {
			lines = append(lines, comment.Body)
		}
		for _, decision := range canvas.CollaborationThread.Decisions {
			lines = append(lines, decision.Summary)
		}
	}
	return strings.Join(lines, "\n")
}

func takeoverActions(request TakeoverRequest) []ConsoleAction {
	reason := ""
	enabled := true
	if request.TargetTeam == "security" {
		enabled = false
		reason = "security takeovers are already escalated"
	}
	return []ConsoleAction{
		{ActionID: "drill-down", Label: "Drill Down", Target: request.RunID, Enabled: true},
		{ActionID: "handoff", Label: "Handoff", Target: request.RunID, Enabled: true},
		{ActionID: "approve", Label: "Approve", Target: request.RunID, Enabled: true},
		{ActionID: "escalate", Label: "Escalate", Target: request.RunID, Enabled: enabled, Reason: reason},
	}
}

func RenderTakeoverQueueReport(queue TakeoverQueue, totalRuns int, view ...SharedViewContext) string {
	lines := []string{
		"# Takeover Queue Report",
		fmt.Sprintf("Pending Requests: %d", queue.PendingRequests),
		fmt.Sprintf("Team Mix: %s", renderIntMap(queue.TeamCounts)),
	}
	if len(view) > 0 {
		lines = append(lines, renderViewState(view[0])...)
	}
	for _, request := range queue.Requests {
		lines = append(lines, fmt.Sprintf("%s: team=%s status=%s task=%s approvals=%s", request.RunID, request.TargetTeam, request.Status, request.TaskID, joinOrNone(request.RequiredApprovals)))
		for _, action := range request.Actions {
			lines = append(lines, renderConsoleAction(action))
		}
	}
	return strings.Join(lines, "\n")
}

func BuildOrchestrationPortfolio(canvases []OrchestrationCanvas, name, period string, takeoverQueue ...TakeoverQueue) OrchestrationPortfolio {
	portfolio := OrchestrationPortfolio{
		Name:               name,
		Period:             period,
		Canvases:           append([]OrchestrationCanvas(nil), canvases...),
		TotalRuns:          len(canvases),
		CollaborationModes: map[string]int{},
		TierCounts:         map[string]int{},
		EntitlementCounts:  map[string]int{},
		BillingModelCounts: map[string]int{},
	}
	for _, canvas := range canvases {
		portfolio.CollaborationModes[canvas.CollaborationMode]++
		portfolio.TierCounts[canvas.Tier]++
		portfolio.EntitlementCounts[canvas.EntitlementStatus]++
		portfolio.BillingModelCounts[canvas.BillingModel]++
		portfolio.TotalEstimatedCostUSD += canvas.EstimatedCostUSD
		portfolio.TotalOverageCostUSD += canvas.OverageCostUSD
		if canvas.UpgradeRequired {
			portfolio.UpgradeRequiredCount++
		}
		if strings.TrimSpace(canvas.HandoffTeam) != "" {
			portfolio.ActiveHandoffs++
		}
	}
	portfolio.TotalEstimatedCostUSD = roundTenth(portfolio.TotalEstimatedCostUSD*10) / 10
	portfolio.TotalOverageCostUSD = roundTenth(portfolio.TotalOverageCostUSD*10) / 10
	if len(takeoverQueue) > 0 {
		queue := takeoverQueue[0]
		portfolio.TakeoverQueue = &queue
	}
	if portfolio.TakeoverQueue != nil && portfolio.TakeoverQueue.TeamCounts["security"] > 0 {
		portfolio.Recommendation = "stabilize-security-takeovers"
	}
	return portfolio
}

func RenderOrchestrationPortfolioReport(portfolio OrchestrationPortfolio, view ...SharedViewContext) string {
	lines := []string{
		"# Orchestration Portfolio Report",
		fmt.Sprintf("- Collaboration Mix: %s", renderIntMap(portfolio.CollaborationModes)),
		fmt.Sprintf("- Tier Mix: %s", renderIntMap(portfolio.TierCounts)),
		fmt.Sprintf("- Entitlement Mix: %s", renderIntMap(portfolio.EntitlementCounts)),
		fmt.Sprintf("- Billing Models: %s", renderIntMap(portfolio.BillingModelCounts)),
		fmt.Sprintf("- Estimated Cost (USD): %.2f", portfolio.TotalEstimatedCostUSD),
		fmt.Sprintf("- Overage Cost (USD): %.2f", portfolio.TotalOverageCostUSD),
	}
	if portfolio.TakeoverQueue != nil {
		lines = append(lines, fmt.Sprintf("- Takeover Queue: pending=%d recommendation=%s", portfolio.TakeoverQueue.PendingRequests, portfolio.TakeoverQueue.Recommendation))
	}
	if len(view) > 0 {
		lines = append(lines, renderViewState(view[0])...)
	}
	for _, canvas := range portfolio.Canvases {
		lines = append(lines, fmt.Sprintf("- %s: mode=%s tier=%s entitlement=%s billing=%s estimated_cost_usd=%.2f overage_cost_usd=%.2f upgrade_required=%t handoff=%s actions=%s", canvas.RunID, canvas.CollaborationMode, canvas.Tier, canvas.EntitlementStatus, canvas.BillingModel, canvas.EstimatedCostUSD, canvas.OverageCostUSD, canvas.UpgradeRequired, canvas.HandoffTeam, renderActionList(canvas.Actions)))
	}
	return strings.Join(lines, "\n")
}

func RenderOrchestrationOverviewPage(portfolio OrchestrationPortfolio) string {
	return fmt.Sprintf("<html><head><title>Orchestration Overview</title></head><body><h1>%s</h1><p>Estimated Cost</p><p>%s</p><pre>%s</pre></body></html>", portfolio.Name, portfolio.Recommendation, RenderOrchestrationPortfolioReport(portfolio))
}

func BuildOrchestrationPortfolioFromLedger(entries []map[string]any, name, period string) OrchestrationPortfolio {
	canvases := make([]OrchestrationCanvas, 0, len(entries))
	for _, entry := range entries {
		canvases = append(canvases, BuildOrchestrationCanvasFromLedgerEntry(entry))
	}
	queue := BuildTakeoverQueueFromLedger(entries, name, period)
	return BuildOrchestrationPortfolio(canvases, name, period, queue)
}

func BuildBillingEntitlementsPage(portfolio OrchestrationPortfolio, workspaceName, planName, billingPeriod string) BillingEntitlementsPage {
	page := BillingEntitlementsPage{
		WorkspaceName:      workspaceName,
		PlanName:           planName,
		BillingPeriod:      billingPeriod,
		RunCount:           len(portfolio.Canvases),
		EntitlementCounts:  map[string]int{},
		BillingModelCounts: map[string]int{},
	}
	blocked := map[string]struct{}{}
	for _, canvas := range portfolio.Canvases {
		page.TotalIncludedUsageUnits += canvas.IncludedUsageUnits
		page.TotalOverageUsageUnits += canvas.OverageUsageUnits
		page.TotalEstimatedCostUSD += canvas.EstimatedCostUSD
		page.TotalOverageCostUSD += canvas.OverageCostUSD
		page.EntitlementCounts[canvas.EntitlementStatus]++
		page.BillingModelCounts[canvas.BillingModel]++
		if canvas.UpgradeRequired {
			page.UpgradeRequiredCount++
		}
		for _, dept := range canvas.BlockedDepartments {
			blocked[dept] = struct{}{}
		}
		page.Charges = append(page.Charges, BillingRunCharge{
			RunID:               canvas.RunID,
			TaskID:              canvas.TaskID,
			EntitlementStatus:   canvas.EntitlementStatus,
			BillingModel:        canvas.BillingModel,
			EstimatedCostUSD:    canvas.EstimatedCostUSD,
			IncludedUsageUnits:  canvas.IncludedUsageUnits,
			OverageUsageUnits:   canvas.OverageUsageUnits,
			OverageCostUSD:      canvas.OverageCostUSD,
			BlockedCapabilities: append([]string(nil), canvas.BlockedDepartments...),
			HandoffTeam:         canvas.HandoffTeam,
			Recommendation:      canvas.Recommendation,
		})
	}
	page.BlockedCapabilities = sortedKeys(blocked)
	if page.UpgradeRequiredCount > 0 {
		page.Recommendation = "resolve-plan-gaps"
	}
	return page
}

func BuildBillingEntitlementsPageFromLedger(entries []map[string]any, workspaceName, planName, billingPeriod string) BillingEntitlementsPage {
	portfolio := BuildOrchestrationPortfolioFromLedger(entries, workspaceName, billingPeriod)
	return BuildBillingEntitlementsPage(portfolio, workspaceName, planName, billingPeriod)
}

func RenderBillingEntitlementsReport(page BillingEntitlementsPage) string {
	lines := []string{
		"# Billing & Entitlements Report",
		fmt.Sprintf("- Workspace: %s", page.WorkspaceName),
		fmt.Sprintf("- Overage Cost (USD): %.2f", page.TotalOverageCostUSD),
	}
	for _, charge := range page.Charges {
		lines = append(lines, fmt.Sprintf("- %s: task=%s entitlement=%s billing=%s", charge.RunID, charge.TaskID, charge.EntitlementStatus, charge.BillingModel))
	}
	return strings.Join(lines, "\n")
}

func RenderBillingEntitlementsPage(page BillingEntitlementsPage) string {
	return fmt.Sprintf("<html><head><title>Billing & Entitlements</title></head><body><h1>%s</h1><p>%s plan for %s</p><h2>Charge Feed</h2><pre>%s</pre></body></html>", page.WorkspaceName, page.PlanName, page.BillingPeriod, RenderBillingEntitlementsReport(page))
}

func renderViewState(view SharedViewContext) []string {
	lines := []string{"## View State"}
	state := "ready"
	summary := "Data loaded for current filters."
	switch {
	case len(view.Errors) > 0:
		state = "error"
		summary = "Unable to load data for the current filters."
	case len(view.PartialData) > 0:
		state = "partial-data"
		summary = "Some data is still loading."
	case view.ResultCount == 0:
		state = "empty"
		summary = firstNonEmpty(view.EmptyMessage, "No records match the current filters.")
	}
	lines = append(lines, fmt.Sprintf("- State: %s", state), fmt.Sprintf("- Summary: %s", summary))
	lines = append(lines, "## Filters")
	for _, filter := range view.Filters {
		lines = append(lines, fmt.Sprintf("- %s: %s", filter.Label, filter.Value))
	}
	if len(view.PartialData) > 0 {
		lines = append(lines, "## Partial Data")
		lines = append(lines, view.PartialData...)
	}
	if len(view.Errors) > 0 {
		lines = append(lines, "## Errors")
		lines = append(lines, view.Errors...)
	}
	return lines
}

func renderActionList(actions []ConsoleAction) string {
	parts := make([]string, 0, len(actions))
	for _, action := range actions {
		parts = append(parts, fmt.Sprintf("%s [%s]", action.Label, action.ActionID))
	}
	return strings.Join(parts, ", ")
}

func renderConsoleAction(action ConsoleAction) string {
	text := fmt.Sprintf("%s [%s] state=%s target=%s", action.Label, action.ActionID, action.State(), action.Target)
	if strings.TrimSpace(action.Reason) != "" {
		text += " reason=" + action.Reason
	}
	return text
}

func renderIntMap(values map[string]int) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, values[key]))
	}
	return strings.Join(parts, " ")
}

func runThread(run observability.TaskRun, surface string) *collaboration.Thread {
	comments := make([]collaboration.Comment, 0, len(run.Comments))
	for _, comment := range run.Comments {
		comments = append(comments, collaboration.Comment{CommentID: comment.CommentID, Author: comment.Author, Body: comment.Body})
	}
	decisions := make([]collaboration.Decision, 0, len(run.Decisions))
	for _, decision := range run.Decisions {
		decisions = append(decisions, collaboration.Decision{DecisionID: decision.DecisionID, Author: decision.Author, Outcome: decision.Outcome, Summary: decision.Summary})
	}
	return collaboration.BuildThread(surface, run.RunID, comments, decisions)
}

func ledgerFloat(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	default:
		return 0
	}
}

func ledgerInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
