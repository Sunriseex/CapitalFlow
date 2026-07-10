package handlers

import (
	"encoding/json"
	"expvar"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var allowedExpvarMetrics = map[string]struct{}{
	"capitalflow_auth_events_total":              {},
	"capitalflow_http_requests_total":            {},
	"capitalflow_http_errors_total":              {},
	"capitalflow_http_request_duration_ms_total": {},
	"capitalflow_http_requests_in_flight":        {},
}

type DBPoolMetrics struct {
	AcquiredConnections int32 `json:"acquired_connections"`
	IdleConnections     int32 `json:"idle_connections"`
	TotalConnections    int32 `json:"total_connections"`
	MaxConnections      int32 `json:"max_connections"`
	EmptyAcquires       int64 `json:"empty_acquires_total"`
	CanceledAcquires    int64 `json:"canceled_acquires_total"`
	AcquireDurationMS   int64 `json:"acquire_duration_ms_total"`
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
	if h.dbPoolMetrics != nil {
		metrics["capitalflow_db_pool"] = h.dbPoolMetrics()
	}
	if h.operationsMetricsDir != "" {
		metrics["capitalflow_operations"] = operationMetrics(h.operationsMetricsDir)
	}

	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		http.Error(w, "encode metrics", http.StatusInternalServerError)
	}
}

func operationMetrics(directory string) map[string]any {
	return map[string]any{
		"backup":   schedulerMetrics(directory, "backup"),
		"interest": schedulerMetrics(directory, "interest"),
	}
}

func schedulerMetrics(directory, scheduler string) map[string]any {
	prefix := "capitalflow-" + scheduler + "-scheduler."
	return map[string]any{
		"heartbeat_age_seconds":    fileAgeSeconds(filepath.Join(directory, prefix+"heartbeat")),
		"last_success_age_seconds": fileAgeSeconds(filepath.Join(directory, prefix+"last-success")),
		"status":                   statusValue(filepath.Join(directory, prefix+"status")),
	}
}

func fileAgeSeconds(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return -1
	}
	age := time.Since(info.ModTime()).Seconds()
	if age < 0 {
		return 0
	}
	return int64(age)
}

func statusValue(path string) string {
	contents, err := os.ReadFile(path) // #nosec G304 -- path is built from an operator-configured metrics directory and fixed names.
	if err != nil {
		return "unknown"
	}
	firstLine, _, _ := strings.Cut(string(contents), "\n")
	status, found := strings.CutPrefix(firstLine, "status=")
	if !found || status == "" {
		return "unknown"
	}
	return status
}
