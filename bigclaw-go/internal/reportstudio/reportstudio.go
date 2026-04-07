package reportstudio

import (
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type ConsoleAction struct {
	ActionID string
	Label    string
	Target   string
	Enabled  bool
	Reason   string
}

func NewConsoleAction(actionID, label, target string, enabled bool, reason string) ConsoleAction {
	return ConsoleAction{
		ActionID: actionID,
		Label:    label,
		Target:   target,
		Enabled:  enabled,
		Reason:   reason,
	}
}

func (action ConsoleAction) State() string {
	if action.Enabled {
		return "enabled"
	}
	return "disabled"
}

type PilotMetric struct {
	Name           string
	Baseline       float64
	Current        float64
	Target         float64
	Unit           string
	HigherIsBetter bool
}

func (metric PilotMetric) Delta() float64 {
	return metric.Current - metric.Baseline
}

func (metric PilotMetric) MetTarget() bool {
	if metric.HigherIsBetter {
		return metric.Current >= metric.Target
	}
	return metric.Current <= metric.Target
}

type PilotScorecard struct {
	IssueID            string
	Customer           string
	Period             string
	Metrics            []PilotMetric
	MonthlyBenefit     float64
	MonthlyCost        float64
	ImplementationCost float64
	BenchmarkScore     *int
	BenchmarkPassed    *bool
}

func (scorecard PilotScorecard) MonthlyNetValue() float64 {
	return scorecard.MonthlyBenefit - scorecard.MonthlyCost
}

func (scorecard PilotScorecard) AnnualizedROI() float64 {
	totalCost := scorecard.ImplementationCost + (scorecard.MonthlyCost * 12)
	if totalCost <= 0 {
		return 0
	}
	annualGain := (scorecard.MonthlyBenefit * 12) - totalCost
	return (annualGain / totalCost) * 100
}

func (scorecard PilotScorecard) PaybackMonths() *float64 {
	if scorecard.MonthlyNetValue() <= 0 {
		return nil
	}
	if scorecard.ImplementationCost <= 0 {
		value := 0.0
		return &value
	}
	value := round1(scorecard.ImplementationCost / scorecard.MonthlyNetValue())
	return &value
}

func (scorecard PilotScorecard) MetricsMet() int {
	count := 0
	for _, metric := range scorecard.Metrics {
		if metric.MetTarget() {
			count++
		}
	}
	return count
}

func (scorecard PilotScorecard) Recommendation() string {
	benchmarkOK := scorecard.BenchmarkPassed == nil || *scorecard.BenchmarkPassed
	if len(scorecard.Metrics) > 0 && scorecard.MetricsMet() == len(scorecard.Metrics) && scorecard.AnnualizedROI() > 0 && benchmarkOK {
		return "go"
	}
	if scorecard.AnnualizedROI() > 0 || scorecard.MetricsMet() > 0 {
		return "iterate"
	}
	return "hold"
}

type PilotPortfolio struct {
	Name       string
	Period     string
	Scorecards []PilotScorecard
}

func (portfolio PilotPortfolio) TotalMonthlyNetValue() float64 {
	total := 0.0
	for _, scorecard := range portfolio.Scorecards {
		total += scorecard.MonthlyNetValue()
	}
	return total
}

func (portfolio PilotPortfolio) AverageROI() float64 {
	if len(portfolio.Scorecards) == 0 {
		return 0
	}
	total := 0.0
	for _, scorecard := range portfolio.Scorecards {
		total += scorecard.AnnualizedROI()
	}
	return round1(total / float64(len(portfolio.Scorecards)))
}

func (portfolio PilotPortfolio) RecommendationCounts() map[string]int {
	counts := map[string]int{"go": 0, "iterate": 0, "hold": 0}
	for _, scorecard := range portfolio.Scorecards {
		counts[scorecard.Recommendation()]++
	}
	return counts
}

func (portfolio PilotPortfolio) Recommendation() string {
	counts := portfolio.RecommendationCounts()
	if len(portfolio.Scorecards) > 0 && counts["go"] == len(portfolio.Scorecards) {
		return "scale"
	}
	if counts["go"] > 0 || counts["iterate"] > 0 {
		return "continue"
	}
	return "stop"
}

type IssueClosureDecision struct {
	IssueID    string
	Allowed    bool
	Reason     string
	ReportPath string
}

type DocumentationArtifact struct {
	Name string
	Path string
}

func (artifact DocumentationArtifact) Available() bool {
	return ValidationReportExists(artifact.Path)
}

type LaunchChecklistItem struct {
	Name     string
	Evidence []string
}

type LaunchChecklist struct {
	IssueID       string
	Documentation []DocumentationArtifact
	Items         []LaunchChecklistItem
}

func (checklist LaunchChecklist) DocumentationStatus() map[string]bool {
	out := make(map[string]bool, len(checklist.Documentation))
	for _, artifact := range checklist.Documentation {
		out[artifact.Name] = artifact.Available()
	}
	return out
}

func (checklist LaunchChecklist) ItemCompleted(item LaunchChecklistItem) bool {
	status := checklist.DocumentationStatus()
	if len(item.Evidence) == 0 {
		return true
	}
	for _, name := range item.Evidence {
		if !status[name] {
			return false
		}
	}
	return true
}

func (checklist LaunchChecklist) CompletedItems() int {
	count := 0
	for _, item := range checklist.Items {
		if checklist.ItemCompleted(item) {
			count++
		}
	}
	return count
}

func (checklist LaunchChecklist) MissingDocumentation() []string {
	out := make([]string, 0)
	for _, artifact := range checklist.Documentation {
		if !artifact.Available() {
			out = append(out, artifact.Name)
		}
	}
	return out
}

func (checklist LaunchChecklist) Ready() bool {
	if len(checklist.MissingDocumentation()) > 0 {
		return false
	}
	for _, item := range checklist.Items {
		if !checklist.ItemCompleted(item) {
			return false
		}
	}
	return true
}

type FinalDeliveryChecklist struct {
	IssueID                  string
	RequiredOutputs          []DocumentationArtifact
	RecommendedDocumentation []DocumentationArtifact
}

func (checklist FinalDeliveryChecklist) RequiredOutputStatus() map[string]bool {
	out := make(map[string]bool, len(checklist.RequiredOutputs))
	for _, artifact := range checklist.RequiredOutputs {
		out[artifact.Name] = artifact.Available()
	}
	return out
}

func (checklist FinalDeliveryChecklist) RecommendedDocumentationStatus() map[string]bool {
	out := make(map[string]bool, len(checklist.RecommendedDocumentation))
	for _, artifact := range checklist.RecommendedDocumentation {
		out[artifact.Name] = artifact.Available()
	}
	return out
}

func (checklist FinalDeliveryChecklist) GeneratedRequiredOutputs() int {
	count := 0
	for _, artifact := range checklist.RequiredOutputs {
		if artifact.Available() {
			count++
		}
	}
	return count
}

func (checklist FinalDeliveryChecklist) GeneratedRecommendedDocumentation() int {
	count := 0
	for _, artifact := range checklist.RecommendedDocumentation {
		if artifact.Available() {
			count++
		}
	}
	return count
}

func (checklist FinalDeliveryChecklist) MissingRequiredOutputs() []string {
	out := make([]string, 0)
	for _, artifact := range checklist.RequiredOutputs {
		if !artifact.Available() {
			out = append(out, artifact.Name)
		}
	}
	return out
}

func (checklist FinalDeliveryChecklist) MissingRecommendedDocumentation() []string {
	out := make([]string, 0)
	for _, artifact := range checklist.RecommendedDocumentation {
		if !artifact.Available() {
			out = append(out, artifact.Name)
		}
	}
	return out
}

func (checklist FinalDeliveryChecklist) Ready() bool {
	return len(checklist.MissingRequiredOutputs()) == 0
}

type NarrativeSection struct {
	Heading  string
	Body     string
	Evidence []string
	Callouts []string
}

func (section NarrativeSection) Ready() bool {
	return strings.TrimSpace(section.Heading) != "" && strings.TrimSpace(section.Body) != ""
}

type ReportStudio struct {
	Name          string
	IssueID       string
	Audience      string
	Period        string
	Summary       string
	Sections      []NarrativeSection
	ActionItems   []string
	SourceReports []string
}

func (studio ReportStudio) Ready() bool {
	if strings.TrimSpace(studio.Summary) == "" || len(studio.Sections) == 0 {
		return false
	}
	for _, section := range studio.Sections {
		if !section.Ready() {
			return false
		}
	}
	return true
}

func (studio ReportStudio) Recommendation() string {
	if studio.Ready() {
		return "publish"
	}
	return "draft"
}

func (studio ReportStudio) ExportSlug() string {
	slug := strings.ToLower(strings.TrimSpace(studio.Name))
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "report-studio"
	}
	return slug
}

