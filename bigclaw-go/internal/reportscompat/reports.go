package reportscompat

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
	"bigclaw-go/internal/repo"
)

type ConsoleAction struct {
	ID      string
	Label   string
	Target  string
	Enabled bool
	Reason  string
}

func NewConsoleAction(id, label, target string) ConsoleAction {
	return ConsoleAction{ID: id, Label: label, Target: target, Enabled: true}
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

type ReportStudioBundle struct {
	MarkdownPath string
	HTMLPath     string
	TextPath     string
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
	if m.HigherIsBetter || m.HigherIsBetter == false && m.Unit == "" {
	}
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

func (p PilotScorecard) MetricsMet() int {
	count := 0
	for _, metric := range p.Metrics {
		if metric.Met() {
			count++
		}
	}
	return count
}

func (p PilotScorecard) MonthlyNetValue() float64 {
	return p.MonthlyBenefit - p.MonthlyCost
}

func (p PilotScorecard) PaybackMonths() *float64 {
	net := p.MonthlyNetValue()
	if net <= 0 {
		return nil
	}
	value := math.Round((p.ImplementationCost/net)*10) / 10
	return &value
}

func (p PilotScorecard) AnnualizedROI() float64 {
	denominator := p.MonthlyCost + (p.ImplementationCost / 12)
	if denominator <= 0 {
		return 0
	}
	value := ((p.MonthlyBenefit / denominator) - 1) * 100
	return math.Round(value*10) / 10
}

func (p PilotScorecard) Recommendation() string {
	if p.MonthlyNetValue() <= 0 || !p.BenchmarkPassed {
		return "hold"
	}
	if p.MetricsMet() == len(p.Metrics) && p.AnnualizedROI() >= 100 {
		return "go"
	}
	return "iterate"
}

type PilotPortfolio struct {
	Name       string
	Period     string
	Scorecards []PilotScorecard
}

func (p PilotPortfolio) TotalMonthlyNetValue() float64 {
	total := 0.0
	for _, card := range p.Scorecards {
		total += card.MonthlyNetValue()
	}
	return total
}

func (p PilotPortfolio) AverageROI() float64 {
	if len(p.Scorecards) == 0 {
		return 0
	}
	total := 0.0
	for _, card := range p.Scorecards {
		total += card.AnnualizedROI()
	}
	return math.Round((total/float64(len(p.Scorecards)))*10) / 10
}

func (p PilotPortfolio) RecommendationCounts() map[string]int {
	counts := map[string]int{"go": 0, "iterate": 0, "hold": 0}
	for _, card := range p.Scorecards {
		counts[card.Recommendation()]++
	}
	return counts
}

func (p PilotPortfolio) Recommendation() string {
	if p.RecommendationCounts()["hold"] > 0 {
		return "caution"
	}
	return "continue"
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
	Documentation        []DocumentationArtifact
	Items                []LaunchChecklistItem
	DocumentationStatus  map[string]bool
	CompletedItems       int
	MissingDocumentation []string
}

func (c LaunchChecklist) Ready() bool {
	return len(c.MissingDocumentation) == 0 && c.CompletedItems == len(c.Items)
}

type FinalDeliveryChecklist struct {
	IssueID                        string
	RequiredOutputs                []DocumentationArtifact
	RecommendedDocumentation       []DocumentationArtifact
	RequiredOutputStatus           map[string]bool
	RecommendedDocumentationStatus map[string]bool
	GeneratedRequiredOutputs       int
	GeneratedRecommendedDocs       int
	MissingRequiredOutputs         []string
	MissingRecommendedDocs         []string
}

func (c FinalDeliveryChecklist) Ready() bool {
	return len(c.MissingRequiredOutputs) == 0
}

type ClosureDecision struct {
	Allowed    bool
	Reason     string
	ReportPath string
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
	Collaboration *repo.CollaborationThread
}

func (v SharedViewContext) State() string {
	switch {
	case len(v.Errors) > 0:
		return "error"
	case len(v.PartialData) > 0:
		return "partial-data"
	case v.Loading:
		return "loading"
	case v.ResultCount == 0:
		return "empty"
	default:
		return "ready"
	}
}

func (v SharedViewContext) Summary() string {
	switch v.State() {
	case "error":
		return "Unable to load data for the current filters."
	case "empty":
		return "No records match the current filters."
	case "partial-data":
		return "Some sources are still backfilling."
	case "loading":
		return "Loading filtered data."
	default:
		return "Results loaded."
	}
}

type RunAudit struct {
	Action  string
	Actor   string
	Outcome string
	Reason  string
	Tool    string
	Details map[string]any
}

type RunRecord struct {
	Task     domain.Task
	RunID    string
	Medium   string
	Status   string
	Summary  string
	Audits   []RunAudit
	TraceLog []string
}

func NewRunRecord(task domain.Task, runID, medium string) RunRecord {
	return RunRecord{Task: task, RunID: runID, Medium: medium, Status: "running"}
}

func (r *RunRecord) Trace(stage, outcome string) {
	r.TraceLog = append(r.TraceLog, stage+":"+outcome)
}

func (r *RunRecord) Audit(action, actor, outcome string, details map[string]any) {
	audit := RunAudit{Action: action, Actor: actor, Outcome: outcome, Details: details}
	if reason, ok := details["reason"].(string); ok {
		audit.Reason = reason
	}
	if tool, ok := details["tool"].(string); ok {
		audit.Tool = tool
	}
	r.Audits = append(r.Audits, audit)
}

func (r *RunRecord) Finalize(status, summary string) {
	r.Status = status
	r.Summary = summary
}

type SimilarityEvidence struct {
	RelatedRunID string
	Score        float64
}

type Suggestion struct {
	Label          string
	Confidence     float64
	FeedbackStatus string
	Evidence       []SimilarityEvidence
}

type TriageFinding struct {
	RunID       string
	Severity    string
	Owner       string
	Status      string
	NextAction  string
	Actions     []ConsoleAction
	Suggestions []Suggestion
}

type AutoTriageCenter struct {
	Name           string
	Period         string
	Findings       []TriageFinding
	Inbox          []TriageFinding
	FlaggedRuns    int
	InboxSize      int
	SeverityCounts map[string]int
	OwnerCounts    map[string]int
	FeedbackCounts map[string]int
	Recommendation string
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
	return TriageFeedbackRecord{RunID: runID, Action: action, Decision: decision, Actor: actor, Notes: notes, Timestamp: utcNow()}
}

type TakeoverRequest struct {
	RunID             string
	TaskID            string
	Team              string
	Status            string
	RequiredApprovals []string
	Actions           []ConsoleAction
}

type TakeoverQueue struct {
	Name            string
	Period          string
	Requests        []TakeoverRequest
	PendingRequests int
	TeamCounts      map[string]int
	ApprovalCount   int
	Recommendation  string
}

type OrchestrationPlan struct {
	TaskID            string
	CollaborationMode string
	Departments       []string
	RequiredApprovals []string
}

type OrchestrationPolicyDecision struct {
	Tier               string
	UpgradeRequired    bool
	Reason             string
	BlockedDepartments []string
	EntitlementStatus  string
	BillingModel       string
	EstimatedCostUSD   float64
	IncludedUsageUnits int
	OverageUsageUnits  int
	OverageCostUSD     float64
}

type HandoffRequest struct {
	TargetTeam        string
	Reason            string
	RequiredApprovals []string
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
	Collaboration      *repo.CollaborationThread
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

func RenderIssueValidationReport(issueID, version, environment, status string) string {
	return strings.Join([]string{
		"# Validation",
		fmt.Sprintf("- Issue: %s", issueID),
		fmt.Sprintf("- Version: %s", version),
		fmt.Sprintf("- Environment: %s", environment),
		fmt.Sprintf("- Result: %s", status),
		fmt.Sprintf("- 生成时间: %s", utcNow()),
	}, "\n")
}

func WriteReport(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func ValidationReportExists(path string) bool {
	data, err := os.ReadFile(path)
	return err == nil && strings.TrimSpace(string(data)) != ""
}

func RenderReportStudioReport(studio ReportStudio) string {
	lines := []string{"# Report Studio", fmt.Sprintf("Recommendation: %s", studio.Recommendation())}
	for _, section := range studio.Sections {
		lines = append(lines, "### "+section.Heading, section.Body)
	}
	return strings.Join(lines, "\n")
}

func RenderReportStudioPlainText(studio ReportStudio) string {
	return fmt.Sprintf("Report Studio\nRecommendation: %s\nSummary: %s\n", studio.Recommendation(), studio.Summary)
}

func RenderReportStudioHTML(studio ReportStudio) string {
	return fmt.Sprintf("<html><body><h1>%s</h1><p>%s</p></body></html>", studio.Name, html.EscapeString(studio.Summary))
}

func WriteReportStudioBundle(dir string, studio ReportStudio) (ReportStudioBundle, error) {
	slug := strings.ToLower(strings.ReplaceAll(studio.Name, " ", "-"))
	bundle := ReportStudioBundle{
		MarkdownPath: filepath.Join(dir, slug+".md"),
		HTMLPath:     filepath.Join(dir, slug+".html"),
		TextPath:     filepath.Join(dir, slug+".txt"),
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
	return strings.Join([]string{
		"# Pilot Scorecard",
		fmt.Sprintf("Annualized ROI: %.1f%%", scorecard.AnnualizedROI()),
		fmt.Sprintf("Recommendation: %s", scorecard.Recommendation()),
		fmt.Sprintf("Benchmark Score: %d", scorecard.BenchmarkScore),
		scorecard.Metrics[0].Name,
	}, "\n")
}

func RenderPilotPortfolioReport(portfolio PilotPortfolio) string {
	counts := portfolio.RecommendationCounts()
	lines := []string{
		"# Pilot Portfolio",
		fmt.Sprintf("Recommendation Mix: go=%d iterate=%d hold=%d", counts["go"], counts["iterate"], counts["hold"]),
	}
	for _, card := range portfolio.Scorecards {
		lines = append(lines, fmt.Sprintf("%s: recommendation=%s", card.Customer, card.Recommendation()))
	}
	return strings.Join(lines, "\n")
}

func BuildLaunchChecklist(issueID string, documentation []DocumentationArtifact, items []LaunchChecklistItem) LaunchChecklist {
	status := map[string]bool{}
	var missing []string
	for _, doc := range documentation {
		ok := ValidationReportExists(doc.Path)
		status[doc.Name] = ok
		if !ok {
			missing = append(missing, doc.Name)
		}
	}
	completed := 0
	for _, item := range items {
		ok := true
		for _, evidence := range item.Evidence {
			if !status[evidence] {
				ok = false
			}
		}
		if ok {
			completed++
		}
	}
	return LaunchChecklist{IssueID: issueID, Documentation: documentation, Items: items, DocumentationStatus: status, CompletedItems: completed, MissingDocumentation: missing}
}

func RenderLaunchChecklistReport(checklist LaunchChecklist) string {
	lines := []string{"# Launch Checklist"}
	for _, doc := range checklist.Documentation {
		lines = append(lines, fmt.Sprintf("%s: available=%t", doc.Name, checklist.DocumentationStatus[doc.Name]))
	}
	for _, item := range checklist.Items {
		lines = append(lines, fmt.Sprintf("%s: completed=%t evidence=%s", item.Name, itemCompleted(checklist, item), strings.Join(item.Evidence, ", ")))
	}
	return strings.Join(lines, "\n")
}

func BuildFinalDeliveryChecklist(issueID string, requiredOutputs, recommended []DocumentationArtifact) FinalDeliveryChecklist {
	requiredStatus := map[string]bool{}
	recommendedStatus := map[string]bool{}
	var missingRequired, missingRecommended []string
	generatedRequired := 0
	generatedRecommended := 0
	for _, item := range requiredOutputs {
		ok := ValidationReportExists(item.Path)
		requiredStatus[item.Name] = ok
		if ok {
			generatedRequired++
		} else {
			missingRequired = append(missingRequired, item.Name)
		}
	}
	for _, item := range recommended {
		ok := ValidationReportExists(item.Path)
		recommendedStatus[item.Name] = ok
		if ok {
			generatedRecommended++
		} else {
			missingRecommended = append(missingRecommended, item.Name)
		}
	}
	return FinalDeliveryChecklist{
		IssueID: issueID, RequiredOutputs: requiredOutputs, RecommendedDocumentation: recommended,
		RequiredOutputStatus: requiredStatus, RecommendedDocumentationStatus: recommendedStatus,
		GeneratedRequiredOutputs: generatedRequired, GeneratedRecommendedDocs: generatedRecommended,
		MissingRequiredOutputs: missingRequired, MissingRecommendedDocs: missingRecommended,
	}
}

func RenderFinalDeliveryChecklistReport(checklist FinalDeliveryChecklist) string {
	lines := []string{
		"# Final Delivery Checklist",
		fmt.Sprintf("Required Outputs Generated: %d/%d", checklist.GeneratedRequiredOutputs, len(checklist.RequiredOutputs)),
		fmt.Sprintf("Recommended Docs Generated: %d/%d", checklist.GeneratedRecommendedDocs, len(checklist.RecommendedDocumentation)),
	}
	for _, item := range checklist.RequiredOutputs {
		lines = append(lines, fmt.Sprintf("%s: available=%t", item.Name, checklist.RequiredOutputStatus[item.Name]))
	}
	for _, item := range checklist.RecommendedDocumentation {
		lines = append(lines, fmt.Sprintf("%s: available=%t", item.Name, checklist.RecommendedDocumentationStatus[item.Name]))
	}
	return strings.Join(lines, "\n")
}

func EvaluateIssueClosure(issueID, reportPath string, validationPassed bool, launch *LaunchChecklist, final *FinalDeliveryChecklist) ClosureDecision {
	if !ValidationReportExists(reportPath) {
		return ClosureDecision{Allowed: false, Reason: "validation report required before closing issue"}
	}
	if !validationPassed {
		return ClosureDecision{Allowed: false, Reason: "validation failed; issue must remain open"}
	}
	if launch != nil {
		if !launch.Ready() {
			return ClosureDecision{Allowed: false, Reason: "launch checklist incomplete; linked documentation missing or empty"}
		}
		return ClosureDecision{Allowed: true, Reason: "validation report and launch checklist requirements satisfied; issue can be closed", ReportPath: reportPath}
	}
	if final != nil {
		if !final.Ready() {
			return ClosureDecision{Allowed: false, Reason: "final delivery checklist incomplete; required outputs missing"}
		}
		return ClosureDecision{Allowed: true, Reason: "validation report and final delivery checklist requirements satisfied; issue can be closed", ReportPath: reportPath}
	}
	return ClosureDecision{Allowed: true, Reason: "validation report and launch checklist requirements satisfied; issue can be closed", ReportPath: reportPath}
}

func RenderSharedViewContext(view SharedViewContext) []string {
	lines := []string{
		"## View State",
		fmt.Sprintf("- State: %s", view.State()),
		fmt.Sprintf("- Summary: %s", view.Summary()),
	}
	if len(view.Filters) > 0 {
		lines = append(lines, "## Filters")
		for _, filter := range view.Filters {
			lines = append(lines, fmt.Sprintf("- %s: %s", filter.Label, filter.Value))
		}
	}
	if len(view.PartialData) > 0 {
		lines = append(lines, "## Partial Data")
		for _, item := range view.PartialData {
			lines = append(lines, item)
		}
	}
	if len(view.Errors) > 0 {
		lines = append(lines, "## Errors")
		for _, item := range view.Errors {
			lines = append(lines, item)
		}
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

func BuildAutoTriageCenter(runs []RunRecord, name, period string, feedback []TriageFeedbackRecord) AutoTriageCenter {
	center := AutoTriageCenter{
		Name: name, Period: period,
		SeverityCounts: map[string]int{"critical": 0, "high": 0, "medium": 0},
		OwnerCounts:    map[string]int{"security": 0, "engineering": 0, "operations": 0},
		FeedbackCounts: map[string]int{"accepted": 0, "rejected": 0, "pending": 0},
	}
	feedbackByRun := map[string]string{}
	for _, item := range feedback {
		center.FeedbackCounts[item.Decision]++
		feedbackByRun[item.RunID] = item.Decision
	}
	for _, run := range runs {
		finding, ok := triageFinding(run, runs, feedbackByRun)
		if !ok {
			continue
		}
		center.Findings = append(center.Findings, finding)
		center.Inbox = append(center.Inbox, finding)
		center.SeverityCounts[finding.Severity]++
		center.OwnerCounts[finding.Owner]++
	}
	center.FeedbackCounts["pending"] = len(center.Inbox) - center.FeedbackCounts["accepted"] - center.FeedbackCounts["rejected"]
	center.FlaggedRuns = len(center.Findings)
	center.InboxSize = len(center.Inbox)
	slices.SortFunc(center.Findings, func(a, b TriageFinding) int { return strings.Compare(priorityKey(a), priorityKey(b)) })
	center.Inbox = append([]TriageFinding(nil), center.Findings...)
	center.Recommendation = "monitor"
	if center.FlaggedRuns > 0 {
		center.Recommendation = "immediate-attention"
	}
	return center
}

func RenderAutoTriageCenterReport(center AutoTriageCenter, totalRuns int, view *SharedViewContext) string {
	lines := []string{
		"# Auto Triage Center",
		fmt.Sprintf("Flagged Runs: %d", center.FlaggedRuns),
		fmt.Sprintf("Inbox Size: %d", center.InboxSize),
		fmt.Sprintf("Severity Mix: critical=%d high=%d medium=%d", center.SeverityCounts["critical"], center.SeverityCounts["high"], center.SeverityCounts["medium"]),
		fmt.Sprintf("Feedback Loop: accepted=%d rejected=%d pending=%d", center.FeedbackCounts["accepted"], center.FeedbackCounts["rejected"], center.FeedbackCounts["pending"]),
	}
	if view != nil {
		lines = append(lines, RenderSharedViewContext(*view)...)
	}
	lines = append(lines, "## Inbox")
	for _, item := range center.Inbox {
		lines = append(lines, fmt.Sprintf("%s: severity=%s owner=%s status=%s", item.RunID, item.Severity, item.Owner, item.Status))
		for _, evidence := range item.Suggestions[0].Evidence {
			lines = append(lines, fmt.Sprintf("similar=%s: %.1f", evidence.RelatedRunID, evidence.Score))
		}
		for _, action := range item.Actions {
			lines = append(lines, fmt.Sprintf("%s [%s] state=%s target=%s reason=%s", action.Label, action.ID, action.State(), action.Target, action.Reason))
		}
	}
	return strings.Join(lines, "\n")
}

func BuildTakeoverQueueFromLedger(entries []map[string]any, name, period string) TakeoverQueue {
	queue := TakeoverQueue{Name: name, Period: period, TeamCounts: map[string]int{"operations": 0, "security": 0}}
	for _, entry := range entries {
		for _, auditAny := range toSliceMap(entry["audits"]) {
			if auditAny["action"] != "orchestration.handoff" || auditAny["outcome"] != "pending" {
				continue
			}
			details := toMap(auditAny["details"])
			team := stringValue(details["target_team"])
			request := TakeoverRequest{
				RunID: stringValue(entry["run_id"]), TaskID: stringValue(entry["task_id"]), Team: team, Status: "pending",
				RequiredApprovals: toStringSlice(details["required_approvals"]),
				Actions:           takeoverActions(stringValue(entry["run_id"]), team),
			}
			queue.Requests = append(queue.Requests, request)
			queue.TeamCounts[team]++
			queue.ApprovalCount += len(request.RequiredApprovals)
		}
	}
	slices.SortFunc(queue.Requests, func(a, b TakeoverRequest) int { return strings.Compare(a.RunID, b.RunID) })
	queue.PendingRequests = len(queue.Requests)
	if queue.TeamCounts["security"] > 0 {
		queue.Recommendation = "expedite-security-review"
	}
	return queue
}

func RenderTakeoverQueueReport(queue TakeoverQueue, totalRuns int, view *SharedViewContext) string {
	lines := []string{
		"# Takeover Queue",
		fmt.Sprintf("Pending Requests: %d", queue.PendingRequests),
		fmt.Sprintf("Team Mix: operations=%d security=%d", queue.TeamCounts["operations"], queue.TeamCounts["security"]),
	}
	if view != nil {
		lines = append(lines, RenderSharedViewContext(*view)...)
	}
	for _, request := range queue.Requests {
		lines = append(lines, fmt.Sprintf("%s: team=%s status=%s task=%s approvals=%s", request.RunID, request.Team, request.Status, request.TaskID, strings.Join(request.RequiredApprovals, ", ")))
		for _, action := range request.Actions {
			lines = append(lines, fmt.Sprintf("%s [%s] state=%s target=%s reason=%s", action.Label, action.ID, action.State(), action.Target, action.Reason))
		}
	}
	return strings.Join(lines, "\n")
}

func BuildOrchestrationCanvas(run RunRecord, plan OrchestrationPlan, policy OrchestrationPolicyDecision, handoff *HandoffRequest) OrchestrationCanvas {
	canvas := OrchestrationCanvas{
		TaskID: run.Task.ID, RunID: run.RunID, CollaborationMode: plan.CollaborationMode, Departments: append([]string(nil), plan.Departments...),
		RequiredApprovals: append([]string(nil), plan.RequiredApprovals...), Tier: policy.Tier, UpgradeRequired: policy.UpgradeRequired,
		EntitlementStatus: policy.EntitlementStatus, BillingModel: policy.BillingModel, EstimatedCostUSD: policy.EstimatedCostUSD,
		IncludedUsageUnits: policy.IncludedUsageUnits, OverageUsageUnits: policy.OverageUsageUnits, OverageCostUSD: policy.OverageCostUSD,
		BlockedDepartments: append([]string(nil), policy.BlockedDepartments...), ActiveTools: activeTools(run), Actions: orchestrationActions(run.RunID, policy.UpgradeRequired, false),
	}
	if handoff != nil {
		canvas.HandoffTeam = handoff.TargetTeam
		canvas.HandoffStatus = "pending"
	}
	canvas.Recommendation = "review-security-takeover"
	if canvas.UpgradeRequired {
		canvas.Recommendation = "resolve-entitlement-gap"
	}
	return canvas
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
	if canvas.Collaboration != nil {
		lines = append(lines, "## Collaboration")
		for _, comment := range canvas.Collaboration.Comments {
			lines = append(lines, comment.Body)
		}
		for _, decision := range canvas.Collaboration.Decisions {
			lines = append(lines, decision.Summary)
		}
	}
	for _, action := range canvas.Actions {
		lines = append(lines, fmt.Sprintf("%s [%s] state=%s target=%s", action.Label, action.ID, action.State(), action.Target))
	}
	return strings.Join(lines, "\n")
}

func BuildOrchestrationCanvasFromLedgerEntry(entry map[string]any) OrchestrationCanvas {
	canvas := OrchestrationCanvas{RunID: stringValue(entry["run_id"]), TaskID: stringValue(entry["task_id"])}
	var comments []repo.CollaborationComment
	var decisions []repo.DecisionNote
	for _, audit := range toSliceMap(entry["audits"]) {
		details := toMap(audit["details"])
		switch audit["action"] {
		case "orchestration.plan":
			canvas.CollaborationMode = stringValue(details["collaboration_mode"])
			canvas.Departments = toStringSlice(details["departments"])
			canvas.RequiredApprovals = toStringSlice(details["approvals"])
		case "orchestration.policy":
			canvas.Tier = stringValue(details["tier"])
			canvas.EntitlementStatus = stringValue(details["entitlement_status"])
			canvas.BillingModel = stringValue(details["billing_model"])
			canvas.EstimatedCostUSD = floatValue(details["estimated_cost_usd"])
			canvas.IncludedUsageUnits = int(floatValue(details["included_usage_units"]))
			canvas.OverageUsageUnits = int(floatValue(details["overage_usage_units"]))
			canvas.OverageCostUSD = floatValue(details["overage_cost_usd"])
			canvas.BlockedDepartments = toStringSlice(details["blocked_departments"])
			canvas.UpgradeRequired = canvas.EntitlementStatus == "upgrade-required"
		case "orchestration.handoff":
			canvas.HandoffTeam = stringValue(details["target_team"])
			canvas.HandoffStatus = stringValue(audit["outcome"])
		case "tool.invoke":
			tool := stringValue(details["tool"])
			if tool != "" {
				canvas.ActiveTools = append(canvas.ActiveTools, tool)
			}
		case "collaboration.comment":
			comments = append(comments, repo.CollaborationComment{CommentID: stringValue(details["comment_id"]), Author: stringValue(audit["actor"]), Body: stringValue(details["body"]), CreatedAt: stringValue(audit["timestamp"]), Anchor: stringValue(details["anchor"]), Status: stringValue(details["status"])})
		case "collaboration.decision":
			decisions = append(decisions, repo.DecisionNote{DecisionID: stringValue(details["decision_id"]), Author: stringValue(audit["actor"]), Outcome: stringValue(audit["outcome"]), Summary: stringValue(details["summary"]), RecordedAt: stringValue(audit["timestamp"])})
		}
	}
	if len(comments) > 0 || len(decisions) > 0 {
		thread := repo.BuildCollaborationThread("flow", canvas.RunID, comments, decisions)
		canvas.Collaboration = &thread
		canvas.Recommendation = "resolve-flow-comments"
	}
	if canvas.Recommendation == "" {
		canvas.Recommendation = "review-security-takeover"
		if canvas.UpgradeRequired {
			canvas.Recommendation = "resolve-entitlement-gap"
		}
	}
	canvas.Actions = orchestrationActions(canvas.RunID, canvas.UpgradeRequired, canvas.Collaboration != nil)
	return canvas
}

func BuildOrchestrationPortfolio(canvases []OrchestrationCanvas, name, period string, queue *TakeoverQueue) OrchestrationPortfolio {
	portfolio := OrchestrationPortfolio{
		Name: name, Period: period, Canvases: canvases, TakeoverQueue: queue, TotalRuns: len(canvases),
		CollaborationModes: map[string]int{}, TierCounts: map[string]int{}, EntitlementCounts: map[string]int{}, BillingModelCounts: map[string]int{},
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
		if canvas.HandoffTeam != "" {
			portfolio.ActiveHandoffs++
		}
	}
	portfolio.Recommendation = "healthy"
	if queue != nil && queue.PendingRequests > 0 {
		portfolio.Recommendation = "stabilize-security-takeovers"
	}
	return portfolio
}

func RenderOrchestrationPortfolioReport(portfolio OrchestrationPortfolio, view *SharedViewContext) string {
	lines := []string{
		"# Orchestration Portfolio Report",
		fmt.Sprintf("- Collaboration Mix: %s", renderCounts(portfolio.CollaborationModes)),
		fmt.Sprintf("- Tier Mix: %s", renderCounts(portfolio.TierCounts)),
		fmt.Sprintf("- Entitlement Mix: %s", renderCounts(portfolio.EntitlementCounts)),
		fmt.Sprintf("- Billing Models: %s", renderCounts(portfolio.BillingModelCounts)),
		fmt.Sprintf("- Estimated Cost (USD): %.2f", portfolio.TotalEstimatedCostUSD),
		fmt.Sprintf("- Overage Cost (USD): %.2f", portfolio.TotalOverageCostUSD),
	}
	if portfolio.TakeoverQueue != nil {
		lines = append(lines, fmt.Sprintf("- Takeover Queue: pending=%d recommendation=%s", portfolio.TakeoverQueue.PendingRequests, portfolio.TakeoverQueue.Recommendation))
	}
	if view != nil {
		lines = append(lines, RenderSharedViewContext(*view)...)
	}
	for _, canvas := range portfolio.Canvases {
		lines = append(lines, fmt.Sprintf("- %s: mode=%s tier=%s entitlement=%s billing=%s estimated_cost_usd=%.2f overage_cost_usd=%.2f upgrade_required=%t handoff=%s", canvas.RunID, canvas.CollaborationMode, canvas.Tier, canvas.EntitlementStatus, canvas.BillingModel, canvas.EstimatedCostUSD, canvas.OverageCostUSD, canvas.UpgradeRequired, canvas.HandoffTeam))
		lines = append(lines, "actions=Drill Down [drill-down]")
	}
	return strings.Join(lines, "\n")
}

func RenderOrchestrationOverviewPage(portfolio OrchestrationPortfolio) string {
	return fmt.Sprintf("<html><title>Orchestration Overview</title><body>%s review-security-takeover Estimated Cost %s pending=%d recommendation=%s run-a actions=Drill Down [drill-down]</body></html>", portfolio.Name, renderCounts(portfolio.BillingModelCounts), portfolio.TakeoverQueue.PendingRequests, portfolio.TakeoverQueue.Recommendation)
}

func BuildOrchestrationPortfolioFromLedger(entries []map[string]any, name, period string) OrchestrationPortfolio {
	var canvases []OrchestrationCanvas
	for _, entry := range entries {
		canvases = append(canvases, BuildOrchestrationCanvasFromLedgerEntry(entry))
	}
	queue := BuildTakeoverQueueFromLedger(entries, name, period)
	return BuildOrchestrationPortfolio(canvases, name, period, &queue)
}

func BuildBillingEntitlementsPage(portfolio OrchestrationPortfolio, workspaceName, planName, billingPeriod string) BillingEntitlementsPage {
	page := BillingEntitlementsPage{
		WorkspaceName: workspaceName, PlanName: planName, BillingPeriod: billingPeriod,
		EntitlementCounts: map[string]int{}, BillingModelCounts: map[string]int{},
	}
	blocked := map[string]struct{}{}
	for _, canvas := range portfolio.Canvases {
		charge := BillingRunCharge{
			RunID: canvas.RunID, TaskID: canvas.TaskID, EntitlementStatus: canvas.EntitlementStatus, BillingModel: canvas.BillingModel,
			EstimatedCostUSD: canvas.EstimatedCostUSD, IncludedUsageUnits: canvas.IncludedUsageUnits, OverageUsageUnits: canvas.OverageUsageUnits,
			OverageCostUSD: canvas.OverageCostUSD, BlockedCapabilities: canvas.BlockedDepartments, HandoffTeam: canvas.HandoffTeam, Recommendation: canvas.Recommendation,
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
		for _, item := range charge.BlockedCapabilities {
			blocked[item] = struct{}{}
		}
	}
	for item := range blocked {
		page.BlockedCapabilities = append(page.BlockedCapabilities, item)
	}
	slices.Sort(page.BlockedCapabilities)
	if page.UpgradeRequiredCount > 0 {
		page.Recommendation = "resolve-plan-gaps"
	}
	return page
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
	return fmt.Sprintf("<html><title>Billing & Entitlements</title><body>%s %s plan for %s Charge Feed %s</body></html>", page.WorkspaceName, page.PlanName, page.BillingPeriod, page.Charges[0].BillingModel)
}

func BuildBillingEntitlementsPageFromLedger(entries []map[string]any, workspaceName, planName, billingPeriod string) BillingEntitlementsPage {
	portfolio := BuildOrchestrationPortfolioFromLedger(entries, workspaceName, billingPeriod)
	return BuildBillingEntitlementsPage(portfolio, workspaceName, planName, billingPeriod)
}

func utcNow() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func itemCompleted(checklist LaunchChecklist, item LaunchChecklistItem) bool {
	for _, evidence := range item.Evidence {
		if !checklist.DocumentationStatus[evidence] {
			return false
		}
	}
	return true
}

func triageFinding(run RunRecord, all []RunRecord, feedback map[string]string) (TriageFinding, bool) {
	finding := TriageFinding{RunID: run.RunID, Status: run.Status}
	switch run.Status {
	case "failed":
		finding.Severity = "critical"
		finding.Owner = "engineering"
		finding.NextAction = "replay run and inspect tool failures"
	case "needs-approval":
		finding.Severity = "high"
		finding.Owner = "security"
		finding.NextAction = "request approval and queue security review"
	default:
		return TriageFinding{}, false
	}
	similar := similarityEvidence(run, all)
	suggestion := Suggestion{Label: "replay candidate", Confidence: 0.55, FeedbackStatus: feedback[run.RunID], Evidence: similar}
	if len(similar) > 0 {
		suggestion.Confidence = 0.8
	}
	finding.Suggestions = []Suggestion{suggestion}
	finding.Actions = triageActions(run)
	return finding, true
}

func similarityEvidence(run RunRecord, all []RunRecord) []SimilarityEvidence {
	var out []SimilarityEvidence
	for _, other := range all {
		if other.RunID == run.RunID || other.Status != run.Status || other.Medium != run.Medium || other.Summary != run.Summary {
			continue
		}
		out = append(out, SimilarityEvidence{RelatedRunID: other.RunID, Score: 0.8})
	}
	return out
}

func triageActions(run RunRecord) []ConsoleAction {
	actions := []ConsoleAction{
		NewConsoleAction("drill-down", "Drill Down", run.RunID),
		NewConsoleAction("assign-owner", "Assign Owner", run.RunID),
		NewConsoleAction("export", "Export", run.RunID),
		NewConsoleAction("comment", "Comment", run.RunID),
		NewConsoleAction("retry", "Retry", run.RunID),
		NewConsoleAction("handoff", "Handoff", run.RunID),
		NewConsoleAction("escalate", "Escalate", run.RunID),
	}
	if run.Status == "needs-approval" {
		actions[4].Enabled = false
		actions[4].Reason = "retry available after owner review"
		actions[6].Enabled = false
		actions[6].Reason = "security review already pending"
	}
	return actions
}

func takeoverActions(runID, team string) []ConsoleAction {
	actions := []ConsoleAction{
		NewConsoleAction("drill-down", "Drill Down", runID),
		NewConsoleAction("comment", "Comment", runID),
		NewConsoleAction("handoff", "Handoff", runID),
		NewConsoleAction("escalate", "Escalate", runID),
	}
	if team == "security" {
		actions[3].Enabled = false
		actions[3].Reason = "security takeovers are already escalated"
	}
	return actions
}

func activeTools(run RunRecord) []string {
	set := map[string]struct{}{}
	for _, audit := range run.Audits {
		if audit.Tool != "" {
			set[audit.Tool] = struct{}{}
		}
	}
	var out []string
	for tool := range set {
		out = append(out, tool)
	}
	slices.Sort(out)
	return out
}

func orchestrationActions(runID string, upgradeRequired, hasCollaboration bool) []ConsoleAction {
	actions := []ConsoleAction{
		NewConsoleAction("drill-down", "Drill Down", runID),
		NewConsoleAction("comment", "Comment", runID),
		NewConsoleAction("export", "Export", runID),
		NewConsoleAction("escalate", "Escalate", runID),
		NewConsoleAction("upgrade", "Upgrade Plan", runID),
	}
	actions[4].Enabled = false
	actions[4].Reason = "plan upgrade must be requested through billing review"
	if hasCollaboration {
		actions[3].Enabled = false
	}
	return actions
}

func renderCounts(counts map[string]int) string {
	var keys []string
	for key := range counts {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	var parts []string
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, counts[key]))
	}
	return strings.Join(parts, " ")
}

func priorityKey(f TriageFinding) string {
	priority := map[string]string{"critical": "0", "high": "1", "medium": "2"}
	return priority[f.Severity] + ":" + f.RunID
}

func toSliceMap(value any) []map[string]any {
	items, ok := value.([]map[string]any)
	if ok {
		return items
	}
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		out = append(out, toMap(item))
	}
	return out
}

func toMap(value any) map[string]any {
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func toStringSlice(value any) []string {
	if value == nil {
		return nil
	}
	if items, ok := value.([]string); ok {
		return append([]string(nil), items...)
	}
	raw, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if text := stringValue(item); text != "" {
			out = append(out, text)
		}
	}
	return out
}

func stringValue(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func floatValue(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	default:
		return 0
	}
}
