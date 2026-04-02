package planning

import (
	"fmt"
	"strings"

	"bigclaw-go/internal/governance"
)

var priorityWeights = map[string]int{
	"P0": 4,
	"P1": 3,
	"P2": 2,
	"P3": 1,
}

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
	Dependencies      []string       `json:"dependencies,omitempty"`
	Blockers          []string       `json:"blockers,omitempty"`
}

func (entry CandidateEntry) ReadinessScore() int {
	base := priorityWeights[strings.ToUpper(strings.TrimSpace(entry.Priority))] * 25
	dependencyPenalty := len(entry.Dependencies) * 10
	blockerPenalty := len(entry.Blockers) * 20
	evidenceBonus := minInt(len(entry.Evidence), 3) * 5
	score := base + evidenceBonus - dependencyPenalty - blockerPenalty
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func (entry CandidateEntry) Ready() bool {
	return len(entry.Capabilities) > 0 && len(entry.Evidence) > 0 && len(entry.Blockers) == 0
}

type CandidateBacklog struct {
	EpicID     string           `json:"epic_id"`
	Title      string           `json:"title"`
	Version    string           `json:"version"`
	Candidates []CandidateEntry `json:"candidates,omitempty"`
}

func (backlog CandidateBacklog) RankedCandidates() []CandidateEntry {
	ranked := append([]CandidateEntry(nil), backlog.Candidates...)
	sortCandidates(ranked)
	return ranked
}

type EntryGate struct {
	GateID                  string   `json:"gate_id"`
	Name                    string   `json:"name"`
	MinReadyCandidates      int      `json:"min_ready_candidates"`
	RequiredCapabilities    []string `json:"required_capabilities,omitempty"`
	RequiredEvidence        []string `json:"required_evidence,omitempty"`
	RequiredBaselineVersion string   `json:"required_baseline_version,omitempty"`
	MaxBlockers             int      `json:"max_blockers"`
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

func (decision EntryGateDecision) Summary() string {
	status := "HOLD"
	if decision.Passed {
		status = "PASS"
	}
	return fmt.Sprintf(
		"%s: ready=%d blocked=%d missing_capabilities=%d missing_evidence=%d baseline_findings=%d",
		status,
		len(decision.ReadyCandidateIDs),
		decision.BlockerCount,
		len(decision.MissingCapabilities),
		len(decision.MissingEvidence),
		len(decision.BaselineFindings),
	)
}

type CandidatePlanner struct{}

func (CandidatePlanner) EvaluateGate(backlog CandidateBacklog, gate EntryGate, baselineAudit *governance.ScopeFreezeAudit) EntryGateDecision {
	readyCandidates := make([]CandidateEntry, 0)
	blockedCandidates := make([]CandidateEntry, 0)
	for _, candidate := range backlog.RankedCandidates() {
		if candidate.Ready() {
			readyCandidates = append(readyCandidates, candidate)
		}
	}
	for _, candidate := range backlog.Candidates {
		if len(candidate.Blockers) > 0 {
			blockedCandidates = append(blockedCandidates, candidate)
		}
	}

	providedCapabilities := map[string]struct{}{}
	providedEvidence := map[string]struct{}{}
	for _, candidate := range readyCandidates {
		for _, capability := range candidate.Capabilities {
			providedCapabilities[capability] = struct{}{}
		}
		for _, evidence := range candidate.Evidence {
			providedEvidence[evidence] = struct{}{}
		}
	}

	missingCapabilities := make([]string, 0)
	for _, capability := range gate.RequiredCapabilities {
		if _, ok := providedCapabilities[capability]; !ok {
			missingCapabilities = append(missingCapabilities, capability)
		}
	}
	missingEvidence := make([]string, 0)
	for _, evidence := range gate.RequiredEvidence {
		if _, ok := providedEvidence[evidence]; !ok {
			missingEvidence = append(missingEvidence, evidence)
		}
	}

	baselineFindings := baselineFindings(gate, baselineAudit)
	baselineReady := len(baselineFindings) == 0
	passed := len(readyCandidates) >= gate.MinReadyCandidates &&
		len(blockedCandidates) <= gate.MaxBlockers &&
		len(missingCapabilities) == 0 &&
		len(missingEvidence) == 0 &&
		baselineReady

	return EntryGateDecision{
		GateID:              gate.GateID,
		Passed:              passed,
		ReadyCandidateIDs:   candidateIDs(readyCandidates),
		BlockedCandidateIDs: candidateIDs(blockedCandidates),
		MissingCapabilities: missingCapabilities,
		MissingEvidence:     missingEvidence,
		BaselineReady:       baselineReady,
		BaselineFindings:    baselineFindings,
		BlockerCount:        len(blockedCandidates),
	}
}

func RenderCandidateBacklogReport(backlog CandidateBacklog, gate EntryGate, decision EntryGateDecision) string {
	lines := []string{
		"# V3 Candidate Backlog Report",
		"",
		fmt.Sprintf("- Epic: %s %s", backlog.EpicID, backlog.Title),
		fmt.Sprintf("- Version: %s", backlog.Version),
		fmt.Sprintf("- Gate: %s", gate.Name),
		fmt.Sprintf("- Decision: %s", decision.Summary()),
		"",
		"## Candidates",
	}
	for _, candidate := range backlog.RankedCandidates() {
		lines = append(lines,
			fmt.Sprintf(
				"- %s: %s priority=%s owner=%s score=%d ready=%s",
				candidate.CandidateID,
				candidate.Title,
				candidate.Priority,
				candidate.Owner,
				candidate.ReadinessScore(),
				titleBool(candidate.Ready()),
			),
			fmt.Sprintf(
				"  theme=%s outcome=%s capabilities=%s evidence=%s blockers=%s",
				candidate.Theme,
				candidate.Outcome,
				joinOrNone(candidate.Capabilities),
				joinOrNone(candidate.Evidence),
				joinOrNone(candidate.Blockers),
			),
			fmt.Sprintf("  validation=%s", candidate.ValidationCommand),
		)
		if len(candidate.Dependencies) > 0 {
			lines = append(lines, fmt.Sprintf("  dependencies=%s", strings.Join(candidate.Dependencies, ",")))
		}
		if len(candidate.EvidenceLinks) > 0 {
			lines = append(lines, "  evidence-links:")
			for _, link := range candidate.EvidenceLinks {
				qualifier := ""
				if link.Capability != "" {
					qualifier = " capability=" + link.Capability
				}
				note := ""
				if link.Note != "" {
					note = " note=" + link.Note
				}
				lines = append(lines, fmt.Sprintf("  - %s -> %s%s%s", link.Label, link.Target, qualifier, note))
			}
		}
	}
	lines = append(lines,
		"",
		"## Gate Findings",
		fmt.Sprintf("- Ready candidates: %s", joinOrNone(decision.ReadyCandidateIDs)),
		fmt.Sprintf("- Blocked candidates: %s", joinOrNone(decision.BlockedCandidateIDs)),
		fmt.Sprintf("- Missing capabilities: %s", joinOrNone(decision.MissingCapabilities)),
		fmt.Sprintf("- Missing evidence: %s", joinOrNone(decision.MissingEvidence)),
		fmt.Sprintf("- Baseline ready: %s", titleBool(decision.BaselineReady)),
		fmt.Sprintf("- Baseline findings: %s", joinOrNone(decision.BaselineFindings)),
	)
	return strings.Join(lines, "\n")
}

func BuildV3CandidateBacklog() CandidateBacklog {
	return CandidateBacklog{
		EpicID:  "BIG-EPIC-20",
		Title:   "v4.0 v3候选与进入条件",
		Version: "v4.0-v3",
		Candidates: []CandidateEntry{
			{
				CandidateID:       "candidate-release-control",
				Title:             "Console release control center",
				Theme:             "console-governance",
				Priority:          "P0",
				Owner:             "product-experience",
				Outcome:           "Converge console shell governance, UI acceptance, and review-pack evidence into one release-control candidate.",
				ValidationCommand: "cd bigclaw-go && go test ./internal/designsystem ./internal/uireview ./internal/planning",
				Capabilities:      []string{"release-gate", "console-shell", "reporting"},
				Evidence:          []string{"acceptance-suite", "validation-report"},
				EvidenceLinks: []EvidenceLink{
					{Label: "design-system-tests", Target: "bigclaw-go/internal/designsystem/designsystem_test.go", Capability: "release-gate", Note: "Go-native component inventory, accessibility, and UI acceptance coverage"},
					{Label: "design-system-surface", Target: "bigclaw-go/internal/designsystem/designsystem.go", Capability: "release-gate", Note: "Go-native information architecture and console chrome contracts"},
					{Label: "review-pack-tests", Target: "bigclaw-go/internal/uireview/uireview_test.go", Capability: "release-gate", Note: "Go-native deterministic review packet validation"},
					{Label: "review-pack-surface", Target: "bigclaw-go/internal/uireview/uireview.go", Capability: "release-gate", Note: "review objectives, wireframes, interaction coverage, and open questions"},
					{Label: "review-pack-render", Target: "bigclaw-go/internal/uireview/render.go", Capability: "release-gate", Note: "Go-native review report and board rendering"},
					{Label: "candidate-planner-tests", Target: "bigclaw-go/internal/planning/planning_test.go", Capability: "reporting", Note: "release candidate validation commands and evidence targets stay Go-only"},
				},
			},
			{
				CandidateID:       "candidate-ops-hardening",
				Title:             "Operations command-center hardening",
				Theme:             "ops-command-center",
				Priority:          "P0",
				Owner:             "engineering-operations",
				Outcome:           "Promote queue control, approval handling, saved views, dashboard builder output, and replay evidence as one operator-ready command center.",
				ValidationCommand: "PYTHONPATH=src python3 -m pytest tests/test_operations.py -q && (cd bigclaw-go && go test ./internal/evaluation ./internal/product ./internal/worker ./internal/workflow ./internal/scheduler)",
				Capabilities:      []string{"ops-control", "saved-views", "rollback-simulation"},
				Evidence:          []string{"weekly-review", "validation-report"},
				EvidenceLinks: []EvidenceLink{
					{Label: "command-center-src", Target: "src/bigclaw/__init__.py", Capability: "ops-control", Note: "folded package command center, dashboard builder, weekly review, and regression surfaces"},
					{Label: "operations-tests", Target: "tests/test_operations.py", Capability: "ops-control", Note: "dashboard, weekly report, regression, and version-center coverage"},
					{Label: "approval-contract", Target: "src/bigclaw/__init__.py", Capability: "ops-control", Note: "folded package approval permission and API role coverage contract"},
					{Label: "approval-workflow", Target: "src/bigclaw/__init__.py", Capability: "ops-control", Note: "folded package approval workflow and closeout flow wiring"},
					{Label: "workflow-tests", Target: "bigclaw-go/internal/workflow/engine_test.go", Capability: "ops-control", Note: "acceptance gate and workpad journal validation"},
					{Label: "execution-flow-tests", Target: "bigclaw-go/internal/worker/runtime_test.go", Capability: "ops-control", Note: "execution handoff, closeout, and routed runtime evidence"},
					{Label: "saved-views-src", Target: "src/bigclaw/__init__.py", Capability: "saved-views", Note: "folded package saved views, digest subscriptions, and governed filters"},
					{Label: "saved-views-tests", Target: "bigclaw-go/internal/product/saved_views_test.go", Capability: "saved-views", Note: "Go-native saved-view audit coverage"},
					{Label: "simulation-src", Target: "bigclaw-go/internal/evaluation/evaluation.go", Capability: "rollback-simulation", Note: "Go-native simulation, replay, and comparison evidence"},
					{Label: "simulation-tests", Target: "bigclaw-go/internal/evaluation/evaluation_test.go", Capability: "rollback-simulation", Note: "Go-native replay and benchmark validation"},
				},
			},
			{
				CandidateID:       "candidate-orchestration-rollout",
				Title:             "Agent orchestration rollout",
				Theme:             "agent-orchestration",
				Priority:          "P0",
				Owner:             "orchestration-office",
				Outcome:           "Carry entitlement-aware orchestration, handoff visibility, and commercialization proof into a candidate ready for release review.",
				ValidationCommand: "cd bigclaw-go && go test ./internal/collaboration ./internal/pilot ./internal/reportstudio",
				Capabilities:      []string{"commercialization", "handoff", "pilot-rollout"},
				Evidence:          []string{"pilot-evidence", "validation-report"},
				EvidenceLinks: []EvidenceLink{
					{Label: "orchestration-plan-src", Target: "src/bigclaw/__init__.py", Capability: "commercialization", Note: "folded package orchestration, entitlement-aware policy, and handoff decisions"},
					{Label: "orchestration-report-src", Target: "bigclaw-go/internal/reportstudio/reportstudio.go", Capability: "commercialization", Note: "Go-native orchestration canvas, portfolio rollups, and narrative exports"},
					{Label: "collaboration-tests", Target: "bigclaw-go/internal/collaboration/thread_test.go", Capability: "handoff", Note: "Go-native thread merge and handoff validation"},
					{Label: "pilot-rollout-tests", Target: "bigclaw-go/internal/pilot/rollout_test.go", Capability: "pilot-rollout", Note: "Go-native rollout scoring and repo narrative validation"},
					{Label: "report-studio-tests", Target: "bigclaw-go/internal/reportstudio/reportstudio_test.go", Capability: "commercialization", Note: "Go-native report exports and downstream evidence sharing"},
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
		MaxBlockers:             0,
	}
}

type WeeklyGoal struct {
	GoalID        string   `json:"goal_id"`
	Title         string   `json:"title"`
	Owner         string   `json:"owner"`
	Status        string   `json:"status"`
	SuccessMetric string   `json:"success_metric"`
	TargetValue   string   `json:"target_value"`
	CurrentValue  string   `json:"current_value,omitempty"`
	Dependencies  []string `json:"dependencies,omitempty"`
	Risks         []string `json:"risks,omitempty"`
}

func (goal WeeklyGoal) StatusRank() int {
	switch strings.ToLower(strings.TrimSpace(goal.Status)) {
	case "done":
		return 4
	case "on-track":
		return 3
	case "at-risk":
		return 2
	case "blocked":
		return 1
	case "not-started":
		return 0
	default:
		return -1
	}
}

func (goal WeeklyGoal) IsComplete() bool {
	return strings.EqualFold(strings.TrimSpace(goal.Status), "done")
}

func (goal WeeklyGoal) IsAtRisk() bool {
	switch strings.ToLower(strings.TrimSpace(goal.Status)) {
	case "at-risk", "blocked":
		return true
	default:
		return false
	}
}

type WeeklyExecutionPlan struct {
	WeekNumber   int          `json:"week_number"`
	Theme        string       `json:"theme"`
	Objective    string       `json:"objective"`
	ExitCriteria []string     `json:"exit_criteria,omitempty"`
	Deliverables []string     `json:"deliverables,omitempty"`
	Goals        []WeeklyGoal `json:"goals,omitempty"`
}

func (week WeeklyExecutionPlan) CompletedGoals() int {
	count := 0
	for _, goal := range week.Goals {
		if goal.IsComplete() {
			count++
		}
	}
	return count
}

func (week WeeklyExecutionPlan) TotalGoals() int {
	return len(week.Goals)
}

func (week WeeklyExecutionPlan) ProgressPercent() int {
	if len(week.Goals) == 0 {
		return 0
	}
	return int(float64(week.CompletedGoals()) / float64(len(week.Goals)) * 100)
}

func (week WeeklyExecutionPlan) AtRiskGoalIDs() []string {
	ids := make([]string, 0)
	for _, goal := range week.Goals {
		if goal.IsAtRisk() {
			ids = append(ids, goal.GoalID)
		}
	}
	return ids
}

type FourWeekExecutionPlan struct {
	PlanID    string                `json:"plan_id"`
	Title     string                `json:"title"`
	Owner     string                `json:"owner"`
	StartDate string                `json:"start_date"`
	Weeks     []WeeklyExecutionPlan `json:"weeks,omitempty"`
}

func (plan FourWeekExecutionPlan) TotalGoals() int {
	total := 0
	for _, week := range plan.Weeks {
		total += week.TotalGoals()
	}
	return total
}

func (plan FourWeekExecutionPlan) CompletedGoals() int {
	total := 0
	for _, week := range plan.Weeks {
		total += week.CompletedGoals()
	}
	return total
}

func (plan FourWeekExecutionPlan) OverallProgressPercent() int {
	if plan.TotalGoals() == 0 {
		return 0
	}
	return int(float64(plan.CompletedGoals()) / float64(plan.TotalGoals()) * 100)
}

func (plan FourWeekExecutionPlan) AtRiskWeeks() []int {
	weeks := make([]int, 0)
	for _, week := range plan.Weeks {
		if len(week.AtRiskGoalIDs()) > 0 {
			weeks = append(weeks, week.WeekNumber)
		}
	}
	return weeks
}

func (plan FourWeekExecutionPlan) GoalStatusCounts() map[string]int {
	counts := make(map[string]int)
	for _, week := range plan.Weeks {
		for _, goal := range week.Goals {
			counts[goal.Status]++
		}
	}
	return counts
}

func (plan FourWeekExecutionPlan) Validate() error {
	weekNumbers := make([]int, 0, len(plan.Weeks))
	for _, week := range plan.Weeks {
		weekNumbers = append(weekNumbers, week.WeekNumber)
	}
	expected := []int{1, 2, 3, 4}
	if len(weekNumbers) != len(expected) {
		return fmt.Errorf("Four-week execution plans must include weeks 1 through 4 in order")
	}
	for i, weekNumber := range weekNumbers {
		if weekNumber != expected[i] {
			return fmt.Errorf("Four-week execution plans must include weeks 1 through 4 in order")
		}
	}
	return nil
}

func BuildBig4701ExecutionPlan() FourWeekExecutionPlan {
	plan := FourWeekExecutionPlan{
		PlanID:    "BIG-4701",
		Title:     "4周执行计划与周目标",
		Owner:     "execution-office",
		StartDate: "2026-03-11",
		Weeks: []WeeklyExecutionPlan{
			{
				WeekNumber:   1,
				Theme:        "Scope freeze and operating baseline",
				Objective:    "Freeze scope, align owners, and establish validation and reporting cadence.",
				ExitCriteria: []string{"Scope freeze board published", "Owners and validation commands assigned for all streams"},
				Deliverables: []string{"Execution baseline report", "Scope freeze audit snapshot"},
				Goals: []WeeklyGoal{
					{GoalID: "w1-scope-freeze", Title: "Lock the v4.0 scope and escalation path", Owner: "program-office", Status: "done", SuccessMetric: "frozen backlog items", TargetValue: "5 epics aligned", CurrentValue: "5 epics aligned"},
					{GoalID: "w1-validation-matrix", Title: "Assign validation commands and evidence owners", Owner: "engineering-ops", Status: "done", SuccessMetric: "streams with validation owners", TargetValue: "5/5 streams", CurrentValue: "5/5 streams"},
				},
			},
			{
				WeekNumber:   2,
				Theme:        "Build and integration",
				Objective:    "Land the highest-risk implementation slices and wire cross-team dependencies.",
				ExitCriteria: []string{"P0 build items merged", "Cross-team dependency review completed"},
				Deliverables: []string{"Integrated build checkpoint", "Dependency burn-down"},
				Goals: []WeeklyGoal{
					{GoalID: "w2-p0-burndown", Title: "Close the top P0 implementation gaps", Owner: "engineering-platform", Status: "on-track", SuccessMetric: "P0 items merged", TargetValue: ">=3 merged", CurrentValue: "2 merged"},
					{GoalID: "w2-handoff-sync", Title: "Resolve orchestration and console handoff dependencies", Owner: "orchestration-office", Status: "at-risk", SuccessMetric: "open handoff blockers", TargetValue: "0 blockers", CurrentValue: "1 blocker", Dependencies: []string{"w2-p0-burndown"}, Risks: []string{"console entitlement contract is pending"}},
				},
			},
			{
				WeekNumber:   3,
				Theme:        "Stabilization and validation",
				Objective:    "Drive regression triage, benchmark replay, and release-readiness evidence.",
				ExitCriteria: []string{"Regression backlog under control threshold", "Benchmark comparison published"},
				Deliverables: []string{"Stabilization report", "Benchmark replay pack"},
				Goals: []WeeklyGoal{
					{GoalID: "w3-regression-triage", Title: "Reduce critical regressions before release gate", Owner: "quality-ops", Status: "not-started", SuccessMetric: "critical regressions", TargetValue: "<=2 open"},
					{GoalID: "w3-benchmark-pack", Title: "Publish replay and weighted benchmark evidence", Owner: "evaluation-lab", Status: "not-started", SuccessMetric: "benchmark evidence bundle", TargetValue: "1 bundle published"},
				},
			},
			{
				WeekNumber:   4,
				Theme:        "Launch decision and weekly operating rhythm",
				Objective:    "Convert validation evidence into launch readiness and the post-launch weekly review cadence.",
				ExitCriteria: []string{"Launch decision signed off", "Weekly operating review template adopted"},
				Deliverables: []string{"Launch readiness packet", "Weekly review operating template"},
				Goals: []WeeklyGoal{
					{GoalID: "w4-launch-decision", Title: "Complete launch readiness review", Owner: "release-governance", Status: "not-started", SuccessMetric: "required sign-offs", TargetValue: "all sign-offs complete"},
					{GoalID: "w4-weekly-rhythm", Title: "Roll out the weekly KPI and issue review cadence", Owner: "engineering-operations", Status: "not-started", SuccessMetric: "weekly review adoption", TargetValue: "1 recurring cadence active"},
				},
			},
		},
	}
	return plan
}

func RenderFourWeekExecutionReport(plan FourWeekExecutionPlan) string {
	statusCounts := plan.GoalStatusCounts()
	lines := []string{
		"# Four-Week Execution Plan",
		"",
		fmt.Sprintf("- Plan: %s %s", plan.PlanID, plan.Title),
		fmt.Sprintf("- Owner: %s", plan.Owner),
		fmt.Sprintf("- Start date: %s", plan.StartDate),
		fmt.Sprintf("- Overall progress: %d/%d goals complete (%d%%)", plan.CompletedGoals(), plan.TotalGoals(), plan.OverallProgressPercent()),
		fmt.Sprintf("- At-risk weeks: %s", joinWeekNumbers(plan.AtRiskWeeks())),
		fmt.Sprintf(
			"- Goal status counts: done=%d on-track=%d at-risk=%d blocked=%d not-started=%d",
			statusCounts["done"],
			statusCounts["on-track"],
			statusCounts["at-risk"],
			statusCounts["blocked"],
			statusCounts["not-started"],
		),
		"",
		"## Weekly Plans",
	}
	for _, week := range plan.Weeks {
		lines = append(lines,
			fmt.Sprintf("- Week %d: %s progress=%d/%d (%d%%)", week.WeekNumber, week.Theme, week.CompletedGoals(), week.TotalGoals(), week.ProgressPercent()),
			fmt.Sprintf("  objective=%s", week.Objective),
			fmt.Sprintf("  exit_criteria=%s", joinOrNone(week.ExitCriteria)),
			fmt.Sprintf("  deliverables=%s", joinOrNone(week.Deliverables)),
		)
		for _, goal := range week.Goals {
			lines = append(lines,
				fmt.Sprintf("  - %s: %s owner=%s status=%s metric=%s current=%s target=%s", goal.GoalID, goal.Title, goal.Owner, goal.Status, goal.SuccessMetric, valueOrNA(goal.CurrentValue), goal.TargetValue),
				fmt.Sprintf("    dependencies=%s risks=%s", joinOrNone(goal.Dependencies), joinOrNone(goal.Risks)),
			)
		}
	}
	return strings.Join(lines, "\n")
}

func baselineFindings(gate EntryGate, audit *governance.ScopeFreezeAudit) []string {
	if gate.RequiredBaselineVersion == "" {
		return nil
	}
	if audit == nil {
		return []string{fmt.Sprintf("missing baseline audit for %s", gate.RequiredBaselineVersion)}
	}
	findings := make([]string, 0)
	if audit.Version != gate.RequiredBaselineVersion {
		findings = append(findings, fmt.Sprintf("baseline version mismatch: expected %s, got %s", gate.RequiredBaselineVersion, audit.Version))
	}
	if !audit.ReleaseReady() {
		findings = append(findings, fmt.Sprintf("baseline %s is not release ready (%.1f)", audit.Version, audit.ReadinessScore()))
	}
	return findings
}

func candidateIDs[T interface{ GetCandidateID() string }](items []T) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.GetCandidateID())
	}
	return ids
}

func (entry CandidateEntry) GetCandidateID() string {
	return entry.CandidateID
}

func joinOrNone(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

func joinWeekNumbers(weeks []int) string {
	if len(weeks) == 0 {
		return "none"
	}
	values := make([]string, 0, len(weeks))
	for _, week := range weeks {
		values = append(values, fmt.Sprintf("%d", week))
	}
	return strings.Join(values, ", ")
}

func valueOrNA(value string) string {
	if strings.TrimSpace(value) == "" {
		return "n/a"
	}
	return value
}

func titleBool(value bool) string {
	if value {
		return "True"
	}
	return "False"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func sortCandidates(items []CandidateEntry) {
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			left := items[i]
			right := items[j]
			if right.ReadinessScore() > left.ReadinessScore() || (right.ReadinessScore() == left.ReadinessScore() && right.CandidateID < left.CandidateID) {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}
