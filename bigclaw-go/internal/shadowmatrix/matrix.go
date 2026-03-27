package shadowmatrix

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"bigclaw-go/internal/shadowcompare"
)

type CompareFunc func(shadowcompare.CompareOptions) (map[string]any, error)

type BuildOptions struct {
	PrimaryBaseURL     string
	ShadowBaseURL      string
	TaskFiles          []string
	CorpusManifestPath string
	ReplayCorpusSlices bool
	Timeout            time.Duration
	HealthTimeout      time.Duration
	Compare            CompareFunc
}

func BuildReport(opts BuildOptions) (map[string]any, error) {
	if len(opts.TaskFiles) == 0 && strings.TrimSpace(opts.CorpusManifestPath) == "" {
		return nil, fmt.Errorf("at least one task file or corpus manifest is required")
	}
	compareFn := opts.Compare
	if compareFn == nil {
		compareFn = shadowcompare.CompareTask
	}

	fixtureEntries, err := loadFixtureEntries(opts.TaskFiles)
	if err != nil {
		return nil, err
	}
	var manifestMeta map[string]any
	var replayEntries []map[string]any
	var corpusSlices []map[string]any
	if strings.TrimSpace(opts.CorpusManifestPath) != "" {
		meta, replay, coverage, err := loadCorpusManifestEntries(opts.CorpusManifestPath, opts.ReplayCorpusSlices)
		if err != nil {
			return nil, err
		}
		manifestMeta = meta
		replayEntries = replay
		corpusSlices = coverage
	}

	executionEntries := append(append([]map[string]any{}, fixtureEntries...), replayEntries...)
	results := make([]map[string]any, 0, len(executionEntries))
	for index, entry := range executionEntries {
		task := cloneMap(nestedMap(entry, "task"))
		baseID := stringValue(task["id"], fmt.Sprintf("matrix-task-%d", index+1))
		task["id"] = fmt.Sprintf("%s-m%d", baseID, index+1)
		result, err := compareFn(shadowcompare.CompareOptions{
			PrimaryBaseURL: opts.PrimaryBaseURL,
			ShadowBaseURL:  opts.ShadowBaseURL,
			Task:           task,
			Timeout:        opts.Timeout,
			HealthTimeout:  opts.HealthTimeout,
		})
		if err != nil {
			return nil, err
		}
		result["source_file"] = stringValue(entry["source_file"], "")
		if corpusSlice := nestedMap(entry, "corpus_slice"); len(corpusSlice) > 0 {
			result["corpus_slice"] = corpusSlice
		}
		results = append(results, result)
	}

	return buildReport(results, fixtureEntries, corpusSlices, manifestMeta), nil
}

