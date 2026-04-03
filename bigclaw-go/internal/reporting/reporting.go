package reporting

import (
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/regression"
	"github.com/pmezard/go-difflib/difflib"
)

type Summary struct {
	TotalRuns          int   `json:"total_runs"`
	CompletedRuns      int   `json:"completed_runs"`
	BlockedRuns        int   `json:"blocked_runs"`
	HighRiskRuns       int   `json:"high_risk_runs"`
	RegressionFindings int   `json:"regression_findings"`
	HumanInterventions int   `json:"human_interventions"`
	BudgetCentsTotal   int64 `json:"budget_cents_total"`
	PremiumRuns        int   `json:"premium_runs"`
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
		zero := 0.0
		return &zero
	}
	value := math.Round((s.ImplementationCost/s.MonthlyNetValue())*10) / 10
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
	return math.Round((total/float64(len(p.Scorecards)))*10) / 10
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
	IssueID    string `json:"issue_id"`
	Allowed    bool   `json:"allowed"`
	Reason     string `json:"reason"`
	ReportPath string `json:"report_path,omitempty"`
}

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
	for _, name := range item.Evidence {
		if !status[name] {
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
	missing := []string{}
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
	missing := []string{}
	for _, artifact := range c.RequiredOutputs {
		if !artifact.Available() {
			missing = append(missing, artifact.Name)
		}
	}
	return missing
}

func (c FinalDeliveryChecklist) MissingRecommendedDocumentation() []string {
	missing := []string{}
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
	if slug := slugifyReportStudioName(s.Name); slug != "" {
		return slug
	}
	return "report-studio"
}

type ReportStudioArtifacts struct {
	RootDir      string `json:"root_dir"`
	MarkdownPath string `json:"markdown_path"`
	HTMLPath     string `json:"html_path"`
	TextPath     string `json:"text_path"`
}

type GitSyncTelemetry struct {
	Status          string   `json:"status"`
	FailureCategory string   `json:"failure_category,omitempty"`
	Summary         string   `json:"summary,omitempty"`
	Branch          string   `json:"branch,omitempty"`
	Remote          string   `json:"remote"`
	RemoteRef       string   `json:"remote_ref,omitempty"`
	AheadBy         int      `json:"ahead_by"`
	BehindBy        int      `json:"behind_by"`
	DirtyPaths      []string `json:"dirty_paths,omitempty"`
	AuthTarget      string   `json:"auth_target,omitempty"`
	Timestamp       string   `json:"timestamp,omitempty"`
}

type PullRequestFreshness struct {
	PRNumber           *int   `json:"pr_number,omitempty"`
	PRURL              string `json:"pr_url,omitempty"`
	BranchState        string `json:"branch_state"`
	BodyState          string `json:"body_state"`
	BranchHeadSHA      string `json:"branch_head_sha,omitempty"`
	PRHeadSHA          string `json:"pr_head_sha,omitempty"`
	ExpectedBodyDigest string `json:"expected_body_digest,omitempty"`
	ActualBodyDigest   string `json:"actual_body_digest,omitempty"`
	CheckedAt          string `json:"checked_at,omitempty"`
}

type RepoSyncAudit struct {
	Sync        GitSyncTelemetry     `json:"sync"`
	PullRequest PullRequestFreshness `json:"pull_request"`
}

func (a RepoSyncAudit) Summary() string {
	parts := []string{fmt.Sprintf("sync=%s", a.Sync.Status)}
	if a.Sync.FailureCategory != "" {
		parts = append(parts, fmt.Sprintf("failure=%s", a.Sync.FailureCategory))
	}
	parts = append(parts, fmt.Sprintf("pr-branch=%s", a.PullRequest.BranchState))
	parts = append(parts, fmt.Sprintf("pr-body=%s", a.PullRequest.BodyState))
	return strings.Join(parts, ", ")
}

type TeamBreakdown struct {
	Key                string `json:"key"`
	TotalRuns          int    `json:"total_runs"`
	CompletedRuns      int    `json:"completed_runs"`
	BlockedRuns        int    `json:"blocked_runs"`
	BudgetCentsTotal   int64  `json:"budget_cents_total"`
	HumanInterventions int    `json:"human_interventions"`
}

type Weekly struct {
	WeekStart     time.Time       `json:"week_start"`
	WeekEnd       time.Time       `json:"week_end"`
	Summary       Summary         `json:"summary"`
	TeamBreakdown []TeamBreakdown `json:"team_breakdown"`
	Highlights    []string        `json:"highlights"`
	Actions       []string        `json:"actions"`
	Markdown      string          `json:"markdown"`
}

type OperationsMetricDefinition struct {
	MetricID     string   `json:"metric_id"`
	Label        string   `json:"label"`
	Unit         string   `json:"unit"`
	Direction    string   `json:"direction"`
	Formula      string   `json:"formula"`
	Description  string   `json:"description"`
	SourceFields []string `json:"source_fields,omitempty"`
}

type OperationsMetricValue struct {
	MetricID     string   `json:"metric_id"`
	Label        string   `json:"label"`
	Value        float64  `json:"value"`
	DisplayValue string   `json:"display_value"`
	Numerator    float64  `json:"numerator"`
	Denominator  float64  `json:"denominator"`
	Unit         string   `json:"unit"`
	Evidence     []string `json:"evidence,omitempty"`
}

type OperationsMetricSpec struct {
	Name         string                       `json:"name"`
	GeneratedAt  time.Time                    `json:"generated_at"`
	PeriodStart  time.Time                    `json:"period_start"`
	PeriodEnd    time.Time                    `json:"period_end"`
	TimezoneName string                       `json:"timezone_name"`
	Definitions  []OperationsMetricDefinition `json:"definitions,omitempty"`
	Values       []OperationsMetricValue      `json:"values,omitempty"`
}

type WeeklyArtifacts struct {
	RootDir              string `json:"root_dir"`
	WeeklyReportPath     string `json:"weekly_report_path"`
	DashboardPath        string `json:"dashboard_path"`
	MetricSpecPath       string `json:"metric_spec_path,omitempty"`
	RegressionCenterPath string `json:"regression_center_path,omitempty"`
	QueueControlPath     string `json:"queue_control_path,omitempty"`
	VersionCenterPath    string `json:"version_center_path,omitempty"`
}

type RepoCollaborationMetrics struct {
	RepoLinkCoverage        float64 `json:"repo_link_coverage"`
	AcceptedCommitRate      float64 `json:"accepted_commit_rate"`
	DiscussionDensity       float64 `json:"discussion_density"`
	AcceptedLineageDepthAvg float64 `json:"accepted_lineage_depth_avg"`
}

type ConsoleAction struct {
	ActionID string `json:"action_id"`
	Label    string `json:"label"`
	Target   string `json:"target"`
	Enabled  bool   `json:"enabled"`
	Reason   string `json:"reason,omitempty"`
}

func (a ConsoleAction) State() string {
	if a.Enabled {
		return "enabled"
	}
	return "disabled"
}

type QueueControlCenter struct {
	QueueDepth          int                        `json:"queue_depth"`
	QueuedByPriority    map[string]int             `json:"queued_by_priority,omitempty"`
	QueuedByRisk        map[string]int             `json:"queued_by_risk,omitempty"`
	ExecutionMedia      map[string]int             `json:"execution_media,omitempty"`
	WaitingApprovalRuns int                        `json:"waiting_approval_runs"`
	BlockedTasks        []string                   `json:"blocked_tasks,omitempty"`
	QueuedTasks         []string                   `json:"queued_tasks,omitempty"`
	Actions             map[string][]ConsoleAction `json:"actions,omitempty"`
}

type TakeoverRequest struct {
	RunID             string          `json:"run_id"`
	TaskID            string          `json:"task_id"`
	Source            string          `json:"source"`
	TargetTeam        string          `json:"target_team"`
	Status            string          `json:"status"`
	Reason            string          `json:"reason"`
	RequiredApprovals []string        `json:"required_approvals,omitempty"`
	Actions           []ConsoleAction `json:"actions,omitempty"`
}

type TakeoverQueue struct {
	Name     string            `json:"name"`
	Period   string            `json:"period"`
	Requests []TakeoverRequest `json:"requests,omitempty"`
}

func (q TakeoverQueue) PendingRequests() int {
	return len(q.Requests)
}

func (q TakeoverQueue) TeamCounts() map[string]int {
	counts := map[string]int{}
	for _, request := range q.Requests {
		counts[request.TargetTeam]++
	}
	return counts
}

func (q TakeoverQueue) ApprovalCount() int {
	total := 0
	for _, request := range q.Requests {
		total += len(request.RequiredApprovals)
	}
	return total
}

func (q TakeoverQueue) Recommendation() string {
	for _, request := range q.Requests {
		if request.TargetTeam == "security" {
			return "expedite-security-review"
		}
	}
	if len(q.Requests) > 0 {
		return "staff-takeover-queue"
	}
	return "monitor"
}

type OrchestrationCanvas struct {
	TaskID             string               `json:"task_id"`
	RunID              string               `json:"run_id"`
	CollaborationMode  string               `json:"collaboration_mode"`
	Departments        []string             `json:"departments,omitempty"`
	RequiredApprovals  []string             `json:"required_approvals,omitempty"`
	Tier               string               `json:"tier"`
	UpgradeRequired    bool                 `json:"upgrade_required"`
	BlockedDepartments []string             `json:"blocked_departments,omitempty"`
	HandoffTeam        string               `json:"handoff_team"`
	HandoffStatus      string               `json:"handoff_status"`
	HandoffReason      string               `json:"handoff_reason,omitempty"`
	ActiveTools        []string             `json:"active_tools,omitempty"`
	EntitlementStatus  string               `json:"entitlement_status"`
	BillingModel       string               `json:"billing_model"`
	EstimatedCostUSD   float64              `json:"estimated_cost_usd"`
	IncludedUsageUnits int                  `json:"included_usage_units"`
	OverageUsageUnits  int                  `json:"overage_usage_units"`
	OverageCostUSD     float64              `json:"overage_cost_usd"`
	Actions            []ConsoleAction      `json:"actions,omitempty"`
	Collaboration      *CollaborationThread `json:"collaboration,omitempty"`
}

func (c OrchestrationCanvas) Recommendation() string {
	if c.Collaboration != nil && c.Collaboration.OpenCommentCount() > 0 {
		return "resolve-flow-comments"
	}
	if c.HandoffTeam == "security" {
		return "review-security-takeover"
	}
	if c.UpgradeRequired {
		return "resolve-entitlement-gap"
	}
	if c.OverageCostUSD > 0 {
		return "review-billing-overage"
	}
	if len(c.Departments) > 1 {
		return "continue-cross-team-execution"
	}
	return "monitor"
}

type OrchestrationPortfolio struct {
	Name          string                `json:"name"`
	Period        string                `json:"period"`
	Canvases      []OrchestrationCanvas `json:"canvases,omitempty"`
	TakeoverQueue *TakeoverQueue        `json:"takeover_queue,omitempty"`
}

func (p OrchestrationPortfolio) TotalRuns() int {
	return len(p.Canvases)
}

func (p OrchestrationPortfolio) CollaborationModes() map[string]int {
	counts := map[string]int{}
	for _, canvas := range p.Canvases {
		counts[canvas.CollaborationMode]++
	}
	return counts
}

func (p OrchestrationPortfolio) TierCounts() map[string]int {
	counts := map[string]int{}
	for _, canvas := range p.Canvases {
		counts[canvas.Tier]++
	}
	return counts
}

func (p OrchestrationPortfolio) UpgradeRequiredCount() int {
	total := 0
	for _, canvas := range p.Canvases {
		if canvas.UpgradeRequired {
			total++
		}
	}
	return total
}

func (p OrchestrationPortfolio) ActiveHandoffs() int {
	total := 0
	for _, canvas := range p.Canvases {
		if canvas.HandoffTeam != "none" {
			total++
		}
	}
	return total
}

func (p OrchestrationPortfolio) EntitlementCounts() map[string]int {
	counts := map[string]int{}
	for _, canvas := range p.Canvases {
		counts[canvas.EntitlementStatus]++
	}
	return counts
}

func (p OrchestrationPortfolio) BillingModelCounts() map[string]int {
	counts := map[string]int{}
	for _, canvas := range p.Canvases {
		counts[canvas.BillingModel]++
	}
	return counts
}

func (p OrchestrationPortfolio) TotalEstimatedCostUSD() float64 {
	total := 0.0
	for _, canvas := range p.Canvases {
		total += canvas.EstimatedCostUSD
	}
	return math.Round(total*100) / 100
}

func (p OrchestrationPortfolio) TotalOverageCostUSD() float64 {
	total := 0.0
	for _, canvas := range p.Canvases {
		total += canvas.OverageCostUSD
	}
	return math.Round(total*100) / 100
}

func (p OrchestrationPortfolio) Recommendation() string {
	if p.TakeoverQueue != nil && p.TakeoverQueue.Recommendation() == "expedite-security-review" {
		return "stabilize-security-takeovers"
	}
	if p.UpgradeRequiredCount() > 0 {
		return "close-entitlement-gaps"
	}
	if p.ActiveHandoffs() > 0 {
		return "manage-cross-team-flow"
	}
	return "monitor"
}

type BillingRunCharge struct {
	RunID               string   `json:"run_id"`
	TaskID              string   `json:"task_id"`
	BillingModel        string   `json:"billing_model"`
	EntitlementStatus   string   `json:"entitlement_status"`
	EstimatedCostUSD    float64  `json:"estimated_cost_usd"`
	IncludedUsageUnits  int      `json:"included_usage_units"`
	OverageUsageUnits   int      `json:"overage_usage_units"`
	OverageCostUSD      float64  `json:"overage_cost_usd"`
	BlockedCapabilities []string `json:"blocked_capabilities,omitempty"`
	HandoffTeam         string   `json:"handoff_team"`
	Recommendation      string   `json:"recommendation"`
}

type BillingEntitlementsPage struct {
	WorkspaceName string             `json:"workspace_name"`
	PlanName      string             `json:"plan_name"`
	BillingPeriod string             `json:"billing_period"`
	Charges       []BillingRunCharge `json:"charges,omitempty"`
}

func (p BillingEntitlementsPage) RunCount() int {
	return len(p.Charges)
}

func (p BillingEntitlementsPage) TotalEstimatedCostUSD() float64 {
	total := 0.0
	for _, charge := range p.Charges {
		total += charge.EstimatedCostUSD
	}
	return math.Round(total*100) / 100
}

func (p BillingEntitlementsPage) TotalIncludedUsageUnits() int {
	total := 0
	for _, charge := range p.Charges {
		total += charge.IncludedUsageUnits
	}
	return total
}

func (p BillingEntitlementsPage) TotalOverageUsageUnits() int {
	total := 0
	for _, charge := range p.Charges {
		total += charge.OverageUsageUnits
	}
	return total
}

func (p BillingEntitlementsPage) TotalOverageCostUSD() float64 {
	total := 0.0
	for _, charge := range p.Charges {
		total += charge.OverageCostUSD
	}
	return math.Round(total*100) / 100
}

func (p BillingEntitlementsPage) UpgradeRequiredCount() int {
	total := 0
	for _, charge := range p.Charges {
		if charge.EntitlementStatus == "upgrade-required" {
			total++
		}
	}
	return total
}

func (p BillingEntitlementsPage) BillingModelCounts() map[string]int {
	counts := map[string]int{}
	for _, charge := range p.Charges {
		counts[charge.BillingModel]++
	}
	return counts
}

func (p BillingEntitlementsPage) EntitlementCounts() map[string]int {
	counts := map[string]int{}
	for _, charge := range p.Charges {
		counts[charge.EntitlementStatus]++
	}
	return counts
}

func (p BillingEntitlementsPage) BlockedCapabilities() []string {
	var capabilities []string
	for _, charge := range p.Charges {
		for _, capability := range charge.BlockedCapabilities {
			if !containsString(capabilities, capability) {
				capabilities = append(capabilities, capability)
			}
		}
	}
	return capabilities
}

func (p BillingEntitlementsPage) Recommendation() string {
	if p.UpgradeRequiredCount() > 0 {
		return "resolve-plan-gaps"
	}
	if p.TotalOverageCostUSD() > 0 {
		return "optimize-billed-usage"
	}
	for _, charge := range p.Charges {
		if charge.HandoffTeam != "none" {
			return "monitor-shared-capacity"
		}
	}
	return "healthy"
}

type TriageCluster struct {
	Reason   string   `json:"reason"`
	RunIDs   []string `json:"run_ids,omitempty"`
	TaskIDs  []string `json:"task_ids,omitempty"`
	Statuses []string `json:"statuses,omitempty"`
}

func (c TriageCluster) Occurrences() int {
	return len(c.RunIDs)
}

type OperationsSnapshot struct {
	TotalRuns           int             `json:"total_runs"`
	StatusCounts        map[string]int  `json:"status_counts,omitempty"`
	SuccessRate         float64         `json:"success_rate"`
	ApprovalQueueDepth  int             `json:"approval_queue_depth"`
	SLATargetMinutes    int             `json:"sla_target_minutes"`
	SLABreachCount      int             `json:"sla_breach_count"`
	AverageCycleMinutes float64         `json:"average_cycle_minutes"`
	TopBlockers         []TriageCluster `json:"top_blockers,omitempty"`
}

type BenchmarkCaseResult struct {
	CaseID string `json:"case_id"`
	Score  int    `json:"score"`
	Passed bool   `json:"passed"`
}

type BenchmarkComparison struct {
	CaseID        string `json:"case_id"`
	BaselineScore int    `json:"baseline_score"`
	CurrentScore  int    `json:"current_score"`
	Delta         int    `json:"delta"`
	Changed       bool   `json:"changed"`
}

type BenchmarkSuiteResult struct {
	Results []BenchmarkCaseResult `json:"results,omitempty"`
	Version string                `json:"version"`
}

func (s BenchmarkSuiteResult) Compare(baseline BenchmarkSuiteResult) []BenchmarkComparison {
	baselineByCase := make(map[string]BenchmarkCaseResult, len(baseline.Results))
	for _, result := range baseline.Results {
		baselineByCase[result.CaseID] = result
	}
	comparisons := make([]BenchmarkComparison, 0, len(s.Results))
	for _, result := range s.Results {
		baselineScore := 0
		if baselineResult, ok := baselineByCase[result.CaseID]; ok {
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

type BenchmarkRegressionFinding struct {
	CaseID        string `json:"case_id"`
	BaselineScore int    `json:"baseline_score"`
	CurrentScore  int    `json:"current_score"`
	Delta         int    `json:"delta"`
	Severity      string `json:"severity"`
	Summary       string `json:"summary"`
}

type BenchmarkRegressionCenter struct {
	Name            string                       `json:"name"`
	BaselineVersion string                       `json:"baseline_version"`
	CurrentVersion  string                       `json:"current_version"`
	Regressions     []BenchmarkRegressionFinding `json:"regressions,omitempty"`
	ImprovedCases   []string                     `json:"improved_cases,omitempty"`
	UnchangedCases  []string                     `json:"unchanged_cases,omitempty"`
}

func (c BenchmarkRegressionCenter) RegressionCount() int {
	return len(c.Regressions)
}

type SharedViewFilter struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type CollaborationComment struct {
	CommentID string   `json:"comment_id"`
	Author    string   `json:"author"`
	Body      string   `json:"body"`
	CreatedAt string   `json:"created_at,omitempty"`
	Mentions  []string `json:"mentions,omitempty"`
	Anchor    string   `json:"anchor,omitempty"`
	Status    string   `json:"status,omitempty"`
}

type DecisionNote struct {
	DecisionID        string   `json:"decision_id"`
	Author            string   `json:"author"`
	Outcome           string   `json:"outcome"`
	Summary           string   `json:"summary"`
	RecordedAt        string   `json:"recorded_at,omitempty"`
	Mentions          []string `json:"mentions,omitempty"`
	RelatedCommentIDs []string `json:"related_comment_ids,omitempty"`
	FollowUp          string   `json:"follow_up,omitempty"`
}

type CollaborationThread struct {
	Surface   string                 `json:"surface"`
	TargetID  string                 `json:"target_id"`
	Comments  []CollaborationComment `json:"comments,omitempty"`
	Decisions []DecisionNote         `json:"decisions,omitempty"`
}

func (t CollaborationThread) ParticipantCount() int {
	participants := map[string]struct{}{}
	for _, comment := range t.Comments {
		if comment.Author != "" {
			participants[comment.Author] = struct{}{}
		}
	}
	for _, decision := range t.Decisions {
		if decision.Author != "" {
			participants[decision.Author] = struct{}{}
		}
	}
	return len(participants)
}

func (t CollaborationThread) MentionCount() int {
	total := 0
	for _, comment := range t.Comments {
		total += len(comment.Mentions)
	}
	for _, decision := range t.Decisions {
		total += len(decision.Mentions)
	}
	return total
}

func (t CollaborationThread) OpenCommentCount() int {
	total := 0
	for _, comment := range t.Comments {
		if comment.Status != "resolved" {
			total++
		}
	}
	return total
}

func (t CollaborationThread) Recommendation() string {
	if len(t.Decisions) > 0 {
		return "share-latest-decision"
	}
	if t.OpenCommentCount() > 0 {
		return "resolve-open-comments"
	}
	if len(t.Comments) > 0 {
		return "monitor-collaboration"
	}
	return "no-collaboration-recorded"
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
	return TriageFeedbackRecord{
		RunID:     runID,
		Action:    action,
		Decision:  decision,
		Actor:     actor,
		Notes:     notes,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

type TriageFinding struct {
	RunID      string          `json:"run_id"`
	TaskID     string          `json:"task_id"`
	Source     string          `json:"source"`
	Severity   string          `json:"severity"`
	Owner      string          `json:"owner"`
	Status     string          `json:"status"`
	Reason     string          `json:"reason"`
	NextAction string          `json:"next_action"`
	Actions    []ConsoleAction `json:"actions,omitempty"`
}

type TriageSimilarityEvidence struct {
	RelatedRunID  string  `json:"related_run_id"`
	RelatedTaskID string  `json:"related_task_id"`
	Score         float64 `json:"score"`
	Reason        string  `json:"reason"`
}

type TriageSuggestion struct {
	Label          string                     `json:"label"`
	Action         string                     `json:"action"`
	Owner          string                     `json:"owner"`
	Confidence     float64                    `json:"confidence"`
	Evidence       []TriageSimilarityEvidence `json:"evidence,omitempty"`
	FeedbackStatus string                     `json:"feedback_status"`
}

type TriageInboxItem struct {
	RunID       string             `json:"run_id"`
	TaskID      string             `json:"task_id"`
	Source      string             `json:"source"`
	Status      string             `json:"status"`
	Severity    string             `json:"severity"`
	Owner       string             `json:"owner"`
	Summary     string             `json:"summary"`
	SubmittedAt string             `json:"submitted_at"`
	Suggestions []TriageSuggestion `json:"suggestions,omitempty"`
}

type AutoTriageRunTrace struct {
	Span   string `json:"span"`
	Status string `json:"status"`
}

type AutoTriageRunAudit struct {
	Action  string         `json:"action"`
	Outcome string         `json:"outcome"`
	Details map[string]any `json:"details,omitempty"`
}

type AutoTriageArtifact struct {
	Kind string `json:"kind"`
}

type AutoTriageRun struct {
	RunID     string               `json:"run_id"`
	TaskID    string               `json:"task_id"`
	Source    string               `json:"source"`
	Title     string               `json:"title"`
	Summary   string               `json:"summary"`
	Medium    string               `json:"medium"`
	Status    string               `json:"status"`
	StartedAt string               `json:"started_at,omitempty"`
	EndedAt   string               `json:"ended_at,omitempty"`
	Traces    []AutoTriageRunTrace `json:"traces,omitempty"`
	Audits    []AutoTriageRunAudit `json:"audits,omitempty"`
	Artifacts []AutoTriageArtifact `json:"artifacts,omitempty"`
}

type AutoTriageCenter struct {
	Name     string                 `json:"name"`
	Period   string                 `json:"period"`
	Findings []TriageFinding        `json:"findings,omitempty"`
	Inbox    []TriageInboxItem      `json:"inbox,omitempty"`
	Feedback []TriageFeedbackRecord `json:"feedback,omitempty"`
}

func (c AutoTriageCenter) FlaggedRuns() int { return len(c.Findings) }

func (c AutoTriageCenter) InboxSize() int { return len(c.Inbox) }

func (c AutoTriageCenter) SeverityCounts() map[string]int {
	counts := map[string]int{"critical": 0, "high": 0, "medium": 0}
	for _, finding := range c.Findings {
		counts[finding.Severity]++
	}
	return counts
}

func (c AutoTriageCenter) OwnerCounts() map[string]int {
	counts := map[string]int{"security": 0, "engineering": 0, "operations": 0}
	for _, finding := range c.Findings {
		counts[finding.Owner]++
	}
	return counts
}

func (c AutoTriageCenter) FeedbackCounts() map[string]int {
	counts := map[string]int{"accepted": 0, "rejected": 0, "pending": 0}
	for _, record := range c.Feedback {
		counts[record.Decision]++
	}
	pending := 0
	for _, item := range c.Inbox {
		for _, suggestion := range item.Suggestions {
			if suggestion.FeedbackStatus == "pending" {
				pending++
			}
		}
	}
	counts["pending"] = pending
	return counts
}

func (c AutoTriageCenter) Recommendation() string {
	severity := c.SeverityCounts()
	feedback := c.FeedbackCounts()
	if severity["critical"] > 0 {
		return "immediate-attention"
	}
	if feedback["rejected"] > feedback["accepted"] {
		return "retune-suggestions"
	}
	if severity["high"] > 0 {
		return "review-queue"
	}
	return "monitor"
}

type SharedViewContext struct {
	Filters       []SharedViewFilter   `json:"filters,omitempty"`
	ResultCount   *int                 `json:"result_count,omitempty"`
	Loading       bool                 `json:"loading,omitempty"`
	Errors        []string             `json:"errors,omitempty"`
	PartialData   []string             `json:"partial_data,omitempty"`
	EmptyMessage  string               `json:"empty_message,omitempty"`
	LastUpdated   string               `json:"last_updated,omitempty"`
	Collaboration *CollaborationThread `json:"collaboration,omitempty"`
}

func (v SharedViewContext) State() string {
	switch {
	case v.Loading:
		return "loading"
	case len(v.Errors) > 0 && (v.ResultCount == nil || *v.ResultCount == 0):
		return "error"
	case v.ResultCount != nil && *v.ResultCount == 0 && len(v.PartialData) == 0:
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

type BenchmarkReplayRecord struct {
	TaskID   string `json:"task_id"`
	RunID    string `json:"run_id"`
	Medium   string `json:"medium"`
	Approved bool   `json:"approved"`
	Status   string `json:"status"`
}

type BenchmarkReplayOutcome struct {
	Matched      bool                  `json:"matched"`
	ReplayRecord BenchmarkReplayRecord `json:"replay_record"`
	Mismatches   []string              `json:"mismatches,omitempty"`
	ReportPath   string                `json:"report_path,omitempty"`
}

type BenchmarkCriterion struct {
	Name   string `json:"name"`
	Weight int    `json:"weight"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type BenchmarkRunIndexRecord struct {
	TaskID     string `json:"task_id"`
	Medium     string `json:"medium"`
	Status     string `json:"status"`
	ReportPath string `json:"report_path,omitempty"`
}

type BenchmarkCase struct {
	CaseID           string      `json:"case_id"`
	Task             domain.Task `json:"task"`
	ExpectedMedium   string      `json:"expected_medium,omitempty"`
	ExpectedApproved *bool       `json:"expected_approved,omitempty"`
	ExpectedStatus   string      `json:"expected_status,omitempty"`
	RequireReport    bool        `json:"require_report,omitempty"`
}

type BenchmarkResultRecord struct {
	Medium     string `json:"medium"`
	Approved   bool   `json:"approved"`
	Status     string `json:"status"`
	ReportPath string `json:"report_path,omitempty"`
}

type BenchmarkResult struct {
	CaseID         string                 `json:"case_id"`
	Score          int                    `json:"score"`
	Passed         bool                   `json:"passed"`
	Criteria       []BenchmarkCriterion   `json:"criteria,omitempty"`
	Record         BenchmarkResultRecord  `json:"record"`
	Replay         BenchmarkReplayOutcome `json:"replay"`
	DetailPagePath string                 `json:"detail_page_path,omitempty"`
}

type BenchmarkRunner struct {
	StorageDir string `json:"storage_dir,omitempty"`
}

type EngineeringOverviewPermission struct {
	ViewerRole     string   `json:"viewer_role"`
	AllowedModules []string `json:"allowed_modules,omitempty"`
}

func (p EngineeringOverviewPermission) CanView(module string) bool {
	module = strings.TrimSpace(module)
	for _, allowed := range p.AllowedModules {
		if strings.EqualFold(strings.TrimSpace(allowed), module) {
			return true
		}
	}
	return false
}

type DashboardWidgetSpec struct {
	WidgetID      string `json:"widget_id"`
	Title         string `json:"title"`
	Module        string `json:"module"`
	DataSource    string `json:"data_source"`
	DefaultWidth  int    `json:"default_width"`
	DefaultHeight int    `json:"default_height"`
	MinWidth      int    `json:"min_width"`
	MaxWidth      int    `json:"max_width"`
}

type DashboardWidgetPlacement struct {
	PlacementID   string   `json:"placement_id"`
	WidgetID      string   `json:"widget_id"`
	Column        int      `json:"column"`
	Row           int      `json:"row"`
	Width         int      `json:"width"`
	Height        int      `json:"height"`
	TitleOverride string   `json:"title_override,omitempty"`
	Filters       []string `json:"filters,omitempty"`
}

type DashboardLayout struct {
	LayoutID   string                     `json:"layout_id"`
	Name       string                     `json:"name"`
	Columns    int                        `json:"columns"`
	Placements []DashboardWidgetPlacement `json:"placements,omitempty"`
}

type DashboardBuilder struct {
	Name                  string                        `json:"name"`
	Period                string                        `json:"period"`
	Owner                 string                        `json:"owner"`
	Permissions           EngineeringOverviewPermission `json:"permissions"`
	Widgets               []DashboardWidgetSpec         `json:"widgets,omitempty"`
	Layouts               []DashboardLayout             `json:"layouts,omitempty"`
	DocumentationComplete bool                          `json:"documentation_complete"`
}

func (b DashboardBuilder) WidgetIndex() map[string]DashboardWidgetSpec {
	out := make(map[string]DashboardWidgetSpec, len(b.Widgets))
	for _, widget := range b.Widgets {
		out[widget.WidgetID] = widget
	}
	return out
}

func NormalizeDashboardLayout(layout DashboardLayout, widgets []DashboardWidgetSpec) DashboardLayout {
	widgetIndex := make(map[string]DashboardWidgetSpec, len(widgets))
	for _, widget := range widgets {
		widgetIndex[widget.WidgetID] = widget
	}

	columnCount := layout.Columns
	if columnCount <= 0 {
		columnCount = 12
	}
	normalized := make([]DashboardWidgetPlacement, 0, len(layout.Placements))
	for _, placement := range layout.Placements {
		minWidth := 1
		maxWidth := columnCount
		if spec, ok := widgetIndex[placement.WidgetID]; ok {
			minWidth = spec.MinWidth
			if minWidth <= 0 {
				minWidth = 1
			}
			if spec.MaxWidth > 0 {
				maxWidth = minInt(spec.MaxWidth, columnCount)
			}
		}
		if maxWidth < minWidth {
			maxWidth = minWidth
		}

		width := maxInt(minWidth, minInt(placement.Width, maxWidth))
		column := maxInt(0, placement.Column)
		if column+width > columnCount {
			column = maxInt(0, columnCount-width)
		}

		normalized = append(normalized, DashboardWidgetPlacement{
			PlacementID:   placement.PlacementID,
			WidgetID:      placement.WidgetID,
			Column:        column,
			Row:           maxInt(0, placement.Row),
			Width:         width,
			Height:        maxInt(1, placement.Height),
			TitleOverride: placement.TitleOverride,
			Filters:       append([]string(nil), placement.Filters...),
		})
	}

	sort.SliceStable(normalized, func(i, j int) bool {
		if normalized[i].Row == normalized[j].Row {
			if normalized[i].Column == normalized[j].Column {
				return normalized[i].PlacementID < normalized[j].PlacementID
			}
			return normalized[i].Column < normalized[j].Column
		}
		return normalized[i].Row < normalized[j].Row
	})

	return DashboardLayout{
		LayoutID:   layout.LayoutID,
		Name:       layout.Name,
		Columns:    columnCount,
		Placements: normalized,
	}
}

type DashboardBuilderAudit struct {
	Name                  string   `json:"name"`
	TotalWidgets          int      `json:"total_widgets"`
	LayoutCount           int      `json:"layout_count"`
	PlacedWidgets         int      `json:"placed_widgets"`
	DuplicatePlacementIDs []string `json:"duplicate_placement_ids,omitempty"`
	MissingWidgetDefs     []string `json:"missing_widget_defs,omitempty"`
	InaccessibleWidgets   []string `json:"inaccessible_widgets,omitempty"`
	OverlappingPlacements []string `json:"overlapping_placements,omitempty"`
	OutOfBoundsPlacements []string `json:"out_of_bounds_placements,omitempty"`
	EmptyLayouts          []string `json:"empty_layouts,omitempty"`
	DocumentationComplete bool     `json:"documentation_complete"`
}

func (a DashboardBuilderAudit) ReleaseReady() bool {
	return len(a.DuplicatePlacementIDs) == 0 &&
		len(a.MissingWidgetDefs) == 0 &&
		len(a.InaccessibleWidgets) == 0 &&
		len(a.OverlappingPlacements) == 0 &&
		len(a.OutOfBoundsPlacements) == 0 &&
		len(a.EmptyLayouts) == 0 &&
		a.DocumentationComplete
}

type EngineeringOverviewKPI struct {
	Name      string  `json:"name"`
	Value     float64 `json:"value"`
	Target    float64 `json:"target"`
	Unit      string  `json:"unit,omitempty"`
	Direction string  `json:"direction,omitempty"`
}

func (k EngineeringOverviewKPI) Healthy() bool {
	if strings.EqualFold(strings.TrimSpace(k.Direction), "down") {
		return k.Value <= k.Target
	}
	return k.Value >= k.Target
}

type EngineeringFunnelStage struct {
	Name  string  `json:"name"`
	Count int     `json:"count"`
	Share float64 `json:"share"`
}

type EngineeringOverviewBlocker struct {
	Summary       string   `json:"summary"`
	AffectedRuns  int      `json:"affected_runs"`
	AffectedTasks []string `json:"affected_tasks,omitempty"`
	Owner         string   `json:"owner,omitempty"`
	Severity      string   `json:"severity,omitempty"`
}

type EngineeringActivity struct {
	Timestamp string `json:"timestamp"`
	RunID     string `json:"run_id,omitempty"`
	TaskID    string `json:"task_id,omitempty"`
	Status    string `json:"status"`
	Summary   string `json:"summary"`
}

type EngineeringOverview struct {
	Name        string                        `json:"name"`
	Period      string                        `json:"period"`
	Permissions EngineeringOverviewPermission `json:"permissions"`
	KPIs        []EngineeringOverviewKPI      `json:"kpis,omitempty"`
	Funnel      []EngineeringFunnelStage      `json:"funnel,omitempty"`
	Blockers    []EngineeringOverviewBlocker  `json:"blockers,omitempty"`
	Activities  []EngineeringActivity         `json:"activities,omitempty"`
}

type VersionedArtifact struct {
	ArtifactType string `json:"artifact_type"`
	ArtifactID   string `json:"artifact_id"`
	Version      string `json:"version"`
	UpdatedAt    string `json:"updated_at"`
	Author       string `json:"author"`
	Summary      string `json:"summary"`
	Content      string `json:"content"`
	ChangeTicket string `json:"change_ticket,omitempty"`
}

type VersionChangeSummary struct {
	FromVersion  string   `json:"from_version"`
	ToVersion    string   `json:"to_version"`
	Additions    int      `json:"additions"`
	Deletions    int      `json:"deletions"`
	ChangedLines int      `json:"changed_lines"`
	Preview      []string `json:"preview,omitempty"`
}

func (s VersionChangeSummary) HasChanges() bool {
	return s.ChangedLines > 0
}

type VersionedArtifactHistory struct {
	ArtifactType     string                `json:"artifact_type"`
	ArtifactID       string                `json:"artifact_id"`
	CurrentVersion   string                `json:"current_version"`
	CurrentUpdatedAt string                `json:"current_updated_at"`
	CurrentAuthor    string                `json:"current_author"`
	CurrentSummary   string                `json:"current_summary"`
	RevisionCount    int                   `json:"revision_count"`
	Revisions        []VersionedArtifact   `json:"revisions,omitempty"`
	RollbackVersion  string                `json:"rollback_version,omitempty"`
	RollbackReady    bool                  `json:"rollback_ready"`
	ChangeSummary    *VersionChangeSummary `json:"change_summary,omitempty"`
}

type PolicyPromptVersionCenter struct {
	Name        string                     `json:"name"`
	GeneratedAt string                     `json:"generated_at"`
	Histories   []VersionedArtifactHistory `json:"histories,omitempty"`
}

func (c PolicyPromptVersionCenter) ArtifactCount() int {
	return len(c.Histories)
}

func (c PolicyPromptVersionCenter) RollbackReadyCount() int {
	count := 0
	for _, history := range c.Histories {
		if history.RollbackReady {
			count++
		}
	}
	return count
}

func Build(tasks []domain.Task, events []domain.Event, weekStart, weekEnd time.Time) Weekly {
	weekly := Weekly{WeekStart: weekStart, WeekEnd: weekEnd}
	byTeam := make(map[string]*TeamBreakdown)
	interventions := interventionCounts(events)
	for _, task := range tasks {
		if !within(task.UpdatedAt, weekStart, weekEnd) {
			continue
		}
		weekly.Summary.TotalRuns++
		weekly.Summary.BudgetCentsTotal += task.BudgetCents
		if task.State == domain.TaskSucceeded {
			weekly.Summary.CompletedRuns++
		}
		if task.State == domain.TaskBlocked || task.State == domain.TaskDeadLetter || task.State == domain.TaskFailed {
			weekly.Summary.BlockedRuns++
		}
		if task.RiskLevel == domain.RiskHigh {
			weekly.Summary.HighRiskRuns++
		}
		if regressionCount(task) > 0 {
			weekly.Summary.RegressionFindings += regressionCount(task)
		}
		if strings.EqualFold(strings.TrimSpace(task.Metadata["plan"]), "premium") {
			weekly.Summary.PremiumRuns++
		}
		weekly.Summary.HumanInterventions += interventions[task.ID]
		team := firstNonEmpty(task.Metadata["team"], "unassigned")
		entry := byTeam[team]
		if entry == nil {
			entry = &TeamBreakdown{Key: team}
			byTeam[team] = entry
		}
		entry.TotalRuns++
		entry.BudgetCentsTotal += task.BudgetCents
		entry.HumanInterventions += interventions[task.ID]
		if task.State == domain.TaskSucceeded {
			entry.CompletedRuns++
		}
		if task.State == domain.TaskBlocked || task.State == domain.TaskDeadLetter || task.State == domain.TaskFailed {
			entry.BlockedRuns++
		}
	}
	for _, entry := range byTeam {
		weekly.TeamBreakdown = append(weekly.TeamBreakdown, *entry)
	}
	sort.SliceStable(weekly.TeamBreakdown, func(i, j int) bool {
		if weekly.TeamBreakdown[i].TotalRuns == weekly.TeamBreakdown[j].TotalRuns {
			return weekly.TeamBreakdown[i].Key < weekly.TeamBreakdown[j].Key
		}
		return weekly.TeamBreakdown[i].TotalRuns > weekly.TeamBreakdown[j].TotalRuns
	})
	weekly.Highlights = buildHighlights(weekly)
	weekly.Actions = buildActions(weekly)
	weekly.Markdown = RenderMarkdown(weekly)
	return weekly
}

func RenderMarkdown(weekly Weekly) string {
	builder := strings.Builder{}
	builder.WriteString("# BigClaw Weekly Ops Report\n\n")
	builder.WriteString(fmt.Sprintf("Window: %s -> %s\n\n", weekly.WeekStart.Format("2006-01-02"), weekly.WeekEnd.Format("2006-01-02")))
	builder.WriteString("## Summary\n")
	builder.WriteString(fmt.Sprintf("- Total runs: %d\n", weekly.Summary.TotalRuns))
	builder.WriteString(fmt.Sprintf("- Completed runs: %d\n", weekly.Summary.CompletedRuns))
	builder.WriteString(fmt.Sprintf("- Blocked runs: %d\n", weekly.Summary.BlockedRuns))
	builder.WriteString(fmt.Sprintf("- High risk runs: %d\n", weekly.Summary.HighRiskRuns))
	builder.WriteString(fmt.Sprintf("- Human interventions: %d\n", weekly.Summary.HumanInterventions))
	builder.WriteString(fmt.Sprintf("- Regressions: %d\n", weekly.Summary.RegressionFindings))
	builder.WriteString(fmt.Sprintf("- Premium runs: %d\n", weekly.Summary.PremiumRuns))
	builder.WriteString(fmt.Sprintf("- Budget cents: %d\n\n", weekly.Summary.BudgetCentsTotal))
	builder.WriteString("## Team Breakdown\n")
	if len(weekly.TeamBreakdown) == 0 {
		builder.WriteString("- None\n\n")
	} else {
		for _, team := range weekly.TeamBreakdown {
			builder.WriteString(fmt.Sprintf("- %s: total=%d completed=%d blocked=%d budget_cents=%d interventions=%d\n", team.Key, team.TotalRuns, team.CompletedRuns, team.BlockedRuns, team.BudgetCentsTotal, team.HumanInterventions))
		}
		builder.WriteString("\n")
	}
	builder.WriteString("## Highlights\n")
	for _, highlight := range weekly.Highlights {
		builder.WriteString("- " + highlight + "\n")
	}
	builder.WriteString("\n")
	builder.WriteString("## Actions\n")
	for _, action := range weekly.Actions {
		builder.WriteString("- " + action + "\n")
	}
	return builder.String()
}

func RenderOperationsDashboard(weekly Weekly) string {
	builder := strings.Builder{}
	builder.WriteString("# Operations Dashboard\n\n")
	builder.WriteString(fmt.Sprintf("- Window: %s -> %s\n", weekly.WeekStart.Format("2006-01-02"), weekly.WeekEnd.Format("2006-01-02")))
	builder.WriteString(fmt.Sprintf("- Total Runs: %d\n", weekly.Summary.TotalRuns))
	builder.WriteString(fmt.Sprintf("- Completed Runs: %d\n", weekly.Summary.CompletedRuns))
	builder.WriteString(fmt.Sprintf("- Blocked Runs: %d\n", weekly.Summary.BlockedRuns))
	builder.WriteString(fmt.Sprintf("- High Risk Runs: %d\n", weekly.Summary.HighRiskRuns))
	builder.WriteString(fmt.Sprintf("- Premium Runs: %d\n", weekly.Summary.PremiumRuns))
	builder.WriteString(fmt.Sprintf("- Human Interventions: %d\n", weekly.Summary.HumanInterventions))
	builder.WriteString(fmt.Sprintf("- Regression Findings: %d\n", weekly.Summary.RegressionFindings))
	builder.WriteString(fmt.Sprintf("- Budget Cents Total: %d\n\n", weekly.Summary.BudgetCentsTotal))
	builder.WriteString("## Team Lanes\n")
	if len(weekly.TeamBreakdown) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, team := range weekly.TeamBreakdown {
			builder.WriteString(fmt.Sprintf("- %s: total=%d completed=%d blocked=%d interventions=%d\n", team.Key, team.TotalRuns, team.CompletedRuns, team.BlockedRuns, team.HumanInterventions))
		}
	}
	return builder.String() + "\n"
}

func RenderOperationsSnapshotDashboard(snapshot OperationsSnapshot, view *SharedViewContext) string {
	builder := strings.Builder{}
	builder.WriteString("# Operations Dashboard\n\n")
	builder.WriteString(fmt.Sprintf("- Total Runs: %d\n", snapshot.TotalRuns))
	builder.WriteString(fmt.Sprintf("- Success Rate: %.1f%%\n", snapshot.SuccessRate))
	builder.WriteString(fmt.Sprintf("- Approval Queue Depth: %d\n", snapshot.ApprovalQueueDepth))
	builder.WriteString(fmt.Sprintf("- SLA Target: %d minutes\n", snapshot.SLATargetMinutes))
	builder.WriteString(fmt.Sprintf("- SLA Breaches: %d\n", snapshot.SLABreachCount))
	builder.WriteString(fmt.Sprintf("- Average Cycle Time: %.1f minutes\n\n", snapshot.AverageCycleMinutes))
	builder.WriteString("## Status Counts\n\n")
	builder.WriteString(renderSharedViewContext(view))
	if len(snapshot.StatusCounts) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, status := range sortedMapKeys(snapshot.StatusCounts) {
			builder.WriteString(fmt.Sprintf("- %s: %d\n", status, snapshot.StatusCounts[status]))
		}
	}
	builder.WriteString("\n## Top Blockers\n\n")
	if len(snapshot.TopBlockers) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, cluster := range snapshot.TopBlockers {
			builder.WriteString(fmt.Sprintf("- %s: occurrences=%d statuses=%s tasks=%s\n", cluster.Reason, cluster.Occurrences(), joinOrNone(cluster.Statuses), joinOrNone(cluster.TaskIDs)))
		}
	}
	return builder.String() + "\n"
}

func BuildOperationsMetricSpec(tasks []domain.Task, events []domain.Event, periodStart, periodEnd time.Time, timezoneName string, slaTargetMinutes int) OperationsMetricSpec {
	if timezoneName == "" {
		timezoneName = "UTC"
	}
	if slaTargetMinutes <= 0 {
		slaTargetMinutes = 60
	}
	interventions := interventionCounts(events)
	totalRuns := len(tasks)
	runsInWindow := 0
	cycleSum := 0.0
	cycleCount := 0
	slaCompliantRuns := 0
	intervenedRuns := 0
	regressionFindings := 0
	riskSum := 0.0
	riskCount := 0
	budgetUSDTotal := 0.0

	for _, task := range tasks {
		if within(task.CreatedAt, periodStart, periodEnd) || within(task.UpdatedAt, periodStart, periodEnd) {
			runsInWindow++
		}
		if cycle, ok := cycleMinutes(task); ok {
			cycleSum += cycle
			cycleCount++
			if cycle <= float64(slaTargetMinutes) {
				slaCompliantRuns++
			}
		}
		if interventions[task.ID] > 0 || strings.EqualFold(strings.TrimSpace(task.Metadata["approval_status"]), "needs-approval") {
			intervenedRuns++
		}
		regressionFindings += regressionCount(task)
		if score, ok := riskScoreForTask(task); ok {
			riskSum += score
			riskCount++
		}
		budgetUSDTotal += float64(task.BudgetCents) / 100.0
	}

	avgCycle := 0.0
	if cycleCount > 0 {
		avgCycle = roundTenth(cycleSum / float64(cycleCount))
	}
	interventionRate := 0.0
	if totalRuns > 0 {
		interventionRate = roundTenth((float64(intervenedRuns) / float64(totalRuns)) * 100)
	}
	slaCompliance := 0.0
	if cycleCount > 0 {
		slaCompliance = roundTenth((float64(slaCompliantRuns) / float64(cycleCount)) * 100)
	}
	avgRisk := 0.0
	if riskCount > 0 {
		avgRisk = roundTenth(riskSum / float64(riskCount))
	}
	budgetUSDTotal = math.Round(budgetUSDTotal*100) / 100

	definitions := []OperationsMetricDefinition{
		{MetricID: "runs-window", Label: "Runs In Window", Unit: "runs", Direction: "up", Formula: "count(tasks.created_at|updated_at within period)", Description: "Number of runs active inside the reporting window.", SourceFields: []string{"task.created_at", "task.updated_at"}},
		{MetricID: "avg-cycle-minutes", Label: "Avg Cycle Minutes", Unit: "m", Direction: "down", Formula: "sum(updated_at - created_at) / measured_runs", Description: "Average measured run cycle time in minutes.", SourceFields: []string{"task.created_at", "task.updated_at"}},
		{MetricID: "intervention-rate", Label: "Intervention Rate", Unit: "%", Direction: "down", Formula: "intervened_runs / total_runs", Description: "Share of runs that required manual intervention or approval handling.", SourceFields: []string{"events", "task.metadata.approval_status"}},
		{MetricID: "sla-compliance", Label: "SLA Compliance", Unit: "%", Direction: "up", Formula: "runs within SLA target / measured_runs", Description: "Measured runs that completed inside the SLA target.", SourceFields: []string{"task.created_at", "task.updated_at"}},
		{MetricID: "regression-findings", Label: "Regression Findings", Unit: "findings", Direction: "down", Formula: "sum(task.metadata.regression_count)", Description: "Regression findings attached to the reporting window.", SourceFields: []string{"task.metadata.regression_count", "task.metadata.regression"}},
		{MetricID: "avg-risk-score", Label: "Avg Risk Score", Unit: "score", Direction: "down", Formula: "avg(mapped risk_level)", Description: "Average mapped risk score using low=25, medium=60, high=90.", SourceFields: []string{"task.risk_level"}},
		{MetricID: "budget-spend", Label: "Budget Spend", Unit: "USD", Direction: "down", Formula: "sum(task.budget_cents) / 100", Description: "Budget total represented by the reporting slice.", SourceFields: []string{"task.budget_cents"}},
	}

	return OperationsMetricSpec{
		Name:         "Operations Metric Spec",
		GeneratedAt:  time.Now().UTC(),
		PeriodStart:  periodStart.UTC(),
		PeriodEnd:    periodEnd.UTC(),
		TimezoneName: timezoneName,
		Definitions:  definitions,
		Values: []OperationsMetricValue{
			{MetricID: "runs-window", Label: "Runs In Window", Value: float64(runsInWindow), DisplayValue: strconv.Itoa(runsInWindow), Numerator: float64(runsInWindow), Denominator: float64(totalRuns), Unit: "runs", Evidence: []string{fmt.Sprintf("%d of %d runs were created or updated inside the reporting window.", runsInWindow, totalRuns)}},
			{MetricID: "avg-cycle-minutes", Label: "Avg Cycle Minutes", Value: avgCycle, DisplayValue: fmt.Sprintf("%.1fm", avgCycle), Numerator: roundTenth(cycleSum), Denominator: float64(cycleCount), Unit: "m", Evidence: []string{fmt.Sprintf("%d runs had valid created_at and updated_at timestamps.", cycleCount)}},
			{MetricID: "intervention-rate", Label: "Intervention Rate", Value: interventionRate, DisplayValue: fmt.Sprintf("%.1f%%", interventionRate), Numerator: float64(intervenedRuns), Denominator: float64(totalRuns), Unit: "%", Evidence: []string{fmt.Sprintf("%d runs had intervention events or approval-required status.", intervenedRuns)}},
			{MetricID: "sla-compliance", Label: "SLA Compliance", Value: slaCompliance, DisplayValue: fmt.Sprintf("%.1f%%", slaCompliance), Numerator: float64(slaCompliantRuns), Denominator: float64(cycleCount), Unit: "%", Evidence: []string{fmt.Sprintf("SLA target: %d minutes.", slaTargetMinutes), fmt.Sprintf("%d of %d measured runs met target.", slaCompliantRuns, cycleCount)}},
			{MetricID: "regression-findings", Label: "Regression Findings", Value: float64(regressionFindings), DisplayValue: strconv.Itoa(regressionFindings), Numerator: float64(regressionFindings), Denominator: float64(totalRuns), Unit: "findings", Evidence: []string{"Regression count uses task metadata fields `regression_count`, `regressions`, or boolean `regression`."}},
			{MetricID: "avg-risk-score", Label: "Avg Risk Score", Value: avgRisk, DisplayValue: fmt.Sprintf("%.1f", avgRisk), Numerator: roundTenth(riskSum), Denominator: float64(riskCount), Unit: "score", Evidence: []string{"Risk score mapping is low=25, medium=60, high=90."}},
			{MetricID: "budget-spend", Label: "Budget Spend", Value: budgetUSDTotal, DisplayValue: fmt.Sprintf("$%.2f", budgetUSDTotal), Numerator: budgetUSDTotal, Denominator: float64(totalRuns), Unit: "USD", Evidence: []string{"Budget spend is derived from `task.budget_cents`."}},
		},
	}
}

func RenderOperationsMetricSpec(spec OperationsMetricSpec) string {
	builder := strings.Builder{}
	builder.WriteString("# Operations Metric Spec\n\n")
	builder.WriteString(fmt.Sprintf("- Name: %s\n", strings.TrimSpace(spec.Name)))
	builder.WriteString(fmt.Sprintf("- Generated At: %s\n", spec.GeneratedAt.UTC().Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("- Period Start: %s\n", spec.PeriodStart.UTC().Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("- Period End: %s\n", spec.PeriodEnd.UTC().Format(time.RFC3339)))
	builder.WriteString(fmt.Sprintf("- Timezone: %s\n\n", firstNonEmpty(spec.TimezoneName, "UTC")))
	builder.WriteString("## Definitions\n\n")
	if len(spec.Definitions) == 0 {
		builder.WriteString("- None\n\n")
	} else {
		for _, definition := range spec.Definitions {
			builder.WriteString(fmt.Sprintf("### %s\n\n", firstNonEmpty(definition.Label, definition.MetricID)))
			builder.WriteString(fmt.Sprintf("- Metric ID: %s\n", definition.MetricID))
			builder.WriteString(fmt.Sprintf("- Unit: %s\n", definition.Unit))
			builder.WriteString(fmt.Sprintf("- Direction: %s\n", definition.Direction))
			builder.WriteString(fmt.Sprintf("- Formula: %s\n", definition.Formula))
			builder.WriteString(fmt.Sprintf("- Description: %s\n", definition.Description))
			builder.WriteString(fmt.Sprintf("- Source Fields: %s\n\n", strings.Join(definition.SourceFields, ", ")))
		}
	}
	builder.WriteString("## Values\n")
	if len(spec.Values) == 0 {
		builder.WriteString("\n- None\n")
		return builder.String()
	}
	builder.WriteString("\n")
	for _, value := range spec.Values {
		evidence := "none"
		if len(value.Evidence) > 0 {
			evidence = strings.Join(value.Evidence, " | ")
		}
		builder.WriteString(fmt.Sprintf("- %s: value=%s numerator=%.1f denominator=%.1f unit=%s evidence=%s\n", firstNonEmpty(value.Label, value.MetricID), firstNonEmpty(value.DisplayValue, formatMetricValue(value.Value)), value.Numerator, value.Denominator, value.Unit, evidence))
	}
	return builder.String()
}

func WriteReport(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func BuildLaunchChecklist(issueID string, documentation []DocumentationArtifact, items []LaunchChecklistItem) LaunchChecklist {
	return LaunchChecklist{
		IssueID:       issueID,
		Documentation: documentation,
		Items:         items,
	}
}

func BuildFinalDeliveryChecklist(issueID string, requiredOutputs, recommendedDocumentation []DocumentationArtifact) FinalDeliveryChecklist {
	return FinalDeliveryChecklist{
		IssueID:                  issueID,
		RequiredOutputs:          requiredOutputs,
		RecommendedDocumentation: recommendedDocumentation,
	}
}

func RenderReportStudioReport(studio ReportStudio) string {
	builder := strings.Builder{}
	builder.WriteString("# Report Studio\n\n")
	builder.WriteString(fmt.Sprintf("- Name: %s\n", studio.Name))
	builder.WriteString(fmt.Sprintf("- Issue ID: %s\n", studio.IssueID))
	builder.WriteString(fmt.Sprintf("- Audience: %s\n", studio.Audience))
	builder.WriteString(fmt.Sprintf("- Period: %s\n", studio.Period))
	builder.WriteString(fmt.Sprintf("- Sections: %d\n", len(studio.Sections)))
	builder.WriteString(fmt.Sprintf("- Recommendation: %s\n\n", studio.Recommendation()))
	builder.WriteString("## Narrative Summary\n\n")
	if studio.Summary != "" {
		builder.WriteString(studio.Summary)
	} else {
		builder.WriteString("No summary drafted.")
	}
	builder.WriteString("\n\n## Sections\n\n")
	if len(studio.Sections) == 0 {
		builder.WriteString("- None\n\n")
	} else {
		for _, section := range studio.Sections {
			builder.WriteString(fmt.Sprintf("### %s\n\n", section.Heading))
			if section.Body != "" {
				builder.WriteString(section.Body)
			} else {
				builder.WriteString("No narrative drafted.")
			}
			builder.WriteString("\n\n")
			evidence := "None"
			if len(section.Evidence) > 0 {
				evidence = strings.Join(section.Evidence, ", ")
			}
			callouts := "None"
			if len(section.Callouts) > 0 {
				callouts = strings.Join(section.Callouts, ", ")
			}
			builder.WriteString(fmt.Sprintf("- Evidence: %s\n", evidence))
			builder.WriteString(fmt.Sprintf("- Callouts: %s\n\n", callouts))
		}
	}
	builder.WriteString("## Action Items\n\n")
	if len(studio.ActionItems) == 0 {
		builder.WriteString("- None\n\n")
	} else {
		for _, item := range studio.ActionItems {
			builder.WriteString(fmt.Sprintf("- %s\n", item))
		}
		builder.WriteString("\n")
	}
	builder.WriteString("## Sources\n\n")
	if len(studio.SourceReports) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, path := range studio.SourceReports {
			builder.WriteString(fmt.Sprintf("- %s\n", path))
		}
	}
	return builder.String() + "\n"
}

func RenderReportStudioPlainText(studio ReportStudio) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("%s (%s)\n", studio.Name, studio.IssueID))
	builder.WriteString(fmt.Sprintf("Audience: %s\n", studio.Audience))
	builder.WriteString(fmt.Sprintf("Period: %s\n", studio.Period))
	builder.WriteString(fmt.Sprintf("Recommendation: %s\n\n", studio.Recommendation()))
	if studio.Summary != "" {
		builder.WriteString(studio.Summary)
	} else {
		builder.WriteString("No summary drafted.")
	}
	builder.WriteString("\n\n")
	for _, section := range studio.Sections {
		builder.WriteString(strings.ToUpper(section.Heading) + "\n")
		if section.Body != "" {
			builder.WriteString(section.Body)
		} else {
			builder.WriteString("No narrative drafted.")
		}
		builder.WriteString("\n")
		if len(section.Callouts) > 0 {
			builder.WriteString("Callouts: " + strings.Join(section.Callouts, "; ") + "\n")
		}
		if len(section.Evidence) > 0 {
			builder.WriteString("Evidence: " + strings.Join(section.Evidence, "; ") + "\n")
		}
		builder.WriteString("\n")
	}
	if len(studio.ActionItems) > 0 {
		builder.WriteString("Action Items:\n")
		for _, item := range studio.ActionItems {
			builder.WriteString(fmt.Sprintf("- %s\n", item))
		}
		builder.WriteString("\n")
	}
	return strings.TrimRight(builder.String(), "\n") + "\n"
}

func RenderReportStudioHTML(studio ReportStudio) string {
	sectionHTML := strings.Builder{}
	for _, section := range studio.Sections {
		evidence := "None"
		if len(section.Evidence) > 0 {
			evidence = strings.Join(section.Evidence, ", ")
		}
		callouts := "None"
		if len(section.Callouts) > 0 {
			callouts = strings.Join(section.Callouts, ", ")
		}
		sectionHTML.WriteString(fmt.Sprintf(`
        <section class="section">
          <h2>%s</h2>
          <p>%s</p>
          <p class="meta">Evidence: %s</p>
          <p class="meta">Callouts: %s</p>
        </section>
        `, html.EscapeString(section.Heading), html.EscapeString(section.Body), html.EscapeString(evidence), html.EscapeString(callouts)))
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
	sections := sectionHTML.String()
	if sections == "" {
		sections = `<section class="section"><p>No sections drafted.</p></section>`
	}
	summary := studio.Summary
	if summary == "" {
		summary = "No summary drafted."
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
`, html.EscapeString(studio.Name), html.EscapeString(studio.IssueID), html.EscapeString(studio.Audience), html.EscapeString(studio.Period), html.EscapeString(studio.Name), html.EscapeString(studio.Recommendation()), html.EscapeString(summary), sections, actionHTML, sourceHTML)
}

func RenderLaunchChecklistReport(checklist LaunchChecklist) string {
	builder := strings.Builder{}
	builder.WriteString("# Launch Checklist\n\n")
	builder.WriteString(fmt.Sprintf("- Issue ID: %s\n", checklist.IssueID))
	builder.WriteString(fmt.Sprintf("- Linked Documentation: %d\n", len(checklist.Documentation)))
	builder.WriteString(fmt.Sprintf("- Completed Items: %d/%d\n", checklist.CompletedItems(), len(checklist.Items)))
	builder.WriteString(fmt.Sprintf("- Ready: %t\n\n", checklist.Ready()))
	builder.WriteString("## Documentation\n\n")
	if len(checklist.Documentation) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, artifact := range checklist.Documentation {
			builder.WriteString(fmt.Sprintf("- %s: available=%t path=%s\n", artifact.Name, artifact.Available(), artifact.Path))
		}
	}
	builder.WriteString("\n## Checklist\n\n")
	if len(checklist.Items) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, item := range checklist.Items {
			evidence := "none"
			if len(item.Evidence) > 0 {
				evidence = strings.Join(item.Evidence, ", ")
			}
			builder.WriteString(fmt.Sprintf("- %s: completed=%t evidence=%s\n", item.Name, checklist.ItemCompleted(item), evidence))
		}
	}
	return builder.String() + "\n"
}

func RenderFinalDeliveryChecklistReport(checklist FinalDeliveryChecklist) string {
	builder := strings.Builder{}
	builder.WriteString("# Final Delivery Checklist\n\n")
	builder.WriteString(fmt.Sprintf("- Issue ID: %s\n", checklist.IssueID))
	builder.WriteString(fmt.Sprintf("- Required Outputs Generated: %d/%d\n", checklist.GeneratedRequiredOutputs(), len(checklist.RequiredOutputs)))
	builder.WriteString(fmt.Sprintf("- Recommended Docs Generated: %d/%d\n", checklist.GeneratedRecommendedDocumentation(), len(checklist.RecommendedDocumentation)))
	builder.WriteString(fmt.Sprintf("- Ready: %t\n\n", checklist.Ready()))
	builder.WriteString("## Required Outputs\n\n")
	if len(checklist.RequiredOutputs) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, artifact := range checklist.RequiredOutputs {
			builder.WriteString(fmt.Sprintf("- %s: available=%t path=%s\n", artifact.Name, artifact.Available(), artifact.Path))
		}
	}
	builder.WriteString("\n## Recommended Documentation\n\n")
	if len(checklist.RecommendedDocumentation) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, artifact := range checklist.RecommendedDocumentation {
			builder.WriteString(fmt.Sprintf("- %s: available=%t path=%s\n", artifact.Name, artifact.Available(), artifact.Path))
		}
	}
	return builder.String() + "\n"
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

func RenderRepoSyncAuditReport(audit RepoSyncAudit) string {
	builder := strings.Builder{}
	builder.WriteString("# Repo Sync Audit\n\n")
	builder.WriteString("## Sync Status\n\n")
	builder.WriteString(fmt.Sprintf("- Status: %s\n", audit.Sync.Status))
	failureCategory := audit.Sync.FailureCategory
	if failureCategory == "" {
		failureCategory = "none"
	}
	builder.WriteString(fmt.Sprintf("- Failure Category: %s\n", failureCategory))
	summary := audit.Sync.Summary
	if summary == "" {
		summary = "none"
	}
	builder.WriteString(fmt.Sprintf("- Summary: %s\n", summary))
	branch := audit.Sync.Branch
	if branch == "" {
		branch = "unknown"
	}
	builder.WriteString(fmt.Sprintf("- Branch: %s\n", branch))
	remote := audit.Sync.Remote
	if remote == "" {
		remote = "origin"
	}
	builder.WriteString(fmt.Sprintf("- Remote: %s\n", remote))
	remoteRef := audit.Sync.RemoteRef
	if remoteRef == "" {
		remoteRef = "unknown"
	}
	builder.WriteString(fmt.Sprintf("- Remote Ref: %s\n", remoteRef))
	builder.WriteString(fmt.Sprintf("- Ahead By: %d\n", audit.Sync.AheadBy))
	builder.WriteString(fmt.Sprintf("- Behind By: %d\n", audit.Sync.BehindBy))
	dirtyPaths := "none"
	if len(audit.Sync.DirtyPaths) > 0 {
		dirtyPaths = strings.Join(audit.Sync.DirtyPaths, ", ")
	}
	builder.WriteString(fmt.Sprintf("- Dirty Paths: %s\n", dirtyPaths))
	authTarget := audit.Sync.AuthTarget
	if authTarget == "" {
		authTarget = "none"
	}
	builder.WriteString(fmt.Sprintf("- Auth Target: %s\n", authTarget))
	checkedAt := audit.Sync.Timestamp
	if checkedAt == "" {
		checkedAt = "unknown"
	}
	builder.WriteString(fmt.Sprintf("- Checked At: %s\n", checkedAt))
	builder.WriteString("\n## Pull Request Freshness\n\n")
	prNumber := "unknown"
	if audit.PullRequest.PRNumber != nil {
		prNumber = strconv.Itoa(*audit.PullRequest.PRNumber)
	}
	builder.WriteString(fmt.Sprintf("- PR Number: %s\n", prNumber))
	prURL := audit.PullRequest.PRURL
	if prURL == "" {
		prURL = "none"
	}
	builder.WriteString(fmt.Sprintf("- PR URL: %s\n", prURL))
	builder.WriteString(fmt.Sprintf("- Branch State: %s\n", audit.PullRequest.BranchState))
	builder.WriteString(fmt.Sprintf("- Body State: %s\n", audit.PullRequest.BodyState))
	branchHead := audit.PullRequest.BranchHeadSHA
	if branchHead == "" {
		branchHead = "unknown"
	}
	builder.WriteString(fmt.Sprintf("- Branch Head SHA: %s\n", branchHead))
	prHead := audit.PullRequest.PRHeadSHA
	if prHead == "" {
		prHead = "unknown"
	}
	builder.WriteString(fmt.Sprintf("- PR Head SHA: %s\n", prHead))
	expectedDigest := audit.PullRequest.ExpectedBodyDigest
	if expectedDigest == "" {
		expectedDigest = "unknown"
	}
	builder.WriteString(fmt.Sprintf("- Expected Body Digest: %s\n", expectedDigest))
	actualDigest := audit.PullRequest.ActualBodyDigest
	if actualDigest == "" {
		actualDigest = "unknown"
	}
	builder.WriteString(fmt.Sprintf("- Actual Body Digest: %s\n", actualDigest))
	prCheckedAt := audit.PullRequest.CheckedAt
	if prCheckedAt == "" {
		prCheckedAt = "unknown"
	}
	builder.WriteString(fmt.Sprintf("- Checked At: %s\n", prCheckedAt))
	builder.WriteString("\n## Summary\n\n")
	builder.WriteString(fmt.Sprintf("- %s\n", audit.Summary()))
	return builder.String()
}

func EvaluateIssueClosure(issueID, reportPath string, validationPassed bool, launchChecklist *LaunchChecklist, finalDeliveryChecklist *FinalDeliveryChecklist) IssueClosureDecision {
	resolvedPath := ""
	if strings.TrimSpace(reportPath) != "" {
		resolvedPath = filepath.Clean(reportPath)
	}
	if !ValidationReportExists(reportPath) {
		return IssueClosureDecision{
			IssueID:    issueID,
			Allowed:    false,
			Reason:     "validation report required before closing issue",
			ReportPath: resolvedPath,
		}
	}
	if !validationPassed {
		return IssueClosureDecision{
			IssueID:    issueID,
			Allowed:    false,
			Reason:     "validation failed; issue must remain open",
			ReportPath: resolvedPath,
		}
	}
	if finalDeliveryChecklist != nil && !finalDeliveryChecklist.Ready() {
		return IssueClosureDecision{
			IssueID:    issueID,
			Allowed:    false,
			Reason:     "final delivery checklist incomplete; required outputs missing",
			ReportPath: resolvedPath,
		}
	}
	if launchChecklist != nil && !launchChecklist.Ready() {
		return IssueClosureDecision{
			IssueID:    issueID,
			Allowed:    false,
			Reason:     "launch checklist incomplete; linked documentation missing or empty",
			ReportPath: resolvedPath,
		}
	}
	if finalDeliveryChecklist != nil {
		return IssueClosureDecision{
			IssueID:    issueID,
			Allowed:    true,
			Reason:     "validation report and final delivery checklist requirements satisfied; issue can be closed",
			ReportPath: resolvedPath,
		}
	}
	return IssueClosureDecision{
		IssueID:    issueID,
		Allowed:    true,
		Reason:     "validation report and launch checklist requirements satisfied; issue can be closed",
		ReportPath: resolvedPath,
	}
}

func slugifyReportStudioName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastHyphen := false
	for _, r := range value {
		isLetter := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if isLetter || isDigit {
			builder.WriteRune(r)
			lastHyphen = false
			continue
		}
		if !lastHyphen && builder.Len() > 0 {
			builder.WriteByte('-')
			lastHyphen = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func RenderIssueValidationReport(issueID, version, environment, summary string) string {
	return "# Issue Validation Report\n\n" +
		"- Issue ID: " + issueID + "\n" +
		"- 版本号: " + version + "\n" +
		"- 测试环境: " + environment + "\n" +
		"- 生成时间: " + time.Now().UTC().Format(time.RFC3339) + "\n\n" +
		"## 结论\n\n" +
		summary + "\n"
}

func BuildOrchestrationCanvasFromLedgerEntry(entry map[string]any) OrchestrationCanvas {
	audits := mapsFromAny(entry["audits"])
	planAudit := latestNamedAudit(audits, "orchestration.plan")
	policyAudit := latestNamedAudit(audits, "orchestration.policy")
	handoffAudit := latestHandoffAudit(audits)
	planDetails := map[string]any{}
	if planAudit != nil {
		planDetails = mapFromAny(planAudit["details"])
	}
	policyDetails := map[string]any{}
	if policyAudit != nil {
		policyDetails = mapFromAny(policyAudit["details"])
	}
	handoffDetails := map[string]any{}
	if handoffAudit != nil {
		handoffDetails = mapFromAny(handoffAudit["details"])
	}
	activeTools := map[string]struct{}{}
	for _, audit := range audits {
		if stringValue(audit["action"]) != "tool.invoke" {
			continue
		}
		tool := stringValue(mapFromAny(audit["details"])["tool"])
		if tool != "" {
			activeTools[tool] = struct{}{}
		}
	}
	canvas := OrchestrationCanvas{
		TaskID:             stringValue(entry["task_id"]),
		RunID:              stringValue(entry["run_id"]),
		CollaborationMode:  firstNonEmpty(stringValue(planDetails["collaboration_mode"]), "single-team"),
		Departments:        stringListFromAny(planDetails["departments"]),
		RequiredApprovals:  stringListFromAny(planDetails["approvals"]),
		Tier:               firstNonEmpty(stringValue(policyDetails["tier"]), "standard"),
		UpgradeRequired:    policyAudit != nil && stringValue(policyAudit["outcome"]) == "upgrade-required",
		BlockedDepartments: stringListFromAny(policyDetails["blocked_departments"]),
		HandoffTeam:        "none",
		HandoffStatus:      "none",
		HandoffReason:      "",
		ActiveTools:        sortedKeys(activeTools),
		EntitlementStatus:  firstNonEmpty(stringValue(policyDetails["entitlement_status"]), "included"),
		BillingModel:       firstNonEmpty(stringValue(policyDetails["billing_model"]), "standard-included"),
		EstimatedCostUSD:   anyFloat(policyDetails["estimated_cost_usd"]),
		IncludedUsageUnits: int(anyFloat(policyDetails["included_usage_units"])),
		OverageUsageUnits:  int(anyFloat(policyDetails["overage_usage_units"])),
		OverageCostUSD:     anyFloat(policyDetails["overage_cost_usd"]),
	}
	if handoffAudit != nil {
		canvas.HandoffTeam = firstNonEmpty(stringValue(handoffDetails["target_team"]), "none")
		canvas.HandoffStatus = firstNonEmpty(stringValue(handoffAudit["outcome"]), "none")
		canvas.HandoffReason = stringValue(handoffDetails["reason"])
	}
	canvas.Collaboration = buildCollaborationThreadFromAudits(audits, "flow", canvas.RunID)
	canvas.Actions = buildConsoleActions(
		canvas.RunID,
		handoffAudit == nil || canvas.HandoffStatus != "pending",
		handoffAudit == nil || canvas.HandoffStatus != "completed",
		policyAudit != nil && stringValue(policyAudit["outcome"]) == "upgrade-required",
	)
	return canvas
}

func BuildTakeoverQueueFromLedger(entries []map[string]any, period string) TakeoverQueue {
	queue := TakeoverQueue{Name: "Human Takeover Queue", Period: period}
	for _, entry := range entries {
		audits := mapsFromAny(entry["audits"])
		handoffAudit := latestHandoffAudit(audits)
		if handoffAudit == nil {
			continue
		}
		details := mapFromAny(handoffAudit["details"])
		queue.Requests = append(queue.Requests, TakeoverRequest{
			RunID:             stringValue(entry["run_id"]),
			TaskID:            stringValue(entry["task_id"]),
			Source:            stringValue(entry["source"]),
			TargetTeam:        firstNonEmpty(stringValue(details["target_team"]), "operations"),
			Status:            firstNonEmpty(stringValue(handoffAudit["outcome"]), "pending"),
			Reason:            firstNonEmpty(stringValue(details["reason"]), firstNonEmpty(stringValue(entry["summary"]), "handoff requested")),
			RequiredApprovals: stringListFromAny(details["required_approvals"]),
			Actions: []ConsoleAction{
				{ActionID: "drill-down", Label: "Drill Down", Target: stringValue(entry["run_id"]), Enabled: true},
				{ActionID: "export", Label: "Export", Target: stringValue(entry["run_id"]), Enabled: true},
				{ActionID: "add-note", Label: "Add Note", Target: stringValue(entry["run_id"]), Enabled: true},
				{
					ActionID: "escalate",
					Label:    "Escalate",
					Target:   stringValue(entry["run_id"]),
					Enabled:  stringValue(details["target_team"]) != "security",
					Reason:   disabledReason(stringValue(details["target_team"]) != "security", "security takeovers are already escalated"),
				},
				{
					ActionID: "retry",
					Label:    "Retry",
					Target:   stringValue(entry["run_id"]),
					Enabled:  false,
					Reason:   "retry is blocked while takeover is pending",
				},
				{
					ActionID: "pause",
					Label:    "Pause",
					Target:   stringValue(entry["run_id"]),
					Enabled:  stringValue(handoffAudit["outcome"]) == "pending",
					Reason:   disabledReason(stringValue(handoffAudit["outcome"]) == "pending", "only pending takeovers can be paused"),
				},
				{ActionID: "audit", Label: "Audit Trail", Target: stringValue(entry["run_id"]), Enabled: true},
			},
		})
	}
	sort.Slice(queue.Requests, func(i, j int) bool {
		if queue.Requests[i].TargetTeam != queue.Requests[j].TargetTeam {
			return queue.Requests[i].TargetTeam < queue.Requests[j].TargetTeam
		}
		return queue.Requests[i].RunID < queue.Requests[j].RunID
	})
	return queue
}

func RenderTakeoverQueueReport(queue TakeoverQueue, totalRuns *int, view *SharedViewContext) string {
	teamCounts := queue.TeamCounts()
	teamKeys := make([]string, 0, len(teamCounts))
	for team := range teamCounts {
		teamKeys = append(teamKeys, team)
	}
	sort.Strings(teamKeys)
	teamMix := "none"
	if len(teamKeys) > 0 {
		parts := make([]string, 0, len(teamKeys))
		for _, team := range teamKeys {
			parts = append(parts, fmt.Sprintf("%s=%d", team, teamCounts[team]))
		}
		teamMix = strings.Join(parts, " ")
	}
	reportTotalRuns := queue.PendingRequests()
	if totalRuns != nil {
		reportTotalRuns = *totalRuns
	}
	builder := strings.Builder{}
	builder.WriteString("# Human Takeover Queue\n\n")
	builder.WriteString(fmt.Sprintf("- Queue: %s\n", queue.Name))
	builder.WriteString(fmt.Sprintf("- Period: %s\n", queue.Period))
	builder.WriteString(fmt.Sprintf("- Pending Requests: %d\n", queue.PendingRequests()))
	builder.WriteString(fmt.Sprintf("- Total Runs: %d\n", reportTotalRuns))
	builder.WriteString(fmt.Sprintf("- Recommendation: %s\n", queue.Recommendation()))
	builder.WriteString(fmt.Sprintf("- Team Mix: %s\n", teamMix))
	builder.WriteString(fmt.Sprintf("- Required Approvals: %d\n\n", queue.ApprovalCount()))
	builder.WriteString("## Requests\n\n")
	builder.WriteString(renderSharedViewContext(view))
	if len(queue.Requests) == 0 {
		builder.WriteString("- None\n")
		return builder.String() + "\n"
	}
	for _, request := range queue.Requests {
		approvals := "none"
		if len(request.RequiredApprovals) > 0 {
			approvals = strings.Join(request.RequiredApprovals, ",")
		}
		builder.WriteString(fmt.Sprintf("- %s: team=%s status=%s task=%s approvals=%s reason=%s actions=%s\n", request.RunID, request.TargetTeam, request.Status, request.TaskID, approvals, request.Reason, RenderConsoleActions(request.Actions)))
	}
	return builder.String() + "\n"
}

func RenderOrchestrationCanvas(canvas OrchestrationCanvas) string {
	builder := strings.Builder{}
	builder.WriteString("# Orchestration Canvas\n\n")
	builder.WriteString(fmt.Sprintf("- Task ID: %s\n", canvas.TaskID))
	builder.WriteString(fmt.Sprintf("- Run ID: %s\n", canvas.RunID))
	builder.WriteString(fmt.Sprintf("- Collaboration Mode: %s\n", canvas.CollaborationMode))
	builder.WriteString(fmt.Sprintf("- Departments: %s\n", joinOrNone(canvas.Departments)))
	builder.WriteString(fmt.Sprintf("- Required Approvals: %s\n", joinOrNone(canvas.RequiredApprovals)))
	builder.WriteString(fmt.Sprintf("- Tier: %s\n", canvas.Tier))
	builder.WriteString(fmt.Sprintf("- Upgrade Required: %t\n", canvas.UpgradeRequired))
	builder.WriteString(fmt.Sprintf("- Entitlement Status: %s\n", canvas.EntitlementStatus))
	builder.WriteString(fmt.Sprintf("- Billing Model: %s\n", canvas.BillingModel))
	builder.WriteString(fmt.Sprintf("- Blocked Departments: %s\n", joinOrNone(canvas.BlockedDepartments)))
	builder.WriteString(fmt.Sprintf("- Handoff Team: %s\n", canvas.HandoffTeam))
	builder.WriteString(fmt.Sprintf("- Handoff Status: %s\n", canvas.HandoffStatus))
	builder.WriteString(fmt.Sprintf("- Recommendation: %s\n\n", canvas.Recommendation()))
	builder.WriteString("## Execution Context\n\n")
	builder.WriteString(fmt.Sprintf("- Active Tools: %s\n", joinOrNone(canvas.ActiveTools)))
	builder.WriteString(fmt.Sprintf("- Estimated Cost (USD): %.2f\n", canvas.EstimatedCostUSD))
	builder.WriteString(fmt.Sprintf("- Included Usage Units: %d\n", canvas.IncludedUsageUnits))
	builder.WriteString(fmt.Sprintf("- Overage Usage Units: %d\n", canvas.OverageUsageUnits))
	builder.WriteString(fmt.Sprintf("- Overage Cost (USD): %.2f\n", canvas.OverageCostUSD))
	builder.WriteString(fmt.Sprintf("- Handoff Reason: %s\n\n", firstNonEmpty(canvas.HandoffReason, "none")))
	builder.WriteString("## Actions\n\n")
	builder.WriteString(fmt.Sprintf("- %s\n", RenderConsoleActions(canvas.Actions)))
	builder.WriteString(renderCollaborationLines(canvas.Collaboration))
	return builder.String() + "\n"
}

func BuildOrchestrationPortfolio(canvases []OrchestrationCanvas, name, period string, takeoverQueue *TakeoverQueue) OrchestrationPortfolio {
	return OrchestrationPortfolio{
		Name:          name,
		Period:        period,
		Canvases:      append([]OrchestrationCanvas(nil), canvases...),
		TakeoverQueue: takeoverQueue,
	}
}

func BuildOrchestrationPortfolioFromLedger(entries []map[string]any, name, period string) OrchestrationPortfolio {
	canvases := make([]OrchestrationCanvas, 0, len(entries))
	for _, entry := range entries {
		if latestNamedAudit(mapsFromAny(entry["audits"]), "orchestration.plan") == nil {
			continue
		}
		canvases = append(canvases, BuildOrchestrationCanvasFromLedgerEntry(entry))
	}
	queue := BuildTakeoverQueueFromLedger(entries, period)
	queue.Name = name + " Takeovers"
	return BuildOrchestrationPortfolio(canvases, name, period, &queue)
}

func BuildBillingEntitlementsPage(portfolio OrchestrationPortfolio, workspaceName, planName, billingPeriod string) BillingEntitlementsPage {
	charges := make([]BillingRunCharge, 0, len(portfolio.Canvases))
	for _, canvas := range portfolio.Canvases {
		charges = append(charges, BillingRunCharge{
			RunID:               canvas.RunID,
			TaskID:              canvas.TaskID,
			BillingModel:        canvas.BillingModel,
			EntitlementStatus:   canvas.EntitlementStatus,
			EstimatedCostUSD:    canvas.EstimatedCostUSD,
			IncludedUsageUnits:  canvas.IncludedUsageUnits,
			OverageUsageUnits:   canvas.OverageUsageUnits,
			OverageCostUSD:      canvas.OverageCostUSD,
			BlockedCapabilities: append([]string(nil), canvas.BlockedDepartments...),
			HandoffTeam:         canvas.HandoffTeam,
			Recommendation:      canvas.Recommendation(),
		})
	}
	if billingPeriod == "" {
		billingPeriod = portfolio.Period
	}
	return BillingEntitlementsPage{
		WorkspaceName: workspaceName,
		PlanName:      planName,
		BillingPeriod: billingPeriod,
		Charges:       charges,
	}
}

func BuildBillingEntitlementsPageFromLedger(entries []map[string]any, workspaceName, planName, billingPeriod string) BillingEntitlementsPage {
	portfolio := BuildOrchestrationPortfolioFromLedger(entries, workspaceName, billingPeriod)
	return BuildBillingEntitlementsPage(portfolio, workspaceName, planName, billingPeriod)
}

func RenderOrchestrationPortfolioReport(portfolio OrchestrationPortfolio, view *SharedViewContext) string {
	takeoverSummary := "none"
	if portfolio.TakeoverQueue != nil {
		takeoverSummary = fmt.Sprintf("pending=%d recommendation=%s", portfolio.TakeoverQueue.PendingRequests(), portfolio.TakeoverQueue.Recommendation())
	}
	builder := strings.Builder{}
	builder.WriteString("# Orchestration Portfolio Report\n\n")
	builder.WriteString(fmt.Sprintf("- Portfolio: %s\n", portfolio.Name))
	builder.WriteString(fmt.Sprintf("- Period: %s\n", portfolio.Period))
	builder.WriteString(fmt.Sprintf("- Total Runs: %d\n", portfolio.TotalRuns()))
	builder.WriteString(fmt.Sprintf("- Recommendation: %s\n", portfolio.Recommendation()))
	builder.WriteString(fmt.Sprintf("- Collaboration Mix: %s\n", renderSortedCounts(portfolio.CollaborationModes(), " ")))
	builder.WriteString(fmt.Sprintf("- Tier Mix: %s\n", renderSortedCounts(portfolio.TierCounts(), " ")))
	builder.WriteString(fmt.Sprintf("- Entitlement Mix: %s\n", renderSortedCounts(portfolio.EntitlementCounts(), " ")))
	builder.WriteString(fmt.Sprintf("- Billing Models: %s\n", renderSortedCounts(portfolio.BillingModelCounts(), " ")))
	builder.WriteString(fmt.Sprintf("- Upgrade Required Count: %d\n", portfolio.UpgradeRequiredCount()))
	builder.WriteString(fmt.Sprintf("- Estimated Cost (USD): %.2f\n", portfolio.TotalEstimatedCostUSD()))
	builder.WriteString(fmt.Sprintf("- Overage Cost (USD): %.2f\n", portfolio.TotalOverageCostUSD()))
	builder.WriteString(fmt.Sprintf("- Active Handoffs: %d\n", portfolio.ActiveHandoffs()))
	builder.WriteString(fmt.Sprintf("- Takeover Queue: %s\n\n", takeoverSummary))
	builder.WriteString("## Runs\n\n")
	builder.WriteString(renderSharedViewContext(view))
	if len(portfolio.Canvases) == 0 {
		builder.WriteString("- None\n")
		return builder.String() + "\n"
	}
	for _, canvas := range portfolio.Canvases {
		collaborationSummary := "comments=0 decisions=0"
		if canvas.Collaboration != nil {
			collaborationSummary = fmt.Sprintf("comments=%d decisions=%d", len(canvas.Collaboration.Comments), len(canvas.Collaboration.Decisions))
		}
		builder.WriteString(fmt.Sprintf("- %s: mode=%s tier=%s entitlement=%s billing=%s estimated_cost_usd=%.2f overage_cost_usd=%.2f upgrade_required=%t handoff=%s collaboration=%s recommendation=%s actions=%s\n", canvas.RunID, canvas.CollaborationMode, canvas.Tier, canvas.EntitlementStatus, canvas.BillingModel, canvas.EstimatedCostUSD, canvas.OverageCostUSD, canvas.UpgradeRequired, canvas.HandoffTeam, collaborationSummary, canvas.Recommendation(), RenderConsoleActions(actionsOrDefault(canvas))))
	}
	return builder.String() + "\n"
}

func RenderBillingEntitlementsReport(page BillingEntitlementsPage, view *SharedViewContext) string {
	builder := strings.Builder{}
	builder.WriteString("# Billing & Entitlements Report\n\n")
	builder.WriteString(fmt.Sprintf("- Workspace: %s\n", page.WorkspaceName))
	builder.WriteString(fmt.Sprintf("- Plan: %s\n", page.PlanName))
	builder.WriteString(fmt.Sprintf("- Billing Period: %s\n", page.BillingPeriod))
	builder.WriteString(fmt.Sprintf("- Runs: %d\n", page.RunCount()))
	builder.WriteString(fmt.Sprintf("- Recommendation: %s\n", page.Recommendation()))
	builder.WriteString(fmt.Sprintf("- Entitlement Mix: %s\n", renderSortedCounts(page.EntitlementCounts(), " ")))
	builder.WriteString(fmt.Sprintf("- Billing Models: %s\n", renderSortedCounts(page.BillingModelCounts(), " ")))
	builder.WriteString(fmt.Sprintf("- Included Usage Units: %d\n", page.TotalIncludedUsageUnits()))
	builder.WriteString(fmt.Sprintf("- Overage Usage Units: %d\n", page.TotalOverageUsageUnits()))
	builder.WriteString(fmt.Sprintf("- Estimated Cost (USD): %.2f\n", page.TotalEstimatedCostUSD()))
	builder.WriteString(fmt.Sprintf("- Overage Cost (USD): %.2f\n", page.TotalOverageCostUSD()))
	builder.WriteString(fmt.Sprintf("- Upgrade Required Count: %d\n", page.UpgradeRequiredCount()))
	builder.WriteString(fmt.Sprintf("- Blocked Capabilities: %s\n\n", joinOrNone(page.BlockedCapabilities())))
	builder.WriteString("## Charges\n\n")
	builder.WriteString(renderSharedViewContext(view))
	if len(page.Charges) == 0 {
		builder.WriteString("- None\n")
		return builder.String() + "\n"
	}
	for _, charge := range page.Charges {
		builder.WriteString(fmt.Sprintf("- %s: task=%s entitlement=%s billing=%s included_units=%d overage_units=%d estimated_cost_usd=%.2f overage_cost_usd=%.2f blocked=%s handoff=%s recommendation=%s\n", charge.RunID, charge.TaskID, charge.EntitlementStatus, charge.BillingModel, charge.IncludedUsageUnits, charge.OverageUsageUnits, charge.EstimatedCostUSD, charge.OverageCostUSD, joinOrNone(charge.BlockedCapabilities), charge.HandoffTeam, charge.Recommendation))
	}
	return builder.String() + "\n"
}

func RenderOrchestrationOverviewPage(portfolio OrchestrationPortfolio) string {
	builder := strings.Builder{}
	builder.WriteString("<!doctype html>\n<html lang=\"en\">\n<head>\n")
	builder.WriteString("  <meta charset=\"utf-8\">\n")
	builder.WriteString("  <title>Orchestration Overview · " + html.EscapeString(portfolio.Name) + "</title>\n")
	builder.WriteString("</head>\n<body>\n")
	builder.WriteString("  <h1>Orchestration Overview</h1>\n")
	builder.WriteString("  <p>" + html.EscapeString(portfolio.Name) + " · " + html.EscapeString(portfolio.Period) + "</p>\n")
	builder.WriteString("  <div><strong>Estimated Cost</strong> $" + fmt.Sprintf("%.2f", portfolio.TotalEstimatedCostUSD()) + "</div>\n")
	if portfolio.TakeoverQueue != nil {
		builder.WriteString("  <div>" + html.EscapeString(fmt.Sprintf("pending=%d recommendation=%s", portfolio.TakeoverQueue.PendingRequests(), portfolio.TakeoverQueue.Recommendation())) + "</div>\n")
	}
	builder.WriteString("  <ul>\n")
	for _, canvas := range portfolio.Canvases {
		builder.WriteString("    <li><strong>" + html.EscapeString(canvas.RunID) + "</strong> · mode=" + html.EscapeString(canvas.CollaborationMode) + " · tier=" + html.EscapeString(canvas.Tier) + " · entitlement=" + html.EscapeString(canvas.EntitlementStatus) + " · billing=" + html.EscapeString(canvas.BillingModel) + " · cost=$" + fmt.Sprintf("%.2f", canvas.EstimatedCostUSD) + " · handoff=" + html.EscapeString(canvas.HandoffTeam) + " · comments=0 · decisions=0 · recommendation=" + html.EscapeString(canvas.Recommendation()) + " · actions=" + html.EscapeString(RenderConsoleActions(actionsOrDefault(canvas))) + "</li>\n")
	}
	builder.WriteString("  </ul>\n</body>\n</html>\n")
	return builder.String()
}

func RenderBillingEntitlementsPage(page BillingEntitlementsPage) string {
	builder := strings.Builder{}
	builder.WriteString("<!doctype html>\n<html lang=\"en\">\n<head>\n")
	builder.WriteString("  <meta charset=\"utf-8\">\n")
	builder.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	builder.WriteString("  <title>Billing & Entitlements · " + html.EscapeString(page.WorkspaceName) + "</title>\n")
	builder.WriteString("</head>\n<body>\n")
	builder.WriteString("  <main>\n")
	builder.WriteString("    <section>\n")
	builder.WriteString("      <h1>" + html.EscapeString(page.WorkspaceName) + "</h1>\n")
	builder.WriteString("      <p>" + html.EscapeString(page.PlanName) + " plan for " + html.EscapeString(page.BillingPeriod) + ". Recommendation: " + html.EscapeString(page.Recommendation()) + ".</p>\n")
	builder.WriteString("    </section>\n")
	builder.WriteString("    <section>\n      <h2>Charge Feed</h2>\n      <ul>\n")
	for _, charge := range page.Charges {
		builder.WriteString("        <li><strong>" + html.EscapeString(charge.RunID) + "</strong> · task=" + html.EscapeString(charge.TaskID) + " · entitlement=" + html.EscapeString(charge.EntitlementStatus) + " · billing=" + html.EscapeString(charge.BillingModel) + " · included=" + strconv.Itoa(charge.IncludedUsageUnits) + " · overage=" + strconv.Itoa(charge.OverageUsageUnits) + " · cost=$" + fmt.Sprintf("%.2f", charge.EstimatedCostUSD) + " · recommendation=" + html.EscapeString(charge.Recommendation) + "</li>\n")
	}
	builder.WriteString("      </ul>\n    </section>\n  </main>\n</body>\n</html>\n")
	return builder.String()
}

func BuildAutoTriageCenter(runs []AutoTriageRun, name, period string, feedback []TriageFeedbackRecord) AutoTriageCenter {
	findings := make([]TriageFinding, 0, len(runs))
	inbox := make([]TriageInboxItem, 0, len(runs))
	for _, run := range runs {
		if !runRequiresTriage(run) {
			continue
		}
		severity := triageSeverity(run)
		owner := triageOwner(run)
		reason := triageReason(run)
		next := triageNextAction(severity, owner)
		suggestions := buildTriageSuggestions(run, runs, severity, owner, feedback)
		findings = append(findings, TriageFinding{
			RunID:      run.RunID,
			TaskID:     run.TaskID,
			Source:     run.Source,
			Severity:   severity,
			Owner:      owner,
			Status:     run.Status,
			Reason:     reason,
			NextAction: next,
			Actions:    buildAutoTriageActions(run.RunID, severity, owner, run.Status),
		})
		inbox = append(inbox, TriageInboxItem{
			RunID:       run.RunID,
			TaskID:      run.TaskID,
			Source:      run.Source,
			Status:      run.Status,
			Severity:    severity,
			Owner:       owner,
			Summary:     reason,
			SubmittedAt: firstNonEmpty(run.EndedAt, run.StartedAt),
			Suggestions: suggestions,
		})
	}
	severityRank := map[string]int{"critical": 0, "high": 1, "medium": 2}
	sort.Slice(findings, func(i, j int) bool {
		if severityRank[findings[i].Severity] != severityRank[findings[j].Severity] {
			return severityRank[findings[i].Severity] < severityRank[findings[j].Severity]
		}
		if findings[i].Owner != findings[j].Owner {
			return findings[i].Owner < findings[j].Owner
		}
		return findings[i].RunID < findings[j].RunID
	})
	sort.Slice(inbox, func(i, j int) bool {
		if severityRank[inbox[i].Severity] != severityRank[inbox[j].Severity] {
			return severityRank[inbox[i].Severity] < severityRank[inbox[j].Severity]
		}
		if inbox[i].Owner != inbox[j].Owner {
			return inbox[i].Owner < inbox[j].Owner
		}
		return inbox[i].RunID < inbox[j].RunID
	})
	return AutoTriageCenter{Name: name, Period: period, Findings: findings, Inbox: inbox, Feedback: feedback}
}

func RenderAutoTriageCenterReport(center AutoTriageCenter, totalRuns *int, view *SharedViewContext) string {
	severity := center.SeverityCounts()
	owners := center.OwnerCounts()
	feedback := center.FeedbackCounts()
	reportTotalRuns := center.FlaggedRuns()
	if totalRuns != nil {
		reportTotalRuns = *totalRuns
	}
	builder := strings.Builder{}
	builder.WriteString("# Auto Triage Center\n\n")
	builder.WriteString(fmt.Sprintf("- Center: %s\n", center.Name))
	builder.WriteString(fmt.Sprintf("- Period: %s\n", center.Period))
	builder.WriteString(fmt.Sprintf("- Flagged Runs: %d\n", center.FlaggedRuns()))
	builder.WriteString(fmt.Sprintf("- Inbox Size: %d\n", center.InboxSize()))
	builder.WriteString(fmt.Sprintf("- Total Runs: %d\n", reportTotalRuns))
	builder.WriteString(fmt.Sprintf("- Recommendation: %s\n", center.Recommendation()))
	builder.WriteString(fmt.Sprintf("- Severity Mix: critical=%d high=%d medium=%d\n", severity["critical"], severity["high"], severity["medium"]))
	builder.WriteString(fmt.Sprintf("- Owner Mix: security=%d engineering=%d operations=%d\n", owners["security"], owners["engineering"], owners["operations"]))
	builder.WriteString(fmt.Sprintf("- Feedback Loop: accepted=%d rejected=%d pending=%d\n\n", feedback["accepted"], feedback["rejected"], feedback["pending"]))
	builder.WriteString("## Queue\n\n")
	builder.WriteString(renderSharedViewContext(view))
	if len(center.Findings) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, finding := range center.Findings {
			builder.WriteString(fmt.Sprintf("- %s: severity=%s owner=%s status=%s task=%s reason=%s next=%s actions=%s\n", finding.RunID, finding.Severity, finding.Owner, finding.Status, finding.TaskID, finding.Reason, finding.NextAction, RenderConsoleActions(finding.Actions)))
		}
	}
	builder.WriteString("\n## Inbox\n\n")
	if len(center.Inbox) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, item := range center.Inbox {
			suggestionSummary := "none"
			if len(item.Suggestions) > 0 {
				parts := make([]string, 0, len(item.Suggestions))
				for _, suggestion := range item.Suggestions {
					parts = append(parts, fmt.Sprintf("%s(%s, confidence=%.2f)", suggestion.Action, suggestion.FeedbackStatus, suggestion.Confidence))
				}
				suggestionSummary = strings.Join(parts, "; ")
			}
			evidenceSummary := "none"
			if len(item.Suggestions) > 0 {
				var parts []string
				for _, suggestion := range item.Suggestions {
					for _, evidence := range suggestion.Evidence {
						parts = append(parts, fmt.Sprintf("%s:%.2f", evidence.RelatedRunID, evidence.Score))
					}
				}
				if len(parts) > 0 {
					evidenceSummary = strings.Join(parts, ", ")
				}
			}
			builder.WriteString(fmt.Sprintf("- %s: severity=%s owner=%s status=%s summary=%s suggestions=%s similar=%s\n", item.RunID, item.Severity, item.Owner, item.Status, item.Summary, suggestionSummary, evidenceSummary))
		}
	}
	return builder.String() + "\n"
}

func RenderPilotPortfolioReport(portfolio PilotPortfolio) string {
	counts := portfolio.RecommendationCounts()
	builder := strings.Builder{}
	builder.WriteString("# Pilot Portfolio Report\n\n")
	builder.WriteString(fmt.Sprintf("- Portfolio: %s\n", portfolio.Name))
	builder.WriteString(fmt.Sprintf("- Period: %s\n", portfolio.Period))
	builder.WriteString(fmt.Sprintf("- Scorecards: %d\n", len(portfolio.Scorecards)))
	builder.WriteString(fmt.Sprintf("- Recommendation: %s\n", portfolio.Recommendation()))
	builder.WriteString(fmt.Sprintf("- Total Monthly Net Value: %.2f\n", portfolio.TotalMonthlyNetValue()))
	builder.WriteString(fmt.Sprintf("- Average ROI: %.1f%%\n", portfolio.AverageROI()))
	builder.WriteString(fmt.Sprintf("- Recommendation Mix: go=%d iterate=%d hold=%d\n\n", counts["go"], counts["iterate"], counts["hold"]))
	builder.WriteString("## Customers\n\n")
	if len(portfolio.Scorecards) == 0 {
		builder.WriteString("- None\n")
		return builder.String() + "\n"
	}
	for _, scorecard := range portfolio.Scorecards {
		benchmark := "n/a"
		if scorecard.BenchmarkScore != nil {
			benchmark = strconv.Itoa(*scorecard.BenchmarkScore)
		}
		builder.WriteString(fmt.Sprintf("- %s: recommendation=%s roi=%.1f%% monthly-net=%.2f benchmark=%s\n", scorecard.Customer, scorecard.Recommendation(), scorecard.AnnualizedROI(), scorecard.MonthlyNetValue(), benchmark))
	}
	return builder.String() + "\n"
}

func RenderPilotScorecard(scorecard PilotScorecard) string {
	builder := strings.Builder{}
	builder.WriteString("# Pilot Scorecard\n\n")
	builder.WriteString(fmt.Sprintf("- Issue ID: %s\n", scorecard.IssueID))
	builder.WriteString(fmt.Sprintf("- Customer: %s\n", scorecard.Customer))
	builder.WriteString(fmt.Sprintf("- Period: %s\n", scorecard.Period))
	builder.WriteString(fmt.Sprintf("- Recommendation: %s\n", scorecard.Recommendation()))
	builder.WriteString(fmt.Sprintf("- Metrics Met: %d/%d\n", scorecard.MetricsMet(), len(scorecard.Metrics)))
	builder.WriteString(fmt.Sprintf("- Monthly Net Value: %.2f\n", scorecard.MonthlyNetValue()))
	builder.WriteString(fmt.Sprintf("- Annualized ROI: %.1f%%\n", scorecard.AnnualizedROI()))
	if payback := scorecard.PaybackMonths(); payback == nil {
		builder.WriteString("- Payback Months: n/a\n")
	} else {
		builder.WriteString(fmt.Sprintf("- Payback Months: %.1f\n", *payback))
	}
	if scorecard.BenchmarkScore != nil {
		builder.WriteString(fmt.Sprintf("- Benchmark Score: %d\n", *scorecard.BenchmarkScore))
	}
	if scorecard.BenchmarkPassed != nil {
		builder.WriteString(fmt.Sprintf("- Benchmark Passed: %t\n", *scorecard.BenchmarkPassed))
	}
	builder.WriteString("\n## KPI Progress\n\n")
	if len(scorecard.Metrics) == 0 {
		builder.WriteString("- None\n")
		return builder.String() + "\n"
	}
	for _, metric := range scorecard.Metrics {
		comparator := ">="
		if !metric.HigherIsBetter {
			comparator = "<="
		}
		unitSuffix := ""
		if metric.Unit != "" {
			unitSuffix = " " + metric.Unit
		}
		builder.WriteString(fmt.Sprintf("- %s: baseline=%v%s current=%v%s target%s%v%s delta=%+.2f%s met=%t\n", metric.Name, metric.Baseline, unitSuffix, metric.Current, unitSuffix, comparator, metric.Target, unitSuffix, metric.Delta(), unitSuffix, metric.MetTarget()))
	}
	return builder.String() + "\n"
}

func WriteWeeklyOperationsBundle(rootDir string, weekly Weekly, metricSpec *OperationsMetricSpec) (WeeklyArtifacts, error) {
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return WeeklyArtifacts{}, err
	}
	artifacts := WeeklyArtifacts{
		RootDir:          rootDir,
		WeeklyReportPath: filepath.Join(rootDir, "weekly-operations.md"),
		DashboardPath:    filepath.Join(rootDir, "operations-dashboard.md"),
	}
	if err := WriteReport(artifacts.WeeklyReportPath, RenderMarkdown(weekly)); err != nil {
		return WeeklyArtifacts{}, err
	}
	if err := WriteReport(artifacts.DashboardPath, RenderOperationsDashboard(weekly)); err != nil {
		return WeeklyArtifacts{}, err
	}
	if metricSpec != nil {
		artifacts.MetricSpecPath = filepath.Join(rootDir, "operations-metric-spec.md")
		if err := WriteReport(artifacts.MetricSpecPath, RenderOperationsMetricSpec(*metricSpec)); err != nil {
			return WeeklyArtifacts{}, err
		}
	}
	return artifacts, nil
}

func WriteWeeklyOperationsBundleWithVersionCenter(rootDir string, weekly Weekly, metricSpec *OperationsMetricSpec, versionCenter *PolicyPromptVersionCenter) (WeeklyArtifacts, error) {
	return WriteWeeklyOperationsBundleWithCenters(rootDir, weekly, metricSpec, "", nil, nil, versionCenter)
}

func WriteWeeklyOperationsBundleWithCenters(rootDir string, weekly Weekly, metricSpec *OperationsMetricSpec, regressionName string, regressionCenter *regression.Center, queueControl *QueueControlCenter, versionCenter *PolicyPromptVersionCenter) (WeeklyArtifacts, error) {
	artifacts, err := WriteWeeklyOperationsBundle(rootDir, weekly, metricSpec)
	if err != nil {
		return WeeklyArtifacts{}, err
	}
	if regressionCenter != nil {
		artifacts.RegressionCenterPath = filepath.Join(rootDir, "regression-center.md")
		if err := WriteReport(artifacts.RegressionCenterPath, RenderRegressionCenter(regressionName, *regressionCenter)); err != nil {
			return WeeklyArtifacts{}, err
		}
	}
	if queueControl != nil {
		artifacts.QueueControlPath = filepath.Join(rootDir, "queue-control-center.md")
		if err := WriteReport(artifacts.QueueControlPath, RenderQueueControlCenter(*queueControl)); err != nil {
			return WeeklyArtifacts{}, err
		}
	}
	if versionCenter == nil {
		return artifacts, nil
	}
	artifacts.VersionCenterPath = filepath.Join(rootDir, "policy-prompt-version-center.md")
	if err := WriteReport(artifacts.VersionCenterPath, RenderPolicyPromptVersionCenter(*versionCenter)); err != nil {
		return WeeklyArtifacts{}, err
	}
	return artifacts, nil
}

func AuditDashboardBuilder(dashboard DashboardBuilder) DashboardBuilderAudit {
	widgetIndex := dashboard.WidgetIndex()
	placementCounts := make(map[string]int)
	missingWidgetDefs := make(map[string]struct{})
	inaccessibleWidgets := make(map[string]struct{})
	overlappingPlacements := make(map[string]struct{})
	outOfBoundsPlacements := make(map[string]struct{})
	emptyLayouts := make([]string, 0)
	placedWidgets := 0

	for _, layout := range dashboard.Layouts {
		if len(layout.Placements) == 0 {
			emptyLayouts = append(emptyLayouts, layout.LayoutID)
			continue
		}

		placedWidgets += len(layout.Placements)
		for _, placement := range layout.Placements {
			placementCounts[placement.PlacementID]++
			spec, ok := widgetIndex[placement.WidgetID]
			if !ok {
				missingWidgetDefs[placement.WidgetID] = struct{}{}
			} else if !dashboard.Permissions.CanView(spec.Module) {
				inaccessibleWidgets[placement.WidgetID] = struct{}{}
			}
			if placement.Column+placement.Width > layout.Columns {
				outOfBoundsPlacements[placement.PlacementID] = struct{}{}
			}
		}

		for index, placement := range layout.Placements {
			for _, other := range layout.Placements[index+1:] {
				if placementsOverlap(placement, other) {
					key := fmt.Sprintf("%s:%s<->%s", layout.LayoutID, placement.PlacementID, other.PlacementID)
					overlappingPlacements[key] = struct{}{}
				}
			}
		}
	}

	duplicateIDs := make([]string, 0)
	for placementID, count := range placementCounts {
		if count > 1 {
			duplicateIDs = append(duplicateIDs, placementID)
		}
	}
	sort.Strings(duplicateIDs)
	sort.Strings(emptyLayouts)

	return DashboardBuilderAudit{
		Name:                  dashboard.Name,
		TotalWidgets:          len(dashboard.Widgets),
		LayoutCount:           len(dashboard.Layouts),
		PlacedWidgets:         placedWidgets,
		DuplicatePlacementIDs: duplicateIDs,
		MissingWidgetDefs:     sortedKeys(missingWidgetDefs),
		InaccessibleWidgets:   sortedKeys(inaccessibleWidgets),
		OverlappingPlacements: sortedKeys(overlappingPlacements),
		OutOfBoundsPlacements: sortedKeys(outOfBoundsPlacements),
		EmptyLayouts:          emptyLayouts,
		DocumentationComplete: dashboard.DocumentationComplete,
	}
}

func RenderDashboardBuilderReport(dashboard DashboardBuilder, audit DashboardBuilderAudit) string {
	builder := strings.Builder{}
	builder.WriteString("# Dashboard Builder\n\n")
	builder.WriteString(fmt.Sprintf("- Name: %s\n", dashboard.Name))
	builder.WriteString(fmt.Sprintf("- Period: %s\n", dashboard.Period))
	builder.WriteString(fmt.Sprintf("- Owner: %s\n", dashboard.Owner))
	builder.WriteString(fmt.Sprintf("- Viewer Role: %s\n", dashboard.Permissions.ViewerRole))
	builder.WriteString(fmt.Sprintf("- Available Widgets: %d\n", len(dashboard.Widgets)))
	builder.WriteString(fmt.Sprintf("- Layouts: %d\n", len(dashboard.Layouts)))
	builder.WriteString(fmt.Sprintf("- Release Ready: %t\n\n", audit.ReleaseReady()))
	builder.WriteString("## Governance\n\n")
	builder.WriteString(fmt.Sprintf("- Documentation Complete: %t\n", audit.DocumentationComplete))
	builder.WriteString(fmt.Sprintf("- Duplicate Placement IDs: %s\n", joinOrNone(audit.DuplicatePlacementIDs)))
	builder.WriteString(fmt.Sprintf("- Missing Widget Definitions: %s\n", joinOrNone(audit.MissingWidgetDefs)))
	builder.WriteString(fmt.Sprintf("- Inaccessible Widgets: %s\n", joinOrNone(audit.InaccessibleWidgets)))
	builder.WriteString(fmt.Sprintf("- Overlaps: %s\n", joinOrNone(audit.OverlappingPlacements)))
	builder.WriteString(fmt.Sprintf("- Out Of Bounds: %s\n", joinOrNone(audit.OutOfBoundsPlacements)))
	builder.WriteString(fmt.Sprintf("- Empty Layouts: %s\n\n", joinOrNone(audit.EmptyLayouts)))
	builder.WriteString("## Layouts\n\n")

	widgetIndex := dashboard.WidgetIndex()
	if len(dashboard.Layouts) == 0 {
		builder.WriteString("- None\n")
		return builder.String()
	}
	for _, layout := range dashboard.Layouts {
		builder.WriteString(fmt.Sprintf("- %s: name=%s columns=%d placements=%d\n", layout.LayoutID, layout.Name, layout.Columns, len(layout.Placements)))
		for _, placement := range layout.Placements {
			title := placement.TitleOverride
			if strings.TrimSpace(title) == "" {
				if widget, ok := widgetIndex[placement.WidgetID]; ok {
					title = widget.Title
				} else {
					title = placement.WidgetID
				}
			}
			builder.WriteString(fmt.Sprintf("- %s: widget=%s title=%s grid=(%d,%d) size=%dx%d filters=%s\n", placement.PlacementID, placement.WidgetID, title, placement.Column, placement.Row, placement.Width, placement.Height, joinOrNone(placement.Filters)))
		}
	}
	return builder.String()
}

func WriteDashboardBuilderBundle(rootDir string, dashboard DashboardBuilder, audit DashboardBuilderAudit) (string, error) {
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(rootDir, "dashboard-builder.md")
	if err := WriteReport(path, RenderDashboardBuilderReport(dashboard, audit)); err != nil {
		return "", err
	}
	return path, nil
}

func BuildEngineeringOverview(name, period, viewerRole string, tasks []domain.Task, events []domain.Event, slaTargetMinutes, topNBlockers, recentActivityLimit int) EngineeringOverview {
	if slaTargetMinutes <= 0 {
		slaTargetMinutes = 60
	}
	if topNBlockers <= 0 {
		topNBlockers = 3
	}
	if recentActivityLimit <= 0 {
		recentActivityLimit = 5
	}

	statusCounts := make(map[domain.TaskState]int)
	completed := 0
	approvalQueueDepth := 0
	slaBreachCount := 0
	totalCycleMinutes := 0.0
	cycleCount := 0
	blockerGroups := make(map[string]*EngineeringOverviewBlocker)
	recentEvents := latestEventsByTask(events)

	for _, task := range tasks {
		statusCounts[task.State]++
		if task.State == domain.TaskSucceeded {
			completed++
		}
		if strings.EqualFold(task.Metadata["approval_status"], "needs-approval") {
			approvalQueueDepth++
		}
		if minutes, ok := cycleMinutes(task); ok {
			totalCycleMinutes += minutes
			cycleCount++
			if minutes > float64(slaTargetMinutes) {
				slaBreachCount++
			}
		}
		if blockerReason, blocked := blockerReason(task, recentEvents[task.ID]); blocked {
			entry := blockerGroups[blockerReason]
			if entry == nil {
				entry = &EngineeringOverviewBlocker{
					Summary:  blockerReason,
					Owner:    blockerOwner(blockerReason),
					Severity: blockerSeverity(task.State),
				}
				blockerGroups[blockerReason] = entry
			}
			entry.AffectedRuns++
			entry.AffectedTasks = append(entry.AffectedTasks, task.ID)
			if blockerSeverity(task.State) == "high" {
				entry.Severity = "high"
			}
		}
	}

	totalRuns := len(tasks)
	successRate := 0.0
	if totalRuns > 0 {
		successRate = roundTenth((float64(completed) / float64(totalRuns)) * 100)
	}
	averageCycleMinutes := 0.0
	if cycleCount > 0 {
		averageCycleMinutes = roundTenth(totalCycleMinutes / float64(cycleCount))
	}

	overview := EngineeringOverview{
		Name:        name,
		Period:      period,
		Permissions: permissionsForRole(viewerRole),
		KPIs: []EngineeringOverviewKPI{
			{Name: "success-rate", Value: successRate, Target: 90.0, Unit: "%", Direction: "up"},
			{Name: "approval-queue-depth", Value: float64(approvalQueueDepth), Target: 2.0, Direction: "down"},
			{Name: "sla-breaches", Value: float64(slaBreachCount), Target: 0.0, Direction: "down"},
			{Name: "average-cycle-minutes", Value: averageCycleMinutes, Target: float64(slaTargetMinutes), Unit: "m", Direction: "down"},
		},
		Funnel:     buildEngineeringFunnel(statusCounts, totalRuns),
		Blockers:   topBlockers(blockerGroups, topNBlockers),
		Activities: buildRecentActivities(tasks, recentEvents, recentActivityLimit),
	}
	return overview
}

func RenderEngineeringOverview(overview EngineeringOverview) string {
	builder := strings.Builder{}
	builder.WriteString("# Engineering Overview\n\n")
	builder.WriteString(fmt.Sprintf("- Name: %s\n", overview.Name))
	builder.WriteString(fmt.Sprintf("- Period: %s\n", overview.Period))
	builder.WriteString(fmt.Sprintf("- Viewer Role: %s\n", overview.Permissions.ViewerRole))
	builder.WriteString(fmt.Sprintf("- Visible Modules: %s\n", joinOrNone(overview.Permissions.AllowedModules)))

	if overview.Permissions.CanView("kpis") {
		builder.WriteString("\n## KPI Modules\n\n")
		if len(overview.KPIs) == 0 {
			builder.WriteString("- None\n")
		} else {
			for _, kpi := range overview.KPIs {
				builder.WriteString(fmt.Sprintf("- %s: value=%.1f%s target=%.1f%s healthy=%t\n", kpi.Name, kpi.Value, kpi.Unit, kpi.Target, kpi.Unit, kpi.Healthy()))
			}
		}
	}

	if overview.Permissions.CanView("funnel") {
		builder.WriteString("\n## Funnel Modules\n\n")
		if len(overview.Funnel) == 0 {
			builder.WriteString("- None\n")
		} else {
			for _, stage := range overview.Funnel {
				builder.WriteString(fmt.Sprintf("- %s: count=%d share=%.1f%%\n", stage.Name, stage.Count, stage.Share))
			}
		}
	}

	if overview.Permissions.CanView("blockers") {
		builder.WriteString("\n## Blocker Modules\n\n")
		if len(overview.Blockers) == 0 {
			builder.WriteString("- None\n")
		} else {
			for _, blocker := range overview.Blockers {
				builder.WriteString(fmt.Sprintf("- %s: severity=%s owner=%s affected_runs=%d tasks=%s\n", blocker.Summary, blocker.Severity, blocker.Owner, blocker.AffectedRuns, joinOrNone(blocker.AffectedTasks)))
			}
		}
	}

	if overview.Permissions.CanView("activity") {
		builder.WriteString("\n## Activity Modules\n\n")
		if len(overview.Activities) == 0 {
			builder.WriteString("- None\n")
		} else {
			for _, activity := range overview.Activities {
				builder.WriteString(fmt.Sprintf("- %s: %s task=%s status=%s summary=%s\n", activity.Timestamp, firstNonEmpty(activity.RunID, "n/a"), firstNonEmpty(activity.TaskID, "n/a"), activity.Status, activity.Summary))
			}
		}
	}

	return builder.String()
}

func WriteEngineeringOverviewBundle(rootDir string, overview EngineeringOverview) (string, error) {
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(rootDir, "engineering-overview.md")
	if err := WriteReport(path, RenderEngineeringOverview(overview)); err != nil {
		return "", err
	}
	return path, nil
}

func BuildRepoCollaborationMetrics(runs []map[string]any) RepoCollaborationMetrics {
	total := len(runs)
	linked := 0
	accepted := 0
	discussionPosts := 0.0
	lineageDepthSum := 0.0
	lineageDepthCount := 0

	for _, run := range runs {
		if closeout, ok := run["closeout"].(map[string]any); ok {
			if links, ok := closeout["run_commit_links"].([]any); ok && len(links) > 0 {
				linked++
			}
			if strings.TrimSpace(anyString(closeout["accepted_commit_hash"])) != "" {
				accepted++
			}
		}
		discussionPosts += anyFloat(run["repo_discussion_posts"])
		if depth, ok := anyFloatOK(run["accepted_lineage_depth"]); ok {
			lineageDepthSum += depth
			lineageDepthCount++
		}
	}

	metrics := RepoCollaborationMetrics{}
	if total > 0 {
		metrics.RepoLinkCoverage = roundTo((float64(linked)/float64(total))*100, 1)
		metrics.AcceptedCommitRate = roundTo((float64(accepted)/float64(total))*100, 1)
		metrics.DiscussionDensity = roundTo(discussionPosts/float64(total), 2)
	}
	if lineageDepthCount > 0 {
		metrics.AcceptedLineageDepthAvg = roundTo(lineageDepthSum/float64(lineageDepthCount), 2)
	}
	return metrics
}

func BuildTriageClusters(runs []map[string]any) []TriageCluster {
	clusters := make(map[string]*TriageCluster)
	for _, run := range runs {
		status := firstNonEmpty(anyString(run["status"]), "unknown")
		if !isActionableStatus(status) {
			continue
		}

		reason := primaryReason(run)
		cluster, ok := clusters[reason]
		if !ok {
			cluster = &TriageCluster{Reason: reason}
			clusters[reason] = cluster
		}

		if runID := strings.TrimSpace(anyString(run["run_id"])); runID != "" && !containsString(cluster.RunIDs, runID) {
			cluster.RunIDs = append(cluster.RunIDs, runID)
		}
		if taskID := strings.TrimSpace(anyString(run["task_id"])); taskID != "" && !containsString(cluster.TaskIDs, taskID) {
			cluster.TaskIDs = append(cluster.TaskIDs, taskID)
		}
		if !containsString(cluster.Statuses, status) {
			cluster.Statuses = append(cluster.Statuses, status)
		}
	}

	out := make([]TriageCluster, 0, len(clusters))
	for _, cluster := range clusters {
		out = append(out, *cluster)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Occurrences() == out[j].Occurrences() {
			return out[i].Reason < out[j].Reason
		}
		return out[i].Occurrences() > out[j].Occurrences()
	})
	return out
}

func BuildOperationsSnapshot(runs []map[string]any, slaTargetMinutes int, topNBlockers int) OperationsSnapshot {
	if slaTargetMinutes <= 0 {
		slaTargetMinutes = 60
	}
	if topNBlockers <= 0 {
		topNBlockers = 3
	}

	statusCounts := make(map[string]int)
	totalCycleMinutes := 0.0
	cycleCount := 0
	completed := 0
	approvalQueueDepth := 0
	slaBreachCount := 0

	for _, run := range runs {
		status := firstNonEmpty(anyString(run["status"]), "unknown")
		statusCounts[status]++
		if strings.EqualFold(status, "needs-approval") {
			approvalQueueDepth++
		}

		if cycleMinutes, ok := runCycleMinutes(run); ok {
			totalCycleMinutes += cycleMinutes
			cycleCount++
			if cycleMinutes > float64(slaTargetMinutes) {
				slaBreachCount++
			}
		}

		if isCompleteStatus(status) {
			completed++
		}
	}

	successRate := 0.0
	if len(runs) > 0 {
		successRate = roundTo((float64(completed)/float64(len(runs)))*100, 1)
	}
	averageCycleMinutes := 0.0
	if cycleCount > 0 {
		averageCycleMinutes = roundTo(totalCycleMinutes/float64(cycleCount), 1)
	}

	blockers := BuildTriageClusters(runs)
	if topNBlockers > 0 && len(blockers) > topNBlockers {
		blockers = blockers[:topNBlockers]
	}

	return OperationsSnapshot{
		TotalRuns:           len(runs),
		StatusCounts:        statusCounts,
		SuccessRate:         successRate,
		ApprovalQueueDepth:  approvalQueueDepth,
		SLATargetMinutes:    slaTargetMinutes,
		SLABreachCount:      slaBreachCount,
		AverageCycleMinutes: averageCycleMinutes,
		TopBlockers:         blockers,
	}
}

func AnalyzeBenchmarkRegressions(current BenchmarkSuiteResult, baseline BenchmarkSuiteResult) []BenchmarkRegressionFinding {
	baselineResults := make(map[string]BenchmarkCaseResult, len(baseline.Results))
	currentResults := make(map[string]BenchmarkCaseResult, len(current.Results))
	for _, result := range baseline.Results {
		baselineResults[result.CaseID] = result
	}
	for _, result := range current.Results {
		currentResults[result.CaseID] = result
	}

	findings := make([]BenchmarkRegressionFinding, 0)
	for _, comparison := range current.Compare(baseline) {
		baselineResult, baselineOK := baselineResults[comparison.CaseID]
		currentResult := currentResults[comparison.CaseID]
		if comparison.Delta >= 0 && !(baselineOK && baselineResult.Passed && !currentResult.Passed) {
			continue
		}

		severity := "medium"
		if comparison.Delta <= -20 || (baselineOK && baselineResult.Passed && !currentResult.Passed) {
			severity = "high"
		}
		summary := "case regressed from passing to failing"
		if comparison.Delta < 0 {
			summary = fmt.Sprintf("score dropped from %d to %d", comparison.BaselineScore, comparison.CurrentScore)
		}
		findings = append(findings, BenchmarkRegressionFinding{
			CaseID:        comparison.CaseID,
			BaselineScore: comparison.BaselineScore,
			CurrentScore:  comparison.CurrentScore,
			Delta:         comparison.Delta,
			Severity:      severity,
			Summary:       summary,
		})
	}

	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Delta == findings[j].Delta {
			return findings[i].CaseID < findings[j].CaseID
		}
		return findings[i].Delta < findings[j].Delta
	})
	return findings
}

