package uireview

import (
	"os"
	"path/filepath"
)

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func WriteUIReviewPackBundle(rootDir string, pack UIReviewPack) (UIReviewPackArtifacts, error) {
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		return UIReviewPackArtifacts{}, err
	}
	slug := sanitizeSlug(pack.IssueID)
	artifacts := UIReviewPackArtifacts{
		RootDir:                        rootDir,
		MarkdownPath:                   filepath.Join(rootDir, slug+"-review-pack.md"),
		HTMLPath:                       filepath.Join(rootDir, slug+"-review-pack.html"),
		DecisionLogPath:                filepath.Join(rootDir, slug+"-decision-log.md"),
		ReviewSummaryBoardPath:         filepath.Join(rootDir, slug+"-review-summary-board.md"),
		ObjectiveCoverageBoardPath:     filepath.Join(rootDir, slug+"-objective-coverage-board.md"),
		PersonaReadinessBoardPath:      filepath.Join(rootDir, slug+"-persona-readiness-board.md"),
		WireframeReadinessBoardPath:    filepath.Join(rootDir, slug+"-wireframe-readiness-board.md"),
		InteractionCoverageBoardPath:   filepath.Join(rootDir, slug+"-interaction-coverage-board.md"),
		OpenQuestionTrackerPath:        filepath.Join(rootDir, slug+"-open-question-tracker.md"),
		ChecklistTraceabilityBoardPath: filepath.Join(rootDir, slug+"-checklist-traceability-board.md"),
		DecisionFollowupTrackerPath:    filepath.Join(rootDir, slug+"-decision-followup-tracker.md"),
		RoleMatrixPath:                 filepath.Join(rootDir, slug+"-role-matrix.md"),
		RoleCoverageBoardPath:          filepath.Join(rootDir, slug+"-role-coverage-board.md"),
		SignoffDependencyBoardPath:     filepath.Join(rootDir, slug+"-signoff-dependency-board.md"),
		SignoffLogPath:                 filepath.Join(rootDir, slug+"-signoff-log.md"),
		SignoffSLADashboardPath:        filepath.Join(rootDir, slug+"-signoff-sla-dashboard.md"),
		SignoffReminderQueuePath:       filepath.Join(rootDir, slug+"-signoff-reminder-queue.md"),
		ReminderCadenceBoardPath:       filepath.Join(rootDir, slug+"-reminder-cadence-board.md"),
		SignoffBreachBoardPath:         filepath.Join(rootDir, slug+"-signoff-breach-board.md"),
		EscalationDashboardPath:        filepath.Join(rootDir, slug+"-escalation-dashboard.md"),
		EscalationHandoffLedgerPath:    filepath.Join(rootDir, slug+"-escalation-handoff-ledger.md"),
		HandoffAckLedgerPath:           filepath.Join(rootDir, slug+"-handoff-ack-ledger.md"),
		OwnerEscalationDigestPath:      filepath.Join(rootDir, slug+"-owner-escalation-digest.md"),
		OwnerWorkloadBoardPath:         filepath.Join(rootDir, slug+"-owner-workload-board.md"),
		BlockerLogPath:                 filepath.Join(rootDir, slug+"-blocker-log.md"),
		BlockerTimelinePath:            filepath.Join(rootDir, slug+"-blocker-timeline.md"),
		FreezeExceptionBoardPath:       filepath.Join(rootDir, slug+"-freeze-exception-board.md"),
		FreezeApprovalTrailPath:        filepath.Join(rootDir, slug+"-freeze-approval-trail.md"),
		FreezeRenewalTrackerPath:       filepath.Join(rootDir, slug+"-freeze-renewal-tracker.md"),
		ExceptionLogPath:               filepath.Join(rootDir, slug+"-exception-log.md"),
		ExceptionMatrixPath:            filepath.Join(rootDir, slug+"-exception-matrix.md"),
		AuditDensityBoardPath:          filepath.Join(rootDir, slug+"-audit-density-board.md"),
		OwnerReviewQueuePath:           filepath.Join(rootDir, slug+"-owner-review-queue.md"),
		BlockerTimelineSummaryPath:     filepath.Join(rootDir, slug+"-blocker-timeline-summary.md"),
	}
	audit := UIReviewPackAuditor{}.Audit(pack)
	writes := map[string]string{
		artifacts.MarkdownPath:                   RenderUIReviewPackReport(pack, audit),
		artifacts.HTMLPath:                       RenderUIReviewPackHTML(pack, audit),
		artifacts.DecisionLogPath:                RenderUIReviewDecisionLog(pack),
		artifacts.ReviewSummaryBoardPath:         RenderUIReviewReviewSummaryBoard(pack),
		artifacts.ObjectiveCoverageBoardPath:     RenderUIReviewObjectiveCoverageBoard(pack),
		artifacts.PersonaReadinessBoardPath:      RenderUIReviewPersonaReadinessBoard(pack),
		artifacts.WireframeReadinessBoardPath:    RenderUIReviewWireframeReadinessBoard(pack),
		artifacts.InteractionCoverageBoardPath:   RenderUIReviewInteractionCoverageBoard(pack),
		artifacts.OpenQuestionTrackerPath:        RenderUIReviewOpenQuestionTracker(pack),
		artifacts.ChecklistTraceabilityBoardPath: RenderUIReviewChecklistTraceabilityBoard(pack),
		artifacts.DecisionFollowupTrackerPath:    RenderUIReviewDecisionFollowupTracker(pack),
		artifacts.RoleMatrixPath:                 RenderUIReviewRoleMatrix(pack),
		artifacts.RoleCoverageBoardPath:          RenderUIReviewRoleCoverageBoard(pack),
		artifacts.SignoffDependencyBoardPath:     RenderUIReviewSignoffDependencyBoard(pack),
		artifacts.SignoffLogPath:                 RenderUIReviewSignoffLog(pack),
		artifacts.SignoffSLADashboardPath:        RenderUIReviewSignoffSLADashboard(pack),
		artifacts.SignoffReminderQueuePath:       RenderUIReviewSignoffReminderQueue(pack),
		artifacts.ReminderCadenceBoardPath:       RenderUIReviewReminderCadenceBoard(pack),
		artifacts.SignoffBreachBoardPath:         RenderUIReviewSignoffBreachBoard(pack),
		artifacts.EscalationDashboardPath:        RenderUIReviewEscalationDashboard(pack),
		artifacts.EscalationHandoffLedgerPath:    RenderUIReviewEscalationHandoffLedger(pack),
		artifacts.HandoffAckLedgerPath:           RenderUIReviewHandoffAckLedger(pack),
		artifacts.OwnerEscalationDigestPath:      RenderUIReviewOwnerEscalationDigest(pack),
		artifacts.OwnerWorkloadBoardPath:         RenderUIReviewOwnerWorkloadBoard(pack),
		artifacts.BlockerLogPath:                 RenderUIReviewBlockerLog(pack),
		artifacts.BlockerTimelinePath:            RenderUIReviewBlockerTimeline(pack),
		artifacts.FreezeExceptionBoardPath:       RenderUIReviewFreezeExceptionBoard(pack),
		artifacts.FreezeApprovalTrailPath:        RenderUIReviewFreezeApprovalTrail(pack),
		artifacts.FreezeRenewalTrackerPath:       RenderUIReviewFreezeRenewalTracker(pack),
		artifacts.ExceptionLogPath:               RenderUIReviewExceptionLog(pack),
		artifacts.ExceptionMatrixPath:            RenderUIReviewExceptionMatrix(pack),
		artifacts.AuditDensityBoardPath:          RenderUIReviewAuditDensityBoard(pack),
		artifacts.OwnerReviewQueuePath:           RenderUIReviewOwnerReviewQueue(pack),
		artifacts.BlockerTimelineSummaryPath:     RenderUIReviewBlockerTimelineSummary(pack),
	}
	for path, content := range writes {
		if err := writeFile(path, content); err != nil {
			return UIReviewPackArtifacts{}, err
		}
	}
	return artifacts, nil
}
