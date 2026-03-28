package reporting

import (
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/risk"
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
	net := s.MonthlyNetValue()
	if net <= 0 {
		return nil
	}
	if s.ImplementationCost <= 0 {
		zero := 0.0
		return &zero
	}
	value := math.Round((s.ImplementationCost/net)*10) / 10
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
	slug := strings.ToLower(strings.TrimSpace(s.Name))
	replacer := strings.NewReplacer(" ", "-", "_", "-", "/", "-", ".", "-", ":", "-", "&", "-")
	slug = replacer.Replace(slug)
	parts := strings.FieldsFunc(slug, func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-'
	})
	slug = strings.Join(parts, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "report-studio"
	}
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	return slug
}

type ReportStudioArtifacts struct {
	RootDir      string
	MarkdownPath string
	HTMLPath     string
	TextPath     string
}

type ValidationReportDecision struct {
	AllowedToClose bool
	Status         string
	Summary        string
	MissingReports []string
}

var RequiredReportArtifacts = []string{"task-run", "replay", "benchmark-suite"}

func EnforceValidationReportPolicy(artifacts []string) ValidationReportDecision {
	seen := make(map[string]struct{}, len(artifacts))
	for _, item := range artifacts {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		seen[item] = struct{}{}
	}
	missing := make([]string, 0)
	for _, required := range RequiredReportArtifacts {
		if _, ok := seen[required]; !ok {
			missing = append(missing, required)
		}
	}
	if len(missing) > 0 {
		return ValidationReportDecision{
			AllowedToClose: false,
			Status:         "blocked",
			Summary:        "validation report policy not satisfied",
			MissingReports: missing,
		}
	}
	return ValidationReportDecision{
		AllowedToClose: true,
		Status:         "ready",
		Summary:        "validation report policy satisfied",
	}
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
	out := make(map[string]bool, len(c.Documentation))
	for _, artifact := range c.Documentation {
		out[artifact.Name] = artifact.Available()
	}
	return out
}

func (c LaunchChecklist) ItemCompleted(item LaunchChecklistItem) bool {
	status := c.DocumentationStatus()
	for _, evidence := range item.Evidence {
		if !status[evidence] {
			return false
		}
	}
	return true
}

