package events

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
)

type LogServiceStore interface {
	EventLog
	CheckpointStore
}

func NewEventLogServiceHandler(store LogServiceStore) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeEventLogJSON(w, http.StatusOK, map[string]any{"ok": true})
	})
	mux.HandleFunc("/record", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var event domain.Event
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := store.Write(r.Context(), event); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeEventLogJSON(w, http.StatusOK, map[string]any{"recorded": true})
	})

	mux.HandleFunc("/watermark", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		provider, ok := any(store).(RetentionWatermarkProvider)
		if !ok {
			http.Error(w, "retention watermark unavailable", http.StatusServiceUnavailable)
			return
		}
		watermark, err := provider.RetentionWatermark()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeEventLogJSON(w, http.StatusOK, map[string]any{"retention_watermark": watermark})
	})

	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		taskID := r.URL.Query().Get("task_id")
		traceID := r.URL.Query().Get("trace_id")
		afterID := strings.TrimSpace(r.URL.Query().Get("after_id"))
		history, err := queryLogStore(store, taskID, traceID, afterID, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeEventLogJSON(w, http.StatusOK, map[string]any{"events": history})
	})
	mux.HandleFunc("/checkpoints/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/checkpoints/")
		historyRequest := strings.HasSuffix(path, "/history")
		subscriberID := strings.TrimSuffix(path, "/history")
		subscriberID = strings.TrimSuffix(subscriberID, "/")
		if subscriberID == "" {
			http.Error(w, "missing subscriber id", http.StatusBadRequest)
			return
		}
		if historyRequest {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			historyProvider, ok := any(store).(CheckpointResetHistoryProvider)
			if !ok {
				http.Error(w, "checkpoint reset history unavailable", http.StatusServiceUnavailable)
				return
			}
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			history, err := historyProvider.CheckpointResetHistory(subscriberID, limit)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeEventLogJSON(w, http.StatusOK, map[string]any{"history": history})
			return
		}
		switch r.Method {
		case http.MethodGet:
			checkpoint, err := store.Checkpoint(subscriberID)
			if err != nil {
				if IsNoEventLog(err) {
					http.Error(w, "checkpoint not found", http.StatusNotFound)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeEventLogJSON(w, http.StatusOK, map[string]any{"checkpoint": checkpoint})
		case http.MethodPost:
			var request checkpointAckRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if request.AckedAt.IsZero() {
				request.AckedAt = time.Now().UTC()
			}
			checkpoint, err := store.Acknowledge(subscriberID, strings.TrimSpace(request.EventID), request.AckedAt)
			if err != nil {
				if IsNoEventLog(err) {
					http.Error(w, "event not found", http.StatusNotFound)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeEventLogJSON(w, http.StatusOK, map[string]any{"checkpoint": checkpoint})
		case http.MethodDelete:
			manager, ok := any(store).(CheckpointResetManager)
			if ok {
				var request CheckpointResetRequest
				if r.Body != nil {
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil && !errors.Is(err, io.EOF) {
						http.Error(w, err.Error(), http.StatusBadRequest)
						return
					}
				}
				record, err := manager.ResetCheckpointWithAudit(subscriberID, request)
				if err != nil {
					if IsNoEventLog(err) {
						http.Error(w, "checkpoint not found", http.StatusNotFound)
						return
					}
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				writeEventLogJSON(w, http.StatusOK, map[string]any{"subscriber_id": subscriberID, "reset": true, "reset_audit": record})
				return
			}
			resetter, ok := any(store).(CheckpointResetter)
			if !ok {
				http.Error(w, "checkpoint reset unavailable", http.StatusServiceUnavailable)
				return
			}
			if err := resetter.ResetCheckpoint(subscriberID); err != nil {
				if IsNoEventLog(err) {
					http.Error(w, "checkpoint not found", http.StatusNotFound)
					return
				}
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeEventLogJSON(w, http.StatusOK, map[string]any{"subscriber_id": subscriberID, "reset": true})
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	return mux
}

func queryLogStore(store EventLog, taskID, traceID, afterID string, limit int) ([]domain.Event, error) {
	if afterID != "" {
		switch {
		case taskID != "":
			return store.EventsByTaskAfter(taskID, afterID, limit)
		case traceID != "":
			return store.EventsByTraceAfter(traceID, afterID, limit)
		default:
			return store.ReplayAfter(afterID, limit)
		}
	}
	switch {
	case taskID != "":
		return store.EventsByTask(taskID, limit)
	case traceID != "":
		return store.EventsByTrace(traceID, limit)
	default:
		return store.Replay(limit)
	}
}

func writeEventLogJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
