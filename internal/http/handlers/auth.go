package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sunriseex/capitalflow/internal/http/dto"
	appmiddleware "github.com/sunriseex/capitalflow/internal/http/middleware"
	"github.com/sunriseex/capitalflow/internal/models"
	"github.com/sunriseex/capitalflow/internal/services"
)

func (h *Handler) authStatus(w http.ResponseWriter, r *http.Request) {
	setupRequired, err := h.app.Auth.SetupRequired(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.AuthStatusResponse{
		SetupRequired: setupRequired,
		Version:       h.appVersion,
	})
}

func (h *Handler) authSetup(w http.ResponseWriter, r *http.Request) {
	var req dto.AuthRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	session, err := h.authService().Setup(r.Context(), services.AuthRequest{
		Email:           req.Email,
		Password:        req.Password,
		PrimaryCurrency: req.PrimaryCurrency,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	h.setRefreshCookie(w, session)
	writeJSON(w, http.StatusCreated, authResponse(session))
}

func (h *Handler) authLogin(w http.ResponseWriter, r *http.Request) {
	var req dto.AuthRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	session, err := h.authService().Login(r.Context(), services.AuthRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	h.setRefreshCookie(w, session)
	writeJSON(w, http.StatusOK, authResponse(session))
}

func (h *Handler) authRefresh(w http.ResponseWriter, r *http.Request) {
	session, err := h.authService().Refresh(r.Context(), h.refreshTokenFromCookie(r))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	h.setRefreshCookie(w, session)
	writeJSON(w, http.StatusOK, authResponse(session))
}

func (h *Handler) authLogout(w http.ResponseWriter, r *http.Request) {
	refreshToken := h.refreshTokenFromCookie(r)

	h.clearRefreshCookie(w)

	if err := h.authService().Logout(r.Context(), refreshToken); err != nil {
		writeServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := appmiddleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
		return
	}

	var req dto.ChangePasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid request body", nil)
		return
	}

	if err := h.authService().ChangePassword(r.Context(), services.ChangePasswordRequest{
		UserID:          claims,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	}); err != nil {
		writeServiceError(w, err)
		return
	}
	h.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listSessions(w http.ResponseWriter, r *http.Request) {
	claims, ok := appmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
		return
	}

	sessions, err := h.authService().ListSessions(r.Context(), claims.UserID, claims.SessionID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, authSessionsResponse(sessions))
}

func (h *Handler) revokeSession(w http.ResponseWriter, r *http.Request) {
	claims, ok := appmiddleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
		return
	}

	sessionID := chi.URLParam(r, "id")
	if err := h.authService().RevokeSession(r.Context(), claims.UserID, sessionID); err != nil {
		writeServiceError(w, err)
		return
	}

	if sessionID == claims.SessionID {
		h.clearRefreshCookie(w)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) authService() *services.AuthService {
	return h.app.Auth
}

func authResponse(session *services.AuthSession) dto.AuthResponse {
	return dto.AuthResponse{
		User:            authUser(session.User),
		AccessToken:     session.AccessToken,
		AccessExpiresAt: session.AccessExpiresAt,
	}
}

func authUser(user *models.User) dto.AuthUser {
	return dto.AuthUser{
		ID:              user.ID,
		Email:           user.Email,
		PrimaryCurrency: user.PrimaryCurrency,
	}
}

func authSessionsResponse(sessions []services.SessionInfo) dto.AuthSessionsResponse {
	response := dto.AuthSessionsResponse{
		Sessions: make([]dto.AuthSessionInfo, 0, len(sessions)),
	}
	for _, session := range sessions {
		response.Sessions = append(response.Sessions, dto.AuthSessionInfo{
			ID:        session.ID,
			ExpiresAt: session.ExpiresAt,
			RevokedAt: session.RevokedAt,
			CreatedAt: session.CreatedAt,
			Active:    session.Active,
			Current:   session.Current,
		})
	}
	return response
}

const (
	refreshCookieName         = "__Secure-capitalflow_refresh"
	insecureRefreshCookieName = "capitalflow_refresh"
)

func (h *Handler) setRefreshCookie(w http.ResponseWriter, session *services.AuthSession) {
	// #nosec G124 -- Secure and SameSite are controlled by validated runtime config.
	http.SetCookie(w, &http.Cookie{
		Name:     h.refreshCookieName(),
		Value:    session.RefreshToken,
		Path:     "/auth",
		Expires:  session.RefreshExpiresAt,
		MaxAge:   max(1, int(time.Until(session.RefreshExpiresAt).Seconds())),
		Secure:   h.cookieSecure,
		HttpOnly: true,
		SameSite: h.cookieSameSite,
	})
}

func (h *Handler) clearRefreshCookie(w http.ResponseWriter) {
	// #nosec G124 -- deletion must use the same configurable cookie attributes.
	http.SetCookie(w, &http.Cookie{
		Name:     h.refreshCookieName(),
		Value:    "",
		Path:     "/auth",
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
		Secure:   h.cookieSecure,
		HttpOnly: true,
		SameSite: h.cookieSameSite,
	})
}

func (h *Handler) refreshCookieName() string {
	if h.cookieSecure {
		return refreshCookieName
	}
	return insecureRefreshCookieName
}

func cookieSameSiteMode(value string) http.SameSite {
	switch value {
	case "Lax":
		return http.SameSiteLaxMode
	case "None":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteStrictMode
	}
}

func (h *Handler) refreshTokenFromCookie(r *http.Request) string {
	cookie, err := r.Cookie(h.refreshCookieName())
	if err != nil {
		return ""
	}
	return cookie.Value
}
