package scheduler

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type FairnessStore interface {
	ShouldThrottle(now time.Time, tenantID string, rules RoutingRules) bool
	RecordAccepted(now time.Time, tenantID string, rules RoutingRules)
	Snapshot(now time.Time, rules RoutingRules) FairnessSnapshot
}

type FairnessRules struct {
	WindowSeconds               int `json:"window_seconds"`
	MaxRecentDecisionsPerTenant int `json:"max_recent_decisions_per_tenant"`
}

type FairnessTenantSnapshot struct {
	TenantID            string    `json:"tenant_id"`
	RecentAcceptedCount int       `json:"recent_accepted_count"`
	OldestAcceptedAt    time.Time `json:"oldest_accepted_at,omitempty"`
	LatestAcceptedAt    time.Time `json:"latest_accepted_at,omitempty"`
}

type FairnessSnapshot struct {
	Enabled                     bool                     `json:"enabled"`
	Shared                      bool                     `json:"shared"`
	Backend                     string                   `json:"backend"`
	Healthy                     bool                     `json:"healthy,omitempty"`
	Endpoint                    string                   `json:"endpoint,omitempty"`
	LastError                   string                   `json:"last_error,omitempty"`
	WindowSeconds               int                      `json:"window_seconds"`
	MaxRecentDecisionsPerTenant int                      `json:"max_recent_decisions_per_tenant"`
	ActiveTenants               int                      `json:"active_tenants"`
	Tenants                     []FairnessTenantSnapshot `json:"tenants"`
}

type fairnessTracker struct {
	mu      sync.Mutex
	history map[string][]time.Time
}

func NewFairnessStore(path string) (FairnessStore, error) {
	return NewFairnessStoreWithRemote(path, "", "")
}

func NewFairnessStoreWithRemote(path, remoteURL, bearerToken string) (FairnessStore, error) {
	if strings.TrimSpace(remoteURL) != "" {
		return NewHTTPFairnessStore(remoteURL, bearerToken)
	}
	if strings.TrimSpace(path) == "" {
		return newFairnessTracker(), nil
	}
	return NewSQLiteFairnessStore(path)
}

func newFairnessTracker() *fairnessTracker {
	return &fairnessTracker{history: make(map[string][]time.Time)}
}

func fairnessEnabled(rules RoutingRules) bool {
	return rules.Fairness.WindowSeconds > 0 && rules.Fairness.MaxRecentDecisionsPerTenant > 0
}

func fairnessBaseSnapshot(rules RoutingRules, backend string, shared bool) FairnessSnapshot {
	return FairnessSnapshot{
		Enabled:                     fairnessEnabled(rules),
		Shared:                      shared,
		Backend:                     backend,
		Healthy:                     true,
		WindowSeconds:               rules.Fairness.WindowSeconds,
		MaxRecentDecisionsPerTenant: rules.Fairness.MaxRecentDecisionsPerTenant,
	}
}

func (t *fairnessTracker) ShouldThrottle(now time.Time, tenantID string, rules RoutingRules) bool {
	if t == nil || !fairnessEnabled(rules) {
		return false
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return false
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pruneLocked(now, rules)
	recent := t.history[tenantID]
	if len(recent) < rules.Fairness.MaxRecentDecisionsPerTenant {
		return false
	}
	for otherTenant, timestamps := range t.history {
		if otherTenant == tenantID || len(timestamps) == 0 {
			continue
		}
		return true
	}
	return false
}

func (t *fairnessTracker) RecordAccepted(now time.Time, tenantID string, rules RoutingRules) {
	if t == nil || !fairnessEnabled(rules) {
		return
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pruneLocked(now, rules)
	t.history[tenantID] = append(t.history[tenantID], now)
}

func (t *fairnessTracker) Snapshot(now time.Time, rules RoutingRules) FairnessSnapshot {
	snapshot := fairnessBaseSnapshot(rules, "memory", false)
	if t == nil {
		return snapshot
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	t.pruneLocked(now, rules)
	for tenantID, timestamps := range t.history {
		if len(timestamps) == 0 {
			continue
		}
		entry := FairnessTenantSnapshot{
			TenantID:            tenantID,
			RecentAcceptedCount: len(timestamps),
			OldestAcceptedAt:    timestamps[0],
			LatestAcceptedAt:    timestamps[len(timestamps)-1],
		}
		snapshot.Tenants = append(snapshot.Tenants, entry)
	}
	snapshot.ActiveTenants = len(snapshot.Tenants)
	sort.SliceStable(snapshot.Tenants, func(i, j int) bool {
		if snapshot.Tenants[i].RecentAcceptedCount == snapshot.Tenants[j].RecentAcceptedCount {
			return snapshot.Tenants[i].TenantID < snapshot.Tenants[j].TenantID
		}
		return snapshot.Tenants[i].RecentAcceptedCount > snapshot.Tenants[j].RecentAcceptedCount
	})
	return snapshot
}

func (t *fairnessTracker) pruneLocked(now time.Time, rules RoutingRules) {
	if !fairnessEnabled(rules) {
		t.history = make(map[string][]time.Time)
		return
	}
	cutoff := now.Add(-time.Duration(rules.Fairness.WindowSeconds) * time.Second)
	for tenantID, timestamps := range t.history {
		kept := timestamps[:0]
		for _, timestamp := range timestamps {
			if timestamp.Before(cutoff) {
				continue
			}
			kept = append(kept, timestamp)
		}
		if len(kept) == 0 {
			delete(t.history, tenantID)
			continue
		}
		copied := append([]time.Time(nil), kept...)
		t.history[tenantID] = copied
	}
}

func fairnessThrottleReason(tenantID string, rules RoutingRules) string {
	return fmt.Sprintf("fairness window throttled tenant %s after %d recent accepted decisions in %ds window", tenantID, rules.Fairness.MaxRecentDecisionsPerTenant, rules.Fairness.WindowSeconds)
}
