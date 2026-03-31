package planningcompat

import (
	"fmt"
	"slices"
	"strings"
)

type EvidenceLink struct {
	Label      string
	Target     string
	Capability string
	Note       string
}

type CandidateEntry struct {
	CandidateID       string
	Title             string
	Theme             string
	Priority          string
	Owner             string
	Outcome           string
	ValidationCommand string
	Capabilities      []string
	Evidence          []string
	EvidenceLinks     []EvidenceLink
	Blockers          []string
}

func (c CandidateEntry) Ready() bool {
	return len(c.Blockers) == 0
}

func (c CandidateEntry) Score() int {
	if c.Ready() {
		return 100
	}
	return 50
}

type CandidateBacklog struct {
	EpicID     string
	Title      string
	Version    string
	Candidates []CandidateEntry
}

func (b CandidateBacklog) RankedCandidates() []CandidateEntry {
	out := append([]CandidateEntry(nil), b.Candidates...)
	slices.SortFunc(out, func(a, c CandidateEntry) int {
		if a.Ready() != c.Ready() {
			if a.Ready() {
				return -1
			}
			return 1
		}
		if a.Priority != c.Priority {
			return strings.Compare(a.Priority, c.Priority)
		}
		return strings.Compare(a.CandidateID, c.CandidateID)
	})
	return out
}

type ScopeFreezeAudit struct {
	BoardName         string
	Version           string
	TotalItems        int
	MissingValidation []string
}

func (a ScopeFreezeAudit) Ready() bool {
	return len(a.MissingValidation) == 0
}

func (a ScopeFreezeAudit) Score() float64 {
	score := 100.0 - float64(len(a.MissingValidation))*12.5
	if score < 0 {
		return 0
	}
	return score
}

type EntryGate struct {
	GateID                  string
	Name                    string
	MinReadyCandidates      int
	RequiredCapabilities    []string
	RequiredEvidence        []string
	RequiredBaselineVersion string
}

type EntryGateDecision struct {
	GateID              string
	Passed              bool
	ReadyCandidateIDs   []string
	BlockedCandidateIDs []string
	MissingCapabilities []string
	MissingEvidence     []string
	BaselineReady       bool
	BaselineFindings    []string
	BlockerCount        int
}

type CandidatePlanner struct{}

func (p CandidatePlanner) EvaluateGate(backlog CandidateBacklog, gate EntryGate, baseline *ScopeFreezeAudit) EntryGateDecision {
	decision := EntryGateDecision{GateID: gate.GateID}
	capabilities := map[string]struct{}{}
	evidence := map[string]struct{}{}
	for _, candidate := range backlog.RankedCandidates() {
		if candidate.Ready() {
			decision.ReadyCandidateIDs = append(decision.ReadyCandidateIDs, candidate.CandidateID)
			for _, item := range candidate.Capabilities {
				capabilities[item] = struct{}{}
			}
			for _, item := range candidate.Evidence {
				evidence[item] = struct{}{}
			}
		} else {
			decision.BlockedCandidateIDs = append(decision.BlockedCandidateIDs, candidate.CandidateID)
			decision.BlockerCount += len(candidate.Blockers)
		}
	}
	for _, item := range gate.RequiredCapabilities {
		if _, ok := capabilities[item]; !ok {
			decision.MissingCapabilities = append(decision.MissingCapabilities, item)
		}
	}
	for _, item := range gate.RequiredEvidence {
		if _, ok := evidence[item]; !ok {
			decision.MissingEvidence = append(decision.MissingEvidence, item)
		}
	}
	if baseline == nil {
		decision.BaselineReady = false
		decision.BaselineFindings = []string{fmt.Sprintf("missing baseline audit for %s", gate.RequiredBaselineVersion)}
	} else if !baseline.Ready() {
		decision.BaselineReady = false
		decision.BaselineFindings = []string{fmt.Sprintf("baseline %s is not release ready (%.1f)", gate.RequiredBaselineVersion, baseline.Score())}
	} else {
		decision.BaselineReady = true
	}
	decision.Passed = len(decision.ReadyCandidateIDs) >= gate.MinReadyCandidates &&
		len(decision.MissingCapabilities) == 0 &&
		len(decision.MissingEvidence) == 0 &&
		decision.BaselineReady
	return decision
}

