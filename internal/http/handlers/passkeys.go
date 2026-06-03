package handlers

import (
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	appmiddleware "github.com/sunriseex/capitalflow/internal/http/middleware"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/services"
)

func (h *Handler) passkeyLoginOptions(w http.ResponseWriter, r *http.Request) {
	options, err := h.passkeyService().LoginOptions(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, options)
}

func (h *Handler) passkeyLoginVerify(w http.ResponseWriter, r *http.Request) {
	body, err := readPasskeyBody(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	session, err := h.passkeyService().VerifyLogin(r.Context(), body)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	h.setRefreshCookie(w, session)
	writeJSON(w, http.StatusOK, authResponse(session))
}

func (h *Handler) listPasskeys(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmiddleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
		return
	}
	credentials, err := h.passkeyService().List(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, passkeyCredentialsResponse(credentials))
}

func (h *Handler) passkeyRegistrationOptions(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmiddleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
		return
	}
	var req dto.PasskeyRegistrationOptionsRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
			return
		}
	}
	options, err := h.passkeyService().RegistrationOptions(r.Context(), services.PasskeyRegistrationOptionsRequest{
		UserID:   userID,
		Password: req.Password,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, options)
}

func (h *Handler) passkeyRegistrationVerify(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmiddleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
		return
	}
	body, err := readPasskeyBody(r)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	credential, err := h.passkeyService().VerifyRegistration(r.Context(), userID, body)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, passkeyCredentialResponse(credential))
}

func (h *Handler) renamePasskey(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmiddleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
		return
	}
	var req dto.PasskeyRenameRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}
	credential, err := h.passkeyService().Rename(r.Context(), services.PasskeyRenameRequest{
		UserID: userID,
		ID:     chi.URLParam(r, "id"),
		Name:   req.Name,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, passkeyCredentialResponse(credential))
}

func (h *Handler) deletePasskey(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmiddleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
		return
	}
	if err := h.passkeyService().Delete(r.Context(), services.PasskeyDeleteRequest{
		UserID: userID,
		ID:     chi.URLParam(r, "id"),
	}); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) passkeyService() *services.PasskeyService {
	authService := h.authService()
	service, err := services.NewPasskeyService(
		h.store.Users(),
		h.store.Passkeys(),
		authService,
		h.store.AuthAuditEvents(),
		services.WebAuthnConfig{
			RPDisplayName: h.webAuthnRPDisplayName,
			RPID:          h.webAuthnRPID,
			Origins:       h.webAuthnOrigins,
		},
	)
	if err != nil {
		panic(fmt.Sprintf("passkey service is not configured: %v", err))
	}
	return service
}

func passkeyCredentialsResponse(credentials []models.PasskeyCredential) dto.PasskeyCredentialsResponse {
	response := dto.PasskeyCredentialsResponse{Passkeys: make([]dto.PasskeyCredentialResponse, 0, len(credentials))}
	for i := range credentials {
		response.Passkeys = append(response.Passkeys, passkeyCredentialResponse(&credentials[i]))
	}
	return response
}

func passkeyCredentialResponse(credential *models.PasskeyCredential) dto.PasskeyCredentialResponse {
	return dto.PasskeyCredentialResponse{
		ID:             credential.ID,
		Name:           credential.Name,
		BackupEligible: credential.BackupEligible,
		BackupState:    credential.BackupState,
		LastUsedAt:     credential.LastUsedAt,
		CreatedAt:      credential.CreatedAt,
	}
}

func readPasskeyBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, services.ValidationError("passkey response is required")
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read passkey response: %w", err)
	}
	if len(body) == 0 {
		return nil, services.ValidationError("passkey response is required")
	}
	return body, nil
}
