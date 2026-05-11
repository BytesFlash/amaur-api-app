package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	appfu "amaur/api/internal/application/followuptask"
	"amaur/api/internal/delivery/http/middleware"
	"amaur/api/internal/delivery/http/response"
	"amaur/api/pkg/pagination"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type FollowUpTaskHandler struct {
	svc *appfu.Service
}

func NewFollowUpTaskHandler(svc *appfu.Service) *FollowUpTaskHandler {
	return &FollowUpTaskHandler{svc: svc}
}

func (h *FollowUpTaskHandler) List(w http.ResponseWriter, r *http.Request) {
	p := pagination.FromRequest(r)
	claims := middleware.ClaimsFromContext(r.Context())
	professionalID := r.URL.Query().Get("professional_id")
	if claims.WorkerID != nil && !claims.HasRole("super_admin") && !claims.HasRole("admin") {
		professionalID = claims.WorkerID.String()
	}
	items, total, err := h.svc.List(
		r.Context(),
		r.URL.Query().Get("patient_id"),
		professionalID,
		r.URL.Query().Get("status"),
		r.URL.Query().Get("due_before"),
		p.Limit, p.Offset,
	)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Paginated(w, items, pagination.NewMeta(p, total))
}

func (h *FollowUpTaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req appfu.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	if req.ProfessionalID == nil && claims.WorkerID != nil {
		req.ProfessionalID = claims.WorkerID
	}
	item, err := h.svc.Create(r.Context(), req, claims.UserID)
	if err != nil {
		response.BadRequest(w, "CREATE_ERROR", err.Error())
		return
	}
	response.Created(w, item)
}

func (h *FollowUpTaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid follow-up task id")
		return
	}
	var req appfu.UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	item, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, appfu.ErrNotFound):
			response.NotFound(w, "TASK_NOT_FOUND", "Follow-up task not found")
		case errors.Is(err, appfu.ErrInvalidStatus):
			response.BadRequest(w, "INVALID_STATUS", err.Error())
		default:
			response.InternalError(w)
		}
		return
	}
	response.OK(w, item)
}

func (h *FollowUpTaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid follow-up task id")
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, appfu.ErrNotFound) {
			response.NotFound(w, "TASK_NOT_FOUND", "Follow-up task not found")
			return
		}
		response.InternalError(w)
		return
	}
	response.NoContent(w)
}
