package legacyshim

import "testing"

func TestPythonBuildSurfaceReportRetiresPackagingMetadata(t *testing.T) {
	report := PythonBuildSurfaceReport()
	if report.SurfaceID != "legacy-python-build-surface" || report.Status != "go-owned" {
		t.Fatalf("unexpected build surface report header: %+v", report)
	}
	if report.ReplacementCommand != "bash scripts/ops/bigclawctl legacy-python build-surface --json" {
		t.Fatalf("unexpected replacement command: %q", report.ReplacementCommand)
	}
	if len(report.RetiredAssets) != 2 {
		t.Fatalf("expected retired packaging assets, got %+v", report.RetiredAssets)
	}
	if report.RetiredAssets[0].Path != "pyproject.toml" || report.RetiredAssets[1].Path != "setup.py" {
		t.Fatalf("unexpected retired asset paths: %+v", report.RetiredAssets)
	}
	if len(report.ValidationCommands) < 4 {
		t.Fatalf("expected validation commands, got %+v", report.ValidationCommands)
	}
}

func TestPythonBuildSurfaceReportKeepsStandaloneToolConfigs(t *testing.T) {
	report := PythonBuildSurfaceReport()
	paths := map[string]BuildSurfaceAsset{}
	for _, asset := range report.ActiveAssets {
		paths[asset.Path] = asset
	}
	if paths["pytest.ini"].Kind != "python_tool_config" {
		t.Fatalf("expected pytest.ini tool config, got %+v", paths["pytest.ini"])
	}
	if paths[".ruff.toml"].Kind != "python_tool_config" {
		t.Fatalf("expected .ruff.toml tool config, got %+v", paths[".ruff.toml"])
	}
	if paths["scripts/dev_bootstrap.sh"].Kind != "migration_bootstrap" {
		t.Fatalf("expected bootstrap asset, got %+v", paths["scripts/dev_bootstrap.sh"])
	}
}
