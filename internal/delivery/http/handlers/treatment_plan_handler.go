package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	apptp "amaur/api/internal/application/treatmentplan"
	"amaur/api/internal/delivery/http/middleware"
	"amaur/api/internal/delivery/http/response"
	"amaur/api/pkg/pagination"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type TreatmentPlanHandler struct {
	svc *apptp.Service
}

func NewTreatmentPlanHandler(svc *apptp.Service) *TreatmentPlanHandler {
	return &TreatmentPlanHandler{svc: svc}
}

func (h *TreatmentPlanHandler) List(w http.ResponseWriter, r *http.Request) {
	p := pagination.FromRequest(r)
	items, total, err := h.svc.List(
		r.Context(),
		r.URL.Query().Get("patient_id"),
		r.URL.Query().Get("professional_id"),
		r.URL.Query().Get("status"),
		r.URL.Query().Get("service_type_id"),
		p.Limit, p.Offset,
	)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Paginated(w, items, pagination.NewMeta(p, total))
}

func (h *TreatmentPlanHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req apptp.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	item, err := h.svc.Create(r.Context(), req, claims.UserID)
	if err != nil {
		response.BadRequest(w, "CREATE_ERROR", err.Error())
		return
	}
	response.Created(w, item)
}

func (h *TreatmentPlanHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid treatment plan id")
		return
	}
	item, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.NotFound(w, "PLAN_NOT_FOUND", "Treatment plan not found")
		return
	}
	response.OK(w, item)
}

func (h *TreatmentPlanHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid treatment plan id")
		return
	}
	var req apptp.UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	item, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, apptp.ErrNotFound):
			response.NotFound(w, "PLAN_NOT_FOUND", "Treatment plan not found")
		case errors.Is(err, apptp.ErrPlanTerminal):
			response.BadRequest(w, "PLAN_TERMINAL", err.Error())
		default:
			response.InternalError(w)
		}
		return
	}
	response.OK(w, item)
}

func (h *TreatmentPlanHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid treatment plan id")
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	item, err := h.svc.UpdateStatus(r.Context(), id, body.Status)
	if err != nil {
		switch {
		case errors.Is(err, apptp.ErrNotFound):
			response.NotFound(w, "PLAN_NOT_FOUND", "Treatment plan not found")
		case errors.Is(err, apptp.ErrInvalidStatus):
			response.BadRequest(w, "INVALID_STATUS", err.Error())
		default:
			response.InternalError(w)
		}
		return
	}
	response.OK(w, item)
}

func (h *TreatmentPlanHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid treatment plan id")
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, apptp.ErrNotFound) {
			response.NotFound(w, "PLAN_NOT_FOUND", "Treatment plan not found")
			return
		}
		response.InternalError(w)
		return
	}
	response.NoContent(w)
}

func (h *TreatmentPlanHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid treatment plan id")
		return
	}
	p := pagination.FromRequest(r)
	items, total, err := h.svc.GetHistory(r.Context(), id, p.Limit, p.Offset)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Paginated(w, items, pagination.NewMeta(p, total))
}

func (h *TreatmentPlanHandler) GetAlerts(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	professionalID := r.URL.Query().Get("professional_id")
	// Non-admin workers see only their own alerts
	if claims.WorkerID != nil && !claims.HasRole("super_admin") && !claims.HasRole("admin") {
		professionalID = claims.WorkerID.String()
	}
	alerts, err := h.svc.GetAlerts(r.Context(), professionalID)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, alerts)
}

func (h *TreatmentPlanHandler) GenerateSessions(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid treatment plan id")
		return
	}
	var req apptp.GenerateSessionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	items, err := h.svc.GenerateSessions(r.Context(), id, req, claims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, apptp.ErrNotFound):
			response.NotFound(w, "PLAN_NOT_FOUND", "Treatment plan not found")
		case errors.Is(err, apptp.ErrPlanTerminal):
			response.BadRequest(w, "PLAN_TERMINAL", err.Error())
		case errors.Is(err, apptp.ErrNothingToGenerate):
			response.BadRequest(w, "NOTHING_TO_GENERATE", err.Error())
		default:
			response.BadRequest(w, "GENERATE_ERROR", err.Error())
		}
		return
	}
	response.Created(w, items)
}

// PreviewSessions performs a dry-run of session generation and returns proposed dates
// with availability status per slot — no appointments are created.
func (h *TreatmentPlanHandler) PreviewSessions(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid treatment plan id")
		return
	}
	var req apptp.PreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	result, err := h.svc.PreviewSessions(r.Context(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, apptp.ErrNotFound):
			response.NotFound(w, "PLAN_NOT_FOUND", "Treatment plan not found")
		case errors.Is(err, apptp.ErrPlanTerminal):
			response.BadRequest(w, "PLAN_TERMINAL", err.Error())
		default:
			response.BadRequest(w, "PREVIEW_ERROR", err.Error())
		}
		return
	}
	response.OK(w, result)
}
