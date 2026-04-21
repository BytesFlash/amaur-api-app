package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	appappt "amaur/api/internal/application/appointment"
	"amaur/api/internal/delivery/http/middleware"
	"amaur/api/internal/delivery/http/response"
	domainappt "amaur/api/internal/domain/appointment"
	"amaur/api/pkg/pagination"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type AppointmentHandler struct {
	svc *appappt.Service
}

func NewAppointmentHandler(svc *appappt.Service) *AppointmentHandler {
	return &AppointmentHandler{svc: svc}
}

func (h *AppointmentHandler) List(w http.ResponseWriter, r *http.Request) {
	p := pagination.FromRequest(r)
	patientID := r.URL.Query().Get("patient_id")
	workerID := r.URL.Query().Get("worker_id")
	companyID := r.URL.Query().Get("company_id")
	status := r.URL.Query().Get("status")
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	claims := middleware.ClaimsFromContext(r.Context())
	if middleware.IsPatientScopedRole(claims) {
		if claims.PatientID == nil {
			response.Forbidden(w, "Missing patient scope")
			return
		}
		patientID = claims.PatientID.String()
	}
	if middleware.IsCompanyScopedRole(claims) {
		if claims.CompanyID == nil {
			response.Forbidden(w, "Missing company scope")
			return
		}
		companyID = claims.CompanyID.String()
	}

	items, total, err := h.svc.List(r.Context(), patientID, workerID, companyID, status, dateFrom, dateTo, p.Limit, p.Offset)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Paginated(w, items, pagination.NewMeta(p, total))
}

func (h *AppointmentHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req appappt.CreateAppointmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	if middleware.IsPatientScopedRole(claims) {
		if claims.PatientID == nil {
			response.Forbidden(w, "Missing patient scope")
			return
		}
		req.PatientID = *claims.PatientID
	}
	if middleware.IsCompanyScopedRole(claims) {
		if claims.CompanyID == nil {
			response.Forbidden(w, "Missing company scope")
			return
		}
		req.CompanyID = claims.CompanyID
	}
	items, err := h.svc.Create(r.Context(), req, claims.UserID)
	if err != nil {
		switch {
		case errors.Is(err, appappt.ErrInvalidDate):
			response.BadRequest(w, "INVALID_DATE", err.Error())
		case errors.Is(err, appappt.ErrInvalidRecurrence):
			response.BadRequest(w, "INVALID_RECURRENCE", err.Error())
		case errors.Is(err, appappt.ErrInvalidStatus):
			response.BadRequest(w, "INVALID_STATUS", err.Error())
		case errors.Is(err, appappt.ErrTooSoon):
			response.BadRequest(w, "TIME_TOO_SOON", err.Error())
		case errors.Is(err, appappt.ErrOutsideAvailability):
			response.BadRequest(w, "OUTSIDE_AVAILABILITY", err.Error())
		case errors.Is(err, appappt.ErrWorkerBusy):
			response.Conflict(w, "WORKER_BUSY", err.Error())
		default:
			response.InternalError(w)
		}
		return
	}
	response.Created(w, items)
}

func (h *AppointmentHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid appointment id")
		return
	}
	item, ok := h.getScopedAppointment(w, r, id)
	if !ok {
		return
	}
	response.OK(w, item)
}

func (h *AppointmentHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid appointment id")
		return
	}
	if _, ok := h.getScopedAppointment(w, r, id); !ok {
		return
	}
	var req appappt.UpdateAppointmentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "INVALID_BODY", "Invalid request body")
		return
	}
	claims := middleware.ClaimsFromContext(r.Context())
	item, err := h.svc.Update(r.Context(), id, req, claims.UserID)
	if err != nil {
		if errors.Is(err, appappt.ErrNotFound) {
			response.NotFound(w, "APPOINTMENT_NOT_FOUND", "Appointment not found")
			return
		}
		if errors.Is(err, appappt.ErrInvalidDate) {
			response.BadRequest(w, "INVALID_DATE", err.Error())
			return
		}
		if errors.Is(err, appappt.ErrInvalidStatus) {
			response.BadRequest(w, "INVALID_STATUS", err.Error())
			return
		}
		if errors.Is(err, appappt.ErrTooSoon) {
			response.BadRequest(w, "TIME_TOO_SOON", err.Error())
			return
		}
		if errors.Is(err, appappt.ErrOutsideAvailability) {
			response.BadRequest(w, "OUTSIDE_AVAILABILITY", err.Error())
			return
		}
		if errors.Is(err, appappt.ErrWorkerBusy) {
			response.Conflict(w, "WORKER_BUSY", err.Error())
			return
		}
		response.InternalError(w)
		return
	}
	response.OK(w, item)
}

func (h *AppointmentHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.BadRequest(w, "INVALID_ID", "Invalid appointment id")
		return
	}
	if _, ok := h.getScopedAppointment(w, r, id); !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, appappt.ErrNotFound) {
			response.NotFound(w, "APPOINTMENT_NOT_FOUND", "Appointment not found")
			return
		}
		response.InternalError(w)
		return
	}
	response.NoContent(w)
}

func (h *AppointmentHandler) getScopedAppointment(w http.ResponseWriter, r *http.Request, id uuid.UUID) (*domainappt.Appointment, bool) {
	item, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.NotFound(w, "APPOINTMENT_NOT_FOUND", "Appointment not found")
		return nil, false
	}

	claims := middleware.ClaimsFromContext(r.Context())
	if middleware.IsPatientScopedRole(claims) {
		if claims.PatientID == nil {
			response.Forbidden(w, "Missing patient scope")
			return nil, false
		}
		if item.PatientID != *claims.PatientID {
			response.Forbidden(w, "You do not have access to this appointment")
			return nil, false
		}
	}
	if middleware.IsCompanyScopedRole(claims) {
		if claims.CompanyID == nil {
			response.Forbidden(w, "Missing company scope")
			return nil, false
		}
		if item.CompanyID == nil || *item.CompanyID != *claims.CompanyID {
			response.Forbidden(w, "You do not have access to this appointment")
			return nil, false
		}
	}

	return item, true
}
