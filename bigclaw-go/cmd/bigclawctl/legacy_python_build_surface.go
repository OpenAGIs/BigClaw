package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"bigclaw-go/internal/legacyshim"
)

func runLegacyPythonBuildSurface(args []string) error {
	flags := flag.NewFlagSet("legacy-python build-surface", flag.ContinueOnError)
	repoRoot := flags.String("repo", "..", "repo root")
	asJSON := flags.Bool("json", false, "json")
	if helpText, err := parseFlagsWithHelp(flags, "usage: bigclawctl legacy-python build-surface [flags]", args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			_, _ = os.Stdout.WriteString(helpText)
			return nil
		}
		return err
	}
	report := legacyshim.PythonBuildSurfaceReport()
	if *asJSON {
		return emit(map[string]any{
			"repo":                absPath(*repoRoot),
			"surface_id":          report.SurfaceID,
			"status":              report.Status,
			"summary":             report.Summary,
			"replacement_command": report.ReplacementCommand,
			"active_assets":       report.ActiveAssets,
			"retired_assets":      report.RetiredAssets,
			"removal_conditions":  report.RemovalConditions,
			"validation_commands": report.ValidationCommands,
		}, true, 0)
	}
	_, _ = fmt.Fprintf(os.Stdout, "surface: %s\n", report.SurfaceID)
	_, _ = fmt.Fprintf(os.Stdout, "status: %s\n", report.Status)
	_, _ = fmt.Fprintf(os.Stdout, "repo: %s\n", absPath(*repoRoot))
	_, _ = fmt.Fprintf(os.Stdout, "summary: %s\n", report.Summary)
	_, _ = fmt.Fprintf(os.Stdout, "replacement: %s\n", report.ReplacementCommand)
	_, _ = os.Stdout.WriteString("active assets:\n")
	for _, asset := range report.ActiveAssets {
		_, _ = fmt.Fprintf(os.Stdout, "- %s [%s, %s] %s\n", asset.Path, asset.Kind, asset.Status, asset.Purpose)
	}
	_, _ = os.Stdout.WriteString("retired assets:\n")
	for _, asset := range report.RetiredAssets {
		_, _ = fmt.Fprintf(os.Stdout, "- %s [%s, %s] %s\n", asset.Path, asset.Kind, asset.Status, asset.Purpose)
	}
	_, _ = os.Stdout.WriteString("removal conditions:\n")
	for _, condition := range report.RemovalConditions {
		_, _ = fmt.Fprintf(os.Stdout, "- %s\n", condition)
	}
	_, _ = os.Stdout.WriteString("validation commands:\n")
	for _, command := range report.ValidationCommands {
		_, _ = fmt.Fprintf(os.Stdout, "- %s\n", command)
	}
	return nil
}
