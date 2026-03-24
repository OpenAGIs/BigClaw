package product

import (
	"strings"
	"testing"
)

func TestBuildDefaultClawHostLifecycleRecoveryScorecard(t *testing.T) {
	scorecard := BuildDefaultClawHostLifecycleRecoveryScorecard("platform", "apollo")
	audit := AuditClawHostLifecycleRecoveryScorecard(scorecard)
	if scorecard.ScorecardID != "BIG-PAR-292" || scorecard.Version != "go-v1" {
		t.Fatalf("unexpected recovery scorecard identity: %+v", scorecard)
	}
	if scorecard.Filters["team"] != "platform" || scorecard.Filters["project"] != "apollo" {
		t.Fatalf("unexpected recovery filters: %+v", scorecard.Filters)
	}
	if scorecard.Summary.BotCount != 2 || scorecard.Summary.RecoverableBots != 2 || scorecard.Summary.IsolatedBots != 2 {
		t.Fatalf("unexpected recovery summary: %+v", scorecard.Summary)
	}
	if !audit.ReleaseReady || audit.ReadinessScore != 100 {
		t.Fatalf("expected default recovery scorecard to be release ready, got %+v", audit)
	}
}

func TestAuditClawHostLifecycleRecoveryScorecardDetectsGaps(t *testing.T) {
	scorecard := BuildDefaultClawHostLifecycleRecoveryScorecard("", "")
	scorecard.Lifecycle[0].Evidence = nil
	scorecard.Bots[0].PodIsolation = false
	scorecard.Bots[0].TakeoverTriggers = nil
	scorecard.Bots[0].RecoveryEvidence = nil
	scorecard.Bots[0].RecoveryReadiness = "degraded"

	audit := AuditClawHostLifecycleRecoveryScorecard(scorecard)
	if audit.ReleaseReady {
		t.Fatalf("expected degraded recovery audit, got %+v", audit)
	}
	if len(audit.MissingLifecycleActions) == 0 || len(audit.BotsMissingIsolation) == 0 || len(audit.BotsMissingTakeover) == 0 || len(audit.BotsMissingEvidence) == 0 || len(audit.DegradedBots) == 0 {
		t.Fatalf("expected audit gaps, got %+v", audit)
	}
	if audit.ReadinessScore >= 100 {
		t.Fatalf("expected readiness score penalty, got %+v", audit)
	}
}

func TestRenderClawHostLifecycleRecoveryReport(t *testing.T) {
	scorecard := BuildDefaultClawHostLifecycleRecoveryScorecard("platform", "apollo")
	audit := AuditClawHostLifecycleRecoveryScorecard(scorecard)
	report := RenderClawHostLifecycleRecoveryReport(scorecard, audit)
	for _, want := range []string{
		"# ClawHost Lifecycle Recovery Scorecard",
		"Recoverable Bots: 2/2",
		"Per-Bot Isolation",
		"platform-release-bot",
		"Missing lifecycle coverage: none",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report, got %s", want, report)
		}
	}
}
