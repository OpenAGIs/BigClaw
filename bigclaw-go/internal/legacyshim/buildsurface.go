package legacyshim

type BuildSurfaceAsset struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Status  string `json:"status"`
	Purpose string `json:"purpose"`
}

type BuildSurfaceReport struct {
	SurfaceID          string              `json:"surface_id"`
	Status             string              `json:"status"`
	Summary            string              `json:"summary"`
	ReplacementCommand string              `json:"replacement_command"`
	ActiveAssets       []BuildSurfaceAsset `json:"active_assets"`
	RetiredAssets      []BuildSurfaceAsset `json:"retired_assets"`
	RemovalConditions  []string            `json:"removal_conditions"`
	ValidationCommands []string            `json:"validation_commands"`
}

func PythonBuildSurfaceReport() BuildSurfaceReport {
	return BuildSurfaceReport{
		SurfaceID:          "legacy-python-build-surface",
		Status:             "go-owned",
		Summary:            "Python packaging/build metadata has been retired; the remaining Python surface is frozen for migration-only validation.",
		ReplacementCommand: "bash scripts/ops/bigclawctl legacy-python build-surface --json",
		ActiveAssets: []BuildSurfaceAsset{
			{
				Path:    "src/bigclaw",
				Kind:    "python_runtime_reference",
				Status:  "frozen",
				Purpose: "legacy Python implementation kept only as migration reference until Go parity closes the remaining gaps",
			},
			{
				Path:    "tests",
				Kind:    "python_regression_suite",
				Status:  "frozen",
				Purpose: "migration-only regression coverage for the remaining legacy modules",
			},
			{
				Path:    "pytest.ini",
				Kind:    "python_tool_config",
				Status:  "active",
				Purpose: "standalone pytest configuration replacing pyproject-based test configuration",
			},
			{
				Path:    ".ruff.toml",
				Kind:    "python_tool_config",
				Status:  "active",
				Purpose: "standalone Ruff configuration replacing pyproject-based lint configuration",
			},
			{
				Path:    "scripts/dev_bootstrap.sh",
				Kind:    "migration_bootstrap",
				Status:  "active",
				Purpose: "optional bootstrap for frozen Python migration tooling without editable installs or wheel builds",
			},
		},
		RetiredAssets: []BuildSurfaceAsset{
			{
				Path:    "pyproject.toml",
				Kind:    "python_packaging_metadata",
				Status:  "retired",
				Purpose: "removed setuptools build backend and optional dependency packaging surface",
			},
			{
				Path:    "setup.py",
				Kind:    "python_packaging_metadata",
				Status:  "retired",
				Purpose: "removed legacy setuptools compatibility shim",
			},
		},
		RemovalConditions: []string{
			"`src/bigclaw` no longer contains any migration-blocking runtime or operator path that lacks a Go implementation.",
			"`tests/` no longer requires Python-only regression coverage for active operator workflows.",
			"`bash scripts/ops/bigclawctl legacy-python compile-check --json` is no longer needed because the frozen compatibility shims have been deleted.",
			"README and bootstrap flows no longer document any Python migration command as required for normal development or release validation.",
		},
		ValidationCommands: []string{
			"cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl",
			"PYTHONPATH=src python3 -m pytest",
			"python3 -m ruff check src tests scripts",
			"bash scripts/ops/bigclawctl legacy-python build-surface --json",
			"bash scripts/ops/bigclawctl legacy-python compile-check --json",
		},
	}
}
