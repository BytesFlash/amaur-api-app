package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	appvisit "amaur/api/internal/application/visit"
	"amaur/api/internal/delivery/http/middleware"
	"amaur/api/internal/delivery/http/response"
	"amaur/api/pkg/pagination"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type VisitHandler struct {
	svc *appvisit.Service
}

func NewVisitHandler(svc *appvisit.Service) *VisitHandler {
	return &VisitHandler{svc: svc}
}

func (h *VisitHandler) List(w http.ResponseWriter, r *http.Request) {
	p := pagination.FromRequest(r)
	companyID := r.URL.Query().Get("company_id")
	patientID := ""
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
	status := r.URL.Query().Get("status")
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")
	visits, total, err := h.svc.List(r.Context(), companyID, patientID, status, dateFrom, dateTo, p.Limit, p.Offset)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Paginated(w, visits, pagination.NewMeta(p, total))
}

func (h *VisitHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req appvisit.CreateVisitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	req.CoordinatorUserID = &claims.UserID
	visit, err := h.svc.Create(r.Context(), req)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Created(w, visit)
}

func (h *VisitHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid visit id")
		return
	}
	visit, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.NotFound(w, "VISIT_NOT_FOUND", "Visit not found")
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	if middleware.IsCompanyScopedRole(claims) {
		if claims.CompanyID == nil || visit.CompanyID != *claims.CompanyID {
			response.Forbidden(w, "You can only view visits from your company")
			return
		}
	}
	if middleware.IsPatientScopedRole(claims) {
		if claims.PatientID == nil {
			response.Forbidden(w, "Missing patient scope for current user")
			return
		}
		hasParticipation, err := h.svc.HasPatientParticipation(r.Context(), visit.ID, *claims.PatientID)
		if err != nil {
			response.InternalError(w)
			return
		}
		if !hasParticipation {
			response.Forbidden(w, "You can only view visits where you participated")
			return
		}
	}
	response.OK(w, visit)
}

func (h *VisitHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid visit id")
		return
	}
	var req appvisit.UpdateVisitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	visit, err := h.svc.Update(r.Context(), id, req)
	if errors.Is(err, appvisit.ErrNotFound) {
		response.NotFound(w, "VISIT_NOT_FOUND", "Visit not found")
		return
	}
	if err != nil {
		response.InternalError(w)
		return
	}
	response.OK(w, visit)
}

func (h *VisitHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid visit id")
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, appvisit.ErrNotFound) {
			response.NotFound(w, "VISIT_NOT_FOUND", "Visit not found")
			return
		}
		response.InternalError(w)
		return
	}
	response.NoContent(w)
}