func RenderCandidateBacklogReport(backlog CandidateBacklog, gate EntryGate, decision EntryGateDecision) string {
	lines := []string{
		"# V3 Candidate Backlog Report",
		fmt.Sprintf("- Epic: %s %s", backlog.EpicID, backlog.Title),
		fmt.Sprintf("- Decision: %s: ready=%d blocked=%d missing_capabilities=%d missing_evidence=%d baseline_findings=%d", passLabel(decision.Passed), len(decision.ReadyCandidateIDs), len(decision.BlockedCandidateIDs), len(decision.MissingCapabilities), len(decision.MissingEvidence), len(decision.BaselineFindings)),
	}
	for _, candidate := range backlog.RankedCandidates() {
		lines = append(lines, fmt.Sprintf("- %s: %s priority=%s owner=%s score=%d ready=%t", candidate.CandidateID, candidate.Title, candidate.Priority, candidate.Owner, candidate.Score(), candidate.Ready()))
		lines = append(lines, "validation="+candidate.ValidationCommand)
		for _, link := range candidate.EvidenceLinks {
			lines = append(lines, fmt.Sprintf("- %s -> %s capability=%s", link.Label, link.Target, link.Capability))
		}
	}
	lines = append(lines, "- Missing evidence: "+noneOrJoin(decision.MissingEvidence))
	lines = append(lines, fmt.Sprintf("- Baseline ready: %t", decision.BaselineReady))
	lines = append(lines, "- Baseline findings: "+noneOrJoin(decision.BaselineFindings))
	return strings.Join(lines, "\n")
}

type WeeklyGoal struct {
	GoalID        string
	Title         string
	Owner         string
	Status        string
	SuccessMetric string
	TargetValue   string
}

type WeeklyExecutionPlan struct {
	WeekNumber int
	Theme      string
	Objective  string
	Goals      []WeeklyGoal
}

func (w WeeklyExecutionPlan) AtRiskGoalIDs() []string {
	var out []string
	for _, goal := range w.Goals {
		if goal.Status == "at-risk" || goal.Status == "blocked" {
			out = append(out, goal.GoalID)
		}
	}
	return out
}

func (w WeeklyExecutionPlan) CompletedGoals() int {
	count := 0
	for _, goal := range w.Goals {
		if goal.Status == "done" {
			count++
		}
	}
	return count
}

type FourWeekExecutionPlan struct {
	PlanID    string
	Title     string
	Owner     string
	StartDate string
	Weeks     []WeeklyExecutionPlan
}

func (p FourWeekExecutionPlan) Validate() error {
	if len(p.Weeks) != 4 {
		return fmt.Errorf("Four-week execution plans must include weeks 1 through 4 in order")
	}
	for index, week := range p.Weeks {
		if week.WeekNumber != index+1 {
			return fmt.Errorf("Four-week execution plans must include weeks 1 through 4 in order")
		}
	}
	return nil
}

func (p FourWeekExecutionPlan) TotalGoals() int {
	total := 0
	for _, week := range p.Weeks {
		total += len(week.Goals)
	}
	return total
}

func (p FourWeekExecutionPlan) CompletedGoals() int {
	total := 0
	for _, week := range p.Weeks {
		total += week.CompletedGoals()
	}
	return total
}

func (p FourWeekExecutionPlan) OverallProgressPercent() int {
	if p.TotalGoals() == 0 {
		return 0
	}
	return (p.CompletedGoals() * 100) / p.TotalGoals()
}

func (p FourWeekExecutionPlan) AtRiskWeeks() []int {
	var out []int
	for _, week := range p.Weeks {
		if len(week.AtRiskGoalIDs()) > 0 {
			out = append(out, week.WeekNumber)
		}
	}
	return out
}

