package scheduler

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

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
	mu        sync.RWMutex
	path      string
	shared    *SQLitePolicyStore
	rules     RoutingRules
	updatedAt time.Time
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
	return NewPolicyStoreWithSQLite(path, "")
}

func NewPolicyStoreWithSQLite(path string, sqlitePath string) (*PolicyStore, error) {
	store := NewDefaultPolicyStore()
	store.path = strings.TrimSpace(path)
	if strings.TrimSpace(sqlitePath) != "" {
		shared, err := NewSQLitePolicyStore(sqlitePath)
		if err != nil {
			return nil, err
		}
		store.shared = shared
		if err := store.bootstrapSharedState(); err != nil {
			_ = shared.Close()
			return nil, err
		}
		return store, nil
	}
	if store.path == "" {
		return store, nil
	}
	if err := store.Reload(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *PolicyStore) Close() error {
	if s == nil || s.shared == nil {
		return nil
	}
	return s.shared.Close()
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

func (s *PolicyStore) SharedPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.shared == nil {
		return ""
	}
	return s.shared.Path()
}

func (s *PolicyStore) Backend() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	switch {
	case s.shared != nil:
		return "sqlite"
	case s.path != "":
		return "file"
	default:
		return "memory"
	}
}

func (s *PolicyStore) Shared() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.shared != nil
}

func (s *PolicyStore) UpdatedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.updatedAt
}

func (s *PolicyStore) HasSource() bool {
	return strings.TrimSpace(s.SourcePath()) != "" || strings.TrimSpace(s.SharedPath()) != ""
}

func (s *PolicyStore) Reload() error {
	sourcePath := s.SourcePath()
	sharedPath := s.SharedPath()
	switch {
	case sharedPath != "" && sourcePath != "":
		rules, err := LoadRoutingRulesFile(sourcePath)
		if err != nil {
			return err
		}
		updatedAt, err := s.shared.Save(rules, "file:"+sourcePath)
		if err != nil {
			return err
		}
		s.setRules(rules, updatedAt)
		return nil
	case sharedPath != "":
		rules, updatedAt, ok, err := s.shared.Load()
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("shared scheduler policy store is empty")
		}
		s.setRules(rules, updatedAt)
		return nil
	case sourcePath != "":
		rules, err := LoadRoutingRulesFile(sourcePath)
		if err != nil {
			return err
		}
		s.setRules(rules, time.Now())
		return nil
	default:
		return fmt.Errorf("scheduler policy path not configured")
	}
}

func (s *PolicyStore) bootstrapSharedState() error {
	rules, updatedAt, ok, err := s.shared.Load()
	if err != nil {
		return err
	}
	if ok {
		s.setRules(rules, updatedAt)
		return nil
	}
	if strings.TrimSpace(s.path) != "" {
		rules, err = LoadRoutingRulesFile(s.path)
		if err != nil {
			return err
		}
	} else {
		rules = DefaultRoutingRules()
	}
	updatedAt, err = s.shared.Save(rules, sharedPolicySource(s.path))
	if err != nil {
		return err
	}
	s.setRules(rules, updatedAt)
	return nil
}

func (s *PolicyStore) setRules(rules RoutingRules, updatedAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules = cloneRoutingRules(rules)
	s.updatedAt = updatedAt
}

func sharedPolicySource(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "default"
	}
	return "file:" + path
}

func LoadRoutingRulesFile(path string) (RoutingRules, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return RoutingRules{}, fmt.Errorf("read scheduler policy: %w", err)
	}
	return LoadRoutingRulesJSON(content)
}

func LoadRoutingRulesJSON(content []byte) (RoutingRules, error) {
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
