package main

import (
	"errors"
	"flag"
	"os"
	"time"
)

type automationCrossProcessCoordinationSurfaceOptions struct {
	GoRoot                 string
	MultiNodeReportPath    string
	TakeoverReportPath     string
	LiveTakeoverReportPath string
	OutputPath             string
}

func runAutomationCrossProcessCoordinationSurfaceCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e cross-process-coordination-surface", flag.ContinueOnError)
	goRoot := flags.String("go-root", ".", "bigclaw-go repo root")
	multiNodeReport := flags.String("multi-node-report", "docs/reports/multi-node-shared-queue-report.json", "multi-node report path")
	takeoverReport := flags.String("takeover-report", "docs/reports/multi-subscriber-takeover-validation-report.json", "takeover report path")
	liveTakeoverReport := flags.String("live-takeover-report", "docs/reports/live-multi-node-subscriber-takeover-report.json", "live takeover report path")
	output := flags.String("output", "docs/reports/cross-process-coordination-capability-surface.json", "output path")
	asJSON := flags.Bool("json", true, "json")
	pretty := flags.Bool("pretty", false, "pretty-print report to stdout")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e cross-process-coordination-surface [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, err := automationCrossProcessCoordinationSurface(automationCrossProcessCoordinationSurfaceOptions{
		GoRoot:                 absPath(*goRoot),
		MultiNodeReportPath:    *multiNodeReport,
		TakeoverReportPath:     *takeoverReport,
		LiveTakeoverReportPath: *liveTakeoverReport,
		OutputPath:             *output,
	})
	if err != nil {
		return err
	}
	if *pretty {
		return emit(report, true, 0)
	}
	return emit(report, *asJSON, 0)
}

