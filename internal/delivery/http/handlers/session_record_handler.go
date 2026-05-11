package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	appsr "amaur/api/internal/application/sessionrecord"
	"amaur/api/internal/delivery/http/middleware"
	"amaur/api/internal/delivery/http/response"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type SessionRecordHandler struct {
	svc *appsr.Service
}

func NewSessionRecordHandler(svc *appsr.Service) *SessionRecordHandler {
	return &SessionRecordHandler{svc: svc}
}

func (h *SessionRecordHandler) ListByPlan(w http.ResponseWriter, r *http.Request) {
	planID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid treatment plan id")
		return
	}
	items, err := h.svc.ListByPlan(r.Context(), planID)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, items)
}

func (h *SessionRecordHandler) Create(w http.ResponseWriter, r *http.Request) {
	planID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid treatment plan id")
		return
	}
	var req appsr.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	req.TreatmentPlanID = planID
	claims := middleware.ClaimsFromContext(r.Context())
	// Primer fallback: usar worker_id del JWT si no se proveyó professional_id.
	if (req.ProfessionalID == nil || *req.ProfessionalID == uuid.Nil) && claims.WorkerID != nil {
		req.ProfessionalID = claims.WorkerID
	}
	item, err := h.svc.Create(r.Context(), req, claims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, appsr.ErrPlanTerminal):
			response.BadRequest(w, "PLAN_TERMINAL", err.Error())
		case errors.Is(err, appsr.ErrNoProfessional):
			response.BadRequest(w, "NO_PROFESSIONAL", err.Error())
		default:
			response.BadRequest(w, "CREATE_ERROR", err.Error())
		}
		return
	}
	response.Created(w, item)
}

func (h *SessionRecordHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "recordId"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid session record id")
		return
	}
	var req appsr.UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	item, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, appsr.ErrNotFound) {
			response.NotFound(w, "RECORD_NOT_FOUND", "Session record not found")
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, item)
}
