package reportingparity

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ConsoleAction struct {
	Key     string
	Label   string
	Target  string
	Enabled bool
	Reason  string
}

func NewConsoleAction(key string, label string, target string) ConsoleAction {
	return ConsoleAction{Key: key, Label: label, Target: target, Enabled: true}
}

func (a ConsoleAction) State() string {
	if a.Enabled {
		return "enabled"
	}
	return "disabled"
}

type NarrativeSection struct {
	Heading  string
	Body     string
	Evidence []string
	Callouts []string
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

func RenderReportStudioReport(s ReportStudio) string {
	lines := []string{
		"# Report Studio",
		"",
		fmt.Sprintf("- Name: %s", s.Name),
		fmt.Sprintf("- Issue: %s", s.IssueID),
		fmt.Sprintf("- Audience: %s", s.Audience),
		fmt.Sprintf("- Period: %s", s.Period),
		fmt.Sprintf("- Recommendation: %s", s.Recommendation()),
		"",
		"## Summary",
		"",
		s.Summary,
		"",
	}
	for _, section := range s.Sections {
		lines = append(lines, "### "+section.Heading, "", section.Body, "")
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderReportStudioPlainText(s ReportStudio) string {
	lines := []string{
		"Report Studio",
		fmt.Sprintf("Recommendation: %s", s.Recommendation()),
		fmt.Sprintf("Summary: %s", s.Summary),
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderReportStudioHTML(s ReportStudio) string {
	var b strings.Builder
	b.WriteString("<html><body>\n")
	b.WriteString(fmt.Sprintf("<h1>%s</h1>\n", escapeHTML(s.Name)))
	b.WriteString(fmt.Sprintf("<p>Recommendation: %s</p>\n", escapeHTML(s.Recommendation())))
	for _, section := range s.Sections {
		b.WriteString(fmt.Sprintf("<h2>%s</h2>\n<p>%s</p>\n", escapeHTML(section.Heading), escapeHTML(section.Body)))
	}
	b.WriteString("</body></html>\n")
	return b.String()
}

type ReportStudioArtifacts struct {
	MarkdownPath string
	HTMLPath     string
	TextPath     string
}

func WriteReportStudioBundle(root string, studio ReportStudio) (ReportStudioArtifacts, error) {
	slug := slugify(studio.Name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return ReportStudioArtifacts{}, err
	}
	artifacts := ReportStudioArtifacts{
		MarkdownPath: filepath.Join(root, slug+".md"),
		HTMLPath:     filepath.Join(root, slug+".html"),
		TextPath:     filepath.Join(root, slug+".txt"),
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

type PilotMetric struct {
	Name           string
	Baseline       float64
	Current        float64
	Target         float64
	Unit           string
	HigherIsBetter bool
}

func (m PilotMetric) Met() bool {
	if !m.HigherIsBetter {
		return m.Current <= m.Target
	}
	return m.Current >= m.Target
}

type PilotScorecard struct {
	IssueID            string
	Customer           string
	Period             string
	Metrics            []PilotMetric
	MonthlyBenefit     float64
	MonthlyCost        float64
	ImplementationCost float64
	BenchmarkScore     int
	BenchmarkPassed    bool
}

func (s PilotScorecard) MetricsMet() int {
	total := 0
	for _, metric := range s.Metrics {
		if metric.Met() {
			total++
		}
	}
	return total
}

func (s PilotScorecard) MonthlyNetValue() float64 {
	return s.MonthlyBenefit - s.MonthlyCost
}

func (s PilotScorecard) PaybackMonths() *float64 {
	net := s.MonthlyNetValue()
	if net <= 0 {
		return nil
	}
	value := round1(s.ImplementationCost / net)
	return &value
}

func (s PilotScorecard) AnnualizedROI() float64 {
	totalCost := s.ImplementationCost + (s.MonthlyCost * 12)
	if totalCost <= 0 {
		return 0
	}
	annualGain := (s.MonthlyBenefit * 12) - totalCost
	return round1((annualGain / totalCost) * 100)
}

func (s PilotScorecard) Recommendation() string {
	benchmarkOK := s.BenchmarkPassed || s.BenchmarkScore == 0
	if len(s.Metrics) > 0 && s.MetricsMet() == len(s.Metrics) && s.AnnualizedROI() > 0 && benchmarkOK {
		return "go"
	}
	if s.AnnualizedROI() > 0 || s.MetricsMet() > 0 {
		return "iterate"
	}
	return "hold"
}

func RenderPilotScorecard(s PilotScorecard) string {
	lines := []string{
		"# Pilot Scorecard",
		"",
		fmt.Sprintf("- Customer: %s", s.Customer),
		fmt.Sprintf("- Recommendation: %s", s.Recommendation()),
		fmt.Sprintf("- Annualized ROI: %.1f%%", s.AnnualizedROI()),
		fmt.Sprintf("- Benchmark Score: %d", s.BenchmarkScore),
		"",
	}
	for _, metric := range s.Metrics {
		lines = append(lines, fmt.Sprintf("- %s: baseline=%v current=%v target=%v", metric.Name, trimFloat(metric.Baseline), trimFloat(metric.Current), trimFloat(metric.Target)))
	}
	return strings.Join(lines, "\n") + "\n"
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
	return round1(total / float64(len(p.Scorecards)))
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

func RenderPilotPortfolioReport(p PilotPortfolio) string {
	counts := p.RecommendationCounts()
	lines := []string{
		"# Pilot Portfolio Report",
		"",
		fmt.Sprintf("- Recommendation Mix: go=%d iterate=%d hold=%d", counts["go"], counts["iterate"], counts["hold"]),
	}
	for _, scorecard := range p.Scorecards {
		lines = append(lines, fmt.Sprintf("- %s: recommendation=%s", scorecard.Customer, scorecard.Recommendation()))
	}
	return strings.Join(lines, "\n") + "\n"
}

type DocumentationArtifact struct {
	Name string
	Path string
}

type LaunchChecklistItem struct {
	Name     string
	Evidence []string
}

type LaunchChecklist struct {
	IssueID              string
	DocumentationStatus  map[string]bool
	Items                []LaunchChecklistStatus
	CompletedItems       int
	MissingDocumentation []string
	Ready                bool
}

type LaunchChecklistStatus struct {
	Name      string
	Completed bool
	Evidence  []string
}

func BuildLaunchChecklist(issueID string, documentation []DocumentationArtifact, items []LaunchChecklistItem) LaunchChecklist {
	status := make(map[string]bool, len(documentation))
	missing := make([]string, 0)
	for _, doc := range documentation {
		ok := validationReportExists(doc.Path)
		status[doc.Name] = ok
		if !ok {
			missing = append(missing, doc.Name)
		}
	}
	sort.Strings(missing)
	checklist := LaunchChecklist{IssueID: issueID, DocumentationStatus: status, MissingDocumentation: missing, Ready: true}
	for _, item := range items {
		completed := true
		for _, evidence := range item.Evidence {
			if !status[evidence] {
				completed = false
				break
			}
		}
		if completed {
			checklist.CompletedItems++
		} else {
			checklist.Ready = false
		}
		checklist.Items = append(checklist.Items, LaunchChecklistStatus{Name: item.Name, Completed: completed, Evidence: item.Evidence})
	}
	if len(missing) > 0 {
		checklist.Ready = false
	}
	return checklist
}

func RenderLaunchChecklistReport(c LaunchChecklist) string {
	lines := []string{
		"# Launch Checklist",
		"",
	}
	names := make([]string, 0, len(c.DocumentationStatus))
	for name := range c.DocumentationStatus {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		lines = append(lines, fmt.Sprintf("- %s: available=%t", name, c.DocumentationStatus[name]))
	}
	for _, item := range c.Items {
		lines = append(lines, fmt.Sprintf("- %s: completed=%t evidence=%s", item.Name, item.Completed, strings.Join(item.Evidence, ",")))
	}
	return strings.Join(lines, "\n") + "\n"
}

type FinalDeliveryChecklist struct {
	IssueID                        string
	RequiredOutputStatus           map[string]bool
	RecommendedDocumentationStatus map[string]bool
	GeneratedRequiredOutputs       int
	GeneratedRecommendedDocs       int
	MissingRequiredOutputs         []string
	MissingRecommendedDocs         []string
	Ready                          bool
}

func BuildFinalDeliveryChecklist(issueID string, requiredOutputs []DocumentationArtifact, recommendedDocs []DocumentationArtifact) FinalDeliveryChecklist {
	requiredStatus := make(map[string]bool, len(requiredOutputs))
	recommendedStatus := make(map[string]bool, len(recommendedDocs))
	requiredMissing := make([]string, 0)
	recommendedMissing := make([]string, 0)
	generatedRequired := 0
	generatedRecommended := 0
	for _, artifact := range requiredOutputs {
		ok := validationReportExists(artifact.Path)
		requiredStatus[artifact.Name] = ok
		if ok {
			generatedRequired++
		} else {
			requiredMissing = append(requiredMissing, artifact.Name)
		}
	}
	for _, artifact := range recommendedDocs {
		ok := validationReportExists(artifact.Path)
		recommendedStatus[artifact.Name] = ok
		if ok {
			generatedRecommended++
		} else {
			recommendedMissing = append(recommendedMissing, artifact.Name)
		}
	}
	sort.Strings(requiredMissing)
	sort.Strings(recommendedMissing)
	return FinalDeliveryChecklist{
		IssueID:                        issueID,
		RequiredOutputStatus:           requiredStatus,
		RecommendedDocumentationStatus: recommendedStatus,
		GeneratedRequiredOutputs:       generatedRequired,
		GeneratedRecommendedDocs:       generatedRecommended,
		MissingRequiredOutputs:         requiredMissing,
		MissingRecommendedDocs:         recommendedMissing,
		Ready:                          len(requiredMissing) == 0,
	}
}

func RenderFinalDeliveryChecklistReport(c FinalDeliveryChecklist) string {
	lines := []string{
		"# Final Delivery Checklist",
		"",
		fmt.Sprintf("- Required Outputs Generated: %d/%d", c.GeneratedRequiredOutputs, len(c.RequiredOutputStatus)),
		fmt.Sprintf("- Recommended Docs Generated: %d/%d", c.GeneratedRecommendedDocs, len(c.RecommendedDocumentationStatus)),
	}
	requiredNames := make([]string, 0, len(c.RequiredOutputStatus))
	for name := range c.RequiredOutputStatus {
		requiredNames = append(requiredNames, name)
	}
	sort.Strings(requiredNames)
	for _, name := range requiredNames {
		lines = append(lines, fmt.Sprintf("- %s: available=%t", name, c.RequiredOutputStatus[name]))
	}
	recommendedNames := make([]string, 0, len(c.RecommendedDocumentationStatus))
	for name := range c.RecommendedDocumentationStatus {
		recommendedNames = append(recommendedNames, name)
	}
	sort.Strings(recommendedNames)
	for _, name := range recommendedNames {
		lines = append(lines, fmt.Sprintf("- %s: available=%t", name, c.RecommendedDocumentationStatus[name]))
	}
	return strings.Join(lines, "\n") + "\n"
}

type IssueClosureDecision struct {
	Allowed    bool
	Reason     string
	ReportPath string
}

func EvaluateIssueClosure(issueID string, reportPath string, validationPassed *bool, launchChecklist *LaunchChecklist, finalChecklist *FinalDeliveryChecklist) IssueClosureDecision {
	if !ValidationReportExists(reportPath) {
		return IssueClosureDecision{Allowed: false, Reason: "validation report required before closing issue"}
	}
	if validationPassed != nil && !*validationPassed {
		return IssueClosureDecision{Allowed: false, Reason: "validation failed; issue must remain open"}
	}
	if launchChecklist != nil && !launchChecklist.Ready {
		return IssueClosureDecision{Allowed: false, Reason: "launch checklist incomplete; linked documentation missing or empty"}
	}
	if finalChecklist != nil && !finalChecklist.Ready {
		return IssueClosureDecision{Allowed: false, Reason: "final delivery checklist incomplete; required outputs missing"}
	}
	if finalChecklist != nil {
		return IssueClosureDecision{Allowed: true, Reason: "validation report and final delivery checklist requirements satisfied; issue can be closed", ReportPath: reportPath}
	}
	return IssueClosureDecision{Allowed: true, Reason: "validation report and launch checklist requirements satisfied; issue can be closed", ReportPath: reportPath}
}

type CollaborationComment struct {
	CommentID string
	Author    string
	Body      string
	Mentions  []string
	Anchor    string
}

type DecisionNote struct {
	DecisionID string
	Author     string
	Outcome    string
	Summary    string
	Mentions   []string
	FollowUp   string
}

type CollaborationThread struct {
	Surface   string
	SubjectID string
	Comments  []CollaborationComment
	Decisions []DecisionNote
}

func BuildCollaborationThread(surface string, subjectID string, comments []CollaborationComment, decisions []DecisionNote) CollaborationThread {
	return CollaborationThread{Surface: surface, SubjectID: subjectID, Comments: comments, Decisions: decisions}
}

type SharedViewFilter struct {
	Label string
	Value string
}

type SharedViewContext struct {
	Filters       []SharedViewFilter
	ResultCount   int
	Loading       bool
	Errors        []string
	PartialData   []string
	LastUpdated   string
	Collaboration *CollaborationThread
}

func RenderSharedViewContext(view SharedViewContext) []string {
	lines := []string{
		"## Filters",
	}
	for _, filter := range view.Filters {
		lines = append(lines, fmt.Sprintf("- %s: %s", filter.Label, filter.Value))
	}
	if view.Collaboration != nil {
		lines = append(lines, "## Collaboration")
		lines = append(lines, fmt.Sprintf("Surface: %s", view.Collaboration.Surface))
		for _, comment := range view.Collaboration.Comments {
			lines = append(lines, comment.Body)
		}
		for _, decision := range view.Collaboration.Decisions {
			lines = append(lines, decision.Summary)
		}
	}
	return lines
}

func RenderIssueValidationReport(issueID string, version string, environment string, status string) string {
	return fmt.Sprintf("# Validation Report\n\n- Issue: %s\n- Version: %s\n- Environment: %s\n- Status: %s\n", issueID, version, environment, status)
}

func WriteReport(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func ValidationReportExists(path string) bool {
	return validationReportExists(path)
}

func validationReportExists(path string) bool {
	data, err := os.ReadFile(path)
	return err == nil && strings.TrimSpace(string(data)) != ""
}

func slugify(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	replacer := strings.NewReplacer(" ", "-", "/", "-", "_", "-", ".", "-", "&", "-", "--", "-")
	input = replacer.Replace(input)
	input = strings.Trim(input, "-")
	for strings.Contains(input, "--") {
		input = strings.ReplaceAll(input, "--", "-")
	}
	if input == "" {
		return "report-studio"
	}
	return input
}

func escapeHTML(input string) string {
	replacer := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;")
	return replacer.Replace(input)
}

func round1(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}

func trimFloat(value float64) string {
	if float64(int(value)) == value {
		return fmt.Sprintf("%d", int(value))
	}
	return fmt.Sprintf("%.1f", value)
}
