package api

import (
	"strings"
	"testing"

	"bigclaw-go/internal/events"
)

func TestBuildBrokerBootstrapRuntimeGate(t *testing.T) {
	tests := []struct {
		name            string
		surface         brokerBootstrapSurface
		wantFailClosed  bool
		wantContract    bool
		wantStubOnly    bool
		messageContains string
	}{
		{
			name: "contract only target",
			surface: brokerBootstrapSurface{
				BootstrapSummary: events.BrokerBootstrapReviewSummary{
					EventLogBackend: events.DurabilityBackendMemory,
					TargetBackend:   events.DurabilityBackendBrokerReplicated,
				},
				RuntimePosture:         "contract_only",
				ProofBoundary:          "contract-only boundary",
				LiveAdapterImplemented: false,
				BootstrapReady:         false,
			},
			wantContract:    true,
			messageContains: "contract-only",
		},
		{
			name: "stub driver only",
			surface: brokerBootstrapSurface{
				BootstrapSummary: events.BrokerBootstrapReviewSummary{
					EventLogBackend: events.DurabilityBackendBrokerReplicated,
					TargetBackend:   events.DurabilityBackendBrokerReplicated,
				},
				RuntimePosture:         "stub_driver_only",
				ProofBoundary:          "stub boundary",
				LiveAdapterImplemented: false,
				BootstrapReady:         true,
			},
			wantStubOnly:    true,
			messageContains: "stub driver only",
		},
		{
			name: "fail closed until adapter exists",
			surface: brokerBootstrapSurface{
				BootstrapSummary: events.BrokerBootstrapReviewSummary{
					EventLogBackend: events.DurabilityBackendBrokerReplicated,
					TargetBackend:   events.DurabilityBackendBrokerReplicated,
				},
				RuntimePosture:         "fail_closed_until_adapter_exists",
				ProofBoundary:          "fail-closed boundary",
				LiveAdapterImplemented: false,
				BootstrapReady:         true,
			},
			wantFailClosed:  true,
			messageContains: "fails closed",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gate := buildBrokerBootstrapRuntimeGate(test.surface)
			if !gate.Requested {
				t.Fatalf("expected requested gate, got %+v", gate)
			}
			if gate.FailClosed != test.wantFailClosed || gate.ContractOnly != test.wantContract || gate.StubDriverOnly != test.wantStubOnly {
				t.Fatalf("unexpected gate flags: %+v", gate)
			}
			if gate.SafeForLiveTraffic {
				t.Fatalf("expected posture to stay unsafe for live traffic, got %+v", gate)
			}
			if gate.ProofBoundary != test.surface.ProofBoundary {
				t.Fatalf("expected proof boundary %q, got %+v", test.surface.ProofBoundary, gate)
			}
			if !strings.Contains(gate.OperatorMessage, test.messageContains) {
				t.Fatalf("expected operator message to contain %q, got %q", test.messageContains, gate.OperatorMessage)
			}
			if len(gate.TransitionGuide) < 3 {
				t.Fatalf("expected posture transition guide, got %+v", gate)
			}
		})
	}
}
