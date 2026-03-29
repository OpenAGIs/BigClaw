package planning

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
	evidenceBonus := len(c.Evidence)
	if evidenceBonus > 3 {
		evidenceBonus = 3
	}
	score := base + (evidenceBonus * 5) - dependencyPenalty - blockerPenalty
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
	out := append([]CandidateEntry(nil), b.Candidates...)
	sort.SliceStable(out, func(i, j int) bool {
		left := out[i].ReadinessScore()
		right := out[j].ReadinessScore()
		if left == right {
			return out[i].CandidateID < out[j].CandidateID
		}
		return left > right
	})
	return out
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

func (CandidatePlanner) EvaluateGate(backlog CandidateBacklog, gate EntryGate, baseline *governance.ScopeFreezeAudit) EntryGateDecision {
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
			for _, evidence := range candidate.Evidence {
				providedEvidence[evidence] = struct{}{}
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
	for _, evidence := range gate.RequiredEvidence {
		if _, ok := providedEvidence[evidence]; !ok {
			missingEvidence = append(missingEvidence, evidence)
		}
	}
	baselineFindings := baselineFindings(gate, baseline)
	baselineReady := len(baselineFindings) == 0
	decision := EntryGateDecision{
		GateID:              gate.GateID,
		Passed:              len(readyCandidates) >= gate.MinReadyCandidates && len(blockedCandidates) <= gate.MaxBlockers && len(missingCapabilities) == 0 && len(missingEvidence) == 0 && baselineReady,
		BaselineReady:       baselineReady,
		BaselineFindings:    baselineFindings,
		BlockerCount:        len(blockedCandidates),
		MissingCapabilities: missingCapabilities,
		MissingEvidence:     missingEvidence,
	}
	for _, candidate := range readyCandidates {
		decision.ReadyCandidateIDs = append(decision.ReadyCandidateIDs, candidate.CandidateID)
	}
	for _, candidate := range blockedCandidates {
		decision.BlockedCandidateIDs = append(decision.BlockedCandidateIDs, candidate.CandidateID)
	}
	return decision
}

func baselineFindings(gate EntryGate, baseline *governance.ScopeFreezeAudit) []string {
	if strings.TrimSpace(gate.RequiredBaselineVersion) == "" {
		return nil
	}
	if baseline == nil {
		return []string{fmt.Sprintf("missing baseline audit for %s", gate.RequiredBaselineVersion)}
	}
	findings := make([]string, 0)
	if baseline.Version != gate.RequiredBaselineVersion {
		findings = append(findings, fmt.Sprintf("baseline version mismatch: expected %s, got %s", gate.RequiredBaselineVersion, baseline.Version))
	}
	if !baseline.ReleaseReady() {
		findings = append(findings, fmt.Sprintf("baseline %s is not release ready (%.1f)", baseline.Version, baseline.ReadinessScore()))
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
			},
			{
				CandidateID:       "candidate-ops-hardening",
				Title:             "Operations command-center hardening",
				Theme:             "ops-command-center",
				Priority:          "P0",
				Owner:             "engineering-operations",
				Outcome:           "Promote queue control, approval handling, saved views, dashboard builder output, and replay evidence as one operator-ready command center.",
				ValidationCommand: "PYTHONPATH=src python3 -m pytest tests/test_control_center.py tests/test_operations.py tests/test_saved_views.py tests/test_workflow.py tests/test_execution_flow.py tests/test_evaluation.py -q",
				Capabilities:      []string{"ops-control", "saved-views", "rollback-simulation"},
				Evidence:          []string{"weekly-review", "validation-report"},
				EvidenceLinks: []EvidenceLink{
					{Label: "command-center-src", Target: "src/bigclaw/operations.py", Capability: "ops-control", Note: "queue control center, dashboard builder, weekly review, and regression surfaces"},
					{Label: "report-studio-tests", Target: "tests/test_reports.py", Capability: "commercialization", Note: "report exports and downstream evidence sharing"},
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
			},
		},
	}
}

func BuildV3EntryGate() EntryGate {
	return EntryGate{
		GateID:                  "gate-v3-entry",
		Name:                    "v3 entry gate",
		MinReadyCandidates:      2,
		RequiredCapabilities:    []string{"release-gate", "ops-control", "commercialization"},
		RequiredEvidence:        []string{"validation-report"},
		RequiredBaselineVersion: "v4.0",
		MaxBlockers:             0,
	}
}

func MarshalJSON[T any](value T) ([]byte, error) {
	return json.Marshal(value)
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}
