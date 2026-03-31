package reporting

type ValidationReportDecision struct {
	AllowedToClose bool     `json:"allowed_to_close"`
	Status         string   `json:"status"`
	Summary        string   `json:"summary"`
	MissingReports []string `json:"missing_reports,omitempty"`
}

var RequiredReportArtifacts = []string{
	"task-run",
	"replay",
	"benchmark-suite",
}

func EnforceValidationReportPolicy(artifacts []string) ValidationReportDecision {
	existing := make(map[string]struct{}, len(artifacts))
	for _, artifact := range artifacts {
		existing[artifact] = struct{}{}
	}

	missing := make([]string, 0, len(RequiredReportArtifacts))
	for _, artifact := range RequiredReportArtifacts {
		if _, ok := existing[artifact]; !ok {
			missing = append(missing, artifact)
		}
	}
	if len(missing) > 0 {
		return ValidationReportDecision{
			AllowedToClose: false,
			Status:         "blocked",
			Summary:        "validation report policy not satisfied",
			MissingReports: missing,
		}
	}
	return ValidationReportDecision{
		AllowedToClose: true,
		Status:         "ready",
		Summary:        "validation report policy satisfied",
	}
}