func automationCrossProcessCoordinationSurface(opts automationCrossProcessCoordinationSurfaceOptions) (map[string]any, error) {
	root := absPath(opts.GoRoot)
	multiNode, err := e2eReadJSONMap(e2eResolvePath(root, opts.MultiNodeReportPath))
	if err != nil {
		return nil, err
	}
	takeover, err := e2eReadJSONMap(e2eResolvePath(root, opts.TakeoverReportPath))
	if err != nil {
		return nil, err
	}
	liveTakeover, err := e2eReadJSONMap(e2eResolvePath(root, opts.LiveTakeoverReportPath))
	if err != nil {
		return nil, err
	}
	report := map[string]any{
		"generated_at":                    e2eUTCISO(timeNowUTC()),
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
			"shared_queue_report":     opts.MultiNodeReportPath,
			"takeover_harness_report": opts.TakeoverReportPath,
			"live_takeover_report":    opts.LiveTakeoverReportPath,
			"supporting_docs": []any{
				"bigclaw-go/docs/reports/event-bus-reliability-report.md",
				"bigclaw-go/docs/reports/replicated-event-log-durability-rollout-contract.md",
				"bigclaw-go/docs/reports/broker-event-log-adapter-contract.md",
			},
		},
		"target_contracts": []any{
			map[string]any{
				"capability":      "partitioned_topic_routing",
				"contract_anchor": "events.SubscriptionRequest.PartitionRoute",
				"runtime_status":  "contract_only",
				"partitioning": map[string]any{
					"topic":                    "provider-defined shared event stream",
					"supported_partition_keys": []any{"trace_id", "task_id", "event_type"},
					"ordering_scope":           "sequence remains portable within the selected partition route",
					"filter_alignment":         "ReplayRequest task_id/trace_id filters must remain valid when a backend introduces partition routing.",
				},
				"ownership":  map[string]any{},
				"guarantees": []any{"Partition keys are provider-neutral and map to existing trace/task/event_type selectors.", "Partition metadata may vary by backend, but portable replay ordering still uses Position.Sequence.", "No runtime implementation is shipped yet; this row defines the future adapter contract only."},
			},
			map[string]any{
				"capability":      "broker_backed_subscriber_ownership",
				"contract_anchor": "events.SubscriptionRequest.OwnershipContract",
				"runtime_status":  "contract_only",
				"partitioning":    map[string]any{},
				"ownership": map[string]any{
					"subscriber_group": "shared durable consumer identity",
					"mode":             "exclusive",
					"lease_fields":     []any{"epoch", "lease_token"},
					"partition_hints":  "optional partition affinity for future broker-backed consumers",
				},
				"guarantees": []any{"Checkpoint commits remain fenced by epoch plus lease token after ownership transfer.", "Ownership metadata travels through the neutral subscription contract instead of provider-specific APIs.", "No broker-backed runtime implementation is shipped yet; this row defines the future ownership contract only."},
			},
		},
		"summary": map[string]any{
			"shared_queue_total_tasks":               multiNode["count"],
			"shared_queue_cross_node_completions":    multiNode["cross_node_completions"],
			"shared_queue_duplicate_completed_tasks": lenMapSlice(multiNode["duplicate_completed_tasks"]),
			"shared_queue_duplicate_started_tasks":   lenMapSlice(multiNode["duplicate_started_tasks"]),
			"takeover_scenario_count":                lookupMap(takeover, "summary", "scenario_count"),
			"takeover_passing_scenarios":             lookupMap(takeover, "summary", "passing_scenarios"),
			"takeover_duplicate_delivery_count":      lookupMap(takeover, "summary", "duplicate_delivery_count"),
			"takeover_stale_write_rejections":        lookupMap(takeover, "summary", "stale_write_rejections"),
			"live_takeover_scenario_count":           lookupMap(liveTakeover, "summary", "scenario_count"),
			"live_takeover_passing_scenarios":        lookupMap(liveTakeover, "summary", "passing_scenarios"),
			"live_takeover_stale_write_rejections":   lookupMap(liveTakeover, "summary", "stale_write_rejections"),
		},
		"capabilities": []any{
			e2eCapabilityRow("shared_queue_task_coordination", "implemented", "live_proven", true, false, true, []any{"Two independent bigclawd processes share one SQLite-backed queue without duplicate terminal execution.", "Current proof is local and SQLite-backed rather than broker-backed or replicated."}),
			e2eCapabilityRow("subscriber_takeover_semantics", "implemented_with_shared_durable_scaffold", "live_proven", true, true, true, []any{"Lease handoff, stale-writer fencing, and duplicate replay accounting are covered by both the deterministic harness and the live two-node companion proof.", "The live proof now drives both nodes against one shared SQLite lease backend, but the provider-neutral broker-backed ownership contract is still not runtime-proven."}),
			e2eCapabilityRow("cross_process_replay_coordination", "contract_defined", "harness_proven", false, true, true, []any{"Replay cursor and checkpoint expectations are codified across the local takeover harness and durability rollout contract.", "No broker-backed or partitioned live replay proof exists yet."}),
			e2eCapabilityRow("stale_writer_fencing", "implemented_with_shared_durable_scaffold", "live_proven", true, true, true, []any{"The local takeover harness and the live two-node companion proof both show stale checkpoint writers being fenced after ownership transfer.", "The shared durable scaffold is SQLite-backed today, so the broker-backed ownership contract remains future work beyond the current local proof."}),
			e2eCapabilityRow("partitioned_topic_routing", "not_available", "contract_only", false, false, true, []any{"No partitioned topic model exists yet in the runtime.", "Broker-backed target docs reserve partition or quorum log ordering as the future coordination scope."}),
			e2eCapabilityRow("broker_backed_subscriber_ownership", "not_available", "contract_only", false, false, true, []any{"No broker-backed cross-process subscriber ownership model exists yet.", "The durability rollout contract defines the expected checkpoint fencing and failover guarantees before rollout-safe claims."}),
			e2eCapabilityRow("operator_capability_surface", "implemented", "supporting_surface", false, false, true, []any{"The repo exposes provider-neutral durability and rollout metadata through docs and runtime-facing event_durability surfaces.", "This report adds a coordination-specific surface tying together live local proof, deterministic local harnesses, and future targets."}),
		},
		"current_ceiling": []any{
			"no partitioned topic model",
			"no broker-backed cross-process subscriber coordination",
			"no broker-backed or replicated subscriber ownership backend",
		},
		"next_runtime_hooks": []any{
			"emit native takeover transition audit events from the runtime instead of harness-authored artifacts",
			"validate broker-backed replay and ownership semantics against the same report schema",
			"replace the SQLite shared durable scaffold with a broker-backed or replicated ownership backend",
		},
	}
	if err := e2eWriteJSON(e2eResolvePath(root, opts.OutputPath), report); err != nil {
		return nil, err
	}
	return report, nil
}

func e2eCapabilityRow(capability, currentState, runtimeReadiness string, liveLocalProof, deterministicLocalHarness, contractDefinedTarget bool, notes []any) map[string]any {
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

func timeNowUTC() time.Time {
	return time.Now().UTC()
}
