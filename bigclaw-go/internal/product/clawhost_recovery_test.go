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
	if scorecard.Summary.BotCount != 1 || scorecard.Summary.RecoverableBots != 1 || scorecard.Summary.IsolatedBots != 1 {
		t.Fatalf("unexpected recovery summary: %+v", scorecard.Summary)
	}
	if !audit.ReleaseReady || audit.ReadinessScore != 100 {
		t.Fatalf("expected default recovery scorecard to be release ready, got %+v", audit)
	}
}

func TestBuildDefaultClawHostLifecycleRecoveryScorecardScopesInventory(t *testing.T) {
	t.Run("project only", func(t *testing.T) {
		scorecard := BuildDefaultClawHostLifecycleRecoveryScorecard("", "campaigns")
		if scorecard.Filters["team"] != "" || scorecard.Filters["project"] != "campaigns" {
			t.Fatalf("unexpected recovery filters: %+v", scorecard.Filters)
		}
		if scorecard.Summary.BotCount != 1 || scorecard.Summary.RecoverableBots != 1 || scorecard.Summary.IsolatedBots != 1 || scorecard.Summary.DegradedBots != 0 {
			t.Fatalf("unexpected project-scoped recovery summary: %+v", scorecard.Summary)
		}
		if len(scorecard.Bots) != 1 || scorecard.Bots[0].BotID != "bot-growth-1" || scorecard.Bots[0].RecoveryReadiness != "ready" {
			t.Fatalf("expected only ready growth bot in project scope, got %+v", scorecard.Bots)
		}
	})

	t.Run("no matches", func(t *testing.T) {
		scorecard := BuildDefaultClawHostLifecycleRecoveryScorecard("support", "phoenix")
		audit := AuditClawHostLifecycleRecoveryScorecard(scorecard)
		if scorecard.Filters["team"] != "support" || scorecard.Filters["project"] != "phoenix" {
			t.Fatalf("expected no-match recovery scope to persist filters, got %+v", scorecard.Filters)
		}
		if scorecard.Summary.BotCount != 0 || scorecard.Summary.RecoverableBots != 0 || scorecard.Summary.IsolatedBots != 0 || scorecard.Summary.DegradedBots != 0 {
			t.Fatalf("expected empty recovery summary, got %+v", scorecard.Summary)
		}
		if len(scorecard.Bots) != 0 {
			t.Fatalf("expected no scoped recovery bots, got %+v", scorecard.Bots)
		}
		if !audit.ReleaseReady || audit.ReadinessScore != 100 {
			t.Fatalf("expected empty recovery audit to stay lifecycle-ready, got %+v", audit)
		}
	})
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

func TestAuditClawHostLifecycleRecoveryScorecardHandlesEmptyScorecard(t *testing.T) {
	scorecard := BuildDefaultClawHostLifecycleRecoveryScorecard("", "")
	scorecard.Lifecycle = nil
	scorecard.Bots = nil
	scorecard.Summary = ClawHostLifecycleRecoverySummary{}

	audit := AuditClawHostLifecycleRecoveryScorecard(scorecard)
	if audit.ScorecardID != scorecard.ScorecardID || audit.Version != scorecard.Version {
		t.Fatalf("expected audit metadata to mirror empty scorecard, got %+v", audit)
	}
	if audit.ReadinessScore != 0 {
		t.Fatalf("expected empty recovery scorecard readiness score to stay zero, got %+v", audit)
	}
	if audit.ReleaseReady {
		t.Fatalf("expected empty recovery scorecard to stay not release ready, got %+v", audit)
	}
	if got := strings.Join(audit.MissingLifecycleActions, ","); got != "create,delete,restart,start,stop,upgrade" {
		t.Fatalf("expected empty recovery scorecard to report all missing lifecycle actions, got %+v", audit)
	}
	if len(audit.BotsMissingIsolation) != 0 || len(audit.BotsMissingTakeover) != 0 || len(audit.BotsMissingEvidence) != 0 || len(audit.DegradedBots) != 0 {
		t.Fatalf("expected empty recovery scorecard to have no bot-specific gaps, got %+v", audit)
	}
}

func TestRenderClawHostLifecycleRecoveryReport(t *testing.T) {
	scorecard := BuildDefaultClawHostLifecycleRecoveryScorecard("platform", "apollo")
	audit := AuditClawHostLifecycleRecoveryScorecard(scorecard)
	report := RenderClawHostLifecycleRecoveryReport(scorecard, audit)
	for _, want := range []string{
		"# ClawHost Lifecycle Recovery Scorecard",
		"## Filters",
		"- project: apollo",
		"- team: platform",
		"Recoverable Bots: 1/1",
		"Per-Bot Isolation",
		"platform-release-bot",
		"Missing lifecycle coverage: none",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report, got %s", want, report)
		}
	}
}

func TestRenderClawHostLifecycleRecoveryReportHandlesEmptyBots(t *testing.T) {
	scorecard := BuildDefaultClawHostLifecycleRecoveryScorecard("support", "phoenix")
	audit := AuditClawHostLifecycleRecoveryScorecard(scorecard)
	report := RenderClawHostLifecycleRecoveryReport(scorecard, audit)

	for _, want := range []string{
		"# ClawHost Lifecycle Recovery Scorecard",
		"## Filters",
		"- project: phoenix",
		"- team: support",
		"Recoverable Bots: 0/0",
		"## Per-Bot Isolation",
		"Missing lifecycle coverage: none",
		"Bots missing isolation: none",
		"Degraded bots: none",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in empty recovery report, got %s", want, report)
		}
	}
}

func TestRenderClawHostLifecycleRecoveryReportHandlesEmptyFiltersLifecycleAndWarnings(t *testing.T) {
	scorecard := ClawHostLifecycleRecoveryScorecard{
		ScorecardID:      "BIG-PAR-359",
		Version:          "go-v1",
		SourceRepository: "https://github.com/fastclaw-ai/clawhost",
		ControlPlane:     ClawHostControlPlane{Name: "ClawHost"},
		Filters:          nil,
		Lifecycle:        nil,
		Bots: []ClawHostBotRecoveryScore{
			{
				BotID:             "bot-warning-1",
				Name:              "warning-bot",
				Status:            "error",
				RecoveryReadiness: "degraded",
				Warnings:          []string{"bot is missing dedicated pod or service isolation"},
			},
		},
		Summary: ClawHostLifecycleRecoverySummary{
			BotCount:     1,
			DegradedBots: 1,
		},
	}
	audit := ClawHostLifecycleRecoveryAudit{
		ScorecardID:    scorecard.ScorecardID,
		Version:        scorecard.Version,
		DegradedBots:   []string{"bot-warning-1"},
		ReadinessScore: 50,
		ReleaseReady:   false,
	}

	report := RenderClawHostLifecycleRecoveryReport(scorecard, audit)
	for _, want := range []string{
		"## Filters",
		"- none",
		"## Lifecycle Coverage",
		"## Per-Bot Isolation",
		"warning-bot (bot-warning-1)",
		"warnings=bot is missing dedicated pod or service isolation",
		"Degraded bots: bot-warning-1",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in edge-case recovery report, got %s", want, report)
		}
	}
}
