package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/repository"
)

const IdempotencyKeyHeader = "Idempotency-Key"

func RequireIdempotencyKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(IdempotencyKeyHeader) == "" {
			http.Error(w, "idempotency key is required", http.StatusPreconditionRequired)
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

func Idempotency(repo repository.IdempotencyRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if repo == nil {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get(IdempotencyKeyHeader)
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
				http.Error(w, "invalid request body", http.StatusBadRequest)
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
				http.Error(w, "idempotency check failed", http.StatusInternalServerError)
				return
			}
			if !created {
				existing, err := repo.Get(r.Context(), key, claims.UserID, r.Method, r.URL.Path)
				if err != nil {
					if errors.Is(err, repository.ErrNotFound) {
						http.Error(w, "idempotency key conflict", http.StatusConflict)
						return
					}
					http.Error(w, "idempotency check failed", http.StatusInternalServerError)
					return
				}
				if existing.RequestHash != requestHash {
					http.Error(w, "idempotency key reused with different request", http.StatusConflict)
					return
				}
				if existing.StatusCode == nil {
					http.Error(w, "idempotency request is still in progress", http.StatusConflict)
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
				http.Error(w, "idempotency completion failed", http.StatusInternalServerError)
				return
			}
			rec.flushTo(w)
		})
	}
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
