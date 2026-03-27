package crossprocesscoordination

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type BuildOptions struct {
	RepoRoot               string
	MultiNodeReportPath    string
	TakeoverReportPath     string
	LiveTakeoverReportPath string
	GeneratedAt            time.Time
}

type reportPayload struct {
	GeneratedAt                  string            `json:"generated_at"`
	Ticket                       string            `json:"ticket"`
	Title                        string            `json:"title"`
	Status                       string            `json:"status"`
	TargetContractSurfaceVersion string            `json:"target_contract_surface_version"`
	RuntimeReadinessLevels       map[string]string `json:"runtime_readiness_levels"`
	EvidenceInputs               evidenceInputs    `json:"evidence_inputs"`
	TargetContracts              []targetContract  `json:"target_contracts"`
	Summary                      summary           `json:"summary"`
	Capabilities                 []capability      `json:"capabilities"`
	CurrentCeiling               []string          `json:"current_ceiling"`
	NextRuntimeHooks             []string          `json:"next_runtime_hooks"`
}

type evidenceInputs struct {
	SharedQueueReport     string   `json:"shared_queue_report"`
	TakeoverHarnessReport string   `json:"takeover_harness_report"`
	LiveTakeoverReport    string   `json:"live_takeover_report"`
	SupportingDocs        []string `json:"supporting_docs"`
}

type targetContract struct {
	Capability     string         `json:"capability"`
	ContractAnchor string         `json:"contract_anchor"`
	RuntimeStatus  string         `json:"runtime_status"`
	Partitioning   map[string]any `json:"partitioning"`
	Ownership      map[string]any `json:"ownership"`
	Guarantees     []string       `json:"guarantees"`
}

type capability struct {
	Capability              string   `json:"capability"`
	CurrentState            string   `json:"current_state"`
	RuntimeReadiness        string   `json:"runtime_readiness"`
	LiveLocalProof          bool     `json:"live_local_proof"`
	DeterministicLocalProof bool     `json:"deterministic_local_harness"`
	ContractDefinedTarget   bool     `json:"contract_defined_target"`
	Notes                   []string `json:"notes"`
}

type summary struct {
	SharedQueueTotalTasks              int `json:"shared_queue_total_tasks"`
	SharedQueueCrossNodeCompletions    int `json:"shared_queue_cross_node_completions"`
	SharedQueueDuplicateCompletedTasks int `json:"shared_queue_duplicate_completed_tasks"`
	SharedQueueDuplicateStartedTasks   int `json:"shared_queue_duplicate_started_tasks"`
	TakeoverScenarioCount              int `json:"takeover_scenario_count"`
	TakeoverPassingScenarios           int `json:"takeover_passing_scenarios"`
	TakeoverDuplicateDeliveryCount     int `json:"takeover_duplicate_delivery_count"`
	TakeoverStaleWriteRejections       int `json:"takeover_stale_write_rejections"`
	LiveTakeoverScenarioCount          int `json:"live_takeover_scenario_count"`
	LiveTakeoverPassingScenarios       int `json:"live_takeover_passing_scenarios"`
	LiveTakeoverStaleWriteRejections   int `json:"live_takeover_stale_write_rejections"`
}

type multiNodeReport struct {
	Count                   int   `json:"count"`
	CrossNodeCompletions    int   `json:"cross_node_completions"`
	DuplicateCompletedTasks []any `json:"duplicate_completed_tasks"`
	DuplicateStartedTasks   []any `json:"duplicate_started_tasks"`
}

type takeoverReport struct {
	Summary struct {
		ScenarioCount          int `json:"scenario_count"`
		PassingScenarios       int `json:"passing_scenarios"`
		DuplicateDeliveryCount int `json:"duplicate_delivery_count"`
		StaleWriteRejections   int `json:"stale_write_rejections"`
	} `json:"summary"`
}