func BuildBenchmarkRegressionCenter(current BenchmarkSuiteResult, baseline BenchmarkSuiteResult, name string) BenchmarkRegressionCenter {
	if strings.TrimSpace(name) == "" {
		name = "Regression Analysis Center"
	}
	comparisons := current.Compare(baseline)
	improvedCases := make([]string, 0)
	unchangedCases := make([]string, 0)
	for _, comparison := range comparisons {
		switch {
		case comparison.Delta > 0:
			improvedCases = append(improvedCases, comparison.CaseID)
		case comparison.Delta == 0:
			unchangedCases = append(unchangedCases, comparison.CaseID)
		}
	}
	sort.Strings(improvedCases)
	sort.Strings(unchangedCases)

	return BenchmarkRegressionCenter{
		Name:            name,
		BaselineVersion: baseline.Version,
		CurrentVersion:  current.Version,
		Regressions:     AnalyzeBenchmarkRegressions(current, baseline),
		ImprovedCases:   improvedCases,
		UnchangedCases:  unchangedCases,
	}
}

func RenderBenchmarkSuiteReport(suite BenchmarkSuiteResult, baseline *BenchmarkSuiteResult) string {
	builder := strings.Builder{}
	builder.WriteString("# Benchmark Suite Report\n\n")
	builder.WriteString(fmt.Sprintf("- Version: %s\n", suite.Version))
	builder.WriteString(fmt.Sprintf("- Cases: %d\n", len(suite.Results)))
	builder.WriteString(fmt.Sprintf("- Passed: %t\n", benchmarkSuitePassed(suite)))
	builder.WriteString(fmt.Sprintf("- Score: %d\n\n", benchmarkSuiteScore(suite)))
	builder.WriteString("## Cases\n\n")
	if len(suite.Results) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, result := range suite.Results {
			builder.WriteString(fmt.Sprintf("- %s: score=%d passed=%t\n", result.CaseID, result.Score, result.Passed))
		}
	}
	builder.WriteString("\n## Comparison\n\n")
	if baseline == nil {
		builder.WriteString("- No baseline provided\n")
		return builder.String()
	}
	builder.WriteString(fmt.Sprintf("- Baseline Version: %s\n", baseline.Version))
	builder.WriteString(fmt.Sprintf("- Score Delta: %d\n", benchmarkSuiteScore(suite)-benchmarkSuiteScore(*baseline)))
	comparisons := suite.Compare(*baseline)
	if len(comparisons) == 0 {
		builder.WriteString("- No comparable cases\n")
		return builder.String()
	}
	for _, comparison := range comparisons {
		builder.WriteString(fmt.Sprintf("- %s: baseline=%d current=%d delta=%d\n", comparison.CaseID, comparison.BaselineScore, comparison.CurrentScore, comparison.Delta))
	}
	return builder.String()
}

