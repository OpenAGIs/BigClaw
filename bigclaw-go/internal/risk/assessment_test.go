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

func TestAssessmentJSONEmitsPythonContractDefaults(t *testing.T) {
	assessment := Assessment{AssessmentID: "risk-2", TaskID: "OPE-131"}

	payload, err := json.Marshal(assessment)
	if err != nil {
		t.Fatalf("marshal assessment: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode assessment: %v", err)
	}

	for _, key := range []string{"level", "requires_approval", "signals", "mitigations", "reviewer", "notes"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("expected key %q in assessment JSON, got %+v", key, decoded)
		}
	}
	if decoded["level"] != string(domain.RiskLow) {
		t.Fatalf("expected default low level in JSON, got %+v", decoded)
	}

	signalPayload, err := json.Marshal(Signal{Name: "prod", Reason: "check"})
	if err != nil {
		t.Fatalf("marshal signal: %v", err)
	}
	var decodedSignal map[string]any
	if err := json.Unmarshal(signalPayload, &decodedSignal); err != nil {
		t.Fatalf("decode signal: %v", err)
	}
	for _, key := range []string{"source", "metadata"} {
		if _, ok := decodedSignal[key]; !ok {
			t.Fatalf("expected key %q in signal JSON, got %+v", key, decodedSignal)
		}
	}
}