func WriteReport(path string, report map[string]any) error {
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func ExitCode(report map[string]any) int {
	if matchedAll(mapSliceAt(report, "results")) {
		return 0
	}
	return 1
}

func deriveTaskShape(task map[string]any) string {
	features := make([]string, 0)
	executor := stringValue(task["required_executor"], "")
	if executor == "" {
		executor = stringValue(task["executor"], "default")
	}
	features = append(features, "executor:"+executor)
	labels := stringSliceAny(task["labels"])
	sort.Strings(labels)
	if len(labels) > 0 {
		features = append(features, "labels:"+strings.Join(labels, ","))
	}
	if _, ok := task["budget_cents"]; ok && task["budget_cents"] != nil {
		features = append(features, "budgeted")
	}
	if sliceLen(task["acceptance_criteria"]) > 0 {
		features = append(features, "acceptance")
	}
	if sliceLen(task["validation_plan"]) > 0 {
		features = append(features, "validation-plan")
	}
	metadata := nestedMap(task, "metadata")
	if scenario := stringValue(metadata["scenario"], ""); scenario != "" {
		features = append(features, "scenario:"+scenario)
	}
	return strings.Join(features, "|")
}

func makeExecutionEntry(task map[string]any, sourceKind, sourceFile, taskShape string, sliceData map[string]any) map[string]any {
	entry := map[string]any{
		"task":        cloneMap(task),
		"source_kind": sourceKind,
		"source_file": sourceFile,
		"task_shape":  taskShape,
	}
	nestedMap(entry, "task")["_source_file"] = sourceFile
	if len(sliceData) > 0 {
		entry["corpus_slice"] = map[string]any{
			"id":     sliceData["slice_id"],
			"title":  sliceData["title"],
			"weight": sliceData["weight"],
			"tags":   sliceData["tags"],
		}
	}
	return entry
}

func loadFixtureEntries(taskFiles []string) ([]map[string]any, error) {
	entries := make([]map[string]any, 0, len(taskFiles))
	for _, taskFile := range taskFiles {
		task, err := shadowcompare.LoadTask(taskFile)
		if err != nil {
			return nil, err
		}
		entries = append(entries, makeExecutionEntry(task, "fixture", normalizeFixturePath(taskFile), deriveTaskShape(task), nil))
	}
	return entries, nil
}

func loadCorpusManifestEntries(manifestPath string, replayCorpusSlices bool) (map[string]any, []map[string]any, []map[string]any, error) {
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, nil, nil, err
	}
	var manifest map[string]any
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, nil, nil, err
	}
	rawSlices, ok := manifest["slices"].([]any)
	if !ok {
		return nil, nil, nil, fmt.Errorf("corpus manifest must contain a top-level slices array")
	}

	coverageSlices := make([]map[string]any, 0, len(rawSlices))
	replayEntries := make([]map[string]any, 0)
	for index, rawSlice := range rawSlices {
		sliceData, ok := rawSlice.(map[string]any)
		if !ok {
			return nil, nil, nil, fmt.Errorf("invalid corpus slice at index %d", index)
		}
		normalized, err := normalizeCorpusSlice(sliceData, index+1, manifestPath)
		if err != nil {
			return nil, nil, nil, err
		}
		coverageSlices = append(coverageSlices, normalized)
		if replayCorpusSlices && len(nestedMap(normalized, "task")) > 0 {
			replayEntries = append(replayEntries, makeExecutionEntry(
				nestedMap(normalized, "task"),
				"corpus",
				stringValue(normalized["source_file"], ""),
				stringValue(normalized["task_shape"], ""),
				normalized,
			))
		}
	}

	manifestMeta := map[string]any{
		"name":         stringValue(manifest["name"], strings.TrimSuffix(filepath.Base(manifestPath), filepath.Ext(manifestPath))),
		"generated_at": manifest["generated_at"],
		"source_file":  normalizeCoveragePath(manifestPath),
	}
	return manifestMeta, replayEntries, coverageSlices, nil
}

func normalizeCorpusSlice(sliceData map[string]any, index int, manifestPath string) (map[string]any, error) {
	sliceID := stringValue(sliceData["slice_id"], fmt.Sprintf("corpus-slice-%d", index))
	title := stringValue(sliceData["title"], sliceID)
	weight := intValueWithDefault(sliceData["weight"], 1)
	tags := stringSliceAny(sliceData["tags"])
	var task map[string]any
	sourceFile := ""
	if taskFile := stringValue(sliceData["task_file"], ""); taskFile != "" {
		sourceFile = normalizeCoveragePath(resolveManifestTaskFile(manifestPath, taskFile))
		resolved := resolveManifestTaskFile(manifestPath, taskFile)
		loaded, err := shadowcompare.LoadTask(resolved)
		if err != nil {
			return nil, err
		}
		task = loaded
	} else if rawTask := nestedMap(sliceData, "task"); len(rawTask) > 0 {
		task = cloneMap(rawTask)
		sourceFile = filepath.Base(manifestPath) + "#" + sliceID
	}
	taskShape := stringValue(sliceData["task_shape"], "")
	if taskShape == "" && len(task) > 0 {
		taskShape = deriveTaskShape(task)
	}
	if taskShape == "" {
		return nil, fmt.Errorf("corpus slice %s must define task_shape or provide task/task_file", sliceID)
	}
	return map[string]any{
		"slice_id":    sliceID,
		"title":       title,
		"weight":      weight,
		"tags":        tags,
		"task_shape":  taskShape,
		"task":        task,
		"source_file": sourceFile,
		"replay":      boolValue(sliceData["replay"], false),
		"notes":       stringValue(sliceData["notes"], ""),
	}, nil
}