func RenderReplayDetailPage(expected, observed BenchmarkReplayRecord, mismatches []string) string {
	builder := strings.Builder{}
	builder.WriteString("<html><head><title>Replay Detail</title></head><body>\n")
	builder.WriteString(fmt.Sprintf("<h1>Replay Detail</h1><p>Task %s replay comparison for run %s.</p>\n", expected.TaskID, expected.RunID))
	builder.WriteString("<h2>Timeline / Log Sync</h2>\n")
	builder.WriteString(fmt.Sprintf("<p>expected medium=%s observed medium=%s</p>\n", expected.Medium, observed.Medium))
	builder.WriteString("<h2>Split View</h2>\n")
	builder.WriteString(fmt.Sprintf("<p>expected status=%s observed status=%s</p>\n", expected.Status, observed.Status))
	builder.WriteString("<h2>Replay</h2>\n<ul>\n")
	if len(mismatches) == 0 {
		builder.WriteString("<li>None</li>\n")
	} else {
		for _, mismatch := range mismatches {
			builder.WriteString("<li>" + mismatch + "</li>\n")
		}
	}
	builder.WriteString("</ul>\n")
	builder.WriteString("<h2>Reports</h2>\n<p>No standalone reports.</p>\n")
	builder.WriteString("</body></html>\n")
	return builder.String()
}

