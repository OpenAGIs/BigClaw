package reporting

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

type PilotMetric struct {
	Name           string
	Baseline       float64
	Current        float64
	Target         float64
	Unit           string
	HigherIsBetter bool
}

func (m PilotMetric) Delta() float64 {
	return m.Current - m.Baseline
}

func (m PilotMetric) MetTarget() bool {
	if m.HigherIsBetter || m.Unit == "" && !m.HigherIsBetter {
		if m.HigherIsBetter {
			return m.Current >= m.Target
		}
	}
	return m.Current <= m.Target
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
		zero := 0.0
		return &zero
	}
	value := roundTenth(s.ImplementationCost / s.MonthlyNetValue())
	return &value
}

func (s PilotScorecard) MetricsMet() int {
	count := 0
	for _, metric := range s.Metrics {
		if metric.MetTarget() {
			count++
		}
	}
	return count
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
	Name       string
	Period     string
	Scorecards []PilotScorecard
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

func (a DocumentationArtifact) Available() bool {
	return ValidationReportExists(a.Path)
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

func (c LaunchChecklist) DocumentationStatus() map[string]bool {
	status := make(map[string]bool, len(c.Documentation))
	for _, artifact := range c.Documentation {
		status[artifact.Name] = artifact.Available()
	}
	return status
}

func (c LaunchChecklist) ItemCompleted(item LaunchChecklistItem) bool {
	if len(item.Evidence) == 0 {
		return true
	}
	status := c.DocumentationStatus()
	for _, name := range item.Evidence {
		if !status[name] {
			return false
		}
	}
	return true
}

func (c LaunchChecklist) CompletedItems() int {
	count := 0
	for _, item := range c.Items {
		if c.ItemCompleted(item) {
			count++
		}
	}
	return count
}

func (c LaunchChecklist) MissingDocumentation() []string {
	var missing []string
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
	IssueID                  string
	RequiredOutputs          []DocumentationArtifact
	RecommendedDocumentation []DocumentationArtifact
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
	count := 0
	for _, artifact := range c.RequiredOutputs {
		if artifact.Available() {
			count++
		}
	}
	return count
}

func (c FinalDeliveryChecklist) GeneratedRecommendedDocumentation() int {
	count := 0
	for _, artifact := range c.RecommendedDocumentation {
		if artifact.Available() {
			count++
		}
	}
	return count
}

func (c FinalDeliveryChecklist) MissingRequiredOutputs() []string {
	var missing []string
	for _, artifact := range c.RequiredOutputs {
		if !artifact.Available() {
			missing = append(missing, artifact.Name)
		}
	}
	return missing
}

func (c FinalDeliveryChecklist) MissingRecommendedDocumentation() []string {
	var missing []string
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

type NarrativeSection struct {
	Heading  string
	Body     string
	Evidence []string
	Callouts []string
}

func (s NarrativeSection) Ready() bool {
	return strings.TrimSpace(s.Heading) != "" && strings.TrimSpace(s.Body) != ""
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
	slug := slugify(s.Name)
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

type TriageFeedbackRecord struct {
	RunID     string
	Action    string
	Decision  string
	Actor     string
	Notes     string
	Timestamp string
}

func NewTriageFeedbackRecord(runID, action, decision, actor, notes string) TriageFeedbackRecord {
	return TriageFeedbackRecord{
		RunID:     runID,
		Action:    action,
		Decision:  decision,
		Actor:     actor,
		Notes:     notes,
		Timestamp: utcNowISO(),
	}
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
				"### "+section.Heading,
				"",
				firstNonEmpty(section.Body, "No narrative drafted."),
				"",
				"- Evidence: "+firstNonEmpty(strings.Join(section.Evidence, ", "), "None"),
				"- Callouts: "+firstNonEmpty(strings.Join(section.Callouts, ", "), "None"),
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
		for _, path := range studio.SourceReports {
			lines = append(lines, "- "+path)
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
	var sectionHTML strings.Builder
	for _, section := range studio.Sections {
		sectionHTML.WriteString(`
        <section class="section">
          <h2>` + html.EscapeString(section.Heading) + `</h2>
          <p>` + html.EscapeString(section.Body) + `</p>
          <p class="meta">Evidence: ` + html.EscapeString(firstNonEmpty(strings.Join(section.Evidence, ", "), "None")) + `</p>
          <p class="meta">Callouts: ` + html.EscapeString(firstNonEmpty(strings.Join(section.Callouts, ", "), "None")) + `</p>
        </section>
        `)
	}
	actionHTML := "<li>None</li>"
	if len(studio.ActionItems) > 0 {
		var items strings.Builder
		for _, item := range studio.ActionItems {
			items.WriteString("<li>" + html.EscapeString(item) + "</li>")
		}
		actionHTML = items.String()
	}
	sourceHTML := "<li>None</li>"
	if len(studio.SourceReports) > 0 {
		var items strings.Builder
		for _, path := range studio.SourceReports {
			items.WriteString("<li>" + html.EscapeString(path) + "</li>")
		}
		sourceHTML = items.String()
	}
	sections := sectionHTML.String()
	if sections == "" {
		sections = `<section class="section"><p>No sections drafted.</p></section>`
	}
	return "<!DOCTYPE html>\n<html lang=\"en\">\n  <head>\n    <meta charset=\"utf-8\" />\n    <title>" + html.EscapeString(studio.Name) + "</title>\n    <style>\n      body { font-family: Georgia, 'Times New Roman', serif; margin: 40px auto; max-width: 840px; color: #1f2933; line-height: 1.6; }\n      h1, h2 { font-family: 'Avenir Next', 'Segoe UI', sans-serif; }\n      .meta { color: #52606d; font-size: 0.95rem; }\n      .summary { padding: 16px 20px; background: #f7f3e8; border-left: 4px solid #c58b32; }\n      .section { margin-top: 28px; }\n    </style>\n  </head>\n  <body>\n    <header>\n      <p class=\"meta\">" + html.EscapeString(studio.IssueID) + " · " + html.EscapeString(studio.Audience) + " · " + html.EscapeString(studio.Period) + "</p>\n      <h1>" + html.EscapeString(studio.Name) + "</h1>\n      <p class=\"meta\">Recommendation: " + html.EscapeString(studio.Recommendation()) + "</p>\n    </header>\n    <section class=\"summary\">\n      <h2>Narrative Summary</h2>\n      <p>" + html.EscapeString(firstNonEmpty(studio.Summary, "No summary drafted.")) + "</p>\n    </section>\n    " + sections + "\n    <section class=\"section\">\n      <h2>Action Items</h2>\n      <ul>" + actionHTML + "</ul>\n    </section>\n    <section class=\"section\">\n      <h2>Sources</h2>\n      <ul>" + sourceHTML + "</ul>\n    </section>\n  </body>\n</html>\n"
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
		fmt.Sprintf("- Ready: %s", pyBool(checklist.Ready())),
		"",
		"## Documentation",
		"",
	}
	if len(checklist.Documentation) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, artifact := range checklist.Documentation {
			lines = append(lines, fmt.Sprintf("- %s: available=%s path=%s", artifact.Name, pyBool(artifact.Available()), artifact.Path))
		}
	}
	lines = append(lines, "", "## Checklist", "")
	if len(checklist.Items) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, item := range checklist.Items {
			lines = append(lines, fmt.Sprintf("- %s: completed=%s evidence=%s", item.Name, pyBool(checklist.ItemCompleted(item)), firstNonEmpty(strings.Join(item.Evidence, ", "), "none")))
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
		fmt.Sprintf("- Ready: %s", pyBool(checklist.Ready())),
		"",
		"## Required Outputs",
		"",
	}
	if len(checklist.RequiredOutputs) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, artifact := range checklist.RequiredOutputs {
			lines = append(lines, fmt.Sprintf("- %s: available=%s path=%s", artifact.Name, pyBool(artifact.Available()), artifact.Path))
		}
	}
	lines = append(lines, "", "## Recommended Documentation", "")
	if len(checklist.RecommendedDocumentation) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, artifact := range checklist.RecommendedDocumentation {
			lines = append(lines, fmt.Sprintf("- %s: available=%s path=%s", artifact.Name, pyBool(artifact.Available()), artifact.Path))
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
	if scorecard.PaybackMonths() == nil {
		lines = append(lines, "- Payback Months: n/a")
	} else {
		lines = append(lines, fmt.Sprintf("- Payback Months: %.1f", *scorecard.PaybackMonths()))
	}
	if scorecard.BenchmarkScore != nil {
		lines = append(lines, fmt.Sprintf("- Benchmark Score: %d", *scorecard.BenchmarkScore))
	}
	if scorecard.BenchmarkPassed != nil {
		lines = append(lines, fmt.Sprintf("- Benchmark Passed: %s", pyBool(*scorecard.BenchmarkPassed)))
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
			lines = append(lines, fmt.Sprintf("- %s: baseline=%v%s current=%v%s target%s%v%s delta=%+.2f%s met=%s", metric.Name, metric.Baseline, unitSuffix, metric.Current, unitSuffix, comparator, metric.Target, unitSuffix, metric.Delta(), unitSuffix, pyBool(metric.MetTarget())))
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

func WriteReportStudioBundle(rootDir string, studio ReportStudio) (ReportStudioArtifacts, error) {
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return ReportStudioArtifacts{}, err
	}
	artifacts := ReportStudioArtifacts{
		RootDir:      rootDir,
		MarkdownPath: filepath.Join(rootDir, studio.ExportSlug()+".md"),
		HTMLPath:     filepath.Join(rootDir, studio.ExportSlug()+".html"),
		TextPath:     filepath.Join(rootDir, studio.ExportSlug()+".txt"),
	}
	if err := WriteReport(artifacts.MarkdownPath, RenderReportStudioReport(studio)); err != nil {
		return ReportStudioArtifacts{}, err
	}
	if err := WriteReport(artifacts.HTMLPath, RenderReportStudioHTML(studio)); err != nil {
		return ReportStudioArtifacts{}, err
	}
	if err := WriteReport(artifacts.TextPath, RenderReportStudioPlainText(studio)); err != nil {
		return ReportStudioArtifacts{}, err
	}
	return artifacts, nil
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
	resolvedPath := reportPath
	if reportPath != "" {
		resolvedPath = filepath.Clean(reportPath)
	}
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

func utcNowISO() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func slugify(value string) string {
	var builder strings.Builder
	lastHyphen := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen {
			builder.WriteByte('-')
			lastHyphen = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func pyBool(value bool) string {
	if value {
		return "True"
	}
	return "False"
}
