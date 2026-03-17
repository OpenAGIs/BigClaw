package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/events"
)

const (
	coordinationLeaderEndpoint     = "/coordination/leader"
	coordinationLeaderGroupID      = "control-plane"
	coordinationLeaderSubscriberID = "scheduler-leader"
)

type coordinationLeaderAcquirePayload struct {
	ConsumerID string `json:"consumer_id"`
	TTLSeconds int64  `json:"ttl_seconds"`
}

type coordinationLeaderElectionSurface struct {
	Endpoint            string                  `json:"endpoint"`
	GroupID             string                  `json:"group_id"`
	SubscriberID        string                  `json:"subscriber_id"`
	ElectionModel       string                  `json:"election_model"`
	Status              string                  `json:"status"`
	LeaderPresent       bool                    `json:"leader_present"`
	Backend             string                  `json:"backend,omitempty"`
	Lease               *events.SubscriberLease `json:"lease,omitempty"`
	LeaseAgeSeconds     int64                   `json:"lease_age_seconds,omitempty"`
	RemainingTTLSeconds int64                   `json:"remaining_ttl_seconds,omitempty"`
	Notes               []string                `json:"notes,omitempty"`
	Error               string                  `json:"error,omitempty"`
}

func (s *Server) coordinationLeaderElectionPayload() any {
	surface := coordinationLeaderElectionSurface{
		Endpoint:      coordinationLeaderEndpoint,
		GroupID:       coordinationLeaderGroupID,
		SubscriberID:  coordinationLeaderSubscriberID,
		ElectionModel: "subscriber_lease",
		Status:        "unavailable",
		Notes: []string{
			"Uses the subscriber lease store as the current repo-native leader-election scaffold.",
			"Current hardening remains local/shared-store scoped rather than broker-backed or quorum-backed.",
			"Leader-election backend posture is also summarized in the leader_election_capability surface.",
		},
	}
	if s.SubscriberLeases == nil {
		surface.Error = "subscriber lease coordinator unavailable"
		return surface
	}
	if stringer, ok := s.SubscriberLeases.(fmt.Stringer); ok {
		surface.Backend = strings.TrimSpace(stringer.String())
	}
	if surface.Backend == "" {
		surface.Backend = "configured"
	}
	now := time.Now().UTC()
	if s.Now != nil {
		now = s.Now().UTC()
	}
	lease, ok := s.SubscriberLeases.Get(coordinationLeaderGroupID, coordinationLeaderSubscriberID)
	if !ok {
		surface.Status = "idle"
		return surface
	}
	surface.Lease = &lease
	surface.LeaseAgeSeconds = int64(now.Sub(lease.UpdatedAt).Round(time.Second) / time.Second)
	if surface.LeaseAgeSeconds < 0 {
		surface.LeaseAgeSeconds = 0
	}
	remaining := int64(lease.ExpiresAt.Sub(now).Round(time.Second) / time.Second)
	if remaining > 0 {
		surface.Status = "active"
		surface.LeaderPresent = true
		surface.RemainingTTLSeconds = remaining
		return surface
	}
	surface.Status = "expired"
	return surface
}

func (s *Server) handleCoordinationLeader(w http.ResponseWriter, r *http.Request) {
	if s.SubscriberLeases == nil {
		http.Error(w, "subscriber lease coordinator unavailable", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]any{"leader": s.coordinationLeaderElectionPayload(), "leader_election_capability": leaderElectionCapabilitySurfacePayload()})
	case http.MethodPost:
		var payload coordinationLeaderAcquirePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("decode leader election request: %v", err), http.StatusBadRequest)
			return
		}
		now := s.Now()
		previous, hadPrevious := s.SubscriberLeases.Get(coordinationLeaderGroupID, coordinationLeaderSubscriberID)
		lease, err := s.SubscriberLeases.Acquire(events.LeaseRequest{
			GroupID:      coordinationLeaderGroupID,
			SubscriberID: coordinationLeaderSubscriberID,
			ConsumerID:   payload.ConsumerID,
			TTL:          time.Duration(payload.TTLSeconds) * time.Second,
			Now:          now,
		})
		if err != nil {
			status := http.StatusBadRequest
			if err == events.ErrLeaseHeld {
				status = http.StatusConflict
				s.publishSubscriberLeaseEvent(domain.EventSubscriberLeaseRejected, now, lease, map[string]any{
					"attempted_consumer_id": payload.ConsumerID,
					"requested_ttl_seconds": payload.TTLSeconds,
					"reason":                err.Error(),
					"coordination_role":     "leader",
				})
			}
			writeJSON(w, status, map[string]any{"error": err.Error(), "lease": lease, "leader": s.coordinationLeaderElectionPayload(), "leader_election_capability": leaderElectionCapabilitySurfacePayload()})
			return
		}
		expiredTakeover := hadPrevious && previous.ConsumerID != "" && previous.ConsumerID != lease.ConsumerID && !previous.ExpiresAt.IsZero() && !now.Before(previous.ExpiresAt)
		if expiredTakeover {
			s.publishSubscriberLeaseEvent(domain.EventSubscriberLeaseExpired, now, previous, map[string]any{
				"expired_consumer_id":  previous.ConsumerID,
				"takeover_consumer_id": lease.ConsumerID,
				"coordination_role":    "leader",
			})
			s.publishSubscriberLeaseEvent(domain.EventSubscriberTakeoverSucceeded, now, lease, map[string]any{
				"previous_consumer_id":  previous.ConsumerID,
				"requested_ttl_seconds": payload.TTLSeconds,
				"coordination_role":     "leader",
			})
		} else {
			s.publishSubscriberLeaseEvent(domain.EventSubscriberLeaseAcquired, now, lease, map[string]any{
				"renewal":               hadPrevious && previous.ConsumerID == lease.ConsumerID,
				"requested_ttl_seconds": payload.TTLSeconds,
				"coordination_role":     "leader",
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"lease": lease, "leader": s.coordinationLeaderElectionPayload(), "leader_election_capability": leaderElectionCapabilitySurfacePayload()})
	case http.MethodDelete:
		var payload checkpointRequestPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, fmt.Sprintf("decode leader release: %v", err), http.StatusBadRequest)
			return
		}
		err := s.SubscriberLeases.Release(coordinationLeaderGroupID, coordinationLeaderSubscriberID, payload.ConsumerID, payload.LeaseToken, payload.LeaseEpoch)
		if err != nil {
			status := http.StatusConflict
			if err == events.ErrLeaseExpired {
				status = http.StatusGone
			}
			writeJSON(w, status, map[string]any{"error": err.Error(), "leader": s.coordinationLeaderElectionPayload(), "leader_election_capability": leaderElectionCapabilitySurfacePayload()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"released": true, "leader": s.coordinationLeaderElectionPayload(), "leader_election_capability": leaderElectionCapabilitySurfacePayload()})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
