package regression

import (
	"strings"
	"testing"
)

func TestUIReviewPackContractStaysAligned(t *testing.T) {
	root := repoRoot(t)
	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "../src/bigclaw/ui_review.py",
			substrings: []string{
				`issue_id="BIG-4204"`,
				`title="UI评审包输出"`,
				`version="v4.0-design-sprint"`,
				`requires_reviewer_checklist=True`,
				`requires_decision_log=True`,
				`requires_role_matrix=True`,
				`requires_signoff_log=True`,
				`requires_blocker_log=True`,
				`requires_blocker_timeline=True`,
				`objective_id="obj-queue-governance"`,
				`surface_id="wf-run-detail"`,
				`flow_id="flow-triage-handoff"`,
				`question_id="oq-handoff-evidence"`,
				`item_id="chk-run-replay-context"`,
				`decision_id="dec-queue-vp-summary"`,
				`assignment_id="role-run-detail-eng-lead"`,
				`signoff_id="sig-run-detail-eng-lead"`,
				`blocker_id="blk-run-detail-copy-final"`,
				`event_id="evt-run-detail-copy-escalated"`,
				`freeze_reason="Allow the design sprint review pack to ship while tracked copy cleanup lands in the next critique."`,
				`artifact_ref="wf-run-detail#copy-v5"`,
				`"# UI Review Pack"`,
				`"## Review Summary Board"`,
				`"## Objective Coverage Board"`,
				`"## Persona Readiness Board"`,
				`"## Wireframe Readiness Board"`,
				`"## Interaction Coverage Board"`,
				`"## Open Question Tracker"`,
				`"## Checklist Traceability Board"`,
				`"## Decision Follow-up Tracker"`,
				`"## Role Coverage Board"`,
				`"## Signoff Dependency Board"`,
				`"## Sign-off SLA Dashboard"`,
				`"## Sign-off Reminder Queue"`,
				`"## Reminder Cadence Board"`,
				`"## Escalation Dashboard"`,
				`"## Escalation Handoff Ledger"`,
				`"## Handoff Ack Ledger"`,
				`"## Review Exceptions"`,
				`"## Owner Workload Board"`,
				`"## Blocker Log"`,
				`"## Blocker Timeline"`,
				`"## Review Freeze Exception Board"`,
				`"## Freeze Approval Trail"`,
				`"## Freeze Renewal Tracker"`,
				`"## Review Exception Matrix"`,
				`"## Audit Density Board"`,
				`"## Owner Review Queue"`,
				`"## Blocker Timeline Summary"`,
				`markdown_path = str(base / f"{slug}-review-pack.md")`,
				`html_path = str(base / f"{slug}-review-pack.html")`,
				`review_summary_board_path = str(base / f"{slug}-review-summary-board.md")`,
				`interaction_coverage_board_path = str(base / f"{slug}-interaction-coverage-board.md")`,
				`signoff_sla_dashboard_path = str(base / f"{slug}-signoff-sla-dashboard.md")`,
				`escalation_handoff_ledger_path = str(base / f"{slug}-escalation-handoff-ledger.md")`,
				`freeze_renewal_tracker_path = str(base / f"{slug}-freeze-renewal-tracker.md")`,
				`audit_density_board_path = str(base / f"{slug}-audit-density-board.md")`,
				`blocker_timeline_summary_path = str(base / f"{slug}-blocker-timeline-summary.md")`,
				`Path(markdown_path).write_text(render_ui_review_pack_report(pack, audit))`,
				`Path(html_path).write_text(render_ui_review_pack_html(pack, audit))`,
				`Path(owner_workload_board_path).write_text(render_ui_review_owner_workload_board(pack))`,
				`Path(blocker_timeline_summary_path).write_text(render_ui_review_blocker_timeline_summary(pack))`,
			},
		},
		{
			path: "../src/bigclaw/planning.py",
			substrings: []string{
				`tests/test_design_system.py`,
				`tests/test_console_ia.py`,
				`cd bigclaw-go && go test ./internal/regression -run TestUIReviewPackContractStaysAligned`,
				`target="bigclaw-go/internal/regression/ui_review_pack_contract_test.go"`,
				`note="Go regression coverage for the checked-in review-pack contract"`,
			},
		},
	}

	for _, tc := range cases {
		contents := readRepoFile(t, root, tc.path)
		for _, needle := range tc.substrings {
			if !strings.Contains(contents, needle) {
				t.Fatalf("%s missing substring %q", tc.path, needle)
			}
		}
	}
}
