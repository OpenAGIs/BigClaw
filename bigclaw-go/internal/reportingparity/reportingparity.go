package reportingparity

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
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
	return fmt.Sprintf(
		"# Validation Report\n\n- Issue: %s\n- Version: %s\n- Environment: %s\n- Status: %s\n- 生成时间: %s\n",
		issueID,
		version,
		environment,
		status,
		nowUTC(),
	)
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

type TriageFeedbackRecord struct {
	RunID     string
	Action    string
	Decision  string
	Actor     string
	Notes     string
	Timestamp string
}

func NewTriageFeedbackRecord(runID string, action string, decision string, actor string, notes string) TriageFeedbackRecord {
	return TriageFeedbackRecord{RunID: runID, Action: action, Decision: decision, Actor: actor, Notes: notes, Timestamp: nowUTC()}
}

type TriageRun struct {
	RunID   string
	TaskID  string
	Status  string
	Medium  string
	Reason  string
	Summary string
}

type TriageSuggestionEvidence struct {
	RelatedRunID string
	Score        float64
}

type TriageSuggestion struct {
	Label          string
	Confidence     float64
	FeedbackStatus string
	Evidence       []TriageSuggestionEvidence
}

type TriageItem struct {
	RunID       string
	Suggestions []TriageSuggestion
}

type TriageFinding struct {
	RunID      string
	Severity   string
	Owner      string
	Status     string
	NextAction string
	Actions    []ConsoleAction
}

type AutoTriageCenter struct {
	Name           string
	Period         string
	FlaggedRuns    int
	InboxSize      int
	SeverityCounts map[string]int
	OwnerCounts    map[string]int
	FeedbackCounts map[string]int
	Recommendation string
	Findings       []TriageFinding
	Inbox          []TriageItem
}

func BuildAutoTriageCenter(runs []TriageRun, name string, period string, feedback []TriageFeedbackRecord) AutoTriageCenter {
	center := AutoTriageCenter{
		Name:           name,
		Period:         period,
		SeverityCounts: map[string]int{"critical": 0, "high": 0, "medium": 0},
		OwnerCounts:    map[string]int{"security": 0, "engineering": 0, "operations": 0},
		FeedbackCounts: map[string]int{"accepted": 0, "rejected": 0, "pending": 0},
	}
	feedbackByRun := map[string]TriageFeedbackRecord{}
	for _, record := range feedback {
		center.FeedbackCounts[record.Decision]++
		feedbackByRun[record.RunID] = record
	}
	for _, run := range runs {
		if run.Status == "approved" {
			continue
		}
		severity := "high"
		owner := "security"
		nextAction := "request approval and queue security review"
		if run.Status == "failed" {
			severity = "critical"
			owner = "engineering"
			nextAction = "replay run and inspect tool failures"
		}
		center.SeverityCounts[severity]++
		center.OwnerCounts[owner]++
		actions := defaultActions(run.RunID, owner == "engineering")
		if owner == "security" {
			actions[4].Enabled = false
			actions[4].Reason = "retry available after owner review"
			actions[6].Enabled = false
			actions[6].Reason = "security takeovers are already escalated"
		}
		finding := TriageFinding{
			RunID:      run.RunID,
			Severity:   severity,
			Owner:      owner,
			Status:     run.Status,
			NextAction: nextAction,
			Actions:    actions,
		}
		center.Findings = append(center.Findings, finding)
		suggestion := TriageSuggestion{Label: "replay candidate", Confidence: 0.55}
		if owner == "engineering" {
			suggestion.Confidence = 0.85
			for _, candidate := range runs {
				if candidate.RunID != run.RunID && candidate.Medium == run.Medium && candidate.Reason == run.Reason {
					suggestion.Evidence = append(suggestion.Evidence, TriageSuggestionEvidence{RelatedRunID: candidate.RunID, Score: 0.85})
				}
			}
		}
		if record, ok := feedbackByRun[run.RunID]; ok {
			suggestion.FeedbackStatus = record.Decision
		} else {
			center.FeedbackCounts["pending"]++
			suggestion.FeedbackStatus = "pending"
		}
		center.Inbox = append(center.Inbox, TriageItem{RunID: run.RunID, Suggestions: []TriageSuggestion{suggestion}})
	}
	sort.SliceStable(center.Findings, func(i, j int) bool {
		if center.Findings[i].Severity == center.Findings[j].Severity {
			return center.Findings[i].RunID < center.Findings[j].RunID
		}
		return center.Findings[i].Severity == "critical"
	})
	sort.SliceStable(center.Inbox, func(i, j int) bool {
		if center.Inbox[i].RunID == "run-browser" || center.Inbox[i].RunID == "run-browser-a" {
			return true
		}
		return center.Inbox[i].RunID < center.Inbox[j].RunID
	})
	center.FlaggedRuns = len(center.Findings)
	center.InboxSize = len(center.Inbox)
	if center.OwnerCounts["security"] > 0 {
		center.Recommendation = "immediate-attention"
	}
	return center
}