func RenderRunReplayIndexPage(caseID string, record BenchmarkRunIndexRecord, replay BenchmarkReplayOutcome, criteria []BenchmarkCriterion) string {
	reportPath := firstNonEmpty(record.ReportPath, "n/a")
	detailPath := "n/a"
	if strings.TrimSpace(record.ReportPath) != "" {
		detailPath = strings.TrimSuffix(record.ReportPath, filepath.Ext(record.ReportPath)) + ".html"
	}
	replayPath := firstNonEmpty(replay.ReportPath, "n/a")

	builder := strings.Builder{}
	builder.WriteString("<html><head><title>Run Detail Index</title></head><body>\n")
	builder.WriteString(fmt.Sprintf("<h1>Run Detail Index</h1><p>Case %s task %s medium %s.</p>\n", caseID, record.TaskID, record.Medium))
	builder.WriteString("<h2>Timeline / Log Sync</h2>\n")
	for _, criterion := range criteria {
		builder.WriteString(fmt.Sprintf("<div>%s</div>\n", criterion.Name))
	}
	builder.WriteString("<h2>Acceptance</h2>\n<ul>\n")
	if len(criteria) == 0 {
		builder.WriteString("<li>None</li>\n")
	} else {
		for _, criterion := range criteria {
			builder.WriteString(fmt.Sprintf("<li>%s | weight=%d | passed=%t</li>\n", criterion.Name, criterion.Weight, criterion.Passed))
		}
	}
	builder.WriteString("</ul>\n")
	builder.WriteString("<h2>Replay</h2>\n")
	builder.WriteString(fmt.Sprintf("<p>matched=%t run=%s</p>\n", replay.Matched, replay.ReplayRecord.RunID))
	builder.WriteString("<h2>Reports</h2>\n<ul>\n")
	builder.WriteString(fmt.Sprintf("<li>%s</li>\n", reportPath))
	builder.WriteString(fmt.Sprintf("<li>%s</li>\n", detailPath))
	builder.WriteString(fmt.Sprintf("<li>%s</li>\n", replayPath))
	builder.WriteString("</ul>\n")
	builder.WriteString("</body></html>\n")
	return builder.String()
}

