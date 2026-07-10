package middleware

import (
	"expvar"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

var (
	httpRequestsTotal    = expvar.NewMap("capitalflow_http_requests_total")
	httpErrorsTotal      = expvar.NewMap("capitalflow_http_errors_total")
	httpDurationMS       = expvar.NewMap("capitalflow_http_request_duration_ms_total")
	httpRequestsInFlight = expvar.NewInt("capitalflow_http_requests_in_flight")
)

type metricsStatusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *metricsStatusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// RequestMetrics records bounded-cardinality request, error, latency and in-flight metrics.
func RequestMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		httpRequestsInFlight.Add(1)
		defer httpRequestsInFlight.Add(-1)

		recorder := &metricsStatusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)

		route := chi.RouteContext(r.Context()).RoutePattern()
		if route == "" {
			route = "unmatched"
		}
		statusClass := strconv.Itoa(recorder.status/100) + "xx"
		key := r.Method + " " + route + " " + statusClass
		httpRequestsTotal.Add(key, 1)
		httpDurationMS.Add(key, time.Since(started).Milliseconds())
		if recorder.status >= http.StatusBadRequest {
			httpErrorsTotal.Add(key, 1)
		}
	})
}
