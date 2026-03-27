package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type automationShadowMatrixOptions struct {
	PrimaryBaseURL       string
	ShadowBaseURL        string
	TaskFiles            []string
	CorpusManifest       string
	ReplayCorpusSlices   bool
	TimeoutSeconds       int
	HealthTimeoutSeconds int
	ReportPath           string
}

type shadowExecutionEntry struct {
	Task        map[string]any `json:"task"`
	SourceKind  string         `json:"source_kind"`
	SourceFile  string         `json:"source_file"`
	TaskShape   string         `json:"task_shape"`
	CorpusSlice any            `json:"corpus_slice,omitempty"`
}

func runAutomationShadowMatrixCommand(args []string) error {
	flags := flag.NewFlagSet("automation migration shadow-matrix", flag.ContinueOnError)
	primary := flags.String("primary", "", "primary base URL")
	shadow := flags.String("shadow", "", "shadow base URL")
	var taskFiles multiStringFlag
	flags.Var(&taskFiles, "task-file", "task JSON file; repeatable")
	corpusManifest := flags.String("corpus-manifest", "", "corpus manifest path")
	replayCorpusSlices := flags.Bool("replay-corpus-slices", false, "replay approved corpus slices")
	timeoutSeconds := flags.Int("timeout-seconds", 180, "task timeout seconds")
	healthTimeoutSeconds := flags.Int("health-timeout-seconds", 60, "health wait timeout seconds")
	reportPath := flags.String("report-path", "", "report path")
	asJSON := flags.Bool("json", true, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl automation migration shadow-matrix [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	if trim(*primary) == "" || trim(*shadow) == "" {
		return errors.New("--primary and --shadow are required")
	}
	if len(taskFiles.values) == 0 && trim(*corpusManifest) == "" {
		return errors.New("at least one --task-file or --corpus-manifest must be provided")
	}
	report, exitCode, err := automationShadowMatrix(automationShadowMatrixOptions{
		PrimaryBaseURL:       trim(*primary),
		ShadowBaseURL:        trim(*shadow),
		TaskFiles:            taskFiles.values,
		CorpusManifest:       trim(*corpusManifest),
		ReplayCorpusSlices:   *replayCorpusSlices,
		TimeoutSeconds:       *timeoutSeconds,
		HealthTimeoutSeconds: *healthTimeoutSeconds,
		ReportPath:           trim(*reportPath),
	})
	if report != nil {
		return emit(report, *asJSON, exitCode)
	}
	return err
}

func automationShadowMatrix(opts automationShadowMatrixOptions) (map[string]any, int, error) {
	fixtureEntries, err := loadFixtureEntries(opts.TaskFiles)
	if err != nil {
		return nil, 0, err
	}
	var manifestMeta map[string]any
	var corpusSlices []map[string]any
	var replayEntries []shadowExecutionEntry
	if opts.CorpusManifest != "" {
		manifestMeta, replayEntries, corpusSlices, err = loadCorpusManifestEntries(opts.CorpusManifest, opts.ReplayCorpusSlices)
		if err != nil {
			return nil, 0, err
		}
	}
	executionEntries := append([]shadowExecutionEntry{}, fixtureEntries...)
	executionEntries = append(executionEntries, replayEntries...)
	results := make([]any, 0, len(executionEntries))
	matchedAll := true
	for index, entry := range executionEntries {
		task := cloneMap(entry.Task)
		baseID := automationFirstText(task["id"])
		if baseID == "" {
			baseID = fmt.Sprintf("matrix-task-%d", index+1)
		}
		task["id"] = fmt.Sprintf("%s-m%d", baseID, index+1)
		taskFile, err := writeTempTaskFile(task)
		if err != nil {
			return nil, 0, err
		}
		report, _, err := automationShadowCompare(automationShadowCompareOptions{
			PrimaryBaseURL:       opts.PrimaryBaseURL,
			ShadowBaseURL:        opts.ShadowBaseURL,
			TaskFile:             taskFile,
			TimeoutSeconds:       opts.TimeoutSeconds,
			HealthTimeoutSeconds: opts.HealthTimeoutSeconds,
		})
		_ = os.Remove(taskFile)
		if err != nil {
			return nil, 0, err
		}
		item := structToMap(report)
		item["source_file"] = entry.SourceFile
		item["source_kind"] = entry.SourceKind
		item["task_shape"] = entry.TaskShape
		if entry.CorpusSlice != nil {
			item["corpus_slice"] = entry.CorpusSlice
		}
		results = append(results, item)
		diff, _ := item["diff"].(map[string]any)
		if !automationBool(diff["state_equal"]) || !automationBool(diff["event_types_equal"]) {
			matchedAll = false
		}
	}
	report := buildShadowMatrixReport(results, fixtureEntries, corpusSlices, manifestMeta)
	if err := automationWriteReport(".", opts.ReportPath, report); err != nil {
		return nil, 0, err
	}
	if matchedAll {
		return report, 0, nil
	}
	return report, 1, nil
}

func cloneMap(input map[string]any) map[string]any {
	body, _ := json.Marshal(input)
	output := map[string]any{}
	_ = json.Unmarshal(body, &output)
	return output
}

func writeTempTaskFile(task map[string]any) (string, error) {
	file, err := os.CreateTemp("", "shadow-matrix-task-*.json")
	if err != nil {
		return "", err
	}
	defer file.Close()
	body, err := json.Marshal(task)
	if err != nil {
		return "", err
	}
	if _, err := file.Write(body); err != nil {
		return "", err
	}
	return file.Name(), nil
}

func loadFixtureEntries(taskFiles []string) ([]shadowExecutionEntry, error) {
	entries := make([]shadowExecutionEntry, 0, len(taskFiles))
	for _, taskFile := range taskFiles {
		task, err := loadJSONMap(taskFile)
		if err != nil {
			return nil, err
		}
		entries = append(entries, shadowExecutionEntry{
			Task:       task,
			SourceKind: "fixture",
			SourceFile: taskFile,
			TaskShape:  deriveTaskShape(task),
		})
	}
	return entries, nil
}

func loadJSONMap(path string) (map[string]any, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func deriveTaskShape(task map[string]any) string {
	features := []string{}
	executor := automationFirstText(task["required_executor"], task["executor"])
	if executor == "" {
		executor = "default"
	}
	features = append(features, "executor:"+executor)
	if labels, ok := task["labels"].([]any); ok && len(labels) > 0 {
		items := make([]string, 0, len(labels))
		for _, item := range labels {
			items = append(items, automationFirstText(item))
		}
		sort.Strings(items)
		features = append(features, "labels:"+stringsJoin(items, ","))
	}
	if task["budget_cents"] != nil {
		features = append(features, "budgeted")
	}
	if task["acceptance_criteria"] != nil {
		features = append(features, "acceptance")
	}
	if task["validation_plan"] != nil {
		features = append(features, "validation-plan")
	}
	if metadata, ok := task["metadata"].(map[string]any); ok {
		if scenario := automationFirstText(metadata["scenario"]); scenario != "" {
			features = append(features, "scenario:"+scenario)
		}
	}
	return stringsJoin(features, "|")
}

func stringsJoin(items []string, sep string) string {
	result := ""
	for index, item := range items {
		if index > 0 {
			result += sep
		}
		result += item
	}
	return result
}

func loadCorpusManifestEntries(manifestPath string, replay bool) (map[string]any, []shadowExecutionEntry, []map[string]any, error) {
	manifest, err := loadJSONMap(manifestPath)
	if err != nil {
		return nil, nil, nil, err
	}
	rawSlices, ok := manifest["slices"].([]any)
	if !ok {
		return nil, nil, nil, errors.New("corpus manifest must contain a top-level slices array")
	}
	coverageSlices := make([]map[string]any, 0, len(rawSlices))
	replayEntries := []shadowExecutionEntry{}
	for index, raw := range rawSlices {
		sliceMap, _ := raw.(map[string]any)
		normalized, err := normalizeCorpusSlice(sliceMap, index+1, manifestPath)
		if err != nil {
			return nil, nil, nil, err
		}
		coverageSlices = append(coverageSlices, normalized)
		if replay && normalized["task"] != nil {
			task, _ := normalized["task"].(map[string]any)
			replayEntries = append(replayEntries, shadowExecutionEntry{
				Task:       task,
				SourceKind: "corpus",
				SourceFile: automationFirstText(normalized["source_file"]),
				TaskShape:  automationFirstText(normalized["task_shape"]),
				CorpusSlice: map[string]any{
					"id":     normalized["slice_id"],
					"title":  normalized["title"],
					"weight": normalized["weight"],
					"tags":   normalized["tags"],
				},
			})
		}
	}
	meta := map[string]any{
		"name":         automationCoalesce(manifest["name"], stringsTrimSuffix(filepath.Base(manifestPath), filepath.Ext(manifestPath))),
		"generated_at": manifest["generated_at"],
		"source_file":  manifestPath,
	}
	return meta, replayEntries, coverageSlices, nil
}

func stringsTrimSuffix(value, suffix string) string {
	if len(suffix) > 0 && len(value) >= len(suffix) && value[len(value)-len(suffix):] == suffix {
		return value[:len(value)-len(suffix)]
	}
	return value
}

func normalizeCorpusSlice(slice map[string]any, index int, manifestPath string) (map[string]any, error) {
	sliceID := automationFirstText(slice["slice_id"])
	if sliceID == "" {
		sliceID = fmt.Sprintf("corpus-slice-%d", index)
	}
	title := automationFirstText(slice["title"])
	if title == "" {
		title = sliceID
	}
	weight := automationInt(slice["weight"])
	if weight == 0 {
		weight = 1
	}
	tags := []any{}
	if rawTags, ok := slice["tags"].([]any); ok {
		for _, tag := range rawTags {
			tags = append(tags, automationFirstText(tag))
		}
	}
	var task map[string]any
	sourceFile := ""
	if rawTaskFile := automationFirstText(slice["task_file"]); rawTaskFile != "" {
		resolved := resolveManifestTaskFile(manifestPath, rawTaskFile)
		loaded, err := loadJSONMap(resolved)
		if err != nil {
			return nil, err
		}
		task = loaded
		sourceFile = rawTaskFile
	} else if rawTask, ok := slice["task"].(map[string]any); ok {
		task = cloneMap(rawTask)
		sourceFile = fmt.Sprintf("%s#%s", filepath.Base(manifestPath), sliceID)
	}
	taskShape := automationFirstText(slice["task_shape"])
	if taskShape == "" && task != nil {
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
		"replay":      automationBool(slice["replay"]),
		"notes":       automationFirstText(slice["notes"]),
	}, nil
}

func resolveManifestTaskFile(manifestPath, taskFile string) string {
	if filepath.IsAbs(taskFile) {
		return taskFile
	}
	return filepath.Join(filepath.Dir(manifestPath), taskFile)
}

func buildShadowMatrixReport(results []any, fixtureEntries []shadowExecutionEntry, corpusSlices []map[string]any, manifestMeta map[string]any) map[string]any {
	matched := 0
	for _, item := range results {
		entry, _ := item.(map[string]any)
		diff, _ := entry["diff"].(map[string]any)
		if automationBool(diff["state_equal"]) && automationBool(diff["event_types_equal"]) {
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
			"manifest_name":      manifestMeta["name"],
		},
		"results": results,
	}
	if len(corpusSlices) > 0 && len(manifestMeta) > 0 {
		report["corpus_coverage"] = buildCorpusCoverage(fixtureEntries, corpusSlices, manifestMeta)
	}
	return report
}

func buildCorpusCoverage(fixtureEntries []shadowExecutionEntry, corpusSlices []map[string]any, manifestMeta map[string]any) map[string]any {
	fixtureByShape := map[string][]shadowExecutionEntry{}
	for _, entry := range fixtureEntries {
		fixtureByShape[entry.TaskShape] = append(fixtureByShape[entry.TaskShape], entry)
	}
	type aggregate struct {
		SliceCount           int
		ReplayableSliceCount int
		CorpusWeight         int
		SliceIDs             []any
		Titles               []any
	}
	corpusByShape := map[string]*aggregate{}
	for _, slice := range corpusSlices {
		shape := automationFirstText(slice["task_shape"])
		if corpusByShape[shape] == nil {
			corpusByShape[shape] = &aggregate{}
		}
		agg := corpusByShape[shape]
		agg.SliceCount++
		if slice["task"] != nil {
			agg.ReplayableSliceCount++
		}
		agg.CorpusWeight += automationInt(slice["weight"])
		agg.SliceIDs = append(agg.SliceIDs, slice["slice_id"])
		agg.Titles = append(agg.Titles, slice["title"])
	}
	shapes := make([]string, 0, len(corpusByShape))
	for shape := range corpusByShape {
		shapes = append(shapes, shape)
	}
	sort.Slice(shapes, func(i, j int) bool {
		left := corpusByShape[shapes[i]]
		right := corpusByShape[shapes[j]]
		if left.CorpusWeight == right.CorpusWeight {
			return shapes[i] < shapes[j]
		}
		return left.CorpusWeight > right.CorpusWeight
	})
	shapeScorecard := []any{}
	for _, shape := range shapes {
		agg := corpusByShape[shape]
		fixtures := fixtureByShape[shape]
		fixtureSources := []any{}
		for _, entry := range fixtures {
			fixtureSources = append(fixtureSources, entry.SourceFile)
		}
		shapeScorecard = append(shapeScorecard, map[string]any{
			"task_shape":             shape,
			"fixture_task_count":     len(fixtures),
			"fixture_sources":        fixtureSources,
			"corpus_slice_count":     agg.SliceCount,
			"replayable_slice_count": agg.ReplayableSliceCount,
			"corpus_weight":          agg.CorpusWeight,
			"corpus_slice_ids":       agg.SliceIDs,
			"corpus_titles":          agg.Titles,
			"covered_by_fixture":     len(fixtures) > 0,
		})
	}
	uncovered := []any{}
	for _, slice := range corpusSlices {
		if len(fixtureByShape[automationFirstText(slice["task_shape"])]) > 0 {
			continue
		}
		uncovered = append(uncovered, map[string]any{
			"slice_id":    slice["slice_id"],
			"title":       slice["title"],
			"task_shape":  slice["task_shape"],
			"weight":      slice["weight"],
			"replayable":  slice["task"] != nil,
			"source_file": slice["source_file"],
			"tags":        slice["tags"],
			"notes":       slice["notes"],
		})
	}
	return map[string]any{
		"manifest_name":                 manifestMeta["name"],
		"manifest_source_file":          manifestMeta["source_file"],
		"generated_at":                  manifestMeta["generated_at"],
		"fixture_task_count":            len(fixtureEntries),
		"corpus_slice_count":            len(corpusSlices),
		"corpus_replayable_slice_count": countReplayableSlices(corpusSlices),
		"covered_corpus_slice_count":    len(corpusSlices) - len(uncovered),
		"uncovered_corpus_slice_count":  len(uncovered),
		"shape_scorecard":               shapeScorecard,
		"uncovered_slices":              uncovered,
	}
}

func countReplayableSlices(corpusSlices []map[string]any) int {
	count := 0
	for _, slice := range corpusSlices {
		if slice["task"] != nil {
			count++
		}
	}
	return count
}
