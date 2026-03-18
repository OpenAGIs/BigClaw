package risk

import (
	"testing"

	"bigclaw-go/internal/domain"
)

func TestScoreTaskKeepsSimpleLowRiskWorkLow(t *testing.T) {
	score := ScoreTask(domain.Task{ID: "BIG-902-low", Title: "doc cleanup"}, nil)
	if score.Total != 0 || score.Level != domain.RiskLow || score.RequiresApproval {
		t.Fatalf("expected low baseline risk score, got %+v", score)
	}
}

func TestScoreTaskElevatesProdBrowserWork(t *testing.T) {
	score := ScoreTask(domain.Task{
		ID:            "BIG-902-medium",
		Title:         "release verification",
		Labels:        []string{"prod"},
		Priority:      1,
		RequiredTools: []string{"browser"},
	}, nil)
	if score.Total != 40 || score.Level != domain.RiskMedium || score.RequiresApproval {
		t.Fatalf("expected medium risk score, got %+v", score)
	}
}

func TestScoreTaskUsesFailuresRetriesAndRegressions(t *testing.T) {
	score := ScoreTask(domain.Task{
		ID:            "BIG-902-high",
		Title:         "security deploy",
		Labels:        []string{"security", "prod"},
		Priority:      1,
		RequiredTools: []string{"deploy"},
		Metadata: map[string]string{
			"regression_count": "1",
		},
	}, []domain.Event{{Type: domain.EventTaskRetried}, {Type: domain.EventTaskDeadLetter}})
	if score.Total != 90 || score.Level != domain.RiskHigh || !score.RequiresApproval {
		t.Fatalf("expected high explainable risk score, got %+v", score)
	}
	if score.Summary == "baseline=0" || len(score.Factors) == 0 {
		t.Fatalf("expected populated risk factor summary, got %+v", score)
	}
}

func TestScoreTaskFlagsNegativeBudgetForManualReview(t *testing.T) {
	score := ScoreTask(domain.Task{
		ID:          "BIG-902-budget",
		Title:       "backfill billing envelope",
		BudgetCents: -1,
	}, nil)
	if score.Total != 20 || score.Level != domain.RiskLow || score.RequiresApproval {
		t.Fatalf("expected negative budget to add review-only risk without high-risk approval, got %+v", score)
	}
	if len(score.Factors) != 1 || score.Factors[0].Name != "budget" {
		t.Fatalf("expected budget factor, got %+v", score.Factors)
	}
}
