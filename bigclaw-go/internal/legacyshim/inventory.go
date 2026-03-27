package legacyshim

type InventoryEntry struct {
	ScriptPath          string   `json:"script_path"`
	Status              string   `json:"status"`
	Category            string   `json:"category"`
	Wave                string   `json:"wave"`
	ReplacementCommand  string   `json:"replacement_command,omitempty"`
	CompatibilityLayer  string   `json:"compatibility_layer"`
	VerificationCommand string   `json:"verification_command"`
	RegressionSurface   []string `json:"regression_surface,omitempty"`
	Notes               string   `json:"notes,omitempty"`
}

type Inventory struct {
	Issue               string           `json:"issue"`
	BranchSuggestion    string           `json:"branch_suggestion"`
	PRTitleSuggestion   string           `json:"pr_title_suggestion"`
	CompatibilityPolicy []string         `json:"compatibility_policy"`
	Entries             []InventoryEntry `json:"entries"`
}

func MigrationInventory() Inventory {
	regressionCore := []string{
		"CLI parsing and help text",
		"JSON report/output compatibility for existing operators",
		"Python compatibility shim forwarding",
	}
	return Inventory{
		Issue:             "BIG-GO-902",
		BranchSuggestion:  "feat/BIG-GO-902-go-cli-script-migration",
		PRTitleSuggestion: "feat: migrate script-layer automation to bigclawctl",
		CompatibilityPolicy: []string{
			"Keep migrated Python entrypoints as thin shims that forward to bigclawctl.",
			"Do not add net-new behavior to Python shims; new logic belongs in Go.",
			"Retire each shim only after docs, CI, and operators have used the Go command for one rollout cycle.",
		},
		Entries: []InventoryEntry{
			{
				ScriptPath:          "scripts/create_issues.py",
				Status:              "migrated-shim",
				Category:            "ops",
				Wave:                "wave-0",
				ReplacementCommand:  "bash scripts/ops/bigclawctl create-issues --json",
				CompatibilityLayer:  "python shim -> scripts/ops/bigclawctl -> bigclawctl create-issues",
				VerificationCommand: "python3 scripts/create_issues.py --help",
				RegressionSurface:   regressionCore,
			},
			{
				ScriptPath:          "scripts/dev_smoke.py",
				Status:              "migrated-shim",
				Category:            "ops",
				Wave:                "wave-0",
				ReplacementCommand:  "bash scripts/ops/bigclawctl dev-smoke --json",
				CompatibilityLayer:  "python shim -> scripts/ops/bigclawctl -> bigclawctl dev-smoke",
				VerificationCommand: "python3 scripts/dev_smoke.py --json",
				RegressionSurface:   regressionCore,
			},
			{
				ScriptPath:          "scripts/ops/bigclaw_github_sync.py",
				Status:              "migrated-shim",
				Category:            "ops",
				Wave:                "wave-0",
				ReplacementCommand:  "bash scripts/ops/bigclawctl github-sync status --json",
				CompatibilityLayer:  "python shim -> scripts/ops/bigclawctl -> bigclawctl github-sync",
				VerificationCommand: "python3 scripts/ops/bigclaw_github_sync.py status --help",
				RegressionSurface:   regressionCore,
			},
			{
				ScriptPath:          "scripts/ops/bigclaw_refill_queue.py",
				Status:              "migrated-shim",
				Category:            "ops",
				Wave:                "wave-0",
				ReplacementCommand:  "bash scripts/ops/bigclawctl refill --local-issues local-issues.json",
				CompatibilityLayer:  "python shim -> scripts/ops/bigclawctl -> bigclawctl refill",
				VerificationCommand: "python3 scripts/ops/bigclaw_refill_queue.py --help",
				RegressionSurface:   append([]string{}, regressionCore...),
				Notes:               "Queue policy and local issue state promotion remain the main regression area.",
			},
			{
				ScriptPath:          "scripts/ops/bigclaw_workspace_bootstrap.py",
				Status:              "migrated-shim",
				Category:            "workspace",
				Wave:                "wave-0",
				ReplacementCommand:  "bash scripts/ops/bigclawctl workspace bootstrap --help",
				CompatibilityLayer:  "python shim -> scripts/ops/bigclawctl -> bigclawctl workspace bootstrap",
				VerificationCommand: "python3 scripts/ops/bigclaw_workspace_bootstrap.py --help",
				RegressionSurface:   regressionCore,
			},
			{
				ScriptPath:          "scripts/ops/symphony_workspace_bootstrap.py",
				Status:              "migrated-shim",
				Category:            "workspace",
				Wave:                "wave-0",
				ReplacementCommand:  "bash scripts/ops/bigclawctl workspace bootstrap --help",
				CompatibilityLayer:  "python shim -> scripts/ops/bigclawctl -> bigclawctl workspace bootstrap",
				VerificationCommand: "python3 scripts/ops/symphony_workspace_bootstrap.py --help",
				RegressionSurface:   regressionCore,
			},
			{
				ScriptPath:          "scripts/ops/symphony_workspace_validate.py",
				Status:              "migrated-shim",
				Category:            "workspace",
				Wave:                "wave-0",
				ReplacementCommand:  "bash scripts/ops/bigclawctl workspace validate --help",
				CompatibilityLayer:  "python shim -> scripts/ops/bigclawctl -> bigclawctl workspace validate",
				VerificationCommand: "python3 scripts/ops/symphony_workspace_validate.py --help",
				RegressionSurface:   regressionCore,
			},
			{
				ScriptPath:          "bigclaw-go/scripts/e2e/run_task_smoke.py",
				Status:              "migrated-shim",
				Category:            "e2e",
				Wave:                "wave-0",
				ReplacementCommand:  "go run ./cmd/bigclawctl automation e2e run-task-smoke --help",
				CompatibilityLayer:  "python shim -> bigclawctl automation e2e run-task-smoke",
				VerificationCommand: "cd bigclaw-go && python3 scripts/e2e/run_task_smoke.py --help",
				RegressionSurface:   append([]string{}, regressionCore...),
				Notes:               "Autostart flow still needs live coverage because it reserves an ephemeral port before bigclawd binds.",
			},
			{
				ScriptPath:          "bigclaw-go/scripts/benchmark/soak_local.py",
				Status:              "migrated-shim",
				Category:            "benchmark",
				Wave:                "wave-0",
				ReplacementCommand:  "go run ./cmd/bigclawctl automation benchmark soak-local --help",
				CompatibilityLayer:  "python shim -> bigclawctl automation benchmark soak-local",
				VerificationCommand: "cd bigclaw-go && python3 scripts/benchmark/soak_local.py --help",
				RegressionSurface:   append([]string{}, regressionCore...),
				Notes:               "Large local soaks may stress concurrency differently than the legacy Python thread pool.",
			},
			{
				ScriptPath:          "bigclaw-go/scripts/migration/shadow_compare.py",
				Status:              "migrated-shim",
				Category:            "migration",
				Wave:                "wave-0",
				ReplacementCommand:  "go run ./cmd/bigclawctl automation migration shadow-compare --help",
				CompatibilityLayer:  "python shim -> bigclawctl automation migration shadow-compare",
				VerificationCommand: "cd bigclaw-go && python3 scripts/migration/shadow_compare.py --help",
				RegressionSurface:   regressionCore,
			},
			{
				ScriptPath:          "bigclaw-go/scripts/e2e/export_validation_bundle.py",
				Status:              "migrated-shim",
				Category:            "e2e",
				Wave:                "wave-0",
				ReplacementCommand:  "go run ./cmd/bigclawctl automation e2e export-validation-bundle --help",
				CompatibilityLayer:  "python shim -> bigclawctl automation e2e export-validation-bundle",
				VerificationCommand: "cd bigclaw-go && python3 scripts/e2e/export_validation_bundle.py --help",
				RegressionSurface: []string{
					"Artifact copy semantics under docs/reports/",
					"Validation index JSON shape",
					"Latest report refresh behavior",
				},
				Notes: "Migrated first because multiple downstream validation reports consume its bundle layout and latest-report refresh behavior.",
			},
			{
				ScriptPath:          "bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py",
				Status:              "migrated-shim",
				Category:            "e2e",
				Wave:                "wave-0",
				ReplacementCommand:  "go run ./cmd/bigclawctl automation e2e validation-bundle-scorecard --help",
				CompatibilityLayer:  "python shim -> bigclawctl automation e2e validation-bundle-scorecard",
				VerificationCommand: "cd bigclaw-go && python3 scripts/e2e/validation_bundle_continuation_scorecard.py --help",
				RegressionSurface: []string{
					"Score aggregation across recent bundle runs",
					"JSON scorecard schema consumed by reports",
				},
			},
			{
				ScriptPath:          "bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py",
				Status:              "migrated-shim",
				Category:            "e2e",
				Wave:                "wave-0",
				ReplacementCommand:  "go run ./cmd/bigclawctl automation e2e validation-bundle-policy-gate --help",
				CompatibilityLayer:  "python shim -> bigclawctl automation e2e validation-bundle-policy-gate",
				VerificationCommand: "cd bigclaw-go && python3 scripts/e2e/validation_bundle_continuation_policy_gate.py --help",
				RegressionSurface: []string{
					"Gate recommendation policy",
					"Freshness and lane coverage checks",
					"Next-action text used by operators",
				},
			},
			{
				ScriptPath:          "bigclaw-go/scripts/e2e/multi_node_shared_queue.py",
				Status:              "pending-native-python",
				Category:            "e2e",
				Wave:                "wave-4",
				ReplacementCommand:  "go run ./cmd/bigclawctl automation e2e multi-node-shared-queue ...",
				CompatibilityLayer:  "planned python shim after Go harness lands",
				VerificationCommand: "cd bigclaw-go && python3 scripts/e2e/multi_node_shared_queue.py --help",
				RegressionSurface: []string{
					"Multi-node queue fairness and takeover timing",
					"Companion report JSON compatibility",
				},
			},
			{
				ScriptPath:          "bigclaw-go/scripts/e2e/mixed_workload_matrix.py",
				Status:              "pending-native-python",
				Category:            "e2e",
				Wave:                "wave-4",
				ReplacementCommand:  "go run ./cmd/bigclawctl automation e2e mixed-workload-matrix ...",
				CompatibilityLayer:  "planned python shim after Go harness lands",
				VerificationCommand: "cd bigclaw-go && python3 scripts/e2e/mixed_workload_matrix.py --help",
				RegressionSurface: []string{
					"Scenario matrix expansion",
					"Mixed workload aggregation and markdown outputs",
				},
			},
			{
				ScriptPath:          "bigclaw-go/scripts/e2e/external_store_validation.py",
				Status:              "pending-native-python",
				Category:            "e2e",
				Wave:                "wave-4",
				ReplacementCommand:  "go run ./cmd/bigclawctl automation e2e external-store-validation ...",
				CompatibilityLayer:  "planned python shim after Go harness lands",
				VerificationCommand: "cd bigclaw-go && python3 scripts/e2e/external_store_validation.py --help",
				RegressionSurface: []string{
					"External store contract coverage",
					"Validation report schema stability",
				},
			},
			{
				ScriptPath:          "bigclaw-go/scripts/e2e/cross_process_coordination_surface.py",
				Status:              "pending-native-python",
				Category:            "e2e",
				Wave:                "wave-4",
				ReplacementCommand:  "go run ./cmd/bigclawctl automation e2e cross-process-coordination-surface ...",
				CompatibilityLayer:  "planned python shim after Go harness lands",
				VerificationCommand: "cd bigclaw-go && python3 scripts/e2e/cross_process_coordination_surface.py --help",
				RegressionSurface: []string{
					"Cross-process coordination timing assertions",
					"Surface summary JSON compatibility",
				},
			},
			{
				ScriptPath:          "bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py",
				Status:              "pending-native-python",
				Category:            "e2e",
				Wave:                "wave-4",
				ReplacementCommand:  "go run ./cmd/bigclawctl automation e2e broker-failover-stub-matrix ...",
				CompatibilityLayer:  "planned python shim after Go harness lands",
				VerificationCommand: "cd bigclaw-go && python3 scripts/e2e/broker_failover_stub_matrix.py --help",
				RegressionSurface: []string{
					"Broker failover scenario matrix behavior",
					"Stub matrix report JSON compatibility",
				},
			},
			{
				ScriptPath:          "bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py",
				Status:              "pending-native-python",
				Category:            "e2e",
				Wave:                "wave-4",
				ReplacementCommand:  "go run ./cmd/bigclawctl automation e2e subscriber-takeover-fault-matrix ...",
				CompatibilityLayer:  "planned python shim after Go harness lands",
				VerificationCommand: "cd bigclaw-go && python3 scripts/e2e/subscriber_takeover_fault_matrix.py --help",
				RegressionSurface: []string{
					"Subscriber takeover fault timing",
					"Takeover validation report compatibility",
				},
			},
			{
				ScriptPath:          "bigclaw-go/scripts/benchmark/run_matrix.py",
				Status:              "pending-native-python",
				Category:            "benchmark",
				Wave:                "wave-2",
				ReplacementCommand:  "go run ./cmd/bigclawctl automation benchmark run-matrix ...",
				CompatibilityLayer:  "planned python shim after Go matrix runner lands",
				VerificationCommand: "cd bigclaw-go && python3 scripts/benchmark/run_matrix.py --help",
				RegressionSurface: []string{
					"Matrix orchestration over soak-local",
					"Report aggregation and benchmark markdown refresh",
				},
			},
			{
				ScriptPath:          "bigclaw-go/scripts/benchmark/capacity_certification.py",
				Status:              "pending-native-python",
				Category:            "benchmark",
				Wave:                "wave-2",
				ReplacementCommand:  "go run ./cmd/bigclawctl automation benchmark capacity-certification ...",
				CompatibilityLayer:  "planned python shim after Go certifier lands",
				VerificationCommand: "cd bigclaw-go && python3 scripts/benchmark/capacity_certification.py --help",
				RegressionSurface: []string{
					"Capacity threshold policy",
					"Certification summary JSON shape",
				},
			},
			{
				ScriptPath:          "bigclaw-go/scripts/migration/shadow_matrix.py",
				Status:              "pending-native-python",
				Category:            "migration",
				Wave:                "wave-3",
				ReplacementCommand:  "go run ./cmd/bigclawctl automation migration shadow-matrix ...",
				CompatibilityLayer:  "planned python shim after Go matrix runner lands",
				VerificationCommand: "cd bigclaw-go && python3 scripts/migration/shadow_matrix.py --help",
				RegressionSurface: []string{
					"Manifest-driven corpus expansion",
					"Per-scenario aggregation and report JSON layout",
				},
			},
			{
				ScriptPath:          "bigclaw-go/scripts/migration/live_shadow_scorecard.py",
				Status:              "pending-native-python",
				Category:            "migration",
				Wave:                "wave-3",
				ReplacementCommand:  "go run ./cmd/bigclawctl automation migration live-shadow-scorecard ...",
				CompatibilityLayer:  "planned python shim after Go scorecard lands",
				VerificationCommand: "cd bigclaw-go && python3 scripts/migration/live_shadow_scorecard.py --help",
				RegressionSurface: []string{
					"Historical run scanning under docs/reports/live-shadow-runs/",
					"Scorecard JSON compatibility",
				},
			},
			{
				ScriptPath:          "bigclaw-go/scripts/migration/export_live_shadow_bundle.py",
				Status:              "pending-native-python",
				Category:            "migration",
				Wave:                "wave-3",
				ReplacementCommand:  "go run ./cmd/bigclawctl automation migration export-live-shadow-bundle ...",
				CompatibilityLayer:  "planned python shim after Go exporter lands",
				VerificationCommand: "cd bigclaw-go && python3 scripts/migration/export_live_shadow_bundle.py --help",
				RegressionSurface: []string{
					"Bundle export semantics and latest pointer refresh",
					"Cross-linking to live shadow scorecards",
				},
			},
		},
	}
}
