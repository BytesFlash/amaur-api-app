package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	appuser "amaur/api/internal/application/user"
	"amaur/api/internal/delivery/http/middleware"
	"amaur/api/internal/delivery/http/response"
	"amaur/api/pkg/pagination"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type UserHandler struct {
	svc *appuser.Service
}

func NewUserHandler(svc *appuser.Service) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	p := pagination.FromRequest(r)
	users, total, err := h.svc.List(r.Context(), p.Limit, p.Offset)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Paginated(w, users, pagination.NewMeta(p, total))
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req appuser.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	dto, err := h.svc.Create(r.Context(), req, claims.UserID)
	if err != nil {
		if errors.Is(err, appuser.ErrEmailTaken) {
			response.Conflict(w, "EMAIL_TAKEN", "Email already in use")
			return
		}
		if errors.Is(err, appuser.ErrRoleRequired) {
			response.BadRequest(w, "ROLE_REQUIRED", "At least one role is required")
			return
		}
		if errors.Is(err, appuser.ErrCompanyScopeRequired) {
			response.BadRequest(w, "COMPANY_SCOPE_REQUIRED", "company_id is required for company-scoped roles")
			return
		}
		if errors.Is(err, appuser.ErrPatientScopeRequired) {
			response.BadRequest(w, "PATIENT_SCOPE_REQUIRED", "patient_id is required for company_worker role")
			return
		}
		response.InternalError(w)
		return
	}
	response.Created(w, dto)
}

func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid user id")
		return
	}
	dto, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.NotFound(w, "USER_NOT_FOUND", "User not found")
		return
	}
	response.OK(w, dto)
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid user id")
		return
	}
	var req appuser.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	dto, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, appuser.ErrUserNotFound) {
			response.NotFound(w, "USER_NOT_FOUND", "User not found")
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, dto)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid user id")
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, appuser.ErrUserNotFound) {
			response.NotFound(w, "USER_NOT_FOUND", "User not found")
			return
		}
		response.InternalError(w)
		return
	}
	response.NoContent(w)
}

func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid user id")
		return
	}
	var req appuser.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	if err := h.svc.ChangePassword(r.Context(), id, req); err != nil {
		if errors.Is(err, appuser.ErrUserNotFound) {
			response.NotFound(w, "USER_NOT_FOUND", "User not found")
			return
		}
		response.InternalError(w)
		return
	}
	response.NoContent(w)
}

func (h *UserHandler) AssignRoles(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid user id")
		return
	}
	var req appuser.AssignRolesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	if err := h.svc.AssignRoles(r.Context(), id, req, claims.UserID); err != nil {
		if errors.Is(err, appuser.ErrUserNotFound) {
			response.NotFound(w, "USER_NOT_FOUND", "User not found")
			return
		}
		if errors.Is(err, appuser.ErrRoleRequired) {
			response.BadRequest(w, "ROLE_REQUIRED", "At least one role is required")
			return
		}
		if errors.Is(err, appuser.ErrCompanyScopeRequired) {
			response.BadRequest(w, "COMPANY_SCOPE_REQUIRED", "company_id is required for company-scoped roles")
			return
		}
		if errors.Is(err, appuser.ErrPatientScopeRequired) {
			response.BadRequest(w, "PATIENT_SCOPE_REQUIRED", "patient_id is required for company_worker role")
			return
		}
		response.InternalError(w)
		return
	}
	response.NoContent(w)
}

func (h *UserHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.svc.ListRoles(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, roles)
}
