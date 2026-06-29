package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

const (
	IdempotencyKeyHeader    = "Idempotency-Key"
	maxIdempotencyKeyLength = 255
)

const idempotencyCompletionUnknownMessage = "The operation may have completed, but idempotency state could not be persisted. Retry later with the same Idempotency-Key. Do not retry with a new key."

func RequireIdempotencyKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, ok := normalizeIdempotencyKey(w, r)
		if !ok {
			return
		}
		if key == "" {
			writeJSONError(w, http.StatusPreconditionRequired, "idempotency_key_required", "Idempotency key is required", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func MutationOnly(middleware func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if middleware == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost, http.MethodPatch, http.MethodDelete:
				middleware(next).ServeHTTP(w, r)
			default:
				next.ServeHTTP(w, r)
			}
		})
	}
}

type IdempotencyStore interface {
	Get(ctx context.Context, key, userID, method, path string) (*models.IdempotencyRecord, error)
	CreatePending(ctx context.Context, record *models.IdempotencyRecord) (bool, error)
	Complete(ctx context.Context, key, userID, method, path string, statusCode int, responseBody []byte) error
}

func Idempotency(repo IdempotencyStore) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if repo == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key, ok := normalizeIdempotencyKey(w, r)
			if !ok {
				return
			}
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			claims, ok := ClaimsFromContext(r.Context())
			if !ok || claims.UserID == "" {
				next.ServeHTTP(w, r)
				return
			}

			body, err := io.ReadAll(r.Body)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
				return
			}
			_ = r.Body.Close()
			r.Body = io.NopCloser(bytes.NewReader(body))

			requestHash := hashRequestBody(body)
			record := &models.IdempotencyRecord{
				Key:         key,
				UserID:      claims.UserID,
				Method:      r.Method,
				Path:        r.URL.Path,
				RequestHash: requestHash,
				CreatedAt:   time.Now().UTC(),
				ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
			}

			created, err := repo.CreatePending(r.Context(), record)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "idempotency_check_failed", "Idempotency check failed", nil)
				return
			}
			if !created {
				existing, err := repo.Get(r.Context(), key, claims.UserID, r.Method, r.URL.Path)
				if err != nil {
					if errors.Is(err, repository.ErrNotFound) {
						writeJSONError(w, http.StatusConflict, "idempotency_key_conflict", "Idempotency key conflict", nil)
						return
					}
					writeJSONError(w, http.StatusInternalServerError, "idempotency_check_failed", "Idempotency check failed", nil)
					return
				}
				if existing.RequestHash != requestHash {
					writeJSONError(w, http.StatusConflict, "idempotency_key_reused", "Idempotency key reused with different request", nil)
					return
				}
				if existing.StatusCode == nil {
					writeJSONError(w, http.StatusConflict, "idempotency_in_progress", "Idempotency request is still in progress", nil)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(*existing.StatusCode)
				_, _ = w.Write(existing.ResponseBody)
				return
			}

			rec := newCaptureResponseWriter(w)
			next.ServeHTTP(rec, r)

			completeCtx := context.WithoutCancel(r.Context())
			if err := repo.Complete(completeCtx, key, claims.UserID, r.Method, r.URL.Path, rec.statusCode(), rec.body.Bytes()); err != nil {
				if rec.statusCode() >= http.StatusOK && rec.statusCode() < http.StatusMultipleChoices {
					writeIdempotencyCompletionUnknown(w)
					return
				}
				writeJSONError(w, http.StatusInternalServerError, "idempotency_completion_failed", "Idempotency completion failed", nil)
				return
			}
			rec.flushTo(w)
		})
	}
}

func normalizeIdempotencyKey(w http.ResponseWriter, r *http.Request) (string, bool) {
	key := strings.TrimSpace(r.Header.Get(IdempotencyKeyHeader))
	if key == "" {
		return "", true
	}
	if len(key) > maxIdempotencyKeyLength {
		writeJSONError(w, http.StatusBadRequest, "idempotency_key_too_long", "Idempotency key is too long", nil)
		return "", false
	}
	r.Header.Set(IdempotencyKeyHeader, key)
	return key, true
}

func writeIdempotencyCompletionUnknown(w http.ResponseWriter) {
	writeJSONError(w, http.StatusConflict, "idempotency_completion_unknown", idempotencyCompletionUnknownMessage, nil)
}

func hashRequestBody(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

type captureResponseWriter struct {
	status int
	header http.Header
	body   bytes.Buffer
}

func newCaptureResponseWriter(w http.ResponseWriter) *captureResponseWriter {
	return &captureResponseWriter{header: w.Header().Clone()}
}

func (w *captureResponseWriter) Header() http.Header {
	return w.header
}

func (w *captureResponseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
}

func (w *captureResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, _ := w.body.Write(body)
	return n, nil
}

func (w *captureResponseWriter) statusCode() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (w *captureResponseWriter) flushTo(dst http.ResponseWriter) {
	copyHeaders(dst.Header(), w.header)
	dst.WriteHeader(w.statusCode())
	_, _ = dst.Write(w.body.Bytes())
}

func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		dst.Del(key)
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}
