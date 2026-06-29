package handlers

import (
	"net/http"
)

// HealthResponse reports basic service health and release version.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{
		Status:  "ok",
		Version: h.appVersion,
	})
}

func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	if err := h.app.Store.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database_unavailable", "Database is unavailable", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