func BuildReport(opts BuildOptions) (map[string]any, error) {
	repoRoot := opts.RepoRoot
	if repoRoot == "" {
		repoRoot = defaultRepoRoot()
	}
	multiNodePath := defaultString(opts.MultiNodeReportPath, "bigclaw-go/docs/reports/multi-node-shared-queue-report.json")
	takeoverPath := defaultString(opts.TakeoverReportPath, "bigclaw-go/docs/reports/multi-subscriber-takeover-validation-report.json")
	liveTakeoverPath := defaultString(opts.LiveTakeoverReportPath, "bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-report.json")
	generatedAt := opts.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}

	var multiNode multiNodeReport
	if err := readJSON(resolveRepoPath(repoRoot, multiNodePath), &multiNode); err != nil {
		return nil, err
	}
	var takeover takeoverReport
	if err := readJSON(resolveRepoPath(repoRoot, takeoverPath), &takeover); err != nil {
		return nil, err
	}
	var liveTakeover takeoverReport
	if err := readJSON(resolveRepoPath(repoRoot, liveTakeoverPath), &liveTakeover); err != nil {
		return nil, err
	}

	report := reportPayload{
		GeneratedAt:                  generatedAt.Format(time.RFC3339),
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
		EvidenceInputs: evidenceInputs{
			SharedQueueReport:     multiNodePath,
			TakeoverHarnessReport: takeoverPath,
			LiveTakeoverReport:    liveTakeoverPath,
			SupportingDocs: []string{
				"bigclaw-go/docs/reports/event-bus-reliability-report.md",
				"bigclaw-go/docs/reports/replicated-event-log-durability-rollout-contract.md",
				"bigclaw-go/docs/reports/broker-event-log-adapter-contract.md",
			},
		},
		TargetContracts: []targetContract{
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
				Ownership: map[string]any{},
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
				Partitioning:   map[string]any{},
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
		},
		Summary: summary{
			SharedQueueTotalTasks:              multiNode.Count,
			SharedQueueCrossNodeCompletions:    multiNode.CrossNodeCompletions,
			SharedQueueDuplicateCompletedTasks: len(multiNode.DuplicateCompletedTasks),
			SharedQueueDuplicateStartedTasks:   len(multiNode.DuplicateStartedTasks),
			TakeoverScenarioCount:              takeover.Summary.ScenarioCount,
			TakeoverPassingScenarios:           takeover.Summary.PassingScenarios,
			TakeoverDuplicateDeliveryCount:     takeover.Summary.DuplicateDeliveryCount,
			TakeoverStaleWriteRejections:       takeover.Summary.StaleWriteRejections,
			LiveTakeoverScenarioCount:          liveTakeover.Summary.ScenarioCount,
			LiveTakeoverPassingScenarios:       liveTakeover.Summary.PassingScenarios,
			LiveTakeoverStaleWriteRejections:   liveTakeover.Summary.StaleWriteRejections,
		},
		Capabilities: []capability{
			{
				Capability:              "shared_queue_task_coordination",
				CurrentState:            "implemented",
				RuntimeReadiness:        "live_proven",
				LiveLocalProof:          true,
				DeterministicLocalProof: false,
				ContractDefinedTarget:   true,
				Notes: []string{
					"Two independent bigclawd processes share one SQLite-backed queue without duplicate terminal execution.",
					"Current proof is local and SQLite-backed rather than broker-backed or replicated.",
				},
			},
			{
				Capability:              "subscriber_takeover_semantics",
				CurrentState:            "implemented_with_shared_durable_scaffold",
				RuntimeReadiness:        "live_proven",
				LiveLocalProof:          true,
				DeterministicLocalProof: true,
				ContractDefinedTarget:   true,
				Notes: []string{
					"Lease handoff, stale-writer fencing, and duplicate replay accounting are covered by both the deterministic harness and the live two-node companion proof.",
					"The live proof now drives both nodes against one shared SQLite lease backend, but the provider-neutral broker-backed ownership contract is still not runtime-proven.",
				},
			},
			{
				Capability:              "cross_process_replay_coordination",
				CurrentState:            "contract_defined",
				RuntimeReadiness:        "harness_proven",
				LiveLocalProof:          false,
				DeterministicLocalProof: true,
				ContractDefinedTarget:   true,
				Notes: []string{
					"Replay cursor and checkpoint expectations are codified across the local takeover harness and durability rollout contract.",
					"No broker-backed or partitioned live replay proof exists yet.",
				},
			},
			{
				Capability:              "stale_writer_fencing",
				CurrentState:            "implemented_with_shared_durable_scaffold",
				RuntimeReadiness:        "live_proven",
				LiveLocalProof:          true,
				DeterministicLocalProof: true,
				ContractDefinedTarget:   true,
				Notes: []string{
					"The local takeover harness and the live two-node companion proof both show stale checkpoint writers being fenced after ownership transfer.",
					"The shared durable scaffold is SQLite-backed today, so the broker-backed ownership contract remains future work beyond the current local proof.",
				},
			},
			{
				Capability:              "partitioned_topic_routing",
				CurrentState:            "not_available",
				RuntimeReadiness:        "contract_only",
				LiveLocalProof:          false,
				DeterministicLocalProof: false,
				ContractDefinedTarget:   true,
				Notes: []string{
					"No partitioned topic model exists yet in the runtime.",
					"Broker-backed target docs reserve partition or quorum log ordering as the future coordination scope.",
				},
			},
			{
				Capability:              "broker_backed_subscriber_ownership",
				CurrentState:            "not_available",
				RuntimeReadiness:        "contract_only",
				LiveLocalProof:          false,
				DeterministicLocalProof: false,
				ContractDefinedTarget:   true,
				Notes: []string{
					"No broker-backed cross-process subscriber ownership model exists yet.",
					"The durability rollout contract defines the expected checkpoint fencing and failover guarantees before rollout-safe claims.",
				},
			},
			{
				Capability:              "operator_capability_surface",
				CurrentState:            "implemented",
				RuntimeReadiness:        "supporting_surface",
				LiveLocalProof:          false,
				DeterministicLocalProof: false,
				ContractDefinedTarget:   true,
				Notes: []string{
					"The repo exposes provider-neutral durability and rollout metadata through docs and runtime-facing event_durability surfaces.",
					"This report adds a coordination-specific surface tying together live local proof, deterministic local harnesses, and future targets.",
				},
			},
		},
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
	}

	body, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}
	var output map[string]any
	if err := json.Unmarshal(body, &output); err != nil {
		return nil, err
	}
	return output, nil
}

func WriteReport(path string, report map[string]any, pretty bool) error {
	var body []byte
	var err error
	if pretty {
		body, err = json.MarshalIndent(report, "", "  ")
	} else {
		body, err = json.Marshal(report)
	}
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func defaultRepoRoot() string {
	return filepath.Clean(filepath.Join(filepath.Dir(os.Args[0]), "..", "..", ".."))
}

func resolveRepoPath(repoRoot, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(repoRoot, path)
}

func readJSON(path string, dst any) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, dst)
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
