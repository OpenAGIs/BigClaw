package reporting

import (
	"fmt"
	"strings"
	"time"
)

const CrossProcessCoordinationSurfaceGenerator = "bigclaw-go/scripts/e2e/cross_process_coordination_surface/main.go"

type CrossProcessCoordinationSurfaceOptions struct {
	MultiNodeReportPath    string
	TakeoverReportPath     string
	LiveTakeoverReportPath string
	Now                    time.Time
}

func BuildCrossProcessCoordinationSurface(root string, options CrossProcessCoordinationSurfaceOptions) (map[string]any, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("repo root is required")
	}
	if options.MultiNodeReportPath == "" {
		options.MultiNodeReportPath = "bigclaw-go/docs/reports/multi-node-shared-queue-report.json"
	}
	if options.TakeoverReportPath == "" {
		options.TakeoverReportPath = "bigclaw-go/docs/reports/multi-subscriber-takeover-validation-report.json"
	}
	if options.LiveTakeoverReportPath == "" {
		options.LiveTakeoverReportPath = "bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-report.json"
	}
	now := options.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	var multiNode map[string]any
	if err := loadJSON(resolveReportPath(root, options.MultiNodeReportPath), &multiNode); err != nil {
		return nil, err
	}
	var takeover map[string]any
	if err := loadJSON(resolveReportPath(root, options.TakeoverReportPath), &takeover); err != nil {
		return nil, err
	}
	var liveTakeover map[string]any
	if err := loadJSON(resolveReportPath(root, options.LiveTakeoverReportPath), &liveTakeover); err != nil {
		return nil, err
	}

	summary := map[string]any{
		"shared_queue_total_tasks":               asInt(multiNode["count"]),
		"shared_queue_cross_node_completions":    asInt(multiNode["cross_node_completions"]),
		"shared_queue_duplicate_completed_tasks": len(asSlice(multiNode["duplicate_completed_tasks"])),
		"shared_queue_duplicate_started_tasks":   len(asSlice(multiNode["duplicate_started_tasks"])),
		"takeover_scenario_count":                asInt(asMap(takeover["summary"])["scenario_count"]),
		"takeover_passing_scenarios":             asInt(asMap(takeover["summary"])["passing_scenarios"]),
		"takeover_duplicate_delivery_count":      asInt(asMap(takeover["summary"])["duplicate_delivery_count"]),
		"takeover_stale_write_rejections":        asInt(asMap(takeover["summary"])["stale_write_rejections"]),
		"live_takeover_scenario_count":           asInt(asMap(liveTakeover["summary"])["scenario_count"]),
		"live_takeover_passing_scenarios":        asInt(asMap(liveTakeover["summary"])["passing_scenarios"]),
		"live_takeover_stale_write_rejections":   asInt(asMap(liveTakeover["summary"])["stale_write_rejections"]),
	}

	targetContracts := []map[string]any{
		{
			"capability":      "partitioned_topic_routing",
			"contract_anchor": "events.SubscriptionRequest.PartitionRoute",
			"runtime_status":  "contract_only",
			"partitioning": map[string]any{
				"topic":                    "provider-defined shared event stream",
				"supported_partition_keys": []string{"trace_id", "task_id", "event_type"},
				"ordering_scope":           "sequence remains portable within the selected partition route",
				"filter_alignment":         "ReplayRequest task_id/trace_id filters must remain valid when a backend introduces partition routing.",
			},
			"ownership":  map[string]any{},
			"guarantees": []string{"Partition keys are provider-neutral and map to existing trace/task/event_type selectors.", "Partition metadata may vary by backend, but portable replay ordering still uses Position.Sequence.", "No runtime implementation is shipped yet; this row defines the future adapter contract only."},
		},
		{
			"capability":      "broker_backed_subscriber_ownership",
			"contract_anchor": "events.SubscriptionRequest.OwnershipContract",
			"runtime_status":  "contract_only",
			"partitioning":    map[string]any{},
			"ownership": map[string]any{
				"subscriber_group": "shared durable consumer identity",
				"mode":             "exclusive",
				"lease_fields":     []string{"epoch", "lease_token"},
				"partition_hints":  "optional partition affinity for future broker-backed consumers",
			},
			"guarantees": []string{"Checkpoint commits remain fenced by epoch plus lease token after ownership transfer.", "Ownership metadata travels through the neutral subscription contract instead of provider-specific APIs.", "No broker-backed runtime implementation is shipped yet; this row defines the future ownership contract only."},
		},
	}

	capabilities := []map[string]any{
		coordinationCapabilityRow("shared_queue_task_coordination", "implemented", "live_proven", true, false, true, []string{"Two independent bigclawd processes share one SQLite-backed queue without duplicate terminal execution.", "Current proof is local and SQLite-backed rather than broker-backed or replicated."}),
		coordinationCapabilityRow("subscriber_takeover_semantics", "implemented_with_shared_durable_scaffold", "live_proven", true, true, true, []string{"Lease handoff, stale-writer fencing, and duplicate replay accounting are covered by both the deterministic harness and the live two-node companion proof.", "The live proof now drives both nodes against one shared SQLite lease backend, but the provider-neutral broker-backed ownership contract is still not runtime-proven."}),
		coordinationCapabilityRow("cross_process_replay_coordination", "contract_defined", "harness_proven", false, true, true, []string{"Replay cursor and checkpoint expectations are codified across the local takeover harness and durability rollout contract.", "No broker-backed or partitioned live replay proof exists yet."}),
		coordinationCapabilityRow("stale_writer_fencing", "implemented_with_shared_durable_scaffold", "live_proven", true, true, true, []string{"The local takeover harness and the live two-node companion proof both show stale checkpoint writers being fenced after ownership transfer.", "The shared durable scaffold is SQLite-backed today, so the broker-backed ownership contract remains future work beyond the current local proof."}),
		coordinationCapabilityRow("partitioned_topic_routing", "not_available", "contract_only", false, false, true, []string{"No partitioned topic model exists yet in the runtime.", "Broker-backed target docs reserve partition or quorum log ordering as the future coordination scope."}),
		coordinationCapabilityRow("broker_backed_subscriber_ownership", "not_available", "contract_only", false, false, true, []string{"No broker-backed cross-process subscriber ownership model exists yet.", "The durability rollout contract defines the expected checkpoint fencing and failover guarantees before rollout-safe claims."}),
		coordinationCapabilityRow("operator_capability_surface", "implemented", "supporting_surface", false, false, true, []string{"The repo exposes provider-neutral durability and rollout metadata through docs and runtime-facing event_durability surfaces.", "This report adds a coordination-specific surface tying together live local proof, deterministic local harnesses, and future targets."}),
	}

	return map[string]any{
		"generated_at":                    utcISOTime(now),
		"ticket":                          "BIG-PAR-085-local-prework",
		"title":                           "Cross-process coordination capability surface",
		"status":                          "local-capability-surface",
		"target_contract_surface_version": "2026-03-17",
		"runtime_readiness_levels": map[string]any{
			"live_proven":        "Shipped runtime behavior with checked-in live cross-process proof.",
			"harness_proven":     "Deterministic executable harness coverage exists, but no live multi-node proof is checked in.",
			"contract_only":      "Only target contracts or rollout docs define the expected semantics today.",
			"supporting_surface": "The repo exposes reporting or metadata surfaces that describe runtime readiness without proving the coordination behavior itself.",
		},
		"evidence_inputs": map[string]any{
			"shared_queue_report":     options.MultiNodeReportPath,
			"takeover_harness_report": options.TakeoverReportPath,
			"live_takeover_report":    options.LiveTakeoverReportPath,
			"supporting_docs": []string{
				"bigclaw-go/docs/reports/event-bus-reliability-report.md",
				"bigclaw-go/docs/reports/replicated-event-log-durability-rollout-contract.md",
				"bigclaw-go/docs/reports/broker-event-log-adapter-contract.md",
			},
		},
		"target_contracts": targetContracts,
		"summary":          summary,
		"capabilities":     capabilities,
		"current_ceiling": []string{
			"no partitioned topic model",
			"no broker-backed cross-process subscriber coordination",
			"no broker-backed or replicated subscriber ownership backend",
		},
		"next_runtime_hooks": []string{
			"emit native takeover transition audit events from the runtime instead of harness-authored artifacts",
			"validate broker-backed replay and ownership semantics against the same report schema",
			"replace the SQLite shared durable scaffold with a broker-backed or replicated ownership backend",
		},
	}, nil
}

func coordinationCapabilityRow(capability string, currentState string, runtimeReadiness string, liveLocalProof bool, deterministicLocalHarness bool, contractDefinedTarget bool, notes []string) map[string]any {
	return map[string]any{
		"capability":                  capability,
		"current_state":               currentState,
		"runtime_readiness":           runtimeReadiness,
		"live_local_proof":            liveLocalProof,
		"deterministic_local_harness": deterministicLocalHarness,
		"contract_defined_target":     contractDefinedTarget,
		"notes":                       notes,
	}
}

func utcISOTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}
