package reporting

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareShadowTask(t *testing.T) {
	primary := newShadowTestServer([]map[string]any{
		{"id": "1", "type": "task.started", "timestamp": "2026-03-10T10:00:00Z"},
		{"id": "2", "type": "task.completed", "timestamp": "2026-03-10T10:00:03Z"},
	})
	defer primary.Close()
	shadow := newShadowTestServer([]map[string]any{
		{"id": "1", "type": "task.started", "timestamp": "2026-03-10T10:00:00Z"},
		{"id": "2", "type": "task.completed", "timestamp": "2026-03-10T10:00:04Z"},
	})
	defer shadow.Close()

	report, err := CompareShadowTask(primary.URL, shadow.URL, map[string]any{
		"id":                "shadow-task",
		"required_executor": "local",
		"entrypoint":        "echo hi",
	}, 0, 0, nil)
	if err != nil {
		t.Fatalf("compare shadow task: %v", err)
	}
	if asString(report["trace_id"]) != "shadow-task" {
		t.Fatalf("unexpected trace id: %+v", report)
	}
	diff := asMap(report["diff"])
	if !asBool(diff["state_equal"]) || !asBool(diff["event_types_equal"]) {
		t.Fatalf("unexpected diff: %+v", diff)
	}
	if asFloat(diff["primary_timeline_seconds"]) != 3 || asFloat(diff["shadow_timeline_seconds"]) != 4 {
		t.Fatalf("unexpected timelines: %+v", diff)
	}
}

func TestBuildShadowCorpusCoverage(t *testing.T) {
	root := t.TempDir()
	fixtureBasic := filepath.Join(root, "fixture-basic.json")
	writeShadowFixture(t, fixtureBasic, map[string]any{
		"id": "fixture-basic", "required_executor": "local", "entrypoint": "echo basic", "metadata": map[string]any{"scenario": "baseline"},
	})
	fixtureBudget := filepath.Join(root, "fixture-budget.json")
	writeShadowFixture(t, fixtureBudget, map[string]any{
		"id": "fixture-budget", "required_executor": "local", "entrypoint": "echo budget", "budget_cents": 25, "labels": []any{"budget", "shadow"}, "metadata": map[string]any{"scenario": "budget"},
	})
	manifest := filepath.Join(root, "corpus.json")
	writeShadowFixture(t, manifest, map[string]any{
		"name":         "anon-corpus",
		"generated_at": "2026-03-16T10:00:00Z",
		"slices": []map[string]any{
			{"slice_id": "baseline", "title": "Baseline slice", "weight": 10, "task_file": "./fixture-basic.json"},
			{"slice_id": "validation", "title": "Validation slice", "weight": 6, "task": map[string]any{"id": "validation-task", "required_executor": "local", "entrypoint": "echo validation", "acceptance_criteria": []any{"ok"}, "validation_plan": []any{"compare output"}, "metadata": map[string]any{"scenario": "validation"}}},
			{"slice_id": "browser-review", "title": "Browser review", "weight": 3, "task_shape": "executor:browser|labels:browser,human-review|scenario:browser-review", "tags": []any{"browser", "human-review"}},
		},
	})

	fixtureEntries, err := loadShadowFixtureEntries([]string{fixtureBasic, fixtureBudget})
	if err != nil {
		t.Fatalf("load fixture entries: %v", err)
	}
	manifestMeta, replayEntries, corpusSlices, err := loadShadowCorpusManifestEntries(manifest, false)
	if err != nil {
		t.Fatalf("load corpus manifest: %v", err)
	}
	coverage := buildShadowCorpusCoverage(fixtureEntries, corpusSlices, manifestMeta)

	if len(replayEntries) != 0 {
		t.Fatalf("unexpected replay entries: %+v", replayEntries)
	}
	if asString(coverage["manifest_name"]) != "anon-corpus" || asInt(coverage["fixture_task_count"]) != 2 || asInt(coverage["corpus_slice_count"]) != 3 {
		t.Fatalf("unexpected coverage summary: %+v", coverage)
	}
	if asInt(coverage["corpus_replayable_slice_count"]) != 2 || asInt(coverage["covered_corpus_slice_count"]) != 1 || asInt(coverage["uncovered_corpus_slice_count"]) != 2 {
		t.Fatalf("unexpected coverage counts: %+v", coverage)
	}
	uncovered := anyToMapSlice(coverage["uncovered_slices"])
	if asString(uncovered[0]["slice_id"]) != "validation" || asString(uncovered[1]["slice_id"]) != "browser-review" {
		t.Fatalf("unexpected uncovered slices: %+v", uncovered)
	}
}

func TestLoadShadowCorpusManifestCanPromoteReplayableSlices(t *testing.T) {
	root := t.TempDir()
	taskFile := filepath.Join(root, "task.json")
	writeShadowFixture(t, taskFile, map[string]any{
		"id": "replayable-task", "required_executor": "local", "entrypoint": "echo replay", "metadata": map[string]any{"scenario": "baseline"},
	})
	manifest := filepath.Join(root, "corpus.json")
	writeShadowFixture(t, manifest, map[string]any{
		"name": "replay-pack",
		"slices": []map[string]any{
			{"slice_id": "baseline", "title": "Baseline slice", "weight": 2, "task_file": "./task.json", "replay": true},
			{"slice_id": "metadata-only", "title": "Metadata only", "weight": 1, "task_shape": "executor:browser|scenario:review"},
		},
	})

	manifestMeta, replayEntries, corpusSlices, err := loadShadowCorpusManifestEntries(manifest, true)
	if err != nil {
		t.Fatalf("load corpus manifest: %v", err)
	}
	report := buildShadowMatrixReport(nil, nil, corpusSlices, manifestMeta)

	if len(replayEntries) != 1 {
		t.Fatalf("unexpected replay entries: %+v", replayEntries)
	}
	if asString(replayEntries[0]["source_kind"]) != "corpus" || asString(asMap(replayEntries[0]["corpus_slice"])["id"]) != "baseline" {
		t.Fatalf("unexpected replay metadata: %+v", replayEntries[0])
	}
	if asString(asMap(report["inputs"])["manifest_name"]) != "replay-pack" || asInt(asMap(report["corpus_coverage"])["uncovered_corpus_slice_count"]) != 2 {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func writeShadowFixture(t *testing.T, path string, payload any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	contents, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func newShadowTestServer(events []map[string]any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/healthz":
			_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		case r.Method == http.MethodPost && r.URL.Path == "/tasks":
			_ = json.NewEncoder(w).Encode(map[string]any{"accepted": true})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/tasks/"):
			_ = json.NewEncoder(w).Encode(map[string]any{"state": "succeeded"})
		case r.Method == http.MethodGet && r.URL.Path == "/events":
			_ = json.NewEncoder(w).Encode(map[string]any{"events": events})
		default:
			http.NotFound(w, r)
		}
	}))
}