func (r BenchmarkRunner) RunCase(item BenchmarkCase) (BenchmarkResult, error) {
	reportPath := ""
	if item.RequireReport {
		reportPath = r.casePath(item.CaseID, "task-run.md")
		if err := writeFileWithDirs(reportPath, "# Benchmark Task Run\n"); err != nil {
			return BenchmarkResult{}, err
		}
	}

	record := benchmarkExecute(item.Task, reportPath)
	criteria := []BenchmarkCriterion{
		benchmarkCriterion("decision-medium", 40, item.ExpectedMedium, record.Medium),
		benchmarkCriterionBool("approval-gate", 30, item.ExpectedApproved, record.Approved),
		benchmarkCriterion("final-status", 20, item.ExpectedStatus, record.Status),
		{
			Name:   "report-artifact",
			Weight: 10,
			Passed: !item.RequireReport || reportPath != "",
			Detail: ternaryString(!item.RequireReport || reportPath != "", "report emitted", "report missing"),
		},
	}
	replay, err := r.Replay(BenchmarkReplayRecord{
		TaskID:   item.Task.ID,
		RunID:    "benchmark-" + item.CaseID,
		Medium:   record.Medium,
		Approved: record.Approved,
		Status:   record.Status,
	})
	if err != nil {
		return BenchmarkResult{}, err
	}

	totalWeight := 0
	earnedWeight := 0
	allPassed := replay.Matched
	for _, criterion := range criteria {
		totalWeight += criterion.Weight
		if criterion.Passed {
			earnedWeight += criterion.Weight
		} else {
			allPassed = false
		}
	}
	score := 0
	if totalWeight > 0 {
		score = int(math.Round(float64(earnedWeight) / float64(totalWeight) * 100))
	}

	detailPagePath := ""
	if strings.TrimSpace(r.StorageDir) != "" {
		detailPagePath = r.casePath(item.CaseID, "run-detail.html")
		page := RenderRunReplayIndexPage(item.CaseID, BenchmarkRunIndexRecord{
			TaskID:     item.Task.ID,
			Medium:     record.Medium,
			Status:     record.Status,
			ReportPath: reportPath,
		}, replay, criteria)
		if err := writeFileWithDirs(detailPagePath, page); err != nil {
			return BenchmarkResult{}, err
		}
	}

	return BenchmarkResult{
		CaseID:         item.CaseID,
		Score:          score,
		Passed:         allPassed,
		Criteria:       criteria,
		Record:         record,
		Replay:         replay,
		DetailPagePath: detailPagePath,
	}, nil
}

