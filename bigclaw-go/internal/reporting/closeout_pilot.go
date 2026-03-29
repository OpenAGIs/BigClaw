package reporting

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	if m.HigherIsBetter {
		return m.Current >= m.Target
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
	count := 0
	for _, item := range c.Items {
		if c.ItemCompleted(item) {
			count++
		}
	}
	return count
}

func (c LaunchChecklist) MissingDocumentation() []string {
	out := make([]string, 0)
	for _, artifact := range c.Documentation {
		if !artifact.Available() {
			out = append(out, artifact.Name)
		}
	}
	return out
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
	out := make([]string, 0)
	for _, artifact := range c.RequiredOutputs {
		if !artifact.Available() {
			out = append(out, artifact.Name)
		}
	}
	return out
}

func (c FinalDeliveryChecklist) MissingRecommendedDocumentation() []string {
	out := make([]string, 0)
	for _, artifact := range c.RecommendedDocumentation {
		if !artifact.Available() {
			out = append(out, artifact.Name)
		}
	}
	return out
}

func (c FinalDeliveryChecklist) Ready() bool {
	return len(c.MissingRequiredOutputs()) == 0
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
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func RenderIssueValidationReport(issueID, version, environment, summary string) string {
	return fmt.Sprintf("# Issue Validation Report\n\n- Issue ID: %s\n- 版本号: %s\n- 测试环境: %s\n- 生成时间: %s\n\n## 结论\n\n%s\n", issueID, version, environment, time.Now().UTC().Format(time.RFC3339), summary)
}

func BuildLaunchChecklist(issueID string, documentation []DocumentationArtifact, items []LaunchChecklistItem) LaunchChecklist {
	return LaunchChecklist{IssueID: issueID, Documentation: documentation, Items: items}
}

func BuildFinalDeliveryChecklist(issueID string, requiredOutputs []DocumentationArtifact, recommendedDocumentation []DocumentationArtifact) FinalDeliveryChecklist {
	return FinalDeliveryChecklist{IssueID: issueID, RequiredOutputs: requiredOutputs, RecommendedDocumentation: recommendedDocumentation}
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
			evidence := "none"
			if len(item.Evidence) > 0 {
				evidence = strings.Join(item.Evidence, ", ")
			}
			lines = append(lines, fmt.Sprintf("- %s: completed=%t evidence=%s", item.Name, checklist.ItemCompleted(item), evidence))
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

func EvaluateIssueClosure(issueID string, reportPath string, validationPassed bool, launchChecklist *LaunchChecklist, finalDeliveryChecklist *FinalDeliveryChecklist) IssueClosureDecision {
	resolvedPath := ""
	if strings.TrimSpace(reportPath) != "" {
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

func trimFloat(value float64) any {
	if value == float64(int64(value)) {
		return int64(value)
	}
	return value
}
