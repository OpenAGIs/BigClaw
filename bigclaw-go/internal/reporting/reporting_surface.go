package reporting

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bigclaw-go/internal/collaboration"
)

type DocumentationArtifact struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (a DocumentationArtifact) Available() bool {
	return ValidationReportExists(a.Path)
}

type LaunchChecklistItem struct {
	Name     string   `json:"name"`
	Evidence []string `json:"evidence,omitempty"`
}

type LaunchChecklist struct {
	IssueID       string                  `json:"issue_id"`
	Documentation []DocumentationArtifact `json:"documentation,omitempty"`
	Items         []LaunchChecklistItem   `json:"items,omitempty"`
}

func (c LaunchChecklist) DocumentationStatus() map[string]bool {
	status := make(map[string]bool, len(c.Documentation))
	for _, artifact := range c.Documentation {
		status[artifact.Name] = artifact.Available()
	}
	return status
}

func (c LaunchChecklist) ItemCompleted(item LaunchChecklistItem) bool {
	status := c.DocumentationStatus()
	if len(item.Evidence) == 0 {
		return true
	}
	for _, evidence := range item.Evidence {
		if !status[evidence] {
			return false
		}
	}
	return true
}

func (c LaunchChecklist) CompletedItems() int {
	total := 0
	for _, item := range c.Items {
		if c.ItemCompleted(item) {
			total++
		}
	}
	return total
}

func (c LaunchChecklist) MissingDocumentation() []string {
	missing := make([]string, 0)
	for _, artifact := range c.Documentation {
		if !artifact.Available() {
			missing = append(missing, artifact.Name)
		}
	}
	return missing
}

func (c LaunchChecklist) Ready() bool {
	if len(c.MissingDocumentation()) > 0 {
		return false
	}
	for _, item := range c.Items {
		if !c.ItemCompleted(item) {
			return false
		}
	}
	return true
}

type FinalDeliveryChecklist struct {
	IssueID                  string                  `json:"issue_id"`
	RequiredOutputs          []DocumentationArtifact `json:"required_outputs,omitempty"`
	RecommendedDocumentation []DocumentationArtifact `json:"recommended_documentation,omitempty"`
}

func (c FinalDeliveryChecklist) RequiredOutputStatus() map[string]bool {
	status := make(map[string]bool, len(c.RequiredOutputs))
	for _, artifact := range c.RequiredOutputs {
		status[artifact.Name] = artifact.Available()
	}
	return status
}

func (c FinalDeliveryChecklist) RecommendedDocumentationStatus() map[string]bool {
	status := make(map[string]bool, len(c.RecommendedDocumentation))
	for _, artifact := range c.RecommendedDocumentation {
		status[artifact.Name] = artifact.Available()
	}
	return status
}

func (c FinalDeliveryChecklist) GeneratedRequiredOutputs() int {
	total := 0
	for _, artifact := range c.RequiredOutputs {
		if artifact.Available() {
			total++
		}
	}
	return total
}

func (c FinalDeliveryChecklist) GeneratedRecommendedDocumentation() int {
	total := 0
	for _, artifact := range c.RecommendedDocumentation {
		if artifact.Available() {
			total++
		}
	}
	return total
}

func (c FinalDeliveryChecklist) MissingRequiredOutputs() []string {
	missing := make([]string, 0)
	for _, artifact := range c.RequiredOutputs {
		if !artifact.Available() {
			missing = append(missing, artifact.Name)
		}
	}
	return missing
}

func (c FinalDeliveryChecklist) MissingRecommendedDocumentation() []string {
	missing := make([]string, 0)
	for _, artifact := range c.RecommendedDocumentation {
		if !artifact.Available() {
			missing = append(missing, artifact.Name)
		}
	}
	return missing
}

func (c FinalDeliveryChecklist) Ready() bool {
	return len(c.MissingRequiredOutputs()) == 0
}

