package regression

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/testharness"
)

func TestCorpusManifestScorecardMarksUncoveredShapes(t *testing.T) {
	tmpDir := t.TempDir()
	fixtureBasic := filepath.Join(tmpDir, "fixture-basic.json")
	writeShadowMatrixJSON(t, fixtureBasic, map[string]any{
		"id":                "fixture-basic",
		"required_executor": "local",
		"entrypoint":        "echo basic",
		"metadata":          map[string]any{"scenario": "baseline"},
	})
	fixtureBudget := filepath.Join(tmpDir, "fixture-budget.json")
	writeShadowMatrixJSON(t, fixtureBudget, map[string]any{
		"id":                "fixture-budget",
		"required_executor": "local",
		"entrypoint":        "echo budget",
		"budget_cents":      25,
		"labels":            []string{"budget", "shadow"},
		"metadata":          map[string]any{"scenario": "budget"},
	})
	manifestPath := filepath.Join(tmpDir, "corpus.json")
	writeShadowMatrixJSON(t, manifestPath, map[string]any{
		"name":         "anon-corpus",
		"generated_at": "2026-03-16T10:00:00Z",
		"slices": []map[string]any{
			{
				"slice_id":  "baseline",
				"title":     "Baseline slice",
				"weight":    10,
				"task_file": "./fixture-basic.json",
			},
			{
				"slice_id": "validation",
				"title":    "Validation slice",
				"weight":   6,
				"task": map[string]any{
					"id":                  "validation-task",
					"required_executor":   "local",
					"entrypoint":          "echo validation",
					"acceptance_criteria": []string{"ok"},
					"validation_plan":     []string{"compare output"},
					"metadata":            map[string]any{"scenario": "validation"},
				},
			},
			{
				"slice_id":   "browser-review",
				"title":      "Browser review",
				"weight":     3,
				"task_shape": "executor:browser|labels:browser,human-review|scenario:browser-review",
				"tags":       []string{"browser", "human-review"},
			},
		},
	})

	output := runShadowMatrixSnippet(t, strings.Join([]string{
		"fixture_entries = module.load_fixture_entries([r'" + fixtureBasic + "', r'" + fixtureBudget + "'])",
		"manifest_meta, replay_entries, corpus_slices = module.load_corpus_manifest_entries(r'" + manifestPath + "')",
		"coverage = module.build_corpus_coverage(fixture_entries, corpus_slices, manifest_meta)",
		"print(json.dumps({'replay_entries': replay_entries, 'coverage': coverage}))",
	}, "\n"))

	var result struct {
		ReplayEntries []any `json:"replay_entries"`
		Coverage      struct {
			ManifestName               string `json:"manifest_name"`
			FixtureTaskCount           int    `json:"fixture_task_count"`
			CorpusSliceCount           int    `json:"corpus_slice_count"`
			CorpusReplayableSliceCount int    `json:"corpus_replayable_slice_count"`
			CoveredCorpusSliceCount    int    `json:"covered_corpus_slice_count"`
			UncoveredCorpusSliceCount  int    `json:"uncovered_corpus_slice_count"`
			UncoveredSlices            []struct {
				SliceID string `json:"slice_id"`
			} `json:"uncovered_slices"`
			ShapeScorecard []struct {
				TaskShape        string `json:"task_shape"`
				CoveredByFixture bool   `json:"covered_by_fixture"`
			} `json:"shape_scorecard"`
		} `json:"coverage"`
	}
	decodeShadowMatrixJSON(t, output, &result)

	if len(result.ReplayEntries) != 0 {
		t.Fatalf("expected no replay entries, got %+v", result.ReplayEntries)
	}
	coverage := result.Coverage
	if coverage.ManifestName != "anon-corpus" ||
		coverage.FixtureTaskCount != 2 ||
		coverage.CorpusSliceCount != 3 ||
		coverage.CorpusReplayableSliceCount != 2 ||
		coverage.CoveredCorpusSliceCount != 1 ||
		coverage.UncoveredCorpusSliceCount != 2 {
		t.Fatalf("unexpected coverage summary: %+v", coverage)
	}
	if len(coverage.UncoveredSlices) != 2 ||
		coverage.UncoveredSlices[0].SliceID != "validation" ||
		coverage.UncoveredSlices[1].SliceID != "browser-review" {
		t.Fatalf("unexpected uncovered slices: %+v", coverage.UncoveredSlices)
	}
	found := false
	for _, item := range coverage.ShapeScorecard {
		if item.TaskShape == "executor:local|scenario:baseline" && item.CoveredByFixture {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected covered baseline shape in %+v", coverage.ShapeScorecard)
	}
}

func TestCorpusManifestCanPromoteReplayableSlices(t *testing.T) {
	tmpDir := t.TempDir()
	taskFile := filepath.Join(tmpDir, "task.json")
	writeShadowMatrixJSON(t, taskFile, map[string]any{
		"id":                "replayable-task",
		"required_executor": "local",
		"entrypoint":        "echo replay",
		"metadata":          map[string]any{"scenario": "baseline"},
	})
	manifestPath := filepath.Join(tmpDir, "corpus.json")
	writeShadowMatrixJSON(t, manifestPath, map[string]any{
		"name": "replay-pack",
		"slices": []map[string]any{
			{
				"slice_id":  "baseline",
				"title":     "Baseline slice",
				"weight":    2,
				"task_file": "./task.json",
				"replay":    true,
			},
			{
				"slice_id":   "metadata-only",
				"title":      "Metadata only",
				"weight":     1,
				"task_shape": "executor:browser|scenario:review",
			},
		},
	})

	output := runShadowMatrixSnippet(t, strings.Join([]string{
		"manifest_meta, replay_entries, corpus_slices = module.load_corpus_manifest_entries(r'" + manifestPath + "', replay_corpus_slices=True)",
		"report = module.build_report([], [], corpus_slices=corpus_slices, manifest_meta=manifest_meta)",
		"print(json.dumps({'replay_entries': replay_entries, 'report': report}))",
	}, "\n"))

	var result struct {
		ReplayEntries []struct {
			SourceKind  string `json:"source_kind"`
			CorpusSlice struct {
				ID string `json:"id"`
			} `json:"corpus_slice"`
		} `json:"replay_entries"`
		Report struct {
			Inputs struct {
				ManifestName string `json:"manifest_name"`
			} `json:"inputs"`
			CorpusCoverage struct {
				UncoveredCorpusSliceCount int `json:"uncovered_corpus_slice_count"`
			} `json:"corpus_coverage"`
		} `json:"report"`
	}
	decodeShadowMatrixJSON(t, output, &result)

	if len(result.ReplayEntries) != 1 ||
		result.ReplayEntries[0].SourceKind != "corpus" ||
		result.ReplayEntries[0].CorpusSlice.ID != "baseline" {
		t.Fatalf("unexpected replay entries: %+v", result.ReplayEntries)
	}
	if result.Report.Inputs.ManifestName != "replay-pack" ||
		result.Report.CorpusCoverage.UncoveredCorpusSliceCount != 2 {
		t.Fatalf("unexpected report payload: %+v", result.Report)
	}
}

func runShadowMatrixSnippet(t *testing.T, body string) []byte {
	t.Helper()
	scriptPath := testharness.JoinRepoRoot(t, "scripts", "migration", "shadow_matrix.py")
	snippet := strings.Join([]string{
		"import importlib.util, json, pathlib, sys",
		"script_path = pathlib.Path(r'" + scriptPath + "')",
		"sys.path.insert(0, str(script_path.parent))",
		"spec = importlib.util.spec_from_file_location('shadow_matrix', script_path)",
		"module = importlib.util.module_from_spec(spec)",
		"assert spec.loader is not None",
		"spec.loader.exec_module(module)",
		body,
	}, "\n")
	cmd := testharness.PythonCommand(t, "-c", snippet)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run shadow matrix snippet: %v (%s)", err, string(output))
	}
	return output
}

func writeShadowMatrixJSON(t *testing.T, path string, payload map[string]any) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func decodeShadowMatrixJSON(t *testing.T, body []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("decode json: %v (%s)", err, string(body))
	}
}