func resolveManifestTaskFile(manifestPath, taskFile string) string {
	path := taskFile
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(filepath.Dir(manifestPath), path)
}

func buildCorpusCoverage(fixtureEntries, corpusSlices []map[string]any, manifestMeta map[string]any) map[string]any {
	fixtureByShape := map[string][]map[string]any{}
	for _, entry := range fixtureEntries {
		shape := stringValue(entry["task_shape"], "")
		fixtureByShape[shape] = append(fixtureByShape[shape], entry)
	}

	corpusByShape := map[string]map[string]any{}
	for _, sliceData := range corpusSlices {
		shape := stringValue(sliceData["task_shape"], "")
		aggregate := corpusByShape[shape]
		if aggregate == nil {
			aggregate = map[string]any{
				"slice_count":            0,
				"replayable_slice_count": 0,
				"corpus_weight":          0,
				"slice_ids":              []string{},
				"titles":                 []string{},
			}
			corpusByShape[shape] = aggregate
		}
		aggregate["slice_count"] = intValue(aggregate["slice_count"]) + 1
		if len(nestedMap(sliceData, "task")) > 0 {
			aggregate["replayable_slice_count"] = intValue(aggregate["replayable_slice_count"]) + 1
		}
		aggregate["corpus_weight"] = intValue(aggregate["corpus_weight"]) + intValue(sliceData["weight"])
		aggregate["slice_ids"] = append(stringSliceAny(aggregate["slice_ids"]), stringValue(sliceData["slice_id"], ""))
		aggregate["titles"] = append(stringSliceAny(aggregate["titles"]), stringValue(sliceData["title"], ""))
	}

	shapes := make([]string, 0, len(corpusByShape))
	for shape := range corpusByShape {
		shapes = append(shapes, shape)
	}
	sort.Slice(shapes, func(i, j int) bool {
		leftWeight := intValue(corpusByShape[shapes[i]]["corpus_weight"])
		rightWeight := intValue(corpusByShape[shapes[j]]["corpus_weight"])
		if leftWeight != rightWeight {
			return leftWeight > rightWeight
		}
		return shapes[i] < shapes[j]
	})

	shapeScorecard := make([]map[string]any, 0, len(shapes))
	for _, shape := range shapes {
		aggregate := corpusByShape[shape]
		fixtures := fixtureByShape[shape]
		fixtureSources := make([]string, 0, len(fixtures))
		for _, entry := range fixtures {
			fixtureSources = append(fixtureSources, normalizeCoveragePath(stringValue(entry["source_file"], "")))
		}
		shapeScorecard = append(shapeScorecard, map[string]any{
			"task_shape":             shape,
			"fixture_task_count":     len(fixtures),
			"fixture_sources":        fixtureSources,
			"corpus_slice_count":     intValue(aggregate["slice_count"]),
			"replayable_slice_count": intValue(aggregate["replayable_slice_count"]),
			"corpus_weight":          intValue(aggregate["corpus_weight"]),
			"corpus_slice_ids":       stringSliceAny(aggregate["slice_ids"]),
			"corpus_titles":          stringSliceAny(aggregate["titles"]),
			"covered_by_fixture":     len(fixtures) > 0,
		})
	}

	uncovered := make([]map[string]any, 0)
	for _, sliceData := range corpusSlices {
		if len(fixtureByShape[stringValue(sliceData["task_shape"], "")]) > 0 {
			continue
		}
		uncovered = append(uncovered, map[string]any{
			"slice_id":    sliceData["slice_id"],
			"title":       sliceData["title"],
			"task_shape":  sliceData["task_shape"],
			"weight":      sliceData["weight"],
			"replayable":  len(nestedMap(sliceData, "task")) > 0,
			"source_file": valueOrNil(normalizeCoveragePath(stringValue(sliceData["source_file"], ""))),
			"tags":        stringSliceAny(sliceData["tags"]),
			"notes":       stringValue(sliceData["notes"], ""),
		})
	}

	return map[string]any{
		"manifest_name":                 manifestMeta["name"],
		"manifest_source_file":          manifestMeta["source_file"],
		"generated_at":                  manifestMeta["generated_at"],
		"fixture_task_count":            len(fixtureEntries),
		"corpus_slice_count":            len(corpusSlices),
		"corpus_replayable_slice_count": countReplayable(corpusSlices),
		"covered_corpus_slice_count":    len(corpusSlices) - len(uncovered),
		"uncovered_corpus_slice_count":  len(uncovered),
		"shape_scorecard":               shapeScorecard,
		"uncovered_slices":              uncovered,
	}
}