func (r BenchmarkRunner) Replay(expected BenchmarkReplayRecord) (BenchmarkReplayOutcome, error) {
	task := domain.Task{ID: expected.TaskID}
	if expected.TaskID == "BIG-601-replay" || expected.TaskID == "BIG-601" || strings.Contains(expected.RunID, "browser-low-risk") {
		task.RequiredTools = []string{"browser"}
	}
	observed := benchmarkExecute(task, "")
	mismatches := make([]string, 0)
	if observed.Medium != expected.Medium {
		mismatches = append(mismatches, fmt.Sprintf("medium expected %s got %s", expected.Medium, observed.Medium))
	}
	if observed.Approved != expected.Approved {
		mismatches = append(mismatches, fmt.Sprintf("approved expected %t got %t", expected.Approved, observed.Approved))
	}
	if observed.Status != expected.Status {
		mismatches = append(mismatches, fmt.Sprintf("status expected %s got %s", expected.Status, observed.Status))
	}

	reportPath := ""
	if strings.TrimSpace(r.StorageDir) != "" {
		reportPath = r.casePath(expected.RunID, "replay.html")
		page := RenderReplayDetailPage(expected, BenchmarkReplayRecord{
			TaskID:   expected.TaskID,
			RunID:    expected.RunID,
			Medium:   observed.Medium,
			Approved: observed.Approved,
			Status:   observed.Status,
		}, mismatches)
		if err := writeFileWithDirs(reportPath, page); err != nil {
			return BenchmarkReplayOutcome{}, err
		}
	}

	return BenchmarkReplayOutcome{
		Matched: len(mismatches) == 0,
		ReplayRecord: BenchmarkReplayRecord{
			TaskID:   expected.TaskID,
			RunID:    expected.RunID,
			Medium:   observed.Medium,
			Approved: observed.Approved,
			Status:   observed.Status,
		},
		Mismatches: mismatches,
		ReportPath: reportPath,
	}, nil
}

