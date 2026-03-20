package risk

import (
	"encoding/json"
	"reflect"
	"testing"

	"bigclaw-go/internal/domain"
)

func TestAssessmentRoundTripPreservesSignalsAndMitigations(t *testing.T) {
	assessment := Assessment{
		AssessmentID:     "risk-1",
		TaskID:           "OPE-130",
		Level:            domain.RiskHigh,
		TotalScore:       75,
		RequiresApproval: true,
		Signals: []Signal{
			{
				Name:     "prod-deploy",
				Score:    20,
				Reason:   "production deployment surface",
				Source:   "scheduler",
				Metadata: map[string]any{"tool": "deploy"},
			},
		},
		Mitigations: []string{"security review", "rollback plan"},
		Reviewer:    "ops-review",
		Notes:       "Requires explicit sign-off.",
	}

	payload, err := json.Marshal(assessment)
	if err != nil {
		t.Fatalf("marshal assessment: %v", err)
	}

	var restored Assessment
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal assessment: %v", err)
	}

	if !reflect.DeepEqual(restored, assessment) {
		t.Fatalf("assessment mismatch: restored=%+v want=%+v", restored, assessment)
	}
	if restored.Signals[0].Metadata["tool"] != "deploy" {
		t.Fatalf("expected signal metadata to survive round trip, got %+v", restored)
	}
}
