package scheduler

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"bigclaw-go/internal/domain"
)

type RoutingRules struct {
	DefaultExecutor         domain.ExecutorKind            `json:"default_executor"`
	HighRiskExecutor        domain.ExecutorKind            `json:"high_risk_executor"`
	ToolExecutors           map[string]domain.ExecutorKind `json:"tool_executors"`
	UrgentPriorityThreshold int                            `json:"urgent_priority_threshold"`
	Fairness                FairnessRules                  `json:"fairness"`
}

type PolicyStore struct {
	mu    sync.RWMutex
	path  string
	rules RoutingRules
}

type routingRulesFile struct {
	DefaultExecutor         string             `json:"default_executor,omitempty"`
	HighRiskExecutor        string             `json:"high_risk_executor,omitempty"`
	ToolExecutors           map[string]string  `json:"tool_executors,omitempty"`
	UrgentPriorityThreshold *int               `json:"urgent_priority_threshold,omitempty"`
	Fairness                *fairnessRulesFile `json:"fairness,omitempty"`
}

type fairnessRulesFile struct {
	WindowSeconds               *int `json:"window_seconds,omitempty"`
	MaxRecentDecisionsPerTenant *int `json:"max_recent_decisions_per_tenant,omitempty"`
}

func DefaultRoutingRules() RoutingRules {
	return RoutingRules{
		DefaultExecutor:         domain.ExecutorLocal,
		HighRiskExecutor:        domain.ExecutorKubernetes,
		ToolExecutors:           map[string]domain.ExecutorKind{"browser": domain.ExecutorKubernetes, "gpu": domain.ExecutorRay},
		UrgentPriorityThreshold: 1,
	}
}

func NewDefaultPolicyStore() *PolicyStore {
	return &PolicyStore{rules: DefaultRoutingRules()}
}

func NewPolicyStore(path string) (*PolicyStore, error) {
	store := NewDefaultPolicyStore()
	store.path = strings.TrimSpace(path)
	if store.path == "" {
		return store, nil
	}
	if err := store.Reload(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *PolicyStore) Snapshot() RoutingRules {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneRoutingRules(s.rules)
}

func (s *PolicyStore) SourcePath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.path
}

func (s *PolicyStore) HasSource() bool {
	return strings.TrimSpace(s.SourcePath()) != ""
}

func (s *PolicyStore) Reload() error {
	path := s.SourcePath()
	if path == "" {
		return fmt.Errorf("scheduler policy path not configured")
	}
	rules, err := LoadRoutingRulesFile(path)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.rules = rules
	s.mu.Unlock()
	return nil
}

func LoadRoutingRulesFile(path string) (RoutingRules, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return RoutingRules{}, fmt.Errorf("read scheduler policy: %w", err)
	}
	var raw routingRulesFile
	if err := json.Unmarshal(content, &raw); err != nil {
		return RoutingRules{}, fmt.Errorf("decode scheduler policy: %w", err)
	}
	rules, err := raw.normalize()
	if err != nil {
		return RoutingRules{}, err
	}
	return rules, nil
}

func (raw routingRulesFile) normalize() (RoutingRules, error) {
	rules := DefaultRoutingRules()
	if raw.DefaultExecutor != "" {
		executor, err := parseExecutorKind(raw.DefaultExecutor)
		if err != nil {
			return RoutingRules{}, fmt.Errorf("default_executor: %w", err)
		}
		rules.DefaultExecutor = executor
	}
	if raw.HighRiskExecutor != "" {
		executor, err := parseExecutorKind(raw.HighRiskExecutor)
		if err != nil {
			return RoutingRules{}, fmt.Errorf("high_risk_executor: %w", err)
		}
		rules.HighRiskExecutor = executor
	}
	if raw.ToolExecutors != nil {
		for tool, executorName := range raw.ToolExecutors {
			normalizedTool := normalizeToolName(tool)
			if normalizedTool == "" {
				continue
			}
			executor, err := parseExecutorKind(executorName)
			if err != nil {
				return RoutingRules{}, fmt.Errorf("tool_executors[%s]: %w", normalizedTool, err)
			}
			rules.ToolExecutors[normalizedTool] = executor
		}
	}
	if raw.UrgentPriorityThreshold != nil {
		if *raw.UrgentPriorityThreshold <= 0 {
			return RoutingRules{}, fmt.Errorf("urgent_priority_threshold must be greater than zero")
		}
		rules.UrgentPriorityThreshold = *raw.UrgentPriorityThreshold
	}
	if raw.Fairness != nil {
		if raw.Fairness.WindowSeconds != nil {
			if *raw.Fairness.WindowSeconds < 0 {
				return RoutingRules{}, fmt.Errorf("fairness.window_seconds must be zero or greater")
			}
			rules.Fairness.WindowSeconds = *raw.Fairness.WindowSeconds
		}
		if raw.Fairness.MaxRecentDecisionsPerTenant != nil {
			if *raw.Fairness.MaxRecentDecisionsPerTenant < 0 {
				return RoutingRules{}, fmt.Errorf("fairness.max_recent_decisions_per_tenant must be zero or greater")
			}
			rules.Fairness.MaxRecentDecisionsPerTenant = *raw.Fairness.MaxRecentDecisionsPerTenant
		}
	}
	return rules, nil
}

func cloneRoutingRules(rules RoutingRules) RoutingRules {
	clone := rules
	clone.ToolExecutors = make(map[string]domain.ExecutorKind, len(rules.ToolExecutors))
	for key, value := range rules.ToolExecutors {
		clone.ToolExecutors[key] = value
	}
	return clone
}

func normalizeToolName(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func parseExecutorKind(raw string) (domain.ExecutorKind, error) {
	switch domain.ExecutorKind(strings.ToLower(strings.TrimSpace(raw))) {
	case domain.ExecutorLocal:
		return domain.ExecutorLocal, nil
	case domain.ExecutorKubernetes:
		return domain.ExecutorKubernetes, nil
	case domain.ExecutorRay:
		return domain.ExecutorRay, nil
	default:
		return "", fmt.Errorf("unsupported executor %q", raw)
	}
}