func (r BenchmarkRunner) casePath(caseID, fileName string) string {
	if strings.TrimSpace(r.StorageDir) == "" {
		return fileName
	}
	return filepath.Join(r.StorageDir, caseID, fileName)
}

func BuildQueueControlCenter(tasks []domain.Task) QueueControlCenter {
	center := QueueControlCenter{
		QueuedByPriority: map[string]int{"P0": 0, "P1": 0, "P2": 0},
		QueuedByRisk:     map[string]int{"low": 0, "medium": 0, "high": 0},
		ExecutionMedia:   make(map[string]int),
		Actions:          make(map[string][]ConsoleAction),
	}
	for _, task := range tasks {
		if domain.IsActiveTaskState(task.State) {
			center.QueueDepth++
		}
		if task.State == domain.TaskBlocked || strings.EqualFold(task.Metadata["approval_status"], "needs-approval") {
			center.WaitingApprovalRuns++
			center.BlockedTasks = append(center.BlockedTasks, task.ID)
		}
		if task.State == domain.TaskQueued || task.State == domain.TaskLeased || task.State == domain.TaskRetrying {
			center.QueuedTasks = append(center.QueuedTasks, task.ID)
			center.QueuedByPriority[priorityBucket(task.Priority)]++
			center.QueuedByRisk[riskBucket(task.RiskLevel)]++
			medium := firstNonEmpty(string(task.RequiredExecutor), task.Metadata["medium"], "unknown")
			center.ExecutionMedia[medium]++
			center.Actions[task.ID] = buildConsoleActions(task.ID, task.State == domain.TaskBlocked, task.State != domain.TaskBlocked, task.State == domain.TaskBlocked)
		}
	}
	sort.Strings(center.BlockedTasks)
	sort.Strings(center.QueuedTasks)
	return center
}

func RenderQueueControlCenter(center QueueControlCenter) string {
	builder := strings.Builder{}
	builder.WriteString("# Queue Control Center\n\n")
	builder.WriteString(fmt.Sprintf("- Queue Depth: %d\n", center.QueueDepth))
	builder.WriteString(fmt.Sprintf("- Waiting Approval Runs: %d\n", center.WaitingApprovalRuns))
	builder.WriteString(fmt.Sprintf("- Queued Tasks: %s\n\n", joinOrNone(center.QueuedTasks)))
	builder.WriteString("## Queue By Priority\n\n")
	for _, priority := range []string{"P0", "P1", "P2"} {
		builder.WriteString(fmt.Sprintf("- %s: %d\n", priority, center.QueuedByPriority[priority]))
	}
	builder.WriteString("\n## Queue By Risk\n\n")
	for _, risk := range []string{"low", "medium", "high"} {
		builder.WriteString(fmt.Sprintf("- %s: %d\n", risk, center.QueuedByRisk[risk]))
	}
	builder.WriteString("\n## Execution Media\n\n")
	if len(center.ExecutionMedia) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, medium := range sortedMapKeys(center.ExecutionMedia) {
			builder.WriteString(fmt.Sprintf("- %s: %d\n", medium, center.ExecutionMedia[medium]))
		}
	}
	builder.WriteString("\n## Blocked Tasks\n\n")
	if len(center.BlockedTasks) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, taskID := range center.BlockedTasks {
			builder.WriteString(fmt.Sprintf("- %s\n", taskID))
		}
	}
	builder.WriteString("\n## Actions\n\n")
	if len(center.QueuedTasks) == 0 {
		builder.WriteString("- None\n")
		return builder.String()
	}
	for _, taskID := range center.QueuedTasks {
		builder.WriteString(fmt.Sprintf("- %s: %s\n", taskID, RenderConsoleActions(center.Actions[taskID])))
	}
	return builder.String()
}

func WriteQueueControlCenterBundle(rootDir string, center QueueControlCenter) (string, error) {
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(rootDir, "queue-control-center.md")
	if err := WriteReport(path, RenderQueueControlCenter(center)); err != nil {
		return "", err
	}
	return path, nil
}

func BuildPolicyPromptVersionCenter(name string, generatedAt time.Time, artifacts []VersionedArtifact, diffPreviewLines int) PolicyPromptVersionCenter {
	grouped := make(map[string][]VersionedArtifact)
	for _, artifact := range artifacts {
		key := artifact.ArtifactType + "\x00" + artifact.ArtifactID
		grouped[key] = append(grouped[key], artifact)
	}
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	histories := make([]VersionedArtifactHistory, 0, len(keys))
	for _, key := range keys {
		revisions := append([]VersionedArtifact(nil), grouped[key]...)
		sort.SliceStable(revisions, func(i, j int) bool {
			left := parseRFC3339ish(revisions[i].UpdatedAt)
			right := parseRFC3339ish(revisions[j].UpdatedAt)
			if left.Equal(right) {
				if revisions[i].Version == revisions[j].Version {
					return revisions[i].Summary < revisions[j].Summary
				}
				return revisions[i].Version > revisions[j].Version
			}
			return left.After(right)
		})
		current := revisions[0]
		history := VersionedArtifactHistory{
			ArtifactType:     current.ArtifactType,
			ArtifactID:       current.ArtifactID,
			CurrentVersion:   current.Version,
			CurrentUpdatedAt: current.UpdatedAt,
			CurrentAuthor:    current.Author,
			CurrentSummary:   current.Summary,
			RevisionCount:    len(revisions),
			Revisions:        revisions,
		}
		if len(revisions) > 1 {
			previous := revisions[1]
			history.RollbackVersion = previous.Version
			history.RollbackReady = strings.TrimSpace(previous.Content) != ""
			history.ChangeSummary = pointerToChangeSummary(summarizeVersionChange(previous, current, diffPreviewLines))
		}
		histories = append(histories, history)
	}
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	return PolicyPromptVersionCenter{
		Name:        name,
		GeneratedAt: generatedAt.UTC().Format(time.RFC3339),
		Histories:   histories,
	}
}

func RenderPolicyPromptVersionCenter(center PolicyPromptVersionCenter) string {
	return RenderPolicyPromptVersionCenterWithView(center, nil)
}

func RenderPolicyPromptVersionCenterWithView(center PolicyPromptVersionCenter, view *SharedViewContext) string {
	builder := strings.Builder{}
	builder.WriteString("# Policy/Prompt Version Center\n\n")
	builder.WriteString(fmt.Sprintf("- Name: %s\n", center.Name))
	builder.WriteString(fmt.Sprintf("- Generated At: %s\n", center.GeneratedAt))
	builder.WriteString(fmt.Sprintf("- Versioned Artifacts: %d\n", center.ArtifactCount()))
	builder.WriteString(fmt.Sprintf("- Rollback Ready Artifacts: %d\n\n", center.RollbackReadyCount()))
	builder.WriteString("## Artifact Histories\n\n")
	builder.WriteString(renderSharedViewContext(view))
	if len(center.Histories) == 0 {
		builder.WriteString("- None\n")
		return builder.String()
	}
	for _, history := range center.Histories {
		builder.WriteString(fmt.Sprintf("### %s / %s\n\n", history.ArtifactType, history.ArtifactID))
		builder.WriteString(fmt.Sprintf("- Current Version: %s\n", history.CurrentVersion))
		builder.WriteString(fmt.Sprintf("- Updated At: %s\n", history.CurrentUpdatedAt))
		builder.WriteString(fmt.Sprintf("- Updated By: %s\n", history.CurrentAuthor))
		builder.WriteString(fmt.Sprintf("- Summary: %s\n", history.CurrentSummary))
		builder.WriteString(fmt.Sprintf("- Revision Count: %d\n", history.RevisionCount))
		builder.WriteString(fmt.Sprintf("- Rollback Version: %s\n", firstNonEmpty(history.RollbackVersion, "none")))
		builder.WriteString(fmt.Sprintf("- Rollback Ready: %t\n", history.RollbackReady))
		if history.ChangeSummary != nil {
			builder.WriteString(fmt.Sprintf("- Diff Summary: %d additions, %d deletions\n", history.ChangeSummary.Additions, history.ChangeSummary.Deletions))
		}
		builder.WriteString("\n#### Revision History\n\n")
		for _, revision := range history.Revisions {
			builder.WriteString(fmt.Sprintf("- %s: updated_at=%s author=%s ticket=%s summary=%s\n", revision.Version, revision.UpdatedAt, revision.Author, firstNonEmpty(revision.ChangeTicket, "none"), revision.Summary))
		}
		builder.WriteString("\n#### Diff Preview\n\n")
		if history.ChangeSummary != nil && len(history.ChangeSummary.Preview) > 0 {
			builder.WriteString("```diff\n")
			for _, line := range history.ChangeSummary.Preview {
				builder.WriteString(line + "\n")
			}
			builder.WriteString("```\n\n")
			continue
		}
		builder.WriteString("- None\n\n")
	}
	return builder.String()
}

func WritePolicyPromptVersionCenterBundle(rootDir string, center PolicyPromptVersionCenter) (string, error) {
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(rootDir, "policy-prompt-version-center.md")
	if err := WriteReport(path, RenderPolicyPromptVersionCenter(center)); err != nil {
		return "", err
	}
	return path, nil
}

func RenderRegressionCenter(name string, center regression.Center) string {
	return RenderRegressionCenterWithView(name, center, nil)
}

func RenderRegressionCenterWithView(name string, center regression.Center, view *SharedViewContext) string {
	if strings.TrimSpace(name) == "" {
		name = "Regression Analysis Center"
	}
	builder := strings.Builder{}
	builder.WriteString("# Regression Analysis Center\n\n")
	builder.WriteString(fmt.Sprintf("- Name: %s\n", name))
	builder.WriteString(fmt.Sprintf("- Regressions: %d\n", len(center.Findings)))
	builder.WriteString(fmt.Sprintf("- Total Regressions: %d\n", center.Summary.TotalRegressions))
	builder.WriteString(fmt.Sprintf("- Affected Tasks: %d\n", center.Summary.AffectedTasks))
	builder.WriteString(fmt.Sprintf("- Critical Regressions: %d\n", center.Summary.CriticalRegressions))
	builder.WriteString(fmt.Sprintf("- Rework Events: %d\n", center.Summary.ReworkEvents))
	builder.WriteString(fmt.Sprintf("- Top Source: %s\n", firstNonEmpty(center.Summary.TopSource, "none")))
	builder.WriteString(fmt.Sprintf("- Top Workflow: %s\n\n", firstNonEmpty(center.Summary.TopWorkflow, "none")))
	builder.WriteString(renderSharedViewContext(view))

	builder.WriteString("## Findings\n\n")
	if len(center.Findings) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, finding := range center.Findings {
			builder.WriteString(fmt.Sprintf("- %s: severity=%s regressions=%d rework=%d workflow=%s team=%s summary=%s\n", finding.TaskID, finding.Severity, finding.RegressionCount, finding.ReworkEvents, finding.Workflow, finding.Team, finding.Summary))
		}
	}

	builder.WriteString("\n## Hotspots\n\n")
	if len(center.Hotspots) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, hotspot := range center.Hotspots {
			builder.WriteString(fmt.Sprintf("- %s/%s: regressions=%d critical=%d rework=%d\n", hotspot.Dimension, hotspot.Key, hotspot.TotalRegressions, hotspot.CriticalRegressions, hotspot.ReworkEvents))
		}
	}

	builder.WriteString("\n## Workflow Breakdown\n\n")
	if len(center.WorkflowBreakdown) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, item := range center.WorkflowBreakdown {
			builder.WriteString(fmt.Sprintf("- %s: regressions=%d affected_tasks=%d critical=%d rework=%d\n", item.Key, item.TotalRegressions, item.AffectedTasks, item.CriticalRegressions, item.ReworkEvents))
		}
	}

	return builder.String()
}

func WriteRegressionCenterBundle(rootDir, name string, center regression.Center) (string, error) {
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(rootDir, "regression-center.md")
	if err := WriteReport(path, RenderRegressionCenter(name, center)); err != nil {
		return "", err
	}
	return path, nil
}

func buildHighlights(weekly Weekly) []string {
	highlights := []string{
		fmt.Sprintf("Completed %d / %d runs this week.", weekly.Summary.CompletedRuns, weekly.Summary.TotalRuns),
		fmt.Sprintf("Observed %d human interventions across active delivery lanes.", weekly.Summary.HumanInterventions),
	}
	if len(weekly.TeamBreakdown) > 0 {
		highlights = append(highlights, fmt.Sprintf("Top team by throughput: %s.", weekly.TeamBreakdown[0].Key))
	}
	return highlights
}

func buildActions(weekly Weekly) []string {
	actions := make([]string, 0)
	if weekly.Summary.BlockedRuns > 0 {
		actions = append(actions, "Reduce blocked flow count by resolving the top blocker owners first.")
	}
	if weekly.Summary.RegressionFindings > 0 {
		actions = append(actions, "Review regression hotspots and route them through the regression center.")
	}
	if weekly.Summary.HumanInterventions > 0 {
		actions = append(actions, "Audit repeated manual takeovers and convert them into policy or workflow fixes.")
	}
	if len(actions) == 0 {
		actions = append(actions, "No urgent actions detected; maintain current operating cadence.")
	}
	return actions
}

func interventionCounts(events []domain.Event) map[string]int {
	out := make(map[string]int)
	for _, event := range events {
		switch event.Type {
		case domain.EventRunTakeover, domain.EventRunReleased, domain.EventRunAnnotated, domain.EventControlPaused, domain.EventControlResumed:
			if event.TaskID != "" {
				out[event.TaskID]++
			}
		}
	}
	return out
}

func within(anchor time.Time, start time.Time, end time.Time) bool {
	if anchor.IsZero() {
		return false
	}
	if !start.IsZero() && anchor.Before(start) {
		return false
	}
	if !end.IsZero() && anchor.After(end) {
		return false
	}
	return true
}

func regressionCount(task domain.Task) int {
	for _, key := range []string{"regression_count", "regressions"} {
		if value := strings.TrimSpace(task.Metadata[key]); value != "" {
			if parsed, err := strconv.Atoi(value); err == nil {
				return parsed
			}
		}
	}
	if strings.EqualFold(strings.TrimSpace(task.Metadata["regression"]), "true") {
		return 1
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func formatMetricValue(value float64) string {
	if value == float64(int64(value)) {
		return strconv.FormatInt(int64(value), 10)
	}
	return strconv.FormatFloat(value, 'f', 1, 64)
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func sortedMapKeys(values map[string]int) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func buildConsoleActions(target string, allowRetry, allowPause, allowEscalate bool) []ConsoleAction {
	return []ConsoleAction{
		{ActionID: "drill-down", Label: "Drill Down", Target: target, Enabled: true},
		{ActionID: "export", Label: "Export", Target: target, Enabled: true},
		{ActionID: "add-note", Label: "Add Note", Target: target, Enabled: true},
		{ActionID: "escalate", Label: "Escalate", Target: target, Enabled: allowEscalate, Reason: disabledReason(allowEscalate, "escalate is reserved for blocked queue items")},
		{ActionID: "retry", Label: "Retry", Target: target, Enabled: allowRetry, Reason: disabledReason(allowRetry, "retry is reserved for blocked queue items")},
		{ActionID: "pause", Label: "Pause", Target: target, Enabled: allowPause, Reason: disabledReason(allowPause, "approval-blocked tasks should be escalated instead of paused")},
		{ActionID: "audit", Label: "Audit Trail", Target: target, Enabled: true},
	}
}

func RenderConsoleActions(actions []ConsoleAction) string {
	if len(actions) == 0 {
		return "none"
	}
	rendered := make([]string, 0, len(actions))
	for _, action := range actions {
		detail := fmt.Sprintf("%s [%s] state=%s target=%s", action.Label, action.ActionID, action.State(), action.Target)
		if reason := strings.TrimSpace(action.Reason); reason != "" {
			detail += " reason=" + reason
		}
		rendered = append(rendered, detail)
	}
	return strings.Join(rendered, "; ")
}

func actionsOrDefault(canvas OrchestrationCanvas) []ConsoleAction {
	if len(canvas.Actions) > 0 {
		return canvas.Actions
	}
	return buildConsoleActions(
		canvas.RunID,
		canvas.HandoffStatus != "pending",
		canvas.HandoffStatus != "completed",
		canvas.UpgradeRequired,
	)
}

func renderSortedCounts(counts map[string]int, separator string) string {
	if len(counts) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, counts[key]))
	}
	return strings.Join(parts, separator)
}

func buildAutoTriageActions(target, severity, owner, status string) []ConsoleAction {
	allowRetry := severity == "critical" && owner != "security"
	allowPause := status != "failed" && status != "completed" && status != "approved"
	allowReassign := owner != "security"
	return []ConsoleAction{
		{ActionID: "drill-down", Label: "Drill Down", Target: target, Enabled: true},
		{ActionID: "export", Label: "Export", Target: target, Enabled: true},
		{ActionID: "add-note", Label: "Add Note", Target: target, Enabled: true},
		{ActionID: "escalate", Label: "Escalate", Target: target, Enabled: true},
		{ActionID: "retry", Label: "Retry", Target: target, Enabled: allowRetry, Reason: disabledReason(allowRetry, "retry available after owner review")},
		{ActionID: "pause", Label: "Pause", Target: target, Enabled: allowPause, Reason: disabledReason(allowPause, "completed or failed runs cannot be paused")},
		{ActionID: "reassign", Label: "Reassign", Target: target, Enabled: allowReassign, Reason: disabledReason(allowReassign, "security-owned findings stay with the security queue")},
		{ActionID: "audit", Label: "Audit Trail", Target: target, Enabled: true},
	}
}

func runRequiresTriage(run AutoTriageRun) bool {
	if run.Status == "failed" || run.Status == "needs-approval" {
		return true
	}
	for _, trace := range run.Traces {
		if trace.Status == "pending" || trace.Status == "error" || trace.Status == "failed" {
			return true
		}
	}
	for _, audit := range run.Audits {
		if audit.Outcome == "pending" || audit.Outcome == "failed" || audit.Outcome == "rejected" {
			return true
		}
	}
	return false
}

func triageSeverity(run AutoTriageRun) string {
	if run.Status == "failed" {
		return "critical"
	}
	for _, trace := range run.Traces {
		if trace.Status == "error" || trace.Status == "failed" {
			return "critical"
		}
	}
	for _, audit := range run.Audits {
		if audit.Outcome == "failed" || audit.Outcome == "rejected" {
			return "critical"
		}
	}
	if run.Status == "needs-approval" {
		return "high"
	}
	for _, trace := range run.Traces {
		if trace.Status == "pending" {
			return "high"
		}
	}
	for _, audit := range run.Audits {
		if audit.Outcome == "pending" {
			return "high"
		}
	}
	return "medium"
}

func triageOwner(run AutoTriageRun) string {
	var parts []string
	parts = append(parts, run.Summary, run.Title, run.Source, run.Medium)
	for _, trace := range run.Traces {
		parts = append(parts, trace.Status, trace.Span)
	}
	for _, audit := range run.Audits {
		parts = append(parts, audit.Outcome, stringValue(audit.Details["reason"]), fmt.Sprint(audit.Details["approvals"]))
	}
	evidence := strings.ToLower(strings.Join(parts, " "))
	switch {
	case strings.Contains(evidence, "security"), strings.Contains(evidence, "high-risk"), strings.Contains(evidence, "security-review"):
		return "security"
	case run.Medium == "browser":
		return "engineering"
	default:
		for _, artifact := range run.Artifacts {
			if artifact.Kind == "page" {
				return "engineering"
			}
		}
		return "operations"
	}
}

func triageReason(run AutoTriageRun) string {
	for _, audit := range run.Audits {
		if (audit.Outcome == "failed" || audit.Outcome == "rejected" || audit.Outcome == "pending") && stringValue(audit.Details["reason"]) != "" {
			return stringValue(audit.Details["reason"])
		}
	}
	for _, trace := range run.Traces {
		if trace.Status == "error" || trace.Status == "failed" || trace.Status == "pending" {
			return fmt.Sprintf("%s is %s", trace.Span, trace.Status)
		}
	}
	return firstNonEmpty(run.Summary, run.Status)
}