func (p FourWeekExecutionPlan) GoalStatusCounts() map[string]int {
	counts := map[string]int{"done": 0, "on-track": 0, "at-risk": 0, "not-started": 0}
	for _, week := range p.Weeks {
		for _, goal := range week.Goals {
			status := goal.Status
			if status == "blocked" {
				status = "at-risk"
			}
			counts[status]++
		}
	}
	return counts
}

func RenderFourWeekExecutionReport(plan FourWeekExecutionPlan) string {
	lines := []string{
		"# Four-Week Execution Plan",
		fmt.Sprintf("- Plan: %s %s", plan.PlanID, plan.Title),
		fmt.Sprintf("- Overall progress: %d/%d goals complete (%d%%)", plan.CompletedGoals(), plan.TotalGoals(), plan.OverallProgressPercent()),
		fmt.Sprintf("- At-risk weeks: %s", intsToString(plan.AtRiskWeeks())),
	}
	for _, week := range plan.Weeks {
		lines = append(lines, fmt.Sprintf("- Week %d: %s progress=%d/%d (%d%%)", week.WeekNumber, week.Theme, week.CompletedGoals(), len(week.Goals), percent(week.CompletedGoals(), len(week.Goals))))
		for _, goal := range week.Goals {
			lines = append(lines, fmt.Sprintf("- %s: %s owner=%s status=%s", goal.GoalID, goal.Title, goal.Owner, goal.Status))
		}
	}
	return strings.Join(lines, "\n")
}

func BuildV3CandidateBacklog() CandidateBacklog {
	return CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3候选与进入条件",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{
				CandidateID: "candidate-ops-hardening", Title: "Operations command-center hardening", Theme: "ops-command-center", Priority: "P0", Owner: "ops-platform",
				Outcome: "Package command-center and approval surfaces with linked evidence.", ValidationCommand: "python3 -m pytest tests/test_operations.py tests/test_saved_views.py -q",
				Capabilities: []string{"ops-control", "commercialization"}, Evidence: []string{"weekly-review", "validation-report", "pilot-evidence"},
				EvidenceLinks: []EvidenceLink{
					{Label: "command-center-src", Target: "src/bigclaw/operations.py", Capability: "ops-control"},
					{Label: "command-center-tests", Target: "tests/test_control_center.py", Capability: "ops-control"},
					{Label: "operations-tests", Target: "tests/test_operations.py", Capability: "ops-control"},
					{Label: "execution-contract-src", Target: "src/bigclaw/execution_contract.py", Capability: "ops-control"},
					{Label: "workflow-src", Target: "src/bigclaw/workflow.py", Capability: "ops-control"},
					{Label: "workflow-go-tests", Target: "bigclaw-go/internal/workflow/engine_test.go", Capability: "ops-control"},
					{Label: "runtime-go-tests", Target: "bigclaw-go/internal/worker/runtime_test.go", Capability: "ops-control"},
					{Label: "saved-view-src", Target: "src/bigclaw/saved_views.py", Capability: "saved-views"},
					{Label: "saved-view-tests", Target: "tests/test_saved_views.py", Capability: "saved-views"},
					{Label: "evaluation-src", Target: "src/bigclaw/evaluation.py", Capability: "commercialization"},
					{Label: "evaluation-tests", Target: "tests/test_evaluation.py", Capability: "commercialization"},
				},
			},
			{
				CandidateID: "candidate-orchestration-rollout", Title: "Orchestration rollout", Theme: "agent-orchestration", Priority: "P0", Owner: "orchestration",
				Outcome: "Promote cross-team orchestration with commercialization visibility.", ValidationCommand: "python3 -m pytest tests/test_orchestration.py -q",
				Capabilities: []string{"commercialization", "handoff"}, Evidence: []string{"pilot-evidence", "acceptance-suite"},
				EvidenceLinks: []EvidenceLink{{Label: "report-studio-tests", Target: "tests/test_reports.py", Capability: "commercialization"}},
			},
			{
				CandidateID: "candidate-release-control", Title: "Release control center", Theme: "console-governance", Priority: "P1", Owner: "platform-ui",
				Outcome: "Unify console release gates and promotion evidence.", ValidationCommand: "python3 -m pytest tests/test_design_system.py -q",
				Capabilities: []string{"release-gate", "reporting"}, Evidence: []string{"acceptance-suite", "validation-report"},
				EvidenceLinks: []EvidenceLink{{Label: "ui-acceptance", Target: "tests/test_design_system.py", Capability: "release-gate"}},
			},
		},
	}
}

