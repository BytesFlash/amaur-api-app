package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	appcssession "amaur/api/internal/application/caresession"
	"amaur/api/internal/delivery/http/middleware"
	"amaur/api/internal/delivery/http/response"
	"amaur/api/pkg/pagination"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type CareSessionHandler struct {
	svc *appcssession.Service
}

func NewCareSessionHandler(svc *appcssession.Service) *CareSessionHandler {
	return &CareSessionHandler{svc: svc}
}

// GET /care-sessions
func (h *CareSessionHandler) List(w http.ResponseWriter, r *http.Request) {
	p := pagination.FromRequest(r)
	q := r.URL.Query()
	companyID := q.Get("company_id")
	patientID := q.Get("patient_id")
	claims := middleware.ClaimsFromContext(r.Context())
	if middleware.IsCompanyScopedRole(claims) {
		if claims.CompanyID == nil {
			response.Forbidden(w, "Missing company scope for current user")
			return
		}
		companyID = claims.CompanyID.String()
	}
	if middleware.IsPatientScopedRole(claims) {
		if claims.PatientID == nil {
			response.Forbidden(w, "Missing patient scope for current user")
			return
		}
		patientID = claims.PatientID.String()
	}
	items, total, err := h.svc.List(r.Context(),
		patientID, q.Get("worker_id"), companyID,
		q.Get("visit_id"), q.Get("session_type"), q.Get("status"),
		q.Get("date_from"), q.Get("date_to"),
		p.Limit, p.Offset)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Paginated(w, items, pagination.NewMeta(p, total))
}

// POST /care-sessions
func (h *CareSessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req appcssession.CreateCareSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	cs, err := h.svc.Create(r.Context(), req, claims.UserID)
	if err != nil {
		response.BadRequest(w, "INVALID_REQUEST", err.Error())
		return
	}
	response.Created(w, cs)
}

// GET /care-sessions/{id}
func (h *CareSessionHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid care session id")
		return
	}
	cs, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.NotFound(w, "NOT_FOUND", "Care session not found")
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	if middleware.IsCompanyScopedRole(claims) {
		if claims.CompanyID == nil || cs.CompanyID == nil || *cs.CompanyID != *claims.CompanyID {
			response.Forbidden(w, "You can only view care sessions from your company")
			return
		}
	}
	if middleware.IsPatientScopedRole(claims) {
		if claims.PatientID == nil || cs.PatientID != *claims.PatientID {
			response.Forbidden(w, "You can only view your own care sessions")
			return
		}
	}
	response.OK(w, cs)
}

// PATCH /care-sessions/{id}
func (h *CareSessionHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid care session id")
		return
	}
	var req appcssession.UpdateCareSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	cs, err := h.svc.Update(r.Context(), id, req, claims.UserID)
	if err != nil {
		if errors.Is(err, appcssession.ErrNotFound) {
			response.NotFound(w, "NOT_FOUND", "Care session not found")
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, cs)
}

// DELETE /care-sessions/{id}
func (h *CareSessionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid care session id")
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, appcssession.ErrNotFound) {
			response.NotFound(w, "NOT_FOUND", "Care session not found")
			return
		}
		response.InternalError(w)
		return
	}
	response.NoContent(w)
}

// GET /visits/{id}/group-sessions
func (h *CareSessionHandler) ListGroupSessions(w http.ResponseWriter, r *http.Request) {
	visitID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid visit id")
		return
	}
	items, err := h.svc.ListGroupSessions(r.Context(), visitID)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, items)
}

// POST /visits/{id}/group-sessions
func (h *CareSessionHandler) CreateGroupSession(w http.ResponseWriter, r *http.Request) {
	visitID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid visit id")
		return
	}
	var req appcssession.CreateGroupSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	req.VisitID = visitID
	claims := middleware.ClaimsFromContext(r.Context())
	gs, err := h.svc.CreateGroupSession(r.Context(), req, claims.UserID)
	if err != nil {
		response.BadRequest(w, "INVALID_REQUEST", err.Error())
		return
	}
	response.Created(w, gs)
}