func triageNextAction(severity, owner string) string {
	if severity == "critical" {
		if owner == "engineering" {
			return "replay run and inspect tool failures"
		}
		if owner == "security" {
			return "page security reviewer and block rollout"
		}
		return "open incident review and coordinate response"
	}
	if owner == "security" {
		return "request approval and queue security review"
	}
	if owner == "engineering" {
		return "inspect execution evidence and retry when safe"
	}
	return "confirm owner and clear pending workflow gate"
}

func buildTriageSuggestions(run AutoTriageRun, runs []AutoTriageRun, severity, owner string, feedback []TriageFeedbackRecord) []TriageSuggestion {
	action := triageNextAction(severity, owner)
	evidence := similarityEvidence(run, runs, 2)
	return []TriageSuggestion{{
		Label:          triageSuggestionLabel(run, severity, owner),
		Action:         action,
		Owner:          owner,
		Confidence:     triageSuggestionConfidence(run, evidence),
		Evidence:       evidence,
		FeedbackStatus: feedbackStatus(run.RunID, action, feedback),
	}}
}

func triageSuggestionLabel(run AutoTriageRun, severity, owner string) string {
	if severity == "critical" && owner == "engineering" {
		return "replay candidate"
	}
	if owner == "security" {
		return "approval review"
	}
	if run.Status == "failed" {
		return "incident review"
	}
	return "workflow follow-up"
}

func triageSuggestionConfidence(run AutoTriageRun, evidence []TriageSimilarityEvidence) float64 {
	base := 0.45
	if run.Status == "needs-approval" || run.Status == "failed" {
		base = 0.55
	}
	if len(evidence) > 0 {
		candidate := 0.45 + evidence[0].Score/2
		if candidate > 0.95 {
			candidate = 0.95
		}
		if candidate > base {
			base = candidate
		}
	}
	return math.Round(base*100) / 100
}

func feedbackStatus(runID, action string, feedback []TriageFeedbackRecord) string {
	for i := len(feedback) - 1; i >= 0; i-- {
		if feedback[i].RunID == runID && feedback[i].Action == action {
			return feedback[i].Decision
		}
	}
	return "pending"
}

func similarityEvidence(run AutoTriageRun, runs []AutoTriageRun, limit int) []TriageSimilarityEvidence {
	type scoredMatch struct {
		score float64
		run   AutoTriageRun
	}
	var matches []scoredMatch
	for _, candidate := range runs {
		if candidate.RunID == run.RunID {
			continue
		}
		score := runSimilarityScore(run, candidate)
		if score < 0.35 {
			continue
		}
		matches = append(matches, scoredMatch{score: score, run: candidate})
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].run.RunID < matches[j].run.RunID
	})
	out := make([]TriageSimilarityEvidence, 0, minInt(limit, len(matches)))
	for _, match := range matches[:minInt(limit, len(matches))] {
		out = append(out, TriageSimilarityEvidence{
			RelatedRunID:  match.run.RunID,
			RelatedTaskID: match.run.TaskID,
			Score:         math.Round(match.score*100) / 100,
			Reason:        similarityReason(run, match.run),
		})
	}
	return out
}

func runSimilarityScore(run, candidate AutoTriageRun) float64 {
	haystack := strings.ToLower(strings.Join([]string{
		run.Title,
		run.Summary,
		strings.Join(traceSpans(run.Traces), " "),
		strings.Join(auditOutcomes(run.Audits), " "),
	}, " "))
	needle := strings.ToLower(strings.Join([]string{
		candidate.Title,
		candidate.Summary,
		strings.Join(traceSpans(candidate.Traces), " "),
		strings.Join(auditOutcomes(candidate.Audits), " "),
	}, " "))
	statusBonus := 0.0
	if run.Status == candidate.Status {
		statusBonus = 0.15
	}
	ownerBonus := 0.0
	if triageOwner(run) == triageOwner(candidate) {
		ownerBonus = 0.1
	}
	return math.Min(1.0, difflib.NewMatcher([]string{haystack}, []string{needle}).Ratio()+statusBonus+ownerBonus)
}

func similarityReason(run, candidate AutoTriageRun) string {
	var reasons []string
	if run.Status == candidate.Status {
		reasons = append(reasons, "shared status "+run.Status)
	}
	if triageOwner(run) == triageOwner(candidate) {
		reasons = append(reasons, "shared owner "+triageOwner(run))
	}
	if triageReason(run) == triageReason(candidate) {
		reasons = append(reasons, "matching failure reason")
	}
	if len(reasons) == 0 {
		return "similar execution trail"
	}
	return strings.Join(reasons, ", ")
}

func traceSpans(traces []AutoTriageRunTrace) []string {
	out := make([]string, 0, len(traces))
	for _, trace := range traces {
		out = append(out, trace.Span)
	}
	return out
}

func auditOutcomes(audits []AutoTriageRunAudit) []string {
	out := make([]string, 0, len(audits))
	for _, audit := range audits {
		out = append(out, audit.Outcome)
	}
	return out
}

func disabledReason(enabled bool, reason string) string {
	if enabled {
		return ""
	}
	return reason
}

func priorityBucket(priority int) string {
	switch {
	case priority <= 0:
		return "P0"
	case priority == 1:
		return "P1"
	default:
		return "P2"
	}
}

func riskBucket(level domain.RiskLevel) string {
	switch level {
	case domain.RiskHigh:
		return "high"
	case domain.RiskMedium:
		return "medium"
	default:
		return "low"
	}
}

func riskScoreForTask(task domain.Task) (float64, bool) {
	switch task.RiskLevel {
	case domain.RiskHigh:
		return 90, true
	case domain.RiskMedium:
		return 60, true
	case domain.RiskLow:
		return 25, true
	default:
		return 0, false
	}
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func placementsOverlap(left DashboardWidgetPlacement, right DashboardWidgetPlacement) bool {
	leftRight := left.Column + left.Width
	rightRight := right.Column + right.Width
	leftBottom := left.Row + left.Height
	rightBottom := right.Row + right.Height

	return left.Column < rightRight &&
		leftRight > right.Column &&
		left.Row < rightBottom &&
		leftBottom > right.Row
}

func permissionsForRole(viewerRole string) EngineeringOverviewPermission {
	role := strings.ToLower(strings.TrimSpace(viewerRole))
	if role == "" {
		role = "contributor"
	}
	modulesByRole := map[string][]string{
		"executive":           {"kpis", "funnel", "blockers"},
		"engineering-manager": {"kpis", "funnel", "blockers", "activity"},
		"operations":          {"kpis", "funnel", "blockers", "activity"},
		"contributor":         {"kpis", "activity"},
	}
	modules, ok := modulesByRole[role]
	if !ok {
		modules = modulesByRole["contributor"]
	}
	return EngineeringOverviewPermission{
		ViewerRole:     role,
		AllowedModules: append([]string(nil), modules...),
	}
}

func buildEngineeringFunnel(statusCounts map[domain.TaskState]int, totalRuns int) []EngineeringFunnelStage {
	stages := []EngineeringFunnelStage{
		{Name: "queued", Count: statusCounts[domain.TaskQueued]},
		{Name: "in-progress", Count: statusCounts[domain.TaskRunning] + statusCounts[domain.TaskLeased] + statusCounts[domain.TaskRetrying]},
		{Name: "awaiting-approval", Count: statusCounts[domain.TaskBlocked]},
		{Name: "completed", Count: statusCounts[domain.TaskSucceeded]},
	}
	for index := range stages {
		if totalRuns > 0 {
			stages[index].Share = roundTenth((float64(stages[index].Count) / float64(totalRuns)) * 100)
		}
	}
	return stages
}

func latestEventsByTask(events []domain.Event) map[string]domain.Event {
	out := make(map[string]domain.Event)
	for _, event := range events {
		if event.TaskID == "" {
			continue
		}
		existing, ok := out[event.TaskID]
		if !ok || event.Timestamp.After(existing.Timestamp) {
			out[event.TaskID] = event
		}
	}
	return out
}

func cycleMinutes(task domain.Task) (float64, bool) {
	if task.CreatedAt.IsZero() || task.UpdatedAt.IsZero() || task.UpdatedAt.Before(task.CreatedAt) {
		return 0, false
	}
	return roundTenth(task.UpdatedAt.Sub(task.CreatedAt).Minutes()), true
}

func blockerReason(task domain.Task, event domain.Event) (string, bool) {
	switch task.State {
	case domain.TaskBlocked, domain.TaskFailed, domain.TaskDeadLetter, domain.TaskCancelled:
	default:
		return "", false
	}
	if reason := firstNonEmpty(task.Metadata["blocked_reason"], task.Metadata["failure_reason"], task.Metadata["summary"]); reason != "" {
		return reason, true
	}
	if event.Payload != nil {
		if reason, ok := event.Payload["reason"].(string); ok && strings.TrimSpace(reason) != "" {
			return strings.TrimSpace(reason), true
		}
	}
	if strings.TrimSpace(task.Title) != "" {
		return task.Title, true
	}
	return string(task.State), true
}

func blockerOwner(reason string) string {
	details := strings.ToLower(reason)
	switch {
	case strings.Contains(details, "approval"):
		return "operations"
	case strings.Contains(details, "security"):
		return "security"
	default:
		return "engineering"
	}
}

func blockerSeverity(state domain.TaskState) string {
	switch state {
	case domain.TaskFailed, domain.TaskDeadLetter, domain.TaskCancelled:
		return "high"
	default:
		return "medium"
	}
}

func topBlockers(groups map[string]*EngineeringOverviewBlocker, limit int) []EngineeringOverviewBlocker {
	out := make([]EngineeringOverviewBlocker, 0, len(groups))
	for _, blocker := range groups {
		sort.Strings(blocker.AffectedTasks)
		out = append(out, *blocker)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].AffectedRuns == out[j].AffectedRuns {
			return out[i].Summary < out[j].Summary
		}
		return out[i].AffectedRuns > out[j].AffectedRuns
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func buildRecentActivities(tasks []domain.Task, latest map[string]domain.Event, limit int) []EngineeringActivity {
	sortedTasks := append([]domain.Task(nil), tasks...)
	sort.SliceStable(sortedTasks, func(i, j int) bool {
		if sortedTasks[i].UpdatedAt.Equal(sortedTasks[j].UpdatedAt) {
			return sortedTasks[i].ID < sortedTasks[j].ID
		}
		return sortedTasks[i].UpdatedAt.After(sortedTasks[j].UpdatedAt)
	})
	if limit > 0 && len(sortedTasks) > limit {
		sortedTasks = sortedTasks[:limit]
	}
	out := make([]EngineeringActivity, 0, len(sortedTasks))
	for _, task := range sortedTasks {
		runID := task.Metadata["run_id"]
		if event, ok := latest[task.ID]; ok && strings.TrimSpace(runID) == "" {
			runID = event.RunID
		}
		summary := firstNonEmpty(task.Metadata["summary"], task.Metadata["blocked_reason"], task.Title)
		if event, ok := latest[task.ID]; ok {
			if summary == "" {
				if reason, ok := event.Payload["reason"].(string); ok {
					summary = strings.TrimSpace(reason)
				}
			}
		}
		out = append(out, EngineeringActivity{
			Timestamp: task.UpdatedAt.UTC().Format(time.RFC3339),
			RunID:     runID,
			TaskID:    task.ID,
			Status:    string(task.State),
			Summary:   firstNonEmpty(summary, string(task.State)),
		})
	}
	return out
}

func anyString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func containsString(items []string, needle string) bool {
	for _, item := range items {
		if item == needle {
			return true
		}
	}
	return false
}

func renderSharedViewContext(view *SharedViewContext) string {
	if view == nil {
		return ""
	}
	builder := strings.Builder{}
	builder.WriteString("## View State\n\n")
	builder.WriteString(fmt.Sprintf("- State: %s\n", view.State()))
	builder.WriteString(fmt.Sprintf("- Summary: %s\n", view.Summary()))
	if view.ResultCount != nil {
		builder.WriteString(fmt.Sprintf("- Result Count: %d\n", *view.ResultCount))
	}
	if strings.TrimSpace(view.LastUpdated) != "" {
		builder.WriteString(fmt.Sprintf("- Last Updated: %s\n", view.LastUpdated))
	}
	builder.WriteString("\n## Filters\n\n")
	if len(view.Filters) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, filter := range view.Filters {
			builder.WriteString(fmt.Sprintf("- %s: %s\n", filter.Label, filter.Value))
		}
	}
	if len(view.Errors) > 0 {
		builder.WriteString("\n## Errors\n\n")
		for _, message := range view.Errors {
			builder.WriteString(fmt.Sprintf("- %s\n", message))
		}
	}
	if len(view.PartialData) > 0 {
		builder.WriteString("\n## Partial Data\n\n")
		for _, message := range view.PartialData {
			builder.WriteString(fmt.Sprintf("- %s\n", message))
		}
	}
	builder.WriteString(renderCollaborationLines(view.Collaboration))
	builder.WriteString("\n")
	return builder.String()
}

func renderCollaborationLines(thread *CollaborationThread) string {
	if thread == nil {
		return ""
	}
	builder := strings.Builder{}
	builder.WriteString("\n## Collaboration\n\n")
	builder.WriteString(fmt.Sprintf("- Surface: %s\n", thread.Surface))
	builder.WriteString(fmt.Sprintf("- Target: %s\n", thread.TargetID))
	builder.WriteString(fmt.Sprintf("- Participants: %d\n", thread.ParticipantCount()))
	builder.WriteString(fmt.Sprintf("- Comments: %d\n", len(thread.Comments)))
	builder.WriteString(fmt.Sprintf("- Open Comments: %d\n", thread.OpenCommentCount()))
	builder.WriteString(fmt.Sprintf("- Mentions: %d\n", thread.MentionCount()))
	builder.WriteString(fmt.Sprintf("- Decision Notes: %d\n", len(thread.Decisions)))
	builder.WriteString(fmt.Sprintf("- Recommendation: %s\n", thread.Recommendation()))
	builder.WriteString("\n## Comments\n\n")
	if len(thread.Comments) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, comment := range thread.Comments {
			builder.WriteString(fmt.Sprintf("- %s: author=%s status=%s anchor=%s mentions=%s body=%s\n", comment.CommentID, comment.Author, firstNonEmpty(comment.Status, "open"), firstNonEmpty(comment.Anchor, "none"), joinOrNone(comment.Mentions), comment.Body))
		}
	}
	builder.WriteString("\n## Decision Notes\n\n")
	if len(thread.Decisions) == 0 {
		builder.WriteString("- None\n")
	} else {
		for _, decision := range thread.Decisions {
			builder.WriteString(fmt.Sprintf("- %s: outcome=%s author=%s mentions=%s related=%s summary=%s follow_up=%s\n", decision.DecisionID, decision.Outcome, decision.Author, joinOrNone(decision.Mentions), joinOrNone(decision.RelatedCommentIDs), decision.Summary, firstNonEmpty(decision.FollowUp, "none")))
		}
	}
	builder.WriteString("\n")
	return builder.String()
}

func buildCollaborationThreadFromAudits(audits []map[string]any, surface, targetID string) *CollaborationThread {
	var comments []CollaborationComment
	var decisions []DecisionNote
	for _, audit := range audits {
		details := mapFromAny(audit["details"])
		if firstNonEmpty(stringValue(details["surface"]), "run") != surface {
			continue
		}
		switch stringValue(audit["action"]) {
		case "collaboration.comment":
			comments = append(comments, CollaborationComment{
				CommentID: firstNonEmpty(stringValue(details["comment_id"]), ""),
				Author:    stringValue(audit["actor"]),
				Body:      stringValue(details["body"]),
				CreatedAt: stringValue(audit["timestamp"]),
				Mentions:  stringListFromAny(details["mentions"]),
				Anchor:    stringValue(details["anchor"]),
				Status:    firstNonEmpty(stringValue(details["status"]), "open"),
			})
		case "collaboration.decision":
			decisions = append(decisions, DecisionNote{
				DecisionID:        firstNonEmpty(stringValue(details["decision_id"]), ""),
				Author:            stringValue(audit["actor"]),
				Outcome:           stringValue(audit["outcome"]),
				Summary:           stringValue(details["summary"]),
				RecordedAt:        stringValue(audit["timestamp"]),
				Mentions:          stringListFromAny(details["mentions"]),
				RelatedCommentIDs: stringListFromAny(details["related_comment_ids"]),
				FollowUp:          stringValue(details["follow_up"]),
			})
		}
	}
	if len(comments) == 0 && len(decisions) == 0 {
		return nil
	}
	return &CollaborationThread{
		Surface:   surface,
		TargetID:  targetID,
		Comments:  comments,
		Decisions: decisions,
	}
}

func anyFloat(value any) float64 {
	out, _ := anyFloatOK(value)
	return out
}

func anyFloatOK(value any) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case float32:
		return float64(typed), true
	case float64:
		return typed, true
	default:
		return 0, false
	}
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func primaryReason(run map[string]any) string {
	if audits, ok := run["audits"].([]any); ok {
		for _, item := range audits {
			audit, ok := item.(map[string]any)
			if !ok {
				continue
			}
			details, ok := audit["details"].(map[string]any)
			if !ok {
				continue
			}
			reason := strings.TrimSpace(anyString(details["reason"]))
			if reason != "" {
				return reason
			}
		}
	}

	if summary := strings.TrimSpace(anyString(run["summary"])); summary != "" {
		return summary
	}
	return firstNonEmpty(anyString(run["status"]), "unknown")
}

func runCycleMinutes(run map[string]any) (float64, bool) {
	startedAt := strings.TrimSpace(anyString(run["started_at"]))
	endedAt := strings.TrimSpace(anyString(run["ended_at"]))
	if startedAt == "" || endedAt == "" {
		return 0, false
	}
	start := parseRFC3339ish(startedAt)
	end := parseRFC3339ish(endedAt)
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0, false
	}
	return roundTo(end.Sub(start).Minutes(), 1), true
}

func isCompleteStatus(status string) bool {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "approved", "accepted", "completed", "succeeded":
		return true
	default:
		return false
	}
}

func isActionableStatus(status string) bool {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "needs-approval", "failed", "rejected":
		return true
	default:
		return false
	}
}

func benchmarkSuiteScore(suite BenchmarkSuiteResult) int {
	if len(suite.Results) == 0 {
		return 0
	}
	total := 0
	for _, result := range suite.Results {
		total += result.Score
	}
	return int(math.Round(float64(total) / float64(len(suite.Results))))
}

func benchmarkSuitePassed(suite BenchmarkSuiteResult) bool {
	for _, result := range suite.Results {
		if !result.Passed {
			return false
		}
	}
	return true
}

func benchmarkExecute(task domain.Task, reportPath string) BenchmarkResultRecord {
	medium := "docker"
	approved := true
	status := "approved"
	for _, tool := range task.RequiredTools {
		if strings.EqualFold(strings.TrimSpace(tool), "browser") {
			medium = "browser"
			break
		}
	}
	if task.RiskLevel == domain.RiskHigh {
		medium = "vm"
		approved = false
		status = "needs-approval"
	}
	return BenchmarkResultRecord{
		Medium:     medium,
		Approved:   approved,
		Status:     status,
		ReportPath: reportPath,
	}
}

func benchmarkCriterion(name string, weight int, expected string, actual string) BenchmarkCriterion {
	if strings.TrimSpace(expected) == "" {
		return BenchmarkCriterion{Name: name, Weight: weight, Passed: true, Detail: "not asserted"}
	}
	return BenchmarkCriterion{
		Name:   name,
		Weight: weight,
		Passed: expected == actual,
		Detail: fmt.Sprintf("expected %s got %s", expected, actual),
	}
}

func benchmarkCriterionBool(name string, weight int, expected *bool, actual bool) BenchmarkCriterion {
	if expected == nil {
		return BenchmarkCriterion{Name: name, Weight: weight, Passed: true, Detail: "not asserted"}
	}
	return BenchmarkCriterion{
		Name:   name,
		Weight: weight,
		Passed: *expected == actual,
		Detail: fmt.Sprintf("expected %t got %t", *expected, actual),
	}
}

func writeFileWithDirs(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func latestNamedAudit(audits []map[string]any, action string) map[string]any {
	for i := len(audits) - 1; i >= 0; i-- {
		if stringValue(audits[i]["action"]) == action {
			return audits[i]
		}
	}
	return nil
}

func latestHandoffAudit(audits []map[string]any) map[string]any {
	for i := len(audits) - 1; i >= 0; i-- {
		action := stringValue(audits[i]["action"])
		if action == "orchestration.handoff" || action == "execution.manual_takeover" || action == "execution.flow_handoff" {
			return audits[i]
		}
	}
	return nil
}

func mapsFromAny(value any) []map[string]any {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, mapFromAny(item))
	}
	return out
}

func mapFromAny(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func stringValue(value any) string {
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}

func stringListFromAny(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok {
			out = append(out, text)
		}
	}
	return out
}

func ternaryString(condition bool, whenTrue, whenFalse string) string {
	if condition {
		return whenTrue
	}
	return whenFalse
}

func roundTo(value float64, places int) float64 {
	if places < 0 {
		return value
	}
	factor := math.Pow10(places)
	return math.Round(value*factor) / factor
}

func roundTenth(value float64) float64 {
	return math.Round(value*10) / 10
}

func parseRFC3339ish(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err == nil {
		return parsed.UTC()
	}
	return time.Time{}
}

func summarizeVersionChange(previous, current VersionedArtifact, previewLines int) VersionChangeSummary {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(previous.Content),
		B:        difflib.SplitLines(current.Content),
		FromFile: previous.Version,
		ToFile:   current.Version,
		Context:  1,
	}
	text, _ := difflib.GetUnifiedDiffString(diff)
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	additions := 0
	deletions := 0
	preview := make([]string, 0, previewLines)
	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			additions++
		}
		if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			deletions++
		}
		if strings.HasPrefix(line, "@@") {
			continue
		}
		if len(preview) < previewLines {
			preview = append(preview, line)
		}
	}
	return VersionChangeSummary{
		FromVersion:  previous.Version,
		ToVersion:    current.Version,
		Additions:    additions,
		Deletions:    deletions,
		ChangedLines: additions + deletions,
		Preview:      preview,
	}
}

func pointerToChangeSummary(summary VersionChangeSummary) *VersionChangeSummary {
	return &summary
}
