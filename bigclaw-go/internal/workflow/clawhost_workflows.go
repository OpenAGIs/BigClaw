package workflow

import "bigclaw-go/internal/domain"

const (
	ClawHostSkillsWorkflowName      = "clawhost-skills-channels-device-approval"
	clawHostWorkflowReportTemplate  = "docs/reports/{workflow}/{task_id}-{run_id}.json"
	clawHostWorkflowJournalTemplate = "docs/reports/{workflow}/{task_id}-{run_id}.journal.json"
)

func ClawHostSkillsWorkflowDefinition() Definition {
	return Definition{
		Name:                ClawHostSkillsWorkflowName,
		ReportPathTemplate:  clawHostWorkflowReportTemplate,
		JournalPathTemplate: clawHostWorkflowJournalTemplate,
		ValidationEvidence: []string{
			"docs/reports/clawhost-fleet-inventory-surface.json",
			"docs/reports/clawhost-tenant-policy-surface.json",
			"docs/reports/clawhost-proxy-admin-validation-lane.json",
		},
		Approvals: []string{
			"tenant-owner-review",
			"device-approval-review",
		},
		Steps: []Step{
			{
				Name:     "inventory-scope",
				Kind:     "inventory",
				Required: true,
				Metadata: map[string]any{
					"source_report": "docs/reports/clawhost-fleet-inventory-surface.json",
					"selection":     "tenant app and bot scope",
				},
			},
			{
				Name:     "policy-guardrails",
				Kind:     "policy",
				Required: true,
				Metadata: map[string]any{
					"source_report": "docs/reports/clawhost-tenant-policy-surface.json",
					"checks":        []string{"provider_defaults", "approval_mode", "rollout_guardrails"},
				},
			},
			{
				Name:     "stage-parallel-batch",
				Kind:     "batch",
				Required: true,
				Metadata: map[string]any{
					"max_parallelism":  3,
					"takeover_enabled": true,
					"actions":          []string{"skills_sync", "channel_connectivity_refresh", "device_auto_approval"},
				},
			},
			{
				Name:     "validate-im-channel",
				Kind:     "connectivity",
				Required: true,
				Metadata: map[string]any{
					"probe_types":   []string{"http", "websocket"},
					"source_report": "docs/reports/clawhost-proxy-admin-validation-lane.json",
				},
			},
			{
				Name:     "device-approval-gate",
				Kind:     "approval",
				Required: true,
				Metadata: map[string]any{
					"approval_refs":  []string{"tenant-owner-review", "device-approval-review"},
					"human_takeover": true,
				},
			},
			{
				Name:     "export-review-evidence",
				Kind:     "report",
				Required: true,
				Metadata: map[string]any{
					"artifacts": []string{"workflow_report", "approval_packet", "connectivity_summary"},
				},
			},
		},
	}
}

func ClawHostSkillsWorkflowTemplate() WorkflowTemplate {
	return WorkflowTemplate{
		TemplateID:  "clawhost-skills-workflow",
		Name:        "ClawHost Skills, Channels, and Device Approval",
		Version:     "v1",
		Description: "Batch skill sync, IM channel connectivity review, and device approval changes with takeover-aware execution.",
		Trigger:     WorkflowTriggerManual,
		DefaultRisk: domain.RiskHigh,
		Tags:        []string{"clawhost", "workflow", "parallel", "approval"},
		Active:      true,
		Steps: []WorkflowTemplateStep{
			{
				StepID:        "inventory-scope",
				Name:          "Scope bot inventory",
				Kind:          "inventory",
				RequiredTools: []string{"repo", "browser"},
				Metadata: map[string]any{
					"workflow_definition": ClawHostSkillsWorkflowName,
				},
			},
			{
				StepID:        "policy-guardrails",
				Name:          "Review provider and tenant guardrails",
				Kind:          "policy",
				RequiredTools: []string{"repo"},
				Approvals:     []string{"tenant-owner-review"},
			},
			{
				StepID:        "parallel-batch",
				Name:          "Stage parallel skill and channel batch",
				Kind:          "batch",
				RequiredTools: []string{"repo", "terminal"},
				Metadata: map[string]any{
					"max_parallelism": 3,
				},
			},
			{
				StepID:        "device-approval",
				Name:          "Approve device pairing and handoff",
				Kind:          "approval",
				RequiredTools: []string{"browser"},
				Approvals:     []string{"device-approval-review"},
			},
			{
				StepID:        "evidence-export",
				Name:          "Export reviewer evidence",
				Kind:          "report",
				RequiredTools: []string{"repo"},
			},
		},
	}
}
