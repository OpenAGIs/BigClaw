package scheduler

import (
	"encoding/json"
	"net/http"
)

func NewFairnessServiceHandler(store FairnessStore) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeFairnessJSON(w, http.StatusOK, map[string]any{"ok": true})
	})
	mux.HandleFunc("/throttle", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request fairnessThrottleRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeFairnessJSON(w, http.StatusOK, fairnessThrottleResponse{ShouldThrottle: store.ShouldThrottle(request.Now, request.TenantID, request.Rules)})
	})
	mux.HandleFunc("/record", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request fairnessRecordRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		store.RecordAccepted(request.Now, request.TenantID, request.Rules)
		writeFairnessJSON(w, http.StatusOK, map[string]any{"recorded": true})
	})
	mux.HandleFunc("/snapshot", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var request fairnessSnapshotRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeFairnessJSON(w, http.StatusOK, store.Snapshot(request.Now, request.Rules))
	})
	return mux
}

func writeFairnessJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
