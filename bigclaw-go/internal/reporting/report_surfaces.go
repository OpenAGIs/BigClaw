package reporting

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"bigclaw-go/internal/repo"
	"bigclaw-go/internal/workflow"
	"bigclaw-go/internal/workflowexec"
)

type DocumentationArtifact struct {
	Name string `json:"name"`
	Path string `json:"path"`
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

func RenderReportStudioReport(studio ReportStudio) string {
	lines := []string{
		"# Report Studio",
		"",
		fmt.Sprintf("- Name: %s", studio.Name),
		fmt.Sprintf("- Issue ID: %s", studio.IssueID),
		fmt.Sprintf("- Audience: %s", studio.Audience),
		fmt.Sprintf("- Period: %s", studio.Period),
		fmt.Sprintf("- Recommendation: %s", studio.Recommendation()),
		"",
		"## Summary",
		"",
		studio.Summary,
	}
	if len(studio.Sections) > 0 {
		lines = append(lines, "", "## Narrative", "")
		for _, section := range studio.Sections {
			lines = append(lines, "### "+section.Heading, "", section.Body)
			if len(section.Evidence) > 0 {
				lines = append(lines, "", "- Evidence: "+strings.Join(section.Evidence, ", "))
			}
			if len(section.Callouts) > 0 {
				lines = append(lines, "- Callouts: "+strings.Join(section.Callouts, ", "))
			}
			lines = append(lines, "")
		}
	}
	if len(studio.ActionItems) > 0 {
		lines = append(lines, "## Action Items", "")
		for _, item := range studio.ActionItems {
			lines = append(lines, "- "+item)
		}
		lines = append(lines, "")
	}
	if len(studio.SourceReports) > 0 {
		lines = append(lines, "## Sources", "", "- "+strings.Join(studio.SourceReports, "\n- "), "")
	}
	return strings.Join(lines, "\n")
}

func RenderReportStudioPlainText(studio ReportStudio) string {
	return strings.Join([]string{
		"Report Studio",
		fmt.Sprintf("Name: %s", studio.Name),
		fmt.Sprintf("Recommendation: %s", studio.Recommendation()),
		fmt.Sprintf("Summary: %s", studio.Summary),
	}, "\n")
}

func RenderReportStudioHTML(studio ReportStudio) string {
	var b strings.Builder
	b.WriteString("<html><head><title>Report Studio</title></head><body>\n")
	b.WriteString("<h1>" + html.EscapeString(studio.Name) + "</h1>\n")
	b.WriteString("<p>Recommendation: " + html.EscapeString(studio.Recommendation()) + "</p>\n")
	for _, section := range studio.Sections {
		b.WriteString("<h2>" + html.EscapeString(section.Heading) + "</h2>\n")
		b.WriteString("<p>" + html.EscapeString(section.Body) + "</p>\n")
	}
	b.WriteString("</body></html>\n")
	return b.String()
}

func WriteReportStudioBundle(root string, studio ReportStudio) (ReportStudioBundle, error) {
	slug := slugify(studio.Name)
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

type PilotMetric struct {
	Name           string  `json:"name"`
	Baseline       float64 `json:"baseline"`
	Current        float64 `json:"current"`
	Target         float64 `json:"target"`
	Unit           string  `json:"unit,omitempty"`
	HigherIsBetter bool    `json:"higher_is_better,omitempty"`
}

func (m PilotMetric) MetTarget() bool {
	if m.HigherIsBetter {
		return m.Current >= m.Target
	}
	return m.Current <= m.Target
}

func (m PilotMetric) Delta() float64 {
	return m.Current - m.Baseline
}

type PilotScorecard struct {
	IssueID            string        `json:"issue_id"`
	Customer           string        `json:"customer"`
	Period             string        `json:"period"`
	Metrics            []PilotMetric `json:"metrics,omitempty"`
	MonthlyBenefit     float64       `json:"monthly_benefit"`
	MonthlyCost        float64       `json:"monthly_cost"`
	ImplementationCost float64       `json:"implementation_cost"`
	BenchmarkScore     *int          `json:"benchmark_score,omitempty"`
	BenchmarkPassed    *bool         `json:"benchmark_passed,omitempty"`
}

func (p PilotScorecard) MonthlyNetValue() float64 {
	return p.MonthlyBenefit - p.MonthlyCost
}

func (p PilotScorecard) AnnualizedROI() float64 {
	totalCost := p.ImplementationCost + (p.MonthlyCost * 12)
	if totalCost <= 0 {
		return 0
	}
	annualGain := (p.MonthlyBenefit * 12) - totalCost
	return (annualGain / totalCost) * 100
}

func (p PilotScorecard) PaybackMonths() *float64 {
	if p.MonthlyNetValue() <= 0 {
		return nil
	}
	if p.ImplementationCost <= 0 {
		value := 0.0
		return &value
	}
	value := round1(p.ImplementationCost / p.MonthlyNetValue())
	return &value
}

func (p PilotScorecard) MetricsMet() int {
	total := 0
	for _, metric := range p.Metrics {
		if metric.MetTarget() {
			total++
		}
	}
	return total
}

func (p PilotScorecard) Recommendation() string {
	benchmarkOK := p.BenchmarkPassed == nil || *p.BenchmarkPassed
	if len(p.Metrics) > 0 && p.MetricsMet() == len(p.Metrics) && p.AnnualizedROI() > 0 && benchmarkOK {
		return "go"
	}
	if p.AnnualizedROI() > 0 || p.MetricsMet() > 0 {
		return "iterate"
	}
	return "hold"
}

func RenderPilotScorecard(scorecard PilotScorecard) string {
	lines := []string{
		"# Pilot Scorecard",
		"",
		fmt.Sprintf("- Issue ID: %s", scorecard.IssueID),
		fmt.Sprintf("- Customer: %s", scorecard.Customer),
		fmt.Sprintf("- Period: %s", scorecard.Period),
		fmt.Sprintf("- Recommendation: %s", scorecard.Recommendation()),
		fmt.Sprintf("- Metrics Met: %d/%d", scorecard.MetricsMet(), len(scorecard.Metrics)),
		fmt.Sprintf("- Monthly Net Value: %.0f", scorecard.MonthlyNetValue()),
		fmt.Sprintf("- Annualized ROI: %.1f%%", scorecard.AnnualizedROI()),
	}
	if payback := scorecard.PaybackMonths(); payback == nil {
		lines = append(lines, "- Payback Months: n/a")
	} else {
		lines = append(lines, fmt.Sprintf("- Payback Months: %.1f", *payback))
	}
	if scorecard.BenchmarkScore != nil {
		lines = append(lines, fmt.Sprintf("- Benchmark Score: %d", *scorecard.BenchmarkScore))
	}
	if scorecard.BenchmarkPassed != nil {
		lines = append(lines, fmt.Sprintf("- Benchmark Passed: %t", *scorecard.BenchmarkPassed))
	}
	lines = append(lines, "", "## KPI Progress", "")
	for _, metric := range scorecard.Metrics {
		comp := ">="
		if !metric.HigherIsBetter {
			comp = "<="
		}
		unit := ""
		if metric.Unit != "" {
			unit = " " + metric.Unit
		}
		lines = append(lines, fmt.Sprintf("- %s: baseline=%.0f%s current=%.0f%s target%s%.0f%s delta=%+.2f%s met=%t", metric.Name, metric.Baseline, unit, metric.Current, unit, comp, metric.Target, unit, metric.Delta(), unit, metric.MetTarget()))
	}
	return strings.Join(lines, "\n") + "\n"
}

type PilotPortfolio struct {
	Name                 string           `json:"name"`
	Period               string           `json:"period"`
	Scorecards           []PilotScorecard `json:"scorecards,omitempty"`
	TotalMonthlyNetValue float64          `json:"total_monthly_net_value"`
	AverageROI           float64          `json:"average_roi"`
	RecommendationCounts map[string]int   `json:"recommendation_counts,omitempty"`
	Recommendation       string           `json:"recommendation"`
}

func BuildPilotPortfolio(name, period string, scorecards []PilotScorecard) PilotPortfolio {
	portfolio := PilotPortfolio{
		Name:                 strings.TrimSpace(name),
		Period:               strings.TrimSpace(period),
		Scorecards:           append([]PilotScorecard(nil), scorecards...),
		RecommendationCounts: map[string]int{"go": 0, "iterate": 0, "hold": 0},
		Recommendation:       "hold",
	}
	if len(scorecards) == 0 {
		return portfolio
	}
	var roiTotal float64
	for _, scorecard := range scorecards {
		portfolio.TotalMonthlyNetValue += scorecard.MonthlyNetValue()
		roiTotal += scorecard.AnnualizedROI()
		portfolio.RecommendationCounts[scorecard.Recommendation()]++
	}
	portfolio.AverageROI = round1(roiTotal / float64(len(scorecards)))
	if portfolio.RecommendationCounts["go"] > 0 {
		portfolio.Recommendation = "continue"
	} else if portfolio.RecommendationCounts["iterate"] > 0 {
		portfolio.Recommendation = "iterate"
	}
	return portfolio
}

func RenderPilotPortfolioReport(portfolio PilotPortfolio) string {
	lines := []string{
		"# Pilot Portfolio Report",
		"",
		fmt.Sprintf("- Name: %s", portfolio.Name),
		fmt.Sprintf("- Period: %s", portfolio.Period),
		fmt.Sprintf("- Recommendation: %s", portfolio.Recommendation),
		fmt.Sprintf("- Total Monthly Net Value: %.0f", portfolio.TotalMonthlyNetValue),
		fmt.Sprintf("- Average ROI: %.1f%%", portfolio.AverageROI),
		fmt.Sprintf("- Recommendation Mix: go=%d iterate=%d hold=%d", portfolio.RecommendationCounts["go"], portfolio.RecommendationCounts["iterate"], portfolio.RecommendationCounts["hold"]),
		"",
		"## Scorecards",
	}
	for _, scorecard := range portfolio.Scorecards {
		lines = append(lines, fmt.Sprintf("- %s: recommendation=%s", scorecard.Customer, scorecard.Recommendation()))
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderIssueValidationReport(issueID, version, environment, result string) string {
	lines := []string{
		"# Validation",
		"",
		fmt.Sprintf("- Issue ID: %s", strings.TrimSpace(issueID)),
		fmt.Sprintf("- Version: %s", strings.TrimSpace(version)),
		fmt.Sprintf("- Environment: %s", strings.TrimSpace(environment)),
		fmt.Sprintf("- Result: %s", strings.TrimSpace(result)),
		fmt.Sprintf("- 生成时间: %s", time.Now().UTC().Format(time.RFC3339)),
	}
	return strings.Join(lines, "\n") + "\n"
}

func ValidationReportExists(path string) bool {
	body, err := os.ReadFile(path)
	return err == nil && strings.TrimSpace(string(body)) != ""
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
}

func (c LaunchChecklist) Ready() bool {
	return len(c.MissingDocumentation) == 0 && c.CompletedItems == len(c.Items)
}

func BuildLaunchChecklist(issueID string, documentation []DocumentationArtifact, items []LaunchChecklistItem) LaunchChecklist {
	checklist := LaunchChecklist{
		IssueID:             strings.TrimSpace(issueID),
		Documentation:       append([]DocumentationArtifact(nil), documentation...),
		Items:               append([]LaunchChecklistItem(nil), items...),
		DocumentationStatus: map[string]bool{},
	}
	for _, item := range documentation {
		available := ValidationReportExists(item.Path)
		checklist.DocumentationStatus[item.Name] = available
		if !available {
			checklist.MissingDocumentation = append(checklist.MissingDocumentation, item.Name)
		}
	}
	for _, item := range items {
		completed := true
		for _, evidence := range item.Evidence {
			if !checklist.DocumentationStatus[evidence] {
				completed = false
			}
		}
		if completed {
			checklist.CompletedItems++
		}
	}
	sort.Strings(checklist.MissingDocumentation)
	return checklist
}

func RenderLaunchChecklistReport(checklist LaunchChecklist) string {
	lines := []string{
		"# Launch Checklist",
		"",
		fmt.Sprintf("- Issue ID: %s", checklist.IssueID),
		fmt.Sprintf("- Completed Items: %d/%d", checklist.CompletedItems, len(checklist.Items)),
		fmt.Sprintf("- Ready: %t", checklist.Ready()),
		"",
		"## Documentation",
	}
	docNames := sortedBoolMapKeys(checklist.DocumentationStatus)
	for _, name := range docNames {
		lines = append(lines, fmt.Sprintf("- %s: available=%t", name, checklist.DocumentationStatus[name]))
	}
	lines = append(lines, "", "## Items")
	for _, item := range checklist.Items {
		completed := true
		for _, evidence := range item.Evidence {
			if !checklist.DocumentationStatus[evidence] {
				completed = false
			}
		}
		lines = append(lines, fmt.Sprintf("- %s: completed=%t evidence=%s", item.Name, completed, strings.Join(item.Evidence, ", ")))
	}
	return strings.Join(lines, "\n") + "\n"
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
}

func (c FinalDeliveryChecklist) Ready() bool {
	return len(c.MissingRequiredOutputs) == 0
}

func BuildFinalDeliveryChecklist(issueID string, requiredOutputs, recommendedDocumentation []DocumentationArtifact) FinalDeliveryChecklist {
	checklist := FinalDeliveryChecklist{
		IssueID:                        strings.TrimSpace(issueID),
		RequiredOutputs:                append([]DocumentationArtifact(nil), requiredOutputs...),
		RecommendedDocumentation:       append([]DocumentationArtifact(nil), recommendedDocumentation...),
		RequiredOutputStatus:           map[string]bool{},
		RecommendedDocumentationStatus: map[string]bool{},
	}
	for _, item := range requiredOutputs {
		available := ValidationReportExists(item.Path)
		checklist.RequiredOutputStatus[item.Name] = available
		if available {
			checklist.GeneratedRequiredOutputs++
		} else {
			checklist.MissingRequiredOutputs = append(checklist.MissingRequiredOutputs, item.Name)
		}
	}
	for _, item := range recommendedDocumentation {
		available := ValidationReportExists(item.Path)
		checklist.RecommendedDocumentationStatus[item.Name] = available
		if available {
			checklist.GeneratedRecommendedDocumentation++
		} else {
			checklist.MissingRecommendedDocumentation = append(checklist.MissingRecommendedDocumentation, item.Name)
		}
	}
	sort.Strings(checklist.MissingRequiredOutputs)
	sort.Strings(checklist.MissingRecommendedDocumentation)
	return checklist
}

func RenderFinalDeliveryChecklistReport(checklist FinalDeliveryChecklist) string {
	lines := []string{
		"# Final Delivery Checklist",
		"",
		fmt.Sprintf("- Issue ID: %s", checklist.IssueID),
		fmt.Sprintf("- Required Outputs Generated: %d/%d", checklist.GeneratedRequiredOutputs, len(checklist.RequiredOutputs)),
		fmt.Sprintf("- Recommended Docs Generated: %d/%d", checklist.GeneratedRecommendedDocumentation, len(checklist.RecommendedDocumentation)),
		fmt.Sprintf("- Ready: %t", checklist.Ready()),
		"",
		"## Required Outputs",
	}
	for _, name := range sortedBoolMapKeys(checklist.RequiredOutputStatus) {
		lines = append(lines, fmt.Sprintf("- %s: available=%t", name, checklist.RequiredOutputStatus[name]))
	}
	lines = append(lines, "", "## Recommended Documentation")
	for _, name := range sortedBoolMapKeys(checklist.RecommendedDocumentationStatus) {
		lines = append(lines, fmt.Sprintf("- %s: available=%t", name, checklist.RecommendedDocumentationStatus[name]))
	}
	return strings.Join(lines, "\n") + "\n"
}

type IssueClosureDecision struct {
	Allowed    bool   `json:"allowed"`
	Reason     string `json:"reason"`
	ReportPath string `json:"report_path,omitempty"`
}

func EvaluateIssueClosure(issueID, reportPath string, validationPassed bool, launchChecklist *LaunchChecklist, finalDeliveryChecklist *FinalDeliveryChecklist) IssueClosureDecision {
	_ = issueID
	if !ValidationReportExists(reportPath) {
		return IssueClosureDecision{Allowed: false, Reason: "validation report required before closing issue"}
	}
	if !validationPassed {
		return IssueClosureDecision{Allowed: false, Reason: "validation failed; issue must remain open"}
	}
	if launchChecklist != nil {
		if !launchChecklist.Ready() {
			return IssueClosureDecision{Allowed: false, Reason: "launch checklist incomplete; linked documentation missing or empty"}
		}
		return IssueClosureDecision{Allowed: true, Reason: "validation report and launch checklist requirements satisfied; issue can be closed", ReportPath: reportPath}
	}
	if finalDeliveryChecklist != nil {
		if !finalDeliveryChecklist.Ready() {
			return IssueClosureDecision{Allowed: false, Reason: "final delivery checklist incomplete; required outputs missing"}
		}
		return IssueClosureDecision{Allowed: true, Reason: "validation report and final delivery checklist requirements satisfied; issue can be closed", ReportPath: reportPath}
	}
	return IssueClosureDecision{Allowed: true, Reason: "validation report and launch checklist requirements satisfied; issue can be closed", ReportPath: reportPath}
}

type SharedViewFilter struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type SharedViewContext struct {
	Filters       []SharedViewFilter        `json:"filters,omitempty"`
	ResultCount   int                       `json:"result_count"`
	Loading       bool                      `json:"loading"`
	Errors        []string                  `json:"errors,omitempty"`
	PartialData   []string                  `json:"partial_data,omitempty"`
	LastUpdated   string                    `json:"last_updated,omitempty"`
	Collaboration *repo.CollaborationThread `json:"collaboration,omitempty"`
}

func RenderSharedViewContext(view SharedViewContext) []string {
	lines := []string{"## Filters"}
	for _, filter := range view.Filters {
		lines = append(lines, fmt.Sprintf("- %s: %s", filter.Label, filter.Value))
	}
	lines = append(lines, fmt.Sprintf("- Result Count: %d", view.ResultCount))
	if strings.TrimSpace(view.LastUpdated) != "" {
		lines = append(lines, fmt.Sprintf("- Last Updated: %s", view.LastUpdated))
	}
	if view.Collaboration != nil {
		lines = append(lines, "", "## Collaboration", fmt.Sprintf("- Surface: %s", view.Collaboration.Surface))
		for _, comment := range view.Collaboration.Comments {
			lines = append(lines, "- "+comment.Body)
		}
		for _, decision := range view.Collaboration.Decisions {
			lines = append(lines, "- "+decision.Summary)
		}
	}
	return lines
}

type TriageFeedbackRecord struct {
	RunID     string `json:"run_id"`
	Action    string `json:"action"`
	Decision  string `json:"decision"`
	Actor     string `json:"actor"`
	Notes     string `json:"notes,omitempty"`
	Timestamp string `json:"timestamp"`
}

func NewTriageFeedbackRecord(runID, action, decision, actor, notes string) TriageFeedbackRecord {
	return TriageFeedbackRecord{RunID: runID, Action: action, Decision: decision, Actor: actor, Notes: notes, Timestamp: time.Now().UTC().Format(time.RFC3339)}
}

type SimilarityEvidence struct {
	RelatedRunID string  `json:"related_run_id"`
	Score        float64 `json:"score"`
}

type TriageSuggestion struct {
	Label          string               `json:"label"`
	Confidence     float64              `json:"confidence"`
	Action         string               `json:"action"`
	FeedbackStatus string               `json:"feedback_status,omitempty"`
	Evidence       []SimilarityEvidence `json:"evidence,omitempty"`
}

type TriageFinding struct {
	RunID      string          `json:"run_id"`
	Severity   string          `json:"severity"`
	Owner      string          `json:"owner"`
	Status     string          `json:"status"`
	NextAction string          `json:"next_action"`
	Actions    []ConsoleAction `json:"actions,omitempty"`
}

type TriageInboxItem struct {
	RunID       string             `json:"run_id"`
	Suggestions []TriageSuggestion `json:"suggestions,omitempty"`
}

type AutoTriageCenter struct {
	Name           string            `json:"name"`
	Period         string            `json:"period"`
	Findings       []TriageFinding   `json:"findings,omitempty"`
	Inbox          []TriageInboxItem `json:"inbox,omitempty"`
	FlaggedRuns    int               `json:"flagged_runs"`
	InboxSize      int               `json:"inbox_size"`
	SeverityCounts map[string]int    `json:"severity_counts,omitempty"`
	OwnerCounts    map[string]int    `json:"owner_counts,omitempty"`
	FeedbackCounts map[string]int    `json:"feedback_counts,omitempty"`
	Recommendation string            `json:"recommendation"`
}

func BuildAutoTriageCenter(runs []workflowexec.TaskRun, name, period string, feedback []TriageFeedbackRecord) AutoTriageCenter {
	center := AutoTriageCenter{
		Name:           strings.TrimSpace(name),
		Period:         strings.TrimSpace(period),
		SeverityCounts: map[string]int{"critical": 0, "high": 0, "medium": 0},
		OwnerCounts:    map[string]int{"security": 0, "engineering": 0, "operations": 0},
		FeedbackCounts: map[string]int{"accepted": 0, "rejected": 0, "pending": 0},
		Recommendation: "monitor",
	}
	feedbackByRun := map[string]TriageFeedbackRecord{}
	for _, record := range feedback {
		feedbackByRun[record.RunID] = record
		center.FeedbackCounts[record.Decision]++
	}
	for _, run := range runs {
		severity, owner, nextAction, flagged := classifyRun(run)
		if !flagged {
			continue
		}
		center.FlaggedRuns++
		center.SeverityCounts[severity]++
		center.OwnerCounts[owner]++
		actions := buildTriageActions(run.RunID, run.Status, owner)
		finding := TriageFinding{RunID: run.RunID, Severity: severity, Owner: owner, Status: run.Status, NextAction: nextAction, Actions: actions}
		suggestion := TriageSuggestion{
			Label:      suggestionLabel(run.Status),
			Confidence: suggestionConfidence(run.Status),
			Action:     nextAction,
			Evidence:   findSimilarRuns(run, runs),
		}
		if record, ok := feedbackByRun[run.RunID]; ok {
			suggestion.FeedbackStatus = record.Decision
		} else {
			center.FeedbackCounts["pending"]++
		}
		center.Findings = append(center.Findings, finding)
		center.Inbox = append(center.Inbox, TriageInboxItem{RunID: run.RunID, Suggestions: []TriageSuggestion{suggestion}})
	}
	sort.Slice(center.Findings, func(i, j int) bool { return triageRank(center.Findings[i]) < triageRank(center.Findings[j]) })
	sort.Slice(center.Inbox, func(i, j int) bool {
		return triageSortKey(center.Inbox[i].RunID, center.Findings) < triageSortKey(center.Inbox[j].RunID, center.Findings)
	})
	center.InboxSize = len(center.Inbox)
	if center.SeverityCounts["critical"] > 0 || center.SeverityCounts["high"] > 0 {
		center.Recommendation = "immediate-attention"
	}
	return center
}

func RenderAutoTriageCenterReport(center AutoTriageCenter, totalRuns int, view *SharedViewContext) string {
	lines := []string{
		"# Auto Triage Center",
		"",
		fmt.Sprintf("- Name: %s", center.Name),
		fmt.Sprintf("- Period: %s", center.Period),
		fmt.Sprintf("- Total Runs: %d", totalRuns),
		fmt.Sprintf("- Flagged Runs: %d", center.FlaggedRuns),
		fmt.Sprintf("- Inbox Size: %d", center.InboxSize),
		fmt.Sprintf("- Severity Mix: critical=%d high=%d medium=%d", center.SeverityCounts["critical"], center.SeverityCounts["high"], center.SeverityCounts["medium"]),
		fmt.Sprintf("- Feedback Loop: accepted=%d rejected=%d pending=%d", center.FeedbackCounts["accepted"], center.FeedbackCounts["rejected"], center.FeedbackCounts["pending"]),
	}
	if view != nil {
		lines = append(lines, "", "## View State")
		lines = append(lines, renderViewState(*view)...)
	}
	lines = append(lines, "", "## Findings")
	for _, finding := range center.Findings {
		lines = append(lines, fmt.Sprintf("- %s: severity=%s owner=%s status=%s actions=%s", finding.RunID, finding.Severity, finding.Owner, finding.Status, RenderConsoleActions(finding.Actions)))
	}
	lines = append(lines, "", "## Inbox")
	for _, item := range center.Inbox {
		line := fmt.Sprintf("- %s:", item.RunID)
		for _, suggestion := range item.Suggestions {
			line += fmt.Sprintf(" %s confidence=%.2f", suggestion.Label, suggestion.Confidence)
			if len(suggestion.Evidence) > 0 {
				line += fmt.Sprintf(" similar=%s:%.1f", suggestion.Evidence[0].RelatedRunID, suggestion.Evidence[0].Score)
			}
		}
		lines = append(lines, line)
	}
	if view != nil && len(view.PartialData) > 0 {
		lines = append(lines, "", "## Partial Data")
		for _, item := range view.PartialData {
			lines = append(lines, "- "+item)
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

type TakeoverRequest struct {
	RunID             string          `json:"run_id"`
	TaskID            string          `json:"task_id"`
	Team              string          `json:"team"`
	Status            string          `json:"status"`
	RequiredApprovals []string        `json:"required_approvals,omitempty"`
	Actions           []ConsoleAction `json:"actions,omitempty"`
}

type TakeoverQueue struct {
	Name            string            `json:"name"`
	Period          string            `json:"period"`
	Requests        []TakeoverRequest `json:"requests,omitempty"`
	PendingRequests int               `json:"pending_requests"`
	TeamCounts      map[string]int    `json:"team_counts,omitempty"`
	ApprovalCount   int               `json:"approval_count"`
	Recommendation  string            `json:"recommendation"`
}

func BuildTakeoverQueueFromLedger(entries []map[string]any, name, period string) TakeoverQueue {
	queue := TakeoverQueue{
		Name:           strings.TrimSpace(name),
		Period:         strings.TrimSpace(period),
		TeamCounts:     map[string]int{},
		Recommendation: "monitor-queue",
	}
	for _, entry := range entries {
		for _, audit := range mapSlice(entry["audits"]) {
			if stringAny(audit["action"]) != "orchestration.handoff" || stringAny(audit["outcome"]) != "pending" {
				continue
			}
			details := mapAny(audit["details"])
			team := stringAny(details["target_team"])
			request := TakeoverRequest{
				RunID:             stringAny(entry["run_id"]),
				TaskID:            stringAny(entry["task_id"]),
				Team:              team,
				Status:            stringAny(audit["outcome"]),
				RequiredApprovals: stringSliceAny(details["required_approvals"]),
				Actions:           buildTakeoverActions(stringAny(entry["run_id"]), team),
			}
			queue.Requests = append(queue.Requests, request)
			queue.PendingRequests++
			queue.TeamCounts[team]++
			queue.ApprovalCount += len(request.RequiredApprovals)
		}
	}
	sort.Slice(queue.Requests, func(i, j int) bool {
		if queue.Requests[i].Team == queue.Requests[j].Team {
			return queue.Requests[i].RunID < queue.Requests[j].RunID
		}
		return queue.Requests[i].Team < queue.Requests[j].Team
	})
	if queue.TeamCounts["security"] > 0 {
		queue.Recommendation = "expedite-security-review"
	}
	return queue
}

func RenderTakeoverQueueReport(queue TakeoverQueue, totalRuns int, view *SharedViewContext) string {
	lines := []string{
		"# Takeover Queue",
		"",
		fmt.Sprintf("- Name: %s", queue.Name),
		fmt.Sprintf("- Period: %s", queue.Period),
		fmt.Sprintf("- Total Runs: %d", totalRuns),
		fmt.Sprintf("- Pending Requests: %d", queue.PendingRequests),
		fmt.Sprintf("- Team Mix: operations=%d security=%d", queue.TeamCounts["operations"], queue.TeamCounts["security"]),
	}
	if view != nil {
		lines = append(lines, "", "## View State")
		lines = append(lines, renderViewState(*view)...)
	}
	lines = append(lines, "", "## Requests")
	for _, request := range queue.Requests {
		lines = append(lines, fmt.Sprintf("- %s: team=%s status=%s task=%s approvals=%s actions=%s", request.RunID, request.Team, request.Status, request.TaskID, joinedOrNone(request.RequiredApprovals), RenderConsoleActions(request.Actions)))
	}
	if view != nil && len(view.Errors) > 0 {
		lines = append(lines, "", "## Errors")
		for _, item := range view.Errors {
			lines = append(lines, "- "+item)
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

type OrchestrationCanvas struct {
	TaskID             string                            `json:"task_id"`
	RunID              string                            `json:"run_id"`
	CollaborationMode  string                            `json:"collaboration_mode"`
	Departments        []string                          `json:"departments,omitempty"`
	RequiredApprovals  []string                          `json:"required_approvals,omitempty"`
	Tier               string                            `json:"tier"`
	UpgradeRequired    bool                              `json:"upgrade_required"`
	EntitlementStatus  string                            `json:"entitlement_status"`
	BillingModel       string                            `json:"billing_model"`
	EstimatedCostUSD   float64                           `json:"estimated_cost_usd"`
	IncludedUsageUnits int                               `json:"included_usage_units"`
	OverageUsageUnits  int                               `json:"overage_usage_units"`
	OverageCostUSD     float64                           `json:"overage_cost_usd"`
	BlockedDepartments []string                          `json:"blocked_departments,omitempty"`
	HandoffTeam        string                            `json:"handoff_team,omitempty"`
	HandoffStatus      string                            `json:"handoff_status,omitempty"`
	ActiveTools        []string                          `json:"active_tools,omitempty"`
	Collaboration      *workflowexec.CollaborationThread `json:"collaboration,omitempty"`
	Actions            []ConsoleAction                   `json:"actions,omitempty"`
	Recommendation     string                            `json:"recommendation"`
}

func BuildOrchestrationCanvas(run workflowexec.TaskRun, plan workflow.OrchestrationPlan, policy workflow.OrchestrationPolicyDecision, handoff *workflow.HandoffRequest) OrchestrationCanvas {
	canvas := OrchestrationCanvas{
		TaskID:             plan.TaskID,
		RunID:              run.RunID,
		CollaborationMode:  plan.CollaborationMode,
		Departments:        append([]string(nil), plan.Departments()...),
		RequiredApprovals:  append([]string(nil), plan.RequiredApprovals()...),
		Tier:               policy.Tier,
		UpgradeRequired:    policy.UpgradeRequired,
		EntitlementStatus:  policy.EntitlementStatus,
		BillingModel:       policy.BillingModel,
		EstimatedCostUSD:   policy.EstimatedCostUSD,
		IncludedUsageUnits: policy.IncludedUsageUnits,
		OverageUsageUnits:  policy.OverageUsageUnits,
		OverageCostUSD:     policy.OverageCostUSD,
		BlockedDepartments: append([]string(nil), policy.BlockedDepartments...),
		ActiveTools:        extractActiveTools(run.Audits),
		Collaboration:      workflowexec.BuildCollaborationThreadFromAudits(run.Audits, "flow", run.RunID),
	}
	if handoff != nil {
		canvas.HandoffTeam = handoff.TargetTeam
		canvas.HandoffStatus = handoff.Status
	}
	canvas.Actions = buildCanvasActions(canvas.RunID, canvas.HandoffTeam)
	canvas.Recommendation = canvasRecommendation(canvas)
	return canvas
}

func BuildOrchestrationCanvasFromLedgerEntry(entry map[string]any) OrchestrationCanvas {
	canvas := OrchestrationCanvas{
		RunID:         stringAny(entry["run_id"]),
		TaskID:        stringAny(entry["task_id"]),
		Collaboration: workflowexec.BuildCollaborationThreadFromAudits(auditEntriesFromLedger(entry), "flow", stringAny(entry["run_id"])),
	}
	for _, audit := range mapSlice(entry["audits"]) {
		details := mapAny(audit["details"])
		switch stringAny(audit["action"]) {
		case "orchestration.plan":
			canvas.CollaborationMode = stringAny(details["collaboration_mode"])
			canvas.Departments = stringSliceAny(details["departments"])
			canvas.RequiredApprovals = stringSliceAny(details["approvals"])
		case "orchestration.policy":
			canvas.Tier = stringAny(details["tier"])
			canvas.UpgradeRequired = strings.Contains(stringAny(audit["outcome"]), "upgrade")
			canvas.EntitlementStatus = stringAny(details["entitlement_status"])
			canvas.BillingModel = stringAny(details["billing_model"])
			canvas.EstimatedCostUSD = floatAny(details["estimated_cost_usd"])
			canvas.IncludedUsageUnits = int(floatAny(details["included_usage_units"]))
			canvas.OverageUsageUnits = int(floatAny(details["overage_usage_units"]))
			canvas.OverageCostUSD = floatAny(details["overage_cost_usd"])
			canvas.BlockedDepartments = stringSliceAny(details["blocked_departments"])
		case "orchestration.handoff":
			canvas.HandoffTeam = stringAny(details["target_team"])
			canvas.HandoffStatus = stringAny(audit["outcome"])
		case "tool.invoke":
			tool := stringAny(details["tool"])
			if tool != "" && !containsString(canvas.ActiveTools, tool) {
				canvas.ActiveTools = append(canvas.ActiveTools, tool)
			}
		}
	}
	sort.Strings(canvas.ActiveTools)
	canvas.Actions = buildCanvasActions(canvas.RunID, canvas.HandoffTeam)
	canvas.Recommendation = canvasRecommendation(canvas)
	return canvas
}

func RenderOrchestrationCanvas(canvas OrchestrationCanvas) string {
	lines := []string{
		"# Orchestration Canvas",
		"",
		fmt.Sprintf("- Task ID: %s", canvas.TaskID),
		fmt.Sprintf("- Run ID: %s", canvas.RunID),
		fmt.Sprintf("- Tier: %s", canvas.Tier),
		fmt.Sprintf("- Entitlement Status: %s", canvas.EntitlementStatus),
		fmt.Sprintf("- Billing Model: %s", canvas.BillingModel),
		fmt.Sprintf("- Estimated Cost (USD): %.2f", canvas.EstimatedCostUSD),
		fmt.Sprintf("- Handoff Team: %s", valueOr(canvas.HandoffTeam, "none")),
		fmt.Sprintf("- Recommendation: %s", canvas.Recommendation),
		"",
		"## Actions",
	}
	for _, action := range canvas.Actions {
		lines = append(lines, RenderConsoleActions([]ConsoleAction{action}))
	}
	if canvas.Collaboration != nil {
		lines = append(lines, "", "## Collaboration")
		for _, comment := range canvas.Collaboration.Comments {
			lines = append(lines, "- "+comment.Body)
		}
		for _, decision := range canvas.Collaboration.Decisions {
			lines = append(lines, "- "+decision.Summary)
		}
	}
	return strings.Join(lines, "\n") + "\n"
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
	Recommendation        string                `json:"recommendation"`
}

func BuildOrchestrationPortfolio(canvases []OrchestrationCanvas, name, period string, takeoverQueue *TakeoverQueue) OrchestrationPortfolio {
	portfolio := OrchestrationPortfolio{
		Name:               strings.TrimSpace(name),
		Period:             strings.TrimSpace(period),
		Canvases:           append([]OrchestrationCanvas(nil), canvases...),
		TakeoverQueue:      takeoverQueue,
		CollaborationModes: map[string]int{},
		TierCounts:         map[string]int{},
		EntitlementCounts:  map[string]int{},
		BillingModelCounts: map[string]int{},
		Recommendation:     "continue",
	}
	for _, canvas := range canvases {
		portfolio.TotalRuns++
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
	if takeoverQueue != nil && takeoverQueue.TeamCounts["security"] > 0 {
		portfolio.Recommendation = "stabilize-security-takeovers"
	}
	return portfolio
}

func BuildOrchestrationPortfolioFromLedger(entries []map[string]any, name, period string) OrchestrationPortfolio {
	canvases := make([]OrchestrationCanvas, 0, len(entries))
	for _, entry := range entries {
		canvases = append(canvases, BuildOrchestrationCanvasFromLedgerEntry(entry))
	}
	queue := BuildTakeoverQueueFromLedger(entries, "Cross-Team Takeovers", period)
	return BuildOrchestrationPortfolio(canvases, name, period, &queue)
}

func RenderOrchestrationPortfolioReport(portfolio OrchestrationPortfolio, view *SharedViewContext) string {
	lines := []string{
		"# Orchestration Portfolio Report",
		"",
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
	if view != nil {
		lines = append(lines, "", "## View State")
		lines = append(lines, renderViewState(*view)...)
	}
	lines = append(lines, "", "## Runs")
	for _, canvas := range portfolio.Canvases {
		lines = append(lines, fmt.Sprintf("- %s: mode=%s tier=%s entitlement=%s billing=%s estimated_cost_usd=%.2f overage_cost_usd=%.2f upgrade_required=%t handoff=%s actions=%s", canvas.RunID, canvas.CollaborationMode, canvas.Tier, canvas.EntitlementStatus, canvas.BillingModel, canvas.EstimatedCostUSD, canvas.OverageCostUSD, canvas.UpgradeRequired, valueOr(canvas.HandoffTeam, "none"), RenderConsoleActions(canvas.Actions)))
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderOrchestrationOverviewPage(portfolio OrchestrationPortfolio) string {
	var b strings.Builder
	b.WriteString("<html><head><title>Orchestration Overview</title></head><body>\n")
	b.WriteString("<h1>" + html.EscapeString(portfolio.Name) + "</h1>\n")
	b.WriteString("<p>Estimated Cost</p>\n")
	if portfolio.TakeoverQueue != nil {
		b.WriteString("<p>pending=" + fmt.Sprintf("%d", portfolio.TakeoverQueue.PendingRequests) + " recommendation=" + html.EscapeString(portfolio.TakeoverQueue.Recommendation) + "</p>\n")
	}
	for _, canvas := range portfolio.Canvases {
		b.WriteString("<p>" + html.EscapeString(canvas.RunID) + " " + html.EscapeString(canvas.BillingModel) + " " + html.EscapeString(RenderConsoleActions(canvas.Actions)) + "</p>\n")
	}
	b.WriteString("</body></html>\n")
	return b.String()
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
	Recommendation          string             `json:"recommendation"`
}

func BuildBillingEntitlementsPage(portfolio OrchestrationPortfolio, workspaceName, planName, billingPeriod string) BillingEntitlementsPage {
	page := BillingEntitlementsPage{
		WorkspaceName:      workspaceName,
		PlanName:           planName,
		BillingPeriod:      billingPeriod,
		EntitlementCounts:  map[string]int{},
		BillingModelCounts: map[string]int{},
		Recommendation:     "monitor-usage",
	}
	blockedSet := map[string]struct{}{}
	for _, canvas := range portfolio.Canvases {
		page.RunCount++
		page.TotalIncludedUsageUnits += canvas.IncludedUsageUnits
		page.TotalOverageUsageUnits += canvas.OverageUsageUnits
		page.TotalEstimatedCostUSD += canvas.EstimatedCostUSD
		page.TotalOverageCostUSD += canvas.OverageCostUSD
		page.EntitlementCounts[canvas.EntitlementStatus]++
		page.BillingModelCounts[canvas.BillingModel]++
		if canvas.UpgradeRequired {
			page.UpgradeRequiredCount++
		}
		for _, blocked := range canvas.BlockedDepartments {
			blockedSet[blocked] = struct{}{}
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
	page.BlockedCapabilities = sortedKeys(blockedSet)
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
		"",
		fmt.Sprintf("- Workspace: %s", page.WorkspaceName),
		fmt.Sprintf("- Plan: %s", page.PlanName),
		fmt.Sprintf("- Billing Period: %s", page.BillingPeriod),
		fmt.Sprintf("- Overage Cost (USD): %.2f", page.TotalOverageCostUSD),
	}
	for _, charge := range page.Charges {
		lines = append(lines, fmt.Sprintf("- %s: task=%s entitlement=%s billing=%s", charge.RunID, charge.TaskID, charge.EntitlementStatus, charge.BillingModel))
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderBillingEntitlementsPage(page BillingEntitlementsPage) string {
	var b strings.Builder
	b.WriteString("<html><head><title>Billing & Entitlements</title></head><body>\n")
	b.WriteString("<h1>" + html.EscapeString(page.WorkspaceName) + "</h1>\n")
	b.WriteString("<p>" + html.EscapeString(page.PlanName) + " plan for " + html.EscapeString(page.BillingPeriod) + "</p>\n")
	b.WriteString("<h2>Charge Feed</h2>\n")
	for _, charge := range page.Charges {
		b.WriteString("<p>" + html.EscapeString(charge.BillingModel) + "</p>\n")
	}
	b.WriteString("</body></html>\n")
	return b.String()
}

func round1(value float64) float64 {
	if value >= 0 {
		return float64(int(value*10+0.5)) / 10
	}
	return float64(int(value*10-0.5)) / 10
}

func joinedOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "_", "-")
	value = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-':
			return r
		default:
			return -1
		}
	}, value)
	return strings.Trim(value, "-")
}

func sortedBoolMapKeys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func suggestionLabel(status string) string {
	if status == "failed" {
		return "replay candidate"
	}
	return "approval escalation"
}

func suggestionConfidence(status string) float64 {
	if status == "failed" {
		return 0.8
	}
	return 0.6
}

func classifyRun(run workflowexec.TaskRun) (severity, owner, nextAction string, flagged bool) {
	switch run.Status {
	case "failed":
		return "critical", "engineering", "replay run and inspect tool failures", true
	case "needs-approval":
		return "high", "security", "request approval and queue security review", true
	default:
		return "medium", "operations", "", false
	}
}

func buildTriageActions(target, status, owner string) []ConsoleAction {
	retryEnabled := status == "failed"
	escalateEnabled := owner != "security"
	return []ConsoleAction{
		{ActionID: "drill-down", Label: "Drill Down", Target: target, Enabled: true},
		{ActionID: "export", Label: "Export", Target: target, Enabled: true},
		{ActionID: "add-note", Label: "Add Note", Target: target, Enabled: true},
		{ActionID: "escalate", Label: "Escalate", Target: target, Enabled: escalateEnabled, Reason: disabledReason(escalateEnabled, "security takeovers are already escalated")},
		{ActionID: "retry", Label: "Retry", Target: target, Enabled: retryEnabled, Reason: disabledReason(retryEnabled, "retry available after owner review")},
		{ActionID: "pause", Label: "Pause", Target: target, Enabled: status != "failed", Reason: disabledReason(status != "failed", "failed runs should be replayed instead of paused")},
		{ActionID: "resolve", Label: "Resolve", Target: target, Enabled: status == "failed", Reason: disabledReason(status == "failed", "awaiting owner review")},
	}
}

func buildTakeoverActions(target, team string) []ConsoleAction {
	escalateEnabled := team != "security"
	return []ConsoleAction{
		{ActionID: "drill-down", Label: "Drill Down", Target: target, Enabled: true},
		{ActionID: "export", Label: "Export", Target: target, Enabled: true},
		{ActionID: "add-note", Label: "Add Note", Target: target, Enabled: true},
		{ActionID: "escalate", Label: "Escalate", Target: target, Enabled: escalateEnabled, Reason: disabledReason(escalateEnabled, "security takeovers are already escalated")},
	}
}

func triageRank(finding TriageFinding) int {
	switch finding.Severity {
	case "critical":
		return 0
	case "high":
		return 1
	default:
		return 2
	}
}

func triageSortKey(runID string, findings []TriageFinding) string {
	rank := 9
	for _, finding := range findings {
		if finding.RunID == runID {
			rank = triageRank(finding)
			break
		}
	}
	return fmt.Sprintf("%02d:%s", rank, runID)
}

func findSimilarRuns(run workflowexec.TaskRun, runs []workflowexec.TaskRun) []SimilarityEvidence {
	var evidence []SimilarityEvidence
	for _, candidate := range runs {
		if candidate.RunID == run.RunID || candidate.Status != run.Status || candidate.Medium != run.Medium || candidate.Summary != run.Summary {
			continue
		}
		evidence = append(evidence, SimilarityEvidence{RelatedRunID: candidate.RunID, Score: 0.8})
	}
	sort.Slice(evidence, func(i, j int) bool { return evidence[i].RelatedRunID < evidence[j].RelatedRunID })
	return evidence
}

func renderViewState(view SharedViewContext) []string {
	state := "ready"
	summary := "Filtered data loaded."
	switch {
	case view.Loading:
		state = "loading"
		summary = "Data is still loading."
	case len(view.Errors) > 0:
		state = "error"
		summary = "Unable to load data for the current filters."
	case len(view.PartialData) > 0:
		state = "partial-data"
		summary = "Some data sources are still refreshing."
	case view.ResultCount == 0:
		state = "empty"
		summary = "No records match the current filters."
	}
	lines := []string{fmt.Sprintf("- State: %s", state), fmt.Sprintf("- Summary: %s", summary)}
	for _, filter := range view.Filters {
		lines = append(lines, fmt.Sprintf("- %s: %s", filter.Label, filter.Value))
	}
	return lines
}

func mapSlice(value any) []map[string]any {
	raw, ok := value.([]any)
	if !ok {
		if typed, ok := value.([]map[string]any); ok {
			return typed
		}
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if typed, ok := item.(map[string]any); ok {
			out = append(out, typed)
		}
	}
	return out
}

func mapAny(value any) map[string]any {
	typed, _ := value.(map[string]any)
	return typed
}

func stringAny(value any) string {
	typed, _ := value.(string)
	return typed
}

func stringSliceAny(value any) []string {
	raw, ok := value.([]any)
	if ok {
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				out = append(out, text)
			}
		}
		return out
	}
	if typed, ok := value.([]string); ok {
		return append([]string(nil), typed...)
	}
	return nil
}

func floatAny(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	default:
		return 0
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func extractActiveTools(audits []workflowexec.AuditEntry) []string {
	tools := make([]string, 0)
	for _, audit := range audits {
		if audit.Action != "tool.invoke" || audit.Outcome != "success" {
			continue
		}
		tool := ""
		if audit.Details != nil {
			tool = stringAny(audit.Details["tool"])
		}
		if tool != "" && !containsString(tools, tool) {
			tools = append(tools, tool)
		}
	}
	sort.Strings(tools)
	return tools
}

func buildCanvasActions(target, handoffTeam string) []ConsoleAction {
	escalateEnabled := strings.TrimSpace(handoffTeam) != ""
	if handoffTeam == "security" {
		escalateEnabled = false
	}
	return []ConsoleAction{
		{ActionID: "drill-down", Label: "Drill Down", Target: target, Enabled: true},
		{ActionID: "export", Label: "Export", Target: target, Enabled: true},
		{ActionID: "add-note", Label: "Add Note", Target: target, Enabled: true},
		{ActionID: "escalate", Label: "Escalate", Target: target, Enabled: escalateEnabled, Reason: disabledReason(escalateEnabled, "security takeovers are already escalated")},
		{ActionID: "retry", Label: "Retry", Target: target, Enabled: false, Reason: "retry available after owner review"},
	}
}

func canvasRecommendation(canvas OrchestrationCanvas) string {
	if canvas.Collaboration != nil && len(canvas.Collaboration.Comments) > 0 {
		return "resolve-flow-comments"
	}
	if canvas.UpgradeRequired {
		return "resolve-entitlement-gap"
	}
	if canvas.HandoffTeam == "security" {
		return "review-security-takeover"
	}
	return "continue"
}

func auditEntriesFromLedger(entry map[string]any) []workflowexec.AuditEntry {
	raw := mapSlice(entry["audits"])
	out := make([]workflowexec.AuditEntry, 0, len(raw))
	for _, item := range raw {
		out = append(out, workflowexec.AuditEntry{
			Action:    stringAny(item["action"]),
			Actor:     stringAny(item["actor"]),
			Outcome:   stringAny(item["outcome"]),
			Timestamp: stringAny(item["timestamp"]),
			Details:   mapAny(item["details"]),
		})
	}
	return out
}

func renderIntMap(values map[string]int) string {
	if len(values) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(values))
	for _, key := range sortedMapKeys(values) {
		parts = append(parts, fmt.Sprintf("%s=%d", key, values[key]))
	}
	return strings.Join(parts, " ")
}
