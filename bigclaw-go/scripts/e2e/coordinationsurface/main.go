package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type capabilityRow struct {
	Capability                string   `json:"capability"`
	CurrentState              string   `json:"current_state"`
	RuntimeReadiness          string   `json:"runtime_readiness"`
	LiveLocalProof            bool     `json:"live_local_proof"`
	DeterministicLocalHarness bool     `json:"deterministic_local_harness"`
	ContractDefinedTarget     bool     `json:"contract_defined_target"`
	Notes                     []string `json:"notes"`
}

type targetContractRow struct {
	Capability     string         `json:"capability"`
	ContractAnchor string         `json:"contract_anchor"`
	RuntimeStatus  string         `json:"runtime_status"`
	Partitioning   map[string]any `json:"partitioning,omitempty"`
	Ownership      map[string]any `json:"ownership,omitempty"`
	Guarantees     []string       `json:"guarantees"`
}

type report struct {
	GeneratedAt                  string              `json:"generated_at"`
	Ticket                       string              `json:"ticket"`
	Title                        string              `json:"title"`
	Status                       string              `json:"status"`
	TargetContractSurfaceVersion string              `json:"target_contract_surface_version"`
	RuntimeReadinessLevels       map[string]string   `json:"runtime_readiness_levels"`
	EvidenceInputs               map[string]any      `json:"evidence_inputs"`
	TargetContracts              []targetContractRow `json:"target_contracts"`
	Summary                      map[string]any      `json:"summary"`
	Capabilities                 []capabilityRow     `json:"capabilities"`
	CurrentCeiling               []string            `json:"current_ceiling"`
	NextRuntimeHooks             []string            `json:"next_runtime_hooks"`
}

