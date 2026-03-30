package handlers

import (
	"encoding/json"
	"net/http"

	appauth "amaur/api/internal/application/auth"
	"amaur/api/internal/delivery/http/middleware"
	"amaur/api/internal/delivery/http/response"
)

type AuthHandler struct {
	svc *appauth.Service
}

func NewAuthHandler(svc *appauth.Service) *AuthHandler {
	return &AuthHandler{svc: svc}
}

// POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req appauth.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}

	resp, err := h.svc.Login(r.Context(), req)
	if err != nil {
		switch err {
		case appauth.ErrInvalidCredentials:
			response.Unauthorized(w, "Invalid email or password")
		case appauth.ErrUserInactive:
			response.Unauthorized(w, "User account is disabled")
		case appauth.ErrAccountLocked:
			response.Unauthorized(w, "Account temporarily locked")
		default:
			response.InternalError(w)
		}
		return
	}

	response.OK(w, resp)
}

// POST /api/v1/auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req appauth.RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}

	resp, err := h.svc.Refresh(r.Context(), req)
	if err != nil {
		response.Unauthorized(w, "Invalid or expired refresh token")
		return
	}

	response.OK(w, resp)
}

// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req appauth.RefreshRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	claims := middleware.ClaimsFromContext(r.Context())
	if claims != nil && req.RefreshToken != "" {
		_ = h.svc.Logout(r.Context(), claims.UserID, req.RefreshToken)
	}
	response.NoContent(w)
}

// GET /api/v1/auth/me
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		response.Unauthorized(w, "Not authenticated")
		return
	}
	response.OK(w, appauth.UserDTO{
		ID:          claims.UserID,
		Email:       claims.Email,
		Roles:       claims.Roles,
		Permissions: claims.Permissions,
	})
}
