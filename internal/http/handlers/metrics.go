package handlers

import (
	"encoding/json"
	"expvar"
	"net/http"
)

var allowedExpvarMetrics = map[string]struct{}{
	"capitalflow_auth_events_total": {},
}

func (h *Handler) metrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	metrics := make(map[string]any)
	for name := range allowedExpvarMetrics {
		value := expvar.Get(name)
		if value == nil {
			continue
		}
		metrics[name] = json.RawMessage(value.String())
	}

	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		http.Error(w, "encode metrics", http.StatusInternalServerError)
	}
}