type ReportStudioArtifacts struct {
	RootDir      string
	MarkdownPath string
	HTMLPath     string
	TextPath     string
}

func RenderIssueValidationReport(issueID, version, environment, summary string) string {
	return fmt.Sprintf("# Issue Validation Report\n\n- Issue ID: %s\n- 版本号: %s\n- 测试环境: %s\n- 生成时间: %s\n\n## 结论\n\n%s\n", issueID, version, environment, utcNowISO(), summary)
}

func RenderReportStudioReport(studio ReportStudio) string {
	lines := []string{
		"# Report Studio",
		"",
		fmt.Sprintf("- Name: %s", studio.Name),
		fmt.Sprintf("- Issue ID: %s", studio.IssueID),
		fmt.Sprintf("- Audience: %s", studio.Audience),
		fmt.Sprintf("- Period: %s", studio.Period),
		fmt.Sprintf("- Sections: %d", len(studio.Sections)),
		fmt.Sprintf("- Recommendation: %s", studio.Recommendation()),
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
				fmt.Sprintf("### %s", section.Heading),
				"",
				firstNonEmpty(section.Body, "No narrative drafted."),
				"",
				fmt.Sprintf("- Evidence: %s", joinOrNone(section.Evidence)),
				fmt.Sprintf("- Callouts: %s", joinOrNone(section.Callouts)),
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
		for _, source := range studio.SourceReports {
			lines = append(lines, "- "+source)
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderReportStudioPlainText(studio ReportStudio) string {
	lines := []string{
		fmt.Sprintf("%s (%s)", studio.Name, studio.IssueID),
		fmt.Sprintf("Audience: %s", studio.Audience),
		fmt.Sprintf("Period: %s", studio.Period),
		fmt.Sprintf("Recommendation: %s", studio.Recommendation()),
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
	sectionHTML := ""
	for _, section := range studio.Sections {
		sectionHTML += fmt.Sprintf(`<section class="section"><h2>%s</h2><p>%s</p><p class="meta">Evidence: %s</p><p class="meta">Callouts: %s</p></section>`,
			html.EscapeString(section.Heading),
			html.EscapeString(section.Body),
			html.EscapeString(joinOrNone(section.Evidence)),
			html.EscapeString(joinOrNone(section.Callouts)),
		)
	}
	if sectionHTML == "" {
		sectionHTML = `<section class="section"><p>No sections drafted.</p></section>`
	}
	actionHTML := listHTML(studio.ActionItems)
	sourceHTML := listHTML(studio.SourceReports)
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>%s</title>
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
`,
		html.EscapeString(studio.Name),
		html.EscapeString(studio.IssueID),
		html.EscapeString(studio.Audience),
		html.EscapeString(studio.Period),
		html.EscapeString(studio.Name),
		html.EscapeString(studio.Recommendation()),
		html.EscapeString(firstNonEmpty(studio.Summary, "No summary drafted.")),
		sectionHTML,
		actionHTML,
		sourceHTML,
	)
}

func BuildLaunchChecklist(issueID string, documentation []DocumentationArtifact, items []LaunchChecklistItem) LaunchChecklist {
	return LaunchChecklist{IssueID: issueID, Documentation: documentation, Items: items}
}

func BuildFinalDeliveryChecklist(issueID string, requiredOutputs, recommendedDocumentation []DocumentationArtifact) FinalDeliveryChecklist {
	return FinalDeliveryChecklist{
		IssueID:                  issueID,
		RequiredOutputs:          requiredOutputs,
		RecommendedDocumentation: recommendedDocumentation,
	}
}

func RenderLaunchChecklistReport(checklist LaunchChecklist) string {
	lines := []string{
		"# Launch Checklist",
		"",
		fmt.Sprintf("- Issue ID: %s", checklist.IssueID),
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
			lines = append(lines, fmt.Sprintf("- %s: completed=%t evidence=%s", item.Name, checklist.ItemCompleted(item), joinOrNone(item.Evidence)))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderFinalDeliveryChecklistReport(checklist FinalDeliveryChecklist) string {
	lines := []string{
		"# Final Delivery Checklist",
		"",
		fmt.Sprintf("- Issue ID: %s", checklist.IssueID),
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
		fmt.Sprintf("- Issue ID: %s", scorecard.IssueID),
		fmt.Sprintf("- Customer: %s", scorecard.Customer),
		fmt.Sprintf("- Period: %s", scorecard.Period),
		fmt.Sprintf("- Recommendation: %s", scorecard.Recommendation()),
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
			lines = append(lines, fmt.Sprintf("- %s: baseline=%v%s current=%v%s target%s%v%s delta=%+.2f%s met=%t", metric.Name, trimFloat(metric.Baseline), unitSuffix, trimFloat(metric.Current), unitSuffix, comparator, trimFloat(metric.Target), unitSuffix, metric.Delta(), unitSuffix, metric.MetTarget()))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderPilotPortfolioReport(portfolio PilotPortfolio) string {
	counts := portfolio.RecommendationCounts()
	lines := []string{
		"# Pilot Portfolio Report",
		"",
		fmt.Sprintf("- Portfolio: %s", portfolio.Name),
		fmt.Sprintf("- Period: %s", portfolio.Period),
		fmt.Sprintf("- Scorecards: %d", len(portfolio.Scorecards)),
		fmt.Sprintf("- Recommendation: %s", portfolio.Recommendation()),
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

func WriteReport(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func WriteReportStudioBundle(rootDir string, studio ReportStudio) (ReportStudioArtifacts, error) {
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return ReportStudioArtifacts{}, err
	}
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
	return ReportStudioArtifacts{
		RootDir:      rootDir,
		MarkdownPath: markdownPath,
		HTMLPath:     htmlPath,
		TextPath:     textPath,
	}, nil
}

func ValidationReportExists(reportPath string) bool {
	if strings.TrimSpace(reportPath) == "" {
		return false
	}
	body, err := os.ReadFile(reportPath)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(body)) != ""
}

func EvaluateIssueClosure(issueID, reportPath string, validationPassed bool, launchChecklist *LaunchChecklist, finalDeliveryChecklist *FinalDeliveryChecklist) IssueClosureDecision {
	if !ValidationReportExists(reportPath) {
		return IssueClosureDecision{IssueID: issueID, Allowed: false, Reason: "validation report required before closing issue", ReportPath: reportPath}
	}
	if !validationPassed {
		return IssueClosureDecision{IssueID: issueID, Allowed: false, Reason: "validation failed; issue must remain open", ReportPath: reportPath}
	}
	if finalDeliveryChecklist != nil && !finalDeliveryChecklist.Ready() {
		return IssueClosureDecision{IssueID: issueID, Allowed: false, Reason: "final delivery checklist incomplete; required outputs missing", ReportPath: reportPath}
	}
	if launchChecklist != nil && !launchChecklist.Ready() {
		return IssueClosureDecision{IssueID: issueID, Allowed: false, Reason: "launch checklist incomplete; linked documentation missing or empty", ReportPath: reportPath}
	}
	if finalDeliveryChecklist != nil {
		return IssueClosureDecision{IssueID: issueID, Allowed: true, Reason: "validation report and final delivery checklist requirements satisfied; issue can be closed", ReportPath: reportPath}
	}
	return IssueClosureDecision{IssueID: issueID, Allowed: true, Reason: "validation report and launch checklist requirements satisfied; issue can be closed", ReportPath: reportPath}
}

func utcNowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "None"
	}
	return strings.Join(values, ", ")
}

func listHTML(values []string) string {
	if len(values) == 0 {
		return "<li>None</li>"
	}
	items := make([]string, 0, len(values))
	for _, value := range values {
		items = append(items, "<li>"+html.EscapeString(value)+"</li>")
	}
	return strings.Join(items, "")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func round1(value float64) float64 {
	return math.Round(value*10) / 10
}

func trimFloat(value float64) string {
	if value == math.Trunc(value) {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("%.1f", value)
}