func buildReport(results, fixtureEntries, corpusSlices []map[string]any, manifestMeta map[string]any) map[string]any {
	matched := 0
	for _, item := range results {
		if matchedResult(item) {
			matched++
		}
	}
	report := map[string]any{
		"total":      len(results),
		"matched":    matched,
		"mismatched": len(results) - matched,
		"inputs": map[string]any{
			"fixture_task_count": len(fixtureEntries),
			"corpus_slice_count": len(corpusSlices),
			"manifest_name":      valueOrNil(manifestMeta["name"]),
		},
		"results": results,
	}
	if len(corpusSlices) > 0 && len(manifestMeta) > 0 {
		report["corpus_coverage"] = buildCorpusCoverage(fixtureEntries, corpusSlices, manifestMeta)
	}
	return report
}

func matchedAll(results []map[string]any) bool {
	for _, item := range results {
		if !matchedResult(item) {
			return false
		}
	}
	return true
}

func matchedResult(item map[string]any) bool {
	diff := nestedMap(item, "diff")
	return boolValue(diff["state_equal"], false) && boolValue(diff["event_types_equal"], false)
}

func countReplayable(corpusSlices []map[string]any) int {
	total := 0
	for _, sliceData := range corpusSlices {
		if len(nestedMap(sliceData, "task")) > 0 {
			total++
		}
	}
	return total
}

func valueOrNil(value any) any {
	if value == nil {
		return nil
	}
	if text, ok := value.(string); ok && text == "" {
		return nil
	}
	return value
}

func normalizeCoveragePath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	path = filepath.ToSlash(path)
	if index := strings.Index(path, "bigclaw-go/"); index >= 0 {
		return path[index:]
	}
	path = strings.TrimPrefix(path, "./")
	if strings.HasPrefix(path, "examples/") {
		return "bigclaw-go/" + path
	}
	return path
}

func normalizeFixturePath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	path = filepath.ToSlash(path)
	if index := strings.Index(path, "bigclaw-go/examples/"); index >= 0 {
		return "./examples/" + strings.TrimPrefix(path[index:], "bigclaw-go/examples/")
	}
	if strings.HasPrefix(path, "./examples/") {
		return path
	}
	return path
}

func sliceLen(value any) int {
	switch typed := value.(type) {
	case []any:
		return len(typed)
	case []string:
		return len(typed)
	default:
		return 0
	}
}

func cloneMap(source map[string]any) map[string]any {
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func nestedMap(source map[string]any, key string) map[string]any {
	value, _ := source[key].(map[string]any)
	if value == nil {
		return map[string]any{}
	}
	return value
}

func mapSliceAt(source map[string]any, key string) []map[string]any {
	raw, _ := source[key].([]any)
	items := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if mapped, ok := item.(map[string]any); ok {
			items = append(items, mapped)
		}
	}
	return items
}

func stringValue(value any, fallback string) string {
	text, ok := value.(string)
	if !ok {
		return fallback
	}
	return text
}

func stringSliceAny(value any) []string {
	switch typed := value.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				items = append(items, text)
			}
		}
		return items
	default:
		return nil
	}
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

func intValueWithDefault(value any, fallback int) int {
	if value == nil {
		return fallback
	}
	if result := intValue(value); result != 0 {
		return result
	}
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return fallback
	}
}

func boolValue(value any, fallback bool) bool {
	typed, ok := value.(bool)
	if ok {
		return typed
	}
	return fallback
}
