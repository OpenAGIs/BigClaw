package uireview

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const big4204ReviewPackJSON = `{
  "issue_id": "BIG-4204",
  "title": "UI评审包输出",
  "version": "v4.0-design-sprint",
  "objectives": [
    {
      "objective_id": "obj-overview-decision",
      "title": "Validate the executive overview narrative and drill-down posture",
      "persona": "VP Eng",
      "outcome": "Leadership can confirm the overview page balances KPI density with investigation entry points.",
      "success_signal": "Reviewers agree the overview supports release, risk, and queue drill-down without extra walkthroughs.",
      "priority": "P0",
      "dependencies": ["BIG-4203", "OPE-132"]
    },
    {
      "objective_id": "obj-queue-governance",
      "title": "Confirm queue control actions and approval posture",
      "persona": "Platform Admin",
      "outcome": "Operators can assess batch approvals, audit visibility, and denial paths from one frame.",
      "success_signal": "The queue frame clearly shows allowed actions, denied roles, and audit expectations.",
      "priority": "P0",
      "dependencies": ["BIG-4203", "OPE-131", "OPE-132"]
    },
    {
      "objective_id": "obj-run-detail-investigation",
      "title": "Validate replay and audit investigation flow",
      "persona": "Eng Lead",
      "outcome": "Run detail reviewers can trace evidence, replay context, and escalation actions in one surface.",
      "success_signal": "The run-detail frame makes failure replay and escalation decisions reviewable without hidden dependencies.",
      "priority": "P0",
      "dependencies": ["BIG-4203", "OPE-72", "OPE-73"]
    },
    {
      "objective_id": "obj-triage-handoff",
      "title": "Confirm triage and cross-team handoff readiness",
      "persona": "Cross-Team Operator",
      "outcome": "Reviewers can evaluate assignment, handoff, and queue-state transitions as one operator journey.",
      "success_signal": "The triage frame exposes action states, owner switches, and handoff exceptions explicitly.",
      "priority": "P0",
      "dependencies": ["BIG-4203", "OPE-76", "OPE-79", "OPE-132"]
    }
  ],
  "wireframes": [
    {
      "surface_id": "wf-overview",
      "name": "Overview command deck",
      "device": "desktop",
      "entry_point": "/overview",
      "primary_blocks": ["top bar", "kpi strip", "risk panel", "drill-down table"],
      "review_notes": ["Confirm metric density and executive scan path.", "Check alert prominence versus weekly summary card."]
    },
    {
      "surface_id": "wf-queue",
      "name": "Queue control center",
      "device": "desktop",
      "entry_point": "/queue",
      "primary_blocks": ["approval queue", "selection toolbar", "filters", "audit rail"],
      "review_notes": ["Validate batch-approve CTA hierarchy.", "Review denied-role behavior for non-operator personas."]
    },
    {
      "surface_id": "wf-run-detail",
      "name": "Run detail and replay",
      "device": "desktop",
      "entry_point": "/runs/detail",
      "primary_blocks": ["timeline", "artifact drawer", "replay controls", "audit notes"],
      "review_notes": ["Check replay mode discoverability.", "Ensure escalation path is visible next to audit evidence."]
    },
    {
      "surface_id": "wf-triage",
      "name": "Triage and handoff board",
      "device": "desktop",
      "entry_point": "/triage",
      "primary_blocks": ["severity lanes", "bulk actions", "handoff panel", "ownership history"],
      "review_notes": ["Validate cross-team operator workflow.", "Confirm exception path for denied escalation."]
    }
  ],
  "interactions": [
    {
      "flow_id": "flow-overview-drilldown",
      "name": "Overview to investigation drill-down",
      "trigger": "VP Eng selects a KPI card or blocker cluster on the overview page",
      "system_response": "The console pivots into the matching queue or run-detail slice while preserving context filters.",
      "states": ["default", "focus", "handoff-ready"],
      "exceptions": ["Warn when the requested slice is permission-denied.", "Show fallback summary when no matching runs exist."]
    },
    {
      "flow_id": "flow-queue-bulk-approval",
      "name": "Queue batch approval review",
      "trigger": "Platform Admin selects multiple tasks and opens the bulk approval toolbar",
      "system_response": "The queue shows approval scope, audit consequence, and denied-role messaging before submit.",
      "states": ["default", "selection", "confirming", "success"],
      "exceptions": ["Disable submit when tasks cross unauthorized scopes.", "Route to audit timeline when approval policy changes mid-flow."]
    },
    {
      "flow_id": "flow-run-replay",
      "name": "Run replay with evidence audit",
      "trigger": "Eng Lead switches replay mode on a failed run",
      "system_response": "The page updates the timeline, artifacts, and escalation controls while keeping the audit trail visible.",
      "states": ["default", "replay", "compare", "escalated"],
      "exceptions": ["Show replay-unavailable state for incomplete artifacts.", "Require escalation reason before handoff."]
    },
    {
      "flow_id": "flow-triage-handoff",
      "name": "Triage ownership reassignment and handoff",
      "trigger": "Cross-Team Operator bulk-assigns a finding set or opens the handoff panel",
      "system_response": "The triage board updates owner, workflow, and handoff evidence in one acknowledgement step.",
      "states": ["default", "selected", "handoff", "completed"],
      "exceptions": ["Block handoff when reviewer coverage is incomplete.", "Record denied-role attempt in the audit summary."]
    }
  ],
  "open_questions": [
    {
      "question_id": "oq-role-density",
      "theme": "role-matrix",
      "question": "Should VP Eng see queue batch controls in read-only form or be routed to a summary-only state?",
      "owner": "product-experience",
      "impact": "Changes denial-path copy, button placement, and review criteria for queue and triage pages.",
      "status": "open"
    },
    {
      "question_id": "oq-alert-priority",
      "theme": "information-architecture",
      "question": "Should regression alerts outrank approval alerts in the top bar for the design sprint prototype?",
      "owner": "engineering-operations",
      "impact": "Affects alert hierarchy and the scan path used in the overview and triage reviews.",
      "status": "open"
    },
    {
      "question_id": "oq-handoff-evidence",
      "theme": "handoff",
      "question": "How much ownership history must stay visible before the run-detail and triage pages collapse older audit entries?",
      "owner": "orchestration-office",
      "impact": "Shapes the default density of the audit rail and the threshold for the review-ready packet.",
      "status": "open"
    }
  ],
  "reviewer_checklist": [
    {
      "item_id": "chk-overview-kpi-scan",
      "surface_id": "wf-overview",
      "prompt": "Verify the KPI strip still supports one-screen executive scanning before drill-down.",
      "owner": "VP Eng",
      "status": "ready",
      "evidence_links": ["wf-overview", "flow-overview-drilldown"],
      "notes": "Use the overview card hierarchy as the primary decision frame."
    },
    {
      "item_id": "chk-overview-alert-hierarchy",
      "surface_id": "wf-overview",
      "prompt": "Confirm alert priority is readable when approvals and regressions compete for attention.",
      "owner": "engineering-operations",
      "status": "open",
      "evidence_links": ["wf-overview", "oq-alert-priority"],
      "notes": ""
    },
    {
      "item_id": "chk-queue-batch-approval",
      "surface_id": "wf-queue",
      "prompt": "Check that batch approval clearly communicates scope, denial paths, and audit consequences.",
      "owner": "Platform Admin",
      "status": "ready",
      "evidence_links": ["wf-queue", "flow-queue-bulk-approval"],
      "notes": ""
    },
    {
      "item_id": "chk-queue-role-density",
      "surface_id": "wf-queue",
      "prompt": "Validate whether VP Eng should get a summary-only queue variant instead of operator controls.",
      "owner": "product-experience",
      "status": "open",
      "evidence_links": ["wf-queue", "oq-role-density"],
      "notes": ""
    },
    {
      "item_id": "chk-run-replay-context",
      "surface_id": "wf-run-detail",
      "prompt": "Ensure replay, compare, and escalation states remain distinguishable without narration.",
      "owner": "Eng Lead",
      "status": "ready",
      "evidence_links": ["wf-run-detail", "flow-run-replay"],
      "notes": ""
    },
    {
      "item_id": "chk-run-audit-density",
      "surface_id": "wf-run-detail",
      "prompt": "Confirm the audit rail retains enough ownership history before collapsing older entries.",
      "owner": "orchestration-office",
      "status": "open",
      "evidence_links": ["wf-run-detail", "oq-handoff-evidence"],
      "notes": ""
    },
    {
      "item_id": "chk-triage-handoff-clarity",
      "surface_id": "wf-triage",
      "prompt": "Check that cross-team handoff consequences are explicit before ownership changes commit.",
      "owner": "Cross-Team Operator",
      "status": "ready",
      "evidence_links": ["wf-triage", "flow-triage-handoff"],
      "notes": ""
    },
    {
      "item_id": "chk-triage-bulk-assign",
      "surface_id": "wf-triage",
      "prompt": "Validate bulk assignment visibility without burying the audit context.",
      "owner": "Platform Admin",
      "status": "ready",
      "evidence_links": ["wf-triage", "flow-triage-handoff"],
      "notes": ""
    }
  ],
  "requires_reviewer_checklist": true,
  "decision_log": [
    {
      "decision_id": "dec-overview-alert-stack",
      "surface_id": "wf-overview",
      "owner": "product-experience",
      "summary": "Keep approval and regression alerts in one stacked priority rail.",
      "rationale": "Reviewers need one comparison lane before jumping into queue or triage surfaces.",
      "status": "accepted",
      "follow_up": ""
    },
    {
      "decision_id": "dec-queue-vp-summary",
      "surface_id": "wf-queue",
      "owner": "VP Eng",
      "summary": "Route VP Eng to a summary-first queue state instead of operator controls.",
      "rationale": "The VP Eng persona needs queue visibility without accidental action affordances.",
      "status": "proposed",
      "follow_up": "Resolve after the next design critique with policy owners."
    },
    {
      "decision_id": "dec-run-detail-audit-rail",
      "surface_id": "wf-run-detail",
      "owner": "Eng Lead",
      "summary": "Keep audit evidence visible beside replay controls in all replay states.",
      "rationale": "Replay decisions are inseparable from audit context and escalation evidence.",
      "status": "accepted",
      "follow_up": ""
    },
    {
      "decision_id": "dec-triage-handoff-density",
      "surface_id": "wf-triage",
      "owner": "Cross-Team Operator",
      "summary": "Preserve ownership history in the triage rail until handoff is acknowledged.",
      "rationale": "Operators need a stable handoff trail before collapsing older events.",
      "status": "accepted",
      "follow_up": ""
    }
  ],
  "requires_decision_log": true,
  "role_matrix": [
    {
      "assignment_id": "role-overview-vp-eng",
      "surface_id": "wf-overview",
      "role": "VP Eng",
      "responsibilities": ["approve overview scan path", "validate KPI-to-drilldown narrative"],
      "checklist_item_ids": ["chk-overview-kpi-scan"],
      "decision_ids": ["dec-overview-alert-stack"],
      "status": "ready"
    },
    {
      "assignment_id": "role-overview-eng-ops",
      "surface_id": "wf-overview",
      "role": "engineering-operations",
      "responsibilities": ["review alert priority posture"],
      "checklist_item_ids": ["chk-overview-alert-hierarchy"],
      "decision_ids": ["dec-overview-alert-stack"],
      "status": "open"
    },
    {
      "assignment_id": "role-queue-platform-admin",
      "surface_id": "wf-queue",
      "role": "Platform Admin",
      "responsibilities": ["validate batch-approval copy", "confirm permission posture"],
      "checklist_item_ids": ["chk-queue-batch-approval"],
      "decision_ids": ["dec-queue-vp-summary"],
      "status": "ready"
    },
    {
      "assignment_id": "role-queue-product-experience",
      "surface_id": "wf-queue",
      "role": "product-experience",
      "responsibilities": ["tune summary-only VP variant"],
      "checklist_item_ids": ["chk-queue-role-density"],
      "decision_ids": ["dec-queue-vp-summary"],
      "status": "open"
    },
    {
      "assignment_id": "role-run-detail-eng-lead",
      "surface_id": "wf-run-detail",
      "role": "Eng Lead",
      "responsibilities": ["approve replay-state clarity", "confirm escalation adjacency"],
      "checklist_item_ids": ["chk-run-replay-context"],
      "decision_ids": ["dec-run-detail-audit-rail"],
      "status": "ready"
    },
    {
      "assignment_id": "role-run-detail-orchestration-office",
      "surface_id": "wf-run-detail",
      "role": "orchestration-office",
      "responsibilities": ["review audit density threshold"],
      "checklist_item_ids": ["chk-run-audit-density"],
      "decision_ids": ["dec-run-detail-audit-rail"],
      "status": "open"
    },
    {
      "assignment_id": "role-triage-cross-team-operator",
      "surface_id": "wf-triage",
      "role": "Cross-Team Operator",
      "responsibilities": ["approve handoff clarity", "validate ownership transition story"],
      "checklist_item_ids": ["chk-triage-handoff-clarity"],
      "decision_ids": ["dec-triage-handoff-density"],
      "status": "ready"
    },
    {
      "assignment_id": "role-triage-platform-admin",
      "surface_id": "wf-triage",
      "role": "Platform Admin",
      "responsibilities": ["confirm bulk-assign audit visibility"],
      "checklist_item_ids": ["chk-triage-bulk-assign"],
      "decision_ids": ["dec-triage-handoff-density"],
      "status": "ready"
    }
  ],
  "requires_role_matrix": true,
  "signoff_log": [
    {
      "signoff_id": "sig-overview-vp-eng",
      "assignment_id": "role-overview-vp-eng",
      "surface_id": "wf-overview",
      "role": "VP Eng",
      "status": "approved",
      "required": true,
      "evidence_links": ["chk-overview-kpi-scan", "dec-overview-alert-stack"],
      "notes": "Overview narrative approved for design sprint review.",
      "requested_at": "2026-03-10T09:00:00Z",
      "due_at": "2026-03-12T18:00:00Z",
      "escalation_owner": "design-program-manager",
      "sla_status": "met",
      "reminder_status": "scheduled"
    },
    {
      "signoff_id": "sig-queue-platform-admin",
      "assignment_id": "role-queue-platform-admin",
      "surface_id": "wf-queue",
      "role": "Platform Admin",
      "status": "approved",
      "required": true,
      "evidence_links": ["chk-queue-batch-approval", "dec-queue-vp-summary"],
      "notes": "Queue control actions meet operator review criteria.",
      "requested_at": "2026-03-10T11:00:00Z",
      "due_at": "2026-03-13T18:00:00Z",
      "escalation_owner": "platform-ops-manager",
      "sla_status": "met",
      "reminder_status": "scheduled"
    },
    {
      "signoff_id": "sig-run-detail-eng-lead",
      "assignment_id": "role-run-detail-eng-lead",
      "surface_id": "wf-run-detail",
      "role": "Eng Lead",
      "status": "pending",
      "required": true,
      "evidence_links": ["chk-run-replay-context", "dec-run-detail-audit-rail"],
      "notes": "Waiting for final replay-state copy review.",
      "requested_at": "2026-03-12T11:00:00Z",
      "due_at": "2026-03-15T18:00:00Z",
      "escalation_owner": "engineering-director",
      "sla_status": "at-risk",
      "reminder_owner": "design-program-manager",
      "reminder_channel": "slack",
      "last_reminder_at": "2026-03-14T09:45:00Z",
      "next_reminder_at": "2026-03-15T10:00:00Z",
      "reminder_cadence": "daily",
      "reminder_status": "scheduled"
    },
    {
      "signoff_id": "sig-triage-cross-team-operator",
      "assignment_id": "role-triage-cross-team-operator",
      "surface_id": "wf-triage",
      "role": "Cross-Team Operator",
      "status": "approved",
      "required": true,
      "evidence_links": ["chk-triage-handoff-clarity", "dec-triage-handoff-density"],
      "notes": "Cross-team handoff flow approved for prototype review.",
      "requested_at": "2026-03-11T14:00:00Z",
      "due_at": "2026-03-13T12:00:00Z",
      "escalation_owner": "cross-team-program-manager",
      "sla_status": "met",
      "reminder_status": "scheduled"
    }
  ],
  "requires_signoff_log": true,
  "blocker_log": [
    {
      "blocker_id": "blk-run-detail-copy-final",
      "surface_id": "wf-run-detail",
      "signoff_id": "sig-run-detail-eng-lead",
      "owner": "product-experience",
      "summary": "Replay-state copy still needs final wording review before Eng Lead signoff can close.",
      "status": "open",
      "severity": "medium",
      "escalation_owner": "design-program-manager",
      "next_action": "Review replay-state copy with Eng Lead and update the run-detail frame in the next critique.",
      "freeze_exception": true,
      "freeze_owner": "release-director",
      "freeze_until": "2026-03-18T18:00:00Z",
      "freeze_reason": "Allow the design sprint review pack to ship while tracked copy cleanup lands in the next critique.",
      "freeze_approved_by": "release-director",
      "freeze_approved_at": "2026-03-14T08:30:00Z",
      "freeze_renewal_owner": "release-director",
      "freeze_renewal_by": "2026-03-17T12:00:00Z",
      "freeze_renewal_status": "review-needed"
    }
  ],
  "requires_blocker_log": true,
  "blocker_timeline": [
    {
      "event_id": "evt-run-detail-copy-opened",
      "blocker_id": "blk-run-detail-copy-final",
      "actor": "product-experience",
      "status": "opened",
      "summary": "Captured the final replay-state copy gap during design sprint prep.",
      "timestamp": "2026-03-13T10:00:00Z",
      "next_action": "Draft updated replay labels before the Eng Lead review.",
      "ack_status": "pending"
    },
    {
      "event_id": "evt-run-detail-copy-escalated",
      "blocker_id": "blk-run-detail-copy-final",
      "actor": "design-program-manager",
      "status": "escalated",
      "summary": "Scheduled a joint wording review with Eng Lead and product-experience to close the signoff blocker.",
      "timestamp": "2026-03-14T09:30:00Z",
      "next_action": "Refresh the run-detail frame annotations after the wording review completes.",
      "handoff_from": "product-experience",
      "handoff_to": "Eng Lead",
      "channel": "design-critique",
      "artifact_ref": "wf-run-detail#copy-v5",
      "ack_owner": "Eng Lead",
      "ack_at": "2026-03-14T10:15:00Z",
      "ack_status": "acknowledged"
    }
  ],
  "requires_blocker_timeline": true
}`