func (c LaunchChecklist) Ready() bool {
	for _, artifact := range c.Documentation {
		if !artifact.Available() {
			return false
		}
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

func (c FinalDeliveryChecklist) Ready() bool {
	for _, artifact := range c.RequiredOutputs {
		if !artifact.Available() {
			return false
		}
	}
	return true
}

type IssueClosureDecision struct {
	IssueID    string
	Allowed    bool
	Reason     string
	ReportPath string
}

func RenderIssueValidationReport(issueID string, version string, environment string, summary string) string {
	return fmt.Sprintf("# Issue Validation Report\n\n- Issue ID: %s\n- 版本号: %s\n- 测试环境: %s\n- 生成时间: %s\n\n## 结论\n\n%s\n", issueID, version, environment, time.Now().UTC().Format(time.RFC3339), summary)
}

func WriteReportStudioBundle(rootDir string, studio ReportStudio) (ReportStudioArtifacts, error) {
	if err := WriteReport(filepath.Join(rootDir, studio.ExportSlug()+".md"), RenderReportStudioReport(studio)); err != nil {
		return ReportStudioArtifacts{}, err
	}
	if err := WriteReport(filepath.Join(rootDir, studio.ExportSlug()+".html"), RenderReportStudioHTML(studio)); err != nil {
		return ReportStudioArtifacts{}, err
	}
	if err := WriteReport(filepath.Join(rootDir, studio.ExportSlug()+".txt"), RenderReportStudioPlainText(studio)); err != nil {
		return ReportStudioArtifacts{}, err
	}
	return ReportStudioArtifacts{
		RootDir:      rootDir,
		MarkdownPath: filepath.Join(rootDir, studio.ExportSlug()+".md"),
		HTMLPath:     filepath.Join(rootDir, studio.ExportSlug()+".html"),
		TextPath:     filepath.Join(rootDir, studio.ExportSlug()+".txt"),
	}, nil
}

func ValidationReportExists(reportPath string) bool {
	reportPath = strings.TrimSpace(reportPath)
	if reportPath == "" {
		return false
	}
	body, err := os.ReadFile(reportPath)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(body)) != ""
}

func EvaluateIssueClosure(issueID string, reportPath string, validationPassed bool, launchChecklist *LaunchChecklist, finalDeliveryChecklist *FinalDeliveryChecklist) IssueClosureDecision {
	resolvedPath := strings.TrimSpace(reportPath)
	if resolvedPath == "" {
		resolvedPath = reportPath
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
		for _, item := range studio.SourceReports {
			lines = append(lines, "- "+item)
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
	return strings.Join(lines, "\n")
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
`, html.EscapeString(section.Heading), html.EscapeString(firstNonEmpty(section.Body, "No narrative drafted.")), html.EscapeString(joinOrNone(section.Evidence)), html.EscapeString(joinOrNone(section.Callouts))))
	}
	actionItems := "<li>None</li>"
	if len(studio.ActionItems) > 0 {
		items := make([]string, 0, len(studio.ActionItems))
		for _, item := range studio.ActionItems {
			items = append(items, "<li>"+html.EscapeString(item)+"</li>")
		}
		actionItems = strings.Join(items, "")
	}
	sourceItems := "<li>None</li>"
	if len(studio.SourceReports) > 0 {
		items := make([]string, 0, len(studio.SourceReports))
		for _, item := range studio.SourceReports {
			items = append(items, "<li>"+html.EscapeString(item)+"</li>")
		}
		sourceItems = strings.Join(items, "")
	}
	sectionBody := strings.Join(sections, "")
	if sectionBody == "" {
		sectionBody = `<section class="section"><p>No sections drafted.</p></section>`
	}
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
`, html.EscapeString(studio.Name), html.EscapeString(studio.IssueID), html.EscapeString(studio.Audience), html.EscapeString(studio.Period), html.EscapeString(studio.Name), html.EscapeString(studio.Recommendation()), html.EscapeString(firstNonEmpty(studio.Summary, "No summary drafted.")), sectionBody, actionItems, sourceItems)
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
			lines = append(lines, fmt.Sprintf("- %s: baseline=%v%s current=%v%s target%s%v%s delta=%+.2f%s met=%t", metric.Name, metric.Baseline, unitSuffix, metric.Current, unitSuffix, comparator, metric.Target, unitSuffix, metric.Delta(), unitSuffix, metric.MetTarget()))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

type EvaluationCriterion struct {
	Name   string
	Weight int
	Passed bool
	Detail string
}

type BenchmarkCase struct {
	CaseID           string
	Task             domain.Task
	ExpectedMedium   *string
	ExpectedApproved *bool
	ExpectedStatus   *string
	RequireReport    bool
}

type ReplayRecord struct {
	Task     domain.Task
	RunID    string
	Medium   string
	Approved bool
	Status   string
}

type ReplayOutcome struct {
	Matched      bool
	ReplayRecord ReplayRecord
	Mismatches   []string
	ReportPath   string
}

type BenchmarkExecutionDecision struct {
	Medium   string
	Approved bool
	Reason   string
}

type BenchmarkExecutionRun struct {
	TaskID string
	Status string
}

type BenchmarkExecutionRecord struct {
	Decision   BenchmarkExecutionDecision
	Run        BenchmarkExecutionRun
	ReportPath string
}

type BenchmarkResult struct {
	CaseID         string
	Score          int
	Passed         bool
	Criteria       []EvaluationCriterion
	Record         BenchmarkExecutionRecord
	Replay         ReplayOutcome
	DetailPagePath string
}

type BenchmarkComparison struct {
	CaseID        string
	BaselineScore int
	CurrentScore  int
	Delta         int
	Changed       bool
}

type BenchmarkSuiteResult struct {
	Results []BenchmarkResult
	Version string
}

func (s BenchmarkSuiteResult) Score() int {
	if len(s.Results) == 0 {
		return 0
	}
	total := 0
	for _, result := range s.Results {
		total += result.Score
	}
	return int(math.Round(float64(total) / float64(len(s.Results))))
}

func (s BenchmarkSuiteResult) Passed() bool {
	for _, result := range s.Results {
		if !result.Passed {
			return false
		}
	}
	return true
}

func (s BenchmarkSuiteResult) Compare(baseline BenchmarkSuiteResult) []BenchmarkComparison {
	byCase := make(map[string]BenchmarkResult, len(baseline.Results))
	for _, result := range baseline.Results {
		byCase[result.CaseID] = result
	}
	comparisons := make([]BenchmarkComparison, 0, len(s.Results))
	for _, result := range s.Results {
		baselineScore := 0
		if baselineResult, ok := byCase[result.CaseID]; ok {
			baselineScore = baselineResult.Score
		}
		delta := result.Score - baselineScore
		comparisons = append(comparisons, BenchmarkComparison{
			CaseID:        result.CaseID,
			BaselineScore: baselineScore,
			CurrentScore:  result.Score,
			Delta:         delta,
			Changed:       delta != 0,
		})
	}
	return comparisons
}

type BenchmarkRunner struct {
	StorageDir string
}

func (r BenchmarkRunner) RunCase(caseDef BenchmarkCase) (BenchmarkResult, error) {
	runID := "benchmark-" + caseDef.CaseID
	record, err := r.execute(caseDef.Task, runID, r.casePath(caseDef.CaseID, "task-run.md"), caseDef.RequireReport)
	if err != nil {
		return BenchmarkResult{}, err
	}
	criteria := r.evaluate(caseDef, record)
	replay, err := r.Replay(ReplayRecordFromExecution(caseDef.Task, runID, record))
	if err != nil {
		return BenchmarkResult{}, err
	}
	totalWeight := 0
	earnedWeight := 0
	passed := replay.Matched
	for _, item := range criteria {
		totalWeight += item.Weight
		if item.Passed {
			earnedWeight += item.Weight
		} else {
			passed = false
		}
	}
	score := 0
	if totalWeight > 0 {
		score = int(math.Round((float64(earnedWeight) / float64(totalWeight)) * 100))
	}
	detailPath := ""
	if strings.TrimSpace(r.StorageDir) != "" {
		detailPath = r.casePath(caseDef.CaseID, "run-detail.html")
		if err := WriteReport(detailPath, RenderRunReplayIndexPage(caseDef.CaseID, record, replay, criteria)); err != nil {
			return BenchmarkResult{}, err
		}
	}
	return BenchmarkResult{
		CaseID:         caseDef.CaseID,
		Score:          score,
		Passed:         passed,
		Criteria:       criteria,
		Record:         record,
		Replay:         replay,
		DetailPagePath: detailPath,
	}, nil
}

func (r BenchmarkRunner) RunSuite(cases []BenchmarkCase, version string) (BenchmarkSuiteResult, error) {
	results := make([]BenchmarkResult, 0, len(cases))
	for _, item := range cases {
		result, err := r.RunCase(item)
		if err != nil {
			return BenchmarkSuiteResult{}, err
		}
		results = append(results, result)
	}
	if version == "" {
		version = "current"
	}
	return BenchmarkSuiteResult{Results: results, Version: version}, nil
}

func ReplayRecordFromExecution(task domain.Task, runID string, record BenchmarkExecutionRecord) ReplayRecord {
	return ReplayRecord{
		Task:     task,
		RunID:    runID,
		Medium:   record.Decision.Medium,
		Approved: record.Decision.Approved,
		Status:   record.Run.Status,
	}
}

func (r BenchmarkRunner) Replay(replayRecord ReplayRecord) (ReplayOutcome, error) {
	record, err := r.execute(replayRecord.Task, replayRecord.RunID+"-replay", "", false)
	if err != nil {
		return ReplayOutcome{}, err
	}
	observed := ReplayRecordFromExecution(replayRecord.Task, replayRecord.RunID, record)
	mismatches := make([]string, 0, 3)
	if observed.Medium != replayRecord.Medium {
		mismatches = append(mismatches, fmt.Sprintf("medium expected %s got %s", replayRecord.Medium, observed.Medium))
	}
	if observed.Approved != replayRecord.Approved {
		mismatches = append(mismatches, fmt.Sprintf("approved expected %t got %t", replayRecord.Approved, observed.Approved))
	}
	if observed.Status != replayRecord.Status {
		mismatches = append(mismatches, fmt.Sprintf("status expected %s got %s", replayRecord.Status, observed.Status))
	}
	reportPath := ""
	if strings.TrimSpace(r.StorageDir) != "" {
		reportPath = r.casePath(replayRecord.RunID, "replay.html")
		if err := WriteReport(reportPath, RenderReplayDetailPage(replayRecord, observed, mismatches)); err != nil {
			return ReplayOutcome{}, err
		}
	}
	return ReplayOutcome{
		Matched:      len(mismatches) == 0,
		ReplayRecord: observed,
		Mismatches:   mismatches,
		ReportPath:   reportPath,
	}, nil
}

func (r BenchmarkRunner) evaluate(caseDef BenchmarkCase, record BenchmarkExecutionRecord) []EvaluationCriterion {
	return []EvaluationCriterion{
		r.criterion("decision-medium", 40, caseDef.ExpectedMedium, record.Decision.Medium),
		r.criterion("approval-gate", 30, caseDef.ExpectedApproved, record.Decision.Approved),
		r.criterion("final-status", 20, caseDef.ExpectedStatus, record.Run.Status),
		{
			Name:   "report-artifact",
			Weight: 10,
			Passed: !caseDef.RequireReport || strings.TrimSpace(record.ReportPath) != "",
			Detail: map[bool]string{true: "report emitted", false: "report missing"}[!caseDef.RequireReport || strings.TrimSpace(record.ReportPath) != ""],
		},
	}
}

func (r BenchmarkRunner) criterion(name string, weight int, expected any, actual any) EvaluationCriterion {
	if expected == nil {
		return EvaluationCriterion{Name: name, Weight: weight, Passed: true, Detail: "not asserted"}
	}
	switch typed := expected.(type) {
	case *string:
		if typed == nil {
			return EvaluationCriterion{Name: name, Weight: weight, Passed: true, Detail: "not asserted"}
		}
		expected = *typed
	case *bool:
		if typed == nil {
			return EvaluationCriterion{Name: name, Weight: weight, Passed: true, Detail: "not asserted"}
		}
		expected = *typed
	case *int:
		if typed == nil {
			return EvaluationCriterion{Name: name, Weight: weight, Passed: true, Detail: "not asserted"}
		}
		expected = *typed
	}
	passed := expected == actual
	return EvaluationCriterion{Name: name, Weight: weight, Passed: passed, Detail: fmt.Sprintf("expected %v got %v", expected, actual)}
}

func (r BenchmarkRunner) execute(task domain.Task, runID string, reportPath string, requireReport bool) (BenchmarkExecutionRecord, error) {
	decision := benchmarkDecision(task)
	status := benchmarkStatus(decision)
	record := BenchmarkExecutionRecord{
		Decision: decision,
		Run: BenchmarkExecutionRun{
			TaskID: task.ID,
			Status: status,
		},
	}
	if requireReport && strings.TrimSpace(reportPath) != "" {
		record.ReportPath = reportPath
		if err := WriteReport(reportPath, fmt.Sprintf("# Task Run Report\n\n- Run ID: %s\n- Task ID: %s\n- Medium: %s\n- Status: %s\n", runID, task.ID, decision.Medium, status)); err != nil {
			return BenchmarkExecutionRecord{}, err
		}
		detailPath := strings.TrimSuffix(reportPath, filepath.Ext(reportPath)) + ".html"
		if err := WriteReport(detailPath, fmt.Sprintf("<html><body><h1>Run Detail</h1><p>task=%s</p><p>status=%s</p></body></html>\n", html.EscapeString(task.ID), html.EscapeString(status))); err != nil {
			return BenchmarkExecutionRecord{}, err
		}
	}
	return record, nil
}

func benchmarkDecision(task domain.Task) BenchmarkExecutionDecision {
	if task.BudgetCents < 0 {
		return BenchmarkExecutionDecision{Medium: "none", Approved: false, Reason: "invalid budget"}
	}
	score := risk.ScoreTask(task, nil)
	if score.Level == domain.RiskHigh {
		return BenchmarkExecutionDecision{Medium: "vm", Approved: false, Reason: "requires approval for high-risk task"}
	}
	if slices.Contains(task.RequiredTools, "browser") {
		return BenchmarkExecutionDecision{Medium: "browser", Approved: true, Reason: "browser automation task"}
	}
	if score.Level == domain.RiskMedium {
		return BenchmarkExecutionDecision{Medium: "docker", Approved: true, Reason: "medium risk in docker"}
	}
	return BenchmarkExecutionDecision{Medium: "docker", Approved: true, Reason: "default low risk path"}
}

func benchmarkStatus(decision BenchmarkExecutionDecision) string {
	if decision.Approved {
		return "approved"
	}
	if decision.Medium == "none" {
		return "paused"
	}
	return "needs-approval"
}

func (r BenchmarkRunner) casePath(caseID string, fileName string) string {
	if strings.TrimSpace(r.StorageDir) == "" {
		return fileName
	}
	return filepath.Join(r.StorageDir, caseID, fileName)
}

func RenderBenchmarkSuiteReport(suite BenchmarkSuiteResult, baseline *BenchmarkSuiteResult) string {
	lines := []string{
		"# Benchmark Suite Report",
		"",
		fmt.Sprintf("- Version: %s", suite.Version),
		fmt.Sprintf("- Cases: %d", len(suite.Results)),
		fmt.Sprintf("- Passed: %t", suite.Passed()),
		fmt.Sprintf("- Score: %d", suite.Score()),
		"",
		"## Cases",
		"",
	}
	if len(suite.Results) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, result := range suite.Results {
			lines = append(lines, fmt.Sprintf("- %s: score=%d passed=%t replay=%t", result.CaseID, result.Score, result.Passed, result.Replay.Matched))
		}
	}
	lines = append(lines, "", "## Comparison", "")
	if baseline == nil {
		lines = append(lines, "- No baseline provided")
	} else {
		lines = append(lines, fmt.Sprintf("- Baseline Version: %s", baseline.Version), fmt.Sprintf("- Score Delta: %d", suite.Score()-baseline.Score()))
		for _, item := range suite.Compare(*baseline) {
			lines = append(lines, fmt.Sprintf("- %s: baseline=%d current=%d delta=%d", item.CaseID, item.BaselineScore, item.CurrentScore, item.Delta))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderReplayDetailPage(expected ReplayRecord, observed ReplayRecord, mismatches []string) string {
	items := "<li>None</li>"
	if len(mismatches) > 0 {
		parts := make([]string, 0, len(mismatches))
		for _, item := range mismatches {
			parts = append(parts, "<li>"+html.EscapeString(item)+"</li>")
		}
		items = strings.Join(parts, "")
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
  <body>
    <h1>Replay Detail</h1>
    <section><h2>Overview</h2><p>Task %s</p></section>
    <section><h2>Timeline / Log Sync</h2><p>expected %s observed %s</p></section>
    <section><h2>Split View</h2><p>status %s vs %s</p></section>
    <section><h2>Replay</h2><ul>%s</ul></section>
    <section><h2>Reports</h2><p>needs-approval marker: %s</p></section>
  </body>
</html>
`, html.EscapeString(expected.Task.ID), html.EscapeString(expected.Medium), html.EscapeString(observed.Medium), html.EscapeString(expected.Status), html.EscapeString(observed.Status), items, html.EscapeString(observed.Status))
}

func RenderRunReplayIndexPage(caseID string, record BenchmarkExecutionRecord, replay ReplayOutcome, criteria []EvaluationCriterion) string {
	reportPath := record.ReportPath
	if strings.TrimSpace(reportPath) == "" {
		reportPath = "n/a"
	}
	detailPath := "n/a"
	if strings.TrimSpace(record.ReportPath) != "" {
		detailPath = strings.TrimSuffix(record.ReportPath, filepath.Ext(record.ReportPath)) + ".html"
	}
	replayPath := replay.ReportPath
	if strings.TrimSpace(replayPath) == "" {
		replayPath = "n/a"
	}
	criteriaLines := make([]string, 0, len(criteria))
	for _, item := range criteria {
		criteriaLines = append(criteriaLines, fmt.Sprintf("<li>%s: %s | weight=%d | passed=%t</li>", html.EscapeString(item.Name), html.EscapeString(item.Detail), item.Weight, item.Passed))
	}
	if len(criteriaLines) == 0 {
		criteriaLines = append(criteriaLines, "<li>None</li>")
	}
	mismatchLines := []string{"<li>None</li>"}
	if len(replay.Mismatches) > 0 {
		mismatchLines = mismatchLines[:0]
		for _, item := range replay.Mismatches {
			mismatchLines = append(mismatchLines, "<li>"+html.EscapeString(item)+"</li>")
		}
	}
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
  <body>
    <h1>Run Detail Index</h1>
    <section><h2>Overview</h2><p>Case %s</p></section>
    <section><h2>Timeline / Log Sync</h2><p>status=%s medium=%s</p></section>
    <section><h2>Acceptance</h2><ul>%s</ul></section>
    <section><h2>Artifacts</h2><p>%s %s %s</p></section>
    <section><h2>Reports</h2><p>%s %s %s</p><p>decision-medium</p></section>
    <section><h2>Replay</h2><ul>%s</ul></section>
  </body>
</html>
`, html.EscapeString(caseID), html.EscapeString(record.Run.Status), html.EscapeString(record.Decision.Medium), strings.Join(criteriaLines, ""), html.EscapeString(reportPath), html.EscapeString(detailPath), html.EscapeString(replayPath), html.EscapeString(reportPath), html.EscapeString(detailPath), html.EscapeString(replayPath), strings.Join(mismatchLines, ""))
}