type IssueClosureDecision struct {
	IssueID    string `json:"issue_id"`
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

func (s NarrativeSection) Ready() bool {
	return strings.TrimSpace(s.Heading) != "" && strings.TrimSpace(s.Body) != ""
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
		if !section.Ready() {
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

func (s ReportStudio) ExportSlug() string {
	slug := strings.ToLower(strings.TrimSpace(s.Name))
	slug = strings.ReplaceAll(slug, "_", "-")
	fields := strings.FieldsFunc(slug, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
	if len(fields) == 0 {
		return "report-studio"
	}
	return strings.Join(fields, "-")
}

type ReportStudioArtifacts struct {
	RootDir      string `json:"root_dir"`
	MarkdownPath string `json:"markdown_path"`
	HTMLPath     string `json:"html_path"`
	TextPath     string `json:"text_path"`
}

type PilotMetric struct {
	Name           string  `json:"name"`
	Baseline       float64 `json:"baseline"`
	Current        float64 `json:"current"`
	Target         float64 `json:"target"`
	Unit           string  `json:"unit,omitempty"`
	HigherIsBetter bool    `json:"higher_is_better"`
}

func (m PilotMetric) Delta() float64 {
	return m.Current - m.Baseline
}

func (m PilotMetric) MetTarget() bool {
	if m.HigherIsBetter {
		return m.Current >= m.Target
	}
	return m.Current <= m.Target
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

func (s PilotScorecard) MonthlyNetValue() float64 {
	return s.MonthlyBenefit - s.MonthlyCost
}

func (s PilotScorecard) AnnualizedROI() float64 {
	totalCost := s.ImplementationCost + (s.MonthlyCost * 12)
	if totalCost <= 0 {
		return 0
	}
	annualGain := (s.MonthlyBenefit * 12) - totalCost
	return (annualGain / totalCost) * 100
}

func (s PilotScorecard) PaybackMonths() *float64 {
	if s.MonthlyNetValue() <= 0 {
		return nil
	}
	if s.ImplementationCost <= 0 {
		value := 0.0
		return &value
	}
	value := roundTenth(s.ImplementationCost / s.MonthlyNetValue())
	return &value
}

func (s PilotScorecard) MetricsMet() int {
	total := 0
	for _, metric := range s.Metrics {
		if metric.MetTarget() {
			total++
		}
	}
	return total
}

func (s PilotScorecard) Recommendation() string {
	benchmarkOK := s.BenchmarkPassed == nil || *s.BenchmarkPassed
	if len(s.Metrics) > 0 && s.MetricsMet() == len(s.Metrics) && s.AnnualizedROI() > 0 && benchmarkOK {
		return "go"
	}
	if s.AnnualizedROI() > 0 || s.MetricsMet() > 0 {
		return "iterate"
	}
	return "hold"
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
	if len(p.Scorecards) > 0 && counts["go"] == len(p.Scorecards) {
		return "scale"
	}
	if counts["go"] > 0 || counts["iterate"] > 0 {
		return "continue"
	}
	return "stop"
}

func utcNowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func BuildLaunchChecklist(issueID string, documentation []DocumentationArtifact, items []LaunchChecklistItem) LaunchChecklist {
	return LaunchChecklist{IssueID: issueID, Documentation: documentation, Items: items}
}

func BuildFinalDeliveryChecklist(issueID string, requiredOutputs []DocumentationArtifact, recommendedDocumentation []DocumentationArtifact) FinalDeliveryChecklist {
	return FinalDeliveryChecklist{
		IssueID:                  issueID,
		RequiredOutputs:          requiredOutputs,
		RecommendedDocumentation: recommendedDocumentation,
	}
}

func ValidationReportExists(reportPath string) bool {
	if strings.TrimSpace(reportPath) == "" {
		return false
	}
	info, err := os.Stat(reportPath)
	if err != nil || info.IsDir() {
		return false
	}
	body, err := os.ReadFile(reportPath)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(body)) != ""
}

func EvaluateIssueClosure(issueID, reportPath string, validationPassed bool, launchChecklist *LaunchChecklist, finalDeliveryChecklist *FinalDeliveryChecklist) IssueClosureDecision {
	resolvedPath := strings.TrimSpace(reportPath)
	if !ValidationReportExists(reportPath) {
		return IssueClosureDecision{IssueID: issueID, Allowed: false, Reason: "validation report required before closing issue", ReportPath: resolvedPath}
	}
	if !validationPassed {
		return IssueClosureDecision{IssueID: issueID, Allowed: false, Reason: "validation failed; issue must remain open", ReportPath: resolvedPath}
	}
	if finalDeliveryChecklist != nil && !finalDeliveryChecklist.Ready() {
		return IssueClosureDecision{IssueID: issueID, Allowed: false, Reason: "final delivery checklist incomplete; required outputs missing", ReportPath: resolvedPath}
	}
	if launchChecklist != nil && !launchChecklist.Ready() {
		return IssueClosureDecision{IssueID: issueID, Allowed: false, Reason: "launch checklist incomplete; linked documentation missing or empty", ReportPath: resolvedPath}
	}
	if finalDeliveryChecklist != nil {
		return IssueClosureDecision{IssueID: issueID, Allowed: true, Reason: "validation report and final delivery checklist requirements satisfied; issue can be closed", ReportPath: resolvedPath}
	}
	return IssueClosureDecision{IssueID: issueID, Allowed: true, Reason: "validation report and launch checklist requirements satisfied; issue can be closed", ReportPath: resolvedPath}
}

func RenderIssueValidationReport(issueID, version, environment, summary string) string {
	return fmt.Sprintf("# Issue Validation Report\n\n- Issue ID: %s\n- 版本号: %s\n- 测试环境: %s\n- 生成时间: %s\n\n## 结论\n\n%s\n", issueID, version, environment, utcNowISO(), summary)
}

func RenderReportStudioReport(studio ReportStudio) string {
	lines := []string{
		"# Report Studio",
		"",
		"- Name: " + studio.Name,
		"- Issue ID: " + studio.IssueID,
		"- Audience: " + studio.Audience,
		"- Period: " + studio.Period,
		fmt.Sprintf("- Sections: %d", len(studio.Sections)),
		"- Recommendation: " + studio.Recommendation(),
		"",
		"## Narrative Summary",
		"",
		firstNonEmpty(studio.Summary, "No summary drafted."),
		"",
		"## Sections",
		"",
	}
	if len(studio.Sections) == 0 {
		lines = append(lines, "- None", "")
	} else {
		for _, section := range studio.Sections {
			lines = append(lines,
				"### "+section.Heading,
				"",
				firstNonEmpty(section.Body, "No narrative drafted."),
				"",
				"- Evidence: "+joinOrNone(section.Evidence),
				"- Callouts: "+joinOrNone(section.Callouts),
				"",
			)
		}
	}
	lines = append(lines, "## Action Items", "")
	if len(studio.ActionItems) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, item := range studio.ActionItems {
			lines = append(lines, "- "+item)
		}
	}
	lines = append(lines, "", "## Sources", "")
	if len(studio.SourceReports) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, report := range studio.SourceReports {
			lines = append(lines, "- "+report)
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderReportStudioPlainText(studio ReportStudio) string {
	lines := []string{
		fmt.Sprintf("%s (%s)", studio.Name, studio.IssueID),
		"Audience: " + studio.Audience,
		"Period: " + studio.Period,
		"Recommendation: " + studio.Recommendation(),
		"",
		firstNonEmpty(studio.Summary, "No summary drafted."),
		"",
	}
	for _, section := range studio.Sections {
		lines = append(lines, strings.ToUpper(section.Heading), firstNonEmpty(section.Body, "No narrative drafted."))
		if len(section.Callouts) > 0 {
			lines = append(lines, "Callouts: "+strings.Join(section.Callouts, "; "))
		}
		if len(section.Evidence) > 0 {
			lines = append(lines, "Evidence: "+strings.Join(section.Evidence, "; "))
		}
		lines = append(lines, "")
	}
	if len(studio.ActionItems) > 0 {
		lines = append(lines, "Action Items:")
		for _, item := range studio.ActionItems {
			lines = append(lines, "- "+item)
		}
		lines = append(lines, "")
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
}

func RenderReportStudioHTML(studio ReportStudio) string {
	sections := make([]string, 0, len(studio.Sections))
	for _, section := range studio.Sections {
		sections = append(sections, fmt.Sprintf(`
        <section class="section">
          <h2>%s</h2>
          <p>%s</p>
          <p class="meta">Evidence: %s</p>
          <p class="meta">Callouts: %s</p>
        </section>
        `, html.EscapeString(section.Heading), html.EscapeString(section.Body), html.EscapeString(joinOrNone(section.Evidence)), html.EscapeString(joinOrNone(section.Callouts))))
	}
	sectionHTML := strings.Join(sections, "")
	if sectionHTML == "" {
		sectionHTML = `<section class="section"><p>No sections drafted.</p></section>`
	}
	actionHTML := "<li>None</li>"
	if len(studio.ActionItems) > 0 {
		items := make([]string, 0, len(studio.ActionItems))
		for _, item := range studio.ActionItems {
			items = append(items, "<li>"+html.EscapeString(item)+"</li>")
		}
		actionHTML = strings.Join(items, "")
	}
	sourceHTML := "<li>None</li>"
	if len(studio.SourceReports) > 0 {
		items := make([]string, 0, len(studio.SourceReports))
		for _, item := range studio.SourceReports {
			items = append(items, "<li>"+html.EscapeString(item)+"</li>")
		}
		sourceHTML = strings.Join(items, "")
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>%s</title>
    <style>
      body { font-family: Georgia, 'Times New Roman', serif; margin: 40px auto; max-width: 840px; color: #1f2933; line-height: 1.6; }
      h1, h2 { font-family: 'Avenir Next', 'Segoe UI', sans-serif; }
      .meta { color: #52606d; font-size: 0.95rem; }
      .summary { padding: 16px 20px; background: #f7f3e8; border-left: 4px solid #c58b32; }
      .section { margin-top: 28px; }
    </style>
  </head>
  <body>
    <header>
      <p class="meta">%s · %s · %s</p>
      <h1>%s</h1>
      <p class="meta">Recommendation: %s</p>
    </header>
    <section class="summary">
      <h2>Narrative Summary</h2>
      <p>%s</p>
    </section>
    %s
    <section class="section">
      <h2>Action Items</h2>
      <ul>%s</ul>
    </section>
    <section class="section">
      <h2>Sources</h2>
      <ul>%s</ul>
    </section>
  </body>
</html>
`, html.EscapeString(studio.Name), html.EscapeString(studio.IssueID), html.EscapeString(studio.Audience), html.EscapeString(studio.Period), html.EscapeString(studio.Name), html.EscapeString(studio.Recommendation()), html.EscapeString(firstNonEmpty(studio.Summary, "No summary drafted.")), sectionHTML, actionHTML, sourceHTML)
}

func WriteReportStudioBundle(rootDir string, studio ReportStudio) (ReportStudioArtifacts, error) {
	markdownPath := filepath.Join(rootDir, studio.ExportSlug()+".md")
	htmlPath := filepath.Join(rootDir, studio.ExportSlug()+".html")
	textPath := filepath.Join(rootDir, studio.ExportSlug()+".txt")
	if err := WriteReport(markdownPath, RenderReportStudioReport(studio)); err != nil {
		return ReportStudioArtifacts{}, err
	}
	if err := WriteReport(htmlPath, RenderReportStudioHTML(studio)); err != nil {
		return ReportStudioArtifacts{}, err
	}
	if err := WriteReport(textPath, RenderReportStudioPlainText(studio)); err != nil {
		return ReportStudioArtifacts{}, err
	}
	return ReportStudioArtifacts{RootDir: rootDir, MarkdownPath: markdownPath, HTMLPath: htmlPath, TextPath: textPath}, nil
}

func RenderLaunchChecklistReport(checklist LaunchChecklist) string {
	lines := []string{
		"# Launch Checklist",
		"",
		"- Issue ID: " + checklist.IssueID,
		fmt.Sprintf("- Linked Documentation: %d", len(checklist.Documentation)),
		fmt.Sprintf("- Completed Items: %d/%d", checklist.CompletedItems(), len(checklist.Items)),
		fmt.Sprintf("- Ready: %t", checklist.Ready()),
		"",
		"## Documentation",
		"",
	}
	if len(checklist.Documentation) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, artifact := range checklist.Documentation {
			lines = append(lines, fmt.Sprintf("- %s: available=%t path=%s", artifact.Name, artifact.Available(), artifact.Path))
		}
	}
	lines = append(lines, "", "## Checklist", "")
	if len(checklist.Items) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, item := range checklist.Items {
			lines = append(lines, fmt.Sprintf("- %s: completed=%t evidence=%s", item.Name, checklist.ItemCompleted(item), firstNonEmpty(strings.Join(item.Evidence, ", "), "none")))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderFinalDeliveryChecklistReport(checklist FinalDeliveryChecklist) string {
	lines := []string{
		"# Final Delivery Checklist",
		"",
		"- Issue ID: " + checklist.IssueID,
		fmt.Sprintf("- Required Outputs Generated: %d/%d", checklist.GeneratedRequiredOutputs(), len(checklist.RequiredOutputs)),
		fmt.Sprintf("- Recommended Docs Generated: %d/%d", checklist.GeneratedRecommendedDocumentation(), len(checklist.RecommendedDocumentation)),
		fmt.Sprintf("- Ready: %t", checklist.Ready()),
		"",
		"## Required Outputs",
		"",
	}
	if len(checklist.RequiredOutputs) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, artifact := range checklist.RequiredOutputs {
			lines = append(lines, fmt.Sprintf("- %s: available=%t path=%s", artifact.Name, artifact.Available(), artifact.Path))
		}
	}
	lines = append(lines, "", "## Recommended Documentation", "")
	if len(checklist.RecommendedDocumentation) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, artifact := range checklist.RecommendedDocumentation {
			lines = append(lines, fmt.Sprintf("- %s: available=%t path=%s", artifact.Name, artifact.Available(), artifact.Path))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderPilotScorecard(scorecard PilotScorecard) string {
	lines := []string{
		"# Pilot Scorecard",
		"",
		"- Issue ID: " + scorecard.IssueID,
		"- Customer: " + scorecard.Customer,
		"- Period: " + scorecard.Period,
		"- Recommendation: " + scorecard.Recommendation(),
		fmt.Sprintf("- Metrics Met: %d/%d", scorecard.MetricsMet(), len(scorecard.Metrics)),
		fmt.Sprintf("- Monthly Net Value: %.2f", scorecard.MonthlyNetValue()),
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
	if len(scorecard.Metrics) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, metric := range scorecard.Metrics {
			comparator := ">="
			if !metric.HigherIsBetter {
				comparator = "<="
			}
			unitSuffix := ""
			if metric.Unit != "" {
				unitSuffix = " " + metric.Unit
			}
			lines = append(lines, fmt.Sprintf("- %s: baseline=%v%s current=%v%s target%s%v%s delta=%+.2f%s met=%t", metric.Name, metric.Baseline, unitSuffix, metric.Current, unitSuffix, comparator, metric.Target, unitSuffix, metric.Delta(), unitSuffix, metric.MetTarget()))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderPilotPortfolioReport(portfolio PilotPortfolio) string {
	counts := portfolio.RecommendationCounts()
	lines := []string{
		"# Pilot Portfolio Report",
		"",
		"- Portfolio: " + portfolio.Name,
		"- Period: " + portfolio.Period,
		fmt.Sprintf("- Scorecards: %d", len(portfolio.Scorecards)),
		"- Recommendation: " + portfolio.Recommendation(),
		fmt.Sprintf("- Total Monthly Net Value: %.2f", portfolio.TotalMonthlyNetValue()),
		fmt.Sprintf("- Average ROI: %.1f%%", portfolio.AverageROI()),
		fmt.Sprintf("- Recommendation Mix: go=%d iterate=%d hold=%d", counts["go"], counts["iterate"], counts["hold"]),
		"",
		"## Customers",
		"",
	}
	if len(portfolio.Scorecards) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, scorecard := range portfolio.Scorecards {
			benchmark := "n/a"
			if scorecard.BenchmarkScore != nil {
				benchmark = fmt.Sprintf("%d", *scorecard.BenchmarkScore)
			}
			lines = append(lines, fmt.Sprintf("- %s: recommendation=%s roi=%.1f%% monthly-net=%.2f benchmark=%s", scorecard.Customer, scorecard.Recommendation(), scorecard.AnnualizedROI(), scorecard.MonthlyNetValue(), benchmark))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func (v SharedViewContext) State() string {
	switch {
	case v.Loading:
		return "loading"
	case len(v.Errors) > 0 && v.ResultCount == 0:
		return "error"
	case v.ResultCount == 0 && len(v.PartialData) == 0:
		return "empty"
	case len(v.Errors) > 0 || len(v.PartialData) > 0:
		return "partial-data"
	default:
		return "ready"
	}
}

func (v SharedViewContext) Summary() string {
	switch v.State() {
	case "loading":
		return "Loading data for the current filters."
	case "error":
		return "Unable to load data for the current filters."
	case "empty":
		return firstNonEmpty(v.EmptyMessage, "No records match the current filters.")
	case "partial-data":
		return "Showing partial data while one or more sources are unavailable."
	default:
		return "Data is current for the selected filters."
	}
}

func RenderSharedViewContext(view *SharedViewContext) []string {
	if view == nil {
		return nil
	}
	lines := []string{
		"## View State",
		"",
		"- State: " + view.State(),
		"- Summary: " + view.Summary(),
		fmt.Sprintf("- Result Count: %d", view.ResultCount),
	}
	if strings.TrimSpace(view.LastUpdated) != "" {
		lines = append(lines, "- Last Updated: "+view.LastUpdated)
	}
	lines = append(lines, "", "## Filters", "")
	if len(view.Filters) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, filter := range view.Filters {
			lines = append(lines, fmt.Sprintf("- %s: %s", filter.Label, filter.Value))
		}
	}
	if len(view.Errors) > 0 {
		lines = append(lines, "", "## Errors", "")
		for _, message := range view.Errors {
			lines = append(lines, "- "+message)
		}
	}
	if len(view.PartialData) > 0 {
		lines = append(lines, "", "## Partial Data", "")
		for _, message := range view.PartialData {
			lines = append(lines, "- "+message)
		}
	}
	lines = append(lines, collaboration.RenderCollaborationLines(view.Collaboration)...)
	lines = append(lines, "")
	return lines
}

func BuildCollaborationThread(surface, targetID string, comments []collaboration.Comment, decisions []collaboration.Decision) collaboration.Thread {
	return collaboration.BuildCollaborationThread(surface, targetID, comments, decisions)
}
