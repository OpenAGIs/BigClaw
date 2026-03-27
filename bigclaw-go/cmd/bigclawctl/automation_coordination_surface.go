package main

import (
	"errors"
	"flag"
	"os"
	"time"
)

type automationCrossProcessCoordinationSurfaceOptions struct {
	RepoRoot               string
	MultiNodeReportPath    string
	TakeoverReportPath     string
	LiveTakeoverReportPath string
	OutputPath             string
	Now                    func() time.Time
}

func runAutomationCrossProcessCoordinationSurfaceCommand(args []string) error {
	flags := flag.NewFlagSet("automation e2e cross-process-coordination-surface", flag.ContinueOnError)
	repoRoot := flags.String("repo-root", "..", "repository root")
	multiNodeReport := flags.String("multi-node-report", "bigclaw-go/docs/reports/multi-node-shared-queue-report.json", "multi-node shared queue report")
	takeoverReport := flags.String("takeover-report", "bigclaw-go/docs/reports/multi-subscriber-takeover-validation-report.json", "takeover harness report")
	liveTakeoverReport := flags.String("live-takeover-report", "bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-report.json", "live takeover report")
	output := flags.String("output", "bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json", "output path")
	pretty := flags.Bool("pretty", false, "print the generated JSON report")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation e2e cross-process-coordination-surface [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report, _, err := automationCrossProcessCoordinationSurface(automationCrossProcessCoordinationSurfaceOptions{
		RepoRoot:               absPath(*repoRoot),
		MultiNodeReportPath:    trim(*multiNodeReport),
		TakeoverReportPath:     trim(*takeoverReport),
		LiveTakeoverReportPath: trim(*liveTakeoverReport),
		OutputPath:             trim(*output),
	})
	if err != nil {
		return err
	}
	if *pretty {
		return emit(report, true, 0)
	}
	return nil
}

func automationCrossProcessCoordinationSurface(opts automationCrossProcessCoordinationSurfaceOptions) (map[string]any, int, error) {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	repoRoot := opts.RepoRoot
	multiNode, ok := automationReadJSON(automationResolveRepoPath(repoRoot, opts.MultiNodeReportPath)).(map[string]any)
	if !ok {
		return nil, 0, errors.New("failed to load multi-node report")
	}
	takeover, ok := automationReadJSON(automationResolveRepoPath(repoRoot, opts.TakeoverReportPath)).(map[string]any)
	if !ok {
		return nil, 0, errors.New("failed to load takeover report")
	}
	liveTakeover, ok := automationReadJSON(automationResolveRepoPath(repoRoot, opts.LiveTakeoverReportPath)).(map[string]any)
	if !ok {
		return nil, 0, errors.New("failed to load live takeover report")
	}
	takeoverSummary, _ := takeover["summary"].(map[string]any)
	liveTakeoverSummary, _ := liveTakeover["summary"].(map[string]any)
	report := map[string]any{
		"generated_at":                    utcISOTime(now().UTC()),
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
		"target_contracts": buildCoordinationTargetContracts(),
		"summary": map[string]any{
			"shared_queue_total_tasks":               automationInt(multiNode["count"]),
			"shared_queue_cross_node_completions":    automationInt(multiNode["cross_node_completions"]),
			"shared_queue_duplicate_completed_tasks": automationListLen(multiNode["duplicate_completed_tasks"]),
			"shared_queue_duplicate_started_tasks":   automationListLen(multiNode["duplicate_started_tasks"]),
			"takeover_scenario_count":                automationInt(takeoverSummary["scenario_count"]),
			"takeover_passing_scenarios":             automationInt(takeoverSummary["passing_scenarios"]),
			"takeover_duplicate_delivery_count":      automationInt(takeoverSummary["duplicate_delivery_count"]),
			"takeover_stale_write_rejections":        automationInt(takeoverSummary["stale_write_rejections"]),
			"live_takeover_scenario_count":           automationInt(liveTakeoverSummary["scenario_count"]),
			"live_takeover_passing_scenarios":        automationInt(liveTakeoverSummary["passing_scenarios"]),
			"live_takeover_stale_write_rejections":   automationInt(liveTakeoverSummary["stale_write_rejections"]),
		},
		"capabilities": buildCoordinationCapabilities(),
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
	if err := automationWriteJSON(automationResolveRepoPath(repoRoot, opts.OutputPath), report); err != nil {
		return nil, 0, err
	}
	return report, 0, nil
}

func buildCoordinationTargetContracts() []any {
	return []any{
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
			"guarantees": []any{
				"Partition keys are provider-neutral and map to existing trace/task/event_type selectors.",
				"Partition metadata may vary by backend, but portable replay ordering still uses Position.Sequence.",
				"No runtime implementation is shipped yet; this row defines the future adapter contract only.",
			},
		},
		map[string]any{
			"capability":      "broker_backed_subscriber_ownership",
			"contract_anchor": "events.SubscriptionRequest.OwnershipContract",
			"runtime_status":  "contract_only",
			"ownership": map[string]any{
				"subscriber_group": "shared durable consumer identity",
				"mode":             "exclusive",
				"lease_fields":     []any{"epoch", "lease_token"},
				"partition_hints":  "optional partition affinity for future broker-backed consumers",
			},
			"guarantees": []any{
				"Checkpoint commits remain fenced by epoch plus lease token after ownership transfer.",
				"Ownership metadata travels through the neutral subscription contract instead of provider-specific APIs.",
				"No broker-backed runtime implementation is shipped yet; this row defines the future ownership contract only.",
			},
		},
	}
}

func buildCoordinationCapabilities() []any {
	return []any{
		map[string]any{"capability": "shared_queue_task_coordination", "current_state": "implemented", "runtime_readiness": "live_proven", "live_local_proof": true, "deterministic_local_harness": false, "contract_defined_target": true, "notes": []any{"Two independent bigclawd processes share one SQLite-backed queue without duplicate terminal execution.", "Current proof is local and SQLite-backed rather than broker-backed or replicated."}},
		map[string]any{"capability": "subscriber_takeover_semantics", "current_state": "implemented_with_shared_durable_scaffold", "runtime_readiness": "live_proven", "live_local_proof": true, "deterministic_local_harness": true, "contract_defined_target": true, "notes": []any{"Lease handoff, stale-writer fencing, and duplicate replay accounting are covered by both the deterministic harness and the live two-node companion proof.", "The live proof now drives both nodes against one shared SQLite lease backend, but the provider-neutral broker-backed ownership contract is still not runtime-proven."}},
		map[string]any{"capability": "cross_process_replay_coordination", "current_state": "contract_defined", "runtime_readiness": "harness_proven", "live_local_proof": false, "deterministic_local_harness": true, "contract_defined_target": true, "notes": []any{"Replay cursor and checkpoint expectations are codified across the local takeover harness and durability rollout contract.", "No broker-backed or partitioned live replay proof exists yet."}},
		map[string]any{"capability": "stale_writer_fencing", "current_state": "implemented_with_shared_durable_scaffold", "runtime_readiness": "live_proven", "live_local_proof": true, "deterministic_local_harness": true, "contract_defined_target": true, "notes": []any{"The local takeover harness and the live two-node companion proof both show stale checkpoint writers being fenced after ownership transfer.", "The shared durable scaffold is SQLite-backed today, so the broker-backed ownership contract remains future work beyond the current local proof."}},
		map[string]any{"capability": "partitioned_topic_routing", "current_state": "not_available", "runtime_readiness": "contract_only", "live_local_proof": false, "deterministic_local_harness": false, "contract_defined_target": true, "notes": []any{"No partitioned topic model exists yet in the runtime.", "Broker-backed target docs reserve partition or quorum log ordering as the future coordination scope."}},
		map[string]any{"capability": "broker_backed_subscriber_ownership", "current_state": "not_available", "runtime_readiness": "contract_only", "live_local_proof": false, "deterministic_local_harness": false, "contract_defined_target": true, "notes": []any{"No broker-backed cross-process subscriber ownership model exists yet.", "The durability rollout contract defines the expected checkpoint fencing and failover guarantees before rollout-safe claims."}},
		map[string]any{"capability": "operator_capability_surface", "current_state": "implemented", "runtime_readiness": "supporting_surface", "live_local_proof": false, "deterministic_local_harness": false, "contract_defined_target": true, "notes": []any{"The repo exposes provider-neutral durability and rollout metadata through docs and runtime-facing event_durability surfaces.", "This report adds a coordination-specific surface tying together live local proof, deterministic local harnesses, and future targets."}},
	}
}

func utcISOTime(moment time.Time) string {
	return moment.UTC().Format(time.RFC3339)
}
