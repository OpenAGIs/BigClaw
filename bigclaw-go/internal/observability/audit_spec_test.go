package observability

import (
	"reflect"
	"testing"
)

func TestP0AuditEventSpecsDefineRequiredOperationalEvents(t *testing.T) {
	eventTypes := make([]string, 0, len(P0AuditEventSpecs))
	for _, spec := range P0AuditEventSpecs {
		eventTypes = append(eventTypes, spec.EventType)
	}
	wantTypes := []string{
		SchedulerDecisionEvent,
		ManualTakeoverEvent,
		ApprovalRecordedEvent,
		BudgetOverrideEvent,
		FlowHandoffEvent,
	}
	if !reflect.DeepEqual(eventTypes, wantTypes) {
		t.Fatalf("unexpected event types: got %+v want %+v", eventTypes, wantTypes)
	}
	missing := MissingRequiredFields(SchedulerDecisionEvent, map[string]any{
		"task_id": "OPE-134",
		"run_id":  "run-ope-134",
		"medium":  "docker",
	})
	if !reflect.DeepEqual(missing, []string{"approved", "reason", "risk_level", "risk_score"}) {
		t.Fatalf("unexpected missing required fields: %+v", missing)
	}
}
