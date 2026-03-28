package planning

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"bigclaw-go/internal/governance"
)

type EvidenceLink struct {
	Label      string `json:"label"`
	Target     string `json:"target"`
	Capability string `json:"capability,omitempty"`
	Note       string `json:"note,omitempty"`
}

type CandidateEntry struct {
	CandidateID       string         `json:"candidate_id"`
	Title             string         `json:"title"`
	Theme             string         `json:"theme"`
	Priority          string         `json:"priority"`
	Owner             string         `json:"owner"`
	Outcome           string         `json:"outcome"`
	ValidationCommand string         `json:"validation_command"`
	Capabilities      []string       `json:"capabilities,omitempty"`
	Evidence          []string       `json:"evidence,omitempty"`
	EvidenceLinks     []EvidenceLink `json:"evidence_links,omitempty"`
	Blockers          []string       `json:"blockers,omitempty"`
}

func (c CandidateEntry) Ready() bool {
	return len(c.Blockers) == 0
}

func (c CandidateEntry) Score() int {
	score := len(c.Capabilities)*25 + len(c.Evidence)*15 + len(c.EvidenceLinks)*10
	if c.Ready() {
		score += 20
	}
	score -= len(c.Blockers) * 25
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

type CandidateBacklog struct {
	EpicID     string           `json:"epic_id"`
	Title      string           `json:"title"`
	Version    string           `json:"version"`
	Candidates []CandidateEntry `json:"candidates,omitempty"`
}

func (b CandidateBacklog) RankedCandidates() []CandidateEntry {
	ranked := append([]CandidateEntry(nil), b.Candidates...)
	sort.SliceStable(ranked, func(i, j int) bool {
		left, right := ranked[i], ranked[j]
		if left.Ready() != right.Ready() {
			return left.Ready()
		}
		if left.Score() != right.Score() {
			return left.Score() > right.Score()
		}
		return left.CandidateID < right.CandidateID
	})
	return ranked
}

type EntryGate struct {
	GateID                  string   `json:"gate_id"`
	Name                    string   `json:"name"`
	MinReadyCandidates      int      `json:"min_ready_candidates"`
	RequiredCapabilities    []string `json:"required_capabilities,omitempty"`
	RequiredEvidence        []string `json:"required_evidence,omitempty"`
	RequiredBaselineVersion string   `json:"required_baseline_version,omitempty"`
}

type EntryGateDecision struct {
	GateID              string   `json:"gate_id"`
	Passed              bool     `json:"passed"`
	ReadyCandidateIDs   []string `json:"ready_candidate_ids,omitempty"`
	BlockedCandidateIDs []string `json:"blocked_candidate_ids,omitempty"`
	MissingCapabilities []string `json:"missing_capabilities,omitempty"`
	MissingEvidence     []string `json:"missing_evidence,omitempty"`
	BaselineReady       bool     `json:"baseline_ready"`
	BaselineFindings    []string `json:"baseline_findings,omitempty"`
	BlockerCount        int      `json:"blocker_count"`
}

func (d EntryGateDecision) StatusSummary() string {
	state := "HOLD"
	if d.Passed {
		state = "PASS"
	}
	return fmt.Sprintf("%s: ready=%d blocked=%d missing_capabilities=%d missing_evidence=%d baseline_findings=%d",
		state, len(d.ReadyCandidateIDs), len(d.BlockedCandidateIDs), len(d.MissingCapabilities), len(d.MissingEvidence), len(d.BaselineFindings))
}

type CandidatePlanner struct{}

func (CandidatePlanner) EvaluateGate(backlog CandidateBacklog, gate EntryGate, baselineAudit *governance.ScopeFreezeAudit) EntryGateDecision {
	ready := make([]string, 0)
	blocked := make([]string, 0)
	capabilities := make(map[string]struct{})
	evidence := make(map[string]struct{})
	blockerCount := 0
	for _, candidate := range backlog.RankedCandidates() {
		if candidate.Ready() {
			ready = append(ready, candidate.CandidateID)
			for _, capability := range candidate.Capabilities {
				if trimmed := strings.TrimSpace(capability); trimmed != "" {
					capabilities[trimmed] = struct{}{}
				}
			}
			for _, item := range candidate.Evidence {
				if trimmed := strings.TrimSpace(item); trimmed != "" {
					evidence[trimmed] = struct{}{}
				}
			}
			continue
		}
		blocked = append(blocked, candidate.CandidateID)
		blockerCount += len(candidate.Blockers)
	}

	missingCapabilities := missingStrings(gate.RequiredCapabilities, capabilities)
	missingEvidence := missingStrings(gate.RequiredEvidence, evidence)
	baselineReady := true
	baselineFindings := []string{}
	if strings.TrimSpace(gate.RequiredBaselineVersion) != "" {
		if baselineAudit == nil {
			baselineReady = false
			baselineFindings = []string{fmt.Sprintf("missing baseline audit for %s", gate.RequiredBaselineVersion)}
		} else if !baselineAudit.ReleaseReady() || !strings.EqualFold(strings.TrimSpace(baselineAudit.Version), strings.TrimSpace(gate.RequiredBaselineVersion)) {
			baselineReady = false
			baselineFindings = []string{fmt.Sprintf("baseline %s is not release ready (%.1f)", gate.RequiredBaselineVersion, baselineAudit.ReadinessScore())}
		}
	}

	passed := len(ready) >= gate.MinReadyCandidates && len(missingCapabilities) == 0 && len(missingEvidence) == 0 && baselineReady
	return EntryGateDecision{
		GateID:              gate.GateID,
		Passed:              passed,
		ReadyCandidateIDs:   ready,
		BlockedCandidateIDs: blocked,
		MissingCapabilities: missingCapabilities,
		MissingEvidence:     missingEvidence,
		BaselineReady:       baselineReady,
		BaselineFindings:    baselineFindings,
		BlockerCount:        blockerCount,
	}
}

type WeeklyGoal struct {
	GoalID        string `json:"goal_id"`
	Title         string `json:"title"`
	Owner         string `json:"owner"`
	Status        string `json:"status"`
	SuccessMetric string `json:"success_metric"`
	TargetValue   string `json:"target_value"`
}

type WeeklyExecutionPlan struct {
	WeekNumber int          `json:"week_number"`
	Theme      string       `json:"theme"`
	Objective  string       `json:"objective"`
	Goals      []WeeklyGoal `json:"goals,omitempty"`
}

func (w WeeklyExecutionPlan) CompletedGoals() int {
	count := 0
	for _, goal := range w.Goals {
		if strings.EqualFold(goal.Status, "done") {
			count++
		}
	}
	return count
}

func (w WeeklyExecutionPlan) ProgressPercent() int {
	if len(w.Goals) == 0 {
		return 0
	}
	return (w.CompletedGoals() * 100) / len(w.Goals)
}

func (w WeeklyExecutionPlan) AtRiskGoalIDs() []string {
	out := make([]string, 0)
	for _, goal := range w.Goals {
		status := strings.ToLower(strings.TrimSpace(goal.Status))
		if status == "at-risk" || status == "blocked" {
			out = append(out, goal.GoalID)
		}
	}
	return out
}

type FourWeekExecutionPlan struct {
	PlanID    string                `json:"plan_id"`
	Title     string                `json:"title"`
	Owner     string                `json:"owner"`
	StartDate string                `json:"start_date"`
	Weeks     []WeeklyExecutionPlan `json:"weeks,omitempty"`
}

func (p FourWeekExecutionPlan) Validate() error {
	if len(p.Weeks) != 4 {
		return errors.New("Four-week execution plans must include weeks 1 through 4 in order")
	}
	for i, week := range p.Weeks {
		if week.WeekNumber != i+1 {
			return errors.New("Four-week execution plans must include weeks 1 through 4 in order")
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
	total := p.TotalGoals()
	if total == 0 {
		return 0
	}
	return (p.CompletedGoals() * 100) / total
}

func (p FourWeekExecutionPlan) AtRiskWeeks() []int {
	out := make([]int, 0)
	for _, week := range p.Weeks {
		if len(week.AtRiskGoalIDs()) > 0 {
			out = append(out, week.WeekNumber)
		}
	}
	return out
}

func (p FourWeekExecutionPlan) GoalStatusCounts() map[string]int {
	counts := map[string]int{
		"done":        0,
		"on-track":    0,
		"at-risk":     0,
		"not-started": 0,
	}
	for _, week := range p.Weeks {
		for _, goal := range week.Goals {
			status := strings.ToLower(strings.TrimSpace(goal.Status))
			counts[status]++
		}
	}
	return counts
}

func RenderCandidateBacklogReport(backlog CandidateBacklog, gate EntryGate, decision EntryGateDecision) string {
	lines := []string{
		"# V3 Candidate Backlog Report",
		"",
		fmt.Sprintf("- Epic: %s %s", backlog.EpicID, backlog.Title),
		fmt.Sprintf("- Version: %s", backlog.Version),
		fmt.Sprintf("- Gate: %s %s", gate.GateID, gate.Name),
		fmt.Sprintf("- Decision: %s", decision.StatusSummary()),
		"",
		"## Candidates",
		"",
	}
	for _, candidate := range backlog.RankedCandidates() {
		lines = append(lines, fmt.Sprintf("- %s: %s priority=%s owner=%s score=%d ready=%t",
			candidate.CandidateID, candidate.Title, candidate.Priority, candidate.Owner, candidate.Score(), candidate.Ready()))
		lines = append(lines, fmt.Sprintf("  validation=%s", candidate.ValidationCommand))
		for _, link := range candidate.EvidenceLinks {
			line := fmt.Sprintf("  - %s -> %s capability=%s", link.Label, link.Target, link.Capability)
			if strings.TrimSpace(link.Note) != "" {
				line += fmt.Sprintf(" note=%s", link.Note)
			}
			lines = append(lines, line)
		}
	}
	lines = append(lines, "", "## Findings", "")
	lines = append(lines, fmt.Sprintf("- Missing capabilities: %s", joinedOrNone(decision.MissingCapabilities)))
	lines = append(lines, fmt.Sprintf("- Missing evidence: %s", joinedOrNone(decision.MissingEvidence)))
	lines = append(lines, fmt.Sprintf("- Baseline ready: %t", decision.BaselineReady))
	lines = append(lines, fmt.Sprintf("- Baseline findings: %s", joinedOrNone(decision.BaselineFindings)))
	return strings.Join(lines, "\n") + "\n"
}

func RenderFourWeekExecutionReport(plan FourWeekExecutionPlan) string {
	lines := []string{
		"# Four-Week Execution Plan",
		"",
		fmt.Sprintf("- Plan: %s %s", plan.PlanID, plan.Title),
		fmt.Sprintf("- Owner: %s", plan.Owner),
		fmt.Sprintf("- Start date: %s", plan.StartDate),
		fmt.Sprintf("- Overall progress: %d/%d goals complete (%d%%)", plan.CompletedGoals(), plan.TotalGoals(), plan.OverallProgressPercent()),
		fmt.Sprintf("- At-risk weeks: %s", intsOrNone(plan.AtRiskWeeks())),
		"",
		"## Weeks",
		"",
	}
	for _, week := range plan.Weeks {
		lines = append(lines, fmt.Sprintf("- Week %d: %s progress=%d/%d (%d%%)", week.WeekNumber, week.Theme, week.CompletedGoals(), len(week.Goals), week.ProgressPercent()))
		for _, goal := range week.Goals {
			lines = append(lines, fmt.Sprintf("  - %s: %s owner=%s status=%s", goal.GoalID, goal.Title, goal.Owner, goal.Status))
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

func BuildBIG4701ExecutionPlan() FourWeekExecutionPlan {
	return FourWeekExecutionPlan{
		PlanID:    "BIG-4701",
		Title:     "4周执行计划与周目标",
		Owner:     "execution-office",
		StartDate: "2026-03-11",
		Weeks: []WeeklyExecutionPlan{
			{WeekNumber: 1, Theme: "Foundation alignment", Objective: "Align backlog and owners.", Goals: []WeeklyGoal{
				{GoalID: "w1-freeze", Title: "Confirm v4.0 freeze scope", Owner: "pm-office", Status: "done", SuccessMetric: "approved freeze board", TargetValue: "1"},
				{GoalID: "w1-backlog", Title: "Publish candidate backlog", Owner: "platform-ui", Status: "done", SuccessMetric: "published backlog", TargetValue: "1"},
			}},
			{WeekNumber: 2, Theme: "Build and integration", Objective: "Land integration dependencies.", Goals: []WeeklyGoal{
				{GoalID: "w2-handoff-sync", Title: "Resolve orchestration and console handoff dependencies", Owner: "orchestration-office", Status: "at-risk", SuccessMetric: "merged integration plan", TargetValue: "1"},
				{GoalID: "w2-queue-audit", Title: "Harden queue and approval audit paths", Owner: "ops-platform", Status: "not-started", SuccessMetric: "passing regression suite", TargetValue: "1"},
			}},
			{WeekNumber: 3, Theme: "Validation and coverage", Objective: "Prove migration coverage.", Goals: []WeeklyGoal{
				{GoalID: "w3-reports", Title: "Publish cross-surface validation bundle", Owner: "reporting", Status: "on-track", SuccessMetric: "bundle generated", TargetValue: "1"},
				{GoalID: "w3-shadow", Title: "Close remaining shadow gaps", Owner: "runtime", Status: "not-started", SuccessMetric: "gap count", TargetValue: "0"},
			}},
			{WeekNumber: 4, Theme: "Release readiness", Objective: "Prepare rollout decision.", Goals: []WeeklyGoal{
				{GoalID: "w4-rollout", Title: "Finalize release recommendation", Owner: "release-office", Status: "not-started", SuccessMetric: "decision memo", TargetValue: "1"},
				{GoalID: "w4-handoff", Title: "Complete launch handoff artifacts", Owner: "customer-success", Status: "not-started", SuccessMetric: "handoff packet", TargetValue: "1"},
			}},
		},
	}
}

func BuildV3CandidateBacklog() CandidateBacklog {
	return CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3候选与进入条件",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{
				CandidateID:       "candidate-ops-hardening",
				Title:             "Operations command-center hardening",
				Theme:             "ops-command-center",
				Priority:          "P0",
				Owner:             "ops-platform",
				Outcome:           "Package command-center and approval surfaces with linked evidence.",
				ValidationCommand: "cd bigclaw-go && go test ./internal/product ./internal/reporting ./internal/workflow",
				Capabilities:      []string{"ops-control", "saved-views", "commercialization"},
				Evidence:          []string{"weekly-review", "validation-report", "pilot-evidence"},
				EvidenceLinks: []EvidenceLink{
					{Label: "command-center-src", Target: "src/bigclaw/operations.py", Capability: "ops-control"},
					{Label: "control-center-tests", Target: "tests/test_control_center.py", Capability: "ops-control"},
					{Label: "execution-contract-src", Target: "src/bigclaw/execution_contract.py", Capability: "ops-control"},
					{Label: "workflow-src", Target: "src/bigclaw/workflow.py", Capability: "commercialization"},
					{Label: "workflow-tests", Target: "tests/test_workflow.py", Capability: "commercialization"},
					{Label: "execution-flow-tests", Target: "tests/test_execution_flow.py", Capability: "commercialization"},
					{Label: "saved-views-src", Target: "src/bigclaw/saved_views.py", Capability: "saved-views"},
					{Label: "saved-views-tests", Target: "tests/test_saved_views.py", Capability: "saved-views"},
					{Label: "evaluation-src", Target: "src/bigclaw/evaluation.py", Capability: "ops-control"},
					{Label: "evaluation-tests", Target: "tests/test_evaluation.py", Capability: "ops-control"},
				},
			},
			{
				CandidateID:       "candidate-orchestration-rollout",
				Title:             "Orchestration rollout",
				Theme:             "agent-orchestration",
				Priority:          "P1",
				Owner:             "orchestration",
				Outcome:           "Promote cross-team orchestration with commercialization visibility.",
				ValidationCommand: "cd bigclaw-go && go test ./internal/reporting ./internal/scheduler ./internal/workflow",
				Capabilities:      []string{"commercialization", "handoff"},
				Evidence:          []string{"pilot-evidence", "validation-report"},
				EvidenceLinks: []EvidenceLink{
					{Label: "report-studio-tests", Target: "tests/test_reports.py", Capability: "commercialization"},
					{Label: "workflow-tests", Target: "tests/test_workflow.py", Capability: "handoff"},
				},
			},
			{
				CandidateID:       "candidate-release-control",
				Title:             "Release control center",
				Theme:             "console-governance",
				Priority:          "P0",
				Owner:             "platform-ui",
				Outcome:           "Unify console release gates and promotion evidence.",
				ValidationCommand: "cd bigclaw-go && go test ./internal/api ./internal/reporting",
				Capabilities:      []string{"release-gate", "reporting"},
				Evidence:          []string{"acceptance-suite", "validation-report"},
				EvidenceLinks: []EvidenceLink{
					{Label: "ui-acceptance", Target: "tests/test_design_system.py", Capability: "release-gate", Note: "role-permission and audit readiness coverage"},
				},
			},
		},
	}
}

func BuildV3EntryGate() EntryGate {
	return EntryGate{
		GateID:                  "gate-v3-entry",
		Name:                    "V3 Entry Gate",
		MinReadyCandidates:      3,
		RequiredCapabilities:    []string{"release-gate", "ops-control", "commercialization"},
		RequiredEvidence:        []string{"acceptance-suite", "pilot-evidence", "validation-report"},
		RequiredBaselineVersion: "v2.0",
	}
}

func MustJSONRoundTrip[T any](value T) T {
	body, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		panic(err)
	}
	return out
}

func missingStrings(required []string, present map[string]struct{}) []string {
	out := make([]string, 0)
	for _, item := range required {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := present[item]; ok {
			continue
		}
		if !contains(out, item) {
			out = append(out, item)
		}
	}
	return out
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func joinedOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func intsOrNone(values []int) string {
	if len(values) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%d", value))
	}
	return strings.Join(parts, ", ")
}