func BuildBIG4204ReviewPack() UIReviewPack {
	var pack UIReviewPack
	if err := json.Unmarshal([]byte(big4204ReviewPackJSON), &pack); err != nil {
		panic(fmt.Sprintf("unmarshal BIG-4204 review pack: %v", err))
	}
	return pack.ensureDefaults()
}

func isOpenStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "approved", "accepted", "resolved", "waived", "deferred", "ready", "closed", "done", "met":
		return false
	default:
		return true
	}
}

func joinOrNone(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ",")
}

func sortStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}

func signoffByAssignment(pack UIReviewPack) map[string]ReviewSignoff {
	out := map[string]ReviewSignoff{}
	for _, signoff := range pack.SignoffLog {
		out[signoff.AssignmentID] = signoff
	}
	return out
}

func blockersBySignoff(pack UIReviewPack) map[string][]ReviewBlocker {
	out := map[string][]ReviewBlocker{}
	for _, blocker := range pack.BlockerLog {
		out[blocker.SignoffID] = append(out[blocker.SignoffID], blocker)
	}
	return out
}

func latestBlockerEvents(pack UIReviewPack) map[string]ReviewBlockerEvent {
	out := map[string]ReviewBlockerEvent{}
	for _, event := range pack.BlockerTimeline {
		current, ok := out[event.BlockerID]
		if !ok || current.Timestamp < event.Timestamp {
			out[event.BlockerID] = event
		}
	}
	return out
}
