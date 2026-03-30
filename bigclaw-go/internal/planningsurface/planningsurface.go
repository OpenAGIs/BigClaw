package planningsurface

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"bigclaw-go/internal/governance"
)

var priorityWeights = map[string]int{
	"P0": 4,
	"P1": 3,
	"P2": 2,
	"P3": 1,
}

var goalStatusOrder = map[string]int{
	"done":        4,
	"on-track":    3,
	"at-risk":     2,
	"blocked":     1,
	"not-started": 0,
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

func (c CandidateEntry) ReadinessScore() int {
	base := priorityWeights[strings.ToUpper(strings.TrimSpace(c.Priority))] * 25
	dependencyPenalty := len(c.Dependencies) * 10
	blockerPenalty := len(c.Blockers) * 20
	evidenceBonus := min(len(c.Evidence), 3) * 5
	score := base + evidenceBonus - dependencyPenalty - blockerPenalty
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func (c CandidateEntry) Ready() bool {
	return len(c.Capabilities) > 0 && len(c.Evidence) > 0 && len(c.Blockers) == 0
}

type CandidateBacklog struct {
	EpicID     string           `json:"epic_id"`
	Title      string           `json:"title"`
	Version    string           `json:"version"`
	Candidates []CandidateEntry `json:"candidates,omitempty"`
}

func (b CandidateBacklog) RankedCandidates() []CandidateEntry {
	ranked := append([]CandidateEntry(nil), b.Candidates...)
	sort.Slice(ranked, func(i, j int) bool {
		left := ranked[i]
		right := ranked[j]
		if left.ReadinessScore() != right.ReadinessScore() {
			return left.ReadinessScore() > right.ReadinessScore()
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
	MaxBlockers             int      `json:"max_blockers,omitempty"`
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

func (d EntryGateDecision) Summary() string {
	status := "HOLD"
	if d.Passed {
		status = "PASS"
	}
	return fmt.Sprintf(
		"%s: ready=%d blocked=%d missing_capabilities=%d missing_evidence=%d baseline_findings=%d",
		status,
		len(d.ReadyCandidateIDs),
		d.BlockerCount,
		len(d.MissingCapabilities),
		len(d.MissingEvidence),
		len(d.BaselineFindings),
	)
}

type CandidatePlanner struct{}

func (CandidatePlanner) EvaluateGate(backlog CandidateBacklog, gate EntryGate, baselineAudit *governance.ScopeFreezeAudit) EntryGateDecision {
	ranked := backlog.RankedCandidates()
	readyCandidates := make([]CandidateEntry, 0)
	blockedCandidates := make([]CandidateEntry, 0)
	providedCapabilities := map[string]struct{}{}
	providedEvidence := map[string]struct{}{}
	for _, candidate := range ranked {
		if candidate.Ready() {
			readyCandidates = append(readyCandidates, candidate)
			for _, capability := range candidate.Capabilities {
				providedCapabilities[capability] = struct{}{}
			}
			for _, item := range candidate.Evidence {
				providedEvidence[item] = struct{}{}
			}
		}
	}
	for _, candidate := range backlog.Candidates {
		if len(candidate.Blockers) > 0 {
			blockedCandidates = append(blockedCandidates, candidate)
		}
	}
	missingCapabilities := make([]string, 0)
	for _, capability := range gate.RequiredCapabilities {
		if _, ok := providedCapabilities[capability]; !ok {
			missingCapabilities = append(missingCapabilities, capability)
		}
	}
	missingEvidence := make([]string, 0)
	for _, item := range gate.RequiredEvidence {
		if _, ok := providedEvidence[item]; !ok {
			missingEvidence = append(missingEvidence, item)
		}
	}
	baselineFindings := baselineFindings(gate, baselineAudit)
	baselineReady := len(baselineFindings) == 0
	readyIDs := make([]string, 0, len(readyCandidates))
	for _, candidate := range readyCandidates {
		readyIDs = append(readyIDs, candidate.CandidateID)
	}
	blockedIDs := make([]string, 0, len(blockedCandidates))
	for _, candidate := range blockedCandidates {
		blockedIDs = append(blockedIDs, candidate.CandidateID)
	}
	passed := len(readyCandidates) >= gate.MinReadyCandidates &&
		len(blockedCandidates) <= gate.MaxBlockers &&
		len(missingCapabilities) == 0 &&
		len(missingEvidence) == 0 &&
		baselineReady
	return EntryGateDecision{
		GateID:              gate.GateID,
		Passed:              passed,
		ReadyCandidateIDs:   readyIDs,
		BlockedCandidateIDs: blockedIDs,
		MissingCapabilities: missingCapabilities,
		MissingEvidence:     missingEvidence,
		BaselineReady:       baselineReady,
		BaselineFindings:    baselineFindings,
		BlockerCount:        len(blockedCandidates),
	}
}

func baselineFindings(gate EntryGate, baselineAudit *governance.ScopeFreezeAudit) []string {
	if strings.TrimSpace(gate.RequiredBaselineVersion) == "" {
		return nil
	}
	if baselineAudit == nil {
		return []string{fmt.Sprintf("missing baseline audit for %s", gate.RequiredBaselineVersion)}
	}
	findings := make([]string, 0)
	if baselineAudit.Version != gate.RequiredBaselineVersion {
		findings = append(findings, fmt.Sprintf("baseline version mismatch: expected %s, got %s", gate.RequiredBaselineVersion, baselineAudit.Version))
	}
	if !baselineAudit.ReleaseReady() {
		findings = append(findings, fmt.Sprintf("baseline %s is not release ready (%.1f)", baselineAudit.Version, baselineAudit.ReadinessScore()))
	}
	return findings
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
			fmt.Sprintf("- %s: %s priority=%s owner=%s score=%d ready=%t", candidate.CandidateID, candidate.Title, candidate.Priority, candidate.Owner, candidate.ReadinessScore(), candidate.Ready()),
			fmt.Sprintf("  theme=%s outcome=%s capabilities=%s evidence=%s blockers=%s", candidate.Theme, candidate.Outcome, joinOrNone(candidate.Capabilities), joinOrNone(candidate.Evidence), joinOrNone(candidate.Blockers)),
			fmt.Sprintf("  validation=%s", candidate.ValidationCommand),
		)
		if len(candidate.Dependencies) > 0 {
			lines = append(lines, fmt.Sprintf("  dependencies=%s", strings.Join(candidate.Dependencies, ",")))
		}
		if len(candidate.EvidenceLinks) > 0 {
			lines = append(lines, "  evidence-links:")
			for _, link := range candidate.EvidenceLinks {
				qualifier := ""
				if strings.TrimSpace(link.Capability) != "" {
					qualifier = " capability=" + link.Capability
				}
				note := ""
				if strings.TrimSpace(link.Note) != "" {
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
		fmt.Sprintf("- Baseline ready: %t", decision.BaselineReady),
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
				ValidationCommand: "PYTHONPATH=src python3 -m pytest tests/test_design_system.py tests/test_console_ia.py tests/test_ui_review.py -q",
				Capabilities:      []string{"release-gate", "console-shell", "reporting"},
				Evidence:          []string{"acceptance-suite", "validation-report"},
				EvidenceLinks: []EvidenceLink{
					{Label: "design-system-audit", Target: "src/bigclaw/design_system.py", Capability: "release-gate", Note: "component inventory, accessibility, and UI acceptance coverage"},
					{Label: "console-ia-contract", Target: "src/bigclaw/console_ia.py", Capability: "release-gate", Note: "global navigation, top bar, filters, and state contracts"},
					{Label: "ui-review-pack", Target: "src/bigclaw/ui_review.py", Capability: "release-gate", Note: "review objectives, wireframes, interaction coverage, and open questions"},
					{Label: "ui-acceptance-tests", Target: "tests/test_design_system.py", Capability: "release-gate", Note: "role-permission, data accuracy, and performance audits"},
					{Label: "console-shell-tests", Target: "tests/test_console_ia.py", Capability: "release-gate", Note: "console shell and interaction draft release readiness"},
					{Label: "review-pack-tests", Target: "tests/test_ui_review.py", Capability: "release-gate", Note: "deterministic review packet validation"},
				},
			},
			{
				CandidateID:       "candidate-ops-hardening",
				Title:             "Operations command-center hardening",
				Theme:             "ops-command-center",
				Priority:          "P0",
				Owner:             "engineering-operations",
				Outcome:           "Promote queue control, approval handling, saved views, dashboard builder output, and replay evidence as one operator-ready command center.",
				ValidationCommand: "PYTHONPATH=src python3 -m pytest tests/test_control_center.py tests/test_operations.py tests/test_saved_views.py tests/test_evaluation.py -q && (cd bigclaw-go && go test ./internal/worker ./internal/workflow ./internal/scheduler)",
				Capabilities:      []string{"ops-control", "saved-views", "rollback-simulation"},
				Evidence:          []string{"weekly-review", "validation-report"},
				EvidenceLinks: []EvidenceLink{
					{Label: "command-center-src", Target: "src/bigclaw/operations.py", Capability: "ops-control", Note: "queue control center, dashboard builder, weekly review, and regression surfaces"},
					{Label: "command-center-tests", Target: "tests/test_control_center.py", Capability: "ops-control", Note: "queue control center validation"},
					{Label: "operations-tests", Target: "tests/test_operations.py", Capability: "ops-control", Note: "dashboard, weekly report, regression, and version-center coverage"},
					{Label: "approval-contract", Target: "src/bigclaw/execution_contract.py", Capability: "ops-control", Note: "approval permission and API role coverage contract"},
					{Label: "approval-workflow", Target: "src/bigclaw/workflow.py", Capability: "ops-control", Note: "approval workflow and closeout flow wiring"},
					{Label: "workflow-tests", Target: "bigclaw-go/internal/workflow/engine_test.go", Capability: "ops-control", Note: "acceptance gate and workpad journal validation"},
					{Label: "execution-flow-tests", Target: "bigclaw-go/internal/worker/runtime_test.go", Capability: "ops-control", Note: "execution handoff, closeout, and routed runtime evidence"},
					{Label: "saved-views-src", Target: "src/bigclaw/saved_views.py", Capability: "saved-views", Note: "saved views, digest subscriptions, and governed filters"},
					{Label: "saved-views-tests", Target: "tests/test_saved_views.py", Capability: "saved-views", Note: "saved-view audit coverage"},
					{Label: "simulation-src", Target: "src/bigclaw/evaluation.py", Capability: "rollback-simulation", Note: "simulation, replay, and comparison evidence"},
					{Label: "simulation-tests", Target: "tests/test_evaluation.py", Capability: "rollback-simulation", Note: "replay and benchmark validation"},
				},
			},
			{
				CandidateID:       "candidate-orchestration-rollout",
				Title:             "Agent orchestration rollout",
				Theme:             "agent-orchestration",
				Priority:          "P0",
				Owner:             "orchestration-office",
				Outcome:           "Carry entitlement-aware orchestration, handoff visibility, and commercialization proof into a candidate ready for release review.",
				ValidationCommand: "PYTHONPATH=src python3 -m pytest tests/test_orchestration.py tests/test_reports.py -q",
				Capabilities:      []string{"commercialization", "handoff", "pilot-rollout"},
				Evidence:          []string{"pilot-evidence", "validation-report"},
				EvidenceLinks: []EvidenceLink{
					{Label: "orchestration-plan-src", Target: "src/bigclaw/orchestration.py", Capability: "commercialization", Note: "cross-team orchestration, entitlement-aware policy, and handoff decisions"},
					{Label: "orchestration-report-src", Target: "src/bigclaw/reports.py", Capability: "commercialization", Note: "orchestration canvas, portfolio rollups, and narrative exports"},
					{Label: "orchestration-tests", Target: "tests/test_orchestration.py", Capability: "commercialization", Note: "handoff and policy decision validation"},
					{Label: "report-studio-tests", Target: "tests/test_reports.py", Capability: "commercialization", Note: "report exports and downstream evidence sharing"},
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

func (g WeeklyGoal) StatusRank() int {
	return goalStatusOrder[strings.ToLower(strings.TrimSpace(g.Status))]
}

func (g WeeklyGoal) IsComplete() bool {
	return strings.EqualFold(strings.TrimSpace(g.Status), "done")
}

func (g WeeklyGoal) IsAtRisk() bool {
	status := strings.ToLower(strings.TrimSpace(g.Status))
	return status == "at-risk" || status == "blocked"
}

type WeeklyExecutionPlan struct {
	WeekNumber   int          `json:"week_number"`
	Theme        string       `json:"theme"`
	Objective    string       `json:"objective"`
	ExitCriteria []string     `json:"exit_criteria,omitempty"`
	Deliverables []string     `json:"deliverables,omitempty"`
	Goals        []WeeklyGoal `json:"goals,omitempty"`
}

func (w WeeklyExecutionPlan) CompletedGoals() int {
	total := 0
	for _, goal := range w.Goals {
		if goal.IsComplete() {
			total++
		}
	}
	return total
}

func (w WeeklyExecutionPlan) TotalGoals() int {
	return len(w.Goals)
}

func (w WeeklyExecutionPlan) ProgressPercent() int {
	if len(w.Goals) == 0 {
		return 0
	}
	return int(float64(w.CompletedGoals()) / float64(len(w.Goals)) * 100)
}

func (w WeeklyExecutionPlan) AtRiskGoalIDs() []string {
	out := make([]string, 0)
	for _, goal := range w.Goals {
		if goal.IsAtRisk() {
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

func (p FourWeekExecutionPlan) TotalGoals() int {
	total := 0
	for _, week := range p.Weeks {
		total += week.TotalGoals()
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
	return int(float64(p.CompletedGoals()) / float64(p.TotalGoals()) * 100)
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
	counts := map[string]int{}
	for _, week := range p.Weeks {
		for _, goal := range week.Goals {
			counts[goal.Status]++
		}
	}
	return counts
}

func (p FourWeekExecutionPlan) Validate() error {
	weekNumbers := make([]int, 0, len(p.Weeks))
	for _, week := range p.Weeks {
		weekNumbers = append(weekNumbers, week.WeekNumber)
	}
	expected := []int{1, 2, 3, 4}
	if len(weekNumbers) != len(expected) {
		return fmt.Errorf("Four-week execution plans must include weeks 1 through 4 in order")
	}
	for index := range expected {
		if weekNumbers[index] != expected[index] {
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

func RenderFourWeekExecutionReport(plan FourWeekExecutionPlan) (string, error) {
	if err := plan.Validate(); err != nil {
		return "", err
	}
	statusCounts := plan.GoalStatusCounts()
	lines := []string{
		"# Four-Week Execution Plan",
		"",
		fmt.Sprintf("- Plan: %s %s", plan.PlanID, plan.Title),
		fmt.Sprintf("- Owner: %s", plan.Owner),
		fmt.Sprintf("- Start date: %s", plan.StartDate),
		fmt.Sprintf("- Overall progress: %d/%d goals complete (%d%%)", plan.CompletedGoals(), plan.TotalGoals(), plan.OverallProgressPercent()),
		fmt.Sprintf("- At-risk weeks: %s", joinIntsOrNone(plan.AtRiskWeeks())),
		fmt.Sprintf("- Goal status counts: done=%d on-track=%d at-risk=%d blocked=%d not-started=%d", statusCounts["done"], statusCounts["on-track"], statusCounts["at-risk"], statusCounts["blocked"], statusCounts["not-started"]),
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
				fmt.Sprintf("  - %s: %s owner=%s status=%s metric=%s current=%s target=%s", goal.GoalID, goal.Title, goal.Owner, goal.Status, goal.SuccessMetric, ternaryString(goal.CurrentValue != "", goal.CurrentValue, "n/a"), goal.TargetValue),
				fmt.Sprintf("    dependencies=%s risks=%s", joinOrNone(goal.Dependencies), joinOrNone(goal.Risks)),
			)
		}
	}
	return strings.Join(lines, "\n"), nil
}

func MarshalRoundTrip[T any](value T, target *T) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func joinIntsOrNone(values []int) string {
	if len(values) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%d", value))
	}
	return strings.Join(parts, ", ")
}

func ternaryString(condition bool, left, right string) string {
	if condition {
		return left
	}
	return right
}

func min(left, right int) int {
	if left < right {
		return left
	}
	return right
}