func BuildV3EntryGate() EntryGate {
	return EntryGate{
		GateID: "gate-v3-entry", Name: "V3 Entry Gate", MinReadyCandidates: 3,
		RequiredCapabilities:    []string{"release-gate", "ops-control", "commercialization"},
		RequiredEvidence:        []string{"acceptance-suite", "pilot-evidence", "validation-report"},
		RequiredBaselineVersion: "v2.0",
	}
}

func BuildBIG4701ExecutionPlan() FourWeekExecutionPlan {
	return FourWeekExecutionPlan{
		PlanID: "BIG-4701", Title: "4周执行计划与周目标", Owner: "execution-office", StartDate: "2026-03-11",
		Weeks: []WeeklyExecutionPlan{
			{WeekNumber: 1, Theme: "Discovery and framing", Objective: "Lock scope and evidence", Goals: []WeeklyGoal{
				{GoalID: "w1-scope", Title: "Freeze v3 candidate scope", Owner: "execution-office", Status: "done", SuccessMetric: "scope freeze", TargetValue: "1"},
				{GoalID: "w1-evidence", Title: "Confirm baseline evidence", Owner: "platform-ui", Status: "done", SuccessMetric: "evidence pack", TargetValue: "1"},
			}},
			{WeekNumber: 2, Theme: "Build and integration", Objective: "Land high-risk integration work.", Goals: []WeeklyGoal{
				{GoalID: "w2-handoff-sync", Title: "Resolve orchestration and console handoff dependencies", Owner: "orchestration-office", Status: "at-risk", SuccessMetric: "closed dependencies", TargetValue: "0"},
				{GoalID: "w2-ops-landing", Title: "Land operations command center integration", Owner: "ops-platform", Status: "not-started", SuccessMetric: "merged PRs", TargetValue: "2"},
			}},
			{WeekNumber: 3, Theme: "Validation and launch prep", Objective: "Convert integration work into gate-ready evidence.", Goals: []WeeklyGoal{
				{GoalID: "w3-validation", Title: "Run cross-surface validation", Owner: "quality", Status: "on-track", SuccessMetric: "passing suites", TargetValue: "3"},
				{GoalID: "w3-launch", Title: "Draft launch and release notes", Owner: "platform-ui", Status: "not-started", SuccessMetric: "artifacts", TargetValue: "2"},
			}},
			{WeekNumber: 4, Theme: "Decision and rollout", Objective: "Package the decision-ready rollout slice.", Goals: []WeeklyGoal{
				{GoalID: "w4-gate", Title: "Run entry gate review", Owner: "execution-office", Status: "not-started", SuccessMetric: "review", TargetValue: "1"},
				{GoalID: "w4-closeout", Title: "Close launch blockers", Owner: "ops-platform", Status: "not-started", SuccessMetric: "open blockers", TargetValue: "0"},
			}},
		},
	}
}

func passLabel(passed bool) string {
	if passed {
		return "PASS"
	}
	return "FAIL"
}

func noneOrJoin(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

func intsToString(items []int) string {
	if len(items) == 0 {
		return "none"
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, fmt.Sprintf("%d", item))
	}
	return strings.Join(out, ", ")
}

func percent(part, total int) int {
	if total == 0 {
		return 0
	}
	return (part * 100) / total
}
