package validationpolicy

type Decision struct {
	AllowedToClose bool     `json:"allowed_to_close"`
	Status         string   `json:"status"`
	Summary        string   `json:"summary"`
	MissingReports []string `json:"missing_reports,omitempty"`
}

var requiredReportArtifacts = []string{
	"task-run",
	"replay",
	"benchmark-suite",
}

func EnforceValidationReportPolicy(artifacts []string) Decision {
	existing := make(map[string]struct{}, len(artifacts))
	for _, artifact := range artifacts {
		existing[artifact] = struct{}{}
	}

	missing := make([]string, 0)
	for _, required := range requiredReportArtifacts {
		if _, ok := existing[required]; !ok {
			missing = append(missing, required)
		}
	}
	if len(missing) > 0 {
		return Decision{
			AllowedToClose: false,
			Status:         "blocked",
			Summary:        "validation report policy not satisfied",
			MissingReports: missing,
		}
	}
	return Decision{
		AllowedToClose: true,
		Status:         "ready",
		Summary:        "validation report policy satisfied",
	}
}