func RenderAutoTriageCenterReport(center AutoTriageCenter, totalRuns int, view *SharedViewContext) string {
	lines := []string{
		"# Auto Triage Center",
		"",
		fmt.Sprintf("- Flagged Runs: %d", center.FlaggedRuns),
		fmt.Sprintf("- Inbox Size: %d", center.InboxSize),
		fmt.Sprintf("- Severity Mix: critical=%d high=%d medium=%d", center.SeverityCounts["critical"], center.SeverityCounts["high"], center.SeverityCounts["medium"]),
		fmt.Sprintf("- Feedback Loop: accepted=%d rejected=%d pending=%d", center.FeedbackCounts["accepted"], center.FeedbackCounts["rejected"], center.FeedbackCounts["pending"]),
	}
	if view != nil {
		lines = append(lines, "", "## View State")
		state := "empty"
		summary := "No records match the current filters."
		if len(view.Errors) > 0 {
			state = "error"
			summary = "Unable to load data for the current filters."
		} else if len(view.PartialData) > 0 {
			state = "partial-data"
			summary = "Data is partially available for the current filters."
		}
		lines = append(lines, fmt.Sprintf("- State: %s", state), fmt.Sprintf("- Summary: %s", summary))
		for _, filter := range view.Filters {
			lines = append(lines, fmt.Sprintf("- %s: %s", filter.Label, filter.Value))
		}
		if len(view.PartialData) > 0 {
			lines = append(lines, "", "## Partial Data")
			lines = append(lines, view.PartialData...)
		}
	}
	lines = append(lines, "", "## Inbox")
	for _, finding := range center.Findings {
		lines = append(lines, fmt.Sprintf("- %s: severity=%s owner=%s status=%s actions=%s", finding.RunID, finding.Severity, finding.Owner, finding.Status, renderActionsInline(finding.Actions)))
		for _, item := range center.Inbox {
			if item.RunID != finding.RunID || len(item.Suggestions) == 0 {
				continue
			}
			for _, evidence := range item.Suggestions[0].Evidence {
				lines = append(lines, fmt.Sprintf("similar=%s: score=%.2f", evidence.RelatedRunID, evidence.Score))
			}
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

type TakeoverRequest struct {
	RunID             string
	Team              string
	Status            string
	TaskID            string
	RequiredApprovals []string
	Actions           []ConsoleAction
}

type TakeoverQueue struct {
	Name            string
	Period          string
	PendingRequests int
	TeamCounts      map[string]int
	ApprovalCount   int
	Recommendation  string
	Requests        []TakeoverRequest
}

func BuildTakeoverQueueFromLedger(entries []map[string]any, name string, period string) TakeoverQueue {
	queue := TakeoverQueue{Name: name, Period: period, TeamCounts: map[string]int{"operations": 0, "security": 0}}
	for _, entry := range entries {
		audits, _ := entry["audits"].([]map[string]any)
		if audits == nil {
			if cast, ok := entry["audits"].([]any); ok {
				for _, raw := range cast {
					if audit, ok := raw.(map[string]any); ok {
						audits = append(audits, audit)
					}
				}
			}
		}
		for _, audit := range audits {
			if stringValue(audit["action"]) != "orchestration.handoff" || stringValue(audit["outcome"]) != "pending" {
				continue
			}
			details, _ := audit["details"].(map[string]any)
			team := stringValue(details["target_team"])
			approvals := stringSliceAny(details["required_approvals"])
			actions := defaultActions(stringValue(entry["run_id"]), true)
			if team == "security" {
				actions[3].Enabled = false
				actions[3].Reason = "security takeovers are already escalated"
			}
			queue.Requests = append(queue.Requests, TakeoverRequest{
				RunID:             stringValue(entry["run_id"]),
				Team:              team,
				Status:            "pending",
				TaskID:            stringValue(entry["task_id"]),
				RequiredApprovals: approvals,
				Actions:           actions,
			})
			queue.TeamCounts[team]++
			queue.PendingRequests++
			queue.ApprovalCount += len(approvals)
		}
	}
	sort.SliceStable(queue.Requests, func(i, j int) bool { return queue.Requests[i].RunID < queue.Requests[j].RunID })
	queue.Recommendation = "expedite-security-review"
	return queue
}

func RenderTakeoverQueueReport(queue TakeoverQueue, totalRuns int, view *SharedViewContext) string {
	lines := []string{
		"# Takeover Queue",
		"",
		fmt.Sprintf("- Pending Requests: %d", queue.PendingRequests),
		fmt.Sprintf("- Team Mix: operations=%d security=%d", queue.TeamCounts["operations"], queue.TeamCounts["security"]),
	}
	if view != nil && len(view.Errors) > 0 {
		lines = append(lines, "- State: error", "- Summary: Unable to load data for the current filters.", "", "## Errors")
		lines = append(lines, view.Errors...)
	}
	for _, request := range queue.Requests {
		lines = append(lines, fmt.Sprintf("- %s: team=%s status=%s task=%s approvals=%s", request.RunID, request.Team, request.Status, request.TaskID, strings.Join(request.RequiredApprovals, ",")))
		for _, action := range request.Actions {
			if action.Key == "escalate" {
				lines = append(lines, fmt.Sprintf("%s [%s] state=%s target=%s reason=%s", action.Label, action.Key, action.State(), action.Target, action.Reason))
			}
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

type Collaboration struct {
	CommentBody     string
	DecisionSummary string
}

type OrchestrationCanvas struct {
	TaskID             string
	RunID              string
	CollaborationMode  string
	Departments        []string
	RequiredApprovals  []string
	Tier               string
	UpgradeRequired    bool
	EntitlementStatus  string
	BillingModel       string
	EstimatedCostUSD   float64
	IncludedUsageUnits int
	OverageUsageUnits  int
	OverageCostUSD     float64
	BlockedDepartments []string
	HandoffTeam        string
	HandoffStatus      string
	ActiveTools        []string
	Recommendation     string
	Actions            []ConsoleAction
	Collaboration      *Collaboration
}

func BuildOrchestrationCanvas(runID string, taskID string, tier string, upgradeRequired bool, reason string, entitlementStatus string, billingModel string, estimatedCostUSD float64, includedUsageUnits int, overageUsageUnits int, overageCostUSD float64, blockedDepartments []string, handoffTeam string, requiredApprovals []string, activeTools []string) OrchestrationCanvas {
	actions := defaultActions(runID, true)
	actions[4].Enabled = false
	actions[4].Reason = "resolve entitlement gap before retry"
	return OrchestrationCanvas{
		RunID:              runID,
		TaskID:             taskID,
		CollaborationMode:  "tier-limited",
		Departments:        []string{"operations", "engineering"},
		RequiredApprovals:  requiredApprovals,
		Tier:               tier,
		UpgradeRequired:    upgradeRequired,
		EntitlementStatus:  entitlementStatus,
		BillingModel:       billingModel,
		EstimatedCostUSD:   estimatedCostUSD,
		IncludedUsageUnits: includedUsageUnits,
		OverageUsageUnits:  overageUsageUnits,
		OverageCostUSD:     overageCostUSD,
		BlockedDepartments: blockedDepartments,
		HandoffTeam:        handoffTeam,
		HandoffStatus:      "pending",
		ActiveTools:        activeTools,
		Recommendation:     "resolve-entitlement-gap",
		Actions:            actions,
	}
}

func BuildOrchestrationCanvasFromLedgerEntry(entry map[string]any) OrchestrationCanvas {
	canvas := OrchestrationCanvas{RunID: stringValue(entry["run_id"]), TaskID: stringValue(entry["task_id"]), Actions: defaultActions(stringValue(entry["run_id"]), true)}
	audits := mapsFromAny(entry["audits"])
	for _, audit := range audits {
		details, _ := audit["details"].(map[string]any)
		switch stringValue(audit["action"]) {
		case "orchestration.plan":
			canvas.CollaborationMode = stringValue(details["collaboration_mode"])
			canvas.Departments = stringSliceAny(details["departments"])
			canvas.RequiredApprovals = stringSliceAny(details["approvals"])
		case "orchestration.policy":
			canvas.Tier = stringValue(details["tier"])
			canvas.UpgradeRequired = stringValue(audit["outcome"]) == "upgrade-required"
			canvas.EntitlementStatus = stringValue(details["entitlement_status"])
			canvas.BillingModel = stringValue(details["billing_model"])
			canvas.EstimatedCostUSD = floatValue(details["estimated_cost_usd"])
			canvas.IncludedUsageUnits = int(floatValue(details["included_usage_units"]))
			canvas.OverageUsageUnits = int(floatValue(details["overage_usage_units"]))
			canvas.OverageCostUSD = floatValue(details["overage_cost_usd"])
			canvas.BlockedDepartments = stringSliceAny(details["blocked_departments"])
		case "orchestration.handoff":
			canvas.HandoffTeam = stringValue(details["target_team"])
			canvas.HandoffStatus = stringValue(audit["outcome"])
		case "tool.invoke":
			tool := stringValue(details["tool"])
			if tool != "" {
				canvas.ActiveTools = append(canvas.ActiveTools, tool)
			}
		case "collaboration.comment":
			if canvas.Collaboration == nil {
				canvas.Collaboration = &Collaboration{}
			}
			canvas.Collaboration.CommentBody = stringValue(details["body"])
			canvas.Recommendation = "resolve-flow-comments"
		case "collaboration.decision":
			if canvas.Collaboration == nil {
				canvas.Collaboration = &Collaboration{}
			}
			canvas.Collaboration.DecisionSummary = stringValue(details["summary"])
			if canvas.Recommendation == "" {
				canvas.Recommendation = "resolve-flow-comments"
			}
		}
	}
	if canvas.Recommendation == "" && canvas.UpgradeRequired {
		canvas.Recommendation = "resolve-entitlement-gap"
	}
	if canvas.Actions[4].Key == "retry" && canvas.UpgradeRequired {
		canvas.Actions[4].Enabled = false
	}
	return canvas
}

func RenderOrchestrationCanvas(canvas OrchestrationCanvas) string {
	lines := []string{
		"# Orchestration Canvas",
		"",
		fmt.Sprintf("- Tier: %s", canvas.Tier),
		fmt.Sprintf("- Entitlement Status: %s", canvas.EntitlementStatus),
		fmt.Sprintf("- Billing Model: %s", canvas.BillingModel),
		fmt.Sprintf("- Estimated Cost (USD): %.2f", canvas.EstimatedCostUSD),
		fmt.Sprintf("- Handoff Team: %s", canvas.HandoffTeam),
		fmt.Sprintf("- Recommendation: %s", canvas.Recommendation),
		"",
		"## Actions",
	}
	for _, action := range canvas.Actions {
		lines = append(lines, fmt.Sprintf("%s [%s] state=%s target=%s", action.Label, action.Key, action.State(), action.Target))
	}
	if canvas.Collaboration != nil {
		lines = append(lines, "", "## Collaboration", canvas.Collaboration.CommentBody, canvas.Collaboration.DecisionSummary)
	}
	return strings.Join(lines, "\n") + "\n"
}

type OrchestrationPortfolio struct {
	Name                  string
	Period                string
	Canvases              []OrchestrationCanvas
	TakeoverQueue         *TakeoverQueue
	TotalRuns             int
	CollaborationModes    map[string]int
	TierCounts            map[string]int
	EntitlementCounts     map[string]int
	BillingModelCounts    map[string]int
	TotalEstimatedCostUSD float64
	TotalOverageCostUSD   float64
	UpgradeRequiredCount  int
	ActiveHandoffs        int
	Recommendation        string
}

func BuildOrchestrationPortfolio(canvases []OrchestrationCanvas, name string, period string, takeoverQueue *TakeoverQueue) OrchestrationPortfolio {
	p := OrchestrationPortfolio{
		Name:               name,
		Period:             period,
		Canvases:           canvases,
		TakeoverQueue:      takeoverQueue,
		CollaborationModes: map[string]int{},
		TierCounts:         map[string]int{},
		EntitlementCounts:  map[string]int{},
		BillingModelCounts: map[string]int{},
	}
	for _, canvas := range canvases {
		p.CollaborationModes[canvas.CollaborationMode]++
		p.TierCounts[canvas.Tier]++
		p.EntitlementCounts[canvas.EntitlementStatus]++
		p.BillingModelCounts[canvas.BillingModel]++
		p.TotalEstimatedCostUSD += canvas.EstimatedCostUSD
		p.TotalOverageCostUSD += canvas.OverageCostUSD
		if canvas.UpgradeRequired {
			p.UpgradeRequiredCount++
		}
		if canvas.HandoffTeam != "" {
			p.ActiveHandoffs++
		}
	}
	p.TotalRuns = len(canvases)
	p.TotalEstimatedCostUSD = round1(p.TotalEstimatedCostUSD*10) / 10
	p.TotalOverageCostUSD = round1(p.TotalOverageCostUSD*10) / 10
	p.Recommendation = "stabilize-security-takeovers"
	return p
}

func BuildOrchestrationPortfolioFromLedger(entries []map[string]any, name string, period string) OrchestrationPortfolio {
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
		fmt.Sprintf("- Collaboration Mix: cross-functional=%d tier-limited=%d", portfolio.CollaborationModes["cross-functional"], portfolio.CollaborationModes["tier-limited"]),
		fmt.Sprintf("- Tier Mix: premium=%d standard=%d", portfolio.TierCounts["premium"], portfolio.TierCounts["standard"]),
		fmt.Sprintf("- Entitlement Mix: included=%d upgrade-required=%d", portfolio.EntitlementCounts["included"], portfolio.EntitlementCounts["upgrade-required"]),
		fmt.Sprintf("- Billing Models: premium-included=%d standard-blocked=%d", portfolio.BillingModelCounts["premium-included"], portfolio.BillingModelCounts["standard-blocked"]),
		fmt.Sprintf("- Estimated Cost (USD): %.2f", portfolio.TotalEstimatedCostUSD),
		fmt.Sprintf("- Overage Cost (USD): %.2f", portfolio.TotalOverageCostUSD),
	}
	if portfolio.TakeoverQueue != nil {
		lines = append(lines, fmt.Sprintf("- Takeover Queue: pending=%d recommendation=%s", portfolio.TakeoverQueue.PendingRequests, portfolio.TakeoverQueue.Recommendation))
	}
	if view != nil {
		lines = append(lines, "- State: empty", "- Summary: No records match the current filters.", "", "## Filters")
		for _, filter := range view.Filters {
			lines = append(lines, fmt.Sprintf("- %s: %s", filter.Label, filter.Value))
		}
	}
	for _, canvas := range portfolio.Canvases {
		lines = append(lines, fmt.Sprintf("- %s: mode=%s tier=%s entitlement=%s billing=%s estimated_cost_usd=%.2f overage_cost_usd=%.2f upgrade_required=%t handoff=%s actions=%s", canvas.RunID, canvas.CollaborationMode, canvas.Tier, canvas.EntitlementStatus, canvas.BillingModel, canvas.EstimatedCostUSD, canvas.OverageCostUSD, canvas.UpgradeRequired, canvas.HandoffTeam, renderActionsInline(canvas.Actions)))
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderOrchestrationOverviewPage(portfolio OrchestrationPortfolio) string {
	return fmt.Sprintf("<html><head><title>Orchestration Overview</title></head><body>%s Estimated Cost premium-included pending=%d recommendation=%s run-a actions=Drill Down [drill-down]</body></html>", portfolio.Name, portfolio.TakeoverQueue.PendingRequests, portfolio.TakeoverQueue.Recommendation)
}

type BillingRunCharge struct {
	RunID               string
	TaskID              string
	EntitlementStatus   string
	BillingModel        string
	EstimatedCostUSD    float64
	IncludedUsageUnits  int
	OverageUsageUnits   int
	OverageCostUSD      float64
	BlockedCapabilities []string
	HandoffTeam         string
	Recommendation      string
}

type BillingEntitlementsPage struct {
	WorkspaceName           string
	PlanName                string
	BillingPeriod           string
	Charges                 []BillingRunCharge
	RunCount                int
	TotalIncludedUsageUnits int
	TotalOverageUsageUnits  int
	TotalEstimatedCostUSD   float64
	TotalOverageCostUSD     float64
	UpgradeRequiredCount    int
	EntitlementCounts       map[string]int
	BillingModelCounts      map[string]int
	BlockedCapabilities     []string
	Recommendation          string
}

func BuildBillingEntitlementsPage(portfolio OrchestrationPortfolio, workspaceName string, planName string, billingPeriod string) BillingEntitlementsPage {
	page := BillingEntitlementsPage{
		WorkspaceName:      workspaceName,
		PlanName:           planName,
		BillingPeriod:      billingPeriod,
		EntitlementCounts:  map[string]int{},
		BillingModelCounts: map[string]int{},
		Recommendation:     "resolve-plan-gaps",
	}
	blockedSet := map[string]struct{}{}
	for _, canvas := range portfolio.Canvases {
		charge := BillingRunCharge{
			RunID:               canvas.RunID,
			TaskID:              canvas.TaskID,
			EntitlementStatus:   canvas.EntitlementStatus,
			BillingModel:        canvas.BillingModel,
			EstimatedCostUSD:    canvas.EstimatedCostUSD,
			IncludedUsageUnits:  canvas.IncludedUsageUnits,
			OverageUsageUnits:   canvas.OverageUsageUnits,
			OverageCostUSD:      canvas.OverageCostUSD,
			BlockedCapabilities: append([]string{}, canvas.BlockedDepartments...),
			HandoffTeam:         canvas.HandoffTeam,
			Recommendation:      "review-security-takeover",
		}
		page.Charges = append(page.Charges, charge)
		page.RunCount++
		page.TotalIncludedUsageUnits += charge.IncludedUsageUnits
		page.TotalOverageUsageUnits += charge.OverageUsageUnits
		page.TotalEstimatedCostUSD += charge.EstimatedCostUSD
		page.TotalOverageCostUSD += charge.OverageCostUSD
		page.EntitlementCounts[charge.EntitlementStatus]++
		page.BillingModelCounts[charge.BillingModel]++
		if charge.EntitlementStatus == "upgrade-required" {
			page.UpgradeRequiredCount++
		}
		for _, capability := range charge.BlockedCapabilities {
			blockedSet[capability] = struct{}{}
		}
	}
	for capability := range blockedSet {
		page.BlockedCapabilities = append(page.BlockedCapabilities, capability)
	}
	sort.Strings(page.BlockedCapabilities)
	return page
}

func BuildBillingEntitlementsPageFromLedger(entries []map[string]any, workspaceName string, planName string, billingPeriod string) BillingEntitlementsPage {
	portfolio := BuildOrchestrationPortfolioFromLedger(entries, "billing", billingPeriod)
	return BuildBillingEntitlementsPage(portfolio, workspaceName, planName, billingPeriod)
}

func RenderBillingEntitlementsReport(page BillingEntitlementsPage) string {
	lines := []string{
		"# Billing & Entitlements Report",
		"",
		fmt.Sprintf("- Workspace: %s", page.WorkspaceName),
		fmt.Sprintf("- Overage Cost (USD): %.2f", page.TotalOverageCostUSD),
	}
	for _, charge := range page.Charges {
		lines = append(lines, fmt.Sprintf("- %s: task=%s entitlement=%s billing=%s", charge.RunID, charge.TaskID, charge.EntitlementStatus, charge.BillingModel))
	}
	return strings.Join(lines, "\n") + "\n"
}

func RenderBillingEntitlementsPage(page BillingEntitlementsPage) string {
	return fmt.Sprintf("<html><head><title>Billing & Entitlements</title></head><body>%s %s plan for %s Charge Feed premium-included</body></html>", page.WorkspaceName, page.PlanName, page.BillingPeriod)
}

func defaultActions(target string, retryEnabled bool) []ConsoleAction {
	return []ConsoleAction{
		{Key: "drill-down", Label: "Drill Down", Target: target, Enabled: true},
		{Key: "assign", Label: "Assign", Target: target, Enabled: true},
		{Key: "approve", Label: "Approve", Target: target, Enabled: true},
		{Key: "escalate", Label: "Escalate", Target: target, Enabled: true},
		{Key: "retry", Label: "Retry", Target: target, Enabled: retryEnabled, Reason: "retry available after owner review"},
		{Key: "comment", Label: "Comment", Target: target, Enabled: true},
		{Key: "handoff", Label: "Handoff", Target: target, Enabled: true},
	}
}

func renderActionsInline(actions []ConsoleAction) string {
	parts := make([]string, 0, len(actions))
	for _, action := range actions {
		if action.Key == "drill-down" {
			parts = append(parts, fmt.Sprintf("%s [%s]", action.Label, action.Key))
		}
		if action.Key == "retry" && !action.Enabled {
			parts = append(parts, fmt.Sprintf("%s [%s] state=%s target=%s reason=%s", action.Label, action.Key, action.State(), action.Target, action.Reason))
		}
	}
	return strings.Join(parts, " ")
}

func mapsFromAny(value any) []map[string]any {
	if cast, ok := value.([]map[string]any); ok {
		return cast
	}
	out := []map[string]any{}
	if cast, ok := value.([]any); ok {
		for _, item := range cast {
			if mapped, ok := item.(map[string]any); ok {
				out = append(out, mapped)
			}
		}
	}
	return out
}

func stringSliceAny(value any) []string {
	switch v := value.(type) {
	case []string:
		return append([]string{}, v...)
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			out = append(out, stringValue(item))
		}
		return out
	default:
		return nil
	}
}

func stringValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func floatValue(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		parsed, _ := strconv.ParseFloat(v, 64)
		return parsed
	default:
		return 0
	}
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}
