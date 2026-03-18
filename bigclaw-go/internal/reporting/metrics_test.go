package reporting

import (
	"strings"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestBuildMetricSpecComputesCoreOperationsMetrics(t *testing.T) {
	spendPreferred := 12.5
	costFallback := 7.25
	spendFallback := 3.75
	spec, err := BuildMetricSpec([]MetricRun{
		{
			RunID:     "run-1",
			StartedAt: "2026-03-18T00:15:00Z",
			EndedAt:   "2026-03-18T00:45:00Z",
			Status:    "failed",
			RiskLevel: domain.RiskLow,
			RiskScore: &MetricRiskScore{Total: 82},
			SpendUSD:  &spendPreferred,
			CostUSD:   floatPtr(99),
		},
		{
			RunID:     "run-2",
			StartedAt: "2026-03-18T02:00:00Z",
			EndedAt:   "2026-03-18T03:30:00Z",
			Status:    "succeeded",
			RiskLevel: domain.RiskHigh,
			CostUSD:   &costFallback,
		},
		{
			RunID:     "run-3",
			StartedAt: "2026-03-17T23:30:00Z",
			EndedAt:   "2026-03-18T00:00:00Z",
			Status:    "needs-approval",
			RiskLevel: domain.RiskMedium,
			Spend:     &spendFallback,
		},
		{
			RunID:     "run-4",
			StartedAt: "invalid",
			EndedAt:   "2026-03-18T01:00:00Z",
			Status:    "rejected",
		},
	}, MetricSpecOptions{
		PeriodStart:      "2026-03-18T00:00:00Z",
		PeriodEnd:        "2026-03-18T23:59:59Z",
		SLATargetMinutes: 60,
		TimezoneName:     "UTC",
	})
	if err != nil {
		t.Fatalf("build metric spec: %v", err)
	}
	if spec.Name != "Operations Metric Spec" || len(spec.Definitions) != 7 || len(spec.Values) != 7 {
		t.Fatalf("unexpected spec shape: %+v", spec)
	}

	values := metricValuesByID(spec.Values)
	if got := values["runs-today"]; got.Value != 2 || got.Denominator != 4 {
		t.Fatalf("unexpected runs-today: %+v", got)
	}
	if got := values["avg-lead-time"]; got.Value != 50.0 || got.Numerator != 150.0 || got.Denominator != 3 {
		t.Fatalf("unexpected avg-lead-time: %+v", got)
	}
	if got := values["intervention-rate"]; got.Value != 75.0 || got.Numerator != 3 || got.Denominator != 4 {
		t.Fatalf("unexpected intervention-rate: %+v", got)
	}
	if got := values["sla"]; got.Value != 66.7 || got.Numerator != 2 || got.Denominator != 3 {
		t.Fatalf("unexpected sla: %+v", got)
	}
	if got := values["risk"]; got.Value != 77.3 || got.Numerator != 232.0 || got.Denominator != 3 {
		t.Fatalf("unexpected risk: %+v", got)
	}
	if got := values["spend"]; got.Value != 23.5 || got.Numerator != 23.5 || got.Denominator != 4 {
		t.Fatalf("unexpected spend: %+v", got)
	}
}

func TestBuildMetricSpecRejectsInvalidWindow(t *testing.T) {
	_, err := BuildMetricSpec(nil, MetricSpecOptions{
		PeriodStart: "2026-03-18T12:00:00Z",
		PeriodEnd:   "2026-03-18T11:59:59Z",
	})
	if err == nil {
		t.Fatal("expected invalid window error")
	}
}

func TestBuildMetricSpecHandlesEmptyRunsAndRegressionEvidence(t *testing.T) {
	spec, err := BuildMetricSpec(nil, MetricSpecOptions{
		PeriodStart: "2026-03-18T00:00:00Z",
		PeriodEnd:   "2026-03-18T23:59:59Z",
		CurrentBenchmark: &BenchmarkSuite{
			Version: "current",
			Results: []BenchmarkCaseResult{{CaseID: "a", Score: 95, Passed: true}},
		},
	})
	if err != nil {
		t.Fatalf("build empty metric spec: %v", err)
	}
	values := metricValuesByID(spec.Values)
	if values["intervention-rate"].Value != 0 || values["sla"].Value != 0 || values["risk"].Value != 0 {
		t.Fatalf("expected zeroed empty metrics, got %+v", values)
	}
	if values["regression"].Denominator != 1 || values["regression"].Value != 0 {
		t.Fatalf("unexpected regression metric: %+v", values["regression"])
	}
}

func TestAnalyzeRegressionsDetectsScoreDropsAndPassToFail(t *testing.T) {
	findings := AnalyzeRegressions(
		&BenchmarkSuite{
			Version: "current",
			Results: []BenchmarkCaseResult{
				{CaseID: "critical-delta", Score: 60, Passed: true},
				{CaseID: "pass-to-fail", Score: 88, Passed: false},
				{CaseID: "steady", Score: 90, Passed: true},
			},
		},
		&BenchmarkSuite{
			Version: "baseline",
			Results: []BenchmarkCaseResult{
				{CaseID: "critical-delta", Score: 85, Passed: true},
				{CaseID: "pass-to-fail", Score: 88, Passed: true},
				{CaseID: "steady", Score: 90, Passed: true},
			},
		},
	)
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %+v", findings)
	}
	if findings[0].CaseID != "critical-delta" || findings[0].Severity != "high" || findings[0].Delta != -25 {
		t.Fatalf("unexpected first finding: %+v", findings[0])
	}
	if findings[1].CaseID != "pass-to-fail" || findings[1].Severity != "high" || !strings.Contains(findings[1].Summary, "passing to failing") {
		t.Fatalf("unexpected second finding: %+v", findings[1])
	}
}

func TestRenderMetricSpecMarkdownIncludesDefinitionsAndValues(t *testing.T) {
	spec := OperationsMetricSpec{
		Name:         "Operations Metric Spec",
		GeneratedAt:  "2026-03-18T12:00:00Z",
		PeriodStart:  "2026-03-18T00:00:00Z",
		PeriodEnd:    "2026-03-18T23:59:59Z",
		TimezoneName: "UTC",
		Definitions: []OperationsMetricDefinition{
			{MetricID: "runs-today", Label: "Runs Today", Unit: "runs", Direction: "up", Formula: "count(...)", Description: "Count runs", SourceFields: []string{"started_at"}},
		},
		Values: []OperationsMetricValue{
			{MetricID: "runs-today", Label: "Runs Today", DisplayValue: "2", Numerator: 2, Denominator: 4, Unit: "runs", Evidence: []string{"2 of 4 runs started inside the reporting window."}},
		},
	}
	rendered := RenderMetricSpecMarkdown(spec)
	if !strings.Contains(rendered, "# Operations Metric Spec") || !strings.Contains(rendered, "### Runs Today") || !strings.Contains(rendered, "value=2") {
		t.Fatalf("unexpected rendered markdown: %s", rendered)
	}
}

func metricValuesByID(values []OperationsMetricValue) map[string]OperationsMetricValue {
	out := make(map[string]OperationsMetricValue, len(values))
	for _, value := range values {
		out[value.MetricID] = value
	}
	return out
}

func floatPtr(value float64) *float64 {
	return &value
}