func main() {
	goRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	repoRoot := filepath.Clean(filepath.Join(goRoot, ".."))

	flags := flag.NewFlagSet("cross-process-coordination-surface", flag.ExitOnError)
	multiNodeReportPath := flags.String("multi-node-report", "bigclaw-go/docs/reports/multi-node-shared-queue-report.json", "multi-node shared queue report path")
	takeoverReportPath := flags.String("takeover-report", "bigclaw-go/docs/reports/multi-subscriber-takeover-validation-report.json", "takeover harness report path")
	liveTakeoverReportPath := flags.String("live-takeover-report", "bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-report.json", "live takeover report path")
	outputPath := flags.String("output", "bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json", "json output path")
	pretty := flags.Bool("pretty", false, "print the generated JSON report")
	if err := flags.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	rep, err := buildReport(repoRoot, *multiNodeReportPath, *takeoverReportPath, *liveTakeoverReportPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	body, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	resolvedOutputPath := resolveRepoPath(repoRoot, *outputPath)
	if err := os.MkdirAll(filepath.Dir(resolvedOutputPath), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.WriteFile(resolvedOutputPath, append(body, '\n'), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if *pretty {
		fmt.Println(string(body))
	}
}

func buildReport(repoRoot, multiNodeReportPath, takeoverReportPath, liveTakeoverReportPath string) (report, error) {
	var multiNode map[string]any
	if err := readJSON(resolveRepoPath(repoRoot, multiNodeReportPath), &multiNode); err != nil {
		return report{}, err
	}
	var takeover map[string]any
	if err := readJSON(resolveRepoPath(repoRoot, takeoverReportPath), &takeover); err != nil {
		return report{}, err
	}
	var liveTakeover map[string]any
	if err := readJSON(resolveRepoPath(repoRoot, liveTakeoverReportPath), &liveTakeover); err != nil {
		return report{}, err
	}

	summary := map[string]any{
		"shared_queue_total_tasks":               intValue(multiNode["count"]),
		"shared_queue_cross_node_completions":    intValue(multiNode["cross_node_completions"]),
		"shared_queue_duplicate_completed_tasks": listLen(multiNode["duplicate_completed_tasks"]),
		"shared_queue_duplicate_started_tasks":   listLen(multiNode["duplicate_started_tasks"]),
		"takeover_scenario_count":                intValue(nestedMap(takeover, "summary")["scenario_count"]),
		"takeover_passing_scenarios":             intValue(nestedMap(takeover, "summary")["passing_scenarios"]),
		"takeover_duplicate_delivery_count":      intValue(nestedMap(takeover, "summary")["duplicate_delivery_count"]),
		"takeover_stale_write_rejections":        intValue(nestedMap(takeover, "summary")["stale_write_rejections"]),
		"live_takeover_scenario_count":           intValue(nestedMap(liveTakeover, "summary")["scenario_count"]),
		"live_takeover_passing_scenarios":        intValue(nestedMap(liveTakeover, "summary")["passing_scenarios"]),
		"live_takeover_stale_write_rejections":   intValue(nestedMap(liveTakeover, "summary")["stale_write_rejections"]),
	}

	targetContracts := []targetContractRow{
		{
			Capability:     "partitioned_topic_routing",
			ContractAnchor: "events.SubscriptionRequest.PartitionRoute",
			RuntimeStatus:  "contract_only",
			Partitioning: map[string]any{
				"topic":                    "provider-defined shared event stream",
				"supported_partition_keys": []string{"trace_id", "task_id", "event_type"},
				"ordering_scope":           "sequence remains portable within the selected partition route",
				"filter_alignment":         "ReplayRequest task_id/trace_id filters must remain valid when a backend introduces partition routing.",
			},
			Guarantees: []string{
				"Partition keys are provider-neutral and map to existing trace/task/event_type selectors.",
				"Partition metadata may vary by backend, but portable replay ordering still uses Position.Sequence.",
				"No runtime implementation is shipped yet; this row defines the future adapter contract only.",
			},
		},
		{
			Capability:     "broker_backed_subscriber_ownership",
			ContractAnchor: "events.SubscriptionRequest.OwnershipContract",
			RuntimeStatus:  "contract_only",
			Ownership: map[string]any{
				"subscriber_group": "shared durable consumer identity",
				"mode":             "exclusive",
				"lease_fields":     []string{"epoch", "lease_token"},
				"partition_hints":  "optional partition affinity for future broker-backed consumers",
			},
			Guarantees: []string{
				"Checkpoint commits remain fenced by epoch plus lease token after ownership transfer.",
				"Ownership metadata travels through the neutral subscription contract instead of provider-specific APIs.",
				"No broker-backed runtime implementation is shipped yet; this row defines the future ownership contract only.",
			},
		},
	}

	capabilities := []capabilityRow{
		{
			Capability:                "shared_queue_task_coordination",
			CurrentState:              "implemented",
			RuntimeReadiness:          "live_proven",
			LiveLocalProof:            true,
			DeterministicLocalHarness: false,
			ContractDefinedTarget:     true,
			Notes: []string{
				"Two independent bigclawd processes share one SQLite-backed queue without duplicate terminal execution.",
				"Current proof is local and SQLite-backed rather than broker-backed or replicated.",
			},
		},
		{
			Capability:                "subscriber_takeover_semantics",
			CurrentState:              "implemented_with_shared_durable_scaffold",
			RuntimeReadiness:          "live_proven",
			LiveLocalProof:            true,
			DeterministicLocalHarness: true,
			ContractDefinedTarget:     true,
			Notes: []string{
				"Lease handoff, stale-writer fencing, and duplicate replay accounting are covered by both the deterministic harness and the live two-node companion proof.",
				"The live proof now drives both nodes against one shared SQLite lease backend, but the provider-neutral broker-backed ownership contract is still not runtime-proven.",
			},
		},
		{
			Capability:                "cross_process_replay_coordination",
			CurrentState:              "contract_defined",
			RuntimeReadiness:          "harness_proven",
			LiveLocalProof:            false,
			DeterministicLocalHarness: true,
			ContractDefinedTarget:     true,
			Notes: []string{
				"Replay cursor and checkpoint expectations are codified across the local takeover harness and durability rollout contract.",
				"No broker-backed or partitioned live replay proof exists yet.",
			},
		},
		{
			Capability:                "stale_writer_fencing",
			CurrentState:              "implemented_with_shared_durable_scaffold",
			RuntimeReadiness:          "live_proven",
			LiveLocalProof:            true,
			DeterministicLocalHarness: true,
			ContractDefinedTarget:     true,
			Notes: []string{
				"The local takeover harness and the live two-node companion proof both show stale checkpoint writers being fenced after ownership transfer.",
				"The shared durable scaffold is SQLite-backed today, so the broker-backed ownership contract remains future work beyond the current local proof.",
			},
		},
		{
			Capability:                "partitioned_topic_routing",
			CurrentState:              "not_available",
			RuntimeReadiness:          "contract_only",
			LiveLocalProof:            false,
			DeterministicLocalHarness: false,
			ContractDefinedTarget:     true,
			Notes: []string{
				"No partitioned topic model exists yet in the runtime.",
				"Broker-backed target docs reserve partition or quorum log ordering as the future coordination scope.",
			},
		},
		{
			Capability:                "broker_backed_subscriber_ownership",
			CurrentState:              "not_available",
			RuntimeReadiness:          "contract_only",
			LiveLocalProof:            false,
			DeterministicLocalHarness: false,
			ContractDefinedTarget:     true,
			Notes: []string{
				"No broker-backed cross-process subscriber ownership model exists yet.",
				"The durability rollout contract defines the expected checkpoint fencing and failover guarantees before rollout-safe claims.",
			},
		},
		{
			Capability:                "operator_capability_surface",
			CurrentState:              "implemented",
			RuntimeReadiness:          "supporting_surface",
			LiveLocalProof:            false,
			DeterministicLocalHarness: false,
			ContractDefinedTarget:     true,
			Notes: []string{
				"The repo exposes provider-neutral durability and rollout metadata through docs and runtime-facing event_durability surfaces.",
				"This report adds a coordination-specific surface tying together live local proof, deterministic local harnesses, and future targets.",
			},
		},
	}

	return report{
		GeneratedAt:                  utcISO(time.Now().UTC()),
		Ticket:                       "BIG-PAR-085-local-prework",
		Title:                        "Cross-process coordination capability surface",
		Status:                       "local-capability-surface",
		TargetContractSurfaceVersion: "2026-03-17",
		RuntimeReadinessLevels: map[string]string{
			"live_proven":        "Shipped runtime behavior with checked-in live cross-process proof.",
			"harness_proven":     "Deterministic executable harness coverage exists, but no live multi-node proof is checked in.",
			"contract_only":      "Only target contracts or rollout docs define the expected semantics today.",
			"supporting_surface": "The repo exposes reporting or metadata surfaces that describe runtime readiness without proving the coordination behavior itself.",
		},
		EvidenceInputs: map[string]any{
			"shared_queue_report":     multiNodeReportPath,
			"takeover_harness_report": takeoverReportPath,
			"live_takeover_report":    liveTakeoverReportPath,
			"supporting_docs": []string{
				"bigclaw-go/docs/reports/event-bus-reliability-report.md",
				"bigclaw-go/docs/reports/replicated-event-log-durability-rollout-contract.md",
				"bigclaw-go/docs/reports/broker-event-log-adapter-contract.md",
			},
		},
		TargetContracts: targetContracts,
		Summary:         summary,
		Capabilities:    capabilities,
		CurrentCeiling: []string{
			"no partitioned topic model",
			"no broker-backed cross-process subscriber coordination",
			"no broker-backed or replicated subscriber ownership backend",
		},
		NextRuntimeHooks: []string{
			"emit native takeover transition audit events from the runtime instead of harness-authored artifacts",
			"validate broker-backed replay and ownership semantics against the same report schema",
			"replace the SQLite shared durable scaffold with a broker-backed or replicated ownership backend",
		},
	}, nil
}

func readJSON(path string, target any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

func resolveRepoPath(repoRoot, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(repoRoot, path)
}

func utcISO(now time.Time) string {
	return now.UTC().Format(time.RFC3339Nano)
}

func nestedMap(input map[string]any, key string) map[string]any {
	value, ok := input[key].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return value
}

func intValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func listLen(value any) int {
	switch typed := value.(type) {
	case []any:
		return len(typed)
	default:
		return 0
	}
}
